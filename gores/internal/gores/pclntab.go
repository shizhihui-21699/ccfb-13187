package gores

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// pclntab magic values (little endian on disk).
const (
	go12magic  = 0xfffffffb // Go 1.2 – 1.15
	go116magic = 0xfffffffa // Go 1.16 – 1.17
	go118magic = 0xfffffff0 // Go 1.18 – 1.19
	go120magic = 0xfffffff1 // Go 1.20+
)

// PclnTab is a parsed view of runtime.pclntab.
type PclnTab struct {
	data     []byte // raw pclntab bytes
	ByteOrder binary.ByteOrder
	Version  string // "go1.2", "go1.16", "go1.18", "go1.20"
	PtrSize  int
	Quantum  uint32 // instruction size quantum (minLC)

	NFunc     int
	NFiles    uint32
	TextStart uint64 // runtime.text VA (go1.18+; for older, taken from binary)

	FuncnameOff uint64
	CUOff       uint64
	FiletabOff  uint64
	PctabOff    uint64
	PclnOff     uint64

	go118plus bool
	go116plus bool
	hasStartLine bool // go1.20+ _func carries startLine; pcln stores deltas
}

// FindPclnTab locates and parses the pclntab in the binary.
func FindPclnTab(b *BinFile) (*PclnTab, error) {
	// 1. Named section (ELF .gopclntab, Mach-O __gopclntab).
	for _, name := range []string{".gopclntab", "__gopclntab"} {
		if s := b.Section(name); s != nil {
			data, err := b.ReadSectionData(s)
			if err != nil {
				return nil, err
			}
			if t, err := parsePclnTab(data, b); err == nil {
				return t, nil
			}
		}
	}
	// 2. Magic scan over data sections (PE, or stripped section headers).
	for _, s := range b.Sections {
		if s.Size == 0 || s.Size > 512<<20 {
			continue
		}
		data, err := b.ReadSectionData(s)
		if err != nil || len(data) < 16 {
			continue
		}
		for _, magic := range []uint32{go120magic, go118magic, go116magic, go12magic} {
			var mg [4]byte
			binary.LittleEndian.PutUint32(mg[:], magic)
			idx := 0
			for {
				i := bytes.Index(data[idx:], mg[:])
				if i < 0 {
					break
				}
				cand := data[idx+i:]
				if t, err := parsePclnTab(cand, b); err == nil {
					return t, nil
				}
				idx += i + 4
				if idx >= len(data) {
					break
				}
			}
		}
	}
	return nil, errors.New("pclntab not found (not a Go binary or heavily packed)")
}

func parsePclnTab(data []byte, b *BinFile) (*PclnTab, error) {
	if len(data) < 16 {
		return nil, errors.New("pclntab too small")
	}
	t := &PclnTab{data: data, ByteOrder: binary.LittleEndian}
	magic := t.ByteOrder.Uint32(data[:4])
	switch magic {
	case go120magic:
		t.Version = "go1.20"
		t.go118plus, t.go116plus = true, true
		t.hasStartLine = true
	case go118magic:
		t.Version = "go1.18"
		t.go118plus, t.go116plus = true, true
	case go116magic:
		t.Version = "go1.16"
		t.go116plus = true
	case go12magic:
		t.Version = "go1.2"
	default:
		return nil, fmt.Errorf("bad magic 0x%x", magic)
	}
	t.Quantum = uint32(data[6])
	t.PtrSize = int(data[7])
	if t.PtrSize != 4 && t.PtrSize != 8 {
		return nil, fmt.Errorf("bad ptrSize %d", t.PtrSize)
	}
	rd := func(off uint64) uint64 {
		if t.PtrSize == 8 {
			return t.ByteOrder.Uint64(data[off : off+8 : off+8])
		}
		return uint64(t.ByteOrder.Uint32(data[off : off+4 : off+4]))
	}
	if t.go116plus {
		// header: magic(4) pad(2) quantum(1) ptrsize(1) nfunc nfiles
		//         [textStart] funcnameOff cuOff filetabOff pctabOff pclnOff
		if len(data) < 8+2*t.PtrSize {
			return nil, errors.New("header truncated")
		}
		t.NFunc = int(rd(8))
		t.NFiles = uint32(rd(8 + uint64(t.PtrSize)))
		off := uint64(8 + 2*t.PtrSize)
		if t.go118plus {
			t.TextStart = rd(off)
			off += uint64(t.PtrSize)
		}
		fields := []*uint64{&t.FuncnameOff, &t.CUOff, &t.FiletabOff, &t.PctabOff, &t.PclnOff}
		for _, f := range fields {
			if int(off)+t.PtrSize > len(data) {
				return nil, errors.New("header truncated")
			}
			*f = rd(off)
			off += uint64(t.PtrSize)
		}
	} else {
		// go1.2 header: magic(4) pad(2) quantum(1) ptrsize(1) nfunc
		t.NFunc = int(rd(8))
		t.PclnOff = 8 + uint64(t.PtrSize)
		// Remaining tables are located via functab walk; filled in below.
	}
	if t.NFunc <= 0 || t.NFunc > 50_000_000 {
		return nil, fmt.Errorf("implausible nfunc %d", t.NFunc)
	}
	// Sanity: table offsets must be inside the data we have (when known).
	if t.go116plus {
		for _, o := range []uint64{t.FuncnameOff, t.PctabOff, t.PclnOff} {
			if o >= uint64(len(data)) {
				return nil, fmt.Errorf("table offset 0x%x out of range", o)
			}
		}
	}
	if t.TextStart == 0 {
		if ts := b.TextSection(); ts != nil {
			t.TextStart = ts.Addr
		}
	}
	return t, nil
}

