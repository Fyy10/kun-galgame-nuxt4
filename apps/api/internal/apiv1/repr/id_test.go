package repr

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

func TestID(t *testing.T) {
	if got := ID(4121); got != "4121" {
		t.Errorf("ID(4121) = %q", got)
	}
	n, ok := ParseID("4121")
	if !ok || n != 4121 {
		t.Errorf("ParseID = %d %v", n, ok)
	}
	for _, bad := range []DecimalID{"abc", "0", "-3", ""} {
		if _, ok := ParseID(bad); ok {
			t.Errorf("ParseID(%q) succeeded", bad)
		}
	}
}

func TestTimestampAndDate(t *testing.T) {
	ts := time.Date(2026, 9, 18, 10, 11, 12, 345e6, time.FixedZone("CST", 8*3600))
	if got := Timestamp(ts); got != "2026-09-18T02:11:12Z" {
		t.Errorf("Timestamp = %q", got)
	}
	if got := Date(ts); got != "2026-09-18" {
		t.Errorf("Date = %q", got)
	}
	if TimestampPtr(nil) != nil {
		t.Error("TimestampPtr(nil) not nil")
	}
	got := TimestampPtr(&ts)
	if got == nil || *got != "2026-09-18T02:11:12Z" {
		t.Errorf("TimestampPtr = %v", got)
	}
}

func TestDecimalIDFieldDocBecomesDescription(t *testing.T) {
	type sample struct {
		TopicID DecimalID `json:"topic_id" doc:"The topic being addressed."`
	}
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	s := huma.SchemaFromType(reg, reflect.TypeFor[sample]())
	prop := s.Properties["topic_id"]
	if prop == nil {
		t.Fatal("missing topic_id")
	}
	if prop.Description != "The topic being addressed." {
		t.Errorf("description %q", prop.Description)
	}
	if prop.Type != huma.TypeString || prop.Pattern != idPattern {
		t.Errorf("schema type=%q pattern=%q", prop.Type, prop.Pattern)
	}
	if prop.MinLength == nil || *prop.MinLength != 1 || prop.MaxLength == nil || *prop.MaxLength != 20 {
		t.Errorf("length bounds min=%v max=%v", prop.MinLength, prop.MaxLength)
	}
}

func TestListOmitsNextCursorOnLastPage(t *testing.T) {
	out := NewList([]int{1, 2}, nil)
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["next_cursor"]; ok {
		t.Fatalf("next_cursor present: %s", raw)
	}
	if _, ok := m["total"]; ok {
		t.Fatalf("total present: %s", raw)
	}
	items, _ := m["items"].([]any)
	if items == nil {
		t.Fatalf("items null: %s", raw)
	}
	if raw, _ := json.Marshal(List[int]{}); string(raw) != `{"object":"list","items":[]}` {
		t.Errorf("zero List = %s", raw)
	}
	empty := NewList[int](nil, nil)
	raw, _ = json.Marshal(empty)
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	items, ok := m["items"].([]any)
	if !ok || items == nil {
		t.Fatalf("nil items marshalled null: %s", raw)
	}
	cur := "cur_abc"
	n := 3
	page := NewList([]int{1}, &cur)
	page.Total = &n
	raw, _ = json.Marshal(page)
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["next_cursor"] != cur {
		t.Errorf("next_cursor %v", m["next_cursor"])
	}
	if m["total"] != float64(3) {
		t.Errorf("total %v", m["total"])
	}
}

type topicSummarySample struct {
	Object string `json:"object" enum:"topic" doc:"Type discriminant. Always topic."`
	ID     string `json:"id" pattern:"^[0-9]+$" minLength:"1" maxLength:"20" doc:"Topic id."`
}

func TestListSchemaNameIsReadable(t *testing.T) {
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	reg.Schema(reflect.TypeFor[List[topicSummarySample]](), true, "")
	if _, ok := reg.Map()["ListTopicSummarySample"]; !ok {
		t.Fatalf("schemas %v", keys(reg.Map()))
	}
}

func keys(m map[string]*huma.Schema) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
