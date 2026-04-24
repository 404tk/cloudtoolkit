package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
)

const (
	DefaultBaseURL = "https://api.ucloud.cn"
	contentType    = "application/x-www-form-urlencoded"
)

type Request struct {
	Action    string
	Region    string
	ProjectID string
	Params    map[string]any
}

type Option func(*Client)

type Client struct {
	baseURL    string
	credential ucloudauth.Credential
	httpClient *http.Client
	projectID  string
}

type APIError struct {
	Action     string
	Code       int
	Message    string
	RawBody    string
	StatusCode int
}

func (e *APIError) Error() string {
	switch {
	case e.Code > 0 && strings.TrimSpace(e.Message) != "":
		return fmt.Sprintf("ucloud api error: code=%d %s", e.Code, strings.TrimSpace(e.Message))
	case strings.TrimSpace(e.Message) != "":
		return fmt.Sprintf("ucloud api error: %s", strings.TrimSpace(e.Message))
	case e.StatusCode > 0:
		return fmt.Sprintf("ucloud api error: http status %d", e.StatusCode)
	default:
		return "ucloud api error"
	}
}

func NewClient(credential ucloudauth.Credential, opts ...Option) *Client {
	client := &Client{
		baseURL:    DefaultBaseURL,
		credential: credential,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		if strings.TrimSpace(baseURL) == "" {
			return
		}
		c.baseURL = baseURL
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

func WithProjectID(projectID string) Option {
	return func(c *Client) {
		c.projectID = strings.TrimSpace(projectID)
	}
}

func (c *Client) Do(ctx context.Context, req Request, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return err
	}

	action := strings.TrimSpace(req.Action)
	if action == "" {
		return fmt.Errorf("ucloud client: empty action")
	}

	form, err := encodeForm(req.Params)
	if err != nil {
		return err
	}
	form["Action"] = action

	if region := strings.TrimSpace(req.Region); region != "" {
		form["Region"] = region
	}

	projectID := strings.TrimSpace(req.ProjectID)
	if projectID == "" {
		projectID = c.projectID
	}
	if projectID != "" {
		form["ProjectId"] = projectID
	}

	form["PublicKey"] = strings.TrimSpace(c.credential.AccessKey)
	if token := strings.TrimSpace(c.credential.SecurityToken); token != "" {
		form["SecurityToken"] = token
	}
	form["Signature"] = signature(form, c.credential.SecretKey)

	values := url.Values{}
	for key, value := range form {
		values.Set(key, value)
	}

	httpReq, err := c.newHTTPRequest(ctx, values.Encode())
	if err != nil {
		return err
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer closeResponse(httpResp)

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read ucloud response: %w", err)
	}

	return c.decodeResponse(req.Action, httpResp.StatusCode, body, out)
}

func (c *Client) newHTTPRequest(ctx context.Context, body string) (*http.Request, error) {
	baseURL := strings.TrimSpace(c.baseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("ucloud client: invalid base url %q: %w", baseURL, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("ucloud client: invalid base url %q", baseURL)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, parsed.String(), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", contentType)
	return httpReq, nil
}

func (c *Client) decodeResponse(action string, statusCode int, body []byte, out any) error {
	if len(body) == 0 {
		if statusCode >= http.StatusBadRequest {
			return &APIError{Action: action, StatusCode: statusCode, Message: http.StatusText(statusCode)}
		}
		if out == nil {
			return nil
		}
		return fmt.Errorf("decode ucloud response: empty body")
	}

	var base BaseResponse
	baseErr := json.Unmarshal(body, &base)
	if baseErr == nil && (base.RetCode != 0 || statusCode >= http.StatusBadRequest) {
		message := strings.TrimSpace(base.Message)
		if message == "" {
			message = strings.TrimSpace(http.StatusText(statusCode))
		}
		return &APIError{
			Action:     action,
			Code:       int(base.RetCode),
			Message:    message,
			RawBody:    string(body),
			StatusCode: statusCode,
		}
	}
	if baseErr != nil && statusCode >= http.StatusBadRequest {
		return &APIError{
			Action:     action,
			Message:    strings.TrimSpace(string(body)),
			RawBody:    string(body),
			StatusCode: statusCode,
		}
	}
	if out == nil {
		if baseErr != nil {
			return fmt.Errorf("decode ucloud response: %w", baseErr)
		}
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode ucloud response: %w", err)
	}
	return nil
}

func closeResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

func encodeForm(params map[string]any) (map[string]string, error) {
	result := make(map[string]string)
	if len(params) == 0 {
		return result, nil
	}

	for _, key := range sortedAnyKeys(params) {
		if strings.TrimSpace(key) == "" {
			continue
		}
		if err := encodeValue(result, key, reflect.ValueOf(params[key])); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func encodeValue(dst map[string]string, key string, value reflect.Value) error {
	if !value.IsValid() {
		return nil
	}

	for value.Kind() == reflect.Interface || value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if err := encodeValue(dst, fmt.Sprintf("%s.%d", key, i), value.Index(i)); err != nil {
				return err
			}
		}
		return nil
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("ucloud client: unsupported map key type %s", value.Type().Key())
		}

		for _, name := range sortedReflectStringMapKeys(value) {
			if err := encodeValue(dst, key+"."+name, value.MapIndex(reflect.ValueOf(name))); err != nil {
				return err
			}
		}
		return nil
	default:
		encoded, err := encodeScalar(value)
		if err != nil {
			return err
		}
		if encoded != "" {
			dst[key] = encoded
		}
		return nil
	}
}

func encodeScalar(value reflect.Value) (string, error) {
	switch value.Kind() {
	case reflect.String:
		return value.String(), nil
	case reflect.Bool:
		return strconv.FormatBool(value.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(value.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(value.Uint(), 10), nil
	case reflect.Float32:
		return strconv.FormatFloat(value.Float(), 'f', -1, 64), nil
	case reflect.Float64:
		return strconv.FormatFloat(value.Float(), 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("ucloud client: unsupported param type %s", value.Kind())
	}
}

func sortedAnyKeys(params map[string]any) []string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedReflectStringMapKeys(value reflect.Value) []string {
	mapKeys := value.MapKeys()
	keys := make([]string, 0, len(mapKeys))
	for _, mapKey := range mapKeys {
		keys = append(keys, mapKey.String())
	}
	sort.Strings(keys)
	return keys
}
