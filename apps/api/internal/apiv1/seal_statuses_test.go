package apiv1

import (
	"slices"
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

func TestRequiredStatuses(t *testing.T) {
	query := &huma.Param{Name: "q", In: "query"}
	path := &huma.Param{Name: "topic_id", In: "path"}
	idem := &huma.Param{Name: idempotencyHeader, In: "header"}
	body := &huma.RequestBody{}
	for _, c := range []struct {
		name string
		path string
		op   huma.Operation
		want []int
	}{
		{"public, no input", "/x", Public(huma.Operation{}), []int{500}},
		{"a query parameter", "/x", Public(huma.Operation{Parameters: []*huma.Param{query}}), []int{400, 500}},
		{"a path parameter", "/x/{topic_id}", Public(huma.Operation{Parameters: []*huma.Param{path}}), []int{400, 404, 500}},
		{"optional tier", "/x", Optional(huma.Operation{}), []int{401, 403, 500, 503}},
		{"required tier", "/x", Required(huma.Operation{}), []int{401, 403, 500, 503}},
		{"a body", "/x", Public(huma.Operation{RequestBody: body}), []int{400, 415, 422, 500}},
		{"an idempotency key", "/x", Public(huma.Operation{Parameters: []*huma.Param{idem}}), []int{409, 500}},
	} {
		got := RequiredStatuses(c.path, &c.op)
		slices.Sort(got)
		got = slices.Compact(got)
		if !slices.Equal(got, c.want) {
			t.Errorf("%s = %v, want %v", c.name, got, c.want)
		}
	}
}
