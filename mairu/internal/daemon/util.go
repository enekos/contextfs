package daemon

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
)

func mustJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func mustRel(base, p string) (string, error) {
	return filepath.Rel(base, p)
}

func hashText(v string) string {
	sum := sha1.Sum([]byte(v))
	return hex.EncodeToString(sum[:])
}
