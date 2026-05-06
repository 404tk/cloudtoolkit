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
	mu                 sync.Mutex
	bindings           []bindingFixture
	policyEt           int64
	saKeys             map[string][]saKeyFixture
	sqlUsers           map[string][]string
	gcsPolicy          map[string]api.GCSPolicy
	instanceMetadata   map[string]api.InstanceMetadata
	instanceMetadataEt int64
}

func newTransport() *transport {
	return &transport{
		bindings:         seedBindings(),
		saKeys:           seedSAKeys(),
		sqlUsers:         make(map[string][]string),
		gcsPolicy:        make(map[string]api.GCSPolicy),
		instanceMetadata: make(map[string]api.InstanceMetadata),
	}
}

func (t *transport) snapshotGCSPolicy(bucket string) api.GCSPolicy {
	t.mu.Lock()
	defer t.mu.Unlock()
	if p, ok := t.gcsPolicy[bucket]; ok {
		return p
	}
	return api.GCSPolicy{
		Version: 1,
		Etag:    "BwYAAAAAAA=",
		Bindings: []api.GCSPolicyBind{
			{Role: "roles/storage.legacyBucketOwner", Members: []string{"projectOwner:ctk-demo-project"}},
		},
	}
}

func (t *transport) setGCSPolicy(bucket string, policy api.GCSPolicy) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.gcsPolicy[bucket] = policy
}

func (t *transport) addSQLUser(instanceID, name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sqlUsers[instanceID] = append(t.sqlUsers[instanceID], name)
}

func (t *transport) removeSQLUser(instanceID, name string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	users := t.sqlUsers[instanceID]
	for i, u := range users {
		if u == name {
			t.sqlUsers[instanceID] = append(users[:i], users[i+1:]...)
			return true
		}
	}
	return false
}

func (t *transport) snapshotSQLUsers(instanceID string) []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.sqlUsers[instanceID]...)
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
	case "logging.googleapis.com":
		if !verifyBearer(req) {
			return apiErrorResponse(req, http.StatusUnauthorized, "UNAUTHENTICATED",
				"Request had invalid authentication credentials."), nil
		}
		return t.handleLogging(req, body)
	case "sqladmin.googleapis.com":
		if !verifyBearer(req) {
			return apiErrorResponse(req, http.StatusUnauthorized, "UNAUTHENTICATED",
				"Request had invalid authentication credentials."), nil
		}
		return t.handleSQLAdmin(req, body)
	case "storage.googleapis.com":
		if !verifyBearer(req) {
			return apiErrorResponse(req, http.StatusUnauthorized, "UNAUTHENTICATED",
				"Request had invalid authentication credentials."), nil
		}
		return t.handleStorage(req, body)
	case "cloudbilling.googleapis.com":
		if !verifyBearer(req) {
			return apiErrorResponse(req, http.StatusUnauthorized, "UNAUTHENTICATED",
				"Request had invalid authentication credentials."), nil
		}
		return t.handleCloudBilling(req)
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
	case len(parts) == 7 && parts[4] == "zones" && parts[6] != "instances":
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("unsupported compute path: %s", path)), nil
	case len(parts) == 8 && parts[4] == "zones" && parts[6] == "instances":
		return t.handleGetInstance(req, parts[5], parts[7])
	case len(parts) == 9 && parts[4] == "zones" && parts[6] == "instances" && parts[8] == "setMetadata":
		return t.handleSetInstanceMetadata(req, parts[5], parts[7])
	case len(parts) == 9 && parts[4] == "zones" && parts[6] == "instances" && parts[8] == "reset":
		return t.handleResetInstance(req, parts[5], parts[7])
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

