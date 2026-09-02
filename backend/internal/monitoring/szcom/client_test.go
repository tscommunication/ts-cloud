package szcom

import (
	"strings"
	"testing"
)

func TestResolveLearnedMACWithObservedTruncatedShape(
	t *testing.T,
) {
	run := func(
		command string,
	) (
		string,
		error,
	) {
		if strings.Contains(
			command,
			"pon1 12 eth 1",
		) {
			return `
==================================================================
  Index  MAC
==================================================================
  1      00:40:EE:15:73:EE
==================================================================
Belnagor_OLT#`, nil
		}

		return "There is not any MAC address record!\nBelnagor_OLT#", nil
	}

	got, err := resolveLearnedMACWithRunner(
		1,
		[]int{1, 8, 11, 12, 15},
		"40:EE:15:73:EE:C9",
		run,
	)
	if err != nil {
		t.Fatal(err)
	}

	if got == nil {
		t.Fatal("expected SZCOM learned MAC match")
	}

	if got.PONNo != 1 ||
		got.ONUNo != 12 ||
		got.ETHPort != 1 ||
		got.MACAddress != "40:EE:15:73:EE:C9" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestResolveLearnedMACRejectsAmbiguousPrefix(
	t *testing.T,
) {
	run := func(
		command string,
	) (
		string,
		error,
	) {
		if strings.Contains(command, " 12 eth 1") ||
			strings.Contains(command, " 13 eth 1") {
			return "1  00:40:EE:15:73:EE\nOLT#", nil
		}

		return "no records\nOLT#", nil
	}

	got, err := resolveLearnedMACWithRunner(
		1,
		[]int{12, 13},
		"40:EE:15:73:EE:C9",
		run,
	)
	if err != nil {
		t.Fatal(err)
	}

	if got != nil {
		t.Fatalf(
			"expected ambiguous match to soft-fail, got %+v",
			got,
		)
	}
}

func TestOutputMatchesTargetMACExact(t *testing.T) {
	if !outputMatchesTargetMAC(
		"1  40:EE:15:73:EE:C9",
		"40:EE:15:73:EE:C9",
	) {
		t.Fatal("expected exact MAC match")
	}
}
