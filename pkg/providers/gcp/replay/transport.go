package replay

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

type transport struct {
	mu       sync.Mutex
	bindings []bindingFixture
	policyEt int64
	saKeys   map[string][]saKeyFixture
}

func newTransport() *transport {
	return &transport{
		bindings: seedBindings(),
		saKeys:   seedSAKeys(),
	}
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := demoreplay.ReadRequestBody(req)
	if err != nil {
		return nil, err
	}
	host := normalizeHost(req.URL.Hostname())
	switch host {
	case "oauth2.googleapis.com":
		return t.handleToken(req, body)
	case "compute.googleapis.com":
		if !verifyBearer(req) {
			return apiErrorResponse(req, http.StatusUnauthorized, "UNAUTHENTICATED",
				"Request had invalid authentication credentials."), nil
		}
		return t.handleCompute(req)
	case "iam.googleapis.com":
		if !verifyBearer(req) {
			return apiErrorResponse(req, http.StatusUnauthorized, "UNAUTHENTICATED",
				"Request had invalid authentication credentials."), nil
		}
		return t.handleIAM(req, body)
	case "dns.googleapis.com":
		if !verifyBearer(req) {
			return apiErrorResponse(req, http.StatusUnauthorized, "UNAUTHENTICATED",
				"Request had invalid authentication credentials."), nil
		}
		return t.handleDNS(req)
	case "cloudresourcemanager.googleapis.com":
		if !verifyBearer(req) {
			return apiErrorResponse(req, http.StatusUnauthorized, "UNAUTHENTICATED",
				"Request had invalid authentication credentials."), nil
		}
		return t.handleResourceManager(req, body)
	}
	return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
		fmt.Sprintf("unsupported replay host: %s", host)), nil
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if strings.Contains(host, "://") {
		if u, err := url.Parse(host); err == nil && u.Host != "" {
			host = u.Host
		}
	}
	host = strings.TrimSuffix(host, ":443")
	host = strings.TrimSuffix(host, ":80")
	return strings.ToLower(host)
}

func verifyBearer(req *http.Request) bool {
	header := strings.TrimSpace(req.Header.Get("Authorization"))
	if !strings.HasPrefix(header, "Bearer ") {
		return false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	return demoreplay.SubtleEqual(token, demoAccessToken)
}

func (t *transport) handleToken(req *http.Request, body []byte) (*http.Response, error) {
	if !strings.HasSuffix(req.URL.Path, "/token") {
		return tokenErrorResponse(req, http.StatusNotFound, "invalid_request",
			"unsupported token endpoint path: "+req.URL.Path), nil
	}
	form, err := url.ParseQuery(string(body))
	if err != nil {
		return tokenErrorResponse(req, http.StatusBadRequest, "invalid_request", err.Error()), nil
	}
	if grant := strings.TrimSpace(form.Get("grant_type")); grant != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
		return tokenErrorResponse(req, http.StatusBadRequest, "unsupported_grant_type",
			fmt.Sprintf("unsupported grant_type: %s", grant)), nil
	}
	assertion := strings.TrimSpace(form.Get("assertion"))
	if assertion == "" {
		return tokenErrorResponse(req, http.StatusBadRequest, "invalid_request",
			"assertion is required"), nil
	}
	parts := strings.Split(assertion, ".")
	if len(parts) != 3 {
		return tokenErrorResponse(req, http.StatusUnauthorized, "invalid_grant",
			"malformed JWT assertion"), nil
	}
	resp := struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
		Scope       string `json:"scope"`
	}{
		AccessToken: demoAccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		Scope:       "https://www.googleapis.com/auth/cloud-platform",
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func tokenErrorResponse(req *http.Request, statusCode int, code, message string) *http.Response {
	payload := map[string]string{
		"error":             strings.TrimSpace(code),
		"error_description": strings.TrimSpace(message),
	}
	return demoreplay.JSONResponse(req, statusCode, payload)
}

type googleErrorBody struct {
	Error googleError `json:"error"`
}

type googleError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status,omitempty"`
}

