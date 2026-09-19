# API v1 changelog

## 2026-09-18

First `/api/v1` surface.

Offered:

- `GET /api/v1/problems` — error-code catalogue
- `GET /api/v1/problems/reasons` — field-level reason catalogue
- `GET /api/v1/topics` — cursor-paged topic list (`listTopics`)

Error codes in the registry (`pkg/problem`, `openapi/problems.json`):

- platform: `MALFORMED_BODY`, `INVALID_PARAMETER`, `UNKNOWN_ENUM_VALUE`, `LIMIT_TOO_LARGE`, `INVALID_CURSOR`, `UNKNOWN_SORT`, `MISSING_CREDENTIAL`, `INVALID_CREDENTIAL`, `SCOPE_REQUIRED`, `NOT_FOUND`, `METHOD_NOT_ALLOWED`, `IDEMPOTENCY_KEY_REUSED`, `UNSUPPORTED_MEDIA_TYPE`, `VALIDATION_FAILED`, `INTERNAL_ERROR`, `SERVICE_UNAVAILABLE`
- kungal: `ACCOUNT_BANNED`, `IDEMPOTENCY_REQUEST_IN_PROGRESS`

## 2026-09-18 (W0b-1)

- The document no longer writes `"additionalProperties": true` on object schemas. It meant the same as leaving the keyword out, and unknown request-body fields are still accepted. Code generators rendered it as a catch-all map (openapi-typescript: `[key: string]: unknown`), which let a misspelled field compile. Regenerate client types.

## 2026-09-18 (W0b-3)

Both changes break clients generated from the earlier document. No deployment served that document.

- `TopicSummary.user` is now `author`. A field that refers to a person is named for the role that person plays in the resource; `user` is a forbidden name (gate G8).
- List bodies no longer declare `total`. None of the three collections accepts `include_total`, so the field could never appear. A collection that offers `include_total` returns a counted list that declares `total`, and gate F9 keeps the parameter and the field together.

## 2026-09-18 (W1 contract)

Additive.

- The document now defines the content node vocabulary: `ContentDocument`, the `BlockNode` and `InlineNode` unions (`oneOf`, discriminated on `object`) and their member schemas. No operation returns a document yet; the root extension `x-content-document` references it so it stays in the published document, and it goes away when the topic detail read returns `content`. Rules for rendering, including unknown node types, are in `docs/proj/api-v1/03-content-doc.md`.

## 2026-09-19 (W2)

Additive, apart from the removed extension.

- `GET /api/v1/topics/{topic_id}` (`getTopic`), `GET /api/v1/topics/{topic_id}/replies` (`listTopicReplies`), `GET /api/v1/replies/{reply_id}` (`getReply`) and `POST /api/v1/topics/{topic_id}/views` (`recordTopicView`).
- Reading a topic does not count a view. A client calls `recordTopicView` once when a reader actually opens the topic, and never when it prefetches or renders elsewhere.
- Replies are in floor order with a cursor; `from_floor` opens the list at a floor, and `sort=floor_desc` with `from_floor` one below reads back from it. The pinned reply and the best answer come with the topic and also appear at their floors.
- The root extension `x-content-document` is gone: `Topic.content` references `ContentDocument`.
