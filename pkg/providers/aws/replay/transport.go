package replay

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

type transport struct {
	mu           sync.Mutex
	sequence     int
	createdUsers map[string]iamUserFixture
	deletedUsers map[string]bool
}

func newTransport() *transport {
	return &transport{
		createdUsers: make(map[string]iamUserFixture),
		deletedUsers: make(map[string]bool),
	}
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := demoreplay.ReadRequestBody(req)
	if err != nil {
		return nil, err
	}
	switch verifyAuth(req, body) {
	case demoreplay.AuthInvalidAccessKey:
		return apiErrorResponse(req, http.StatusForbidden, "InvalidClientTokenId", "The security token included in the request is invalid."), nil
	case demoreplay.AuthInvalidSignature:
		return apiErrorResponse(req, http.StatusForbidden, "SignatureDoesNotMatch", "The request signature we calculated does not match the signature you provided."), nil
	}

	host := strings.ToLower(strings.TrimSpace(req.URL.Hostname()))
	switch {
	case isSTSHost(host):
		return t.handleSTS(req, body)
	case isEC2Host(host):
		return t.handleEC2(req, body)
	case isIAMHost(host):
		return t.handleIAM(req, body)
	case isS3Host(host):
		return t.handleS3(req, host)
	}
	return apiErrorResponse(req, http.StatusNotFound, "InvalidEndpoint", fmt.Sprintf("unsupported replay host: %s", host)), nil
}

func (t *transport) handleSTS(req *http.Request, body []byte) (*http.Response, error) {
	form, err := parseFormBody(body)
	if err != nil {
		return apiErrorResponse(req, http.StatusBadRequest, "MalformedQueryString", err.Error()), nil
	}
	action := form.Get("Action")
	switch action {
	case "GetCallerIdentity":
		resp := stsGetCallerIdentityResponse{
			Result: stsGetCallerIdentityResult{
				Account: demoAccountID,
				Arn:     demoCallerArn(),
				UserID:  demoCallerUserID(),
			},
			Metadata: awsResponseMetadata{RequestID: "req-replay-sts-caller"},
		}
		return demoreplay.XMLResponse(req, http.StatusOK, resp), nil
	}
	return apiErrorResponse(req, http.StatusBadRequest, "InvalidAction", fmt.Sprintf("unsupported sts action: %s", action)), nil
}

func (t *transport) handleEC2(req *http.Request, body []byte) (*http.Response, error) {
	form, err := parseFormBody(body)
	if err != nil {
		return apiErrorResponse(req, http.StatusBadRequest, "MalformedQueryString", err.Error()), nil
	}
	action := form.Get("Action")
	region := regionFromHost(req.URL.Hostname())
	switch action {
	case "DescribeRegions":
		resp := ec2DescribeRegionsResponse{}
		for _, name := range demoRegions {
			resp.Regions = append(resp.Regions, ec2RegionWire{Name: name})
		}
		return demoreplay.XMLResponse(req, http.StatusOK, resp), nil
	case "DescribeInstances":
		hosts := ec2HostsForRegion(region)
		resp := ec2DescribeInstancesResponse{}
		reservation := ec2ReservationWire{}
		for _, host := range hosts {
			instance := ec2InstanceWire{
				InstanceID:    host.InstanceID,
				PublicIP:      host.PublicIP,
				PrivateIP:     host.PrivateIP,
				PublicDNSName: host.PublicDNSName,
				State:         ec2StateWire{Name: host.State},
			}
			for _, tag := range host.Tags {
				instance.Tags = append(instance.Tags, ec2TagWire{Key: tag.Key, Value: tag.Value})
			}
			reservation.Instances = append(reservation.Instances, instance)
		}
		if len(reservation.Instances) > 0 {
			resp.Reservations = append(resp.Reservations, reservation)
		}
		return demoreplay.XMLResponse(req, http.StatusOK, resp), nil
	}
	return apiErrorResponse(req, http.StatusBadRequest, "InvalidAction", fmt.Sprintf("unsupported ec2 action: %s", action)), nil
}

