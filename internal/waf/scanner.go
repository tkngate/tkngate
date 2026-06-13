package waf

import (
	"bytes"
	"fmt"
	"regexp"
)

var (
	// KnownPromptInjections contains common jailbreak and injection signatures.
	KnownPromptInjections = [][]byte{
		[]byte("ignore all previous instructions"),
		[]byte("ignore previous instructions"),
		[]byte("you are now DAN"),
		[]byte("hypothetical scenario"),
		[]byte("disregard all previous"),
	}

	// PIIRegexes contains compiled patterns for Data Loss Prevention (DLP)
	PIIRegexes = []*regexp.Regexp{
		// Credit Cards (Simple heuristic for 13-19 digit sequences often grouped)
		regexp.MustCompile(`(?i)(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|6(?:011|5[0-9]{2})[0-9]{12}|(?:2131|1800|35\d{3})\d{11})`),
		// Standard API Keys (sk- followed by 32+ alphanumeric characters)
		regexp.MustCompile(`sk-[a-zA-Z0-9]{32,}`),
		// US Social Security Numbers
		regexp.MustCompile(`\b[0-9]{3}-[0-9]{2}-[0-9]{4}\b`),
	}

	RedactionString = []byte("[REDACTED]")
)

// DetectJailbreak scans the payload for known heuristic signatures of prompt injection.
// Returns an error if a threat is detected.
func DetectJailbreak(payload []byte) error {
	lowerPayload := bytes.ToLower(payload)
	for _, signature := range KnownPromptInjections {
		if bytes.Contains(lowerPayload, signature) {
			return fmt.Errorf("request blocked by AI-WAF policy")
		}
	}
	return nil
}

// RedactPII scans the payload and automatically replaces sensitive data with [REDACTED].
func RedactPII(payload []byte) []byte {
	sanitized := payload
	for _, re := range PIIRegexes {
		sanitized = re.ReplaceAll(sanitized, RedactionString)
	}
	return sanitized
}
