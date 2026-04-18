package sms

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestGetResource(t *testing.T) {
	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch action := r.URL.Query().Get("Action"); action {
		case "QuerySmsSignList":
			_, _ = io.WriteString(w, `{"RequestId":"req-signs","Code":"OK","Message":"OK","SmsSignList":[{"SignName":"ctk","BusinessType":"website","AuditStatus":"AUDIT_STATE_PASS"}]}`)
		case "QuerySmsTemplateList":
			_, _ = io.WriteString(w, `{"RequestId":"req-templates","Code":"OK","Message":"OK","SmsTemplateList":[{"TemplateName":"welcome","AuditStatus":"AUDIT_STATE_NOT_PASS","TemplateContent":"hello"}]}`)
		case "QuerySendStatistics":
			if got := r.URL.Query().Get("StartDate"); got != "20260418" {
				t.Fatalf("unexpected start date: %s", got)
			}
			if got := r.URL.Query().Get("EndDate"); got != "20260418" {
				t.Fatalf("unexpected end date: %s", got)
			}
			_, _ = io.WriteString(w, `{"RequestId":"req-stat","Code":"OK","Message":"OK","Data":{"TotalSize":42}}`)
		default:
			t.Fatalf("unexpected action: %s", action)
		}
	}))
	defer server.Close()

	driver := Driver{
		Cred:   aliauth.New("ak", "sk", ""),
		Region: "all",
		clientOptions: []api.Option{
			api.WithBaseURL(server.URL),
			api.WithClock(func() time.Time { return time.Unix(1713376800, 0).UTC() }),
			api.WithNonce(func() string { return "nonce" }),
		},
		now: func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) },
	}

	res, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if len(res.Signs) != 1 || res.Signs[0].Name != "ctk" || res.Signs[0].Status != "审核通过" {
		t.Fatalf("unexpected signs: %+v", res.Signs)
	}
	if len(res.Templates) != 1 || res.Templates[0].Name != "welcome" || res.Templates[0].Status != "审核未通过" {
		t.Fatalf("unexpected templates: %+v", res.Templates)
	}
	if res.DailySize != 42 {
		t.Fatalf("unexpected daily size: %d", res.DailySize)
	}
}