func (t *transport) handleIAM(req *http.Request, body []byte) (*http.Response, error) {
	form, err := parseFormBody(body)
	if err != nil {
		return apiErrorResponse(req, http.StatusBadRequest, "MalformedQueryString", err.Error()), nil
	}
	action := form.Get("Action")
	switch action {
	case "ListUsers":
		users := t.snapshotIAMUsers()
		resp := iamListUsersResponse{
			Metadata: awsResponseMetadata{RequestID: "req-replay-iam-list-users"},
		}
		for _, user := range users {
			resp.Result.Users = append(resp.Result.Users, iamUserWire{
				UserName:         user.UserName,
				UserID:           user.UserID,
				Arn:              user.Arn,
				CreateDate:       user.CreateDate,
				PasswordLastUsed: user.LastLoginDate,
			})
		}
		return demoreplay.XMLResponse(req, http.StatusOK, resp), nil
	case "GetLoginProfile":
		userName := strings.TrimSpace(form.Get("UserName"))
		user, ok := t.findUser(userName)
		if !ok || !user.HasLogin {
			return apiErrorResponse(req, http.StatusNotFound, "NoSuchEntity", fmt.Sprintf("login profile for %s not found", userName)), nil
		}
		resp := iamGetLoginProfileResponse{
			Metadata: awsResponseMetadata{RequestID: "req-replay-iam-get-login-profile"},
		}
		resp.Result.LoginProfile = iamLoginProfileWire{
			CreateDate:            user.CreateDate,
			PasswordResetRequired: false,
		}
		return demoreplay.XMLResponse(req, http.StatusOK, resp), nil
	case "ListAttachedUserPolicies":
		userName := strings.TrimSpace(form.Get("UserName"))
		user, ok := t.findUser(userName)
		if !ok {
			return apiErrorResponse(req, http.StatusNotFound, "NoSuchEntity", fmt.Sprintf("user %s not found", userName)), nil
		}
		resp := iamListAttachedUserPoliciesResponse{
			Metadata: awsResponseMetadata{RequestID: "req-replay-iam-list-policies"},
		}
		for _, policy := range user.AttachedPolicy {
			resp.Result.Policies = append(resp.Result.Policies, iamAttachedPolicyWire{
				PolicyName: policy.Name,
				PolicyArn:  policy.Arn,
			})
		}
		return demoreplay.XMLResponse(req, http.StatusOK, resp), nil
	case "CreateUser":
		userName := strings.TrimSpace(form.Get("UserName"))
		user := t.ensureUser(userName)
		resp := iamCreateUserResponse{
			Metadata: awsResponseMetadata{RequestID: "req-replay-iam-create-user"},
		}
		resp.Result.User = iamUserWire{
			UserName:   user.UserName,
			UserID:     user.UserID,
			Arn:        user.Arn,
			CreateDate: user.CreateDate,
		}
		return demoreplay.XMLResponse(req, http.StatusOK, resp), nil
	case "CreateLoginProfile":
		userName := strings.TrimSpace(form.Get("UserName"))
		t.markLoginProfile(userName, true)
		return demoreplay.XMLResponse(req, http.StatusOK, awsAckResponse{
			Name:     "CreateLoginProfileResponse",
			Metadata: awsResponseMetadata{RequestID: "req-replay-iam-create-login-profile"},
		}), nil
	case "AttachUserPolicy":
		return demoreplay.XMLResponse(req, http.StatusOK, awsAckResponse{
			Name:     "AttachUserPolicyResponse",
			Metadata: awsResponseMetadata{RequestID: "req-replay-iam-attach-user-policy"},
		}), nil
	case "DetachUserPolicy":
		return demoreplay.XMLResponse(req, http.StatusOK, awsAckResponse{
			Name:     "DetachUserPolicyResponse",
			Metadata: awsResponseMetadata{RequestID: "req-replay-iam-detach-user-policy"},
		}), nil
	case "DeleteLoginProfile":
		userName := strings.TrimSpace(form.Get("UserName"))
		t.markLoginProfile(userName, false)
		return demoreplay.XMLResponse(req, http.StatusOK, awsAckResponse{
			Name:     "DeleteLoginProfileResponse",
			Metadata: awsResponseMetadata{RequestID: "req-replay-iam-delete-login-profile"},
		}), nil
	case "DeleteUser":
		userName := strings.TrimSpace(form.Get("UserName"))
		t.deleteUser(userName)
		return demoreplay.XMLResponse(req, http.StatusOK, awsAckResponse{
			Name:     "DeleteUserResponse",
			Metadata: awsResponseMetadata{RequestID: "req-replay-iam-delete-user"},
		}), nil
	}
	return apiErrorResponse(req, http.StatusBadRequest, "InvalidAction", fmt.Sprintf("unsupported iam action: %s", action)), nil
}