func apiErrorResponse(req *http.Request, statusCode int, status, message string) *http.Response {
	payload := googleErrorBody{Error: googleError{
		Code:    statusCode,
		Message: strings.TrimSpace(message),
		Status:  strings.TrimSpace(status),
	}}
	return demoreplay.JSONResponse(req, statusCode, payload)
}

func (t *transport) handleCompute(req *http.Request) (*http.Response, error) {
	path := strings.Trim(req.URL.Path, "/")
	parts := strings.Split(path, "/")
	// expected forms:
	//   compute/v1/projects/{p}/zones
	//   compute/v1/projects/{p}/zones/{zone}/instances
	if len(parts) < 5 || parts[0] != "compute" || parts[1] != "v1" || parts[2] != "projects" {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("unsupported compute path: %s", path)), nil
	}
	project := parts[3]
	if project != demoProjectID {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("project %s not visible to current credentials", project)), nil
	}
	switch {
	case len(parts) == 5 && parts[4] == "zones":
		return handleListZones(req)
	case len(parts) == 7 && parts[4] == "zones" && parts[6] == "instances":
		return handleListInstances(req, parts[5])
	}
	return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
		fmt.Sprintf("unsupported compute path: %s", path)), nil
}

func handleListZones(req *http.Request) (*http.Response, error) {
	resp := api.ListZonesResponse{}
	for _, zone := range demoZones {
		resp.Items = append(resp.Items, api.Zone{Name: zone, Status: "UP"})
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func handleListInstances(req *http.Request, zone string) (*http.Response, error) {
	resp := api.ListInstancesResponse{}
	for _, inst := range instancesForZone(zone) {
		entry := api.Instance{
			Hostname: inst.Hostname,
			Name:     inst.Name,
			Zone:     inst.Zone,
			Status:   inst.Status,
			NetworkInterfaces: []api.NetworkInterface{{
				NetworkIP: inst.PrivateIP,
			}},
		}
		if inst.PublicIP != "" {
			entry.NetworkInterfaces[0].AccessConfigs = []api.AccessConfig{{NatIP: inst.PublicIP}}
		}
		resp.Items = append(resp.Items, entry)
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleIAM(req *http.Request, body []byte) (*http.Response, error) {
	path := strings.Trim(req.URL.Path, "/")
	parts := strings.Split(path, "/")
	// expected:
	//   v1/projects/{p}/serviceAccounts                      (list)
	//   v1/projects/{p}/serviceAccounts/{sa}/keys            (list / create)
	//   v1/projects/{p}/serviceAccounts/{sa}/keys/{keyId}    (delete)
	if len(parts) < 4 || parts[0] != "v1" || parts[1] != "projects" || parts[3] != "serviceAccounts" {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("unsupported iam path: %s", path)), nil
	}
	if parts[2] != demoProjectID {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("project %s not visible to current credentials", parts[2])), nil
	}
	switch len(parts) {
	case 4:
		return handleListServiceAccounts(req)
	case 5, 6, 7:
		// fall through to keys handling
	default:
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("unsupported iam path: %s", path)), nil
	}
	if len(parts) < 6 || parts[5] != "keys" {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("unsupported iam path: %s", path)), nil
	}
	saEmail := parts[4]
	sa, ok := findServiceAccount(saEmail)
	if !ok {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("service account %s not found", saEmail)), nil
	}
	switch len(parts) {
	case 6:
		switch req.Method {
		case http.MethodGet:
			return t.handleListSAKeys(req, sa)
		case http.MethodPost:
			return t.handleCreateSAKey(req, sa, body)
		}
	case 7:
		if req.Method == http.MethodDelete {
			return t.handleDeleteSAKey(req, sa, parts[6])
		}
	}
	return apiErrorResponse(req, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
		fmt.Sprintf("method %s not supported on iam path", req.Method)), nil
}

