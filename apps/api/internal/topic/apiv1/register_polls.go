package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

const pollVoteConflict = "POLL_CLOSED when the poll is past closes_at, or VOTE_ALREADY_CAST when the caller has voted and the poll does not allow changing a vote."

func RegisterPolls(p *Polls) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "listTopicPolls",
			Method:      http.MethodGet,
			Path:        "/topics/{topic_id}/polls",
			Summary:     "List a topic's polls",
			Description: "Lists every poll of the topic, newest first. It is not paginated: a topic holds at most 30 polls. " +
				"Polls by banned authors are left out. " +
				"NOT_FOUND under the same conditions as getTopic.",
			Tags: []string{"topics"},
		}), p.listTopicPolls)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "getPoll",
			Method:      http.MethodGet,
			Path:        "/polls/{poll_id}",
			Summary:     "Get a poll",
			Description: "Returns one poll. results is null as one block whenever the caller may not see the tallies yet, " +
				"and sample_voters is empty for an anonymous poll even when results are visible. " +
				"NOT_FOUND when the poll does not exist, was written by a banned user, " +
				"or belongs to a topic getTopic would not return to the caller.",
			Tags: []string{"topics"},
		}), p.getPoll)

		huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
			OperationID:   "createPoll",
			Method:        http.MethodPost,
			Path:          "/topics/{topic_id}/polls",
			Summary:       "Create a poll",
			DefaultStatus: http.StatusCreated,
			Description: "Creates a poll on the topic and returns it as getPoll would to its author. " +
				"It needs the topic's author or staff holding the create permission, and the topic must be one the caller can read. " +
				"A single-choice poll stores min_choice and max_choice as 1 whatever the body asks for; " +
				"a multiple-choice poll defaults them to 1 and to the number of options. " +
				"closes_at is stored exactly as sent. Creating a poll bumps the topic. " +
				"NOT_FOUND under the same conditions as getTopic.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller is neither the topic's author nor staff; SCOPE_REQUIRED or ACCOUNT_BANNED.",
				422: "VALIDATION_FAILED when a label is blank, closes_at is not a real instant, the choice bounds contradict each other, " +
					"or the topic already holds 30 polls; CONTENT_REJECTED when the trust-and-safety check refuses the text.",
			}),
		})), p.createPoll)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "updatePoll",
			Method:      http.MethodPatch,
			Path:        "/polls/{poll_id}",
			Summary:     "Update a poll",
			Description: "Changes the fields present in the body and returns the poll as getPoll would. Absent fields keep their value. " +
				"It needs can_edit. option_changes adds, relabels and removes options; an option that already holds votes " +
				"keeps its label and cannot be removed, and two options must remain. " +
				"Once the poll holds a vote, an anonymous poll cannot be made public and the choice type cannot change. " +
				"closes_at set to null clears the deadline; leaving the field out keeps it. " +
				"Only text this request submits is checked by trust and safety again. " +
				"NOT_FOUND under the same conditions as getPoll.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller may read but not edit the poll; SCOPE_REQUIRED or ACCOUNT_BANNED.",
				422: "VALIDATION_FAILED, or CONTENT_REJECTED when the trust-and-safety check refuses the text.",
			}),
		}), p.updatePoll)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID:   "deletePoll",
			Method:        http.MethodDelete,
			Path:          "/polls/{poll_id}",
			Summary:       "Delete a poll",
			DefaultStatus: http.StatusNoContent,
			Description: "Deletes the poll with its options and votes. It needs can_delete. Nothing is refunded and no counter is rolled back: " +
				"the options go with the poll. " +
				"NOT_FOUND under the same conditions as getPoll.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller may read but not delete the poll; SCOPE_REQUIRED or ACCOUNT_BANNED.",
			}),
		}), p.deletePoll)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "setPollVote",
			Method:      http.MethodPut,
			Path:        "/polls/{poll_id}/vote",
			Summary:     "Set the caller's vote",
			Description: "Replaces whatever the caller picked before with option_ids and returns the whole poll. " +
				"It is a slot, so sending the same choice again is a 200 that moves no counter and writes no row. " +
				"Every id must be an option of this poll, the same id twice is refused, and the count must sit between " +
				"min_choice and max_choice. It takes no Idempotency-Key: the operation is idempotent by shape. " +
				"NOT_FOUND under the same conditions as getPoll.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				409: pollVoteConflict,
				422: "VALIDATION_FAILED when an id is not an option of this poll, an id repeats, or the number of ids is outside the poll's bounds.",
			}),
		}), p.setPollVote)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "clearPollVote",
			Method:      http.MethodDelete,
			Path:        "/polls/{poll_id}/vote",
			Summary:     "Retract the caller's vote",
			Description: "Removes the caller's vote and returns the whole poll. Retracting when the caller has not voted is a 200 " +
				"that changes nothing. It needs the poll to allow changing a vote. " +
				"NOT_FOUND under the same conditions as getPoll.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				409: pollVoteConflict,
			}),
		}), p.clearPollVote)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "listPollVotes",
			Method:      http.MethodGet,
			Path:        "/polls/{poll_id}/votes",
			Summary:     "List a poll's votes",
			Description: "Lists who voted for what as a cursor page, newest first. A voter in a multiple-choice poll has one entry per option. " +
				"Votes by banned users are left out; the server reads on to fill the page, so continue while next_cursor is present. " +
				"There is no total. " +
				"NOT_FOUND under the same conditions as getPoll.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER, LIMIT_TOO_LARGE, or INVALID_CURSOR.",
				403: "PERMISSION_REQUIRED when the poll is anonymous or the caller may not see its results yet.",
			}),
		}), p.listPollVotes)
	}
}
