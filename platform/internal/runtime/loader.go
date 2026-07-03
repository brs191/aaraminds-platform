package aapruntime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

func LoadManifest(path string) (Manifest, error) {
	var manifest Manifest
	if path == "" {
		return manifest, errors.New("manifest path is required")
	}
	if _, err := loadStructuredFile(path, &manifest); err != nil {
		return manifest, fmt.Errorf("load manifest: %w", err)
	}
	return manifest, nil
}

func LoadToolContract(path string) (ToolContract, error) {
	var contract ToolContract
	if _, err := loadStructuredFile(path, &contract); err != nil {
		return contract, fmt.Errorf("load tool contract %s: %w", path, err)
	}
	return contract, nil
}

func LoadBlockedActions(path string) (BlockedActions, error) {
	var blocked BlockedActions
	if _, err := loadStructuredFile(path, &blocked); err != nil {
		return blocked, fmt.Errorf("load blocked actions: %w", err)
	}
	return blocked, nil
}

func LoadContractsWithSchema(dir, schemaPath string) (map[string]ToolContract, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read contracts dir: %w", err)
	}
	contracts := make(map[string]ToolContract)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if err := ValidateStructuredFile(path, schemaPath); err != nil {
			return nil, err
		}
		contract, err := LoadToolContract(path)
		if err != nil {
			return nil, err
		}
		if contract.ToolName == "" {
			return nil, fmt.Errorf("tool contract %q has empty tool_name", path)
		}
		if err := validateValueAgainstSchema(contract.ExampleInvocation, contract.InputSchema, contract.ToolName+" example_invocation"); err != nil {
			return nil, fmt.Errorf("tool contract %q has invalid example_invocation: %w", path, err)
		}
		if _, exists := contracts[contract.ToolName]; exists {
			return nil, fmt.Errorf("duplicate tool contract for %q", contract.ToolName)
		}
		contracts[contract.ToolName] = contract
	}
	return contracts, nil
}

func ValidateStructuredFile(path, schemaPath string) error {
	doc, err := loadStructuredDocument(path)
	if err != nil {
		return fmt.Errorf("load %s for schema validation: %w", path, err)
	}
	schema, err := compileSchema(schemaPath)
	if err != nil {
		return err
	}
	if err := schema.Validate(doc); err != nil {
		return fmt.Errorf("%s does not match schema %s: %w", path, schemaPath, err)
	}
	return nil
}

func validateValueAgainstSchema(value any, schemaDoc map[string]any, name string) error {
	if schemaDoc == nil {
		return errors.New("schema document is required")
	}
	doc, err := schemaValidationDocument(value)
	if err != nil {
		return fmt.Errorf("normalise %s value: %w", name, err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	if err := compiler.AddResource(name, schemaDoc); err != nil {
		return fmt.Errorf("load %s schema: %w", name, err)
	}
	schema, err := compiler.Compile(name)
	if err != nil {
		return fmt.Errorf("compile %s schema: %w", name, err)
	}
	if err := schema.Validate(doc); err != nil {
		return fmt.Errorf("%s schema violation: %w", name, err)
	}
	return nil
}

func schemaValidationDocument(value any) (any, error) {
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

func compileSchema(path string) (*jsonschema.Schema, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open schema %s: %w", path, err)
	}
	defer f.Close()

	doc, err := jsonschema.UnmarshalJSON(f)
	if err != nil {
		return nil, fmt.Errorf("parse schema %s: %w", path, err)
	}
	name := filepath.ToSlash(path)
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	if err := compiler.AddResource(name, doc); err != nil {
		return nil, fmt.Errorf("load schema %s: %w", path, err)
	}
	schema, err := compiler.Compile(name)
	if err != nil {
		return nil, fmt.Errorf("compile schema %s: %w", path, err)
	}
	return schema, nil
}

func loadStructuredFile(path string, out any) (any, error) {
	doc, err := loadStructuredDocument(path)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("normalise %s: %w", path, err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, fmt.Errorf("unexpected trailing data in %s", path)
	}
	return doc, nil
}

func loadStructuredDocument(path string) (any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	switch filepath.Ext(path) {
	case ".json":
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var doc any
		if err := dec.Decode(&doc); err != nil {
			return nil, err
		}
		if err := ensureEOF(dec); err != nil {
			return nil, err
		}
		return doc, nil
	case ".yaml", ".yml":
		var doc any
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			return nil, err
		}
		return normalizeYAML(doc)
	default:
		return nil, fmt.Errorf("unsupported structured file extension %q", filepath.Ext(path))
	}
}

func ensureEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return errors.New("unexpected trailing JSON value")
}

func normalizeYAML(value any) (any, error) {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, child := range v {
			normalized, err := normalizeYAML(child)
			if err != nil {
				return nil, err
			}
			out[key] = normalized
		}
		return out, nil
	case map[any]any:
		out := make(map[string]any, len(v))
		for key, child := range v {
			keyString, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("yaml map key %v is not a string", key)
			}
			normalized, err := normalizeYAML(child)
			if err != nil {
				return nil, err
			}
			out[keyString] = normalized
		}
		return out, nil
	case []any:
		out := make([]any, len(v))
		for i, child := range v {
			normalized, err := normalizeYAML(child)
			if err != nil {
				return nil, err
			}
			out[i] = normalized
		}
		return out, nil
	default:
		return value, nil
	}
}
