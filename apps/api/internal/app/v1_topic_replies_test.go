package app

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func TestV1ListTopicRepliesTraversalParity(t *testing.T) {
	f := newDetailFix(t)
	for _, tt := range []struct {
		sort      string
		fromFloor int
	}{
		{"floor_asc", 0},
		{"floor_desc", 0},
		{"floor_asc", 3},
		{"floor_desc", 8},
	} {
		t.Run(fmt.Sprintf("%s/from=%d", tt.sort, tt.fromFloor), func(t *testing.T) {
			want := f.sqlReplyOrder(t, d1TopicList, tt.sort, tt.fromFloor)
			got := f.walkReplyIDs(t, d1TopicList, tt.sort, tt.fromFloor, 2, "")
			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Errorf("got %v\nwant %v", got, want)
			}
			seen := map[string]bool{}
			for _, id := range got {
				if seen[id] {
					t.Errorf("duplicate %s", id)
				}
				seen[id] = true
			}
		})
	}
}

func TestV1ListTopicRepliesErrors(t *testing.T) {
	f := newDetailFix(t)
	path := func(q string) string {
		return "/api/v1/topics/" + strconv.Itoa(d1TopicList) + "/replies" + q
	}
	spec := "/topics/{topic_id}/replies"
	resp, body := f.do(t, http.MethodGet, path("?limit=101"), "", spec, nil)
	if resp.StatusCode != 400 {
		t.Fatalf("limit 101: %d %s", resp.StatusCode, body)
	}
	code, _, param, _, _ := decodeProblemBody(t, body)
	if code != problem.CodeLimitTooLarge || param != "limit" {
		t.Errorf("limit 101 %s %s", code, param)
	}
	resp, body = f.do(t, http.MethodGet, path("?sort=nope"), "", spec, nil)
	if resp.StatusCode != 400 {
		t.Fatalf("sort nope: %d %s", resp.StatusCode, body)
	}
	code, _, _, _, _ = decodeProblemBody(t, body)
	if code != problem.CodeUnknownSort {
		t.Errorf("sort nope code %s", code)
	}
	resp, body = f.do(t, http.MethodGet, "/api/v1/topics/920000299/replies", "", spec, nil)
	if resp.StatusCode != 404 {
		t.Errorf("unknown topic: %d %s", resp.StatusCode, body)
	}
	resp, body = f.do(t, http.MethodGet, "/api/v1/topics/"+strconv.Itoa(d1TopicBanned)+"/replies", "", spec, nil)
	if resp.StatusCode != 404 {
		t.Errorf("banned author topic: %d %s", resp.StatusCode, body)
	}

	resp, body = f.do(t, http.MethodGet, path("?sort=floor_asc&limit=2"), "", spec, nil)
	list := decodeReplyList(t, body)
	if list.NextCursor == nil {
		t.Fatal("need cursor")
	}
	cur := url.QueryEscape(*list.NextCursor)
	resp, body = f.do(t, http.MethodGet, path("?sort=floor_desc&cursor="+cur), "", spec, nil)
	if resp.StatusCode != 400 {
		t.Errorf("sort reuse: %d %s", resp.StatusCode, body)
	}
	code, _, _, _, _ = decodeProblemBody(t, body)
	if code != problem.CodeInvalidCursor {
		t.Errorf("sort reuse code %s", code)
	}
	resp, body = f.do(t, http.MethodGet, "/api/v1/topics/"+strconv.Itoa(d1TopicPublic)+"/replies?cursor="+cur, "", spec, nil)
	code, _, _, _, _ = decodeProblemBody(t, body)
	if resp.StatusCode != 400 || code != problem.CodeInvalidCursor {
		t.Errorf("topic reuse: %d %s", resp.StatusCode, body)
	}
	resp, body = f.do(t, http.MethodGet, path("?from_floor=3&cursor="+cur), "", spec, nil)
	code, _, _, _, _ = decodeProblemBody(t, body)
	if resp.StatusCode != 400 || code != problem.CodeInvalidCursor {
		t.Errorf("from_floor reuse: %d %s", resp.StatusCode, body)
	}
}

