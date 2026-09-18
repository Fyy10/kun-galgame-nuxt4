package apiv1

import (
	"context"
	"net/http"
	"sort"

	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

type listObject[T any] struct {
	Object string `json:"object" enum:"list" maxLength:"4" doc:"Type discriminant. Always list."`
	Items  []T    `json:"items" doc:"The members of this list. Empty array, never null." maxItems:"256"`
}

type problemType struct {
	Object      string `json:"object" enum:"problem_type" maxLength:"12" doc:"Type discriminant. Always problem_type."`
	Code        string `json:"code" pattern:"^[A-Z][A-Z0-9_]*[A-Z0-9]$" maxLength:"63" doc:"Top-level error code. UPPER_SNAKE."`
	Domain      string `json:"domain" enum:"platform,kungal" maxLength:"16" doc:"Type URI domain segment."`
	Status      int    `json:"status" minimum:"400" maximum:"599" doc:"HTTP status this code is bound to. One status per code."`
	Type        string `json:"type" format:"uri" maxLength:"256" doc:"Problem type URI. The last path segment is the kebab-case form of code."`
	Title       string `json:"title" maxLength:"128" pattern:"^[ -~]+$" doc:"Stable English phrase for this type. Does not vary per request."`
	Description string `json:"description" maxLength:"512" doc:"English prose. Must not be used as a discriminant."`
}

type problemReason struct {
	Object      string   `json:"object" enum:"problem_reason" maxLength:"14" doc:"Type discriminant. Always problem_reason."`
	Reason      string   `json:"reason" pattern:"^[A-Z][A-Z0-9_]*[A-Z0-9]$" maxLength:"63" doc:"Field-level reason. UPPER_SNAKE. Disjoint from top-level codes."`
	Title       string   `json:"title" maxLength:"128" pattern:"^[ -~]+$" doc:"Stable English phrase for this reason."`
	Description string   `json:"description" maxLength:"512" doc:"English prose. Must not be used as a discriminant."`
	Params      []string `json:"params" doc:"Declared param key names for this reason. Empty array when the reason carries no params." maxItems:"8"`
}

type listProblemTypesOutput struct {
	Body listObject[problemType]
}

type listProblemReasonsOutput struct {
	Body listObject[problemReason]
}

func registerMeta(api huma.API) {
	tags := []string{"meta"}
	huma.Register(api, Public(huma.Operation{
		OperationID: "listProblemReasons",
		Method:      http.MethodGet,
		Path:        "/problems/reasons",
		Summary:     "List every field-level error reason",
		Description: "The closed registry of field-level reasons. Unauthenticated. Values in this list never appear as top-level codes.",
		Tags:        tags,
	}), listProblemReasons)
	huma.Register(api, Public(huma.Operation{
		OperationID: "listProblemTypes",
		Method:      http.MethodGet,
		Path:        "/problems",
		Summary:     "List every top-level error code",
		Description: "The closed registry of top-level error codes. Unauthenticated. Sorted by domain, then code.",
		Tags:        tags,
	}), listProblemTypes)
}

func listProblemTypes(context.Context, *struct{}) (*listProblemTypesOutput, error) {
	return &listProblemTypesOutput{Body: listObject[problemType]{Object: "list", Items: ProblemTypes()}}, nil
}

func listProblemReasons(context.Context, *struct{}) (*listProblemReasonsOutput, error) {
	return &listProblemReasonsOutput{Body: listObject[problemReason]{Object: "list", Items: ProblemReasons()}}, nil
}

func ProblemTypes() []problemType {
	items := make([]problemType, 0, len(problem.Codes))
	for _, d := range problem.Codes {
		items = append(items, problemType{
			Object:      "problem_type",
			Code:        d.Code,
			Domain:      string(d.Domain),
			Status:      d.Status,
			Type:        d.TypeURI(),
			Title:       d.Title,
			Description: d.Description,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		ri, rj := domainRank(items[i].Domain), domainRank(items[j].Domain)
		if ri != rj {
			return ri < rj
		}
		return items[i].Code < items[j].Code
	})
	return items
}

func ProblemReasons() []problemReason {
	items := make([]problemReason, 0, len(problem.Reasons))
	for _, d := range problem.Reasons {
		params := d.Params
		if params == nil {
			params = []string{}
		}
		items = append(items, problemReason{
			Object:      "problem_reason",
			Reason:      d.Reason,
			Title:       d.Title,
			Description: d.Description,
			Params:      params,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Reason < items[j].Reason
	})
	return items
}

func domainRank(domain string) int {
	for i, d := range problem.DomainOrder {
		if string(d) == domain {
			return i
		}
	}
	return len(problem.DomainOrder)
}
