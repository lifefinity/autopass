package engine

import (
	"testing"
)

func TestMatcher_SimpleMatch(t *testing.T) {
	patterns := []Pattern{
		{Match: "(?i)password:", Respond: "secret123", Hidden: true},
	}

	m, err := NewMatcher(patterns)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	resp := m.Check("Enter password:")
	if resp == nil {
		t.Fatal("expected a match")
	}
	if resp.Response != "secret123" {
		t.Fatalf("expected 'secret123', got %q", resp.Response)
	}
	if !resp.Hidden {
		t.Fatal("expected hidden=true")
	}
}

func TestMatcher_NoMatch(t *testing.T) {
	patterns := []Pattern{
		{Match: "(?i)password:", Respond: "secret123", Hidden: true},
	}

	m, _ := NewMatcher(patterns)

	resp := m.Check("Loading configuration...")
	if resp != nil {
		t.Fatal("expected no match")
	}
}

func TestMatcher_FirstMatchWins(t *testing.T) {
	patterns := []Pattern{
		{Match: "password", Respond: "first", Hidden: true},
		{Match: "pass", Respond: "second", Hidden: false},
	}

	m, _ := NewMatcher(patterns)

	resp := m.Check("enter password now")
	if resp == nil {
		t.Fatal("expected a match")
	}
	if resp.Response != "first" {
		t.Fatalf("expected first match to win, got %q", resp.Response)
	}
}

func TestMatcher_InvalidRegex(t *testing.T) {
	patterns := []Pattern{
		{Match: "[broken", Respond: "x", Hidden: false},
	}

	_, err := NewMatcher(patterns)
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}
