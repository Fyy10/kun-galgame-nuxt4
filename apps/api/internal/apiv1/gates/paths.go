package gates

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/danielgtaylor/huma/v2"
)

var pathVar = regexp.MustCompile(`\{([^}]+)\}`)

// huma drops path fields promoted from an embedded struct of an unexported type,
// so W2's first draft published /topics/{topic_id} with no topic_id parameter and
// no derived 400, and every other gate passed it.
func CheckF10(doc *huma.OpenAPI) []string {
	var errs []string
	for path, item := range doc.Paths {
		var vars []string
		for _, m := range pathVar.FindAllStringSubmatch(path, -1) {
			vars = append(vars, m[1])
		}
		for _, op := range pathOps(item) {
			var params []string
			for _, p := range op.Parameters {
				if p.In == "path" {
					params = append(params, p.Name)
				}
			}
			for _, v := range vars {
				if !slices.Contains(params, v) {
					errs = append(errs, fmt.Sprintf("F10: %s %s has no path parameter %s", opMethod(op), path, v))
				}
			}
			for _, p := range params {
				if !slices.Contains(vars, p) {
					errs = append(errs, fmt.Sprintf("F10: %s %s declares path parameter %s that the path does not contain", opMethod(op), path, p))
				}
			}
		}
	}
	return errs
}
