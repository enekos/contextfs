package redact

import "regexp"

type argHeuristic struct {
	kind        string
	re          *regexp.Regexp
	replacement string
}

// argHeuristics is the Layer-2 rule set. Each pattern captures the
// flag/header/env prefix in group 1 and retains it; the secret value is
// replaced with a [REDACTED:<kind>] placeholder. Layer 2 never flips
// EmbeddingSafe — only Layer 1 does.
var argHeuristics = []argHeuristic{
	// -H "Authorization: Bearer ..." / -H "X-Api-Key: ..." / similar
	{
		kind:        "auth_header",
		re:          regexp.MustCompile(`(?i)(-H\s+["']?(?:Authorization|X-[A-Za-z-]*(?:Auth|Key|Token)[A-Za-z-]*):\s*)([^"'\s]+(?:\s+[^"'\s]+)?)`),
		replacement: `${1}[REDACTED:auth_header]`,
	},
	// curl -u user:pass
	{
		kind:        "basic_auth",
		re:          regexp.MustCompile(`(-u\s+)([^\s:]+:[^\s]+)`),
		replacement: `${1}[REDACTED:basic_auth]`,
	},
	// --token=VALUE, --password=VALUE, ... (with =)
	{
		kind:        "sensitive_flag_eq",
		re:          regexp.MustCompile(`(?i)(--(?:token|secret|key|password|passwd|pass|auth|credential|api[-_]?key|access[-_]?token|client[-_]?secret)=)([^\s]+)`),
		replacement: `${1}[REDACTED:sensitive_flag]`,
	},
	// --password VALUE, --token VALUE (space-separated)
	{
		kind:        "sensitive_flag_sp",
		re:          regexp.MustCompile(`(?i)(--(?:token|secret|key|password|passwd|pass|auth|credential|api[-_]?key|access[-_]?token|client[-_]?secret)\s+)([^\s]+)`),
		replacement: `${1}[REDACTED:sensitive_flag]`,
	},
	// Inline env prefix: FOO_TOKEN=bar cmd ... (at start or after ;/&/|)
	{
		kind:        "env_prefix",
		re:          regexp.MustCompile(`(?i)((?:^|[;&|]\s*)[A-Z][A-Z0-9_]*(?:TOKEN|SECRET|KEY|PASSWORD|PASSWD|PASS|AUTH|CREDENTIAL|APIKEY|ACCESS_TOKEN|PRIVATE)[A-Z0-9_]*=)([^\s]+)`),
		replacement: `${1}[REDACTED:env_prefix]`,
	},
}

func scanArguments(input string) (string, []Finding) {
	out := input
	var findings []Finding
	for _, h := range argHeuristics {
		for _, loc := range h.re.FindAllStringIndex(out, -1) {
			findings = append(findings, Finding{
				Layer: LayerArgFlag,
				Kind:  h.kind,
				Start: loc[0],
				End:   loc[1],
			})
		}
		out = h.re.ReplaceAllString(out, h.replacement)
	}
	return out, findings
}
