package rds

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
)

func TestCreateAccountSendsExpectedPayload(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	transport := &routingTransport{
		t: t,
		routes: map[string]routeResponse{
			"GET iam.cn-north-4.myhuaweicloud.com /v3/projects?name=cn-north-4": {
				body: `{"projects":[{"id":"proj-n4","name":"cn-north-4","domain_id":"d-1","enabled":true}]}`,
			},
			"POST rds.cn-north-4.myhuaweicloud.com /v3/proj-n4/instances/db-1/db_user?": {
				body: `{"resp":"successful"}`,
			},
		},
	}
	driver := newTestDriver([]string{"cn-north-4"}, "d-1", transport)
	res, err := driver.CreateAccount(context.Background(), "cn-north-4", "db-1")
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if res.Username != "ctkuser" || res.Password != "Ctk!Pwd2026" {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestDeleteAccountSendsExpectedPayload(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	transport := &routingTransport{
		t: t,
		routes: map[string]routeResponse{
			"GET iam.cn-north-4.myhuaweicloud.com /v3/projects?name=cn-north-4": {
				body: `{"projects":[{"id":"proj-n4","name":"cn-north-4","domain_id":"d-1","enabled":true}]}`,
			},
			"DELETE rds.cn-north-4.myhuaweicloud.com /v3/proj-n4/instances/db-1/db_user/ctkuser?": {
				body: `{"resp":"successful"}`,
			},
		},
	}
	driver := newTestDriver([]string{"cn-north-4"}, "d-1", transport)
	res, err := driver.DeleteAccount(context.Background(), "cn-north-4", "db-1")
	if err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if res.Username != "ctkuser" {
		t.Errorf("unexpected username: %s", res.Username)
	}
}

func TestCreateAccountRejectsMissingConfig(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{})
	transport := &routingTransport{t: t, routes: map[string]routeResponse{}}
	driver := newTestDriver([]string{"cn-north-4"}, "d-1", transport)
	if _, err := driver.CreateAccount(context.Background(), "cn-north-4", "db-1"); err == nil {
		t.Fatalf("expected error for missing rds-account-check")
	}
}

func TestDeleteAccountPropagatesAPIError(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	transport := &routingTransport{
		t: t,
		routes: map[string]routeResponse{
			"GET iam.cn-north-4.myhuaweicloud.com /v3/projects?name=cn-north-4": {
				body: `{"projects":[{"id":"proj-n4","name":"cn-north-4","domain_id":"d-1","enabled":true}]}`,
			},
			"DELETE rds.cn-north-4.myhuaweicloud.com /v3/proj-n4/instances/db-1/db_user/ctkuser?": {
				statusCode: http.StatusNotFound,
				body:       `{"error_code":"DBS.200013","error_msg":"user not found"}`,
			},
		},
	}
	driver := newTestDriver([]string{"cn-north-4"}, "d-1", transport)
	if _, err := driver.DeleteAccount(context.Background(), "cn-north-4", "db-1"); err == nil {
		t.Fatalf("expected error from DeleteAccount")
	} else if !strings.Contains(err.Error(), "DBS.200013") {
		t.Errorf("expected DBS.200013, got %v", err)
	}
}
