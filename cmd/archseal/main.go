package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Divonto/archseal/internal/archseal"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "check":
		runCheck(os.Args[2:])
	case "init":
		runInit()
	case "version", "--version", "-v":
		fmt.Println("archseal", version)
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func runCheck(args []string) {
	configPath := ".archseal.json"
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config":
			if i+1 >= len(args) {
				fatal("--config requires a path")
			}
			i++
			configPath = args[i]
		case "--json":
			jsonOutput = true
		default:
			fatal("unknown flag: " + args[i])
		}
	}

	report, err := archseal.Check(configPath)
	if err != nil {
		fatal(err.Error())
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fatal(err.Error())
		}
	} else {
		fmt.Print(report.Text())
	}

	if len(report.Violations) > 0 {
		os.Exit(1)
	}
}

func runInit() {
	path := ".archseal.json"
	if _, err := os.Stat(path); err == nil {
		fatal(path + " already exists")
	}
	data, err := json.MarshalIndent(archseal.DefaultConfig(), "", "  ")
	if err != nil {
		fatal(err.Error())
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fatal(err.Error())
	}
	abs, _ := filepath.Abs(path)
	fmt.Println("created", abs)
}

func usage() {
	fmt.Print(`archseal — deterministic architecture boundary checks

Usage:
  archseal init
  archseal check [--config path] [--json]
  archseal version

Exit codes:
  0  architecture is sealed
  1  boundary violations found
  2  usage or configuration error
`)
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "archseal:", msg)
	os.Exit(2)
}
