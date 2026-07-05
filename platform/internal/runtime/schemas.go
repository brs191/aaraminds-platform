package aapruntime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type RuntimeSchemas struct {
	MemoryRecord    *jsonschema.Schema
	AuditEvent      *jsonschema.Schema
	ApprovalRequest *jsonschema.Schema
}

func LoadRuntimeSchemas(root string) (RuntimeSchemas, error) {
	memorySchema, err := compileSchema(filepath.Join(root, "schemas", "memory-record.schema.json"))
	if err != nil {
		return RuntimeSchemas{}, err
	}
	auditSchema, err := compileSchema(filepath.Join(root, "schemas", "audit-event.schema.json"))
	if err != nil {
		return RuntimeSchemas{}, err
	}
	approvalSchema, err := compileSchema(filepath.Join(root, "schemas", "approval-request.schema.json"))
	if err != nil {
		return RuntimeSchemas{}, err
	}
	return RuntimeSchemas{
		MemoryRecord:    memorySchema,
		AuditEvent:      auditSchema,
		ApprovalRequest: approvalSchema,
	}, nil
}

func (s RuntimeSchemas) ValidateMemoryRecord(record MemoryRecord) error {
	if s.MemoryRecord == nil {
		return nil
	}
	doc, err := structToSchemaDocument(record)
	if err != nil {
		return fmt.Errorf("prepare memory record for schema validation: %w", err)
	}
	if err := s.MemoryRecord.Validate(doc); err != nil {
		return fmt.Errorf("memory record schema violation: %w", err)
	}
	return nil
}

func (s RuntimeSchemas) ValidateAuditEvent(event AuditEvent) error {
	if s.AuditEvent == nil {
		return nil
	}
	doc, err := structToSchemaDocument(event)
	if err != nil {
		return fmt.Errorf("prepare audit event for schema validation: %w", err)
	}
	if err := s.AuditEvent.Validate(doc); err != nil {
		return fmt.Errorf("audit event schema violation: %w", err)
	}
	return nil
}

func (s RuntimeSchemas) ValidateApprovalRequest(request ApprovalRequest) error {
	if s.ApprovalRequest == nil {
		return nil
	}
	doc, err := structToSchemaDocument(request)
	if err != nil {
		return fmt.Errorf("prepare approval request for schema validation: %w", err)
	}
	if err := s.ApprovalRequest.Validate(doc); err != nil {
		return fmt.Errorf("approval request schema violation: %w", err)
	}
	return nil
}

func structToSchemaDocument(value any) (any, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var doc any
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}
	return doc, ensureEOF(dec)
}
