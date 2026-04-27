package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
)

type Token struct {
	AccessToken string
	ExpiresAt   time.Time
}

type TokenSource struct {
	cred  Credential
	http  *http.Client
	clock func() time.Time

	mu      sync.Mutex
	current Token

	tokenURL string
}

func NewTokenSource(cred Credential, httpClient *http.Client) *TokenSource {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &TokenSource{
		cred:  cred,
		http:  httpClient,
		clock: time.Now,
	}
}

func (s *TokenSource) Token(ctx context.Context) (Token, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := s.cred.Validate(); err != nil {
		return Token{}, err
	}

	now := s.now()
	if token, ok := s.cached(now); ok {
		return token, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now = s.now()
	if token, ok := s.cachedLocked(now); ok {
		return token, nil
	}

	token, err := s.fetch(ctx, now)
	if err != nil {
		return Token{}, err
	}
	s.current = token
	return token, nil
}

func (s *TokenSource) WithScopes(scopes []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(scopes) == 0 {
		s.cred.Scopes = []string{DefaultScope}
	} else {
		s.cred.Scopes = append([]string(nil), scopes...)
	}
	s.current = Token{}
}

func (s *TokenSource) cached(now time.Time) (Token, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cachedLocked(now)
}

func (s *TokenSource) cachedLocked(now time.Time) (Token, bool) {
	if strings.TrimSpace(s.current.AccessToken) == "" {
		return Token{}, false
	}
	if !s.current.ExpiresAt.After(now.Add(60 * time.Second)) {
		return Token{}, false
	}
	return s.current, true
}

func (s *TokenSource) fetch(ctx context.Context, now time.Time) (Token, error) {
	assertion, err := SignAssertion(s.cred, now)
	if err != nil {
		return Token{}, err
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint(), strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.http.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer httpclient.CloseResponse(resp)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Token{}, fmt.Errorf("read gcp oauth2 response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var tokenErr struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if err := json.Unmarshal(body, &tokenErr); err == nil &&
			(strings.TrimSpace(tokenErr.Error) != "" || strings.TrimSpace(tokenErr.ErrorDescription) != "") {
			return Token{}, fmt.Errorf("gcp oauth2: %s: %s", strings.TrimSpace(tokenErr.Error), strings.TrimSpace(tokenErr.ErrorDescription))
		}
		return Token{}, fmt.Errorf("gcp oauth2: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return Token{}, fmt.Errorf("decode gcp oauth2 response: %w", err)
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return Token{}, fmt.Errorf("gcp oauth2: empty access_token")
	}
	if tokenResp.ExpiresIn <= 0 {
		return Token{}, fmt.Errorf("gcp oauth2: invalid expires_in")
	}

	return Token{
		AccessToken: tokenResp.AccessToken,
		ExpiresAt:   now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}

func (s *TokenSource) endpoint() string {
	if strings.TrimSpace(s.tokenURL) != "" {
		return s.tokenURL
	}
	return s.cred.TokenURI
}

func (s *TokenSource) now() time.Time {
	if s.clock != nil {
		return s.clock()
	}
	return time.Now()
}
