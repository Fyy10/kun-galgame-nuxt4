package main

import "kun-galgame-api/internal/galgame/resourcevocab"

type axesRow struct {
	Type      string
	Legacy    string
	Note      string
	Size      string
	Platforms resourcevocab.Keys
	Runtimes  resourcevocab.Keys
}

type axesPatch struct {
	Platforms resourcevocab.Keys
	Runtimes  resourcevocab.Keys
	SetP      bool
	SetR      bool
}

func emptyKeys(k resourcevocab.Keys) bool { return len(k) == 0 }

func placeholderPlats(k resourcevocab.Keys) bool {
	return emptyKeys(k) || (len(k) == 1 && k[0] == "oth")
}

func onlyOther(k resourcevocab.Keys) bool {
	return len(k) == 1 && k[0] == "other"
}

func keysEqual(a, b resourcevocab.Keys) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func dropOthIfConcrete(k resourcevocab.Keys) resourcevocab.Keys {
	out := make(resourcevocab.Keys, 0, len(k))
	for _, p := range k {
		if p != "oth" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return k
	}
	return out
}

func platformsFromRuntimes(runs resourcevocab.Keys) resourcevocab.Keys {
	plats := map[string]bool{}
	for _, r := range runs {
		switch r {
		case "native-win", "winlator", "gamehub":
			plats["win"] = true
		case "native-and", "kirikiroid2", "krkrsdl2", "onscripter",
			"joiplay", "easyrpg", "renpy-android", "tyranor", "tyranor-next":
			plats["and"] = true
		case "native-ios":
			plats["ios"] = true
		}
	}
	out, _ := resourcevocab.Platforms(setKeys(plats))
	return out
}

func planAxes(r axesRow) (axesPatch, bool) {
	plats, runs := r.Platforms, r.Runtimes
	var p axesPatch

	if emptyKeys(runs) {
		g := guessFromText(r.Note, r.Size, r.Legacy)
		switch {
		case len(g.Runtimes) > 0 && !keysEqual(g.Runtimes, r.Runtimes):
			runs = g.Runtimes
			p.Runtimes, p.SetR = runs, true
		case resourcevocab.HasRuntimeAxis(r.Type):
			runs = resourcevocab.Keys{"other"}
			p.Runtimes, p.SetR = runs, true
		}
		gp := dropOthIfConcrete(g.Platforms)
		if p.SetR && placeholderPlats(plats) && !emptyKeys(gp) && !keysEqual(gp, r.Platforms) {
			plats = gp
			p.Platforms, p.SetP = plats, true
		}
	}

	if placeholderPlats(plats) && !emptyKeys(runs) {
		derived := platformsFromRuntimes(runs)
		if emptyKeys(derived) && onlyOther(runs) {
			derived = resourcevocab.Keys{"oth"}
		}
		if !emptyKeys(derived) && !keysEqual(derived, r.Platforms) {
			p.Platforms, p.SetP = derived, true
		}
	}

	return p, p.SetP || p.SetR
}
