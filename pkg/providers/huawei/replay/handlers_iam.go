package replay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

func (t *transport) handleIAM(req *http.Request, _ string, body []byte) (*http.Response, error) {
	path := req.URL.Path
	method := strings.ToUpper(req.Method)
	switch {
	case method == http.MethodGet && strings.HasPrefix(path, "/v3.0/OS-CREDENTIAL/credentials/"):
		return t.handleShowPermanentAccessKey(req, path)
	case method == http.MethodGet && strings.HasPrefix(path, "/v3/users/") && !strings.Contains(strings.TrimPrefix(path, "/v3/users/"), "/"):
		return t.handleShowUser(req, path)
	case method == http.MethodGet && path == "/v3/auth/domains":
		return handleListDomains(req)
	case method == http.MethodGet && path == "/v3/projects":
		return handleListProjects(req)
	case method == http.MethodGet && path == "/v3/regions":
		return handleListRegions(req)
	case method == http.MethodGet && path == "/v5/users":
		return t.handleListUsersV5(req)
	case method == http.MethodGet && path == "/v3/groups":
		return handleListGroups(req)
	case method == http.MethodPost && path == "/v3/users":
		return t.handleCreateUser(req, body)
	case method == http.MethodDelete && strings.HasPrefix(path, "/v3/users/"):
		return t.handleDeleteUser(req, path)
	case method == http.MethodPut && strings.HasPrefix(path, "/v3/groups/") && strings.Contains(path, "/users/"):
		return t.handleAddUserToGroup(req, path)
	}
	return apiErrorResponse(req, http.StatusNotFound, "IAM.0001",
		fmt.Sprintf("unsupported iam path: %s %s", method, path)), nil
}

func (t *transport) handleShowPermanentAccessKey(req *http.Request, path string) (*http.Response, error) {
	ak := strings.TrimSpace(strings.TrimPrefix(path, "/v3.0/OS-CREDENTIAL/credentials/"))
	if ak != demoCredentials.AccessKey {
		return apiErrorResponse(req, http.StatusNotFound, "IAM.0007",
			fmt.Sprintf("access key %s not found", ak)), nil
	}
	resp := api.ShowPermanentAccessKeyResponse{}
	resp.Credential.UserID = demoUserID
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleShowUser(req *http.Request, path string) (*http.Response, error) {
	id := strings.TrimSpace(strings.TrimPrefix(path, "/v3/users/"))
	user, ok := t.iam.findByID(id)
	if !ok {
		return apiErrorResponse(req, http.StatusNotFound, "IAM.0009",
			fmt.Sprintf("user %s not found", id)), nil
	}
	resp := api.ShowUserResponse{}
	resp.User.ID = user.ID
	resp.User.Name = user.Name
	resp.User.DomainID = user.DomainID
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func handleListDomains(req *http.Request) (*http.Response, error) {
	resp := api.ListAuthDomainsResponse{
		Domains: []api.IAMDomain{{ID: demoDomainID, Name: demoDomainName}},
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func handleListProjects(req *http.Request) (*http.Response, error) {
	name := strings.TrimSpace(req.URL.Query().Get("name"))
	resp := api.ListProjectsResponse{}
	if name == "" {
		for _, project := range demoProjects {
			resp.Projects = append(resp.Projects, api.IAMProject{
				ID: project.ID, Name: project.Name, DomainID: project.DomainID, Enabled: true,
			})
		}
	} else if project, ok := findProject(name); ok {
		resp.Projects = append(resp.Projects, api.IAMProject{
			ID: project.ID, Name: project.Name, DomainID: project.DomainID, Enabled: true,
		})
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func handleListRegions(req *http.Request) (*http.Response, error) {
	resp := api.ListRegionsResponse{}
	for _, region := range demoRegions {
		resp.Regions = append(resp.Regions, api.Region{ID: region.ID})
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleListUsersV5(req *http.Request) (*http.Response, error) {
	resp := api.ListUsersV5Response{}
	for _, user := range t.iam.snapshotUsers() {
		resp.Users = append(resp.Users, api.IAMUserV5{
			UserID:   user.ID,
			UserName: user.Name,
			Enabled:  user.Enabled,
		})
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func handleListGroups(req *http.Request) (*http.Response, error) {
	resp := api.ListGroupsResponse{}
	for _, group := range demoIAMGroups {
		resp.Groups = append(resp.Groups, api.IAMGroup{ID: group.ID, Name: group.Name})
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleCreateUser(req *http.Request, body []byte) (*http.Response, error) {
	var payload api.CreateUserRequest
	_ = json.Unmarshal(body, &payload)
	name := strings.TrimSpace(payload.User.Name)
	if name == "" {
		return apiErrorResponse(req, http.StatusBadRequest, "IAM.0002", "user name is required"), nil
	}
	user := t.iam.ensureUser(name)
	resp := api.CreateUserResponse{}
	resp.User.ID = user.ID
	resp.User.DomainID = user.DomainID
	return demoreplay.JSONResponse(req, http.StatusCreated, resp), nil
}

func (t *transport) handleDeleteUser(req *http.Request, path string) (*http.Response, error) {
	id := strings.TrimSpace(strings.TrimPrefix(path, "/v3/users/"))
	if _, ok := t.iam.findByID(id); !ok {
		return apiErrorResponse(req, http.StatusNotFound, "IAM.0009",
			fmt.Sprintf("user %s not found", id)), nil
	}
	t.iam.deleteByID(id)
	return demoreplay.JSONResponse(req, http.StatusNoContent, struct{}{}), nil
}

func (t *transport) handleAddUserToGroup(req *http.Request, path string) (*http.Response, error) {
	rest := strings.TrimPrefix(path, "/v3/groups/")
	parts := strings.SplitN(rest, "/users/", 2)
	if len(parts) != 2 {
		return apiErrorResponse(req, http.StatusBadRequest, "IAM.0002",
			fmt.Sprintf("malformed group membership path: %s", path)), nil
	}
	t.iam.recordGroupMembership(parts[0], parts[1])
	return demoreplay.JSONResponse(req, http.StatusNoContent, struct{}{}), nil
}
