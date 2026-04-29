package replay

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

type transport struct{}

func newTransport() *transport { return &transport{} }

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
		return t.handleIAM(req)
	case "dns.googleapis.com":
		if !verifyBearer(req) {
			return apiErrorResponse(req, http.StatusUnauthorized, "UNAUTHENTICATED",
				"Request had invalid authentication credentials."), nil
		}
		return t.handleDNS(req)
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

func (t *transport) handleIAM(req *http.Request) (*http.Response, error) {
	path := strings.Trim(req.URL.Path, "/")
	parts := strings.Split(path, "/")
	// expected: v1/projects/{p}/serviceAccounts
	if len(parts) != 4 || parts[0] != "v1" || parts[1] != "projects" || parts[3] != "serviceAccounts" {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("unsupported iam path: %s", path)), nil
	}
	if parts[2] != demoProjectID {
		return apiErrorResponse(req, http.StatusNotFound, "NOT_FOUND",
			fmt.Sprintf("project %s not visible to current credentials", parts[2])), nil
	}
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