func TestV1ListTopicRepliesFillsAfterBanned(t *testing.T) {
	f := newDetailFix(t)
	resp, body := f.do(t, http.MethodGet, "/api/v1/topics/"+strconv.Itoa(d1TopicFill)+"/replies?sort=floor_asc&limit=1", "", "/topics/{topic_id}/replies", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	list := decodeReplyList(t, body)
	if len(list.Items) != 0 || list.NextCursor == nil {
		t.Fatalf("six banned floors should leave a short page with a cursor: %s", body)
	}
	resp, body = f.do(t, http.MethodGet, "/api/v1/topics/"+strconv.Itoa(d1TopicFill)+"/replies?sort=floor_asc&limit=1&cursor="+url.QueryEscape(*list.NextCursor), "", "/topics/{topic_id}/replies", nil)
	list = decodeReplyList(t, body)
	if len(list.Items) != 1 || list.Items[0].ID != "920000386" {
		t.Fatalf("page after banned run = %s", body)
	}
}

func TestV1ListTopicRepliesStopsAtTheWindowThatFillsThePage(t *testing.T) {
	f := newDetailFix(t)
	var windows atomic.Int32
	name := "count-reply-windows-" + t.Name()
	if err := f.db.Callback().Query().After("gorm:query").Register(name, func(tx *gorm.DB) {
		sql := tx.Statement.SQL.String()
		if strings.Contains(sql, "topic_reply") && strings.Contains(sql, "ORDER BY floor") {
			windows.Add(1)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.db.Callback().Query().Remove(name) })

	resp, body := f.do(t, http.MethodGet, "/api/v1/topics/"+strconv.Itoa(d1TopicList)+"/replies?limit=2", "", "/topics/{topic_id}/replies", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	if list := decodeReplyList(t, body); len(list.Items) != 2 || list.NextCursor == nil {
		t.Fatalf("first page = %s", body)
	}
	if n := windows.Load(); n != 1 {
		t.Errorf("read %d reply windows for a page the first window filled, want 1", n)
	}
}

func TestV1GetReply(t *testing.T) {
	f := newDetailFix(t)
	spec := "/replies/{reply_id}"
	resp, body := f.do(t, http.MethodGet, "/api/v1/replies/"+strconv.Itoa(d1ReplyPinned), "", spec, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	if keys := objectKeys(t, body); !equalStrings(keys, replyKeys) {
		t.Errorf("Reply keys %v", keys)
	}
	var reply map[string]json.RawMessage
	_ = json.Unmarshal(body, &reply)
	if string(reply["is_pinned"]) != `true` || string(reply["topic_id"]) != `"920000201"` || string(reply["floor"]) != `2` {
		t.Errorf("pinned reply %s", body)
	}
	assertPinnedComments(t, reply["comments"])

	f.putSession(t, "sess-alice", d1UserAlice)
	resp, body = f.do(t, http.MethodGet, "/api/v1/replies/"+strconv.Itoa(d1ReplyPinned), "sess-alice", spec, nil)
	var signed struct {
		Viewer *struct {
			HasLiked    bool `json:"has_liked"`
			HasDisliked bool `json:"has_disliked"`
		} `json:"viewer"`
		Comments []struct {
			ID     string `json:"id"`
			Viewer *struct {
				HasLiked bool `json:"has_liked"`
			} `json:"viewer"`
		} `json:"comments"`
		Reactions []struct {
			Reaction string `json:"reaction"`
			Viewer   *struct {
				HasReacted bool `json:"has_reacted"`
			} `json:"viewer"`
		} `json:"reactions"`
	}
	_ = json.Unmarshal(body, &signed)
	if signed.Viewer == nil || !signed.Viewer.HasLiked || signed.Viewer.HasDisliked {
		t.Errorf("reply viewer %+v", signed.Viewer)
	}
	foundParentLike := false
	for _, c := range signed.Comments {
		if c.ID == "920000401" && c.Viewer != nil && c.Viewer.HasLiked {
			foundParentLike = true
		}
	}
	if !foundParentLike {
		t.Errorf("comment like viewer %+v", signed.Comments)
	}
	if len(signed.Reactions) == 0 || signed.Reactions[0].Viewer == nil || !signed.Reactions[0].Viewer.HasReacted {
		t.Errorf("reply reactions %+v", signed.Reactions)
	}

	resp, body = f.do(t, http.MethodGet, "/api/v1/replies/"+strconv.Itoa(d1ReplyHidden), "", spec, nil)
	if resp.StatusCode != 404 {
		t.Errorf("hidden reply: %d %s", resp.StatusCode, body)
	}
	resp, body = f.do(t, http.MethodGet, "/api/v1/replies/"+strconv.Itoa(d1ReplyBanGet), "", spec, nil)
	if resp.StatusCode != 404 {
		t.Errorf("banned author: %d %s", resp.StatusCode, body)
	}
	resp, body = f.do(t, http.MethodGet, "/api/v1/replies/"+strconv.Itoa(d1ReplyOnHid), "", spec, nil)
	if resp.StatusCode != 404 {
		t.Errorf("reply of hidden topic: %d %s", resp.StatusCode, body)
	}
}

func TestV1RecordTopicView(t *testing.T) {
	f := newDetailFix(t)
	viewBefore := f.topicView(t, d1TopicPublic)
	dayBefore := f.dailyView(t, d1TopicPublic)

	resp, body := f.do(t, http.MethodGet, "/api/v1/topics/"+strconv.Itoa(d1TopicPublic), "", "/topics/{topic_id}", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("get: %d %s", resp.StatusCode, body)
	}
	if f.topicView(t, d1TopicPublic) != viewBefore {
		t.Fatal("getTopic incremented topic.view")
	}
	if f.dailyView(t, d1TopicPublic) != dayBefore {
		t.Fatal("getTopic incremented topic_view_daily")
	}

	resp, body = f.do(t, http.MethodPost, "/api/v1/topics/"+strconv.Itoa(d1TopicPublic)+"/views", "", "/topics/{topic_id}/views", nil)
	if resp.StatusCode != 204 {
		t.Fatalf("views: %d %s", resp.StatusCode, body)
	}
	if f.topicView(t, d1TopicPublic) != viewBefore+1 {
		t.Errorf("view %d, want %d", f.topicView(t, d1TopicPublic), viewBefore+1)
	}
	if f.dailyView(t, d1TopicPublic) != dayBefore+1 {
		t.Errorf("daily %d, want %d", f.dailyView(t, d1TopicPublic), dayBefore+1)
	}

	loginBefore := f.topicView(t, d1TopicLogin)
	loginDay := f.dailyView(t, d1TopicLogin)
	resp, body = f.do(t, http.MethodPost, "/api/v1/topics/"+strconv.Itoa(d1TopicLogin)+"/views", "", "/topics/{topic_id}/views", nil)
	if resp.StatusCode != 404 {
		t.Errorf("anon login views: %d %s", resp.StatusCode, body)
	}
	if f.topicView(t, d1TopicLogin) != loginBefore || f.dailyView(t, d1TopicLogin) != loginDay {
		t.Fatal("404 view beacon incremented")
	}

	banBefore := f.topicView(t, d1TopicBanned)
	banDay := f.dailyView(t, d1TopicBanned)
	resp, body = f.do(t, http.MethodPost, "/api/v1/topics/"+strconv.Itoa(d1TopicBanned)+"/views", "", "/topics/{topic_id}/views", nil)
	if resp.StatusCode != 404 {
		t.Errorf("banned author views: %d %s", resp.StatusCode, body)
	}
	if f.topicView(t, d1TopicBanned) != banBefore || f.dailyView(t, d1TopicBanned) != banDay {
		t.Fatal("banned-author view beacon incremented")
	}
}

func TestV1ListTopicRepliesUserServiceBound(t *testing.T) {
	f := newDetailFix(t)
	f.UserClient.Invalidate(d1UserAlice, d1UserBanned, d1UserBob, d1UserCarol, d1UserStaff, d1UserGone)
	f.batchCalls.Store(0)
	resp, body := f.do(t, http.MethodGet, "/api/v1/topics/"+strconv.Itoa(d1TopicPublic)+"/replies?limit=20", "", "/topics/{topic_id}/replies", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	if n := f.batchCalls.Load(); n > 2 {
		t.Errorf("reply page user-service calls %d, want <= 2", n)
	}
}

type replyListBody struct {
	Items []struct {
		ID string `json:"id"`
	} `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

func decodeReplyList(t *testing.T, body []byte) replyListBody {
	t.Helper()
	var out replyListBody
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("list: %v\n%s", err, body)
	}
	return out
}

func (f *detailFix) walkReplyIDs(t *testing.T, topicID int, sort string, fromFloor, limit int, session string) []string {
	t.Helper()
	q := fmt.Sprintf("/api/v1/topics/%d/replies?sort=%s&limit=%d", topicID, sort, limit)
	if fromFloor > 0 {
		q += "&from_floor=" + strconv.Itoa(fromFloor)
	}
	var got []string
	for page := 0; page < 50; page++ {
		resp, body := f.do(t, http.MethodGet, q, session, "/topics/{topic_id}/replies", nil)
		if resp.StatusCode != 200 {
			t.Fatalf("%s: %d %s", q, resp.StatusCode, body)
		}
		list := decodeReplyList(t, body)
		for _, it := range list.Items {
			got = append(got, it.ID)
		}
		if list.NextCursor == nil {
			return got
		}
		q = fmt.Sprintf("/api/v1/topics/%d/replies?sort=%s&limit=%d&cursor=%s",
			topicID, sort, limit, url.QueryEscape(*list.NextCursor))
		if fromFloor > 0 {
			q += "&from_floor=" + strconv.Itoa(fromFloor)
		}
	}
	t.Fatal("too many pages")
	return got
}

func (f *detailFix) sqlReplyOrder(t *testing.T, topicID int, sort string, fromFloor int) []string {
	t.Helper()
	dir := "ASC"
	if sort == "floor_desc" {
		dir = "DESC"
	}
	pred := "topic_id = ? AND status = 0"
	args := []any{topicID}
	if fromFloor > 0 {
		if sort == "floor_desc" {
			pred += " AND floor <= ?"
		} else {
			pred += " AND floor >= ?"
		}
		args = append(args, fromFloor)
	}
	type row struct {
		ID     int
		UserID int
	}
	var rows []row
	q := fmt.Sprintf("SELECT id, user_id FROM topic_reply WHERE %s ORDER BY floor %s, id %s", pred, dir, dir)
	if err := f.db.Raw(q, args...).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.UserID == d1UserBanned {
			continue
		}
		out = append(out, strconv.Itoa(r.ID))
	}
	return out
}

func (f *detailFix) topicView(t *testing.T, id int) int {
	t.Helper()
	var n int
	if err := f.db.Raw("SELECT view FROM topic WHERE id = ?", id).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *detailFix) dailyView(t *testing.T, id int) int {
	t.Helper()
	var n int
	if err := f.db.Raw("SELECT COALESCE(SUM(count), 0) FROM topic_view_daily WHERE entity_id = ? AND day = CURRENT_DATE", id).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}
