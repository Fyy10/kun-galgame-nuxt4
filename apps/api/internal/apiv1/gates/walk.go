package gates

import (
	"fmt"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
)

func CheckAll(doc *huma.OpenAPI) []string {
	var errs []string
	errs = append(errs, CheckG2(doc)...)
	errs = append(errs, CheckG3(doc)...)
	errs = append(errs, CheckG4(doc)...)
	errs = append(errs, CheckG5G13(doc)...)
	errs = append(errs, CheckG6(doc)...)
	errs = append(errs, CheckG7(doc)...)
	errs = append(errs, CheckG8(doc)...)
	errs = append(errs, CheckG9(doc)...)
	errs = append(errs, CheckG14(doc)...)
	errs = append(errs, CheckG16(doc)...)
	errs = append(errs, CheckG17(doc)...)
	errs = append(errs, CheckF1(doc)...)
	errs = append(errs, CheckF9(doc)...)
	errs = append(errs, CheckF10(doc)...)
	errs = append(errs, CheckF8()...)
	return errs
}

func pathOps(item *huma.PathItem) []*huma.Operation {
	if item == nil {
		return nil
	}
	ops := make([]*huma.Operation, 0, 8)
	for _, op := range []*huma.Operation{
		item.Get, item.Put, item.Post, item.Delete, item.Options, item.Head, item.Patch, item.Trace,
	} {
		if op != nil {
			ops = append(ops, op)
		}
	}
	return ops
}

func opMethod(op *huma.Operation) string {
	if op != nil && op.Method != "" {
		return op.Method
	}
	return "?"
}

func deref(doc *huma.OpenAPI, s *huma.Schema) *huma.Schema {
	if s == nil || s.Ref == "" || doc == nil || doc.Components == nil || doc.Components.Schemas == nil {
		return s
	}
	if r := doc.Components.Schemas.SchemaFromRef(s.Ref); r != nil {
		return r
	}
	return s
}

func walkDoc(doc *huma.OpenAPI, fn func(where string, s *huma.Schema)) {
	seen := map[*huma.Schema]bool{}
	var walk func(where string, s *huma.Schema)
	walk = func(where string, s *huma.Schema) {
		if s == nil || seen[s] {
			return
		}
		seen[s] = true
		fn(where, s)
		for name, p := range s.Properties {
			walk(where+"."+name, p)
		}
		walk(where+"[]", s.Items)
		for i, x := range s.OneOf {
			walk(fmt.Sprintf("%s.oneOf[%d]", where, i), x)
		}
		for i, x := range s.AnyOf {
			walk(fmt.Sprintf("%s.anyOf[%d]", where, i), x)
		}
		for i, x := range s.AllOf {
			walk(fmt.Sprintf("%s.allOf[%d]", where, i), x)
		}
		walk(where+".not", s.Not)
	}
	if doc.Components != nil && doc.Components.Schemas != nil {
		for name, schema := range doc.Components.Schemas.Map() {
			walk("schema "+name, schema)
		}
	}
	for path, item := range doc.Paths {
		for _, op := range pathOps(item) {
			for _, p := range op.Parameters {
				walk(fmt.Sprintf("%s %s param %s", opMethod(op), path, p.Name), p.Schema)
			}
			if op.RequestBody != nil {
				for mt, c := range op.RequestBody.Content {
					walk(fmt.Sprintf("%s %s request %s", opMethod(op), path, mt), c.Schema)
				}
			}
			for status, resp := range op.Responses {
				for mt, c := range resp.Content {
					walk(fmt.Sprintf("%s %s %s %s", opMethod(op), path, status, mt), c.Schema)
				}
			}
		}
	}
}

func unique(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func isSuccess(status string) bool {
	n, err := strconv.Atoi(status)
	return err == nil && n >= 200 && n < 300
}

func CheckF8() []string {
	return scanLocaleText(apiRoot())
}
