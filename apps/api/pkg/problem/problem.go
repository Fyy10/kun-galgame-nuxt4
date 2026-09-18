package problem

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
)

const ContentType = "application/problem+json"

const internalDetail = "An internal error occurred."

type FieldParams struct {
	MaxLength *int      `json:"max_length,omitempty"`
	MinLength *int      `json:"min_length,omitempty"`
	Minimum   *float64  `json:"minimum,omitempty"`
	Maximum   *float64  `json:"maximum,omitempty"`
	MaxItems  *int      `json:"max_items,omitempty"`
	MinItems  *int      `json:"min_items,omitempty"`
	Allowed   *[]string `json:"allowed,omitempty"`
}

type FieldError struct {
	Pointer   *string      `json:"pointer,omitempty"`
	Parameter *string      `json:"parameter,omitempty"`
	Header    *string      `json:"header,omitempty"`
	Reason    string       `json:"reason"`
	Detail    string       `json:"detail"`
	Params    *FieldParams `json:"params,omitempty"`
}

type Problem struct {
	Type      string       `json:"type"`
	Title     string       `json:"title"`
	Status    int          `json:"status"`
	Detail    string       `json:"detail"`
	Instance  string       `json:"instance"`
	Code      string       `json:"code"`
	RequestID string       `json:"request_id"`
	Errors    []FieldError `json:"errors"`
	extra     map[string]any
	cause     error
}

var (
	_ huma.StatusError       = (*Problem)(nil)
	_ huma.ContentTypeFilter = (*Problem)(nil)
)

func (p *Problem) Error() string {
	if p == nil {
		return ""
	}
	if p.Detail != "" {
		return p.Detail
	}
	return p.Title
}

func (p *Problem) GetStatus() int {
	if p == nil {
		return http.StatusInternalServerError
	}
	return p.Status
}

func (p *Problem) ContentType(ct string) string {
	if ct == "application/json" || ct == "" {
		return ContentType
	}
	if ct == "application/cbor" {
		return "application/problem+cbor"
	}
	return ct
}

func New(code, detail string, fields ...FieldError) *Problem {
	def, ok := Lookup(code)
	if !ok {
		def, _ = Lookup(CodeInternalError)
	}
	if fields == nil {
		fields = []FieldError{}
	}
	return &Problem{
		Type:   def.TypeURI(),
		Title:  def.Title,
		Status: def.Status,
		Detail: detail,
		Code:   def.Code,
		Errors: fields,
	}
}

// Internal builds INTERNAL_ERROR. The wrapped text must not appear in detail;
// Write logs the cause with request_id.
func Internal(err error) *Problem {
	p := New(CodeInternalError, internalDetail)
	p.cause = err
	return p
}

func AtPointer(pointer, reason, detail string, params *FieldParams) FieldError {
	return FieldError{Pointer: &pointer, Reason: reason, Detail: detail, Params: params}
}

func AtParameter(name, reason, detail string, params *FieldParams) FieldError {
	return FieldError{Parameter: &name, Reason: reason, Detail: detail, Params: params}
}

func AtHeader(name, reason, detail string, params *FieldParams) FieldError {
	return FieldError{Header: &name, Reason: reason, Detail: detail, Params: params}
}

func (p *Problem) SetExtension(name string, value any) {
	if p == nil {
		panic("problem: SetExtension on nil Problem")
	}
	def, ok := Lookup(p.Code)
	if !ok {
		panic("problem: SetExtension on unknown code " + p.Code)
	}
	allowed := false
	for _, e := range def.Extensions {
		if e.Name == name {
			allowed = true
			break
		}
	}
	if !allowed {
		panic("problem: undeclared extension " + name + " on " + p.Code)
	}
	if p.extra == nil {
		p.extra = map[string]any{}
	}
	p.extra[name] = value
}

func (p *Problem) MarshalJSON() ([]byte, error) {
	type wire struct {
		Type      string       `json:"type"`
		Title     string       `json:"title"`
		Status    int          `json:"status"`
		Detail    string       `json:"detail"`
		Instance  string       `json:"instance"`
		Code      string       `json:"code"`
		RequestID string       `json:"request_id"`
		Errors    []FieldError `json:"errors"`
	}
	w := wire{
		Type:      p.Type,
		Title:     p.Title,
		Status:    p.Status,
		Detail:    p.Detail,
		Instance:  p.Instance,
		Code:      p.Code,
		RequestID: p.RequestID,
		Errors:    p.Errors,
	}
	if w.Errors == nil {
		w.Errors = []FieldError{}
	}
	raw, err := json.Marshal(w)
	if err != nil || len(p.extra) == 0 {
		return raw, err
	}
	obj := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	for k, v := range p.extra {
		ev, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		obj[k] = ev
	}
	return json.Marshal(obj)
}

func Write(c fiber.Ctx, p *Problem) error {
	if p == nil {
		p = New(CodeInternalError, internalDetail)
	}
	if p.Errors == nil {
		p.Errors = []FieldError{}
	}
	p.RequestID = RequestID(c)
	p.Instance = Instance(c)
	if p.cause != nil {
		slog.Error("internal error", "request_id", p.RequestID, "err", p.cause)
	}
	c.Set(HeaderRequestID, p.RequestID)
	c.Set("Cache-Control", "no-store")
	if p.Status == http.StatusUnauthorized {
		c.Set("WWW-Authenticate", `Bearer realm="kungal"`)
	}
	// c.JSON overwrites Content-Type; pass it as the second argument.
	return c.Status(p.Status).JSON(p, ContentType)
}

func Instance(c fiber.Ctx) string {
	path := c.Path()
	if raw := string(c.Request().URI().QueryString()); raw != "" {
		return path + "?" + raw
	}
	return path
}
