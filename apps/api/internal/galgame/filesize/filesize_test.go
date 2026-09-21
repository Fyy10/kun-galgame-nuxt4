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

func TestExtract(t *testing.T) {
	cases := []struct {
		in, out string
		ok      bool
	}{
		{in: "2GB【模拟器可玩】", out: "2 GB", ok: true},
		{in: "【PC+安卓直装+NS+iOS+安卓TY模拟器版】23.43GB", out: "23.43 GB", ok: true},
		{in: "【「CLOCKUP」（钟表社）Galgame 合集&合集包】101GB", out: "101 GB", ok: true},
		{in: "【「あざらしそふと」（海豹社） Galgame 合集】199GB + 317GB", out: "317 GB", ok: true},
		{in: "3,04GB", out: "3.04 GB", ok: true},
		{in: "3（不限速）gb", out: "3 GB", ok: true},
		{in: "6.7(PC+Ty模拟器，附全CG存档)GB", out: "6.7 GB", ok: true},
		{in: "2.6【汉化本体】gb", out: "2.6 GB", ok: true},
		{in: "4.3.GB", out: "4.3 GB", ok: true},
		{in: "939.MB", out: "939 MB", ok: true},
		{in: "20+GB", out: "20 GB", ok: true},
		{in: "664MMB", out: "664 MB", ok: true},
		{in: "10GGB", out: "10 GB", ok: true},
		{in: "300kb", out: "0.29 MB", ok: true},
		{in: "全cg存档 10MB", out: "10 MB", ok: true},
		{in: "1-6全系列汉化无码49GB", out: "49 GB", ok: true},
		{in: "PC+直装+模拟器3部合集7.91gb", out: "7.91 GB", ok: true},
		{in: "3.4·GB", out: "3.4 GB", ok: true},
		{in: "4.3[PC+盖世游戏模拟器Ⅰ全CG+v1.9.3补丁]GB", out: "4.3 GB", ok: true},
		{in: "1014.MB", out: "1014 MB", ok: true},
		{in: "【「夜のひつじ」（夜羊社） Galgame合集&合集包】∞GB"},
		{in: "GB"},
		{in: "20+GB extra text that is not a size", out: "20 GB", ok: true},
	}
	for _, c := range cases {
		got, ok := Extract(c.in)
		if ok != c.ok {
			t.Fatalf("Extract(%q) ok=%v want %v (got %q)", c.in, ok, c.ok, got)
		}
		if c.ok && got != c.out {
			t.Errorf("Extract(%q) = %q want %q", c.in, got, c.out)
		}
	}
}
