package app

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/pkg/problem"
)

type engageListJSON struct {
	Object     string            `json:"object"`
	NextCursor *string           `json:"next_cursor"`
	Items      []json.RawMessage `json:"items"`
}

func decodeEngageList(t *testing.T, body []byte) engageListJSON {
	t.Helper()
	var out engageListJSON
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("list: %v\n%s", err, body)
	}
	return out
}

func (f *engageFix) seedListHistory(t *testing.T) {
	t.Helper()
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	users := []int{e1UserBob, e1UserCarol, e1UserDave, e1UserStaff, e1UserPoor, e1UserBanned, e1UserAlice}
	toks := []string{"like", "heart", "fire", "clap", "party", "eyes", "love"}
	for i, u := range users {
		at := base.Add(time.Duration(i) * time.Second)
		f.runSQL(t, `INSERT INTO topic_reaction (topic_id, user_id, reaction, created) VALUES (?, ?, ?, ?)`,
			e1TopicLists, u, toks[i], at)
		f.runSQL(t, `INSERT INTO topic_upvote (topic_id, user_id, description, created, updated) VALUES (?, ?, ?, ?, ?)`,
			e1TopicLists, u, toks[i], at, at)
		f.runSQL(t, `INSERT INTO topic_reply_reaction (topic_reply_id, user_id, reaction, created) VALUES (?, ?, ?, ?)`,
			e1ReplyReact, u, toks[i], at)
	}
}

func (f *engageFix) walkListIDs(t *testing.T, rawURL, session, spec string) []string {
	t.Helper()
	var got []string
	cursor := ""
	for range 20 {
		path := rawURL + "?limit=2"
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		resp, body := f.do(t, http.MethodGet, path, session, spec, nil, nil)
		if resp.StatusCode != 200 {
			t.Fatalf("page %s: %d %s", path, resp.StatusCode, body)
		}
		list := decodeEngageList(t, body)
		seen := map[string]bool{}
		for _, raw := range list.Items {
			item := jsonObj(t, raw)
			id, _ := item["id"].(string)
			if id == "" {
				t.Fatalf("item without id %s", raw)
			}
			if seen[id] {
				t.Fatalf("duplicate %s on page %s", id, path)
			}
			seen[id] = true
			got = append(got, id)
		}
		if list.NextCursor == nil {
			return got
		}
		cursor = *list.NextCursor
	}
	t.Fatal("list did not terminate")
	return nil
}

func TestV1TopicUpvotesListPagingParity(t *testing.T) {
	f := newEngageFix(t)
	f.seedListHistory(t)
	spec := "/topics/{topic_id}/upvotes"
	base := "/api/v1/topics/" + strID(e1TopicLists) + "/upvotes"
	want := f.sqlIDs(t, `SELECT id FROM topic_upvote WHERE topic_id = ? AND user_id <> ? ORDER BY created DESC, id DESC`,
		e1TopicLists, e1UserBanned)
	got := f.walkListIDs(t, base, "", spec)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v\nwant %v", got, want)
	}

	resp, body := f.do(t, http.MethodGet, base+"?limit=2", "", spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	list := decodeEngageList(t, body)
	if len(list.Items) != 2 || list.NextCursor == nil {
		t.Fatalf("first page should fill after skipping banned: %s", body)
	}
	first := jsonObj(t, list.Items[0])
	second := jsonObj(t, list.Items[1])
	if first["object"] != "topic_upvote" || first["topic_id"] != strID(e1TopicLists) {
		t.Fatalf("first item %+v", first)
	}
	upvoter, _ := first["upvoter"].(map[string]any)
	if first["id"] != want[0] || upvoter["id"] != strID(e1UserAlice) || first["note"] != "love" {
		t.Fatalf("newest %+v", first)
	}
	if _, ok := first["created_at"].(string); !ok {
		t.Fatalf("created_at %v", first["created_at"])
	}
	if second["note"] != "party" {
		t.Fatalf("page filled with banned row: %s", body)
	}
}

func TestV1TopicReactionsListPagingParity(t *testing.T) {
	f := newEngageFix(t)
	f.seedListHistory(t)
	spec := "/topics/{topic_id}/reactions"
	base := "/api/v1/topics/" + strID(e1TopicLists) + "/reactions"
	want := f.sqlIDs(t, `SELECT id FROM topic_reaction WHERE topic_id = ? AND user_id <> ? ORDER BY created DESC, id DESC`,
		e1TopicLists, e1UserBanned)
	got := f.walkListIDs(t, base, f.asBob(t), spec)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v\nwant %v", got, want)
	}
	resp, body := f.do(t, http.MethodGet, base+"?limit=2", "", spec, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	list := decodeEngageList(t, body)
	if len(list.Items) != 2 {
		t.Fatalf("banned skip did not fill page %s", body)
	}
	item := jsonObj(t, list.Items[0])
	reactor, _ := item["reactor"].(map[string]any)
	if item["object"] != "reaction" || item["reaction"] != "love" || reactor["id"] != strID(e1UserAlice) {
		t.Fatalf("newest %+v", item)
	}
}

