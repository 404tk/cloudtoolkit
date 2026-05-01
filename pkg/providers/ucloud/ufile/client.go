package ufile

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
	ucloudapi "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
)

const defaultBucketEndpointFormat = "https://%s.%s.ufileos.com"

// FileClient is the UFile bucket-level client. Distinct from the JSON-RPC
// `api.Client` (`api.ucloud.cn`) which only handles bucket creation /
// listing — object-level operations live on the per-bucket `*.ufileos.com`
// host with HMAC-SHA1 signing per UFile's own auth scheme.
type FileClient struct {
	credential  ucloudauth.Credential
	httpClient  *http.Client
	retryPolicy ucloudapi.RetryPolicy
	now         func() time.Time
	endpointFmt string
}

type FileClientOption func(*FileClient)

func NewFileClient(cred ucloudauth.Credential, opts ...FileClientOption) *FileClient {
	c := &FileClient{
		credential:  cred,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		retryPolicy: ucloudapi.DefaultRetryPolicy(),
		now:         time.Now,
		endpointFmt: defaultBucketEndpointFormat,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithFileHTTPClient(hc *http.Client) FileClientOption {
	return func(c *FileClient) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

func WithFileRetryPolicy(p ucloudapi.RetryPolicy) FileClientOption {
	return func(c *FileClient) {
		c.retryPolicy = p
	}
}

func WithFileClock(now func() time.Time) FileClientOption {
	return func(c *FileClient) {
		if now != nil {
			c.now = now
		}
	}
}

// WithFileEndpointFormat overrides the per-bucket endpoint template. The
// format string takes two `%s` placeholders: bucket and region. Used by
// tests to redirect the per-bucket host onto a single httptest server.
func WithFileEndpointFormat(template string) FileClientOption {
	return func(c *FileClient) {
		template = strings.TrimSpace(template)
		if template == "" {
			return
		}
		c.endpointFmt = template
	}
}

// PrefixFileListResponse mirrors the body returned by `GET /?list`.
type PrefixFileListResponse struct {
	BucketName string         `json:"BucketName" xml:"-"`
	BucketID   string         `json:"BucketId" xml:"-"`
	NextMarker string         `json:"NextMarker" xml:"-"`
	DataSet    []UFileObject  `json:"DataSet" xml:"-"`
}

type UFileObject struct {
	FileName     string `json:"FileName"`
	Hash         string `json:"Hash"`
	MimeType     string `json:"MimeType"`
	Size         int64  `json:"Size"`
	ModifyTime   int64  `json:"ModifyTime"`
	StorageClass string `json:"StorageClass"`
}

type ufileError struct {
	XMLName xml.Name `xml:"Error"`
	RetCode int      `json:"RetCode" xml:"RetCode"`
	ErrMsg  string   `json:"ErrMsg" xml:"ErrMsg"`
}

// PrefixFileList enumerates objects in bucket prefixed by prefix. region is
// the bucket region (e.g. "cn-bj"). marker is the continuation cursor
// returned by the previous call; pass "" for the first page.
func (c *FileClient) PrefixFileList(ctx context.Context, bucket, region, prefix, marker string, limit int) (PrefixFileListResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	bucket = strings.TrimSpace(bucket)
	region = strings.TrimSpace(region)
	if bucket == "" || region == "" {
		return PrefixFileListResponse{}, fmt.Errorf("ucloud ufile: empty bucket or region")
	}
	rawURL := fmt.Sprintf(c.endpointFmt, bucket, region)
	u, err := url.Parse(rawURL)
	if err != nil {
		return PrefixFileListResponse{}, fmt.Errorf("ucloud ufile: invalid endpoint %q: %w", rawURL, err)
	}
	q := u.Query()
	q.Set("list", "")
	if p := strings.TrimSpace(prefix); p != "" {
		q.Set("prefix", p)
	}
	if m := strings.TrimSpace(marker); m != "" {
		q.Set("marker", m)
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	u.RawQuery = q.Encode()

	httpResp, err := c.retryPolicy.Do(ctx, true, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		c.signRequest(req, bucket, "")
		return c.httpClient.Do(req)
	})
	if err != nil {
		return PrefixFileListResponse{}, err
	}
	if httpResp == nil {
		return PrefixFileListResponse{}, fmt.Errorf("ucloud ufile: empty response")
	}
	defer httpclient.CloseResponse(httpResp)
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return PrefixFileListResponse{}, fmt.Errorf("read ucloud ufile response: %w", err)
	}
	if err := decodeFileError(httpResp.StatusCode, body); err != nil {
		return PrefixFileListResponse{}, err
	}
	var out PrefixFileListResponse
	if len(body) == 0 {
		return out, nil
	}
	if err := jsonDecode(body, &out); err != nil {
		return PrefixFileListResponse{}, fmt.Errorf("decode ucloud ufile response: %w", err)
	}
	return out, nil
}

// signRequest applies the UFile HMAC-SHA1 Authorization header to req.
// objectKey is the URL-decoded key when operating on a single object;
// pass "" for bucket-level operations like ?list.
func (c *FileClient) signRequest(req *http.Request, bucket, objectKey string) {
	if req == nil {
		return
	}
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	now := c.now().UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	dateValue := req.Header.Get("Date")
	if dateValue == "" {
		dateValue = now.Format(http.TimeFormat)
		req.Header.Set("Date", dateValue)
	}
	contentMD5 := req.Header.Get("Content-MD5")
	contentType := req.Header.Get("Content-Type")

	canonicalResource := "/" + strings.TrimSpace(bucket) + "/"
	objectKey = strings.TrimSpace(objectKey)
	if objectKey != "" {
		canonicalResource = "/" + bucket + "/" + strings.TrimPrefix(objectKey, "/")
	}

	stringToSign := strings.Join([]string{
		strings.ToUpper(strings.TrimSpace(req.Method)),
		contentMD5,
		contentType,
		dateValue,
		canonicalResource,
	}, "\n")

	mac := hmac.New(sha1.New, []byte(c.credential.SecretKey))
	_, _ = mac.Write([]byte(stringToSign))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	req.Header.Set("Authorization", "UCloud "+c.credential.AccessKey+":"+sig)
}

func decodeFileError(statusCode int, body []byte) error {
	if statusCode < http.StatusBadRequest {
		return nil
	}
	if len(body) == 0 {
		return fmt.Errorf("ucloud ufile: status %d", statusCode)
	}
	var jsonErr ufileError
	if err := jsonDecode(body, &jsonErr); err == nil && (jsonErr.RetCode != 0 || jsonErr.ErrMsg != "") {
		return fmt.Errorf("ucloud ufile: code=%d %s", jsonErr.RetCode, jsonErr.ErrMsg)
	}
	var xmlErr ufileError
	if err := xml.Unmarshal(body, &xmlErr); err == nil && (xmlErr.RetCode != 0 || xmlErr.ErrMsg != "") {
		return fmt.Errorf("ucloud ufile: code=%d %s", xmlErr.RetCode, xmlErr.ErrMsg)
	}
	return fmt.Errorf("ucloud ufile: status %d body %q", statusCode, strings.TrimSpace(string(body)))
}

func jsonDecode(body []byte, out any) error {
	return jsonStdUnmarshal(body, out)
}
