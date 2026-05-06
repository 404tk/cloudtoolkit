package replay

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type iamMutationState struct {
	mu           sync.Mutex
	created      map[string]subUserFixture
	deleted      map[string]bool
	policies     map[string]map[string]bool
	accessKeys   map[string][]jdcloudAccessKeyFixture
	accessKeySeq int
}

func newIAMState() *iamMutationState {
	return &iamMutationState{
		created:    make(map[string]subUserFixture),
		deleted:    make(map[string]bool),
		policies:   make(map[string]map[string]bool),
		accessKeys: make(map[string][]jdcloudAccessKeyFixture),
	}
}

type jdcloudAccessKeyFixture struct {
	AccessKey  string
	SecretKey  string
	Status     string
	CreateTime string
}

func (s *iamMutationState) snapshotUsers() []subUserFixture {
	s.mu.Lock()
	defer s.mu.Unlock()
	users := make([]subUserFixture, 0, len(demoBaseSubUsers)+len(s.created))
	for _, user := range demoBaseSubUsers {
		if s.deleted[user.Name] {
			continue
		}
		users = append(users, user)
	}
	for _, user := range s.created {
		if s.deleted[user.Name] {
			continue
		}
		users = append(users, user)
	}
	return users
}

func (s *iamMutationState) ensureUser(name string) subUserFixture {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.deleted, name)
	if user, ok := s.created[name]; ok {
		return user
	}
	user := subUserFixture{
		Pin:        demoMasterPin + ":" + name,
		Name:       name,
		Account:    name + "@" + demoMasterPin,
		CreateTime: "2026-04-29T12:00:00Z",
	}
	s.created[name] = user
	return user
}

func (s *iamMutationState) deleteUser(name string) bool {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	exists := false
	if _, ok := s.created[name]; ok {
		exists = true
		delete(s.created, name)
	}
	for _, user := range demoBaseSubUsers {
		if user.Name == name {
			exists = true
			break
		}
	}
	if !exists {
		return false
	}
	s.deleted[name] = true
	return true
}

func (s *iamMutationState) attachPolicy(user, policy string) {
	user = strings.TrimSpace(user)
	policy = strings.TrimSpace(policy)
	if user == "" || policy == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.policies[user] == nil {
		s.policies[user] = make(map[string]bool)
	}
	s.policies[user][policy] = true
}

func (s *iamMutationState) detachPolicy(user, policy string) bool {
	user = strings.TrimSpace(user)
	policy = strings.TrimSpace(policy)
	s.mu.Lock()
	defer s.mu.Unlock()
	policies, ok := s.policies[user]
	if !ok || !policies[policy] {
		return false
	}
	delete(policies, policy)
	return true
}

// policiesFor returns the set of policy names currently attached to user.
// Snapshotted under the lock so handlers can iterate without races.
func (s *iamMutationState) policiesFor(user string) []string {
	user = strings.TrimSpace(user)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0)
	for name := range s.policies[user] {
		out = append(out, name)
	}
	return out
}

func (s *iamMutationState) userExists(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deleted[name] {
		return false
	}
	if _, ok := s.created[name]; ok {
		return true
	}
	for _, user := range demoBaseSubUsers {
		if user.Name == name {
			return true
		}
	}
	return false
}

func (s *iamMutationState) snapshotAccessKeys(user string) []jdcloudAccessKeyFixture {
	user = strings.TrimSpace(user)
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]jdcloudAccessKeyFixture(nil), s.accessKeys[user]...)
}

func (s *iamMutationState) mintAccessKey(user string) jdcloudAccessKeyFixture {
	user = strings.TrimSpace(user)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accessKeySeq++
	key := jdcloudAccessKeyFixture{
		AccessKey:  fmt.Sprintf("JDC_AKLTMINT%06d", s.accessKeySeq),
		SecretKey:  fmt.Sprintf("JDCMINTsecret%06d", s.accessKeySeq),
		Status:     "active",
		CreateTime: time.Now().UTC().Format(time.RFC3339),
	}
	s.accessKeys[user] = append(s.accessKeys[user], key)
	return key
}

func (s *iamMutationState) deleteAccessKey(user, accessKey string) bool {
	user = strings.TrimSpace(user)
	accessKey = strings.TrimSpace(accessKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := s.accessKeys[user]
	for i, k := range keys {
		if k.AccessKey == accessKey {
			s.accessKeys[user] = append(keys[:i], keys[i+1:]...)
			return true
		}
	}
	return false
}
