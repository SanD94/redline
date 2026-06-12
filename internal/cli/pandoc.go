package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type PandocInfo struct {
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
	Found   bool   `json:"found"`
}

func findPandoc() (PandocInfo, error) {
	path := os.Getenv("REDLINE_PANDOC")
	var err error
	if path == "" {
		path, err = exec.LookPath("pandoc")
		if err != nil {
			return PandocInfo{Found: false}, fmt.Errorf("pandoc not found; install Pandoc or set REDLINE_PANDOC")
		}
	}
	cmd := exec.Command(path, "--version")
	out, err := cmd.Output()
	if err != nil {
		return PandocInfo{Path: path, Found: false}, fmt.Errorf("pandoc check failed: %w", err)
	}
	version := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	return PandocInfo{Path: path, Version: version, Found: true}, nil
}