func (t *transport) handleS3(req *http.Request, host string) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return s3ErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed", "the specified method is not allowed against this resource"), nil
	}
	path := strings.TrimPrefix(req.URL.Path, "/")
	query := req.URL.Query()
	if path == "" {
		resp := s3ListBucketsResponse{}
		for _, bucket := range demoS3Buckets {
			resp.Buckets = append(resp.Buckets, s3BucketWire{
				Name:         bucket.Name,
				BucketRegion: bucket.Region,
			})
		}
		return demoreplay.XMLResponse(req, http.StatusOK, resp), nil
	}

	bucketName, _, _ := strings.Cut(path, "/")
	bucket, ok := findS3Bucket(bucketName)
	if !ok {
		return s3ErrorResponse(req, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist"), nil
	}

	if query.Has("location") {
		region := bucket.Region
		if region == "us-east-1" {
			region = ""
		}
		return demoreplay.XMLResponse(req, http.StatusOK, s3LocationConstraint{Value: region}), nil
	}

	if query.Get("list-type") == "2" {
		region := regionFromHost(host)
		if region != "" && bucket.Region != region {
			return s3ErrorResponse(req, http.StatusMovedPermanently, "PermanentRedirect", "the bucket is in a different region"), nil
		}
		maxKeys := demoreplay.ParseInt(query.Get("max-keys"), 1000)
		offset := demoreplay.ParseInt(query.Get("continuation-token"), 0)
		window := demoreplay.OffsetWindow(len(bucket.Objects), offset, maxKeys)
		resp := s3ListObjectsV2Response{
			IsTruncated: window.End < len(bucket.Objects),
		}
		if resp.IsTruncated {
			resp.NextContinuationToken = strconv.Itoa(window.End)
		}
		for _, object := range bucket.Objects[window.Start:window.End] {
			resp.Contents = append(resp.Contents, s3ObjectWire{
				Key:          object.Key,
				Size:         object.Size,
				LastModified: object.LastModified,
				StorageClass: object.StorageClass,
			})
		}
		return demoreplay.XMLResponse(req, http.StatusOK, resp), nil
	}

	return s3ErrorResponse(req, http.StatusBadRequest, "InvalidRequest", "unsupported s3 replay request"), nil
}

func (t *transport) snapshotIAMUsers() []iamUserFixture {
	t.mu.Lock()
	defer t.mu.Unlock()
	users := make([]iamUserFixture, 0, len(demoIAMUsers)+len(t.createdUsers))
	for _, user := range demoIAMUsers {
		if t.deletedUsers[user.UserName] {
			continue
		}
		users = append(users, user)
	}
	for _, user := range t.createdUsers {
		if t.deletedUsers[user.UserName] {
			continue
		}
		users = append(users, user)
	}
	return users
}

func (t *transport) findUser(userName string) (iamUserFixture, bool) {
	userName = strings.TrimSpace(userName)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.deletedUsers[userName] {
		return iamUserFixture{}, false
	}
	if user, ok := t.createdUsers[userName]; ok {
		return user, true
	}
	for _, user := range demoIAMUsers {
		if user.UserName == userName {
			return user, true
		}
	}
	return iamUserFixture{}, false
}

func (t *transport) ensureUser(userName string) iamUserFixture {
	userName = strings.TrimSpace(userName)
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.deletedUsers, userName)
	if user, ok := t.createdUsers[userName]; ok {
		return user
	}
	t.sequence++
	user := iamUserFixture{
		UserName:   userName,
		UserID:     fmt.Sprintf("AIDAIOSFODNN7EXAMPLE9%02d", t.sequence),
		Arn:        "arn:aws:iam::" + demoAccountID + ":user/" + userName,
		CreateDate: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}
	t.createdUsers[userName] = user
	return user
}

func (t *transport) markLoginProfile(userName string, enabled bool) {
	userName = strings.TrimSpace(userName)
	t.mu.Lock()
	defer t.mu.Unlock()
	if user, ok := t.createdUsers[userName]; ok {
		user.HasLogin = enabled
		t.createdUsers[userName] = user
	}
}

func (t *transport) deleteUser(userName string) {
	userName = strings.TrimSpace(userName)
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.createdUsers, userName)
	t.deletedUsers[userName] = true
}

