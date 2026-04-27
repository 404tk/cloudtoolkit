package iam

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestListUsersWithPaginationLoginStateAndPolicies(t *testing.T) {
	next := env.Active().Clone()
	next.ListPolicies = true
	env.SetActiveForTest(t, next)

	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch action := r.URL.Query().Get("Action"); action {
		case "ListUsers":
			switch r.URL.Query().Get("Marker") {
			case "":
				_, _ = io.WriteString(w, `{"RequestId":"req-users-1","IsTruncated":true,"Marker":"page-2","Users":{"User":[{"UserName":"alice","UserId":"u-1","CreateDate":"2026-04-18T10:00:00Z"}]}}`)
			case "page-2":
				_, _ = io.WriteString(w, `{"RequestId":"req-users-2","IsTruncated":false,"Marker":"","Users":{"User":[{"UserName":"bob","UserId":"u-2","CreateDate":"2026-04-18T11:00:00Z"}]}}`)
			default:
				t.Fatalf("unexpected marker: %s", r.URL.Query().Get("Marker"))
			}
		case "GetLoginProfile":
			switch r.URL.Query().Get("UserName") {
			case "alice":
				_, _ = io.WriteString(w, `{"RequestId":"req-login-alice","LoginProfile":{"UserName":"alice","CreateDate":"2026-04-18T10:05:00Z"}}`)
			case "bob":
				w.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(w, `{"Code":"EntityNotExist.LoginProfile","Message":"login policy not exists","RequestId":"req-login-bob"}`)
			default:
				t.Fatalf("unexpected login profile user: %s", r.URL.Query().Get("UserName"))
			}
		case "GetUser":
			if got := r.URL.Query().Get("UserName"); got != "alice" {
				t.Fatalf("unexpected get user request: %s", got)
			}
			_, _ = io.WriteString(w, `{"RequestId":"req-get-user","User":{"UserName":"alice","UserId":"u-1","LastLoginDate":"2026-04-18T12:00:00Z"}}`)
		case "ListPoliciesForUser":
			switch r.URL.Query().Get("UserName") {
			case "alice":
				_, _ = io.WriteString(w, `{"RequestId":"req-policies-alice","Policies":{"Policy":[{"PolicyName":"AdministratorAccess","PolicyType":"System"},{"PolicyName":"CustomReadOnly","PolicyType":"Custom"}]}}`)
			case "bob":
				_, _ = io.WriteString(w, `{"RequestId":"req-policies-bob","Policies":{"Policy":[{"PolicyName":"ReadOnlyAccess","PolicyType":"System"}]}}`)
			default:
				t.Fatalf("unexpected policy user: %s", r.URL.Query().Get("UserName"))
			}
		case "GetPolicy":
			if got := r.URL.Query().Get("PolicyName"); got != "CustomReadOnly" {
				t.Fatalf("unexpected policy name: %s", got)
			}
			_, _ = io.WriteString(w, `{"RequestId":"req-policy","DefaultPolicyVersion":{"PolicyDocument":"{\"Statement\":[]}"}}`)
		default:
			t.Fatalf("unexpected action: %s", action)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	users, err := driver.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("unexpected user count: %d", len(users))
	}

	if users[0].UserName != "alice" || users[0].UserId != "u-1" {
		t.Fatalf("unexpected first user: %+v", users[0])
	}
	if !users[0].EnableLogin {
		t.Fatalf("expected alice login enabled: %+v", users[0])
	}
	if !strings.Contains(users[0].Policies, "AdministratorAccess") || !strings.Contains(users[0].Policies, "CustomReadOnly") {
		t.Fatalf("unexpected alice policies: %q", users[0].Policies)
	}
	if users[0].LastLogin == "" {
		t.Fatalf("expected alice last login: %+v", users[0])
	}

	if users[1].UserName != "bob" || users[1].UserId != "u-2" {
		t.Fatalf("unexpected second user: %+v", users[1])
	}
	if users[1].EnableLogin {
		t.Fatalf("expected bob login disabled: %+v", users[1])
	}
	if users[1].LastLogin != "" {
		t.Fatalf("expected empty bob last login: %+v", users[1])
	}
	if users[1].Policies != "ReadOnlyAccess" {
		t.Fatalf("unexpected bob policies: %q", users[1].Policies)
	}
}

