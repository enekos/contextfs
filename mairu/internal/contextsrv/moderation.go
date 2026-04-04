package contextsrv

import (
	"regexp"
	"strings"
)

type ModerationResult struct {
	Status  string
	Reasons []string
}

var (
	softPatterns = []struct {
		label string
		re    *regexp.Regexp
	}{
		{label: "contains credential-like token", re: regexp.MustCompile(`(?i)(password|passwd|api[_-]?key|token)\s*[:=]`)},
		{label: "contains pii-like field", re: regexp.MustCompile(`(?i)(ssn|social security|credit card|cvv)\b`)},
	}
	hardPatterns = []struct {
		label string
		re    *regexp.Regexp
	}{
		{label: "contains private key material", re: regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----`)},
	}
)

func ModerateContent(content string) ModerationResult {
	text := strings.TrimSpace(content)
	if text == "" {
		return ModerationResult{Status: ModerationStatusClean}
	}

	var reasons []string
	for _, p := range hardPatterns {
		if p.re.MatchString(text) {
			reasons = append(reasons, p.label)
		}
	}
	if len(reasons) > 0 {
		return ModerationResult{Status: ModerationStatusRejectHard, Reasons: reasons}
	}

	reasons = reasons[:0]
	for _, p := range softPatterns {
		if p.re.MatchString(text) {
			reasons = append(reasons, p.label)
		}
	}
	if len(reasons) > 0 {
		return ModerationResult{Status: ModerationStatusFlaggedSoft, Reasons: reasons}
	}
	return ModerationResult{Status: ModerationStatusClean}
}