// handleGetInstance serves the GET form of `compute.instances.get` used by the
// vmexec metadata startup-script + reboot path. Only Name / Zone / Status /
// Metadata are projected — that's what the driver consumes.
func (t *transport) handleGetInstance(req *http.Request, zone, instance string) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return apiErrorResponse(req, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
			fmt.Sprintf("method %s not supported on instances.get", req.Method)), nil
	}
	inst, ok := findInstanceByZoneAndName(zone, instance)
	if !ok {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("instance %s not found in zone %s", instance, zone)), nil
	}
	resp := api.InstanceWithMetadata{
		Name:     inst.Name,
		Zone:     inst.Zone,
		Status:   inst.Status,
		Metadata: t.snapshotInstanceMetadata(inst.Name),
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

// handleSetInstanceMetadata serves `compute.instances.setMetadata`. The
// driver supplies the previously-read fingerprint to detect concurrent edits;
// the replay rejects mismatched fingerprints with the same status GCE returns
// (PRECONDITION_FAILED / 412) so the optimistic-concurrency code path is
// exercised.
func (t *transport) handleSetInstanceMetadata(req *http.Request, zone, instance string) (*http.Response, error) {
	if req.Method != http.MethodPost {
		return apiErrorResponse(req, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
			fmt.Sprintf("method %s not supported on instances.setMetadata", req.Method)), nil
	}
	if _, ok := findInstanceByZoneAndName(zone, instance); !ok {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("instance %s not found in zone %s", instance, zone)), nil
	}
	body, err := demoreplay.ReadRequestBody(req)
	if err != nil {
		return apiErrorResponse(req, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error()), nil
	}
	var payload api.InstanceMetadata
	if err := json.Unmarshal(body, &payload); err != nil {
		return apiErrorResponse(req, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error()), nil
	}
	current := t.snapshotInstanceMetadata(instance)
	if strings.TrimSpace(payload.Fingerprint) != current.Fingerprint {
		return apiErrorResponse(req, http.StatusPreconditionFailed, "FAILED_PRECONDITION",
			"fingerprint does not match the current metadata fingerprint"), nil
	}
	t.commitInstanceMetadata(instance, payload)
	return demoreplay.JSONResponse(req, http.StatusOK, api.ComputeOperation{
		Name:          fmt.Sprintf("operation-replay-setmetadata-%s", instance),
		Zone:          fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s", demoProjectID, zone),
		Status:        "DONE",
		OperationType: "setMetadata",
	}), nil
}

// handleResetInstance serves `compute.instances.reset`. The driver does not
// poll the operation, so the response just needs to surface a terminal DONE
// state with no error.
func (t *transport) handleResetInstance(req *http.Request, zone, instance string) (*http.Response, error) {
	if req.Method != http.MethodPost {
		return apiErrorResponse(req, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
			fmt.Sprintf("method %s not supported on instances.reset", req.Method)), nil
	}
	if _, ok := findInstanceByZoneAndName(zone, instance); !ok {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("instance %s not found in zone %s", instance, zone)), nil
	}
	return demoreplay.JSONResponse(req, http.StatusOK, api.ComputeOperation{
		Name:          fmt.Sprintf("operation-replay-reset-%s", instance),
		Zone:          fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s", demoProjectID, zone),
		Status:        "DONE",
		OperationType: "reset",
	}), nil
}

// snapshotInstanceMetadata returns the metadata payload an `instances.get`
// would surface for `instance`. Empty metadata maps to the seed fingerprint
// so the very first setMetadata call has something deterministic to match.
func (t *transport) snapshotInstanceMetadata(instance string) api.InstanceMetadata {
	t.mu.Lock()
	defer t.mu.Unlock()
	if md, ok := t.instanceMetadata[instance]; ok {
		return md
	}
	return api.InstanceMetadata{
		Fingerprint: t.metadataFingerprint(),
	}
}

func (t *transport) commitInstanceMetadata(instance string, payload api.InstanceMetadata) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.instanceMetadataEt++
	payload.Fingerprint = t.metadataFingerprint()
	t.instanceMetadata[instance] = payload
}

func (t *transport) metadataFingerprint() string {
	return fmt.Sprintf("metadata-fp-%d", t.instanceMetadataEt)
}

