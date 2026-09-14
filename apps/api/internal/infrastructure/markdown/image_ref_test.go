package markdown

import (
	"regexp"
	"strings"
	"testing"
)

func TestLegacyStickerMapIntegrity(t *testing.T) {
	if n := len(legacyStickerHashByKey); n != 498 {
		t.Fatalf("got %d entries, want 498", n)
	}

	hex := regexp.MustCompile(`^[0-9a-f]{64}$`)
	counts := map[int]int{}
	seen := make(map[string]struct{}, 498)
	for k, h := range legacyStickerHashByKey {
		counts[k.pack]++
		if !hex.MatchString(h) {
			t.Errorf("pack %d position %d: hash not 64 lowercase hex", k.pack, k.position)
		}
		if _, dup := seen[h]; dup {
			t.Errorf("pack %d position %d: duplicate hash", k.pack, k.position)
		}
		seen[h] = struct{}{}
	}
	if len(seen) != 498 {
		t.Errorf("distinct hashes %d, want 498", len(seen))
	}

	wantCounts := map[int]int{1: 80, 2: 80, 3: 80, 4: 80, 5: 80, 6: 80, 7: 18}
	for pack := 1; pack <= 7; pack++ {
		if counts[pack] != wantCounts[pack] {
			t.Errorf("pack %d: %d entries, want %d", pack, counts[pack], wantCounts[pack])
		}
	}
}

func TestResolveLegacyStickerRefs(t *testing.T) {
	hash, ok := LegacyStickerHash(1, 1)
	if !ok {
		t.Fatal("LegacyStickerHash(1, 1) missing")
	}

	in := "![s](https://sticker.kungal.com/stickers/KUNgal1/1.webp)"
	got := ResolveLegacyStickerRefs(in)
	want := "![s](/image/" + hash + "_320)"
	if got != want {
		t.Errorf("KUNgal1/1:\n got %q\nwant %q", got, want)
	}

	outOfPack := "https://sticker.kungal.com/stickers/KUNgal9/1.webp"
	if got := ResolveLegacyStickerRefs(outOfPack); got != outOfPack {
		t.Errorf("KUNgal9/1: got %q, want unchanged", got)
	}

	outOfPos := "https://sticker.kungal.com/stickers/KUNgal7/19.webp"
	if got := ResolveLegacyStickerRefs(outOfPos); got != outOfPos {
		t.Errorf("KUNgal7/19: got %q, want unchanged", got)
	}

	plain := "no stickers ![x](/image/" + testHash + ")"
	if got := ResolveLegacyStickerRefs(plain); got != plain {
		t.Errorf("no legacy URL: got %q, want unchanged", got)
	}
}

func TestNormalizeImageRefs(t *testing.T) {
	main := "https://image.kungal.iloveren.link/78/35/" + testHash + ".webp"
	v320 := "https://image.kungal.iloveren.link/78/35/" + testHash + "_320.webp"
	mainOld := "https://image.kungal.com/78/35/" + testHash + ".webp"
	v320Old := "https://image.kungal.com/78/35/" + testHash + "_320.webp"
	token := "/image/" + testHash
	tokenV := "/image/" + testHash + "_320"

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"main iloveren", main, token},
		{"variant iloveren", v320, tokenV},
		{"main kungal.com", mainOld, token},
		{"variant kungal.com", v320Old, tokenV},
		{"http main", "http://image.kungal.com/78/35/" + testHash + ".webp", token},
		{"wrong shards", "https://image.kungal.iloveren.link/00/00/" + testHash + ".webp",
			"https://image.kungal.iloveren.link/00/00/" + testHash + ".webp"},
		{"foreign host", "https://example.com/78/35/" + testHash + ".webp",
			"https://example.com/78/35/" + testHash + ".webp"},
		{"uppercase hex", "https://image.kungal.com/78/35/" + strings.ToUpper(testHash) + ".webp",
			"https://image.kungal.com/78/35/" + strings.ToUpper(testHash) + ".webp"},
		{"non-webp", "https://image.kungal.com/78/35/" + testHash + ".png",
			"https://image.kungal.com/78/35/" + testHash + ".png"},
		{"already token", token, token},
		{"already token variant", tokenV, tokenV},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NormalizeImageRefs(c.in); got != c.want {
				t.Errorf("NormalizeImageRefs(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}

	for _, c := range cases {
		t.Run("idempotent "+c.name, func(t *testing.T) {
			once := NormalizeStoredContent(c.in)
			twice := NormalizeStoredContent(once)
			if once != twice {
				t.Errorf("NormalizeStoredContent not idempotent:\n once %q\n twice %q", once, twice)
			}
		})
	}
}

func TestNormalizeStoredContentIdempotentLegacy(t *testing.T) {
	cases := []string{
		"https://sticker.kungal.com/stickers/KUNgal1/1.webp",
		"https://sticker.kungal.com/stickers/KUNgal9/1.webp",
		"https://sticker.kungal.com/stickers/KUNgal7/19.webp",
		"![s](https://sticker.kungal.com/stickers/KUNgal1/1.webp) and " +
			"https://image.kungal.iloveren.link/78/35/" + testHash + "_320.webp",
	}
	for _, in := range cases {
		once := NormalizeStoredContent(in)
		twice := NormalizeStoredContent(once)
		if once != twice {
			t.Errorf("NormalizeStoredContent not idempotent for %q:\n once %q\n twice %q", in, once, twice)
		}
	}
}

func TestRenderResolvesLegacySticker(t *testing.T) {
	SetContentImageCDNBase("https://image.kungal.iloveren.link")
	defer SetContentImageCDNBase("")

	hash, ok := LegacyStickerHash(1, 1)
	if !ok {
		t.Fatal("LegacyStickerHash(1, 1) missing")
	}
	wantSrc := "https://image.kungal.iloveren.link/" + hash[:2] + "/" + hash[2:4] + "/" + hash + "_320.webp"
	md := "![s](https://sticker.kungal.com/stickers/KUNgal1/1.webp)"

	for _, r := range []struct {
		name string
		fn   func(string) string
	}{
		{"Render", Render},
		{"RenderHardWrap", RenderHardWrap},
		{"RenderInline", RenderInline},
	} {
		t.Run(r.name, func(t *testing.T) {
			out := r.fn(md)
			if !strings.Contains(out, wantSrc) {
				t.Errorf("%s: want resolved src %q in output\n got: %s", r.name, wantSrc, out)
			}
			if strings.Contains(out, `src="/image/`) {
				t.Errorf("%s: raw /image/ token leaked unresolved\n got: %s", r.name, out)
			}
			if strings.Contains(out, "sticker.kungal.com") {
				t.Errorf("%s: dead sticker host left in output\n got: %s", r.name, out)
			}
		})
	}
}
