package api

import (
	"net"
	"net/http"
	"time"
)

const (
	DefaultTimeout             = 30 * time.Second
	defaultDialTimeout         = 5 * time.Second
	defaultDialKeepAlive       = 30 * time.Second
	defaultTLSHandshakeTimeout = 10 * time.Second
	defaultExpectContinue      = 1 * time.Second
	defaultIdleConnTimeout     = 90 * time.Second
	defaultMaxIdleConns        = 100
	defaultMaxIdleConnsPerHost = 10
)

func NewHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyFromEnvironment
	transport.DialContext = (&net.Dialer{
		Timeout:   defaultDialTimeout,
		KeepAlive: defaultDialKeepAlive,
	}).DialContext
	transport.ForceAttemptHTTP2 = true
	transport.MaxIdleConns = defaultMaxIdleConns
	transport.MaxIdleConnsPerHost = defaultMaxIdleConnsPerHost
	transport.IdleConnTimeout = defaultIdleConnTimeout
	transport.TLSHandshakeTimeout = defaultTLSHandshakeTimeout
	transport.ExpectContinueTimeout = defaultExpectContinue

	return &http.Client{
		Transport: transport,
		Timeout:   DefaultTimeout,
	}
}
