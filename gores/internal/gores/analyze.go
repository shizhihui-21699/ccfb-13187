package gores

import (
	"io"
	"sort"
)

// Result is the full analysis report returned to the UI / CLI.
type Result struct {
	FileName  string      `json:"fileName"`
	Format    string      `json:"format"`
	Arch      string      `json:"arch"`
	PclnVer   string      `json:"pclnVersion"`
	Build     *BuildInfo  `json:"build"`
	Functions []Function  `json:"functions"`
	Strings   []GoString  `json:"strings"`
	Packages  []PkgStat   `json:"packages"`
	Stats     Stats       `json:"stats"`
}

// PkgStat aggregates function counts per package.
type PkgStat struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

// Stats holds summary counters.
type Stats struct {
	FuncCount       int `json:"funcCount"`
	StringCount     int `json:"stringCount"`
	StringXRefCount int `json:"stringXRefCount"`
	DepCount        int `json:"depCount"`
}

// Analyze runs the full recovery pipeline over a binary.
func Analyze(r io.ReaderAt, fileName string, withLineTables bool) (*Result, error) {
	bf, err := Open(r)
	if err != nil {
		return nil, err
	}
	res := &Result{
		FileName: fileName,
		Format:   bf.Format,
		Arch:     bf.Arch,
		Build:    ReadBuildInfo(r),
	}
	tab, err := FindPclnTab(bf)
	if err != nil {
		return res, err
	}
	res.PclnVer = tab.Version
	funcs, err := tab.ParseFunctions(withLineTables)
	if err != nil {
		return res, err
	}
	res.Functions = funcs

	refs := CollectStringRefs(bf, funcs)
	strs := BuildStrings(bf, refs)
	xrefs := 0
	for _, s := range strs {
		if len(s.XRefs) > 0 {
			xrefs++
		}
	}
	res.Strings = strs

	pkgs := map[string]int{}
	for _, f := range funcs {
		p := f.Package
		if p == "" {
			p = "(unknown)"
		}
		pkgs[p]++
	}
	for p, c := range pkgs {
		res.Packages = append(res.Packages, PkgStat{Path: p, Count: c})
	}
	sort.Slice(res.Packages, func(i, j int) bool { return res.Packages[i].Count > res.Packages[j].Count })

	res.Stats = Stats{
		FuncCount:       len(funcs),
		StringCount:     len(strs),
		StringXRefCount: xrefs,
		DepCount:        len(res.Build.Deps),
	}
	return res, nil
}
