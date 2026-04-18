package redact

import "regexp"

type tokenPattern struct {
	kind string
	re   *regexp.Regexp
}

// knownTokenPatterns is the Layer-1 rule set. Patterns are high-precision —
// prefer false negatives over false positives here, since Layer 3 (entropy)
// catches the long tail. Any hit here flips EmbeddingSafe to false.
//
// Sources: patterns derived from Apache-2.0 licensed rule packs
// (github.com/gitleaks/gitleaks, github.com/trufflesecurity/trufflehog).
var knownTokenPatterns = []tokenPattern{
	{"github_pat_classic", regexp.MustCompile(`ghp_[A-Za-z0-9]{36,}`)},
	{"github_pat_fine", regexp.MustCompile(`github_pat_[A-Za-z0-9_]{22,}`)},
	{"github_oauth", regexp.MustCompile(`gh[osu]_[A-Za-z0-9]{36,}`)},
	{"stripe_key", regexp.MustCompile(`sk_(?:live|test)_[A-Za-z0-9]{24,}`)},
	{"stripe_pub", regexp.MustCompile(`pk_(?:live|test)_[A-Za-z0-9]{24,}`)},
	{"aws_access_key", regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`)},
	{"slack_token", regexp.MustCompile(`xox[abpors]-[A-Za-z0-9-]{10,}`)},
	{"google_api_key", regexp.MustCompile(`AIza[0-9A-Za-z_\-]{35}`)},
	{"jwt", regexp.MustCompile(`eyJ[A-Za-z0-9_\-]+\.eyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+`)},
	{"uri_credentials", regexp.MustCompile(`([A-Za-z][A-Za-z0-9+.\-]*://)[^\s:@/]+:([^\s@/]+)@`)},
	{"pem_private_key", regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`)},
}

// scanKnownTokens replaces every match of every pattern with
// "[REDACTED:<kind>]" and returns the cleaned string, the findings it made,
// and whether any Layer-1 pattern hit (which flips EmbeddingSafe off at the
// orchestrator).
func scanKnownTokens(input string) (string, []Finding, bool) {
	out := input
	var findings []Finding
	hit := false
	for _, p := range knownTokenPatterns {
		locs := p.re.FindAllStringIndex(out, -1)
		if len(locs) == 0 {
			continue
		}
		hit = true
		// Rewrite right-to-left so offsets from FindAllStringIndex remain valid.
		replacement := "[REDACTED:" + p.kind + "]"
		for i := len(locs) - 1; i >= 0; i-- {
			loc := locs[i]
			findings = append(findings, Finding{
				Layer: LayerKnownToken,
				Kind:  p.kind,
				Start: loc[0],
				End:   loc[1],
			})
			out = out[:loc[0]] + replacement + out[loc[1]:]
		}
	}
	return out, findings, hit
}
