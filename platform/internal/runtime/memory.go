package aapruntime

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type MemoryStore struct {
	mu      sync.Mutex
	records []MemoryRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) Write(record MemoryRecord, allowed []string, piiAllowed bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record.MemoryID == "" || record.AgentID == "" || record.EngagementID == "" {
		return errors.New("memory record requires memory_id, agent_id, and engagement_id")
	}
	if record.Classification == "" || record.ContentRef == "" || record.ContentHash == "" {
		return errors.New("memory record requires classification, content_ref, and content_hash")
	}
	if record.SourceCitation.RunID == "" || record.SourceCitation.TraceID == "" || record.SourceCitation.SpanID == "" || record.SourceCitation.SourceRef == "" {
		return errors.New("memory write requires complete source citation")
	}
	now := time.Now().UTC()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	if record.ExpiresAt.IsZero() {
		return errors.New("memory record requires expires_at")
	}
	if !record.ExpiresAt.After(now) {
		return errors.New("memory record expires_at must be in the future")
	}
	if record.Classification == "pii" && !piiAllowed {
		return errors.New("manifest does not allow pii memory")
	}
	if !contains(allowed, record.Classification) {
		return fmt.Errorf("classification %q is not allowed by manifest", record.Classification)
	}
	if record.Status == "" {
		record.Status = "active"
	}
	s.records = append(s.records, record)
	return nil
}

func (s *MemoryStore) Query(engagementID, scope, agentID string) []MemoryRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	if scope == "none" {
		return nil
	}
	now := time.Now().UTC()
	out := make([]MemoryRecord, 0)
	for _, record := range s.records {
		if record.EngagementID != engagementID || record.Status != "active" {
			continue
		}
		if !record.ExpiresAt.IsZero() && !record.ExpiresAt.After(now) {
			continue
		}
		if scope == "agent" && record.AgentID != agentID {
			continue
		}
		if scope == "engagement" || scope == "agent" {
			out = append(out, record)
		}
	}
	return out
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
