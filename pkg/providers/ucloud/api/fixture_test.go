package api

import "testing"

type capturedFixture struct {
	Name     string
	Input    fixtureInput
	Expected fixtureExpected
}

type fixtureInput struct {
	AccessKey     string
	SecretKey     string
	SecurityToken string
	Action        string
	Region        string
	ProjectID     string
	Params        map[string]any
}

type fixtureExpected struct {
	ContentType              string
	FormBody                 string
	FormBodyWithoutSignature string
	SortedKeys               []string
	SigningPayload           string
	StringToSign             string
	Signature                string
}

func loadSignerFixtures(t *testing.T) []capturedFixture {
	t.Helper()
	return signerFixtures
}
