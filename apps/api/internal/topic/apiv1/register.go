package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

func Register(svc *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "listTopics",
			Method:      http.MethodGet,
			Path:        "/topics",
			Summary:     "List topics",
			Description: "Lists topics visible to the caller as a cursor page. " +
				"Anonymous callers see public published topics. Signed-in callers also see login-scoped topics. " +
				"Role and user grants never appear. Hidden topics never appear. " +
				"Topics by banned authors are left out; the server reads on to fill the page, so every page but the last holds limit items " +
				"unless a long run of such topics cuts it short. Continue while next_cursor is present, whatever the page size.",
			Tags: []string{"topics"},
			Responses: map[string]*huma.Response{
				"400": {
					Description: "INVALID_PARAMETER, LIMIT_TOO_LARGE, UNKNOWN_SORT, UNKNOWN_ENUM_VALUE, or INVALID_CURSOR.",
					Content: map[string]*huma.MediaType{
						problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}},
					},
				},
			},
		}), svc.listTopics)
	}
}
