package redaction

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPolicyFailClosedAndTypedCapture(t *testing.T) {
	p, err := Parse([]byte(`{"version":1,"profiles":{"synthetic":{"fields":{"inputs.frame":{"kind":"number"},"output.lane":{"kind":"enum","values":["left","right"]}}}}}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		profile, path string
		v             any
		want          bool
	}{{"synthetic", "inputs.frame", float64(10), true}, {"synthetic", "inputs.frame", "secret", false}, {"other", "inputs.frame", float64(10), false}, {"synthetic", "output.lane", "left", true}, {"synthetic", "output.lane", "unapproved", false}, {"synthetic", "password", "x", false}} {
		_, ok := p.Value(c.profile, c.path, c.v)
		if ok != c.want {
			t.Fatal(c)
		}
	}
	for _, body := range []string{`{"version":2,"profiles":{}}`, `{"version":1,"unknown":true,"profiles":{}}`, `{"version":1,"profiles":{"x":{"fields":{"password":{"kind":"enum","values":["x"]}}}}}`} {
		if _, err := Parse([]byte(body), nil); err == nil {
			t.Fatal("invalid policy accepted")
		}
	}
	for _, body := range []string{`{"version":2,"inputs":[{"name":"frame","inline":10}]}`, `{"version":1,"version":2}`, `{`} {
		clean := p.Envelope("synthetic", body)
		if strings.Contains(clean, "inline") {
			t.Fatal(clean)
		}
	}
}

func TestPlantedSecretsAndHMAC(t *testing.T) {
	key := []byte(strings.Repeat("k", 32))
	p, err := Parse([]byte(`{"version":1,"key_version":"test-v1","profiles":{"synthetic":{"fields":{"email":{"kind":"hmac","value_class":"email","equality":true},"other":{"kind":"hmac","value_class":"email","equality":true},"inputs.frame":{"kind":"number"}}}}}`), key)
	if err != nil {
		t.Fatal(err)
	}
	a, ok := p.Value("synthetic", "email", "person@example.invalid")
	if !ok || !strings.HasPrefix(a.(string), "hmac-sha256:test-v1:") {
		t.Fatal(a)
	}
	b, _ := p.Value("synthetic", "other", "person@example.invalid")
	if a == b {
		t.Fatal("HMAC not field scoped")
	}
	again, _ := p.Value("synthetic", "email", a)
	if a != again {
		t.Fatal("second gate rehashed token")
	}
	for _, secret := range []string{"Bearer synthetic_token_123456", "sk-syntheticSECRET123456789", "4111111111111111", "SELECT * FROM users WHERE email='person@example.invalid'"} {
		if _, ok := p.Value("synthetic", "email", secret); ok {
			t.Fatal("credential/card/SQL admitted")
		}
	}
	clean := p.Envelope("synthetic", `{"version":1,"id":"Bearer synthetic_token_123456","inputs":[{"name":"frame","inline":10,"capture":"allowed"},{"name":"review","inline":{"excerpt":"person@example.invalid"},"capture":"allowed"}],"output":{"inline":"4111111111111111","capture":"allowed"},"error":"sk-syntheticSECRET123456789","artifact_uri":"https://x.invalid/?token=synthetic"}`)
	if !json.Valid([]byte(clean)) || strings.Contains(clean, "example.invalid") || strings.Contains(clean, "411111") || strings.Contains(clean, "syntheticSECRET") || strings.Contains(clean, "https:") {
		t.Fatal(clean)
	}
}

func TestPermittedArtifactIntegrityIsRedactedNonpersonalOnly(t *testing.T) {
	p, err := Parse([]byte(`{"version":1,"profiles":{"synthetic":{"fields":{"output.artifact":{"kind":"artifact-sha256","values":["synthetic artifact"],"nonpersonal":true,"hash_basis":"redacted"}}}}}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := p.Value("synthetic", "output.artifact", "synthetic artifact")
	if !ok {
		t.Fatal("permitted artifact missing")
	}
	again, ok := p.Value("synthetic", "output.artifact", v)
	if !ok {
		t.Fatal("second gate dropped approved digest")
	}
	b, _ := json.Marshal(again)
	if strings.Contains(string(b), "synthetic artifact") || !strings.Contains(string(b), `"hash_basis":"redacted"`) {
		t.Fatal(string(b))
	}
	if _, ok = p.Value("synthetic", "output.artifact", "person@example.invalid"); ok {
		t.Fatal("unapproved artifact admitted")
	}
}
