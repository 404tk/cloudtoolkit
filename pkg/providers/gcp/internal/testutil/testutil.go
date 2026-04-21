package testutil

import (
	"net/http"
	"net/url"
	"strings"
)

const PKCS8PrivateKeyPEM = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDRBeRY9lzO9hsC
kUneHyHVqpnTgCfGcfzqBtv2bHhv8EdxdXt6fOUHFSZWeYzt5ai4oyTZth4ShcHm
B06LmrZWdg8CVq+d9thb3BV+BrRhaQz/FCprWuhgyy6NL/FawYe5ow7BSqrfrSB7
WPtrVjTrbQ8g/unNm7FAePs3dHxzKl05TLWQEYZN+MoElQXGgi7b5rBwDdSB9OKY
3w4ODwWnDlBOPfMlhjHkacOSi9XQf3F9E9LQng85KVSwvnbe24MoKr4Z/hTPZ0S1
baXzJs8F8phJu9iL0zAbGLhYToiyUxUiE9DkbozblXLNr0rllhzVjHH6KQX9fDn0
0Gr2yZIpAgMBAAECggEAU1I6ct4OI/AF11GsNO+LEL3XYPCCqn/w1idS0pntro2F
JSy0QqD7uQWMyUbdz01Pov5hp6mJtk98eiIqhMrw6WlZVVDR47GtEH0cUicBC52R
MTNML4xG+qKz1VMprkhcPrtJm/KUR+KfApx3aJOuN7S7JaeH8s6f6zfuyG3WWB9v
xW5EagS4sK2eGQp27BjDpkutUpnKQn+OPgHk0P14Zd84oU/yEFXtBw6yYwIjO37t
+Fc9xzCJVC+SlckBk3P5cvSFNmvBT73wpbjktAdICD46BbCMtMsdWNM5az6qIW3a
DjGyMN4VNoMRsCTNOg2/UIRPf3MUSHZAc63Rnfw2mwKBgQD2+VcvzmJIsRv2sJ4r
OVdXRRGcnVFJquyBJQU+b/QG5/LYx4hY+JenrThXzUnbfm50Y4FPT1wTq78HxMa4
L8YYg36YpX9UTFAuBwWp5VxkO4vWYlm+vYLE1tb4SUrYKdU7ErSc3DtztEeKniWJ
54EVnyI891WE0fGsVBiLdF/+HwKBgQDYqXySdV8nh/o4cddtjdJzXzLrKdh8eLKF
uUluTXW6Ls6756hGyUTRjd1zsPLWPQ4jLpRto3u+d3sKMLzkXMPFNVu/M16oQQz7
Vb4Kyu0AwL8iVD8+91SgXATvDlomQBrYbZMHEVCqqVAXGjIKz8BrAp6sCOupjCMF
Mt3BBqPWtwKBgAXQkwvuGQRLHzRsrhyoafUFDEgasBpC6vSTcY8pxZ4QAfi2ofAu
UivBeU0f6ThAvssAuL+sR6ey6Hl/WYpmnYxgNC/V3ayXa1/aDHkWjFlTyZQPlrtV
7OlDgaYw25FBUuLkKtpymPe9a93IoWugxrpCl+TFkf7hjoYXKMjHwabTAoGAFF/j
4hX9i8cixboW6yuCFe1m6Wx2+kWTbDXfbOsF3itWr576WSXGPfqcT6vdOj5lnPNd
a+4Kzf+IZ43rxYHfuyToatOW3DW51czbYUJyBTcbAkxv4ij6IVZl9GEiIyS2IZI0
WF7Nei8P5AxHlnKxAp8tcrooBzqxdGSzK9rG/4MCgYEAxTmb7X2OS6f/y0v/3BMp
6i9qznlPPa93ddXzwWsBr2U5HbZPAXD78xhvW2GniujcEzptulpfzTAMfG7CGKPj
EtK6iZawt9es2yuKDzoQcAx+PsUOgqTeLAgQhsYWc1MwAeo3TWrnZB0CL8lRVdtX
YR4B4USYpM2q+fw1YCsMERY=
-----END PRIVATE KEY-----`

const PKCS1PrivateKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA0QXkWPZczvYbApFJ3h8h1aqZ04AnxnH86gbb9mx4b/BHcXV7
enzlBxUmVnmM7eWouKMk2bYeEoXB5gdOi5q2VnYPAlavnfbYW9wVfga0YWkM/xQq
a1roYMsujS/xWsGHuaMOwUqq360ge1j7a1Y0620PIP7pzZuxQHj7N3R8cypdOUy1
kBGGTfjKBJUFxoIu2+awcA3UgfTimN8ODg8Fpw5QTj3zJYYx5GnDkovV0H9xfRPS
0J4POSlUsL523tuDKCq+Gf4Uz2dEtW2l8ybPBfKYSbvYi9MwGxi4WE6IslMVIhPQ
5G6M25Vyza9K5ZYc1Yxx+ikF/Xw59NBq9smSKQIDAQABAoIBAFNSOnLeDiPwBddR
rDTvixC912Dwgqp/8NYnUtKZ7a6NhSUstEKg+7kFjMlG3c9NT6L+YaepibZPfHoi
KoTK8OlpWVVQ0eOxrRB9HFInAQudkTEzTC+MRvqis9VTKa5IXD67SZvylEfinwKc
d2iTrje0uyWnh/LOn+s37sht1lgfb8VuRGoEuLCtnhkKduwYw6ZLrVKZykJ/jj4B
5ND9eGXfOKFP8hBV7QcOsmMCIzt+7fhXPccwiVQvkpXJAZNz+XL0hTZrwU+98KW4
5LQHSAg+OgWwjLTLHVjTOWs+qiFt2g4xsjDeFTaDEbAkzToNv1CET39zFEh2QHOt
0Z38NpsCgYEA9vlXL85iSLEb9rCeKzlXV0URnJ1RSarsgSUFPm/0Bufy2MeIWPiX
p604V81J235udGOBT09cE6u/B8TGuC/GGIN+mKV/VExQLgcFqeVcZDuL1mJZvr2C
xNbW+ElK2CnVOxK0nNw7c7RHip4lieeBFZ8iPPdVhNHxrFQYi3Rf/h8CgYEA2Kl8
knVfJ4f6OHHXbY3Sc18y6ynYfHiyhblJbk11ui7Ou+eoRslE0Y3dc7Dy1j0OIy6U
baN7vnd7CjC85FzDxTVbvzNeqEEM+1W+CsrtAMC/IlQ/PvdUoFwE7w5aJkAa2G2T
BxFQqqlQFxoyCs/AawKerAjrqYwjBTLdwQaj1rcCgYAF0JML7hkESx80bK4cqGn1
BQxIGrAaQur0k3GPKcWeEAH4tqHwLlIrwXlNH+k4QL7LALi/rEensuh5f1mKZp2M
YDQv1d2sl2tf2gx5FoxZU8mUD5a7VezpQ4GmMNuRQVLi5Cracpj3vWvdyKFroMa6
QpfkxZH+4Y6GFyjIx8Gm0wKBgBRf4+IV/YvHIsW6FusrghXtZulsdvpFk2w132zr
Bd4rVq+e+lklxj36nE+r3To+ZZzzXWvuCs3/iGeN68WB37sk6GrTltw1udXM22FC
cgU3GwJMb+Io+iFWZfRhIiMktiGSNFhezXovD+QMR5ZysQKfLXK6KAc6sXRksyva
xv+DAoGBAMU5m+19jkun/8tL/9wTKeovas55Tz2vd3XV88FrAa9lOR22TwFw+/MY
b1thp4ro3BM6bbpaX80wDHxuwhij4xLSuomWsLfXrNsrig86EHAMfj7FDoKk3iwI
EIbGFnNTMAHqN01q52QdAi/JUVXbV2EeAeFEmKTNqvn8NWArDBEW
-----END RSA PRIVATE KEY-----`

const RSAPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0QXkWPZczvYbApFJ3h8h
1aqZ04AnxnH86gbb9mx4b/BHcXV7enzlBxUmVnmM7eWouKMk2bYeEoXB5gdOi5q2
VnYPAlavnfbYW9wVfga0YWkM/xQqa1roYMsujS/xWsGHuaMOwUqq360ge1j7a1Y0
620PIP7pzZuxQHj7N3R8cypdOUy1kBGGTfjKBJUFxoIu2+awcA3UgfTimN8ODg8F
pw5QTj3zJYYx5GnDkovV0H9xfRPS0J4POSlUsL523tuDKCq+Gf4Uz2dEtW2l8ybP
BfKYSbvYi9MwGxi4WE6IslMVIhPQ5G6M25Vyza9K5ZYc1Yxx+ikF/Xw59NBq9smS
KQIDAQAB
-----END PUBLIC KEY-----`

const ECPrivateKeyPEM = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIDyGpdqpAPsl+ov3u9JqbZWQG79hWOpkjRfl4RfZZFxooAoGCCqGSM49
AwEHoUQDQgAE9lthbKPvOUmBRdzVFaoMGh0LurHHCFcBPLfX6jOLNADH6eXAjQ80
prtpmvyZIUdWrHdNra6YPCHVPsgyhKe4Pg==
-----END EC PRIVATE KEY-----`

func RewriteHostsTransport(base http.RoundTripper, rawTarget string, hosts ...string) (http.RoundTripper, error) {
	if base == nil {
		base = http.DefaultTransport
	}
	target, err := url.Parse(rawTarget)
	if err != nil {
		return nil, err
	}
	items := make(map[string]struct{}, len(hosts))
	for _, host := range hosts {
		items[strings.TrimSpace(host)] = struct{}{}
	}
	return hostRewriteTransport{
		base:   base,
		target: target,
		hosts:  items,
	}, nil
}

type hostRewriteTransport struct {
	base   http.RoundTripper
	target *url.URL
	hosts  map[string]struct{}
}

func (rt hostRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if _, ok := rt.hosts[req.URL.Host]; ok {
		clone := req.Clone(req.Context())
		clone.URL.Scheme = rt.target.Scheme
		clone.URL.Host = rt.target.Host
		clone.Host = rt.target.Host
		return rt.base.RoundTrip(clone)
	}
	return rt.base.RoundTrip(req)
}
