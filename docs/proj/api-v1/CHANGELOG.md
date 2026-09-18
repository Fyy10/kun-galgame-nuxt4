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
