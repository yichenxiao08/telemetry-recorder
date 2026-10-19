package telemetry

import (
	"encoding/json"
	"fmt"
	"github.com/signalscout/signalscout/internal/model"
	"strings"
)

// OTLP canonical JSON uses enum names, while imported records may use numbers.
func Severity(s model.Signal) int {
	var raw struct {
		Severity any `json:"severityNumber"`
	}
	if json.Unmarshal(s.Raw, &raw) != nil {
		return 0
	}
	if n, ok := raw.Severity.(float64); ok {
		return int(n)
	}
	text, _ := raw.Severity.(string)
	for i, name := range []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL"} {
		for j := range 4 {
			suffix := ""
			if j > 0 {
				suffix = fmt.Sprint(j + 1)
			}
			if text == "SEVERITY_NUMBER_"+name+suffix {
				return 1 + i*4 + j
			}
		}
	}
	return 0
}
func severitySQL() string {
	var b strings.Builder
	b.WriteString(`CASE json_extract(s.data,'$.raw.severityNumber')`)
	for i, name := range []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL"} {
		for j := range 4 {
			suffix := ""
			if j > 0 {
				suffix = fmt.Sprint(j + 1)
			}
			fmt.Fprintf(&b, " WHEN 'SEVERITY_NUMBER_%s%s' THEN %d", name, suffix, 1+i*4+j)
		}
	}
	b.WriteString(` ELSE coalesce(CAST(json_extract(s.data,'$.raw.severityNumber') AS INTEGER),0) END`)
	return b.String()
}
