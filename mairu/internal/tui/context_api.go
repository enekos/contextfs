package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func (m *model) contextAPIBase() string {
	base := strings.TrimSpace(os.Getenv("MAIRU_CONTEXT_SERVER_URL"))
	if base == "" {
		base = "http://localhost:8788"
	}
	return strings.TrimRight(base, "/")
}

func (m *model) contextToken() string {
	return strings.TrimSpace(os.Getenv("MAIRU_CONTEXT_SERVER_TOKEN"))
}

func (m *model) contextGet(path string, qs map[string]string) ([]byte, error) {
	u, err := url.Parse(m.contextAPIBase() + path)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for k, v := range qs {
		if strings.TrimSpace(v) == "" {
			continue
		}
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	token := m.contextToken()
	if token != "" {
		req.Header.Set("X-Context-Token", token)
	}
	return doContextRequest(req)
}

func (m *model) contextPost(path string, payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, m.contextAPIBase()+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	token := m.contextToken()
	if token != "" {
		req.Header.Set("X-Context-Token", token)
	}
	return doContextRequest(req)
}

func doContextRequest(req *http.Request) ([]byte, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("context api %s %s failed (%d): %s", req.Method, req.URL.Path, resp.StatusCode, string(body))
	}
	return body, nil
}

func prettyJSON(raw []byte) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(out)
}
