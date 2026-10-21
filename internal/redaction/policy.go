// Package redaction implements the opt-in synthetic P0 gate, not consent or authorization.
package redaction

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"
)

type Rule struct {
	Kind        string   `json:"kind"`
	Values      []string `json:"values,omitempty"`
	Equality    bool     `json:"equality,omitempty"`
	ValueClass  string   `json:"value_class,omitempty"`
	Nonpersonal bool     `json:"nonpersonal,omitempty"`
	HashBasis   string   `json:"hash_basis,omitempty"`
}
type Profile struct {
	Fields map[string]Rule `json:"fields"`
}
type Policy struct {
	Version    int                `json:"version"`
	KeyVersion string             `json:"key_version,omitempty"`
	Profiles   map[string]Profile `json:"profiles"`
	key        []byte
}

var sensitive = regexp.MustCompile(`(?i)password|credential|authorization|api.?key|access.?token|secret|cookie|bearer|private.?key`)
var credential = regexp.MustCompile(`(?i)bearer\s|sk[-_]|token[=:]|password[=:]|\b(select|insert|update|delete)\b|\d{11,19}`)
var email = regexp.MustCompile(`(?i)[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}`)
var versionName = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,32}$`)
var fieldPath = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]{0,255}$`)

func Parse(body, key []byte) (*Policy, error) {
	if len(body) > 65536 || !uniqueJSON(body) {
		return nil, fmt.Errorf("policy exceeds 64 KiB")
	}
	var p Policy
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil {
		return nil, fmt.Errorf("invalid capture policy")
	}
	if d.Decode(new(any)) != io.EOF || p.Version != 1 || len(p.Profiles) == 0 || len(p.Profiles) > 32 {
		return nil, fmt.Errorf("unsupported or unbounded capture policy")
	}
	for name, profile := range p.Profiles {
		if !versionName.MatchString(name) || len(profile.Fields) > 128 {
			return nil, fmt.Errorf("invalid profile")
		}
		for path, r := range profile.Fields {
			if !fieldPath.MatchString(path) || sensitive.MatchString(path) {
				return nil, fmt.Errorf("credential/invalid field forbidden")
			}
			switch r.Kind {
			case "number", "boolean":
				if len(r.Values) > 0 || r.Equality {
					return nil, fmt.Errorf("invalid scalar rule")
				}
			case "enum", "artifact-sha256":
				if r.Kind == "artifact-sha256" && (!r.Nonpersonal || r.HashBasis != "redacted") {
					return nil, fmt.Errorf("artifact digest needs nonpersonal approval and redacted basis")
				}
				if len(r.Values) == 0 || len(r.Values) > 32 || r.Equality {
					return nil, fmt.Errorf("finite enum required")
				}
				for _, v := range r.Values {
					if len(v) > 128 || credential.MatchString(v) || email.MatchString(v) || strings.ContainsAny(v, "/?@'\"\\") {
						return nil, fmt.Errorf("unsafe enum literal")
					}
				}
			case "hmac":
				if !r.Equality || r.ValueClass != "email" || len(r.Values) > 0 || len(key) < 32 || !versionName.MatchString(p.KeyVersion) {
					return nil, fmt.Errorf("HMAC requires equality consent, >=32-byte runtime key and key version")
				}
			default:
				return nil, fmt.Errorf("unknown capture rule")
			}
		}
	}
	p.key = append([]byte(nil), key...)
	return &p, nil
}
func (p *Policy) HasProfile(profile string) bool { _, ok := p.Profiles[profile]; return ok }
func (p *Policy) Value(profile, path string, v any) (any, bool) {
	if sensitive.MatchString(path) {
		return nil, false
	}
	r, ok := p.Profiles[profile].Fields[path]
	if !ok { // Containers are traversed only toward explicitly allowlisted leaves.
		if m, yes := v.(map[string]any); yes {
			out := map[string]any{}
			for k, item := range m {
				if clean, allowed := p.Value(profile, path+"."+k, item); allowed {
					out[k] = clean
				}
			}
			return out, len(out) > 0
		}
		return nil, false
	}
	switch r.Kind {
	case "boolean":
		x, yes := v.(bool)
		return x, yes
	case "number":
		x, yes := v.(float64)
		if !yes || math.IsInf(x, 0) || math.IsNaN(x) || math.Abs(x) >= 1e10 {
			return nil, false
		}
		return x, true
	case "enum":
		x, yes := v.(string)
		if yes {
			for _, allowed := range r.Values {
				if x == allowed {
					return x, true
				}
			}
		}
	case "hmac":
		x, yes := v.(string)
		if !yes || len(x) > 256 {
			return nil, false
		}
		prefix := "hmac-sha256:" + p.KeyVersion + ":"
		if strings.HasPrefix(x, prefix) && len(x) == len(prefix)+64 {
			if _, err := hex.DecodeString(strings.TrimPrefix(x, prefix)); err == nil {
				return x, true
			}
		}
		if credential.MatchString(x) || email.FindString(x) != x {
			return nil, false
		}
		mac := hmac.New(sha256.New, p.key)
		_, _ = fmt.Fprintf(mac, "signalscout-p0\x00%s\x00%s\x00%s\x00%s", p.KeyVersion, profile, path, x)
		return prefix + hex.EncodeToString(mac.Sum(nil)), true
	case "artifact-sha256":
		for _, content := range r.Values {
			digest := sha256.Sum256([]byte(content))
			text := hex.EncodeToString(digest[:])
			if x, ok := v.(string); ok && x == content {
				return map[string]any{"sha256": text, "hash_basis": "redacted", "nonpersonal": true}, true
			}
			if x, ok := v.(map[string]any); ok && len(x) == 3 && x["sha256"] == text && x["hash_basis"] == "redacted" && x["nonpersonal"] == true {
				return x, true
			}
		}
	}
	return nil, false
}