func (t *transport) handleIAM(req *http.Request, body []byte) (*http.Response, error) {
	path := strings.Trim(req.URL.Path, "/")
	parts := strings.Split(path, "/")
	// expected:
	//   v1/projects/{p}/serviceAccounts                      (list)
	//   v1/projects/{p}/serviceAccounts/{sa}                 (get / enable / disable)
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
	if len(parts) == 5 && (strings.HasSuffix(parts[4], ":enable") || strings.HasSuffix(parts[4], ":disable")) {
		// :enable / :disable verbs return an empty 200 body on success.
		return demoreplay.JSONResponse(req, http.StatusOK, struct{}{}), nil
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

// handleLogging routes Cloud Logging requests. Cloudlist `log` asset uses
// `GET /v2/projects/{p}/logs` to enumerate log names; `event-check` uses
// `POST /v2/entries:list` to read recent audit entries.
func (t *transport) handleLogging(req *http.Request, _ []byte) (*http.Response, error) {
	path := req.URL.Path
	switch {
	case req.Method == http.MethodPost && strings.HasSuffix(path, "/v2/entries:list"):
		resp := api.ListLogEntriesResponse{Entries: demoCloudAuditLogEntries()}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case req.Method == http.MethodGet && strings.HasSuffix(path, "/logs") && strings.Contains(path, "/v2/projects/"):
		project := extractLoggingProject(path)
		if project != "" && project != demoProjectID {
			return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
				fmt.Sprintf("project %s not visible to current credentials", project)), nil
		}
		resp := api.ListLogsResponse{LogNames: demoLogNames(demoProjectID)}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	}
	return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
		fmt.Sprintf("unsupported logging path: %s %s", req.Method, req.URL.Path)), nil
}

// extractLoggingProject parses `/v2/projects/{project}/logs`.
func extractLoggingProject(path string) string {
	const prefix = "/v2/projects/"
	idx := strings.Index(path, prefix)
	if idx < 0 {
		return ""
	}
	rest := path[idx+len(prefix):]
	end := strings.Index(rest, "/")
	if end < 0 {
		return rest
	}
	return rest[:end]
}

func demoLogNames(project string) []string {
	return []string{
		fmt.Sprintf("projects/%s/logs/cloudaudit.googleapis.com%%2Factivity", project),
		fmt.Sprintf("projects/%s/logs/cloudaudit.googleapis.com%%2Fdata_access", project),
		fmt.Sprintf("projects/%s/logs/run.googleapis.com%%2Fstdout", project),
	}
}

func (t *transport) handleSQLAdmin(req *http.Request, _ []byte) (*http.Response, error) {
	path := strings.TrimSuffix(req.URL.Path, "/")
	parts := strings.Split(strings.TrimPrefix(path, "/sql/v1beta4/"), "/")
	// Minimum well-formed sqladmin path is `projects/{p}/instances` (3 parts).
	// Deeper paths (`.../instances/{id}/users[...]`) are validated below.
	if len(parts) < 3 || parts[0] != "projects" || parts[2] != "instances" {
		return apiErrorResponse(req, http.StatusBadRequest, "INVALID_ARGUMENT",
			"malformed sqladmin path"), nil
	}
	if parts[1] != demoProjectID {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("project %s not visible to current credentials", parts[1])), nil
	}
	// `/sql/v1beta4/projects/{p}/instances` (no further segments) → instances.list
	if len(parts) == 3 && req.Method == http.MethodGet {
		resp := api.SQLInstancesListResponse{Kind: "sql#instancesList", Items: demoSQLInstances()}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	}
	if !strings.Contains(path, "/users") {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			"unsupported sqladmin path: "+path), nil
	}
	instanceID := parts[3]
	switch req.Method {
	case http.MethodPost:
		t.addSQLUser(instanceID, "ctkuser")
		return demoreplay.JSONResponse(req, http.StatusOK, api.SQLOperation{Name: "operation-1", Status: "DONE", OperationType: "CREATE_USER"}), nil
	case http.MethodDelete:
		name := strings.TrimSpace(req.URL.Query().Get("name"))
		if name == "" {
			return apiErrorResponse(req, http.StatusBadRequest, "INVALID_ARGUMENT", "name parameter required"), nil
		}
		if !t.removeSQLUser(instanceID, name) {
			return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
				fmt.Sprintf("user %s not found", name)), nil
		}
		return demoreplay.JSONResponse(req, http.StatusOK, api.SQLOperation{Name: "operation-2", Status: "DONE", OperationType: "DELETE_USER"}), nil
	case http.MethodGet:
		users := t.snapshotSQLUsers(instanceID)
		resp := api.SQLUsersListResponse{Kind: "sql#usersList"}
		for _, u := range users {
			resp.Items = append(resp.Items, api.SQLUser{Name: u, Instance: instanceID})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	}
	return apiErrorResponse(req, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
		"unsupported sqladmin method"), nil
}

