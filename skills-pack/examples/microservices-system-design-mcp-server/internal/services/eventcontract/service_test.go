package eventcontract

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      Input
		wantErr bool
	}{
		{name: "ok", in: Input{SystemName: "sys", EventName: "OrderCreated", Producer: "order-service", Fields: []Field{{Name: "order_id", Type: "uuid"}}}},
		{name: "missing system", in: Input{EventName: "OrderCreated", Producer: "p", Fields: []Field{{Name: "x", Type: "uuid"}}}, wantErr: true},
		{name: "missing event_name", in: Input{SystemName: "s", Producer: "p", Fields: []Field{{Name: "x", Type: "uuid"}}}, wantErr: true},
		{name: "missing producer", in: Input{SystemName: "s", EventName: "X", Fields: []Field{{Name: "x", Type: "uuid"}}}, wantErr: true},
		{name: "no fields", in: Input{SystemName: "s", EventName: "X", Producer: "p"}, wantErr: true},
		{name: "field missing name", in: Input{SystemName: "s", EventName: "X", Producer: "p", Fields: []Field{{Type: "uuid"}}}, wantErr: true},
		{name: "field missing type", in: Input{SystemName: "s", EventName: "X", Producer: "p", Fields: []Field{{Name: "x"}}}, wantErr: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewService().Generate(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerate_WarnsOnCommandShape(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName: "sys",
		EventName:  "CreateOrder",
		Producer:   "order-service",
		Fields:     []Field{{Name: "order_id", Type: "uuid", Required: true}},
		Consumers:  []string{"notification-service"},
	})
	commandWarn := false
	for _, w := range out.Warnings {
		if strings.Contains(w, "command") || strings.Contains(w, "past-tense") {
			commandWarn = true
		}
	}
	if !commandWarn {
		t.Errorf("expected a warning about command-shaped event name, got %v", out.Warnings)
	}
}

func TestGenerate_ServiceBusTransportDefault(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName: "sys",
		EventName:  "OrderConfirmed",
		Producer:   "order-service",
		Fields:     []Field{{Name: "order_id", Type: "uuid", Required: true}},
		Consumers:  []string{"notification-service", "billing-service"},
	})
	if out.Transport.AzureService != "Azure Service Bus" {
		t.Errorf("transport = %q, want Azure Service Bus", out.Transport.AzureService)
	}
	if len(out.Transport.Subscriptions) != 2 {
		t.Errorf("expected 2 subscriptions, got %d", len(out.Transport.Subscriptions))
	}
}

func TestGenerate_EventHubsHasNoDLQWarning(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName: "sys",
		EventName:  "MetricRecorded",
		Producer:   "telemetry-service",
		Transport:  "event_hubs",
		Fields:     []Field{{Name: "metric_id", Type: "string", Required: true}},
		Consumers:  []string{"analytics-service"},
	})
	noDlq := false
	for _, w := range out.Warnings {
		if strings.Contains(w, "Event Hubs") && strings.Contains(w, "DLQ") {
			noDlq = true
		}
	}
	if !noDlq {
		t.Error("expected warning about Event Hubs lacking native DLQ")
	}
}

func TestGenerate_SensitiveFieldHandlingNote(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName: "sys",
		EventName:  "PatientRecordViewed",
		Producer:   "records-service",
		Fields: []Field{
			{Name: "patient_id", Type: "uuid", Required: true},
			{Name: "ssn", Type: "string", Required: true, Sensitive: true},
		},
		Consumers: []string{"audit-service"},
	})
	hasNote := false
	for _, f := range out.Schema.DataFields {
		if f.Name == "ssn" && strings.Contains(f.HandlingNote, "redact") {
			hasNote = true
		}
	}
	if !hasNote {
		t.Error("expected handling note on sensitive field")
	}
	sensitiveWarn := false
	for _, w := range out.Warnings {
		if strings.Contains(w, "sensitive") {
			sensitiveWarn = true
		}
	}
	if !sensitiveWarn {
		t.Error("expected a sensitive-fields warning")
	}
}

func TestGenerate_AlwaysAddsEnvelopeFields(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName: "sys",
		EventName:  "OrderConfirmed",
		Producer:   "order-service",
		Fields:     []Field{{Name: "order_id", Type: "uuid", Required: true}},
		Consumers:  []string{"notification-service"},
	})
	want := map[string]bool{"event_id": false, "occurred_at": false, "correlation_id": false}
	for _, f := range out.Schema.DataFields {
		if _, ok := want[f.Name]; ok {
			want[f.Name] = true
		}
	}
	for k, v := range want {
		if !v {
			t.Errorf("expected envelope field %q in schema", k)
		}
	}
}

func TestGenerate_MarkdownStable(t *testing.T) {
	t.Parallel()
	in := Input{
		SystemName: "sys",
		EventName:  "OrderConfirmed",
		Producer:   "order-service",
		Fields:     []Field{{Name: "order_id", Type: "uuid", Required: true}},
		Consumers:  []string{"notification-service"},
	}
	a, _ := NewService().Generate(in)
	b, _ := NewService().Generate(in)
	if a.Markdown != b.Markdown {
		t.Error("Markdown not byte-stable across calls")
	}
}
