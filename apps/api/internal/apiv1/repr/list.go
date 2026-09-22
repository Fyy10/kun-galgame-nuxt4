package repr

import "encoding/json"

type List[T any] struct {
	Object     string  `json:"object" enum:"list" maxLength:"4" doc:"Type discriminant. Always list."`
	Items      []T     `json:"items" doc:"Members of this page. Empty array, never null."`
	NextCursor *string `json:"next_cursor,omitempty" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512" doc:"Opaque keyset cursor. Omitted on the last page."`
}

func NewList[T any](items []T, nextCursor *string) List[T] {
	if items == nil {
		items = []T{}
	}
	return List[T]{Object: "list", Items: items, NextCursor: nextCursor}
}

func (l List[T]) MarshalJSON() ([]byte, error) {
	items := l.Items
	if items == nil {
		items = []T{}
	}
	type wire struct {
		Object     string  `json:"object"`
		Items      []T     `json:"items"`
		NextCursor *string `json:"next_cursor,omitempty"`
	}
	return json.Marshal(wire{"list", items, l.NextCursor})
}

// CountedList is the body of a collection that embeds collect.Total; gate F9
// holds the two together.
type CountedList[T any] struct {
	Object     string  `json:"object" enum:"list" maxLength:"4" doc:"Type discriminant. Always list."`
	Items      []T     `json:"items" doc:"Members of this page. Empty array, never null."`
	NextCursor *string `json:"next_cursor,omitempty" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512" doc:"Opaque keyset cursor. Omitted on the last page."`
	Total      *int    `json:"total,omitempty" minimum:"0" doc:"Present only when include_total=true. Same visibility gate as items."`
}

func NewCountedList[T any](items []T, nextCursor *string, total *int) CountedList[T] {
	l := NewList(items, nextCursor)
	return CountedList[T]{Object: l.Object, Items: l.Items, NextCursor: l.NextCursor, Total: total}
}

func (l CountedList[T]) MarshalJSON() ([]byte, error) {
	items := l.Items
	if items == nil {
		items = []T{}
	}
	type wire struct {
		Object     string  `json:"object"`
		Items      []T     `json:"items"`
		NextCursor *string `json:"next_cursor,omitempty"`
		Total      *int    `json:"total,omitempty"`
	}
	return json.Marshal(wire{"list", items, l.NextCursor, l.Total})
}

// BatchList is the body of an ids= batch read (infra api-v2 05 §4): it is not
// paginated, and every requested id that did not come back sits in missing.
// The three reasons an id can be missing — it does not exist, the caller may
// not see it, a filter dropped it — are deliberately not distinguished, so the
// face cannot be used as an existence oracle. Silent non-matching is what this
// replaces: without missing, a caller cannot tell "gone" from "filtered".
type BatchList[T any] struct {
	Object  string      `json:"object" enum:"list" maxLength:"4" doc:"Type discriminant. Always list."`
	Items   []T         `json:"items" doc:"One member per requested id that the caller may see. Empty array, never null."`
	Missing []DecimalID `json:"missing" doc:"Requested ids that did not come back, in the order they were requested. Empty array, never null. The reason is deliberately not given."`
}

func NewBatchList[T any](items []T, missing []DecimalID) BatchList[T] {
	if items == nil {
		items = []T{}
	}
	if missing == nil {
		missing = []DecimalID{}
	}
	return BatchList[T]{Object: "list", Items: items, Missing: missing}
}

func (l BatchList[T]) MarshalJSON() ([]byte, error) {
	items := l.Items
	if items == nil {
		items = []T{}
	}
	missing := l.Missing
	if missing == nil {
		missing = []DecimalID{}
	}
	type wire struct {
		Object  string      `json:"object"`
		Items   []T         `json:"items"`
		Missing []DecimalID `json:"missing"`
	}
	return json.Marshal(wire{"list", items, missing})
}