// handleCloudBilling serves `cloudbilling.googleapis.com/v1/billingAccounts`
// (GET, list) used by the cloudlist `balance` asset on GCP.
func (t *transport) handleCloudBilling(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet || !strings.HasSuffix(req.URL.Path, "/v1/billingAccounts") {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("unsupported cloudbilling path: %s %s", req.Method, req.URL.Path)), nil
	}
	resp := api.ListBillingAccountsResponse{
		BillingAccounts: []api.BillingAccount{
			{Name: "billingAccounts/01-AAAA-BBBB", DisplayName: "Production", Open: true},
			{Name: "billingAccounts/02-CCCC-DDDD", DisplayName: "Sandbox", Open: false},
		},
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func demoSQLInstances() []api.SQLInstance {
	return []api.SQLInstance{
		{
			Name:            "ctk-demo-mysql",
			DatabaseVersion: "MYSQL_8_0",
			Region:          "us-central1",
			State:           "RUNNABLE",
			IPAddresses: []api.SQLInstanceIPAddress{{
				Type:      "PRIMARY",
				IPAddress: "203.0.113.61",
			}},
			BackendType:    "SECOND_GEN",
			InstanceType:   "CLOUD_SQL_INSTANCE",
			GceZone:        "us-central1-a",
			ConnectionName: demoProjectID + ":us-central1:ctk-demo-mysql",
			Settings: api.SQLInstanceSettings{
				Tier: "db-n1-standard-1",
				IPConfiguration: struct {
					IPv4Enabled    bool   `json:"ipv4Enabled"`
					PrivateNetwork string `json:"privateNetwork,omitempty"`
				}{IPv4Enabled: true},
			},
		},
	}
}

func demoCloudAuditLogEntries() []api.LogEntry {
	return []api.LogEntry{
		buildAuditEntry(
			"audit-evt-0001",
			"google.iam.admin.v1.CreateServiceAccountKey",
			"projects/ctk-demo-project/serviceAccounts/ctk-demo@ctk-demo-project.iam.gserviceaccount.com",
			"2026-04-22T09:11:00.000000Z",
			"ctk-demo@ctk-demo-project.iam.gserviceaccount.com",
			"203.0.113.91",
			0,
		),
		buildAuditEntry(
			"audit-evt-0002",
			"storage.buckets.update",
			"projects/_/buckets/ctk-demo-public",
			"2026-04-22T09:14:30.000000Z",
			"ctk-demo@ctk-demo-project.iam.gserviceaccount.com",
			"203.0.113.91",
			0,
		),
		buildAuditEntry(
			"audit-evt-0003",
			"google.iam.admin.v1.DeleteServiceAccountKey",
			"projects/ctk-demo-project/serviceAccounts/ctk-demo@ctk-demo-project.iam.gserviceaccount.com/keys/zzz",
			"2026-04-22T09:18:42.000000Z",
			"ctk-demo@ctk-demo-project.iam.gserviceaccount.com",
			"203.0.113.91",
			7,
		),
	}
}

func buildAuditEntry(insertID, methodName, resourceName, ts, principal, ip string, statusCode int) api.LogEntry {
	return api.LogEntry{
		InsertID:  insertID,
		LogName:   "projects/ctk-demo-project/logs/cloudaudit.googleapis.com%2Factivity",
		Timestamp: ts,
		Severity:  "NOTICE",
		Resource: api.LogEntryResource{
			Type:   "service_account",
			Labels: map[string]string{"project_id": "ctk-demo-project"},
		},
		ProtoPayload: api.LogProtoPayload{
			Type:         "type.googleapis.com/google.cloud.audit.AuditLog",
			ServiceName:  "iam.googleapis.com",
			MethodName:   methodName,
			ResourceName: resourceName,
			AuthInfo:     api.LogProtoAuthInfo{PrincipalEmail: principal},
			RequestMeta:  api.LogProtoRequestMeta{CallerIP: ip, CallerSuppliedUserAgent: "ctk/validation"},
			Status:       api.LogProtoStatus{Code: statusCode},
		},
	}
}

func (t *transport) handleStorage(req *http.Request, _ []byte) (*http.Response, error) {
	path := strings.TrimPrefix(strings.Trim(req.URL.Path, "/"), "storage/v1/")
	parts := strings.Split(path, "/")
	switch {
	case len(parts) == 1 && parts[0] == "b" && req.Method == http.MethodGet:
		resp := api.GCSBucketsListResponse{Items: demoGCSBuckets()}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case len(parts) == 3 && parts[0] == "b" && parts[2] == "o" && req.Method == http.MethodGet:
		resp := api.GCSObjectsListResponse{Items: demoGCSObjects(parts[1])}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case len(parts) == 3 && parts[0] == "b" && parts[2] == "iam" && req.Method == http.MethodGet:
		policy := t.snapshotGCSPolicy(parts[1])
		return demoreplay.JSONResponse(req, http.StatusOK, policy), nil
	case len(parts) == 3 && parts[0] == "b" && parts[2] == "iam" && req.Method == http.MethodPut:
		body, _ := demoreplay.ReadRequestBody(req)
		var policy api.GCSPolicy
		_ = json.Unmarshal(body, &policy)
		t.setGCSPolicy(parts[1], policy)
		return demoreplay.JSONResponse(req, http.StatusOK, policy), nil
	}
	return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
		fmt.Sprintf("unsupported storage path: %s %s", req.Method, req.URL.Path)), nil
}

