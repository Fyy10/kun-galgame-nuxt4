package main

import (
	"testing"

	"kun-galgame-api/internal/galgame/resourcevocab"
)

func TestPlanAxesFillsAndroidFromKirikiroid2(t *testing.T) {
	p, ok := planAxes(axesRow{
		Type: "game", Legacy: "emulator",
		Runtimes: resourcevocab.Keys{"kirikiroid2"},
	})
	if !ok || !p.SetP || p.SetR {
		t.Fatalf("patch %+v ok=%v", p, ok)
	}
	if !contains(p.Platforms, "and") {
		t.Fatalf("platforms %v", p.Platforms)
	}
}

func TestPlanAxesIsNoopWhenAxesFilled(t *testing.T) {
	_, ok := planAxes(axesRow{
		Type: "game", Legacy: "emulator",
		Platforms: resourcevocab.Keys{"and"},
		Runtimes:  resourcevocab.Keys{"kirikiroid2"},
	})
	if ok {
		t.Fatal("already filled row must be a no-op")
	}
}

func TestPlanAxesUnknownEmulatorGetsOther(t *testing.T) {
	p, ok := planAxes(axesRow{Type: "game", Legacy: "emulator"})
	if !ok || !p.SetR || !contains(p.Runtimes, "other") {
		t.Fatalf("runtimes %+v ok=%v", p, ok)
	}
	if !p.SetP || !contains(p.Platforms, "oth") {
		t.Fatalf("platforms %v", p.Platforms)
	}
}

func TestPlanAxesLeavesVoiceRuntimeEmpty(t *testing.T) {
	_, ok := planAxes(axesRow{Type: "voice", Legacy: "others"})
	if ok {
		t.Fatal("voice has no runtime axis")
	}
}

func TestPlanAxesReplacesOthPlaceholder(t *testing.T) {
	p, ok := planAxes(axesRow{
		Type: "game", Legacy: "others",
		Note:      "PC_Win 汉化",
		Platforms: resourcevocab.Keys{"oth"},
	})
	if !ok || !p.SetR || !contains(p.Runtimes, "native-win") {
		t.Fatalf("runtimes %+v", p)
	}
	if !p.SetP || !contains(p.Platforms, "win") {
		t.Fatalf("platforms %v", p.Platforms)
	}
}
