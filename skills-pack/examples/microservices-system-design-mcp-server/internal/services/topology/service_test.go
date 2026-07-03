package topology

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
		{name: "ok minimum", in: Input{SystemName: "sys", Services: []Service{{Name: "api"}}}},
		{name: "missing system", in: Input{Services: []Service{{Name: "api"}}}, wantErr: true},
		{name: "no services", in: Input{SystemName: "sys"}, wantErr: true},
		{name: "service missing name", in: Input{SystemName: "sys", Services: []Service{{}}}, wantErr: true},
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

func TestGenerate_DefaultPlatform(t *testing.T) {
	t.Parallel()
	out, err := NewService().Generate(Input{
		SystemName: "sys",
		Services:   []Service{{Name: "api", Type: "api", Criticality: "high"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Platform != "container_apps" {
		t.Errorf("Platform = %q, want container_apps", out.Platform)
	}
}

func TestGenerate_HighCriticalityHasReplicaFloor(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName: "sys",
		Services:   []Service{{Name: "api", Type: "api", Criticality: "high"}},
	})
	if got := out.ServicePlacements[0].Replicas; !strings.HasPrefix(got, "2-") {
		t.Errorf("high-criticality replicas = %q, want a range starting with 2-", got)
	}
}

func TestGenerate_GatewayIsExternal(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName: "sys",
		Services:   []Service{{Name: "gw", Type: "gateway", Criticality: "high"}},
	})
	if got := out.ServicePlacements[0].Ingress; got != "external" {
		t.Errorf("gateway ingress = %q, want external", got)
	}
}

func TestGenerate_WorkerHasQueueScale(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName: "sys",
		Services:   []Service{{Name: "w", Type: "worker", Criticality: "low"}},
	})
	if got := out.ServicePlacements[0].ScaleRule; got != "queue_depth" {
		t.Errorf("worker scale rule = %q, want queue_depth", got)
	}
	if got := out.ServicePlacements[0].Ingress; got != "none" {
		t.Errorf("worker ingress = %q, want none", got)
	}
	if !strings.HasPrefix(out.ServicePlacements[0].Replicas, "0-") {
		t.Errorf("low-crit worker should scale to zero, replicas = %q", out.ServicePlacements[0].Replicas)
	}
}

func TestGenerate_SensitiveDataGetsIsolationBoundary(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName: "sys",
		Services:   []Service{{Name: "api"}},
		DataStores: []DataStore{{Name: "patients-db", Kind: "postgres", Classification: "phi"}},
	})
	found := false
	for _, b := range out.NetworkBoundaries {
		if strings.HasPrefix(b.Name, "isolation:phi") {
			found = true
		}
	}
	if !found {
		t.Error("expected an isolation boundary for PHI data store")
	}
	if got := out.DataPlacements[0].Subnet; got != "dedicated" {
		t.Errorf("PHI subnet = %q, want dedicated", got)
	}
	if got := out.DataPlacements[0].Encryption; got != "at_rest_cmk" {
		t.Errorf("PHI encryption = %q, want at_rest_cmk", got)
	}
}

func TestGenerate_MultiRegionFlaggedOnHighCrit(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName: "sys",
		Services:   []Service{{Name: "api", Criticality: "high"}},
		NFR:        NFR{MultiRegion: true},
	})
	if !strings.Contains(out.ServicePlacements[0].Notes, "two regions") {
		t.Errorf("expected multi-region note, got %q", out.ServicePlacements[0].Notes)
	}
}

func TestGenerate_GapsIdentified(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName: "sys",
		Services:   []Service{{Name: "api"}},
	})
	if len(out.Gaps) == 0 {
		t.Error("expected gaps to be identified for thin input")
	}
	if out.Score >= 95 {
		t.Errorf("Score = %d, want <95 for input with gaps", out.Score)
	}
}

func TestGenerate_StableOrdering(t *testing.T) {
	t.Parallel()
	in := Input{
		SystemName: "sys",
		Services:   []Service{{Name: "z", Type: "api"}, {Name: "a", Type: "api"}},
		DataStores: []DataStore{{Name: "z-db", Kind: "postgres"}, {Name: "a-db", Kind: "postgres"}},
	}
	a, _ := NewService().Generate(in)
	b, _ := NewService().Generate(in)
	if a.ServicePlacements[0].Service != "a" || a.DataPlacements[0].Name != "a-db" {
		t.Error("expected alphabetical ordering of services and data placements")
	}
	if a.Summary != b.Summary {
		t.Error("summary not byte-stable across calls")
	}
}
