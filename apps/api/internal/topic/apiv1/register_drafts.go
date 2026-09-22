package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

const draftPrivacy = "A draft is private to its author. Another author's draft is NOT_FOUND, never 403: a stranger must not learn that the id exists."

func RegisterDrafts(d *Drafts) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "listTopicDrafts",
			Method:      http.MethodGet,
			Path:        "/me/topic-drafts",
			Summary:     "List the caller's topic drafts",
			Description: "Lists the caller's own topic drafts, most recently written first, ties broken by descending id. " +
				"There is one sort and no sort parameter. " + draftPrivacy,
			Tags: []string{"topics"},
		}), d.listTopicDrafts)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getTopicDraft",
			Method:      http.MethodGet,
			Path:        "/me/topic-drafts/{draft_id}",
			Summary:     "Get a topic draft",
			Description: "Returns one of the caller's own drafts, with the body as stored. " + draftPrivacy,
			Tags:        []string{"topics"},
			Responses: problemResponses(map[int]string{
				404: "NOT_FOUND when the draft does not exist or belongs to another author.",
			}),
		}), d.getTopicDraft)

		huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
			OperationID:   "createTopicDraft",
			Method:        http.MethodPost,
			Path:          "/me/topic-drafts",
			Summary:       "Save a topic draft",
			DefaultStatus: http.StatusCreated,
			Description: "Saves the editor's current buffer as a new draft and returns it. " +
				"Every field is optional because a draft is unfinished by definition; sections are not checked against category. " +
				"**This always creates.** There is no update face and this one never overwrites: a draft is a snapshot, " +
				"the client tracks no draft id, and loading one into the editor and saving again is meant to leave two. " +
				"Deleting is the only way to make room once the cap is reached.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				409: "DRAFT_LIMIT_REACHED when the caller already holds 30 drafts.",
				422: "VALIDATION_FAILED when the title and the body are both blank after trimming whitespace.",
			}),
		})), d.createTopicDraft)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID:   "deleteTopicDraft",
			Method:        http.MethodDelete,
			Path:          "/me/topic-drafts/{draft_id}",
			Summary:       "Delete a topic draft",
			DefaultStatus: http.StatusNoContent,
			Description:   "Deletes one of the caller's own drafts. Deleting it again is NOT_FOUND. " + draftPrivacy,
			Tags:          []string{"topics"},
			Responses: problemResponses(map[int]string{
				404: "NOT_FOUND when the draft does not exist or belongs to another author.",
			}),
		}), d.deleteTopicDraft)
	}
}
