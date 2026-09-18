package main

import (
	"fmt"
	"os"
	"path/filepath"

	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/app"
)

func main() {
	root, err := moduleRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if err := apiv1.WriteSpecFiles(filepath.Join(root, "openapi"), app.V1Spec()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func moduleRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", wd)
		}
		dir = parent
	}
}
