package engine

import (
	"fmt"
	"regexp"
)

// Pattern represents a prompt pattern to match and respond to.
type Pattern struct {
	Match   string
	Respond string
	Hidden  bool
}

type MatchResult struct {
	Response string
	Hidden   bool
}

type compiledPattern struct {
	regex    *regexp.Regexp
	response string
	hidden   bool
}

type Matcher struct {
	patterns []compiledPattern
}

func NewMatcher(patterns []Pattern) (*Matcher, error) {
	compiled := make([]compiledPattern, len(patterns))
	for i, p := range patterns {
		re, err := regexp.Compile(p.Match)
		if err != nil {
			return nil, fmt.Errorf("compiling pattern %q: %w", p.Match, err)
		}
		compiled[i] = compiledPattern{
			regex:    re,
			response: p.Respond,
			hidden:   p.Hidden,
		}
	}
	return &Matcher{patterns: compiled}, nil
}

func (m *Matcher) Check(line string) *MatchResult {
	for _, p := range m.patterns {
		if p.regex.MatchString(line) {
			return &MatchResult{
				Response: p.response,
				Hidden:   p.hidden,
			}
		}
	}
	return nil
}