func handleListServiceAccounts(req *http.Request) (*http.Response, error) {
	resp := api.ListServiceAccountsResponse{}
	for _, sa := range demoServiceAccounts {
		resp.Accounts = append(resp.Accounts, api.ServiceAccount{
			Name:           sa.Name,
			ProjectID:      demoProjectID,
			UniqueID:       sa.UniqueID,
			Email:          sa.Email,
			DisplayName:    sa.DisplayName,
			OAuth2ClientID: sa.OAuth2ClientID,
			Disabled:       false,
		})
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleListSAKeys(req *http.Request, sa serviceAccountFixture) (*http.Response, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	resp := api.ListServiceAccountKeysResponse{}
	for _, k := range t.saKeys[sa.Email] {
		resp.Keys = append(resp.Keys, api.ServiceAccountKey{
			Name:            fmt.Sprintf("%s/keys/%s", sa.Name, k.KeyID),
			KeyType:         k.KeyType,
			KeyAlgorithm:    "KEY_ALG_RSA_2048",
			ValidAfterTime:  k.ValidAfter,
			ValidBeforeTime: k.ValidBefore,
		})
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleCreateSAKey(req *http.Request, sa serviceAccountFixture, body []byte) (*http.Response, error) {
	if len(body) > 0 {
		var payload api.CreateServiceAccountKeyRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			return apiErrorResponse(req, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error()), nil
		}
	}
	keyID, err := newSAKeyID()
	if err != nil {
		return apiErrorResponse(req, http.StatusInternalServerError, "INTERNAL", err.Error()), nil
	}
	now := time.Now().UTC()
	t.mu.Lock()
	t.saKeys[sa.Email] = append(t.saKeys[sa.Email], saKeyFixture{
		KeyID:       keyID,
		KeyType:     "USER_MANAGED",
		ValidAfter:  now.Format(time.RFC3339),
		ValidBefore: now.AddDate(2, 0, 0).Format(time.RFC3339),
	})
	t.mu.Unlock()
	demoCredJSON := buildDemoServiceAccountKeyJSON(sa.Email, keyID)
	resp := api.ServiceAccountKey{
		Name:            fmt.Sprintf("%s/keys/%s", sa.Name, keyID),
		KeyType:         "USER_MANAGED",
		KeyAlgorithm:    "KEY_ALG_RSA_2048",
		PrivateKeyType:  "TYPE_GOOGLE_CREDENTIALS_FILE",
		PrivateKeyData:  base64.StdEncoding.EncodeToString([]byte(demoCredJSON)),
		ValidAfterTime:  now.Format(time.RFC3339),
		ValidBeforeTime: now.AddDate(2, 0, 0).Format(time.RFC3339),
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleDeleteSAKey(req *http.Request, sa serviceAccountFixture, keyID string) (*http.Response, error) {
	t.mu.Lock()
	keys := t.saKeys[sa.Email]
	idx := -1
	for i, k := range keys {
		if k.KeyID == keyID {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.mu.Unlock()
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("service account key %s not found", keyID)), nil
	}
	t.saKeys[sa.Email] = append(keys[:idx], keys[idx+1:]...)
	t.mu.Unlock()
	return demoreplay.JSONResponse(req, http.StatusOK, struct{}{}), nil
}

// handleResourceManager dispatches getIamPolicy / setIamPolicy under
// cloudresourcemanager.googleapis.com.
func (t *transport) handleResourceManager(req *http.Request, body []byte) (*http.Response, error) {
	path := strings.Trim(req.URL.Path, "/")
	// expected: v1/projects/{p}:getIamPolicy or v1/projects/{p}:setIamPolicy
	if !strings.HasPrefix(path, "v1/projects/") {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("unsupported resource manager path: %s", path)), nil
	}
	tail := strings.TrimPrefix(path, "v1/projects/")
	colon := strings.LastIndex(tail, ":")
	if colon < 0 {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("unsupported resource manager path: %s", path)), nil
	}
	project := tail[:colon]
	verb := tail[colon+1:]
	if project != demoProjectID {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("project %s not visible to current credentials", project)), nil
	}
	switch verb {
	case "getIamPolicy":
		return t.handleGetIamPolicy(req)
	case "setIamPolicy":
		return t.handleSetIamPolicy(req, body)
	}
	return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
		fmt.Sprintf("unsupported resource manager verb: %s", verb)), nil
}

func (t *transport) handleGetIamPolicy(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	resp := api.IamPolicy{
		Version:  3,
		Etag:     t.currentEtag(),
		Bindings: cloneBindingsAPI(t.bindings),
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleSetIamPolicy(req *http.Request, body []byte) (*http.Response, error) {
	var payload api.SetIamPolicyRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		return apiErrorResponse(req, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error()), nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	expected := t.currentEtag()
	if strings.TrimSpace(payload.Policy.Etag) != expected {
		return apiErrorResponse(req, http.StatusConflict, "ABORTED",
			"etag does not match current policy version"), nil
	}
	bindings := make([]bindingFixture, 0, len(payload.Policy.Bindings))
	for _, b := range payload.Policy.Bindings {
		bindings = append(bindings, bindingFixture{
			Role:    b.Role,
			Members: append([]string(nil), b.Members...),
		})
	}
	t.bindings = bindings
	t.policyEt++
	resp := api.IamPolicy{
		Version:  3,
		Etag:     t.currentEtag(),
		Bindings: cloneBindingsAPI(t.bindings),
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) currentEtag() string {
	return "etag-" + strconv.FormatInt(t.policyEt, 10)
}

func cloneBindingsAPI(bindings []bindingFixture) []api.Binding {
	if len(bindings) == 0 {
		return nil
	}
	out := make([]api.Binding, len(bindings))
	for i, b := range bindings {
		out[i] = api.Binding{
			Role:    b.Role,
			Members: append([]string(nil), b.Members...),
		}
	}
	return out
}

// newSAKeyID returns a 40-char hex string mimicking GCP service account key IDs.
func newSAKeyID() (string, error) {
	var b [20]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b[:]), nil
}

// buildDemoServiceAccountKeyJSON renders a fake credentials JSON the replay
// returns from CreateKey. The key is unusable for real auth (the private key
// material is a placeholder) but the shape mirrors GCP output so callers can
// validate parsing.
func buildDemoServiceAccountKeyJSON(email, keyID string) string {
	doc := map[string]string{
		"type":           "service_account",
		"project_id":     demoProjectID,
		"private_key_id": keyID,
		"private_key":    "-----BEGIN PRIVATE KEY-----\nDEMO_REPLAY_PLACEHOLDER\n-----END PRIVATE KEY-----\n",
		"client_email":   email,
		"client_id":      "100000000000000000099",
		"auth_uri":       "https://accounts.google.com/o/oauth2/auth",
		"token_uri":      "https://oauth2.googleapis.com/token",
	}
	body, _ := json.Marshal(doc)
	return string(body)
}

func (t *transport) handleDNS(req *http.Request) (*http.Response, error) {
	path := strings.Trim(req.URL.Path, "/")
	parts := strings.Split(path, "/")
	// expected:
	//   dns/v1/projects/{p}/managedZones
	//   dns/v1/projects/{p}/managedZones/{zone}/rrsets
	if len(parts) < 5 || parts[0] != "dns" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "managedZones" {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("unsupported dns path: %s", path)), nil
	}
	if parts[3] != demoProjectID {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("project %s not visible to current credentials", parts[3])), nil
	}
	switch {
	case len(parts) == 5:
		resp := api.ListManagedZonesResponse{}
		for _, zone := range demoManagedZones {
			resp.ManagedZones = append(resp.ManagedZones, api.ManagedZone{
				Name:    zone.Name,
				DNSName: zone.DNSName,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case len(parts) == 7 && parts[6] == "rrsets":
		zoneName := parts[5]
		zone, ok := findManagedZone(zoneName)
		if !ok {
			return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
				fmt.Sprintf("managed zone %s not found", zoneName)), nil
		}
		resp := api.ListRRSetsResponse{}
		for _, record := range zone.Records {
			resp.RRSets = append(resp.RRSets, api.RRSet{
				Name:    record.Name,
				Type:    record.Type,
				RRDatas: append([]string(nil), record.RRDatas...),
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	}
	return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
		fmt.Sprintf("unsupported dns path: %s", path)), nil
}
