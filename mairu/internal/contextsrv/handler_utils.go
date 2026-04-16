package contextsrv

import (
	"encoding/json"
	"net/http"
)

func writeJSONError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func marshalMetadata(m map[string]any) []byte {
	if len(m) == 0 {
		return nil
	}
	b, _ := json.Marshal(m)
	return b
}

func writeJSONErrorString(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": message})
}
