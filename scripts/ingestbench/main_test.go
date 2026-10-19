package main

import (
	"context"
	"reflect"
	"testing"
)

func TestPayloadGenerationIsSeeded(t *testing.T) {
	for _, kind := range []string{"span", "log", "metric"} {
		first, err := payloads(kind, 17, 8, 1)
		if err != nil {
			t.Fatal(err)
		}
		second, err := payloads(kind, 17, 8, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(first) != 3 || !reflect.DeepEqual(first, second) {
			t.Fatalf("%s payloads not reproducible", kind)
		}
		other, err := payloads(kind, 17, 8, 2)
		if err != nil {
			t.Fatal(err)
		}
		if reflect.DeepEqual(first, other) {
			t.Fatalf("%s seed has no effect", kind)
		}
	}
}

func TestMeasurementVerifiesPersistedCounts(t *testing.T) {
	for _, kind := range []string{"span", "log", "metric"} {
		t.Run(kind, func(t *testing.T) {
			m, err := measure(context.Background(), kind, 17, 8, 1)
			if err != nil {
				t.Fatal(err)
			}
			if m.Records != 17 || m.PerSecond <= 0 {
				t.Fatalf("invalid measurement: %+v", m)
			}
		})
	}
}