func TestAddUserCreatesLoginProfileAndPrintsLoginURL(t *testing.T) {
	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch action := r.URL.Query().Get("Action"); action {
		case "CreateUser":
			if got := r.URL.Query().Get("UserName"); got != "alice" {
				t.Fatalf("unexpected create user name: %s", got)
			}
			_, _ = io.WriteString(w, `{"RequestId":"req-create-user","User":{"UserName":"alice","UserId":"u-1"}}`)
		case "CreateLoginProfile":
			if got := r.URL.Query().Get("Password"); got != "Secret!1" {
				t.Fatalf("unexpected password: %s", got)
			}
			_, _ = io.WriteString(w, `{"RequestId":"req-create-login","LoginProfile":{"UserName":"alice"}}`)
		case "AttachPolicyToUser":
			if got := r.URL.Query().Get("PolicyName"); got != "AdministratorAccess" {
				t.Fatalf("unexpected user policy: %s", got)
			}
			_, _ = io.WriteString(w, `{"RequestId":"req-attach-user"}`)
		case "GetAccountAlias":
			_, _ = io.WriteString(w, `{"RequestId":"req-alias","AccountAlias":"demo-account"}`)
		default:
			t.Fatalf("unexpected action: %s", action)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	driver.UserName = "alice"
	driver.Password = "Secret!1"

	output := captureStdout(t, func() {
		driver.AddUser()
	})
	if !strings.Contains(output, "alice") {
		t.Fatalf("expected username in output: %s", output)
	}
	if !strings.Contains(output, "Secret!1") {
		t.Fatalf("expected password in output: %s", output)
	}
	if !strings.Contains(output, "https://signin.aliyun.com/demo-account/login.htm") {
		t.Fatalf("expected login url in output: %s", output)
	}
}

func TestDelUserIgnoresMissingAttachmentAndDeletesUser(t *testing.T) {
	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("Action")
		actions = append(actions, action)
		switch action {
		case "DetachPolicyFromUser":
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"Code":"EntityNotExist.User","Message":"user not found","RequestId":"req-detach-user"}`)
		case "DeleteUser":
			_, _ = io.WriteString(w, `{"RequestId":"req-delete-user"}`)
		default:
			t.Fatalf("unexpected action: %s", action)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	driver.UserName = "alice"
	driver.DelUser()

	if strings.Join(actions, ",") != "DetachPolicyFromUser,DeleteUser" {
		t.Fatalf("unexpected action sequence: %v", actions)
	}
}

func TestAddRoleCreatesRoleAndPrintsSwitchURL(t *testing.T) {
	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch action := r.URL.Query().Get("Action"); action {
		case "CreateRole":
			if got := r.URL.Query().Get("RoleName"); got != "auditor" {
				t.Fatalf("unexpected role name: %s", got)
			}
			if got := r.URL.Query().Get("AssumeRolePolicyDocument"); !strings.Contains(got, "1234567890123456") {
				t.Fatalf("unexpected assume role policy: %s", got)
			}
			_, _ = io.WriteString(w, `{"RequestId":"req-create-role","Role":{"RoleName":"auditor","RoleId":"r-1"}}`)
		case "AttachPolicyToRole":
			if got := r.URL.Query().Get("PolicyName"); got != "AdministratorAccess" {
				t.Fatalf("unexpected role policy: %s", got)
			}
			_, _ = io.WriteString(w, `{"RequestId":"req-attach-role"}`)
		case "GetAccountAlias":
			_, _ = io.WriteString(w, `{"RequestId":"req-alias","AccountAlias":"demo-account"}`)
		default:
			t.Fatalf("unexpected action: %s", action)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	driver.RoleName = "auditor"
	driver.AccountId = "1234567890123456"

	output := captureStdout(t, func() {
		driver.AddRole()
	})
	if !strings.Contains(output, "demo-account") {
		t.Fatalf("expected account alias in output: %s", output)
	}
	if !strings.Contains(output, "auditor") {
		t.Fatalf("expected role name in output: %s", output)
	}
	if !strings.Contains(output, "https://signin.aliyun.com/switchRole.htm") {
		t.Fatalf("expected switch url in output: %s", output)
	}
}

func TestDelRoleDetachesPolicyAndDeletesRole(t *testing.T) {
	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("Action")
		actions = append(actions, action)
		switch action {
		case "DetachPolicyFromRole":
			_, _ = io.WriteString(w, `{"RequestId":"req-detach-role"}`)
		case "DeleteRole":
			_, _ = io.WriteString(w, `{"RequestId":"req-delete-role"}`)
		default:
			t.Fatalf("unexpected action: %s", action)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	driver.RoleName = "auditor"
	driver.DelRole()

	if strings.Join(actions, ",") != "DetachPolicyFromRole,DeleteRole" {
		t.Fatalf("unexpected action sequence: %v", actions)
	}
}

func newTestDriver(baseURL string) Driver {
	return Driver{
		Cred:   aliauth.New("ak", "sk", ""),
		Region: "all",
		clientOptions: []api.Option{
			api.WithBaseURL(baseURL),
			api.WithClock(func() time.Time { return time.Unix(1713376800, 0).UTC() }),
			api.WithNonce(func() string { return "nonce" }),
		},
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer

	done := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(reader)
		done <- string(data)
	}()

	fn()

	_ = writer.Close()
	os.Stdout = originalStdout
	return <-done
}
