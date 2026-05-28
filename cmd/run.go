package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/lifefinity/autopass/internal/crypto"
	"github.com/lifefinity/autopass/internal/engine"
)

func runProfileWithSteps(profileName string, runOpts profileRunOpts) error {
	d, err := loadData()
	if err != nil {
		return err
	}

	profile, ok := d.Profiles[profileName]
	if !ok {
		return fmt.Errorf("profile %q not found", profileName)
	}

	command := splitCommand(profile.Command)
	timeout := profile.Timeout.Duration
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Decrypt the profile's secret
	var secret string
	if profile.Secret != "" {
		key, err := deriveKey()
		if err != nil {
			return err
		}

		ciphertext, err := base64.StdEncoding.DecodeString(profile.Secret)
		if err != nil {
			return fmt.Errorf("decoding secret: %w", err)
		}

		plaintext, err := crypto.Decrypt(key, ciphertext)
		if err != nil {
			return fmt.Errorf("decrypting secret: %w", err)
		}
		secret = string(plaintext)
	}

	// Build engine patterns: each pattern's response is the decrypted secret
	enginePatterns := make([]engine.Pattern, len(profile.Patterns))
	for i, p := range profile.Patterns {
		enginePatterns[i] = engine.Pattern{
			Match:   p.Match,
			Respond: secret,
			Hidden:  p.Hidden,
		}
	}

	// Load post-login steps
	steps, err := loadSteps(runOpts)
	if err != nil {
		return err
	}

	// Determine prompt pattern (CLI override > profile > empty)
	prompt := runOpts.prompt
	if prompt == "" {
		prompt = profile.Prompt
	}

	exitCode, err := engine.Run(engine.Options{
		Command:  command,
		Patterns: enginePatterns,
		Timeout:  timeout,
		Steps:    steps,
		Prompt:   prompt,
	})

	if err != nil {
		return err
	}

	os.Exit(exitCode)
	return nil
}

func splitCommand(cmd string) []string {
	fields := []string{}
	current := ""
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		if inQuote {
			if c == quoteChar {
				inQuote = false
			} else {
				current += string(c)
			}
		} else if c == '"' || c == '\'' {
			inQuote = true
			quoteChar = c
		} else if c == ' ' {
			if current != "" {
				fields = append(fields, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		fields = append(fields, current)
	}
	return fields
}
