package main

import (
	"errors"

	"github.com/beevik/go6502"
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
	AddressRange() (start go6502.Address, end go6502.Address)
	LoadByte(addr go6502.Address) byte
	LoadAddress(addr go6502.Address) go6502.Address
	StoreByte(addr go6502.Address, v byte)
	StoreAddress(addr go6502.Address, v go6502.Address)
}

// The access bit mask is used to indicate memory access: read and/or write.
type access int

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
	banks map[MemoryBank]access
	pages [256]page
}

// NewMMU creates a new Apple2 memory management unit.
func NewMMU() *MMU {
	return &MMU{
		banks: make(map[MemoryBank]access),
	}
}

// LoadByte loads a single byte from the address and returns it.
func (m *MMU) LoadByte(addr go6502.Address) (byte, error) {
	b := m.pages[addr>>8].read
	if b == nil {
		return 0, ErrMemoryOutOfBounds
	}
	return b.LoadByte(addr), nil
}

// LoadBytes loads multiple bytes from the address and stores them into
// the buffer 'b'.
func (m *MMU) LoadBytes(addr go6502.Address, b []byte) error {
	var err error
	for i, n := 0, len(b); i < n; i++ {
		b[i], err = m.LoadByte(addr + go6502.Address(i))
		if err != nil {
			return err
		}
	}
	return nil
}

// LoadAddress loads a 16-bit address value from the requested address and
// returns it.
func (m *MMU) LoadAddress(addr go6502.Address) (go6502.Address, error) {
	b := m.pages[addr>>8].read
	if b == nil {
		return 0, ErrMemoryOutOfBounds
	}
	return b.LoadAddress(addr), nil
}

// StoreByte stores a byte to the requested address.
func (m *MMU) StoreByte(addr go6502.Address, v byte) error {
	b := m.pages[addr>>8].write
	if b == nil {
		return ErrMemoryOutOfBounds
	}
	b.StoreByte(addr, v)
	return nil
}

