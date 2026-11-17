package gores

import (
	"debug/buildinfo"
	"io"
	"regexp"
)

// Module is one Go module (main or dependency).
type Module struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	Sum     string `json:"sum,omitempty"`
	Replace string `json:"replace,omitempty"`
	Main    bool   `json:"main,omitempty"`
}

// BuildInfo is the recovered toolchain/module metadata.
type BuildInfo struct {
	GoVersion string   `json:"goVersion"`
	Path      string   `json:"path"`
	Main      *Module  `json:"main,omitempty"`
	Deps      []Module `json:"deps"`
	Settings  []KV     `json:"settings"`
}

// KV is a build setting key/value pair.
type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var goVersionRe = regexp.MustCompile(`go1\.[0-9]+(\.[0-9]+)?((beta|rc)[0-9]+)?`)

// ReadBuildInfo extracts module/build metadata. Falls back to scanning
// for the runtime.buildVersion string when the buildinfo blob is absent.
func ReadBuildInfo(r io.ReaderAt) *BuildInfo {
	bi, err := buildinfo.Read(r)
	if err == nil && bi != nil {
		out := &BuildInfo{GoVersion: bi.GoVersion, Path: bi.Path}
		if bi.Main.Path != "" || bi.Main.Version != "" {
			m := Module{Path: bi.Main.Path, Version: bi.Main.Version, Sum: bi.Main.Sum, Main: true}
			if bi.Main.Replace != nil {
				m.Replace = bi.Main.Replace.Path + " " + bi.Main.Replace.Version
			}
			out.Main = &m
		}
		for _, d := range bi.Deps {
			m := Module{Path: d.Path, Version: d.Version, Sum: d.Sum}
			if d.Replace != nil {
				m.Replace = d.Replace.Path + " " + d.Replace.Version
			}
			out.Deps = append(out.Deps, m)
		}
		for _, s := range bi.Settings {
			out.Settings = append(out.Settings, KV{Key: s.Key, Value: s.Value})
		}
		return out
	}
	// Fallback: scan for a go1.x version string (runtime.buildVersion).
	const chunk = 1 << 20
	var tail []byte
	for off := int64(0); ; off += chunk {
		buf := make([]byte, chunk+16)
		n, err := r.ReadAt(buf, off)
		if n > 0 {
			data := append(tail, buf[:n]...)
			if m := goVersionRe.Find(data); m != nil {
				return &BuildInfo{GoVersion: string(m)}
			}
			if len(data) >= 16 {
				tail = data[len(data)-16:]
			}
		}
		if err != nil {
			break
		}
	}
	return &BuildInfo{}
}
