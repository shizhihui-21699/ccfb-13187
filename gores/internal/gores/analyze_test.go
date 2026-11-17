package gores

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression test over the stripped samples in samples/bin. Each sample
// is built from samples/hello (see samples/build.sh); we know its main
// package functions, its Go version and one third-party dep.
func TestAnalyzeSamples(t *testing.T) {
	dir := "../../../samples/bin"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skip("samples not built:", err)
	}
	for _, e := range entries {
		e := e
		t.Run(e.Name(), func(t *testing.T) {
			f, err := os.Open(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			res, err := Analyze(f, e.Name(), false)
			if err != nil {
				t.Fatalf("Analyze: %v", err)
			}
			if res.Stats.FuncCount < 3000 {
				t.Errorf("suspiciously few functions: %d", res.Stats.FuncCount)
			}
			// main.main must resolve to main.go:61 (func main in sample).
			var mainMain *Function
			for i := range res.Functions {
				if res.Functions[i].Name == "main.main" {
					mainMain = &res.Functions[i]
					break
				}
			}
			if mainMain == nil {
				t.Fatal("main.main not recovered")
			}
			if !strings.HasSuffix(mainMain.File, "hello/main.go") {
				t.Errorf("main.main file = %q", mainMain.File)
			}
			if mainMain.Line != 61 {
				t.Errorf("main.main line = %d, want 61", mainMain.Line)
			}
			// third-party dep must be detected
			foundDep := false
			for _, d := range res.Build.Deps {
				if d.Path == "github.com/google/uuid" && d.Version == "v1.6.0" {
					foundDep = true
				}
			}
			if !foundDep {
				t.Errorf("uuid dep not found: %+v", res.Build.Deps)
			}
			// planted C2 string must be xref'd to main.beacon
			nameByEntry := map[uint64]string{}
			for _, fn := range res.Functions {
				nameByEntry[fn.Entry] = fn.Name
			}
			foundStr := false
			for _, s := range res.Strings {
				if strings.Contains(s.Text, "cdn-metrics.example.net") {
					for _, x := range s.XRefs {
						if nameByEntry[x] == "main.beacon" {
							foundStr = true
						}
					}
				}
			}
			if res.Arch == "amd64" || res.Arch == "arm64" || res.Arch == "386" {
				if !foundStr {
					t.Errorf("C2 string not xref'd to main.beacon")
				}
			}
			// file coverage: nearly all functions should have a file
			nofile := 0
			for _, fn := range res.Functions {
				if fn.File == "" {
					nofile++
				}
			}
			if nofile > len(res.Functions)/20 {
				t.Errorf("too many functions without file: %d/%d", nofile, len(res.Functions))
			}
		})
	}
}
