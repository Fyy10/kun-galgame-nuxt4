package problem

import (
	"net/http"
	"regexp"
	"strings"
)

const TypeURIPrefix = "https://developer.nextmoe.dev/problems/"

type Domain string

const (
	DomainPlatform Domain = "platform"
	DomainKungal   Domain = "kungal"
)

var DomainOrder = []Domain{DomainPlatform, DomainKungal}

type ExtDef struct {
	Name string
	Type string
}

type Def struct {
	Code        string
	Domain      Domain
	Status      int
	Title       string
	Description string
	Extensions  []ExtDef
}

type ReasonDef struct {
	Reason      string
	Title       string
	Description string
	Params      []string
}

const (
	CodeMalformedBody                = "MALFORMED_BODY"
	CodeInvalidParameter             = "INVALID_PARAMETER"
	CodeUnknownEnumValue             = "UNKNOWN_ENUM_VALUE"
	CodeLimitTooLarge                = "LIMIT_TOO_LARGE"
	CodeInvalidCursor                = "INVALID_CURSOR"
	CodeUnknownSort                  = "UNKNOWN_SORT"
	CodeMissingCredential            = "MISSING_CREDENTIAL"
	CodeInvalidCredential            = "INVALID_CREDENTIAL"
	CodeScopeRequired                = "SCOPE_REQUIRED"
	CodeAccountBanned                = "ACCOUNT_BANNED"
	CodeNotFound                     = "NOT_FOUND"
	CodeMethodNotAllowed             = "METHOD_NOT_ALLOWED"
	CodeIdempotencyKeyReused         = "IDEMPOTENCY_KEY_REUSED"
	CodeIdempotencyRequestInProgress = "IDEMPOTENCY_REQUEST_IN_PROGRESS"
	CodeUnsupportedMediaType         = "UNSUPPORTED_MEDIA_TYPE"
	CodeValidationFailed             = "VALIDATION_FAILED"
	CodeInternalError                = "INTERNAL_ERROR"
	CodeServiceUnavailable           = "SERVICE_UNAVAILABLE"
)

const (
	ReasonRequired         = "REQUIRED"
	ReasonInvalidFormat    = "INVALID_FORMAT"
	ReasonOutOfRange       = "OUT_OF_RANGE"
	ReasonTooLong          = "TOO_LONG"
	ReasonTooShort         = "TOO_SHORT"
	ReasonTooManyItems     = "TOO_MANY_ITEMS"
	ReasonTooFewItems      = "TOO_FEW_ITEMS"
	ReasonDuplicateItem    = "DUPLICATE_ITEM"
	ReasonUnknownValue     = "UNKNOWN_VALUE"
	ReasonNotAllowedValue  = "NOT_ALLOWED_VALUE"
	ReasonUnknownReference = "UNKNOWN_REFERENCE"
	ReasonImmutable        = "IMMUTABLE"
	ReasonInconsistentWith = "INCONSISTENT_WITH"
	ReasonNotPermitted     = "NOT_PERMITTED"
)

const (
	ParamMaxLength = "max_length"
	ParamMinLength = "min_length"
	ParamMinimum   = "minimum"
	ParamMaximum   = "maximum"
	ParamMaxItems  = "max_items"
	ParamMinItems  = "min_items"
	ParamAllowed   = "allowed"
)

var Codes = []Def{
	{CodeMalformedBody, DomainPlatform, http.StatusBadRequest, "Malformed body", "Request body is not valid JSON, or is not a valid instance of the declared media type.", nil},
	{CodeInvalidParameter, DomainPlatform, http.StatusBadRequest, "Invalid parameter", "A parameter is syntactically wrong: a boolean that is not true/false, an integer that is not an integer, a date that is not YYYY-MM-DD.", nil},
	{CodeUnknownEnumValue, DomainPlatform, http.StatusBadRequest, "Unknown enum value", "A closed vocabulary received an unknown token at parse time.", nil},
	{CodeLimitTooLarge, DomainPlatform, http.StatusBadRequest, "Limit too large", "limit is greater than 100. The value is not clamped.", nil},
	{CodeInvalidCursor, DomainPlatform, http.StatusBadRequest, "Invalid cursor", "The cursor cannot be parsed or is no longer valid.", nil},
	{CodeUnknownSort, DomainPlatform, http.StatusBadRequest, "Unknown sort", "sort= received a key this collection has not declared.", nil},
	{CodeMissingCredential, DomainPlatform, http.StatusUnauthorized, "Missing credential", "The request has no Authorization header.", nil},
	{CodeInvalidCredential, DomainPlatform, http.StatusUnauthorized, "Invalid credential", "A credential was sent but it is invalid, expired, or revoked.", nil},
	{CodeScopeRequired, DomainPlatform, http.StatusForbidden, "Scope required", "The credential is valid but lacks the scope this operation needs.", nil},
	{CodeAccountBanned, DomainKungal, http.StatusForbidden, "Account banned", "The signed-in user's account is banned.", nil},
	{CodeNotFound, DomainPlatform, http.StatusNotFound, "Not found", "Nothing visible exists at this URL.", nil},
	{CodeMethodNotAllowed, DomainPlatform, http.StatusMethodNotAllowed, "Method not allowed", "The path exists but this method does not.", nil},
	{CodeIdempotencyKeyReused, DomainPlatform, http.StatusConflict, "Idempotency key reused", "The same Idempotency-Key was sent with a different request body.", nil},
	{CodeIdempotencyRequestInProgress, DomainKungal, http.StatusConflict, "Idempotency request in progress", "A request with the same Idempotency-Key is still being processed. Retry after it completes.", nil},
	{CodeUnsupportedMediaType, DomainPlatform, http.StatusUnsupportedMediaType, "Unsupported media type", "The request body media type is not supported.", nil},
	{CodeValidationFailed, DomainPlatform, http.StatusUnprocessableEntity, "Validation failed", "The request is syntactically valid but semantically not. errors[] is present and non-empty.", nil},
	{CodeInternalError, DomainPlatform, http.StatusInternalServerError, "Internal error", "A bug on our side, including the output of panic recovery.", nil},
	{CodeServiceUnavailable, DomainPlatform, http.StatusServiceUnavailable, "Service unavailable", "A dependency is unavailable. The request may be retried.", nil},
}

