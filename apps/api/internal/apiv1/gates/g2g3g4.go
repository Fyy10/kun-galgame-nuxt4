package gates

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

func CheckG2(doc *huma.OpenAPI) []string {
	var errs []string
	if doc == nil {
		return []string{"G2: document is nil"}
	}
	for path, item := range doc.Paths {
		for _, op := range pathOps(item) {
			if strings.TrimSpace(op.Description) == "" && strings.TrimSpace(op.Summary) == "" {
				errs = append(errs, fmt.Sprintf("G2: %s %s has no description or summary", opMethod(op), path))
			}
			for _, p := range op.Parameters {
				if p == nil {
					continue
				}
				if strings.TrimSpace(p.Description) == "" {
					errs = append(errs, fmt.Sprintf("G2: %s %s parameter %s has no description", opMethod(op), path, p.Name))
				}
				errs = append(errs, schemaG2(fmt.Sprintf("%s %s param %s", opMethod(op), path, p.Name), p.Schema)...)
			}
			for status, resp := range op.Responses {
				if resp == nil {
					continue
				}
				if strings.TrimSpace(resp.Description) == "" {
					errs = append(errs, fmt.Sprintf("G2: %s %s response %s has no description", opMethod(op), path, status))
				}
				for mt, content := range resp.Content {
					if content == nil {
						continue
					}
					errs = append(errs, schemaG2(fmt.Sprintf("%s %s %s %s", opMethod(op), path, status, mt), content.Schema)...)
				}
			}
			if op.RequestBody != nil {
				for mt, content := range op.RequestBody.Content {
					if content != nil {
						errs = append(errs, schemaG2(fmt.Sprintf("%s %s request %s", opMethod(op), path, mt), content.Schema)...)
					}
				}
			}
		}
	}
	if doc.Components != nil && doc.Components.Schemas != nil {
		for name, schema := range doc.Components.Schemas.Map() {
			errs = append(errs, schemaG2("schema "+name, schema)...)
		}
	}
	return errs
}

func schemaG2(where string, s *huma.Schema) []string {
	if s == nil {
		return nil
	}
	if s.Ref != "" {
		return nil
	}
	var errs []string
	if len(s.Enum) > 0 && strings.TrimSpace(s.Description) == "" {
		errs = append(errs, fmt.Sprintf("G2: %s enum has no description", where))
	}
	for name, prop := range s.Properties {
		if prop == nil {
			continue
		}
		if prop.Ref == "" && strings.TrimSpace(prop.Description) == "" {
			errs = append(errs, fmt.Sprintf("G2: %s property %s has no description", where, name))
		}
		errs = append(errs, schemaG2(where+"."+name, prop)...)
	}
	if s.Items != nil {
		errs = append(errs, schemaG2(where+"[]", s.Items)...)
	}
	for i, x := range s.OneOf {
		errs = append(errs, schemaG2(fmt.Sprintf("%s.oneOf[%d]", where, i), x)...)
	}
	for i, x := range s.AnyOf {
		errs = append(errs, schemaG2(fmt.Sprintf("%s.anyOf[%d]", where, i), x)...)
	}
	return errs
}

func CheckG3(doc *huma.OpenAPI) []string {
	var errs []string
	walkDoc(doc, func(where string, s *huma.Schema) {
		if len(s.Enum) == 0 {
			return
		}
		closed, ok := s.Extensions["x-vocabulary-closed"].(bool)
		switch {
		case !ok:
			errs = append(errs, "G3: "+where+" enum has no x-vocabulary-closed")
		case !closed && s.Extensions["x-vocabulary"] == nil:
			errs = append(errs, "G3: "+where+" open enum has no x-vocabulary")
		}
	})
	return errs
}

var codeToken = regexp.MustCompile(`\b[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+\b`)

func CheckG4(doc *huma.OpenAPI) []string {
	var errs []string
	for path, item := range doc.Paths {
		for _, op := range pathOps(item) {
			derived := map[string]bool{}
			for _, code := range apiv1.RequiredStatuses(path, op) {
				derived[strconv.Itoa(code)] = true
				if op.Responses[strconv.Itoa(code)] == nil {
					errs = append(errs, fmt.Sprintf("G4: %s %s does not declare %d", opMethod(op), path, code))
				}
			}
			for status, resp := range op.Responses {
				if isSuccess(status) || status == "304" {
					continue
				}
				where := fmt.Sprintf("G4: %s %s response %s", opMethod(op), path, status)
				if status == "default" {
					errs = append(errs, where+" is a catch-all; declare each error status")
					continue
				}
				c := resp.Content[problem.ContentType]
				if c == nil || c.Schema == nil || c.Schema.Ref != apiv1.ProblemRef {
					errs = append(errs, where+" is not application/problem+json with $ref Problem")
				}
				named := false
				for _, tok := range codeToken.FindAllString(resp.Description, -1) {
					def, ok := problem.Lookup(tok)
					switch {
					case !ok:
						errs = append(errs, fmt.Sprintf("%s names %s, which is not in the code registry", where, tok))
					case strconv.Itoa(def.Status) != status:
						errs = append(errs, fmt.Sprintf("%s names %s, whose status is %d", where, tok, def.Status))
					default:
						named = true
					}
				}
				if !derived[status] && !named {
					errs = append(errs, where+" is not derived from the operation and names no registry code with that status")
				}
			}
		}
	}
	return unique(errs)
}
