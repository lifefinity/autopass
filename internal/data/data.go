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

// Config holds user-editable settings (~/.autopass/config.json)
type Config struct {
	KeyFile    string `json:"key_file"`
	KeyCommand string `json:"key_command,omitempty"`
}

// Profiles holds encrypted profile data (~/.autopass/data/profiles.json)
type Profiles struct {
	Entries map[string]Profile `json:"profiles"`
}

// Data combines Config + Profiles for backward-compatible API surface
type Data struct {
	Config
	Profiles
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

// Dir returns the autopass root directory (~/.autopass)
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".autopass"), nil
}

// ConfigPath returns ~/.autopass/config.json
func ConfigPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// ProfilesPath returns ~/.autopass/data/profiles.json
func ProfilesPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "data", "profiles.json"), nil
}

// LoadConfig reads config.json
func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(stripComments(raw), &c); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &c, nil
}

// SaveConfig writes config.json
func SaveConfig(path string, c *Config) error {
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	return os.WriteFile(path, raw, 0600)
}

// LoadProfiles reads data/profiles.json
func LoadProfiles(path string) (*Profiles, error) {
	raw, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		if os.IsNotExist(err) {
			return &Profiles{Entries: make(map[string]Profile)}, nil
		}
		return nil, fmt.Errorf("reading profiles: %w", err)
	}
	var p Profiles
	if err := json.Unmarshal(stripComments(raw), &p); err != nil {
		return nil, fmt.Errorf("parsing profiles: %w", err)
	}
	if p.Entries == nil {
		p.Entries = make(map[string]Profile)
	}
	if err := validate(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

// SaveProfiles writes data/profiles.json
func SaveProfiles(path string, p *Profiles) error {
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling profiles: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	return os.WriteFile(path, raw, 0600)
}

// --- Legacy compatibility: Load/Save single file ---

// Load reads the old single-file format (data.json) OR new split format.
// Returns a combined Data struct either way.
func Load(path string) (*Data, error) {
	raw, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		if os.IsNotExist(err) {
			return &Data{Profiles: Profiles{Entries: make(map[string]Profile)}}, nil
		}
		return nil, fmt.Errorf("reading data file: %w", err)
	}

	// Try legacy single-file format
	var legacy struct {
		KeyFile    string             `json:"key_file"`
		KeyCommand string             `json:"key_command,omitempty"`
		Profiles   map[string]Profile `json:"profiles"`
	}
	if err := json.Unmarshal(stripComments(raw), &legacy); err != nil {
		return nil, fmt.Errorf("parsing data file: %w", err)
	}

	d := &Data{
		Config:   Config{KeyFile: legacy.KeyFile, KeyCommand: legacy.KeyCommand},
		Profiles: Profiles{Entries: legacy.Profiles},
	}
	if d.Profiles.Entries == nil {
		d.Profiles.Entries = make(map[string]Profile)
	}
	if err := validate(&d.Profiles); err != nil {
		return nil, err
	}
	return d, nil
}

// Save writes the old single-file format (for backward compat during transition)
func Save(path string, d *Data) error {
	legacy := struct {
		KeyFile    string             `json:"key_file"`
		KeyCommand string             `json:"key_command,omitempty"`
		Profiles   map[string]Profile `json:"profiles"`
	}{
		KeyFile:    d.Config.KeyFile,
		KeyCommand: d.Config.KeyCommand,
		Profiles:   d.Profiles.Entries,
	}
	raw, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling data: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	return os.WriteFile(path, raw, 0600)
}

// --- Profile operations ---

var validProfileName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

func (p *Profiles) AddProfile(name string, profile Profile) error {
	if reservedNames[name] {
		return fmt.Errorf("profile name %q conflicts with a subcommand", name)
	}
	if name == "" {
		return fmt.Errorf("profile name must not be empty")
	}
	if !validProfileName.MatchString(name) {
		return fmt.Errorf("profile name %q is invalid: must start with alphanumeric and contain only alphanumeric, dot, dash, or underscore", name)
	}
	if p.Entries == nil {
		p.Entries = make(map[string]Profile)
	}
	if _, exists := p.Entries[name]; exists {
		return fmt.Errorf("profile %q already exists (use 'autopass update %s' to modify)", name, name)
	}
	p.Entries[name] = profile
	return nil
}

func (p *Profiles) RemoveProfile(name string) error {
	if _, ok := p.Entries[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	delete(p.Entries, name)
	return nil
}

func (p *Profiles) ListProfiles() []string {
	names := make([]string, 0, len(p.Entries))
	for name := range p.Entries {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// --- Helpers ---

func stripComments(data []byte) []byte {
	var out []byte
	inString := false
	for i := 0; i < len(data); i++ {
		if inString {
			out = append(out, data[i])
			if data[i] == '\\' && i+1 < len(data) {
				i++
				out = append(out, data[i])
			} else if data[i] == '"' {
				inString = false
			}
			continue
		}
		if data[i] == '"' {
			inString = true
			out = append(out, data[i])
		} else if data[i] == '/' && i+1 < len(data) && data[i+1] == '/' {
			for i < len(data) && data[i] != '\n' {
				i++
			}
			if i < len(data) {
				out = append(out, '\n')
			}
		} else {
			out = append(out, data[i])
		}
	}
	return out
}

func validate(p *Profiles) error {
	for name, profile := range p.Entries {
		if reservedNames[name] {
			return fmt.Errorf("profile name %q conflicts with a subcommand", name)
		}
		for _, pat := range profile.Patterns {
			if _, err := regexp.Compile(pat.Match); err != nil {
				return fmt.Errorf("invalid regex in profile %q pattern %q: %w", name, pat.Match, err)
			}
		}
	}
	return nil
}
