package testdb

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestOpenFailsWhenRequiredWithoutDSN(t *testing.T) {
	if os.Getenv("TEST_OPEN_CHILD") == "1" {
		t.Setenv("KUN_REQUIRE_TEST_DB", "1")
		_ = os.Unsetenv(EnvVar)
		Open(t)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestOpenFailsWhenRequiredWithoutDSN$", "-test.v")
	env := make([]string, 0, len(os.Environ())+2)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, EnvVar+"=") || strings.HasPrefix(e, "KUN_REQUIRE_TEST_DB=") || strings.HasPrefix(e, "TEST_OPEN_CHILD=") {
			continue
		}
		env = append(env, e)
	}
	cmd.Env = append(env, "TEST_OPEN_CHILD=1", "KUN_REQUIRE_TEST_DB=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("child passed, want fail:\n%s", out)
	}
	if !strings.Contains(string(out), EnvVar) {
		t.Fatalf("child output missing %s:\n%s", EnvVar, out)
	}
}
