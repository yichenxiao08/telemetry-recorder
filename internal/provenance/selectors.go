package provenance

import (
	"math"
	"strconv"
	"strings"
)

// SelectorParts preserves the historical dot split unless the profile opts in.
func SelectorParts(path, syntax string) ([]string, bool) {
	if path == "" || len(path) > 256 {
		return nil, false
	}
	if syntax == "" {
		parts := strings.Split(path, ".")
		return parts, len(parts) <= 8
	}
	if syntax != "escaped-v1" {
		return nil, false
	}
	parts := []string{}
	var part strings.Builder
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '\\':
			i++
			if i == len(path) || path[i] != '.' && path[i] != '\\' {
				return nil, false
			}
			part.WriteByte(path[i])
		case '.':
			if part.Len() == 0 {
				return nil, false
			}
			parts = append(parts, part.String())
			part.Reset()
		default:
			part.WriteByte(path[i])
		}
	}
	if part.Len() == 0 {
		return nil, false
	}
	parts = append(parts, part.String())
	return parts, len(parts) <= 8
}

func GetWithSyntax(values map[string]any, path, syntax string) (any, bool) {
	parts, ok := SelectorParts(path, syntax)
	if !ok {
		return nil, false
	}
	return getParts(values, parts)
}

func GetProfile(values map[string]any, path string, pSyntax string) (any, bool) {
	if pSyntax == "" {
		return Get(values, path)
	}
	return GetWithSyntax(values, path, pSyntax)
}

// Category keeps types distinct. Compare retains its historical string-only
// path; this encoding is used only when an arm contains integer/bool categories.
func Category(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return "string:" + strconv.Quote(v), true
	case bool:
		return "boolean:" + strconv.FormatBool(v), true
	case int:
		return "integer:" + strconv.Itoa(v), true
	case float64:
		if !math.IsNaN(v) && !math.IsInf(v, 0) && v == math.Trunc(v) && math.Abs(v) <= 1<<53 {
			return "integer:" + strconv.FormatFloat(v, 'f', 0, 64), true
		}
	}
	return "", false
}