func verifyAuth(req *http.Request, body []byte) demoreplay.AuthFailureKind {
	authHeader := strings.TrimSpace(req.Header.Get("Authorization"))
	parsed, ok := parseSigV4Auth(authHeader)
	if !ok {
		return demoreplay.AuthInvalidSignature
	}
	if parsed.AccessKey != DemoAccessKeyID {
		return demoreplay.AuthInvalidAccessKey
	}
	amzDate := strings.TrimSpace(req.Header.Get("X-Amz-Date"))
	timestamp, err := time.Parse("20060102T150405Z", amzDate)
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	host := strings.TrimSpace(req.Host)
	if host == "" {
		host = req.URL.Host
	}
	extra := req.Header.Clone()
	extra.Del("Authorization")
	extra.Del("Host")
	signed, err := (api.SigV4Signer{}).Sign(auth.New(DemoAccessKeyID, DemoAccessKeySecret, ""), api.SignInput{
		Method:      req.Method,
		Service:     parsed.Service,
		Region:      parsed.Region,
		Host:        host,
		Path:        req.URL.Path,
		Query:       req.URL.Query(),
		ContentType: strings.TrimSpace(req.Header.Get("Content-Type")),
		Payload:     body,
		Timestamp:   timestamp,
		Headers:     extra,
	})
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	if !demoreplay.SubtleEqual(signed.Authorization, authHeader) {
		return demoreplay.AuthInvalidSignature
	}
	return demoreplay.AuthOK
}

type sigV4AuthHeader struct {
	AccessKey string
	ShortDate string
	Region    string
	Service   string
}

func parseSigV4Auth(value string) (sigV4AuthHeader, bool) {
	const prefix = "AWS4-HMAC-SHA256 "
	if !strings.HasPrefix(value, prefix) {
		return sigV4AuthHeader{}, false
	}
	rest := strings.TrimPrefix(value, prefix)
	parts := strings.Split(rest, ", ")
	if len(parts) < 1 {
		return sigV4AuthHeader{}, false
	}
	credPart := strings.TrimPrefix(parts[0], "Credential=")
	scope := strings.Split(credPart, "/")
	if len(scope) < 5 {
		return sigV4AuthHeader{}, false
	}
	return sigV4AuthHeader{
		AccessKey: scope[0],
		ShortDate: scope[1],
		Region:    scope[2],
		Service:   scope[3],
	}, true
}

func parseFormBody(body []byte) (url.Values, error) {
	return url.ParseQuery(string(body))
}

func regionFromHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	parts := strings.Split(host, ".")
	if len(parts) >= 4 {
		return parts[1]
	}
	return ""
}

func isSTSHost(host string) bool {
	return strings.HasPrefix(host, "sts.")
}

func isEC2Host(host string) bool {
	return strings.HasPrefix(host, "ec2.")
}

func isIAMHost(host string) bool {
	return host == "iam.amazonaws.com" || host == "iam.cn-north-1.amazonaws.com.cn"
}

func isS3Host(host string) bool {
	return strings.HasPrefix(host, "s3.")
}

type awsResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

type stsGetCallerIdentityResponse struct {
	XMLName  xml.Name                   `xml:"GetCallerIdentityResponse"`
	Result   stsGetCallerIdentityResult `xml:"GetCallerIdentityResult"`
	Metadata awsResponseMetadata        `xml:"ResponseMetadata"`
}

type stsGetCallerIdentityResult struct {
	Account string `xml:"Account"`
	Arn     string `xml:"Arn"`
	UserID  string `xml:"UserId"`
}

type ec2DescribeRegionsResponse struct {
	XMLName xml.Name        `xml:"DescribeRegionsResponse"`
	Regions []ec2RegionWire `xml:"regionInfo>item"`
}

type ec2RegionWire struct {
	Name string `xml:"regionName"`
}

type ec2DescribeInstancesResponse struct {
	XMLName      xml.Name             `xml:"DescribeInstancesResponse"`
	Reservations []ec2ReservationWire `xml:"reservationSet>item"`
}

type ec2ReservationWire struct {
	Instances []ec2InstanceWire `xml:"instancesSet>item"`
}

type ec2InstanceWire struct {
	InstanceID    string       `xml:"instanceId"`
	PublicIP      string       `xml:"ipAddress"`
	PrivateIP     string       `xml:"privateIpAddress"`
	PublicDNSName string       `xml:"dnsName"`
	State         ec2StateWire `xml:"instanceState"`
	Tags          []ec2TagWire `xml:"tagSet>item"`
}

type ec2StateWire struct {
	Name string `xml:"name"`
}

type ec2TagWire struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

type iamUserWire struct {
	UserName         string `xml:"UserName"`
	UserID           string `xml:"UserId"`
	Arn              string `xml:"Arn"`
	CreateDate       string `xml:"CreateDate,omitempty"`
	PasswordLastUsed string `xml:"PasswordLastUsed,omitempty"`
}

type iamListUsersResponse struct {
	XMLName  xml.Name             `xml:"ListUsersResponse"`
	Result   iamListUsersResult   `xml:"ListUsersResult"`
	Metadata awsResponseMetadata  `xml:"ResponseMetadata"`
}

