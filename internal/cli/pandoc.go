package cli

import (
	"encoding/json"
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
	info := inspectPandoc()
	if !info.Found {
		return info, fmt.Errorf("pandoc not found; install Pandoc or set REDLINE_PANDOC")
	}
	return info, nil
}

func inspectPandoc() PandocInfo {
	path := os.Getenv("REDLINE_PANDOC")
	var err error
	if path == "" {
		path, err = exec.LookPath("pandoc")
		if err != nil {
			return PandocInfo{Found: false}
		}
	}
	cmd := exec.Command(path, "--version")
	out, err := cmd.Output()
	if err != nil {
		return PandocInfo{Path: path, Found: false}
	}
	version := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	return PandocInfo{Path: path, Version: version, Found: true}
}

func RunCheck(args []string) error {
	jsonOutput := false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		default:
			return fmt.Errorf("unknown check option: %s", arg)
		}
	}
	report := map[string]any{
		"version": "dev",
		"pandoc":  inspectPandoc(),
	}
	if jsonOutput {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(data))
		return nil
	}
	pandoc := report["pandoc"].(PandocInfo)
	fmt.Fprintf(os.Stdout, "Redline version: dev\n")
	if pandoc.Found {
		fmt.Fprintf(os.Stdout, "Pandoc: found at %s (%s)\n", pandoc.Path, pandoc.Version)
	} else {
		fmt.Fprintf(os.Stdout, "Pandoc: not found (install Pandoc or set REDLINE_PANDOC for disappear/pandoc)\n")
	}
	return nil
}

func RunPandoc(args []string) error {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	pandoc, err := findPandoc()
	if err != nil {
		return err
	}
	cmd := exec.Command(pandoc.Path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pandoc failed: %w", err)
	}
	return nil
}
