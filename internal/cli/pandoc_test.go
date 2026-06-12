package cli

import "testing"

func TestRunCheckJSONDoesNotRequirePandoc(t *testing.T) {
	t.Setenv("REDLINE_PANDOC", "")
	if err := RunCheck([]string{"--json"}); err != nil {
		t.Fatalf("RunCheck() error = %v", err)
	}
}
