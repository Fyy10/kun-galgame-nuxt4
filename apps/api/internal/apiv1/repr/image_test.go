package repr

import (
	"encoding/json"
	"strings"
	"testing"

	"kun-galgame-api/pkg/imageclient"
)

const testHash = "7835f792543f8564cf95e7f84d4828f2a3ef735293f0844bf8ddf8f39371171d"

func TestNewImageFromMeta(t *testing.T) {
	cdn := "https://image.example"
	grade := int16(1)
	meta := &imageclient.ImageMeta{Width: 800, Height: 1131, Thumbhash: "abc+", Sexual: &grade}
	img := NewImage(cdn, testHash, meta)
	if img == nil {
		t.Fatal("nil image")
	}
	wantURL := imageclient.MainURL(cdn, testHash, "webp")
	if img.URL != wantURL || img.Hash != testHash {
		t.Errorf("url/hash %s %s", img.URL, img.Hash)
	}
	if img.Width == nil || *img.Width != 800 || img.Height == nil || *img.Height != 1131 {
		t.Errorf("dims %v %v", img.Width, img.Height)
	}
	if img.Sexual == nil || *img.Sexual != "suggestive" {
		t.Errorf("sexual %v", img.Sexual)
	}
}

func TestNewImageZeroSizeIsNull(t *testing.T) {
	meta := &imageclient.ImageMeta{Width: 0, Height: 0}
	img := NewImage("https://cdn", testHash, meta)
	if img.Width != nil || img.Height != nil || img.Thumbhash != nil {
		t.Errorf("unknown dims must be null: %+v", img)
	}
}

func TestNewImageAbsentSexualIsNull(t *testing.T) {
	img := NewImage("https://cdn", testHash, &imageclient.ImageMeta{})
	if img.Sexual != nil {
		t.Errorf("absent sexual mapped to %v, want null", *img.Sexual)
	}
	img = NewImage("https://cdn", testHash, nil)
	if img.Sexual != nil {
		t.Errorf("nil meta sexual mapped to %v, want null", *img.Sexual)
	}
}

func TestNewImageSexualGrades(t *testing.T) {
	for grade, want := range map[int16]string{0: "safe", 1: "suggestive", 2: "explicit"} {
		g := grade
		img := NewImage("https://cdn", testHash, &imageclient.ImageMeta{Sexual: &g})
		if img.Sexual == nil || *img.Sexual != want {
			t.Errorf("grade %d = %v, want %s", grade, img.Sexual, want)
		}
	}
	other := int16(9)
	img := NewImage("https://cdn", testHash, &imageclient.ImageMeta{Sexual: &other})
	if img.Sexual != nil {
		t.Errorf("unknown grade mapped to %v", *img.Sexual)
	}
}

func TestNewImageFromToken(t *testing.T) {
	cdn := "https://image.example"
	img := NewImageFromToken(cdn, "/image/"+testHash, nil)
	if img == nil || img.Hash != testHash {
		t.Fatalf("token image %+v", img)
	}
	if img.URL != imageclient.MainURL(cdn, testHash, "webp") {
		t.Errorf("url %s", img.URL)
	}
	variant := NewImageFromToken(cdn, "/image/"+testHash+"_mini", nil)
	if variant == nil || variant.URL != imageclient.MainURL(cdn, testHash, "webp") {
		t.Errorf("variant token %+v, want the original image", variant)
	}
	if NewImageFromToken(cdn, "https://hotlink.example/a.png", nil) != nil {
		t.Error("external URL produced an Image")
	}
	for _, hash := range []string{"", "abcd", strings.ToUpper(testHash)} {
		if NewImage(cdn, hash, nil) != nil {
			t.Errorf("hash %q produced an Image", hash)
		}
	}
}

func TestImageJSONKeys(t *testing.T) {
	img := NewImage("https://cdn", testHash, nil)
	raw, err := json.Marshal(img)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"url", "hash", "width", "height", "thumbhash", "sexual"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing key %s: %s", k, raw)
		}
	}
	if _, ok := m["violence"]; ok {
		t.Error("violence present")
	}
	if _, ok := m["source"]; ok {
		t.Error("source present")
	}
	if m["width"] != nil || m["sexual"] != nil {
		t.Errorf("nulls %s", raw)
	}
}
