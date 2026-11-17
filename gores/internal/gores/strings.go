package gores

import (
	"sort"
	"strings"
)

// GoString is an extracted string constant.
type GoString struct {
	Addr  uint64   `json:"addr"`
	Text  string   `json:"text"`
	XRefs []uint64 `json:"xrefs,omitempty"` // entry PCs of functions referencing it
}

// minStringLen is the minimum run length to keep.
const minStringLen = 4

// maxStringLen caps a single recovered string (referenced strings are
// split at the next reference; unreferenced runs are truncated).
const maxStringLen = 4096

// stringSections are section names scanned for string data.
var stringSections = []string{".rodata", "__rodata", ".rdata", "__cstring"}

// strRun is a printable byte run in rodata.
type strRun struct {
	start, end uint64 // virtual addresses
	text       string
}

// extractRuns pulls printable runs out of read-only data sections.
// Go concatenates string literals without separators, so runs are
// further split at referenced addresses in BuildStrings.
func extractRuns(b *BinFile) []strRun {
	var out []strRun
	seen := map[string]bool{}
	for _, name := range stringSections {
		s := b.Section(name)
		if s == nil || seen[s.Name] {
			continue
		}
		seen[s.Name] = true
		data, err := b.ReadSectionData(s)
		if err != nil {
			continue
		}
		out = append(out, runsOf(data, s.Addr)...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].start < out[j].start })
	return out
}

func runsOf(data []byte, base uint64) []strRun {
	var out []strRun
	start := -1
	flush := func(end int) {
		if start >= 0 && end-start >= minStringLen {
			out = append(out, strRun{
				start: base + uint64(start),
				end:   base + uint64(end),
				text:  string(data[start:end]),
			})
		}
		start = -1
	}
	for i, c := range data {
		if c >= 0x20 && c < 0x7f {
			if start < 0 {
				start = i
			}
		} else {
			flush(i)
		}
	}
	flush(len(data))
	return out
}

// BuildStrings produces the final string table: runs are split at every
// referenced address so each Go string literal becomes its own entry,
// with xrefs attached.
func BuildStrings(b *BinFile, refs map[uint64][]uint64) []GoString {
	runs := extractRuns(b)
	var out []GoString
	for _, r := range runs {
		// referenced addresses strictly inside this run
		var cuts []uint64
		for a := range refs {
			if a > r.start && a < r.end {
				cuts = append(cuts, a)
			}
		}
		sort.Slice(cuts, func(i, j int) bool { return cuts[i] < cuts[j] })
		bounds := append([]uint64{r.start}, cuts...)
		for i, bs := range bounds {
			be := r.end
			if i+1 < len(bounds) {
				be = bounds[i+1]
			}
			if be-bs > maxStringLen {
				be = bs + maxStringLen
			}
			if be-bs < minStringLen {
				continue
			}
			off := bs - r.start
			gs := GoString{Addr: bs, Text: r.text[off : off+(be-bs)]}
			if xr, ok := refs[bs]; ok {
				gs.XRefs = xr
			}
			out = append(out, gs)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Addr < out[j].Addr })
	return out
}

// InterestingStrings filters strings that look analyst-relevant
// (URLs, paths, commands, keys) — used by the UI "highlights" view.
func InterestingStrings(strs []GoString) []string {
	var out []string
	for _, s := range strs {
		t := s.Text
		if strings.Contains(t, "http://") || strings.Contains(t, "https://") ||
			strings.Contains(t, "C:\\") || strings.HasPrefix(t, "/etc/") ||
			strings.HasPrefix(t, "/tmp/") || strings.Contains(t, "powershell") ||
			strings.Contains(t, "cmd.exe") || strings.Contains(t, "/bin/sh") {
			out = append(out, t)
		}
	}
	return out
}