func demoGCSBuckets() []api.GCSBucket {
	return []api.GCSBucket{
		{
			Kind:         "storage#bucket",
			ID:           "ctk-demo-public",
			Name:         "ctk-demo-public",
			StorageClass: "STANDARD",
			Location:     "US",
			TimeCreated:  "2026-04-01T08:00:00Z",
		},
		{
			Kind:         "storage#bucket",
			ID:           "ctk-demo-archive",
			Name:         "ctk-demo-archive",
			StorageClass: "COLDLINE",
			Location:     "US-CENTRAL1",
			TimeCreated:  "2026-04-02T08:00:00Z",
		},
	}
}

func demoGCSObjects(bucket string) []api.GCSObject {
	return []api.GCSObject{
		{
			Kind:         "storage#object",
			ID:           bucket + "/audit/2026-04-22.log",
			Name:         "audit/2026-04-22.log",
			Bucket:       bucket,
			StorageClass: "STANDARD",
			Size:         "12480",
			Updated:      "2026-04-22T23:59:00.000Z",
			TimeCreated:  "2026-04-22T23:59:00.000Z",
		},
		{
			Kind:         "storage#object",
			ID:           bucket + "/exports/inventory.csv",
			Name:         "exports/inventory.csv",
			Bucket:       bucket,
			StorageClass: "STANDARD",
			Size:         "1069548",
			Updated:      "2026-04-15T18:30:00.000Z",
			TimeCreated:  "2026-04-15T18:30:00.000Z",
		},
	}
}
