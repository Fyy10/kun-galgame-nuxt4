package content_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/userclient"
)

const (
	testCDN  = "https://cdn.example"
	testSite = "https://www.kungal.com"
	testHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

func defaultConverter() *content.Converter {
	return &content.Converter{
		CDNBase:  testCDN,
		SiteBase: testSite,
		Images: func(hashes []string) map[string]imageclient.ImageMeta {
			return map[string]imageclient.ImageMeta{}
		},
		Users: func(ctx context.Context, ids []int) (map[int]userclient.User, error) {
			out := make(map[int]userclient.User, len(ids))
			for _, id := range ids {
				out[id] = userclient.User{ID: id, Name: "u" + strconv.Itoa(id)}
			}
			return out, nil
		},
	}
}

func convertOne(t *testing.T, conv *content.Converter, src string) content.ContentDocument {
	t.Helper()
	if conv == nil {
		conv = defaultConverter()
	}
	docs, err := conv.Convert(context.Background(), []string{src})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("len=%d", len(docs))
	}
	checkDocument(t, docs[0])
	return docs[0]
}

func marshalDoc(t *testing.T, doc content.ContentDocument) string {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

type convertCase struct {
	name string
	src  string
	want string
	conv *content.Converter
}

func TestConvertRules(t *testing.T) {
	for _, c := range allConvertCases() {
		t.Run(c.name, func(t *testing.T) {
			got := marshalDoc(t, convertOne(t, c.conv, c.src))
			if got != c.want {
				t.Errorf("src %q\n got %s\nwant %s", c.src, got, c.want)
			}
		})
	}
}

func fuzzSeeds() []string {
	var out []string
	for _, c := range allConvertCases() {
		out = append(out, c.src)
	}
	return out
}
