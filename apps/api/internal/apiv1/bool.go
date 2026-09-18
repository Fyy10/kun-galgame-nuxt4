package apiv1

import (
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

func strictBooleans(ctx huma.Context, next func(huma.Context)) {
	op := ctx.Operation()
	if op == nil {
		next(ctx)
		return
	}
	u := ctx.URL()
	q := u.Query()
	for _, p := range op.Parameters {
		if p == nil || p.In != "query" || !schemaIsBoolean(p.Schema) {
			continue
		}
		values, ok := q[p.Name]
		if !ok {
			continue
		}
		for _, raw := range values {
			if raw == "true" || raw == "false" {
				continue
			}
			writeProblem(ctx, problem.New(
				problem.CodeInvalidParameter,
				"Boolean query parameters accept only true or false.",
				problem.AtParameter(p.Name, problem.ReasonInvalidFormat, "expected true or false", nil),
			))
			return
		}
	}
	next(ctx)
}

func schemaIsBoolean(s *huma.Schema) bool {
	if s == nil {
		return false
	}
	return s.Type == huma.TypeBoolean
}
