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

// Write validates and stores a memory record. It returns the memory_id of a
// superseded record (empty when the write supersedes nothing) so the caller
// can audit the supersession.
//
// Consolidation invariant: at most one active, unexpired record per
// (engagement_id, claim_key). A write whose claim_key collides with an
// existing active record is denied unless it names that record in
// supersedes_memory_id. This keeps contradictory claims from silently
// coexisting — the conflict either resolves (supersede) or fails closed.
func (s *MemoryStore) Write(record MemoryRecord, allowed []string, piiAllowed bool) (supersededID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record.MemoryID == "" || record.AgentID == "" || record.EngagementID == "" {
		return "", errors.New("memory record requires memory_id, agent_id, and engagement_id")
	}
	if record.Classification == "" || record.ContentRef == "" || record.ContentHash == "" {
		return "", errors.New("memory record requires classification, content_ref, and content_hash")
	}
	if record.SourceCitation.RunID == "" || record.SourceCitation.TraceID == "" || record.SourceCitation.SpanID == "" || record.SourceCitation.SourceRef == "" {
		return "", errors.New("memory write requires complete source citation")
	}
	now := time.Now().UTC()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	if record.ExpiresAt.IsZero() {
		return "", errors.New("memory record requires expires_at")
	}
	if !record.ExpiresAt.After(now) {
		return "", errors.New("memory record expires_at must be in the future")
	}
	if record.Classification == "pii" && !piiAllowed {
		return "", errors.New("manifest does not allow pii memory")
	}
	if !contains(allowed, record.Classification) {
		return "", fmt.Errorf("classification %q is not allowed by manifest", record.Classification)
	}
	if record.Status == "" {
		record.Status = "active"
	}

	// Resolve the supersede target before checking claim conflicts so a
	// valid supersede of the conflicting record is allowed.
	target := -1
	if record.SupersedesMemoryID != "" {
		if record.SupersedesMemoryID == record.MemoryID {
			return "", errors.New("memory record cannot supersede itself")
		}
		for i, existing := range s.records {
			if existing.MemoryID != record.SupersedesMemoryID {
				continue
			}
			if existing.EngagementID != record.EngagementID || existing.AgentID != record.AgentID {
				return "", fmt.Errorf("supersede target %q is outside the record's engagement/agent scope", record.SupersedesMemoryID)
			}
			if existing.Status != "active" {
				return "", fmt.Errorf("supersede target %q is not active", record.SupersedesMemoryID)
			}
			target = i
			break
		}
		if target == -1 {
			return "", fmt.Errorf("supersede target %q does not exist", record.SupersedesMemoryID)
		}
	}

	// Claim-key conflict gate: deny a second active claim unless this write
	// supersedes the record holding it.
	if record.ClaimKey != "" {
		for i, existing := range s.records {
			if existing.EngagementID != record.EngagementID || existing.Status != "active" {
				continue
			}
			if existing.ClaimKey != record.ClaimKey {
				continue
			}
			if !existing.ExpiresAt.IsZero() && !existing.ExpiresAt.After(now) {
				continue
			}
			if i != target {
				return "", fmt.Errorf("active record %q already holds claim_key %q; write must supersede it", existing.MemoryID, record.ClaimKey)
			}
		}
	}

	if target >= 0 {
		s.records[target].Status = "superseded"
		supersededID = s.records[target].MemoryID
	}
	s.records = append(s.records, record)
	return supersededID, nil
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