// StoreBytes stores multiple bytes to the requested address.
func (m *MMU) StoreBytes(addr go6502.Address, b []byte) error {
	var err error
	for i, n := 0, len(b); i < n; i++ {
		err = m.StoreByte(addr+go6502.Address(i), b[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// StoreAddress stores a 16-bit address 'v' to the requested address.
func (m *MMU) StoreAddress(addr go6502.Address, v go6502.Address) error {
	b := m.pages[addr>>8].write
	if b == nil {
		return ErrMemoryOutOfBounds
	}
	b.StoreAddress(addr, v)
	return nil
}

// AddBank adds a memory bank to be managed by the MMU. The
// bank starts inactive for reads and writes.
func (m *MMU) AddBank(b MemoryBank) {
	m.banks[b] = 0
}

// RemoveBank removes a memory bank from being managed by the MMU.
// If it was active for reads or writes, it is deactivated first.
func (m *MMU) RemoveBank(b MemoryBank) {
	active, ok := m.banks[b]
	if !ok {
		return
	}

	if active != 0 {
		m.DeactivateBank(b, active)
	}
	delete(m.banks, b)
}

// ActivateBank activates a memory bank in the MMU so that it handles all
// accesses to its addresses. Read and write access may be configured
// independently.
func (m *MMU) ActivateBank(b MemoryBank, access access) {
	active, ok := m.banks[b]
	if !ok {
		return
	}

	enableReads := (access&read) != 0 && (active&read) == 0
	enableWrites := (access&write) != 0 && (active&write) == 0
	if !enableReads && !enableWrites {
		return
	}

	m.banks[b] = m.banks[b] | access

	start, end := b.AddressRange()
	for i, j := start>>8, end>>8; i < j; i++ {
		if enableReads {
			m.pages[i].read = b
		}
		if enableWrites {
			m.pages[i].write = b
		}
	}
}

// DeactivateBank deactivates a memory bank in the MMU so that it no longer
// handles accesses to its addresses. Read and write access may be configured
// independently.
func (m *MMU) DeactivateBank(b MemoryBank, access access) {
	active, ok := m.banks[b]
	if !ok {
		return
	}

	disableReads := (access&read) != 0 && (active&read) != 0
	disableWrites := (access&write) != 0 && (active&write) != 0
	if !disableReads && !disableWrites {
		return
	}

	m.banks[b] = m.banks[b] &^ access

	start, end := b.AddressRange()
	for i, j := start>>8, end>>8; i < j; i++ {
		if disableReads {
			m.pages[i].read = nil
		}
		if disableWrites {
			m.pages[i].write = nil
		}
	}
}

// RAM represents a random-access memory bank that can be read and written.
type RAM struct {
	start go6502.Address
	end   go6502.Address
	buf   []byte
}

// NewRAM creates a new RAM memory bank of the requested size. Its
// contents are initialized to zeroes.
func NewRAM(addr go6502.Address, size int) *RAM {
	if int(addr)+size > 0x10000 {
		panic("RAM address exceeds 64K")
	}
	if size&0xff != 0 {
		panic("RAM size must be a multiple of the 256-byte page size")
	}
	return &RAM{
		start: addr,
		end:   addr + go6502.Address(size),
		buf:   make([]byte, size),
	}
}

// AddressRange returns the range of addresses in the RAM bank.
func (r *RAM) AddressRange() (start go6502.Address, end go6502.Address) {
	return r.start, r.end
}

// LoadByte returns the value of a byte of memory at the requested address.
func (r *RAM) LoadByte(addr go6502.Address) byte {
	return r.buf[addr-r.start]
}

// LoadAddress loads a 16-bit address from the requested memory address.
func (r *RAM) LoadAddress(addr go6502.Address) go6502.Address {
	i := int(addr - r.start)
	if (i & 0xff) == 0xff {
		return go6502.Address(r.buf[i]) | go6502.Address(r.buf[i-0xff])<<8
	}
	return go6502.Address(r.buf[i]) | go6502.Address(r.buf[i+1])<<8
}

// StoreByte stores a byte value at the requested address.
func (r *RAM) StoreByte(addr go6502.Address, b byte) {
	r.buf[addr] = b
}

// ROM represents a bank of read-only memory.
type ROM struct {
	start go6502.Address
	end   go6502.Address
	buf   []byte
}

// NewROM creates a new ROM memory bank initialized with the contents of the
// provided buffer.
func NewROM(addr go6502.Address, b []byte) *ROM {
	if int(addr)+len(b) > 0x10000 {
		panic("ROM address space exceeds 64K")
	}
	if len(b)&0xff != 0 {
		panic("ROM size must be a multiple of the 256-byte page size")
	}
	rom := &ROM{
		start: addr,
		end:   addr + go6502.Address(len(b)),
		buf:   make([]byte, len(b)),
	}
	copy(rom.buf, b)
	return rom
}

// AddressRange returns the range of addresses in the ROM bank.
func (r *ROM) AddressRange() (start go6502.Address, end go6502.Address) {
	return r.start, r.end
}

// LoadByte returns the value of a byte of memory at the requested address.
func (r *ROM) LoadByte(addr go6502.Address) byte {
	return r.buf[addr-r.start]
}

// LoadAddress loads a 16-bit address from the requested memory address.
func (r *ROM) LoadAddress(addr go6502.Address) go6502.Address {
	i := int(addr - r.start)
	if (i & 0xff) == 0xff {
		return go6502.Address(r.buf[i]) | go6502.Address(r.buf[i-0xff])<<8
	}
	return go6502.Address(r.buf[i]) | go6502.Address(r.buf[i+1])<<8
}

// StoreByte does nothing for ROM.
func (r *ROM) StoreByte(addr go6502.Address, b byte) {
	// Do nothing
}
