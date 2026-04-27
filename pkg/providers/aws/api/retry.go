package api

import "github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"

type RetryPolicy = httpclient.RetryPolicy

func DefaultRetryPolicy() RetryPolicy {
	return httpclient.DefaultRetryPolicy()
}
