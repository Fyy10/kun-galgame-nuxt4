package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

func CheckG5G13(doc *huma.OpenAPI) []string {
	var errs []string
	errs = append(errs, checkRegistryInvariants()...)
	errs = append(errs, checkProblemsFile()...)
	errs = append(errs, checkFieldErrorSchema(doc)...)
	errs = append(errs, scanProblemUses(apiRoot())...)
	return errs
}

func checkRegistryInvariants() []string {
	var errs []string
	seen := map[string]int{}
	uri := map[string]string{}
	for _, d := range problem.Codes {
		if _, ok := seen[d.Code]; ok {
			errs = append(errs, "G13: duplicate code "+d.Code)
		}
		seen[d.Code] = d.Status
		if prev, ok := uri[d.TypeURI()]; ok {
			errs = append(errs, "G13: type URI collision "+d.TypeURI()+" for "+prev+" and "+d.Code)
		}
		uri[d.TypeURI()] = d.Code
		if problem.CodeFromKebab(problem.Kebab(d.Code)) != d.Code {
			errs = append(errs, "G13: "+d.Code+" is not bijective with its type URI")
		}
		if !strings.HasPrefix(d.TypeURI(), problem.TypeURIPrefix) {
			errs = append(errs, "G5: "+d.Code+" type URI is not under developer.nextmoe.dev/problems")
		}
	}
	for _, r := range problem.Reasons {
		if _, ok := seen[r.Reason]; ok {
			errs = append(errs, "G13: reason "+r.Reason+" collides with a top-level code")
		}
	}
	ft := reflect.TypeOf(problem.FieldError{})
	loc := 0
	for i := 0; i < ft.NumField(); i++ {
		name := strings.Split(ft.Field(i).Tag.Get("json"), ",")[0]
		if name == "code" {
			errs = append(errs, "G13: errors[] must not have a json field named code")
		}
		if name == "pointer" || name == "parameter" || name == "header" {
			loc++
		}
	}
	if loc != 3 {
		errs = append(errs, "G13: FieldError must expose pointer, parameter, and header")
	}
	return errs
}

func checkProblemsFile() []string {
	raw, err := os.ReadFile(filepath.Join(apiRoot(), "openapi", "problems.json"))
	if err != nil {
		return []string{"G5: read problems.json: " + err.Error()}
	}
	var file struct {
		Codes []struct {
			Code   string `json:"code"`
			Status int    `json:"status"`
			Type   string `json:"type"`
		} `json:"codes"`
		Reasons []struct {
			Reason string `json:"reason"`
		} `json:"reasons"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return []string{"G5: problems.json: " + err.Error()}
	}
	var errs []string
	if len(file.Codes) != len(problem.Codes) {
		errs = append(errs, fmt.Sprintf("G5: problems.json has %d codes, registry has %d", len(file.Codes), len(problem.Codes)))
	}
	byCode := map[string]problem.Def{}
	for _, d := range problem.Codes {
		byCode[d.Code] = d
	}
	seen := map[string]bool{}
	for _, c := range file.Codes {
		d, ok := byCode[c.Code]
		if !ok {
			errs = append(errs, "G5: problems.json has unknown code "+c.Code)
			continue
		}
		seen[c.Code] = true
		if c.Status != d.Status || c.Type != d.TypeURI() {
			errs = append(errs, "G5: problems.json mismatch for "+c.Code)
		}
	}
	for _, d := range problem.Codes {
		if !seen[d.Code] {
			errs = append(errs, "G5: problems.json missing "+d.Code)
		}
	}
	if len(file.Reasons) != len(problem.Reasons) {
		errs = append(errs, fmt.Sprintf("G5: problems.json has %d reasons, registry has %d", len(file.Reasons), len(problem.Reasons)))
	}
	byReason := map[string]bool{}
	for _, r := range problem.Reasons {
		byReason[r.Reason] = true
	}
	seenR := map[string]bool{}
	for _, r := range file.Reasons {
		if !byReason[r.Reason] {
			errs = append(errs, "G5: problems.json has unknown reason "+r.Reason)
			continue
		}
		seenR[r.Reason] = true
	}
	for _, r := range problem.Reasons {
		if !seenR[r.Reason] {
			errs = append(errs, "G5: problems.json missing "+r.Reason)
		}
	}
	return errs
}

func checkFieldErrorSchema(doc *huma.OpenAPI) []string {
	if doc == nil || doc.Components == nil || doc.Components.Schemas == nil {
		return nil
	}
	s := doc.Components.Schemas.Map()["FieldError"]
	if s == nil {
		return nil
	}
	var errs []string
	if _, ok := s.Properties["code"]; ok {
		errs = append(errs, "G13: FieldError schema has a code property")
	}
	for _, name := range []string{"pointer", "parameter", "header"} {
		if s.Properties[name] == nil {
			errs = append(errs, "G13: FieldError schema missing "+name)
		}
	}
	return errs
}