var Reasons = []ReasonDef{
	{ReasonRequired, "Required", "A required field is missing or null.", nil},
	{ReasonInvalidFormat, "Invalid format", "The value does not match the expected format (date, URI, hash, id string).", nil},
	{ReasonOutOfRange, "Out of range", "A numeric value is out of range, including a date outside the allowed interval.", []string{ParamMinimum, ParamMaximum}},
	{ReasonTooLong, "Too long", "A string is longer than its maxLength.", []string{ParamMaxLength}},
	{ReasonTooShort, "Too short", "A string is shorter than its minLength.", []string{ParamMinLength}},
	{ReasonTooManyItems, "Too many items", "An array exceeds its item limit.", []string{ParamMaxItems}},
	{ReasonTooFewItems, "Too few items", "The array has fewer items than its minimum.", []string{ParamMinItems}},
	{ReasonDuplicateItem, "Duplicate item", "An array that must be unique contains a duplicate.", nil},
	{ReasonUnknownValue, "Unknown value", "The value is not in this field's closed vocabulary.", []string{ParamAllowed}},
	{ReasonNotAllowedValue, "Not allowed value", "The value is in the vocabulary but is not accepted in this context.", nil},
	{ReasonUnknownReference, "Unknown reference", "The value refers to an entity that is not visible. Absence, merge, and visibility filtering are not distinguished.", nil},
	{ReasonImmutable, "Immutable", "This field cannot be changed in the current state.", nil},
	{ReasonInconsistentWith, "Inconsistent with", "The value contradicts another field. detail names that field's pointer.", nil},
	{ReasonNotPermitted, "Not permitted", "The caller is not allowed to act on this position. Unlike IMMUTABLE this is about the actor, not the field's state.", nil},
}

var (
	codeByName   = indexCodes()
	reasonByName = indexReasons()
	NamePattern  = regexp.MustCompile(`^[A-Z][A-Z0-9_]*[A-Z0-9]$`)
)

func indexCodes() map[string]Def {
	m := make(map[string]Def, len(Codes))
	for _, d := range Codes {
		m[d.Code] = d
	}
	return m
}

func indexReasons() map[string]ReasonDef {
	m := make(map[string]ReasonDef, len(Reasons))
	for _, d := range Reasons {
		m[d.Reason] = d
	}
	return m
}

func Lookup(code string) (Def, bool) {
	d, ok := codeByName[code]
	return d, ok
}

func LookupReason(reason string) (ReasonDef, bool) {
	d, ok := reasonByName[reason]
	return d, ok
}

func (d Def) TypeURI() string {
	return TypeURIPrefix + string(d.Domain) + "/" + Kebab(d.Code)
}

func Kebab(code string) string {
	return strings.ToLower(strings.ReplaceAll(code, "_", "-"))
}

func CodeFromKebab(kebab string) string {
	return strings.ToUpper(strings.ReplaceAll(kebab, "-", "_"))
}

func StatusToCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return CodeInvalidParameter
	case http.StatusUnauthorized:
		return CodeMissingCredential
	case http.StatusForbidden:
		return CodeScopeRequired
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusMethodNotAllowed:
		return CodeMethodNotAllowed
	case http.StatusConflict:
		return CodeIdempotencyKeyReused
	case http.StatusUnsupportedMediaType:
		return CodeUnsupportedMediaType
	case http.StatusUnprocessableEntity:
		return CodeValidationFailed
	case http.StatusServiceUnavailable:
		return CodeServiceUnavailable
	default:
		return CodeInternalError
	}
}

func reasonAllowsParam(reason, key string) bool {
	d, ok := LookupReason(reason)
	if !ok {
		return false
	}
	for _, k := range d.Params {
		if k == key {
			return true
		}
	}
	return false
}
