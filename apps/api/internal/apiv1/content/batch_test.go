package content_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/userclient"
)

func TestConvertBatchDedupAndOrder(t *testing.T) {
	src := "[@a](kungal-user:3) ![](/image/" + testHash + ")\n\n[@b](kungal-user:3) ![](/image/" + testHash + "_320)"
	var imageCalls, userCalls int
	var gotHashes []string
	var gotIDs []int
	conv := &content.Converter{
		CDNBase:  testCDN,
		SiteBase: testSite,
		Images: func(hashes []string) map[string]imageclient.ImageMeta {
			imageCalls++
			gotHashes = append([]string(nil), hashes...)
			return map[string]imageclient.ImageMeta{}
		},
		Users: func(ctx context.Context, ids []int) (map[int]userclient.User, error) {
			userCalls++
			gotIDs = append([]int(nil), ids...)
			return map[int]userclient.User{3: {ID: 3, Name: "u3"}}, nil
		},
	}
	docs, err := conv.Convert(context.Background(), []string{src, "hello " + src, src})
	if err != nil {
		t.Fatal(err)
	}
	if imageCalls != 1 {
		t.Errorf("Images called %d times, want 1", imageCalls)
	}
	if userCalls != 1 {
		t.Errorf("Users called %d times, want 1", userCalls)
	}
	if len(gotHashes) != 1 || gotHashes[0] != testHash {
		t.Errorf("hashes %v, want [%s]", gotHashes, testHash)
	}
	if len(gotIDs) != 1 || gotIDs[0] != 3 {
		t.Errorf("ids %v, want [3]", gotIDs)
	}
	if len(docs) != 3 {
		t.Fatalf("len %d", len(docs))
	}
	for i, d := range docs {
		checkDocument(t, d)
		if i == 1 {
			continue
		}
		if marshalDoc(t, docs[0]) != marshalDoc(t, d) {
			t.Errorf("doc %d differs from doc 0", i)
		}
	}
	if !strings.Contains(marshalDoc(t, docs[1]), "hello") {
		t.Error("middle document lost leading text")
	}
}

func TestConvertBatchNoImagesNoMentions(t *testing.T) {
	conv := &content.Converter{
		CDNBase:  testCDN,
		SiteBase: testSite,
		Images: func(hashes []string) map[string]imageclient.ImageMeta {
			t.Error("Images should not be called")
			return nil
		},
		Users: func(ctx context.Context, ids []int) (map[int]userclient.User, error) {
			t.Error("Users should not be called")
			return nil, nil
		},
	}
	docs, err := conv.Convert(context.Background(), []string{"hello", "  "})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("len %d", len(docs))
	}
	checkDocument(t, docs[0])
	checkDocument(t, docs[1])
	if marshalDoc(t, docs[1]) != `{"object":"document","children":[]}` {
		t.Errorf("whitespace doc %s", marshalDoc(t, docs[1]))
	}
}

func TestConvertUsersError(t *testing.T) {
	boom := errors.New("unavailable")
	conv := &content.Converter{
		CDNBase:  testCDN,
		SiteBase: testSite,
		Users: func(ctx context.Context, ids []int) (map[int]userclient.User, error) {
			return nil, boom
		},
	}
	docs, err := conv.Convert(context.Background(), []string{"[@a](kungal-user:3)"})
	if err == nil || !errors.Is(err, boom) {
		t.Fatalf("err %v, want wrapped unavailable", err)
	}
	if docs != nil {
		t.Errorf("docs %v, want nil", docs)
	}
}
