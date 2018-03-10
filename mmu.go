package main

import (
	"errors"
)

// MMU errors
var (
	ErrMemoryOutOfBounds = errors.New("Memory out of bounds")
	ErrMemoryReadOnly    = errors.New("Memory is read-only")
)

// A MemoryBank represents a region of memory managed by the MMU. This
// interface is used for every type of MMU-managed memory, including RAM,
// ROM, IO and peripheral buffers.
type MemoryBank interface {
	AddressRange() Range
	LoadByte(addr uint16) byte
	LoadAddress(addr uint16) uint16
	StoreByte(addr uint16, v byte)
	StoreAddress(addr uint16, v uint16)
}

// Range represents an address range.
type Range struct {
	Start uint16
	End   uint16
}

// The access bit mask is used to indicate memory access: read and/or write.
type access uint8

const (
	read access = 1 << iota
	write
)

// A page is a 256-byte chunk of memory.
type page struct {
	read  MemoryBank // memory bank used for this page's reads
	write MemoryBank // memory bank used for this page's writes
}

// An MMU represents the Apple2 memory management unit. It manages multiple
// memory banks, each with different address ranges and access patterns.
type MMU struct {
	mainRAM   []byte // entire 64K main RAM address space
	auxRAM    []byte // entire 64K aux RAM address space
	systemROM []byte // entire 16K system ROM address space

	pages [256]page // 256-byte pages covering 64K address space
}

// NewMMU creates a new Apple2 memory management unit.
func NewMMU() *MMU {
	mainRAM := make([]byte, 64*1024)
	auxRAM := make([]byte, 64*1024)
	systemROM := make([]byte, 16*1024)

	// TODO: Create memory banks
	// TODO: Activate initial memory banks

	return &MMU{
		mainRAM:   mainRAM,
		auxRAM:    auxRAM,
		systemROM: systemROM,
	}
}

// LoadByte loads a single byte from the address and returns it.
func (m *MMU) LoadByte(addr uint16) (byte, error) {
	b := m.pages[addr>>8].read
	if b == nil {
		return 0, ErrMemoryOutOfBounds
	}
	return b.LoadByte(addr), nil
}

// LoadBytes loads multiple bytes from the address and stores them into
// the buffer 'b'.
func (m *MMU) LoadBytes(addr uint16, b []byte) error {
	var err error
	for i, n := 0, len(b); i < n; i++ {
		b[i], err = m.LoadByte(addr + uint16(i))
		if err != nil {
			return err
		}
	}
	return nil
}

// LoadAddress loads a 16-bit address value from the requested address and
// returns it.
func (m *MMU) LoadAddress(addr uint16) (uint16, error) {
	b := m.pages[addr>>8].read
	if b == nil {
		return 0, ErrMemoryOutOfBounds
	}
	return b.LoadAddress(addr), nil
}

// StoreByte stores a byte to the requested address.
func (m *MMU) StoreByte(addr uint16, v byte) error {
	b := m.pages[addr>>8].write
	if b == nil {
		return ErrMemoryOutOfBounds
	}
	b.StoreByte(addr, v)
	return nil
}

