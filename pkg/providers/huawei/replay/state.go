package replay

import (
	"fmt"
	"strings"
	"sync"
)

type iamMutationState struct {
	mu       sync.Mutex
	created  map[string]iamUserFixture
	deleted  map[string]bool
	memberOf map[string]map[string]bool
	sequence int
}

func newIAMMutationState() *iamMutationState {
	return &iamMutationState{
		created:  make(map[string]iamUserFixture),
		deleted:  make(map[string]bool),
		memberOf: make(map[string]map[string]bool),
	}
}

func (s *iamMutationState) snapshotUsers() []iamUserFixture {
	s.mu.Lock()
	defer s.mu.Unlock()
	users := make([]iamUserFixture, 0, len(demoBaseIAMUsers)+len(s.created))
	for _, user := range demoBaseIAMUsers {
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

func (s *iamMutationState) findByName(name string) (iamUserFixture, bool) {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deleted[name] {
		return iamUserFixture{}, false
	}
	if user, ok := s.created[name]; ok {
		return user, true
	}
	for _, user := range demoBaseIAMUsers {
		if user.Name == name {
			return user, true
		}
	}
	return iamUserFixture{}, false
}

func (s *iamMutationState) findByID(id string) (iamUserFixture, bool) {
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, user := range s.created {
		if user.ID == id && !s.deleted[user.Name] {
			return user, true
		}
	}
	for _, user := range demoBaseIAMUsers {
		if user.ID == id && !s.deleted[user.Name] {
			return user, true
		}
	}
	return iamUserFixture{}, false
}

func (s *iamMutationState) ensureUser(name string) iamUserFixture {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.deleted, name)
	if user, ok := s.created[name]; ok {
		return user
	}
	for _, user := range demoBaseIAMUsers {
		if user.Name == name {
			return user
		}
	}
	s.sequence++
	user := iamUserFixture{
		ID:       newSyntheticUserID(s.sequence),
		Name:     name,
		Enabled:  true,
		DomainID: demoDomainID,
	}
	s.created[name] = user
	return user
}

func (s *iamMutationState) deleteByID(id string) {
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, user := range s.created {
		if user.ID == id {
			s.deleted[name] = true
			delete(s.created, name)
			return
		}
	}
	for _, user := range demoBaseIAMUsers {
		if user.ID == id {
			s.deleted[user.Name] = true
			return
		}
	}
}

func (s *iamMutationState) recordGroupMembership(groupID, userID string) {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	if groupID == "" || userID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.memberOf[groupID] == nil {
		s.memberOf[groupID] = make(map[string]bool)
	}
	s.memberOf[groupID][userID] = true
}

func (s *iamMutationState) removeGroupMembership(groupID, userID string) bool {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	if groupID == "" || userID == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	members, ok := s.memberOf[groupID]
	if !ok {
		return false
	}
	if !members[userID] {
		return false
	}
	delete(members, userID)
	return true
}

// groupsForUser returns the group IDs that userID currently belongs to.
// Membership is mutation-only here; demo-baseline users start with no groups
// so list/add/list cycles surface deterministic state in replay.
func (s *iamMutationState) groupsForUser(userID string) []string {
	userID = strings.TrimSpace(userID)
	s.mu.Lock()
	defer s.mu.Unlock()
	groups := make([]string, 0)
	for groupID, members := range s.memberOf {
		if members[userID] {
			groups = append(groups, groupID)
		}
	}
	return groups
}

func newSyntheticUserID(sequence int) string {
	return fmt.Sprintf("06f1d2dca680f0a02fa4c01acc0e9%03d", sequence%1000)
}
