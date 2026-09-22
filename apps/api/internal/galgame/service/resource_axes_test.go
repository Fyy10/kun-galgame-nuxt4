package service

import "testing"

func TestParseResourceAxesRequiresRuntimeForGame(t *testing.T) {
	_, err := parseResourceAxes("game", "", "", []string{"zh-cn"}, []string{"win"}, nil, "", "")
	if err == nil {
		t.Fatal("game without runtime must fail")
	}
	got, err := parseResourceAxes("game", "", "官方最新", []string{"zh-cn"}, []string{"win"}, []string{"native-win"}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Platform != "windows" || got.Language != "zh-cn" {
		t.Fatalf("compat scalars %q %q", got.Platform, got.Language)
	}
}

func TestParseResourceAxesOstSkipsRuntime(t *testing.T) {
	got, err := parseResourceAxes("ost", "", "", []string{"ja-jp"}, []string{"win"}, []string{"native-win"}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Runtimes) != 0 {
		t.Fatalf("ost kept runtimes %v", got.Runtimes)
	}
}

func TestParseResourceAxesLegacyWindows(t *testing.T) {
	got, err := parseResourceAxes("game", "", "", nil, nil, nil, "zh-cn", "windows")
	if err != nil {
		t.Fatal(err)
	}
	if got.Platform != "windows" || len(got.Runtimes) != 1 || got.Runtimes[0] != "native-win" {
		t.Fatalf("%+v", got)
	}
}
