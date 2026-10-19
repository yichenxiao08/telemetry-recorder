package runs

import "regexp"

// Normalizer owns immutable compiled expressions, avoiding package mutable state.
type Normalizer struct{ expressions []*regexp.Regexp }

func NewNormalizer() *Normalizer {
	patterns := []string{`"[^"\\]*(?:\\.[^"\\]*)*"|'[^']*'`, `(?i)\b[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}\b`, `\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`, `\b-?\d+(?:\.\d+)?\b`}
	n := &Normalizer{}
	for _, pattern := range patterns {
		n.expressions = append(n.expressions, regexp.MustCompile(pattern))
	}
	return n
}

func (n *Normalizer) Template(body string) string {
	for _, expression := range n.expressions {
		body = expression.ReplaceAllString(body, "{}")
	}
	return body
}
