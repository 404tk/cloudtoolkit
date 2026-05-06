// Package graph implements the Microsoft Graph slice that Azure's account
// inventory, iam-user, and iam-credential validation flows need.
package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/azure/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
)

const apiVersion = "v1.0"

// Client is a thin Microsoft Graph wrapper that signs requests with a
// Graph-scoped token source.
type Client struct {
	tokenSource *auth.TokenSource
	httpClient  *http.Client
	baseURL     string
}

// NewClient returns a Graph client. Callers typically build the token source
// via `auth.NewTokenSourceForScope(cred, httpClient, baseURL+".default")`.
func NewClient(ts *auth.TokenSource, httpClient *http.Client, baseURL string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://graph.microsoft.com/"
	}
	return &Client{tokenSource: ts, httpClient: httpClient, baseURL: baseURL}
}

// PasswordCredential mirrors the Microsoft Graph passwordCredential resource.
type PasswordCredential struct {
	KeyID         string `json:"keyId"`
	DisplayName   string `json:"displayName,omitempty"`
	StartDateTime string `json:"startDateTime,omitempty"`
	EndDateTime   string `json:"endDateTime,omitempty"`
	SecretText    string `json:"secretText,omitempty"`
	Hint          string `json:"hint,omitempty"`
}

// UserPasswordProfile carries the initial password assigned at user creation.
type UserPasswordProfile struct {
	Password                      string `json:"password"`
	ForceChangePasswordNextSignIn bool   `json:"forceChangePasswordNextSignIn"`
}

// User is a slim projection of the Microsoft Graph user resource for the
// validation flow's needs.
type User struct {
	ID                string               `json:"id,omitempty"`
	AccountEnabled    bool                 `json:"accountEnabled"`
	DisplayName       string               `json:"displayName"`
	MailNickname      string               `json:"mailNickname"`
	UserPrincipalName string               `json:"userPrincipalName"`
	CreatedDateTime   string               `json:"createdDateTime,omitempty"`
	SignInActivity    *SignInActivity      `json:"signInActivity,omitempty"`
	PasswordProfile   *UserPasswordProfile `json:"passwordProfile,omitempty"`
}

type SignInActivity struct {
	LastSignInDateTime string `json:"lastSignInDateTime,omitempty"`
}

// Application is a partial projection of the Graph application resource
// covering only the fields the iam-credential driver needs.
type Application struct {
	ID                  string               `json:"id"`
	DisplayName         string               `json:"displayName"`
	AppID               string               `json:"appId"`
	PasswordCredentials []PasswordCredential `json:"passwordCredentials"`
}

type applicationsListResponse struct {
	Value []Application `json:"value"`
}

type usersListResponse struct {
	Value         []User `json:"value"`
	ODataNextLink string `json:"@odata.nextLink"`
}

type addPasswordRequest struct {
	PasswordCredential PasswordCredential `json:"passwordCredential"`
}

type removePasswordRequest struct {
	KeyID string `json:"keyId"`
}

// ListPasswordCredentials returns the password credentials attached to the
// application identified by appOrObjectID. The argument may be either an
// objectId (preferred — used directly) or an appId (resolved via /applications
// filter).
func (c *Client) ListPasswordCredentials(ctx context.Context, appOrObjectID string) (Application, error) {
	app, err := c.resolveApplication(ctx, appOrObjectID)
	if err != nil {
		return Application{}, err
	}
	return app, nil
}

// AddPassword mints a fresh client secret on the named application.
// displayName is optional metadata stored alongside the credential.
func (c *Client) AddPassword(ctx context.Context, appOrObjectID, displayName string) (PasswordCredential, error) {
	app, err := c.resolveApplication(ctx, appOrObjectID)
	if err != nil {
		return PasswordCredential{}, err
	}
	body, err := json.Marshal(addPasswordRequest{
		PasswordCredential: PasswordCredential{DisplayName: displayName},
	})
	if err != nil {
		return PasswordCredential{}, err
	}
	var resp PasswordCredential
	if err := c.do(ctx, http.MethodPost,
		"/applications/"+url.PathEscape(app.ID)+"/addPassword", body, &resp); err != nil {
		return PasswordCredential{}, err
	}
	return resp, nil
}

// RemovePassword revokes a password credential by keyId.
func (c *Client) RemovePassword(ctx context.Context, appOrObjectID, keyID string) error {
	app, err := c.resolveApplication(ctx, appOrObjectID)
	if err != nil {
		return err
	}
	body, err := json.Marshal(removePasswordRequest{KeyID: keyID})
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost,
		"/applications/"+url.PathEscape(app.ID)+"/removePassword", body, nil)
}

