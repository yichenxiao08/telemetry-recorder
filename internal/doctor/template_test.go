package doctor

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestDoctorTemplateOutputByteIdentical(t *testing.T) {
	b, e := os.ReadFile("testdata/template-before.json")
	if e != nil {
		t.Fatal(e)
	}
	var want []string
	if e = json.Unmarshal(b, &want); e != nil {
		t.Fatal(e)
	}
	got := []string{}
	for _, p := range []Python{{}, {Command: `C:\Python's folder\python.exe`, Adapter: `C:\adapter folder\日本語`}, {Command: `C:\Windows\py.exe`, Arguments: []string{"-3"}, Adapter: `C:\adapter`}} {
		r := Report{SchemaVersion: 1, Binary: Binary{Path: "signalscout.exe", Version: "fixed"}, Python: p, Checks: []Check{{ID: "fixed", Status: Pass, Message: "fixed check"}}, Summary: Summary{Pass: 1}, Examples: Examples("signalscout.exe", p, "127.0.0.1:4318")}
		got = append(got, r.Text())
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("doctor output changed from pre-extraction golden")
	}
}
