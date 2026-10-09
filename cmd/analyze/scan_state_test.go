package main

import (
	"errors"
	"testing"
)

// measurementState is the single decision point that separates "we measured some
// of this" from "we measured nothing". A-2/A-3 of the upstream partial-coverage
// chain depend on this exact table, so it is pinned before any caller exists.
func TestMeasurementStateClassifiesCoverage(t *testing.T) {
	probeErr := errors.New("permission denied")

	tests := []struct {
		name string
		size int64
		err  error
		want scanState
	}{
		{name: "no error, empty dir is still complete", size: 0, err: nil, want: scanComplete},
		{name: "no error, sized", size: 4096, err: nil, want: scanComplete},
		{name: "error with measured bytes stays partial", size: 4096, err: probeErr, want: scanPartial},
		{name: "error with one byte measured stays partial", size: 1, err: probeErr, want: scanPartial},
		{name: "error with nothing measured is unavailable", size: 0, err: probeErr, want: scanUnavailable},
		{name: "negative size never reads as partial", size: -1, err: probeErr, want: scanUnavailable},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := measurementState(tc.size, tc.err); got != tc.want {
				t.Fatalf("measurementState(%d, %v) = %s, want %s", tc.size, tc.err, got, tc.want)
			}
		})
	}
}

// A partial result must keep its byte count; the state is a coverage marker, not
// a reason to discard what was measured.
func TestMeasurementStateKeepsPartialBytesUsable(t *testing.T) {
	const measured = 4096
	state := measurementState(measured, errors.New("open /root/locked: permission denied"))
	if state != scanPartial {
		t.Fatalf("state = %s, want partial", state)
	}
	if state == scanUnavailable {
		t.Fatal("partial must not collapse into unavailable: the caller would drop 4096 measured bytes")
	}
}

// Cache entries written before coverage tracking existed decode State as 0, and
// the zero value must therefore mean complete - otherwise every existing cache
// file would start reporting itself as unmeasured.
func TestScanStateZeroValueMeansComplete(t *testing.T) {
	var zero scanState
	if zero != scanComplete {
		t.Fatalf("zero value = %s, want complete", zero)
	}
	if got := zero.String(); got != "complete" {
		t.Fatalf("zero value String() = %q", got)
	}
}

func TestScanStateString(t *testing.T) {
	tests := []struct {
		state scanState
		want  string
	}{
		{state: scanComplete, want: "complete"},
		{state: scanPartial, want: "partial"},
		{state: scanUnavailable, want: "unavailable"},
		{state: scanState(99), want: "complete"},
	}
	for _, tc := range tests {
		if got := tc.state.String(); got != tc.want {
			t.Fatalf("scanState(%d).String() = %q, want %q", uint8(tc.state), got, tc.want)
		}
	}
}