// ListUsers enumerates Microsoft Graph users for cloudlist account inventory.
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	query := url.Values{}
	query.Set("$select", "id,displayName,userPrincipalName,accountEnabled,createdDateTime,signInActivity")
	path := "/users?" + query.Encode()
	out := make([]User, 0)
	for page := 0; page < 50; page++ {
		var resp usersListResponse
		if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
			return out, err
		}
		out = append(out, resp.Value...)
		if strings.TrimSpace(resp.ODataNextLink) == "" {
			return out, nil
		}
		nextPath, err := graphPathFromNextLink(resp.ODataNextLink)
		if err != nil {
			return out, err
		}
		path = nextPath
	}
	return out, fmt.Errorf("azure graph: user pagination exceeded 50 pages")
}

// CreateUser provisions a Microsoft Graph user (Azure AD user) with the
// supplied initial password.
func (c *Client) CreateUser(ctx context.Context, displayName, userPrincipalName, password string) (User, error) {
	body, err := json.Marshal(User{
		AccountEnabled:    true,
		DisplayName:       displayName,
		MailNickname:      mailNicknameFromUPN(userPrincipalName),
		UserPrincipalName: userPrincipalName,
		PasswordProfile: &UserPasswordProfile{
			Password:                      password,
			ForceChangePasswordNextSignIn: false,
		},
	})
	if err != nil {
		return User{}, err
	}
	var resp User
	if err := c.do(ctx, http.MethodPost, "/users", body, &resp); err != nil {
		return User{}, err
	}
	return resp, nil
}

// DeleteUser removes a Microsoft Graph user by objectId or userPrincipalName.
func (c *Client) DeleteUser(ctx context.Context, idOrUPN string) error {
	idOrUPN = strings.TrimSpace(idOrUPN)
	if idOrUPN == "" {
		return fmt.Errorf("azure graph: user id required")
	}
	return c.do(ctx, http.MethodDelete, "/users/"+url.PathEscape(idOrUPN), nil, nil)
}

func graphPathFromNextLink(nextLink string) (string, error) {
	nextLink = strings.TrimSpace(nextLink)
	if nextLink == "" {
		return "", fmt.Errorf("azure graph: empty nextLink")
	}
	if strings.HasPrefix(nextLink, "/") {
		return trimGraphVersion(nextLink), nil
	}
	parsed, err := url.Parse(nextLink)
	if err != nil {
		return "", fmt.Errorf("azure graph: parse nextLink: %w", err)
	}
	path := trimGraphVersion(parsed.EscapedPath())
	if parsed.RawQuery != "" {
		path += "?" + parsed.RawQuery
	}
	return path, nil
}

func trimGraphVersion(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	prefix := "/" + apiVersion
	if path == prefix {
		return "/"
	}
	if strings.HasPrefix(path, prefix+"/") {
		return strings.TrimPrefix(path, prefix)
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

func mailNicknameFromUPN(upn string) string {
	upn = strings.TrimSpace(upn)
	if i := strings.Index(upn, "@"); i > 0 {
		return upn[:i]
	}
	return upn
}

func (c *Client) resolveApplication(ctx context.Context, appOrObjectID string) (Application, error) {
	appOrObjectID = strings.TrimSpace(appOrObjectID)
	if appOrObjectID == "" {
		return Application{}, fmt.Errorf("azure graph: principal (application id) required")
	}
	directQuery := url.Values{}
	directQuery.Set("$select", "id,displayName,appId,passwordCredentials")
	var direct Application
	err := c.do(ctx, http.MethodGet,
		"/applications/"+url.PathEscape(appOrObjectID)+"?"+directQuery.Encode(), nil, &direct)
	if err == nil && direct.ID != "" {
		return direct, nil
	}
	listQuery := url.Values{}
	listQuery.Set("$filter", fmt.Sprintf("appId eq '%s'", appOrObjectID))
	listQuery.Set("$select", "id,displayName,appId,passwordCredentials")
	var list applicationsListResponse
	if err := c.do(ctx, http.MethodGet,
		"/applications?"+listQuery.Encode(), nil, &list); err != nil {
		return Application{}, err
	}
	if len(list.Value) == 0 {
		return Application{}, fmt.Errorf("azure graph: application %q not found", appOrObjectID)
	}
	return list.Value[0], nil
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, out any) error {
	if c.tokenSource == nil {
		return fmt.Errorf("azure graph: nil token source")
	}
	token, err := c.tokenSource.Token(ctx)
	if err != nil {
		return err
	}
	endpoint := strings.TrimRight(c.baseURL, "/") + "/" + apiVersion + path
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httpclient.CloseResponse(resp)
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("azure graph: read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("azure graph: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	return json.Unmarshal(respBody, out)
}
