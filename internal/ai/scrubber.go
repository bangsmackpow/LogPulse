package ai

import (
	"regexp"
)

type ScrubMatch struct {
	Type        string `json:"type"`
	Original    string `json:"original"`
	Replacement string `json:"replacement"`
}

var (
	reIPv4 = regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`)
	reIPv6 = regexp.MustCompile(`\b(?:[A-F0-9]{1,4}:){7}[A-F0-9]{1,4}\b|\b(?:[a-f0-9]{1,4}:){7}[a-f0-9]{1,4}\b`)
	reJWT  = regexp.MustCompile(`\beyJ[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+\b`)
	reAPIKey = regexp.MustCompile(`\b(?:sk-[a-zA-Z0-9_-]{20,}|AKIA[0-9A-Z]{16}|AIza[0-9A-Za-z-_]{35})\b`)
	reEmail = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
)

func ScrubPII(text string) (string, []ScrubMatch) {
	var matches []ScrubMatch

	// Scrub JWTs first (high entropy dots)
	text = reJWT.ReplaceAllStringFunc(text, func(m string) string {
		rep := "[REDACTED_JWT]"
		matches = append(matches, ScrubMatch{Type: "JWT", Original: m, Replacement: rep})
		return rep
	})

	// Scrub API Keys
	text = reAPIKey.ReplaceAllStringFunc(text, func(m string) string {
		rep := "[REDACTED_API_KEY]"
		matches = append(matches, ScrubMatch{Type: "API_KEY", Original: m, Replacement: rep})
		return rep
	})

	// Scrub IPv4
	text = reIPv4.ReplaceAllStringFunc(text, func(m string) string {
		rep := "[REDACTED_IP]"
		matches = append(matches, ScrubMatch{Type: "IPv4", Original: m, Replacement: rep})
		return rep
	})

	// Scrub IPv6
	text = reIPv6.ReplaceAllStringFunc(text, func(m string) string {
		rep := "[REDACTED_IPV6]"
		matches = append(matches, ScrubMatch{Type: "IPv6", Original: m, Replacement: rep})
		return rep
	})

	// Scrub Emails
	text = reEmail.ReplaceAllStringFunc(text, func(m string) string {
		rep := "[REDACTED_EMAIL]"
		matches = append(matches, ScrubMatch{Type: "EMAIL", Original: m, Replacement: rep})
		return rep
	})

	return text, matches
}
