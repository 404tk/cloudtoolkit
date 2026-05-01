package replay

import (
	"strings"
	"sync"
)

type iamMutationState struct {
	mu      sync.Mutex
	created map[string]subUserFixture
	deleted map[string]bool
	policies map[string]map[string]bool
}

func newIAMState() *iamMutationState {
	return &iamMutationState{
		created:  make(map[string]subUserFixture),
		deleted:  make(map[string]bool),
		policies: make(map[string]map[string]bool),
	}
}

func (s *iamMutationState) snapshotUsers() []subUserFixture {
	s.mu.Lock()
	defer s.mu.Unlock()
	users := make([]subUserFixture, 0, len(demoBaseSubUsers)+len(s.created))
	for _, user := range demoBaseSubUsers {
		if s.deleted[user.UserName] {
			continue
		}
		users = append(users, user)
	}
	for _, user := range s.created {
		if s.deleted[user.UserName] {
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
		UserName:    name,
		DisplayName: name,
		Email:       name + "@ctk.demo",
		Status:      "Active",
		CreatedAt:   1714694400,
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
		if user.UserName == name {
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
