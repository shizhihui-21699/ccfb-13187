package gores

import (
	"fmt"
	"sort"
)

// Function is one recovered function.
type Function struct {
	Name      string `json:"name"`
	Entry     uint64 `json:"entry"`
	End       uint64 `json:"end"`
	File      string `json:"file"`
	Line      int32  `json:"line"` // declaration line (startLine)
	Package   string `json:"package"`
	IsMethod  bool   `json:"isMethod"`
	LineTable []LineEntry `json:"lineTable,omitempty"`
}

// LineEntry maps a PC range [Start,End) to a source line.
type LineEntry struct {
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
	Line  int32  `json:"line"`
}

// _func field offsets (go1.16+ layout).
const (
	funcEntryOff   = 0
	funcNameOff    = 4
	funcPcfile     = 20
	funcPcln       = 24
	funcCuOffset   = 32
	funcStartLine  = 36
	funcStructSize = 44
)

// ParseFunctions recovers the full function table.
func (t *PclnTab) ParseFunctions(withLineTables bool) ([]Function, error) {
	if !t.go116plus {
		return nil, fmt.Errorf("pclntab version %s not supported for function parsing (need go1.16+)", t.Version)
	}
	funcs := make([]Function, 0, t.NFunc)
	for i := 0; i < t.NFunc; i++ {
		entryPC, funcOff, err := t.functabEntry(i)
		if err != nil {
			return nil, err
		}
		foff := t.PclnOff + funcOff
		if foff+40 > uint64(len(t.data)) {
			return nil, fmt.Errorf("func %d: _func out of range", i)
		}
		nameOff := int32(t.u32(foff + funcNameOff))
		name := ""
		if nameOff >= 0 {
			name = t.cstring(t.FuncnameOff + uint64(uint32(nameOff)))
		}
		cuOffset := t.u32(foff + funcCuOffset)
		pcfileOff := t.u32(foff + funcPcfile)
		pclnOff := t.u32(foff + funcPcln)

		// go1.20+ has startLine and the pcln table stores deltas from
		// it; older versions store absolute line numbers.
		var startLine int32
		if t.hasStartLine {
			startLine = int32(t.u32(foff + funcStartLine))
		} else {
			startLine = t.pcvalue(pclnOff, entryPC, entryPC)
		}

		f := Function{
			Name:  name,
			Entry: entryPC,
			Line:  startLine,
		}
		// End = next functab entry.
		if i+1 <= t.NFunc {
			if next, _, err := t.functabEntry(i + 1); err == nil {
				f.End = next
			}
		}
		// Source file via pcfile at entry PC.
		fileno := t.pcvalue(pcfileOff, entryPC, entryPC)
		f.File = t.fileName(cuOffset, fileno)
		f.Package, f.IsMethod = packageOf(name)
		if withLineTables {
			steps := t.decodePcTable(pclnOff, entryPC)
			prev := entryPC
			for _, s := range steps {
				abs := s.Value
				if t.hasStartLine {
					abs += startLine
				}
				f.LineTable = append(f.LineTable, LineEntry{Start: prev, End: s.EndPC, Line: abs})
				prev = s.EndPC
			}
		}
		funcs = append(funcs, f)
	}
	sort.Slice(funcs, func(i, j int) bool { return funcs[i].Entry < funcs[j].Entry })
	return funcs, nil
}

// functabEntry returns (entryPC, funcOff) for functab index i.
// i may be NFunc (the sentinel end entry).
func (t *PclnTab) functabEntry(i int) (uint64, uint64, error) {
	if t.go118plus {
		off := t.PclnOff + uint64(i)*8
		if off+8 > uint64(len(t.data)) {
			return 0, 0, fmt.Errorf("functab entry %d out of range", i)
		}
		entryOff := t.u32(off)
		funcOff := t.u32(off + 4)
		return t.TextStart + uint64(entryOff), uint64(funcOff), nil
	}
	// go1.16: pairs of uintptr, entry is absolute PC.
	sz := uint64(t.PtrSize)
	off := t.PclnOff + uint64(i)*2*sz
	if off+2*sz > uint64(len(t.data)) {
		return 0, 0, fmt.Errorf("functab entry %d out of range", i)
	}
	var entry, funcOff uint64
	if t.PtrSize == 8 {
		entry = t.ByteOrder.Uint64(t.data[off : off+8])
		funcOff = t.ByteOrder.Uint64(t.data[off+sz : off+sz+8])
	} else {
		entry = uint64(t.u32(off))
		funcOff = uint64(t.u32(off + sz))
	}
	return entry, funcOff, nil
}

// fileName resolves a (cuOffset, fileno) pair to a source file path.
// cuOffset is an element index into cutab; the CU's file list lives in
// cutab itself: fileoff = cutab[cuOffset+fileno], name = filetab[fileoff].
func (t *PclnTab) fileName(cuOffset uint32, fileno int32) string {
	if fileno < 0 {
		return ""
	}
	fileOff := t.u32(t.CUOff + (uint64(cuOffset)+uint64(fileno))*4)
	if fileOff == ^uint32(0) {
		return ""
	}
	return t.cstring(t.FiletabOff + uint64(fileOff))
}

// packageOf extracts the package path from a symbol name like
// "net/http.(*Client).Get" or "main.main".
func packageOf(name string) (pkg string, isMethod bool) {
	s := name
	// strip method receiver: pkg/path.(*Type).Method or pkg/path.Type.Method
	rest := s
	if i := lastSlash(s); i >= 0 {
		rest = s[i+1:]
	}
	dot := indexByte(rest, '.')
	if dot < 0 {
		return "", false
	}
	pkg = s[:len(s)-len(rest)+dot]
	after := rest[dot+1:]
	isMethod = len(after) > 0 && (after[0] == '(' || containsDotBeforeEnd(after))
	return pkg, isMethod
}

func lastSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func containsDotBeforeEnd(s string) bool {
	if len(s) > 0 && s[0] == '(' {
		return true
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return true
		}
	}
	return false
}
