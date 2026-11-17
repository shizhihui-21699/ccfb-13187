package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"gores/internal/gores"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "analyze":
		cmdAnalyze(os.Args[2:])
	case "serve":
		cmdServe(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `gores - Go binary symbol/metadata recovery for stripped malware

Usage:
  gores analyze <binary> [-o out.json] [--line-tables]
  gores serve [-addr :8080] [-web <dir>]

`)
}

func cmdAnalyze(args []string) {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	out := fs.String("o", "", "output JSON file (default stdout)")
	lineTables := fs.Bool("line-tables", false, "include full per-function line tables")
	// flag.Parse stops at the first positional arg; loop to allow
	// flags and positionals in any order.
	var positional []string
	fs.Parse(args)
	for fs.NArg() > 0 {
		positional = append(positional, fs.Arg(0))
		fs.Parse(fs.Args()[1:])
	}
	if len(positional) < 1 {
		usage()
		os.Exit(2)
	}
	path := positional[0]
	f, err := os.Open(path)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	res, err := gores.Analyze(f, path, *lineTables)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
	enc := json.NewEncoder(os.Stdout)
	w := os.Stdout
	if *out != "" {
		of, err := os.Create(*out)
		if err != nil {
			fatal(err)
		}
		defer of.Close()
		w = of
	}
	enc = json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(res); err != nil {
		fatal(err)
	}
	fmt.Fprintf(os.Stderr, "format=%s arch=%s pcln=%s go=%s funcs=%d strings=%d xrefs=%d deps=%d\n",
		res.Format, res.Arch, res.PclnVer, res.Build.GoVersion,
		res.Stats.FuncCount, res.Stats.StringCount, res.Stats.StringXRefCount, res.Stats.DepCount)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
