package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Divonto/archseal/internal/archseal"
)

var version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "check":
		runCheck(os.Args[2:])
	case "doctor":
		runDoctor(os.Args[2:])
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
	format := "text"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config":
			if i+1 >= len(args) {
				fatal("--config requires a path")
			}
			i++
			configPath = args[i]
		case "--format":
			if i+1 >= len(args) {
				fatal("--format requires text, json, or sarif")
			}
			i++
			format = args[i]
		case "--json":
			format = "json"
		default:
			fatal("unknown flag: " + args[i])
		}
	}

	report, err := archseal.Check(configPath)
	if err != nil {
		fatal(err.Error())
	}

	switch format {
	case "text":
		fmt.Print(report.Text())
	case "json":
		writeJSON(report)
	case "sarif":
		writeJSON(report.SARIF())
	default:
		fatal("unsupported format: " + format)
	}

	if report.Failed() {
		os.Exit(1)
	}
}

func runDoctor(args []string) {
	configPath := ".archseal.json"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config":
			if i+1 >= len(args) {
				fatal("--config requires a path")
			}
			i++
			configPath = args[i]
		default:
			fatal("unknown flag: " + args[i])
		}
	}

	report, err := archseal.Doctor(configPath)
	if err != nil {
		fatal(err.Error())
	}
	fmt.Print(report.Text())
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

func writeJSON(value any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		fatal(err.Error())
	}
}

func usage() {
	fmt.Print(`archseal — deterministic architecture contracts

Usage:
  archseal init
  archseal doctor [--config path]
  archseal check [--config path] [--format text|json|sarif]
  archseal version

Compatibility:
  archseal check --json

Exit codes:
  0  architecture is sealed
  1  boundary violations or dependency cycles found
  2  usage or configuration error
`)
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "archseal:", msg)
	os.Exit(2)
}
