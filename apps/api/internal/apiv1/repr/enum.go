package repr

import "github.com/danielgtaylor/huma/v2"

func ClosedEnum(values ...string) *huma.Schema {
	enum := make([]any, len(values))
	maxLen := 0
	for i, v := range values {
		enum[i] = v
		if n := len(v); n > maxLen {
			maxLen = n
		}
	}
	return &huma.Schema{
		Type:      huma.TypeString,
		Enum:      enum,
		MaxLength: &maxLen,
		Extensions: map[string]any{
			"x-vocabulary-closed": true,
		},
	}
}

func OpenEnum(name string, maxLen int) *huma.Schema {
	if maxLen <= 0 {
		maxLen = 64
	}
	return &huma.Schema{
		Type:      huma.TypeString,
		MaxLength: &maxLen,
		Extensions: map[string]any{
			"x-vocabulary-closed": false,
			"x-vocabulary":        name,
		},
	}
}
