package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

var reservedNames = map[string]bool{
	"init":    true,
	"add":     true,
	"update":  true,
	"list":    true,
	"remove":  true,
	"export":  true,
	"import":  true,
	"backup":  true,
	"restore": true,
	"version": true,
	"help":    true,
}

type Data struct {
	SSHKey   string             `json:"ssh_key"`
	Profiles map[string]Profile `json:"profiles"`
}

type Profile struct {
	Command     string    `json:"command"`
	Description string    `json:"description,omitempty"`
	Patterns    []Pattern `json:"patterns"`
	Secret      string    `json:"secret,omitempty"`
	Prompt      string    `json:"prompt,omitempty"`
	Timeout     Duration  `json:"timeout"`
	Steps       []string  `json:"steps,omitempty"`
	After       []string  `json:"after,omitempty"`
}

type Pattern struct {
	Match         string `json:"match"`
	Hidden        bool   `json:"hidden"`
	CaseSensitive bool   `json:"case_sensitive,omitempty"`
}

// Duration is a wrapper around time.Duration that supports JSON marshaling
// as a human-readable string (e.g. "30s").
type Duration struct {
	time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = dur
	return nil
}

func Load(path string) (*Data, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- path is from user config
	if err != nil {
		if os.IsNotExist(err) {
			return &Data{Profiles: make(map[string]Profile)}, nil
		}
		return nil, fmt.Errorf("reading data file: %w", err)
	}

	var d Data
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, fmt.Errorf("parsing data file: %w", err)
	}

	if d.Profiles == nil {
		d.Profiles = make(map[string]Profile)
	}

	if err := validate(&d); err != nil {
		return nil, err
	}

	return &d, nil
}

func validate(d *Data) error {
	for name, profile := range d.Profiles {
		if reservedNames[name] {
			return fmt.Errorf("profile name %q conflicts with a subcommand", name)
		}
		for _, p := range profile.Patterns {
			if _, err := regexp.Compile(p.Match); err != nil {
				return fmt.Errorf("invalid regex in profile %q pattern %q: %w", name, p.Match, err)
			}
		}
	}
	return nil
}

func Save(path string, d *Data) error {
	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling data: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	if err := os.WriteFile(path, raw, 0600); err != nil {
		return fmt.Errorf("writing data file: %w", err)
	}

	return nil
}

var validProfileName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

func (d *Data) AddProfile(name string, profile Profile) error {
	if reservedNames[name] {
		return fmt.Errorf("profile name %q conflicts with a subcommand", name)
	}
	if name == "" {
		return fmt.Errorf("profile name must not be empty")
	}
	if !validProfileName.MatchString(name) {
		return fmt.Errorf("profile name %q is invalid: must start with alphanumeric and contain only alphanumeric, dot, dash, or underscore", name)
	}
	if d.Profiles == nil {
		d.Profiles = make(map[string]Profile)
	}
	d.Profiles[name] = profile
	return nil
}

func (d *Data) RemoveProfile(name string) error {
	if _, ok := d.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	delete(d.Profiles, name)
	return nil
}

func (d *Data) ListProfiles() []string {
	names := make([]string, 0, len(d.Profiles))
	for name := range d.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
