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

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "getTopic",
			Method:      http.MethodGet,
			Path:        "/topics/{topic_id}",
			Summary:     "Get a topic",
			Description: "Returns one topic with its body. " +
				"NOT_FOUND when the topic does not exist, is hidden and the caller is neither its author nor staff, " +
				"is scoped to readers the caller is not among, or was written by a banned user; the four are indistinguishable. " +
				"Reading does not count a view; see recordTopicView.",
			Tags: []string{"topics"},
		}), svc.getTopic)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "listTopicReplies",
			Method:      http.MethodGet,
			Path:        "/topics/{topic_id}/replies",
			Summary:     "List a topic's replies",
			Description: "Lists the replies of a topic as a cursor page in floor order. " +
				"Hidden replies and replies by banned users are left out; the server reads on to fill the page, " +
				"so continue while next_cursor is present, whatever the page size. " +
				"The pinned reply and the best answer appear at their floors like any other reply. " +
				"To open at a floor, pass from_floor; to read back from it, pass sort=floor_desc with from_floor one below. " +
				"NOT_FOUND under the same conditions as getTopic.",
			Tags: []string{"topics"},
			Responses: map[string]*huma.Response{
				"400": {
					Description: "INVALID_PARAMETER, LIMIT_TOO_LARGE, UNKNOWN_SORT, or INVALID_CURSOR.",
					Content: map[string]*huma.MediaType{
						problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}},
					},
				},
			},
		}), svc.listTopicReplies)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "getReply",
			Method:      http.MethodGet,
			Path:        "/replies/{reply_id}",
			Summary:     "Get a reply",
			Description: "Returns one reply with its comments. " +
				"NOT_FOUND when the reply does not exist, is hidden, was written by a banned user, " +
				"or belongs to a topic getTopic would not return to the caller.",
			Tags: []string{"topics"},
		}), svc.getReply)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "recordTopicView",
			Method:      http.MethodPost,
			Path:        "/topics/{topic_id}/views",
			Summary:     "Record a topic view",
			Description: "Counts one view of the topic. A client calls it once when a reader actually opens the topic, " +
				"not when it prefetches, renders on a server or builds a share card. Anonymous callers may call it. " +
				"It takes no Idempotency-Key: a retried view counts twice, as a reload does. " +
				"NOT_FOUND under the same conditions as getTopic.",
			Tags:          []string{"topics"},
			DefaultStatus: http.StatusNoContent,
		}), svc.recordTopicView)
	}
}
