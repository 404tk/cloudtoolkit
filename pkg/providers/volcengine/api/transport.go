package api

import (
	"net"
	"net/http"
	"time"
)

const (
	DefaultTimeout               = 30 * time.Second
	defaultDialTimeout           = 5 * time.Second
	defaultDialKeepAlive         = 30 * time.Second
	defaultTLSHandshakeTimeout   = 5 * time.Second
	defaultResponseHeaderTimeout = 15 * time.Second
	defaultExpectContinueTimeout = 1 * time.Second
	defaultIdleConnTimeout       = 90 * time.Second
	defaultMaxIdleConns          = 64
	defaultMaxIdleConnsPerHost   = 16
)

func NewTransport() *http.Transport {
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
	transport.ExpectContinueTimeout = defaultExpectContinueTimeout
	transport.ResponseHeaderTimeout = defaultResponseHeaderTimeout
	return transport
}

func NewHTTPClient() *http.Client {
	return &http.Client{
		Transport: NewTransport(),
		Timeout:   DefaultTimeout,
	}
}
