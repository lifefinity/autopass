package data

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestLoad_NonExistentFile(t *testing.T) {
	d, err := Load("/nonexistent/path/data.json")
	if err != nil {
		t.Fatalf("Load should not error on nonexistent file: %v", err)
	}
	if d.Profiles == nil {
		t.Fatal("Profiles map should be initialized")
	}
	if len(d.Profiles) != 0 {
		t.Fatalf("expected empty profiles, got %d", len(d.Profiles))
	}
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	content := `{
  "ssh_key": "/home/user/.ssh/id_ed25519",
  "profiles": {
    "mwinit": {
      "command": "mwinit -s -o",
      "patterns": [
        {"match": "(?i)midway PIN:", "hidden": true}
      ],
      "secret": "c29tZWVuY3J5cHRlZGRhdGE=",
      "timeout": "30s"
    }
  }
}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	d, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if d.SSHKey != "/home/user/.ssh/id_ed25519" {
		t.Fatalf("unexpected ssh_key: %s", d.SSHKey)
	}
	if len(d.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(d.Profiles))
	}
	p := d.Profiles["mwinit"]
	if p.Command != "mwinit -s -o" {
		t.Fatalf("unexpected command: %s", p.Command)
	}
	if len(p.Patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(p.Patterns))
	}
	if p.Patterns[0].Match != "(?i)midway PIN:" {
		t.Fatalf("unexpected match: %s", p.Patterns[0].Match)
	}
	if !p.Patterns[0].Hidden {
		t.Fatal("expected hidden=true")
	}
	if p.Secret != "c29tZWVuY3J5cHRlZGRhdGE=" {
		t.Fatalf("unexpected secret: %s", p.Secret)
	}
	if p.Timeout.Duration != 30*time.Second {
		t.Fatalf("unexpected timeout: %v", p.Timeout.Duration)
	}
}

func TestLoad_InvalidRegex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	content := `{
  "ssh_key": "/home/user/.ssh/id_ed25519",
  "profiles": {
    "bad": {
      "command": "echo",
      "patterns": [{"match": "[invalid", "hidden": false}],
      "timeout": "10s"
    }
  }
}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestLoad_ReservedName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	content := `{
  "ssh_key": "/home/user/.ssh/id_ed25519",
  "profiles": {
    "init": {
      "command": "something",
      "patterns": [{"match": "x", "hidden": false}],
      "timeout": "10s"
    }
  }
}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for reserved profile name")
	}
}

func TestSave_And_Load(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")

	d := &Data{
		SSHKey: "/path/to/key",
		Profiles: map[string]Profile{
			"test": {
				Command:  "echo hello",
				Patterns: []Pattern{{Match: "(?i)prompt:", Hidden: true}},
				Secret:   "encrypted-base64",
				Timeout:  Duration{30 * time.Second},
			},
		},
	}

	if err := Save(path, d); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load after Save failed: %v", err)
	}

	if loaded.SSHKey != d.SSHKey {
		t.Fatalf("SSHKey mismatch: %s vs %s", loaded.SSHKey, d.SSHKey)
	}
	if len(loaded.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(loaded.Profiles))
	}
	p := loaded.Profiles["test"]
	if p.Command != "echo hello" {
		t.Fatalf("Command mismatch: %s", p.Command)
	}
	if p.Secret != "encrypted-base64" {
		t.Fatalf("Secret mismatch: %s", p.Secret)
	}
}

func TestSave_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permissions not applicable on Windows")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")

	d := &Data{SSHKey: "/key", Profiles: map[string]Profile{}}
	if err := Save(path, d); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Fatalf("expected 0600 permissions, got %o", perm)
	}
}

func TestAddProfile(t *testing.T) {
	d := &Data{Profiles: make(map[string]Profile)}

	err := d.AddProfile("myprofile", Profile{
		Command:  "ssh host",
		Patterns: []Pattern{{Match: "password:", Hidden: true}},
		Secret:   "enc",
		Timeout:  Duration{10 * time.Second},
	})
	if err != nil {
		t.Fatalf("AddProfile failed: %v", err)
	}

	if _, ok := d.Profiles["myprofile"]; !ok {
		t.Fatal("profile not added")
	}
}

func TestAddProfile_ReservedName(t *testing.T) {
	d := &Data{Profiles: make(map[string]Profile)}

	err := d.AddProfile("init", Profile{Command: "x"})
	if err == nil {
		t.Fatal("expected error for reserved name")
	}
}

func TestRemoveProfile(t *testing.T) {
	d := &Data{Profiles: map[string]Profile{
		"one": {Command: "echo one"},
		"two": {Command: "echo two"},
	}}

	err := d.RemoveProfile("one")
	if err != nil {
		t.Fatalf("RemoveProfile failed: %v", err)
	}
	if _, ok := d.Profiles["one"]; ok {
		t.Fatal("profile should be removed")
	}
	if len(d.Profiles) != 1 {
		t.Fatalf("expected 1 profile remaining, got %d", len(d.Profiles))
	}
}

func TestRemoveProfile_NotFound(t *testing.T) {
	d := &Data{Profiles: make(map[string]Profile)}

	err := d.RemoveProfile("ghost")
	if err == nil {
		t.Fatal("expected error for nonexistent profile")
	}
}

func TestListProfiles(t *testing.T) {
	d := &Data{Profiles: map[string]Profile{
		"beta":  {Command: "b"},
		"alpha": {Command: "a"},
		"gamma": {Command: "g"},
	}}

	names := d.ListProfiles()
	if len(names) != 3 {
		t.Fatalf("expected 3, got %d", len(names))
	}
	if names[0] != "alpha" || names[1] != "beta" || names[2] != "gamma" {
		t.Fatalf("expected sorted order, got %v", names)
	}
}
