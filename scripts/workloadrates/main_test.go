package main

import (
	"testing"
	"time"
)

func TestSlidingPeak(t *testing.T) {
	for _, tc := range []struct {
		xs   []int64
		want int
	}{{nil, 0}, {[]int64{0, 1, int64(time.Second)}, 2}, {[]int64{0, int64(time.Second)}, 1}} {
		if got := peak(tc.xs); got != tc.want {
			t.Fatalf("%v: %d want %d", tc.xs, got, tc.want)
		}
	}
}
