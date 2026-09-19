package content_test

import (
	"context"
	"testing"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/pkg/userclient"
)

func FuzzConvert(f *testing.F) {
	for _, src := range fuzzSeeds() {
		f.Add(src)
	}
	conv := &content.Converter{
		CDNBase:  testCDN,
		SiteBase: testSite,
		Users: func(ctx context.Context, ids []int) (map[int]userclient.User, error) {
			out := make(map[int]userclient.User, len(ids))
			for _, id := range ids {
				out[id] = userclient.User{ID: id, Name: "u"}
			}
			return out, nil
		},
	}
	f.Fuzz(func(t *testing.T, src string) {
		docs, err := conv.Convert(context.Background(), []string{src})
		if err != nil {
			t.Fatalf("Convert: %v", err)
		}
		if len(docs) != 1 {
			t.Fatalf("len=%d", len(docs))
		}
		checkDocument(t, docs[0])
	})
}
