package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterWrites(w *Writes) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
			OperationID:   "createTopic",
			Method:        http.MethodPost,
			Path:          "/topics",
			Summary:       "Create a topic",
			DefaultStatus: http.StatusCreated,
			Description: "Creates a topic and returns it as getTopic would to its author. " +
				"A caller may create as many topics in any 24 hours as their cached moemoepoint balance divided by 10, plus one. " +
				"The sections g-seeking, g-other and t-help cost 10 moemoepoint and need that balance; the others earn 3. " +
				"Mentions in the body notify up to 10 users.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "MOEMOEPOINT_INSUFFICIENT when a paid section is chosen without the balance for it; SCOPE_REQUIRED or ACCOUNT_BANNED.",
				422: "VALIDATION_FAILED, or CONTENT_REJECTED when the trust-and-safety check refuses the title or body.",
				429: "TOPIC_DAILY_LIMIT_REACHED; limit is the caller's allowance.",
			}),
		})), w.createTopic)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "updateTopic",
			Method:      http.MethodPatch,
			Path:        "/topics/{topic_id}",
			Summary:     "Update a topic",
			Description: "Changes the fields present in the body and returns the topic as getTopic would. Absent fields keep their value. " +
				"Editing needs can_edit; state needs can_hide or can_unhide. " +
				"A change of the title or body sets edited_at and bumps the topic. " +
				"Moving between paid and free sections charges or refunds the author the difference. " +
				"NOT_FOUND under the same conditions as getTopic.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks the capability a present field needs; SCOPE_REQUIRED or ACCOUNT_BANNED.",
				422: "VALIDATION_FAILED, or CONTENT_REJECTED when the trust-and-safety check refuses the title or body.",
			}),
		}), w.updateTopic)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getTopicSource",
			Method:      http.MethodGet,
			Path:        "/topics/{topic_id}/source",
			Summary:     "Get a topic's editable source",
			Description: "Returns the stored Markdown and every editable field, to fill an edit form. It needs can_edit. " +
				"NOT_FOUND under the same conditions as getTopic.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller may read but not edit the topic; SCOPE_REQUIRED or ACCOUNT_BANNED.",
			}),
		}), w.getTopicSource)

		huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
			OperationID:   "createReply",
			Method:        http.MethodPost,
			Path:          "/topics/{topic_id}/replies",
			Summary:       "Reply to a topic",
			DefaultStatus: http.StatusCreated,
			Description: "Creates a reply at the next floor and returns it as getReply would to its author. " +
				"Floors are never reused, so deleting the last reply leaves a gap. " +
				"The topic author earns 1 moemoepoint from another user's reply and is notified; mentions notify up to 10 users. " +
				"NOT_FOUND under the same conditions as getTopic.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				422: "VALIDATION_FAILED, or CONTENT_REJECTED when the trust-and-safety check refuses the body.",
			}),
		})), w.createReply)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "updateReply",
			Method:      http.MethodPatch,
			Path:        "/replies/{reply_id}",
			Summary:     "Update a reply",
			Description: "Changes the reply body and returns the reply as getReply would. It needs can_edit. " +
				"A changed body sets edited_at; the topic is not bumped. " +
				"NOT_FOUND under the same conditions as getReply.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller may read but not edit the reply; SCOPE_REQUIRED or ACCOUNT_BANNED.",
				422: "VALIDATION_FAILED, or CONTENT_REJECTED when the trust-and-safety check refuses the body.",
			}),
		}), w.updateReply)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID:   "deleteReply",
			Method:        http.MethodDelete,
			Path:          "/replies/{reply_id}",
			Summary:       "Delete a reply",
			DefaultStatus: http.StatusNoContent,
			Description: "Deletes the reply with its comments and reactions. It needs can_delete. Its floor is not reused. " +
				"An author deleting their own reply pays 3 moemoepoint times one plus its likes plus its comments, and needs that balance; " +
				"staff deleting it charges the reply's author 3. " +
				"NOT_FOUND under the same conditions as getReply.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED, or MOEMOEPOINT_INSUFFICIENT when an author cannot pay for deleting their own reply; SCOPE_REQUIRED or ACCOUNT_BANNED.",
			}),
		}), w.deleteReply)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getReplySource",
			Method:      http.MethodGet,
			Path:        "/replies/{reply_id}/source",
			Summary:     "Get a reply's editable source",
			Description: "Returns the stored Markdown of the reply, to fill an edit form. It needs can_edit. " +
				"NOT_FOUND under the same conditions as getReply.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller may read but not edit the reply; SCOPE_REQUIRED or ACCOUNT_BANNED.",
			}),
		}), w.getReplySource)
	}
}
