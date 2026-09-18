package apiv1

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

type pageIn struct {
	collect.Page
	collect.Total
	Sort string `query:"sort" enum:"created_desc,bumped_desc" doc:"Sort token for this collection."`
}

type pageOut struct {
	Body repr.List[string]
}

func registerPage(api huma.API) {
	huma.Register(api, Public(huma.Operation{
		OperationID: "testPage",
		Method:      http.MethodGet,
		Path:        "/_test/page",
		Summary:     "Test page",
		Description: "Collection helper probe.",
	}), func(_ context.Context, in *pageIn) (*pageOut, error) {
		fp := collect.Fingerprint(in.Sort)
		if _, err := collect.DecodeCursor(in.Cursor, in.Sort, fp); err != nil {
			return nil, err
		}
		return &pageOut{Body: repr.NewList([]string{strconv.Itoa(in.Limit)}, nil)}, nil
	})
}

func TestLimitAbove100IsLimitTooLarge(t *testing.T) {
	app, _ := newTestAPI(t, Deps{}, registerPage)
	resp := do(t, app, http.MethodGet, "/api/v1/_test/page?limit=101", "", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d body %s", resp.StatusCode, readBody(t, resp))
	}
	p := decodeProblem(t, resp)
	if p.Code != problem.CodeLimitTooLarge {
		t.Fatalf("code %s, want LIMIT_TOO_LARGE", p.Code)
	}
}

func TestIncludeTotalOneIsInvalidParameter(t *testing.T) {
	app, _ := newTestAPI(t, Deps{}, registerPage)
	resp := do(t, app, http.MethodGet, "/api/v1/_test/page?include_total=1", "", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d body %s", resp.StatusCode, readBody(t, resp))
	}
	p := decodeProblem(t, resp)
	if p.Code != problem.CodeInvalidParameter {
		t.Fatalf("code %s, want INVALID_PARAMETER", p.Code)
	}
	if len(p.Errors) == 0 || p.Errors[0].Parameter == nil || *p.Errors[0].Parameter != "include_total" {
		t.Fatalf("errors %+v", p.Errors)
	}
}

func TestIncludeTotalTrueIsAccepted(t *testing.T) {
	app, _ := newTestAPI(t, Deps{}, registerPage)
	resp := do(t, app, http.MethodGet, "/api/v1/_test/page?include_total=true", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d body %s", resp.StatusCode, readBody(t, resp))
	}
}

func TestCursorWrongSortIsInvalidCursor(t *testing.T) {
	app, _ := newTestAPI(t, Deps{}, registerPage)
	fp := collect.Fingerprint("created_desc")
	cur := collect.EncodeCursor("created_desc", fp, "1")
	resp := do(t, app, http.MethodGet, "/api/v1/_test/page?sort=bumped_desc&cursor="+cur, "", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d body %s", resp.StatusCode, readBody(t, resp))
	}
	p := decodeProblem(t, resp)
	if p.Code != problem.CodeInvalidCursor {
		t.Fatalf("code %s, want INVALID_CURSOR", p.Code)
	}
}

func TestUnknownSortIsUnknownSort(t *testing.T) {
	app, _ := newTestAPI(t, Deps{}, registerPage)
	resp := do(t, app, http.MethodGet, "/api/v1/_test/page?sort=nope", "", nil)
	if p := decodeProblem(t, resp); resp.StatusCode != http.StatusBadRequest || p.Code != problem.CodeUnknownSort {
		t.Fatalf("%d %s, want 400 UNKNOWN_SORT", resp.StatusCode, p.Code)
	}
}

func TestMalformedCursorIsInvalidCursor(t *testing.T) {
	app, _ := newTestAPI(t, Deps{}, registerPage)
	for _, cur := range []string{"nope", "cur_bm90LWpzb24"} {
		resp := do(t, app, http.MethodGet, "/api/v1/_test/page?cursor="+cur, "", nil)
		p := decodeProblem(t, resp)
		if resp.StatusCode != http.StatusBadRequest || p.Code != problem.CodeInvalidCursor {
			t.Errorf("cursor %s = %d %s, want 400 INVALID_CURSOR", cur, resp.StatusCode, p.Code)
		}
		if len(p.Errors) != 1 || p.Errors[0].Parameter == nil || *p.Errors[0].Parameter != "cursor" || p.Errors[0].Reason != problem.ReasonInvalidFormat {
			t.Errorf("cursor %s errors %+v", cur, p.Errors)
		}
	}
}

func TestCursorAndLimitDefaults(t *testing.T) {
	app, _ := newTestAPI(t, Deps{}, registerPage)
	cur := collect.EncodeCursor("bumped_desc", collect.Fingerprint("bumped_desc"), "1")
	for query, want := range map[string]string{
		"":                                "20",
		"?limit=100":                      "100",
		"?sort=bumped_desc&cursor=" + cur: "20",
	} {
		resp := do(t, app, http.MethodGet, "/api/v1/_test/page"+query, "", nil)
		var list struct{ Items []string }
		if err := json.Unmarshal(readBody(t, resp), &list); err != nil || resp.StatusCode != http.StatusOK || len(list.Items) != 1 || list.Items[0] != want {
			t.Errorf("%q = %d %v, want limit %s", query, resp.StatusCode, list.Items, want)
		}
	}
}

func TestStrictBooleanRejectsSynonyms(t *testing.T) {
	app, _ := newTestAPI(t, Deps{}, registerPage)
	for _, v := range []string{"TRUE", "t", "yes", "on", "0", "true&include_total=1"} {
		resp := do(t, app, http.MethodGet, "/api/v1/_test/page?include_total="+v, "", nil)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s status %d", v, resp.StatusCode)
		}
	}
}
