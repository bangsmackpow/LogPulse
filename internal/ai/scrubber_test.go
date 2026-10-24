package ai_test

import (
	"strings"
	"testing"

	"logpulse/internal/ai"
)

func TestScrubPII_IPv4AndIPv6(t *testing.T) {
	input := "Failed to connect to 192.168.1.105:8080 and 2001:0db8:85a3:0000:0000:8a2e:0370:7334"
	scrubbed, matches := ai.ScrubPII(input)

	if strings.Contains(scrubbed, "192.168.1.105") {
		t.Errorf("IPv4 was not redacted: %s", scrubbed)
	}
	if strings.Contains(scrubbed, "2001:0db8") {
		t.Errorf("IPv6 was not redacted: %s", scrubbed)
	}
	if len(matches) < 2 {
		t.Errorf("expected at least 2 scrub matches, got %d", len(matches))
	}
}

func TestScrubPII_JWTAndAPIKeys(t *testing.T) {
	input := "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c and api_key=sk-ant-api03-abcdefghijklmn1234567890"
	scrubbed, matches := ai.ScrubPII(input)

	if strings.Contains(scrubbed, "eyJhbGciOi") {
		t.Errorf("JWT token was not redacted: %s", scrubbed)
	}
	if strings.Contains(scrubbed, "sk-ant-api03") {
		t.Errorf("API key was not redacted: %s", scrubbed)
	}
	if len(matches) < 2 {
		t.Errorf("expected at least 2 matches, got %d", len(matches))
	}
}

func TestScrubPII_Emails(t *testing.T) {
	input := "User support@example.com reported failure at admin.dept@corp.internal.net"
	scrubbed, matches := ai.ScrubPII(input)

	if strings.Contains(scrubbed, "support@example.com") {
		t.Errorf("Email was not redacted: %s", scrubbed)
	}
	if len(matches) < 2 {
		t.Errorf("expected 2 email matches, got %d", len(matches))
	}
}