// Envelope rebuilds a known schema from permitted fields, never keeping its raw
// JSON alongside the sanitized values. Unknown versions fail closed entirely.
func (p *Policy) Envelope(profile, body string) string {
	unknown := `{"version":0,"capture":"off","detail":"unknown"}`
	if len(body) > 8192 {
		return unknown
	}
	var e struct {
		Version int `json:"version"`
		Inputs  []struct {
			Name    string
			Capture string
			Inline  any
		}
		Output struct {
			Capture string
			Inline  any
		}
	}
	if err := json.Unmarshal([]byte(body), &e); err != nil || e.Version != 1 || len(e.Inputs) > 32 {
		return unknown
	}
	// Reject duplicate keys anywhere; ambiguous policy/record JSON is never trusted.
	if !uniqueJSON([]byte(body)) {
		return unknown
	}
	inputs := []map[string]any{}
	seen := map[string]bool{}
	for _, in := range e.Inputs {
		if seen[in.Name] {
			return unknown
		}
		seen[in.Name] = true
		clean, ok := p.Value(profile, "inputs."+in.Name, in.Inline)
		if in.Capture != "allowed" {
			ok = false
		}
		if ok {
			inputs = append(inputs, map[string]any{"name": in.Name, "capture": "allowed", "inline": clean})
		}
	}
	output := map[string]any{"capture": "off"}
	if clean, ok := p.Value(profile, "output", e.Output.Inline); ok && e.Output.Capture == "allowed" {
		output = map[string]any{"capture": "allowed", "inline": clean}
	}
	b, _ := json.Marshal(map[string]any{"version": 1, "inputs": inputs, "output": output, "omitted_inputs": len(e.Inputs) - len(inputs), "detail": "redacted; identity and semantic metadata unknown"})
	return string(b)
}
func uniqueJSON(body []byte) bool {
	d := json.NewDecoder(bytes.NewReader(body))
	var walk func(int) bool
	walk = func(depth int) bool {
		if depth > 16 {
			return false
		}
		v, err := d.Token()
		if err != nil {
			return false
		}
		delim, ok := v.(json.Delim)
		if !ok {
			return true
		}
		seen := map[string]bool{}
		for d.More() {
			if delim == '{' {
				k, err := d.Token()
				if err != nil {
					return false
				}
				key, ok := k.(string)
				key = strings.ToLower(key)
				if !ok || seen[key] {
					return false
				}
				seen[key] = true
			}
			if !walk(depth + 1) {
				return false
			}
		}
		_, err = d.Token()
		return err == nil
	}
	return walk(0)
}
