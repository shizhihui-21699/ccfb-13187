// Package gores recovers Go runtime metadata (function names, source
// files, line numbers, build info, string references) from stripped Go
// binaries. It supports ELF, PE and Mach-O containers on x86, x86-64,
// ARM and ARM64.
package gores

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"errors"
	"fmt"
	"io"
)

// Section is a unified view of a binary section.
type Section struct {
	Name   string
	Addr   uint64 // virtual address
	Size   uint64 // virtual size
	Offset uint64 // file offset
	Raw    []byte // lazily filled by OpenFile for sections we need
}

// BinFile abstracts ELF / PE / Mach-O.
type BinFile struct {
	Format    string // "elf" | "pe" | "macho"
	Arch      string // "amd64" | "386" | "arm64" | "arm" | ...
	Sections  []*Section
	ImageBase uint64
	ReaderAt  io.ReaderAt

	elf   *elf.File
	pe    *pe.File
	macho *macho.File
}

// Open detects the container format and returns a BinFile.
func Open(r io.ReaderAt) (*BinFile, error) {
	if f, err := elf.NewFile(r); err == nil {
		return fromELF(f, r)
	}
	if f, err := pe.NewFile(r); err == nil {
		return fromPE(f, r)
	}
	if f, err := macho.NewFile(r); err == nil {
		return fromMachO(f, r)
	}
	return nil, errors.New("unrecognized binary format (not ELF/PE/Mach-O)")
}

func fromELF(f *elf.File, r io.ReaderAt) (*BinFile, error) {
	b := &BinFile{Format: "elf", ReaderAt: r, elf: f}
	switch f.Machine {
	case elf.EM_X86_64:
		b.Arch = "amd64"
	case elf.EM_386:
		b.Arch = "386"
	case elf.EM_AARCH64:
		b.Arch = "arm64"
	case elf.EM_ARM:
		b.Arch = "arm"
	case elf.EM_PPC64:
		b.Arch = "ppc64"
	case elf.EM_RISCV:
		b.Arch = "riscv64"
	default:
		b.Arch = f.Machine.String()
	}
	for _, s := range f.Sections {
		b.Sections = append(b.Sections, &Section{
			Name: s.Name, Addr: s.Addr, Size: s.Size, Offset: s.Offset,
		})
	}
	return b, nil
}

func fromPE(f *pe.File, r io.ReaderAt) (*BinFile, error) {
	b := &BinFile{Format: "pe", ReaderAt: r, pe: f}
	switch f.Machine {
	case pe.IMAGE_FILE_MACHINE_AMD64:
		b.Arch = "amd64"
	case pe.IMAGE_FILE_MACHINE_I386:
		b.Arch = "386"
	case pe.IMAGE_FILE_MACHINE_ARM64:
		b.Arch = "arm64"
	case pe.IMAGE_FILE_MACHINE_ARMNT:
		b.Arch = "arm"
	default:
		b.Arch = fmt.Sprintf("machine-0x%x", f.Machine)
	}
	switch oh := f.OptionalHeader.(type) {
	case *pe.OptionalHeader64:
		b.ImageBase = uint64(oh.ImageBase)
	case *pe.OptionalHeader32:
		b.ImageBase = uint64(oh.ImageBase)
	}
	for _, s := range f.Sections {
		b.Sections = append(b.Sections, &Section{
			Name:   s.Name,
			Addr:   b.ImageBase + uint64(s.VirtualAddress),
			Size:   uint64(s.VirtualSize),
			Offset: uint64(s.Offset),
		})
	}
	return b, nil
}

func fromMachO(f *macho.File, r io.ReaderAt) (*BinFile, error) {
	b := &BinFile{Format: "macho", ReaderAt: r, macho: f}
	switch f.Cpu {
	case macho.CpuAmd64:
		b.Arch = "amd64"
	case macho.CpuArm64:
		b.Arch = "arm64"
	case macho.CpuArm:
		b.Arch = "arm"
	default:
		b.Arch = f.Cpu.String()
	}
	for _, s := range f.Sections {
		b.Sections = append(b.Sections, &Section{
			Name: s.Name, Addr: s.Addr, Size: s.Size, Offset: uint64(s.Offset),
		})
	}
	return b, nil
}

// Section returns the first section with the given name ("" matches none).
func (b *BinFile) Section(name string) *Section {
	for _, s := range b.Sections {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// ReadSectionData loads a section's raw bytes (cached in Section.Raw).
func (b *BinFile) ReadSectionData(s *Section) ([]byte, error) {
	if s.Raw != nil {
		return s.Raw, nil
	}
	if s.Size == 0 {
		return nil, nil
	}
	data := make([]byte, s.Size)
	n, err := b.ReaderAt.ReadAt(data, int64(s.Offset))
	if err != nil && err != io.EOF {
		return nil, err
	}
	s.Raw = data[:n]
	return s.Raw, nil
}

// VAtoOffset converts a virtual address to a file offset.
func (b *BinFile) VAtoOffset(va uint64) (uint64, bool) {
	for _, s := range b.Sections {
		if va >= s.Addr && va < s.Addr+s.Size {
			return s.Offset + (va - s.Addr), true
		}
	}
	return 0, false
}

// ReadAtVA reads len(buf) bytes at virtual address va.
func (b *BinFile) ReadAtVA(buf []byte, va uint64) error {
	off, ok := b.VAtoOffset(va)
	if !ok {
		return fmt.Errorf("va 0x%x not mapped", va)
	}
	_, err := b.ReaderAt.ReadAt(buf, int64(off))
	return err
}

// TextSection returns the executable section (".text" / "__text").
func (b *BinFile) TextSection() *Section {
	for _, n := range []string{".text", "__text"} {
		if s := b.Section(n); s != nil {
			return s
		}
	}
	// fallback: first section containing "text"
	for _, s := range b.Sections {
		if s.Name != "" && (s.Name == ".text" || s.Name == "__text") {
			return s
		}
	}
	return nil
}