func TestV1ReplyReactionsListPagingParity(t *testing.T) {
	f := newEngageFix(t)
	f.seedListHistory(t)
	spec := "/replies/{reply_id}/reactions"
	base := "/api/v1/replies/" + strID(e1ReplyReact) + "/reactions"
	want := f.sqlIDs(t, `SELECT id FROM topic_reply_reaction WHERE topic_reply_id = ? AND user_id <> ? ORDER BY created DESC, id DESC`,
		e1ReplyReact, e1UserBanned)
	got := f.walkListIDs(t, base, "", spec)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v\nwant %v", got, want)
	}
	resp, body := f.do(t, http.MethodGet, base+"?limit=2", "", spec, nil, nil)
	list := decodeEngageList(t, body)
	if resp.StatusCode != 200 || len(list.Items) != 2 {
		t.Fatalf("fill %d %s", resp.StatusCode, body)
	}
	item := jsonObj(t, list.Items[0])
	if item["object"] != "reaction" || item["reaction"] != "love" {
		t.Fatalf("newest %+v", item)
	}
}

func TestV1EngageListsVisibilityAndErrors(t *testing.T) {
	f := newEngageFix(t)
	f.seedListHistory(t)
	alice := f.asAlice(t)
	bob := f.asBob(t)

	hiddenUp := "/api/v1/topics/" + strID(e1TopicHiddenAuthor) + "/upvotes"
	resp, body := f.do(t, http.MethodGet, hiddenUp, bob, "/topics/{topic_id}/upvotes", nil, nil)
	if resp.StatusCode != 404 {
		t.Fatalf("stranger hidden upvotes %d %s", resp.StatusCode, body)
	}
	resp, body = f.do(t, http.MethodGet, hiddenUp, alice, "/topics/{topic_id}/upvotes", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("author hidden upvotes %d %s", resp.StatusCode, body)
	}

	hiddenReact := "/api/v1/topics/" + strID(e1TopicHiddenAuthor) + "/reactions"
	resp, body = f.do(t, http.MethodGet, hiddenReact, bob, "/topics/{topic_id}/reactions", nil, nil)
	if resp.StatusCode != 404 {
		t.Fatalf("stranger hidden reactions %d %s", resp.StatusCode, body)
	}
	resp, body = f.do(t, http.MethodGet, hiddenReact, alice, "/topics/{topic_id}/reactions", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("author hidden reactions %d %s", resp.StatusCode, body)
	}

	hiddenReply := "/api/v1/replies/" + strID(e1ReplyOnHidden) + "/reactions"
	resp, body = f.do(t, http.MethodGet, hiddenReply, bob, "/replies/{reply_id}/reactions", nil, nil)
	if resp.StatusCode != 404 {
		t.Fatalf("stranger hidden reply reactions %d %s", resp.StatusCode, body)
	}
	resp, body = f.do(t, http.MethodGet, hiddenReply, alice, "/replies/{reply_id}/reactions", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("author hidden reply reactions %d %s", resp.StatusCode, body)
	}

	for _, tc := range []struct {
		url, spec string
	}{
		{"/api/v1/topics/" + strID(e1TopicLists) + "/upvotes?limit=101", "/topics/{topic_id}/upvotes"},
		{"/api/v1/topics/" + strID(e1TopicLists) + "/reactions?limit=101", "/topics/{topic_id}/reactions"},
		{"/api/v1/replies/" + strID(e1ReplyReact) + "/reactions?limit=101", "/replies/{reply_id}/reactions"},
	} {
		resp, body = f.do(t, http.MethodGet, tc.url, "", tc.spec, nil, nil)
		if resp.StatusCode != 400 {
			t.Errorf("%s: %d %s", tc.url, resp.StatusCode, body)
			continue
		}
		code, _, param, _, _ := decodeProblemBody(t, body)
		if code != problem.CodeLimitTooLarge || param != "limit" {
			t.Errorf("%s code %s param %s", tc.url, code, param)
		}
	}

	resp, body = f.do(t, http.MethodGet, "/api/v1/topics/"+strID(e1TopicLists)+"/upvotes?limit=2", "", "/topics/{topic_id}/upvotes", nil, nil)
	up := decodeEngageList(t, body)
	if up.NextCursor == nil {
		t.Fatal("need upvote cursor")
	}
	cur := url.QueryEscape(*up.NextCursor)
	resp, body = f.do(t, http.MethodGet, "/api/v1/topics/"+strID(e1TopicLists)+"/reactions?cursor="+cur, "", "/topics/{topic_id}/reactions", nil, nil)
	code, _, _, _, _ := decodeProblemBody(t, body)
	if resp.StatusCode != 400 || code != problem.CodeInvalidCursor {
		t.Fatalf("cross-list cursor %d %s", resp.StatusCode, body)
	}

	resp, body = f.do(t, http.MethodGet, "/api/v1/topics/"+strID(e1TopicLists)+"/reactions?limit=2", "", "/topics/{topic_id}/reactions", nil, nil)
	tr := decodeEngageList(t, body)
	if tr.NextCursor == nil {
		t.Fatal("need reaction cursor")
	}
	cur = url.QueryEscape(*tr.NextCursor)
	resp, body = f.do(t, http.MethodGet, "/api/v1/topics/"+strID(e1TopicPublic)+"/reactions?cursor="+cur, "", "/topics/{topic_id}/reactions", nil, nil)
	code, _, _, _, _ = decodeProblemBody(t, body)
	if resp.StatusCode != 400 || code != problem.CodeInvalidCursor {
		t.Fatalf("cross-target cursor %d %s", resp.StatusCode, body)
	}
}
