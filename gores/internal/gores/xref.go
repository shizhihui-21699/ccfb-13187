package gores

import (
	"sort"

	"golang.org/x/arch/arm64/arm64asm"
	"golang.org/x/arch/x86/x86asm"
)

// FunctionIndex locates the function containing an address.
type FunctionIndex struct {
	entries []uint64 // sorted function entry PCs
}

func NewFunctionIndex(funcs []Function) *FunctionIndex {
	ix := &FunctionIndex{entries: make([]uint64, len(funcs))}
	for i, f := range funcs {
		ix.entries[i] = f.Entry
	}
	sort.Slice(ix.entries, func(i, j int) bool { return ix.entries[i] < ix.entries[j] })
	return ix
}

// FuncAt returns the entry PC of the function containing addr.
func (ix *FunctionIndex) FuncAt(addr uint64) (uint64, bool) {
	i := sort.Search(len(ix.entries), func(i int) bool { return ix.entries[i] > addr }) - 1
	if i < 0 {
		return 0, false
	}
	return ix.entries[i], true
}

// CollectStringRefs disassembles the .text section and returns a map of
// referenced data addresses -> entry PCs of the referencing functions.
// Supports amd64, 386 and arm64.
func CollectStringRefs(b *BinFile, funcs []Function) map[uint64][]uint64 {
	refs := map[uint64][]uint64{}
	text := b.TextSection()
	if text == nil {
		return refs
	}
	data, err := b.ReadSectionData(text)
	if err != nil {
		return refs
	}
	fi := NewFunctionIndex(funcs)
	add := func(target, pc uint64) {
		if entry, ok := fi.FuncAt(pc); ok {
			lst := refs[target]
			if n := len(lst); n == 0 || lst[n-1] != entry {
				refs[target] = append(lst, entry)
			}
		}
	}
	switch b.Arch {
	case "amd64":
		xrefX86(data, text.Addr, 64, add)
	case "386":
		xrefX86(data, text.Addr, 32, add)
	case "arm64":
		xrefARM64(data, text.Addr, add)
	}
	return refs
}

// xrefX86 walks x86 instructions looking for RIP-relative (amd64) or
// absolute (386) memory operands pointing into data.
func xrefX86(code []byte, base uint64, bits int, add func(target, pc uint64)) {
	pc := base
	off := 0
	for off < len(code) {
		inst, err := x86asm.Decode(code[off:], bits)
		if err != nil {
			off++
			pc++
			continue
		}
		for _, a := range inst.Args {
			mem, ok := a.(x86asm.Mem)
			if !ok {
				continue
			}
			if bits == 64 && mem.Base == x86asm.RIP {
				add(uint64(int64(pc)+int64(inst.Len)+mem.Disp), pc)
			} else if mem.Base == 0 && mem.Index == 0 && mem.Disp > 0 {
				add(uint64(mem.Disp), pc) // absolute addressing (386)
			}
		}
		off += inst.Len
		pc += uint64(inst.Len)
	}
}

// xrefARM64 tracks ADRP/ADD pairs to recover referenced addresses.
func xrefARM64(code []byte, base uint64, add func(target, pc uint64)) {
	// adrpValue[reg] = page address most recently loaded by ADRP.
	var adrpValue [32]uint64
	var adrpValid [32]bool
	for off := 0; off+4 <= len(code); off += 4 {
		inst, err := arm64asm.Decode(code[off : off+4])
		if err != nil {
			continue
		}
		pc := base + uint64(off)
		switch inst.Op {
		case arm64asm.ADRP:
			// Args is a fixed [5]Arg array; unused slots are nil.
			if rd, ok := inst.Args[0].(arm64asm.Reg); ok {
				if rel, ok := inst.Args[1].(arm64asm.PCRel); ok {
					r := int(rd - arm64asm.X0)
					if r >= 0 && r < 32 {
						adrpValue[r] = (pc &^ 0xfff) + uint64(int64(rel))
						adrpValid[r] = true
					}
				}
			}
		case arm64asm.ADD:
			// arm64asm decodes the immediate as ImmShift with
			// unexported fields, so decode the raw word instead:
			// ADD (immediate, 64-bit): sf=1 op=0 S=0 | sh[23:22] |
			// imm12[21:10] | Rn[9:5] | Rd[4:0]
			word := uint32(code[off]) | uint32(code[off+1])<<8 |
				uint32(code[off+2])<<16 | uint32(code[off+3])<<24
			if word>>24 != 0x91 {
				continue // not ADD (immediate, 64-bit)
			}
			imm := uint64((word>>10)&0xfff) << ((word >> 22) & 0x3)
			rnIdx := int((word >> 5) & 0x1f)
			rdIdx := int(word & 0x1f)
			if adrpValid[rnIdx] {
				addr := adrpValue[rnIdx] + imm
				add(addr, pc)
				adrpValue[rdIdx] = addr
				adrpValid[rdIdx] = true
			}
		}
	}
}