// u32 reads a little-endian uint32 from the pclntab data.
func (t *PclnTab) u32(off uint64) uint32 {
	if off+4 > uint64(len(t.data)) {
		return 0
	}
	return t.ByteOrder.Uint32(t.data[off : off+4 : off+4])
}

func (t *PclnTab) cstring(off uint64) string {
	if off >= uint64(len(t.data)) {
		return ""
	}
	s := t.data[off:]
	if i := bytes.IndexByte(s, 0); i >= 0 {
		s = s[:i]
	}
	return string(s)
}

// readUvarint decodes a uvarint from the pctab stream.
func readUvarint(p []byte) (v uint32, n int) {
	var shift uint
	for i, c := range p {
		v |= uint32(c&0x7f) << shift
		if c < 0x80 {
			return v, i + 1
		}
		shift += 7
		if shift >= 35 {
			break
		}
	}
	return 0, 0
}

// readVarint decodes a signed varint (sign in low bit).
func readVarint(p []byte) (int32, int) {
	u, n := readUvarint(p)
	v := int32(u >> 1)
	if u&1 != 0 {
		v = ^v
	}
	return v, n
}

// pcvalue evaluates a pc-value table (pcsp/pcfile/pcln) at targetpc.
// entry is the function entry PC; off is the table offset into pctab.
func (t *PclnTab) pcvalue(off uint32, entry, targetpc uint64) int32 {
	if off == 0 || t.PctabOff+uint64(off) >= uint64(len(t.data)) {
		return -1
	}
	p := t.data[t.PctabOff+uint64(off):]
	val := int32(-1)
	pc := entry
	for {
		vdelta, n := readVarint(p)
		if n == 0 {
			return -1
		}
		p = p[n:]
		if vdelta == 0 && pc != entry {
			return -1 // terminator (uvdelta==0 after first step)
		}
		val += vdelta
		pcdelta, n := readUvarint(p)
		if n == 0 {
			return -1
		}
		p = p[n:]
		pc += uint64(pcdelta) * uint64(t.Quantum)
		if targetpc < pc {
			return val
		}
	}
}

// PcValueTable fully decodes a pc-value table into (endPC, value) steps.
type PcValue struct {
	EndPC uint64
	Value int32
}

func (t *PclnTab) decodePcTable(off uint32, entry uint64) []PcValue {
	if off == 0 || t.PctabOff+uint64(off) >= uint64(len(t.data)) {
		return nil
	}
	p := t.data[t.PctabOff+uint64(off):]
	val := int32(-1)
	pc := entry
	var out []PcValue
	first := true
	for len(out) < 1_000_000 {
		vdelta, n := readVarint(p)
		if n == 0 {
			break
		}
		p = p[n:]
		if vdelta == 0 && !first {
			break
		}
		first = false
		val += vdelta
		pcdelta, n := readUvarint(p)
		if n == 0 {
			break
		}
		p = p[n:]
		pc += uint64(pcdelta) * uint64(t.Quantum)
		out = append(out, PcValue{EndPC: pc, Value: val})
	}
	return out
}
