package filesize

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		in, out string
		ok      bool
	}{
		{in: "15 MB", out: "15 MB", ok: true},
		{in: "3.8gb", out: "3.8 GB", ok: true},
		{in: "128mb", out: "128 MB", ok: true},
		{in: "1.50 GB", out: "1.5 GB", ok: true},
		{in: "1GB", out: "1 GB", ok: true},
		{in: "22.59GB", out: "22.59 GB", ok: true},
		{in: ""},
		{in: "820 KB"},
		{in: "很大"},
		{in: "15 MB 左右"},
		{in: "-5 GB"},
		{in: "1.234 GB"},
		{in: "2GB【模拟器可玩】"},
		{in: "GB"},
	}
	for _, c := range cases {
		got, ok := Parse(c.in)
		if ok != c.ok {
			t.Fatalf("Parse(%q) ok=%v want %v", c.in, ok, c.ok)
		}
		if c.ok && got != c.out {
			t.Errorf("Parse(%q) = %q want %q", c.in, got, c.out)
		}
	}
}