type iamListUsersResult struct {
	Users       []iamUserWire `xml:"Users>member"`
	IsTruncated bool          `xml:"IsTruncated"`
	Marker      string        `xml:"Marker,omitempty"`
}

type iamGetLoginProfileResponse struct {
	XMLName  xml.Name                 `xml:"GetLoginProfileResponse"`
	Result   iamGetLoginProfileResult `xml:"GetLoginProfileResult"`
	Metadata awsResponseMetadata      `xml:"ResponseMetadata"`
}

type iamGetLoginProfileResult struct {
	LoginProfile iamLoginProfileWire `xml:"LoginProfile"`
}

type iamLoginProfileWire struct {
	CreateDate            string `xml:"CreateDate"`
	PasswordResetRequired bool   `xml:"PasswordResetRequired"`
}

type iamListAttachedUserPoliciesResponse struct {
	XMLName  xml.Name                          `xml:"ListAttachedUserPoliciesResponse"`
	Result   iamListAttachedUserPoliciesResult `xml:"ListAttachedUserPoliciesResult"`
	Metadata awsResponseMetadata               `xml:"ResponseMetadata"`
}

type iamListAttachedUserPoliciesResult struct {
	Policies    []iamAttachedPolicyWire `xml:"AttachedPolicies>member"`
	IsTruncated bool                    `xml:"IsTruncated"`
	Marker      string                  `xml:"Marker,omitempty"`
}

type iamAttachedPolicyWire struct {
	PolicyName string `xml:"PolicyName"`
	PolicyArn  string `xml:"PolicyArn"`
}

type iamCreateUserResponse struct {
	XMLName  xml.Name             `xml:"CreateUserResponse"`
	Result   iamCreateUserResult  `xml:"CreateUserResult"`
	Metadata awsResponseMetadata  `xml:"ResponseMetadata"`
}

type iamCreateUserResult struct {
	User iamUserWire `xml:"User"`
}

type awsAckResponse struct {
	XMLName  xml.Name            `xml:"-"`
	Name     string              `xml:"-"`
	Metadata awsResponseMetadata `xml:"ResponseMetadata"`
}

func (a awsAckResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	start := xml.StartElement{Name: xml.Name{Local: a.Name}}
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	if err := e.EncodeElement(a.Metadata, xml.StartElement{Name: xml.Name{Local: "ResponseMetadata"}}); err != nil {
		return err
	}
	return e.EncodeToken(start.End())
}

type s3ListBucketsResponse struct {
	XMLName xml.Name       `xml:"ListAllMyBucketsResult"`
	Buckets []s3BucketWire `xml:"Buckets>Bucket"`
}

type s3BucketWire struct {
	Name         string `xml:"Name"`
	BucketRegion string `xml:"BucketRegion"`
}

type s3LocationConstraint struct {
	XMLName xml.Name `xml:"LocationConstraint"`
	Value   string   `xml:",chardata"`
}

type s3ListObjectsV2Response struct {
	XMLName               xml.Name       `xml:"ListBucketResult"`
	IsTruncated           bool           `xml:"IsTruncated"`
	NextContinuationToken string         `xml:"NextContinuationToken,omitempty"`
	Contents              []s3ObjectWire `xml:"Contents"`
}

type s3ObjectWire struct {
	Key          string `xml:"Key"`
	Size         int64  `xml:"Size"`
	LastModified string `xml:"LastModified"`
	StorageClass string `xml:"StorageClass"`
}

func apiErrorResponse(req *http.Request, statusCode int, code, message string) *http.Response {
	return demoreplay.XMLResponse(req, statusCode, awsErrorEnvelope{
		Error: awsErrorBody{
			Type:    "Sender",
			Code:    code,
			Message: message,
		},
		RequestID: "req-replay-error",
	})
}

func s3ErrorResponse(req *http.Request, statusCode int, code, message string) *http.Response {
	return demoreplay.XMLResponse(req, statusCode, s3ErrorEnvelope{
		Code:      code,
		Message:   message,
		RequestID: "req-replay-s3-error",
	})
}

type awsErrorEnvelope struct {
	XMLName   xml.Name     `xml:"ErrorResponse"`
	Error     awsErrorBody `xml:"Error"`
	RequestID string       `xml:"RequestId"`
}

type awsErrorBody struct {
	Type    string `xml:"Type"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

type s3ErrorEnvelope struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	RequestID string   `xml:"RequestId"`
}
