package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
)

func TestClientDescribeECSInstancesUsesLocationResolvedEndpoint(t *testing.T) {
	t.Parallel()

	client := NewClient(
		auth.New("ak", "sk", ""),
		WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch req.URL.Host {
				case locationReadonlyEndpoint:
					if got := req.URL.Query().Get("ServiceCode"); got != "ecs" {
						t.Fatalf("unexpected location service code: %s", got)
					}
					if got := req.URL.Query().Get("Id"); got != "cn-guangzhou" {
						t.Fatalf("unexpected location region: %s", got)
					}
					if got := req.URL.Query().Get("RegionId"); got != "" {
						t.Fatalf("unexpected location RegionId: %s", got)
					}
					return jsonResponse(http.StatusOK, `{"Success":true,"Endpoints":{"Endpoint":[{"Endpoint":"ecs.cn-guangzhou.aliyuncs.com"}]}}`), nil
				case "ecs.cn-guangzhou.aliyuncs.com":
					if got := req.URL.Query().Get("Action"); got != "DescribeInstances" {
						t.Fatalf("unexpected ecs action: %s", got)
					}
					return jsonResponse(http.StatusOK, `{"PageSize":100,"PageNumber":1,"TotalCount":0,"Instances":{"Instance":[]}}`), nil
				default:
					t.Fatalf("unexpected host: %s", req.URL.Host)
					return nil, nil
				}
			}),
		}),
		WithClock(func() time.Time { return time.Unix(1713376800, 0).UTC() }),
		WithNonce(func() string { return "nonce" }),
	)

	if _, err := client.DescribeECSInstances(context.Background(), "cn-guangzhou", 1, 100); err != nil {
		t.Fatalf("DescribeECSInstances() error = %v", err)
	}
}

func TestClientDescribeECSInstancesFallsBackToGlobalEndpointWhenLocationMisses(t *testing.T) {
	t.Parallel()

	client := NewClient(
		auth.New("ak", "sk", ""),
		WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch req.URL.Host {
				case locationReadonlyEndpoint:
					return jsonResponse(http.StatusOK, `{"Success":true,"Endpoints":{"Endpoint":[]}}`), nil
				case ecsGlobalEndpoint:
					return jsonResponse(http.StatusOK, `{"PageSize":100,"PageNumber":1,"TotalCount":0,"Instances":{"Instance":[]}}`), nil
				default:
					t.Fatalf("unexpected host: %s", req.URL.Host)
					return nil, nil
				}
			}),
		}),
		WithClock(func() time.Time { return time.Unix(1713376800, 0).UTC() }),
		WithNonce(func() string { return "nonce" }),
	)

	if _, err := client.DescribeECSInstances(context.Background(), "cn-fuzhou", 1, 100); err != nil {
		t.Fatalf("DescribeECSInstances() error = %v", err)
	}
}

func TestClientDescribeECSInstancesUsesStaticRegionalEndpoint(t *testing.T) {
	t.Parallel()

	client := NewClient(
		auth.New("ak", "sk", ""),
		WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Host != "ecs.ap-northeast-1.aliyuncs.com" {
					t.Fatalf("unexpected host: %s", req.URL.Host)
				}
				if strings.Contains(req.URL.RawQuery, "ServiceCode=ecs") {
					t.Fatal("location resolver should not be used for static regional endpoint")
				}
				return jsonResponse(http.StatusOK, `{"PageSize":100,"PageNumber":1,"TotalCount":0,"Instances":{"Instance":[]}}`), nil
			}),
		}),
		WithClock(func() time.Time { return time.Unix(1713376800, 0).UTC() }),
		WithNonce(func() string { return "nonce" }),
	)

	if _, err := client.DescribeECSInstances(context.Background(), "ap-northeast-1", 1, 100); err != nil {
		t.Fatalf("DescribeECSInstances() error = %v", err)
	}
}

func TestClientDescribeECSInstancesRetriesWithLocationAfterNotSupportedEndpoint(t *testing.T) {
	t.Parallel()

	var initialAttempts int
	client := NewClient(
		auth.New("ak", "sk", ""),
		WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch req.URL.Host {
				case "ecs-cn-hangzhou.aliyuncs.com":
					initialAttempts++
					return jsonResponse(http.StatusNotFound, `{"Code":"InvalidOperation.NotSupportedEndpoint","Message":"The specified endpoint cant operate this region.","RequestId":"req-1"}`), nil
				case locationReadonlyEndpoint:
					if got := req.URL.Query().Get("Id"); got != "ap-southeast-1" {
						t.Fatalf("unexpected location region: %s", got)
					}
					return jsonResponse(http.StatusOK, `{"Success":true,"Endpoints":{"Endpoint":[{"Endpoint":"ecs.ap-southeast-1.aliyuncs.com"}]}}`), nil
				case "ecs.ap-southeast-1.aliyuncs.com":
					return jsonResponse(http.StatusOK, `{"PageSize":100,"PageNumber":1,"TotalCount":0,"Instances":{"Instance":[]}}`), nil
				default:
					t.Fatalf("unexpected host: %s", req.URL.Host)
					return nil, nil
				}
			}),
		}),
		WithClock(func() time.Time { return time.Unix(1713376800, 0).UTC() }),
		WithNonce(func() string { return "nonce" }),
	)

	if _, err := client.DescribeECSInstances(context.Background(), "ap-southeast-1", 1, 100); err != nil {
		t.Fatalf("DescribeECSInstances() error = %v", err)
	}
	if initialAttempts != 1 {
		t.Fatalf("unexpected initial attempts: %d", initialAttempts)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
