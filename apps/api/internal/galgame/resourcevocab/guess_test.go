package resourcevocab

import "testing"

func TestGuessFromEmulatorNote(t *testing.T) {
	g := GuessFromText("【PC+安卓直装+KR&TY模拟器双端】", "11.06GB", "emulator")
	if !contains(g.Runtimes, "tyranor") || !contains(g.Runtimes, "kirikiroid2") ||
		!contains(g.Runtimes, "native-and") {
		t.Fatalf("runtimes %v", g.Runtimes)
	}
	if !contains(g.Platforms, "win") || !contains(g.Platforms, "and") {
		t.Fatalf("platforms %v", g.Platforms)
	}
}

func TestGuessDoesNotTripOnEnglishTy(t *testing.T) {
	g := GuessFromText("quality entity notes", "2 GB", "windows")
	if contains(g.Runtimes, "tyranor") {
		t.Fatalf("false tyranor %v", g.Runtimes)
	}
}

func contains(keys Keys, v string) bool {
	for _, k := range keys {
		if k == v {
			return true
		}
	}
	return false
}
