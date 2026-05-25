package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lifefinity/autopass/internal/data"
)

// setupTestData creates a temporary data.json for testing.
func setupTestData(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".autopass", "data.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("creating test dir: %v", err)
	}

	d := &data.Data{
		SSHKey: filepath.Join(dir, "fake_key"),
		Profiles: map[string]data.Profile{
			"myserver": {
				Command:  "ssh user@host",
				Patterns: []data.Pattern{{Match: "(?i)password:", Hidden: true}},
				Secret:   "ZW5jcnlwdGVk", // base64("encrypted")
				Timeout:  data.Duration{Duration: 30 * time.Second},
			},
			"mydb": {
				Command:  "psql -h localhost -U admin",
				Patterns: []data.Pattern{{Match: "(?i)password", Hidden: true}},
				Secret:   "ZGJzZWNyZXQ=", // base64("dbsecret")
				Prompt:   "=>\\s*$",
				Timeout:  data.Duration{Duration: 60 * time.Second},
			},
		},
	}

	raw, _ := json.MarshalIndent(d, "", "  ")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatalf("writing test data: %v", err)
	}

	return path, func() { os.RemoveAll(dir) }
}

func TestDataLoad_Integration(t *testing.T) {
	path, cleanup := setupTestData(t)
	defer cleanup()

	d, err := data.Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(d.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(d.Profiles))
	}

	p := d.Profiles["myserver"]
	if p.Command != "ssh user@host" {
		t.Errorf("unexpected command: %s", p.Command)
	}
	if p.Patterns[0].Match != "(?i)password:" {
		t.Errorf("unexpected pattern: %s", p.Patterns[0].Match)
	}
}

func TestDataAddRemove_Integration(t *testing.T) {
	path, cleanup := setupTestData(t)
	defer cleanup()

	d, err := data.Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Add a new profile
	err = d.AddProfile("redis", data.Profile{
		Command:  "redis-cli -h cache.example.com",
		Patterns: []data.Pattern{{Match: "(?i)password:", Hidden: true}},
		Secret:   "cmVkaXM=",
		Timeout:  data.Duration{Duration: 10 * time.Second},
	})
	if err != nil {
		t.Fatalf("AddProfile failed: %v", err)
	}

	if err := data.Save(path, d); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Reload and verify
	d2, err := data.Load(path)
	if err != nil {
		t.Fatalf("Load after add failed: %v", err)
	}
	if len(d2.Profiles) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(d2.Profiles))
	}
	if d2.Profiles["redis"].Command != "redis-cli -h cache.example.com" {
		t.Fatalf("unexpected redis command: %s", d2.Profiles["redis"].Command)
	}

	// Remove
	if err := d2.RemoveProfile("redis"); err != nil {
		t.Fatalf("RemoveProfile failed: %v", err)
	}
	if err := data.Save(path, d2); err != nil {
		t.Fatalf("Save after remove failed: %v", err)
	}

	d3, _ := data.Load(path)
	if len(d3.Profiles) != 2 {
		t.Fatalf("expected 2 profiles after remove, got %d", len(d3.Profiles))
	}
}

func TestDataUpdate_Integration(t *testing.T) {
	path, cleanup := setupTestData(t)
	defer cleanup()

	d, err := data.Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Update command field
	p := d.Profiles["myserver"]
	p.Command = "ssh newuser@newhost"
	d.Profiles["myserver"] = p

	if err := data.Save(path, d); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Reload and verify
	d2, _ := data.Load(path)
	if d2.Profiles["myserver"].Command != "ssh newuser@newhost" {
		t.Fatalf("update not persisted: %s", d2.Profiles["myserver"].Command)
	}
	// Other fields unchanged
	if d2.Profiles["myserver"].Patterns[0].Match != "(?i)password:" {
		t.Fatalf("pattern should be unchanged: %s", d2.Profiles["myserver"].Patterns[0].Match)
	}
}

func TestDataReservedNames_Integration(t *testing.T) {
	reserved := []string{"init", "add", "update", "list", "remove", "version", "help"}

	d := &data.Data{Profiles: make(map[string]data.Profile)}

	for _, name := range reserved {
		err := d.AddProfile(name, data.Profile{Command: "test"})
		if err == nil {
			t.Errorf("expected error for reserved name %q", name)
		}
	}
}

func TestDataListProfiles_Integration(t *testing.T) {
	path, cleanup := setupTestData(t)
	defer cleanup()

	d, _ := data.Load(path)
	names := d.ListProfiles()

	if len(names) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(names))
	}
	// Should be sorted
	if names[0] != "mydb" || names[1] != "myserver" {
		t.Fatalf("expected sorted [mydb, myserver], got %v", names)
	}
}
