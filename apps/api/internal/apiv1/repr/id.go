package repr

import (
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const (
	idPattern        = `^[0-9]+$`
	timestampLayout  = "2006-01-02T15:04:05Z"
	timestampPattern = `^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$`
	dateLayout       = "2006-01-02"
	datePattern      = `^[0-9]{4}-[0-9]{2}-[0-9]{2}$`
	FreeTextSentence = "Free text; never use it as a decision input."
)

func ID(n int) DecimalID {
	return DecimalID(strconv.Itoa(n))
}

func ParseID(s DecimalID) (int, bool) {
	n, err := strconv.Atoi(string(s))
	return n, err == nil && n > 0
}

func Timestamp(t time.Time) DateTime {
	return DateTime(t.UTC().Format(timestampLayout))
}

func TimestampPtr(t *time.Time) *DateTime {
	if t == nil {
		return nil
	}
	s := Timestamp(*t)
	return &s
}

func Date(t time.Time) CalendarDate {
	return CalendarDate(t.UTC().Format(dateLayout))
}

type DecimalID string

func (DecimalID) Schema(huma.Registry) *huma.Schema {
	minLen, maxLen := 1, 20
	return &huma.Schema{
		Type:      huma.TypeString,
		Pattern:   idPattern,
		MinLength: &minLen,
		MaxLength: &maxLen,
	}
}

type DateTime string

func (DateTime) Schema(huma.Registry) *huma.Schema {
	n := len(timestampLayout)
	return &huma.Schema{
		Type:      huma.TypeString,
		Format:    "date-time",
		Pattern:   timestampPattern,
		MinLength: &n,
		MaxLength: &n,
	}
}

type CalendarDate string

func (CalendarDate) Schema(huma.Registry) *huma.Schema {
	n := len(dateLayout)
	return &huma.Schema{
		Type:      huma.TypeString,
		Format:    "date",
		Pattern:   datePattern,
		MinLength: &n,
		MaxLength: &n,
	}
}
