package main

import (
	"fmt"
	"os"

	"github.com/SanD94/redline/internal/cli"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: redline <command> [args]\n\n")
		fmt.Fprintf(os.Stderr, "commands:\n")
		fmt.Fprintf(os.Stderr, "  reveal   split a docx into structured markdown and review data\n")
		fmt.Fprintf(os.Stderr, "  audit    compare current markdown sources to the last reveal baseline\n")
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error

	switch cmd {
	case "reveal":
		err = cli.RunReveal(args)
	case "audit":
		err = cli.RunAudit(args)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		fmt.Fprintf(os.Stderr, "usage: redline <command> [args]\n")
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
