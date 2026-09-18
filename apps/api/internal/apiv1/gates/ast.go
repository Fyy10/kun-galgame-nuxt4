package gates

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"kun-galgame-api/pkg/problem"
)

type goFile struct {
	fset *token.FileSet
	file *ast.File
}

func apiRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}

func sourceFiles(root string) ([]goFile, []string) {
	var files []goFile
	var errs []string
	dirs := []string{filepath.Join(root, "internal", "apiv1"), filepath.Join(root, "pkg", "problem")}
	domains, _ := filepath.Glob(filepath.Join(root, "internal", "*", "apiv1"))
	for i, dir := range append(dirs, domains...) {
		found := 0
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			files = append(files, goFile{fset, f})
			found++
			return nil
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, "source gates: scan "+dir+": "+err.Error())
		} else if found == 0 && i < len(dirs) {
			errs = append(errs, "source gates: no Go files under "+dir)
		}
	}
	return files, errs
}

func problemConsts(files []goFile) map[string]string {
	consts := map[string]string{}
	for _, gf := range files {
		if gf.file.Name.Name != "problem" {
			continue
		}
		ast.Inspect(gf.file, func(n ast.Node) bool {
			vs, ok := n.(*ast.ValueSpec)
			if !ok {
				return true
			}
			for i, name := range vs.Names {
				if i >= len(vs.Values) {
					break
				}
				if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					consts[name.Name], _ = strconv.Unquote(lit.Value)
				}
			}
			return true
		})
	}
	return consts
}

func scanProblemUses(root string) []string {
	files, errs := sourceFiles(root)
	consts := problemConsts(files)
	for _, gf := range files {
		errs = append(errs, problemUses(gf, consts)...)
	}
	return errs
}

var codeArgs = map[string]int{
	"New": 0, "Lookup": 0, "LookupReason": 0,
	"AtPointer": 1, "AtParameter": 1, "AtHeader": 1,
}

func problemUses(gf goFile, consts map[string]string) []string {
	var errs []string
	at := func(n ast.Node) string {
		p := gf.fset.Position(n.Pos())
		return fmt.Sprintf("%s:%d", filepath.Base(p.Filename), p.Line)
	}
	literal := func(e ast.Expr) bool {
		lit, ok := e.(*ast.BasicLit)
		return ok && lit.Kind == token.STRING
	}
	inProblem := gf.file.Name.Name == "problem"
	ast.Inspect(gf.file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			name, ok := problemFunc(x.Fun, inProblem)
			if i, takes := codeArgs[name]; ok && takes && i < len(x.Args) && literal(x.Args[i]) {
				errs = append(errs, fmt.Sprintf("G5: %s string literal passed to problem.%s where a registry constant belongs", at(x), name))
			}
		case *ast.CompositeLit:
			name, ok := problemFunc(x.Type, inProblem)
			if !ok || (name != "FieldError" && name != "Problem") {
				return true
			}
			for _, el := range x.Elts {
				kv, ok := el.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				if key, _ := kv.Key.(*ast.Ident); key != nil && (key.Name == "Code" || key.Name == "Reason") && literal(kv.Value) {
					errs = append(errs, fmt.Sprintf("G5: %s string literal in problem.%s.%s", at(x), name, key.Name))
				}
			}
		case *ast.SelectorExpr:
			pkg, ok := x.X.(*ast.Ident)
			if !ok || pkg.Name != "problem" {
				return true
			}
			val, known := consts[x.Sel.Name]
			if !known {
				return true
			}
			switch {
			case strings.HasPrefix(x.Sel.Name, "Code"):
				if _, found := problem.Lookup(val); !found {
					errs = append(errs, fmt.Sprintf("G5: %s problem.%s (%s) is not in the code registry", at(x), x.Sel.Name, val))
				}
			case strings.HasPrefix(x.Sel.Name, "Reason"):
				if _, found := problem.LookupReason(val); !found {
					errs = append(errs, fmt.Sprintf("G5: %s problem.%s (%s) is not in the reason registry", at(x), x.Sel.Name, val))
				}
			}
		}
		return true
	})
	return errs
}

func problemFunc(e ast.Expr, inProblem bool) (string, bool) {
	switch fn := e.(type) {
	case *ast.Ident:
		return fn.Name, inProblem
	case *ast.SelectorExpr:
		pkg, ok := fn.X.(*ast.Ident)
		return fn.Sel.Name, ok && pkg.Name == "problem"
	}
	return "", false
}

func checkOmitempty(root string) []string {
	files, errs := sourceFiles(root)
	for _, gf := range files {
		ast.Inspect(gf.file, func(n ast.Node) bool {
			f, ok := n.(*ast.Field)
			if !ok || f.Tag == nil {
				return true
			}
			if _, pointer := f.Type.(*ast.StarExpr); pointer {
				return true
			}
			tag, _ := strconv.Unquote(f.Tag.Value)
			_, opts, _ := strings.Cut(reflect.StructTag(tag).Get("json"), ",")
			for opt := range strings.SplitSeq(opts, ",") {
				if opt == "omitempty" || opt == "omitzero" {
					p := gf.fset.Position(f.Pos())
					errs = append(errs, fmt.Sprintf("G9: %s:%d %s on a non-pointer field", filepath.Base(p.Filename), p.Line, opt))
				}
			}
			return true
		})
	}
	return errs
}

var localeText = regexp.MustCompile(`[\p{Han}\p{Hiragana}\p{Katakana}\p{Hangul}]`)

func scanLocaleText(root string) []string {
	files, errs := sourceFiles(root)
	for _, gf := range files {
		ast.Inspect(gf.file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if ok && lit.Kind == token.STRING && localeText.MatchString(lit.Value) {
				p := gf.fset.Position(lit.Pos())
				errs = append(errs, fmt.Sprintf("F8: %s:%d string literal in a human language; the client localizes, so send a code or a null", filepath.Base(p.Filename), p.Line))
			}
			return true
		})
	}
	return errs
}
