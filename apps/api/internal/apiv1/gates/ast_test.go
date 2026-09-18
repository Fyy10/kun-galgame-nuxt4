package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for path, src := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func expectExactly(t *testing.T, errs []string, want ...string) {
	t.Helper()
	if len(errs) != len(want) {
		t.Fatalf("got %d violations, want %d:\n%s", len(errs), len(want), strings.Join(errs, "\n"))
	}
	for i := range want {
		if !strings.Contains(errs[i], want[i]) {
			t.Errorf("violation %d = %q, want it to contain %q", i, errs[i], want[i])
		}
	}
}

func TestProblemUseScan(t *testing.T) {
	root := tree(t, map[string]string{
		"pkg/problem/codes.go": "package problem\n\nconst (\n\tCodeNotFound = \"NOT_FOUND\"\n\tCodeGhost    = \"GHOST\"\n\tReasonGhost  = \"GHOST_REASON\"\n)\n",
		"internal/apiv1/ok.go": "package apiv1\n\nimport \"kun-galgame-api/pkg/problem\"\n\nvar _ = problem.New(problem.CodeNotFound, \"x\")\n",
		"internal/topic/apiv1/bad.go": `package apiv1
import "kun-galgame-api/pkg/problem"
var (
	_ = problem.New("NOT_FOUND", "x")
	_ = problem.New(problem.CodeGhost, "x")
	_ = problem.AtParameter("q", "REQUIRED", "x", nil)
	_ = problem.FieldError{Reason: problem.ReasonGhost}
	_ = problem.FieldError{Reason: "REQUIRED"}
)
`,
		"internal/topic/apiv1/bad_test.go": "package apiv1\n\nimport \"kun-galgame-api/pkg/problem\"\n\nvar _ = problem.New(\"IN_A_TEST\", \"x\")\n",
	})
	expectExactly(t, scanProblemUses(root),
		"bad.go:4 string literal passed to problem.New",
		"bad.go:5 problem.CodeGhost (GHOST) is not in the code registry",
		"bad.go:6 string literal passed to problem.AtParameter",
		"bad.go:7 problem.ReasonGhost (GHOST_REASON) is not in the reason registry",
		"bad.go:8 string literal in problem.FieldError.Reason",
	)
}

func TestSourceScanOfAnEmptyTreeFails(t *testing.T) {
	expectExactly(t, scanProblemUses(t.TempDir()), "no Go files under", "no Go files under")
}

func TestSourceScanReadsTheRealTree(t *testing.T) {
	files, errs := sourceFiles(apiRoot())
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	var names []string
	for _, f := range files {
		names = append(names, filepath.Base(f.fset.File(f.file.Pos()).Name()))
	}
	for _, want := range []string{"seal.go", "problem.go", "cursor.go"} {
		if !strings.Contains(strings.Join(names, " "), want) {
			t.Errorf("scan missed %s: %v", want, names)
		}
	}
}

func TestOmitemptyScan(t *testing.T) {
	root := tree(t, map[string]string{
		"pkg/problem/p.go": "package problem\n",
		"internal/apiv1/dto.go": "package apiv1\n\ntype T struct {\n" +
			"\tA string `json:\"a,omitempty\"`\n" +
			"\tB *string `json:\"b,omitempty\"`\n" +
			"\tC []int `json:\"c,omitzero\"`\n" +
			"\tD string `json:\"d\" doc:\"omitempty\"`\n" +
			"}\n",
	})
	expectExactly(t, checkOmitempty(root), "dto.go:4 omitempty on a non-pointer field", "dto.go:6 omitzero on a non-pointer field")
}
