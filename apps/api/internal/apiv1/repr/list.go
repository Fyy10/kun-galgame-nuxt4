package repr

import "encoding/json"

type List[T any] struct {
	Object     string  `json:"object" enum:"list" maxLength:"4" doc:"Type discriminant. Always list."`
	Items      []T     `json:"items" doc:"Members of this page. Empty array, never null."`
	NextCursor *string `json:"next_cursor,omitempty" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512" doc:"Opaque keyset cursor. Omitted on the last page."`
	Total      *int    `json:"total,omitempty" minimum:"0" doc:"Present only when include_total=true. Same visibility gate as items."`
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
		Total      *int    `json:"total,omitempty"`
	}
	return json.Marshal(wire{"list", items, l.NextCursor, l.Total})
}