// StoreBytes stores multiple bytes to the requested address.
func (m *MMU) StoreBytes(addr uint16, b []byte) error {
	var err error
	for i, n := 0, len(b); i < n; i++ {
		err = m.StoreByte(addr+uint16(i), b[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// StoreAddress stores a 16-bit address 'v' to the requested address.
func (m *MMU) StoreAddress(addr uint16, v uint16) error {
	b := m.pages[addr>>8].write
	if b == nil {
		return ErrMemoryOutOfBounds
	}
	b.StoreAddress(addr, v)
	return nil
}

// ActivateBank activates a range of addresses within a memory bank so
// that all accesses to addresses within that range are handled by the bank.
// Read and write access may be configured independently.
func (m *MMU) ActivateBank(b MemoryBank, r Range, access access) {
	enableReads := (access & read) != 0
	enableWrites := (access & write) != 0
	for i, j := r.Start>>8, r.End>>8; i < j; i++ {
		if enableReads {
			m.pages[i].read = b
		}
		if enableWrites {
			m.pages[i].write = b
		}
	}
}

// DeactivateBank deactivates a range of addresses within a memory bank so
// that the bank no longer handles accesses to address within that range.
// Read and write access may be configured independently.
func (m *MMU) DeactivateBank(b MemoryBank, r Range, access access) {
	disableReads := (access & read) != 0
	disableWrites := (access & write) != 0
	for i, j := r.Start>>8, r.End>>8; i < j; i++ {
		if disableReads {
			m.pages[i].read = nil
		}
		if disableWrites {
			m.pages[i].write = nil
		}
	}
}

// RAM represents a bank of random-access memory that can be read and written.
type RAM struct {
	r   Range
	buf []byte // entire memory
}

// NewRAM creates a new RAM memory bank of the requested size.
func NewRAM(r Range, buf []byte) *RAM {
	if r.End > uint16(len(buf)) {
		panic("RAM address exceeds 64K")
	}
	if r.Start > r.End {
		panic("Invalid address range")
	}
	if (r.End-r.Start)&0xff != 0 {
		panic("RAM size must be a multiple of the 256-byte page size")
	}
	return &RAM{
		r:   r,
		buf: buf,
	}
}

// AddressRange returns the range of addresses in the RAM bank.
func (r *RAM) AddressRange() Range {
	return r.r
}

// LoadByte returns the value of a byte of memory at the requested address.
func (r *RAM) LoadByte(addr uint16) byte {
	return r.buf[addr]
}

// LoadAddress loads a 16-bit address from the requested memory address.
func (r *RAM) LoadAddress(addr uint16) uint16 {
	if (addr & 0xff) == 0xff {
		return uint16(r.buf[addr]) | uint16(r.buf[addr-0xff])<<8
	}
	return uint16(r.buf[addr]) | uint16(r.buf[addr+1])<<8
}

// StoreByte stores a byte value at the requested address.
func (r *RAM) StoreByte(addr uint16, b byte) {
	r.buf[addr] = b
}

// ROM represents a bank of read-only memory.
type ROM struct {
	r    Range
	base uint16
	buf  []byte
}

// NewROM creates a new ROM memory bank within the provided memory buffer.
// The address is the start address of the bank's memory range, the size is
// the number of bytes in the bank, the buffer contains the entire memory
// space for which the bank is a window, and the base address is the memory
// address of the buffer's first byte.
func NewROM(r Range, buf []byte, baseAddr uint16) *ROM {
	if r.End > uint16(len(buf)) {
		panic("ROM address space exceeds 64K")
	}
	if r.Start > r.End {
		panic("Invalid address range")
	}
	if (r.End-r.Start)&0xff != 0 {
		panic("ROM size must be a multiple of the 256-byte page size")
	}
	rom := &ROM{
		r:    r,
		base: baseAddr,
		buf:  buf,
	}
	return rom
}

// AddressRange returns the range of addresses in the ROM bank.
func (r *ROM) AddressRange() Range {
	return r.r
}

// LoadByte returns the value of a byte of memory at the requested address.
func (r *ROM) LoadByte(addr uint16) byte {
	return r.buf[addr-r.base]
}

// LoadAddress loads a 16-bit address from the requested memory address.
func (r *ROM) LoadAddress(addr uint16) uint16 {
	addr -= r.base
	if (addr & 0xff) == 0xff {
		return uint16(r.buf[addr]) | uint16(r.buf[addr-0xff])<<8
	}
	return uint16(r.buf[addr]) | uint16(r.buf[addr+1])<<8
}

// StoreByte does nothing for ROM.
func (r *ROM) StoreByte(addr uint16, b byte) {
	// Do nothing
}
