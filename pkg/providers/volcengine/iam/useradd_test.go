package iam

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDriverAddUserCreatesLoginProfileAndAttachesAdminPolicy(t *testing.T) {
	var actions []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			t.Fatalf("parse query: %v", err)
		}
		action := values.Get("Action")
		actions = append(actions, action)

		switch action {
		case "CreateUser":
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			if values.Get("UserName") != "demo-user" {
				t.Fatalf("unexpected username: %s", values.Get("UserName"))
			}
			if values.Get("DisplayName") != "demo-user" {
				t.Fatalf("unexpected display name: %s", values.Get("DisplayName"))
			}
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-create-user"},"Result":{"User":{"UserName":"demo-user","AccountId":1001,"CreateDate":"20260419T120000Z"}}}`))
		case "CreateLoginProfile":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			got := strings.TrimSpace(string(body))
			want := `{"UserName":"demo-user","Password":"TempPassw0rd!","LoginAllowed":true,"PasswordResetRequired":false}`
			if got != want {
				t.Fatalf("unexpected login profile body: %s", got)
			}
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-create-profile"},"Result":{"LoginProfile":{"UserName":"demo-user","LoginAllowed":true,"PasswordResetRequired":false}}}`))
		case "AttachUserPolicy":
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			if values.Get("UserName") != "demo-user" || values.Get("PolicyName") != "AdministratorAccess" || values.Get("PolicyType") != "System" {
				t.Fatalf("unexpected policy args: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-attach-policy"}}`))
		case "ListProjects":
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-projects"},"Result":{"Projects":[{"ProjectName":"demo","AccountID":1234567890}],"Total":1}}`))
		default:
			t.Fatalf("unexpected action: %s", action)
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client:   newTestClient(server.URL),
		Region:   "",
		UserName: "demo-user",
		Password: "TempPassw0rd!",
	}
	driver.AddUser()

	got := strings.Join(actions, ",")
	if got != "CreateUser,CreateLoginProfile,AttachUserPolicy,ListProjects" {
		t.Fatalf("unexpected actions: %s", got)
	}
}

func TestDriverDelUserIgnoresMissingAttachmentAndLoginProfile(t *testing.T) {
	var actions []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			t.Fatalf("parse query: %v", err)
		}
		action := values.Get("Action")
		actions = append(actions, action)

		switch action {
		case "DetachUserPolicy":
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-detach","Error":{"Code":"EntityNotExist.PolicyAttachment","Message":"attachment not found"}}}`))
		case "DeleteLoginProfile":
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-profile","Error":{"Code":"EntityNotExist.LoginProfile","Message":"login profile not found"}}}`))
		case "DeleteUser":
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			if values.Get("UserName") != "demo-user" {
				t.Fatalf("unexpected username: %s", values.Get("UserName"))
			}
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-delete-user"}}`))
		default:
			t.Fatalf("unexpected action: %s", action)
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client:   newTestClient(server.URL),
		Region:   "",
		UserName: "demo-user",
	}
	driver.DelUser()

	got := strings.Join(actions, ",")
	if got != "DetachUserPolicy,DeleteLoginProfile,DeleteUser" {
		t.Fatalf("unexpected actions: %s", got)
	}
}
