package main

import (
	"errors"
)

// MMU errors
var (
	errMemoryFault = errors.New("Memory fault")
)

// A memoryBank represents a region of memory managed by the MMU. This
// interface is used for every type of MMU-managed memory, including RAM,
// ROM, display memory, and I/O switches.
type memoryBank interface {
	ID() uint8
	AddressRange() addrRange
	LoadByte(addr uint16) byte
	LoadAddress(addr uint16) uint16
	StoreByte(addr uint16, v byte)
	StoreAddress(addr, v uint16)
}

// Bank identifiers
const (
	bankIDSystemROM         uint8 = iota // $C000..$FFFF
	bankIDMainRAM                        // $0000..$BFFF
	bankIDMainRAMDBank1                  // $D000..$DFFF
	bankIDMainRAMDBank2                  // $D000..$DFFF
	bankIDMainRAMEF                      // $E000..$FFFF
	bankIDMainDisplayText1               // $0400..$07FF
	bankIDMainDisplayText2               // $0800..$0BFF
	bankIDMainDisplayHiRes1              // $2000..$3FFF
	bankIDMainDisplayHiRes2              // $4000..$5FFF
	bankIDAuxRAM                         // $0000..$BFFF
	bankIDAuxRAMDBank1                   // $D000..$DFFF
	bankIDAuxRAMDBank2                   // $D000..$DFFF
	bankIDAuxRAMEF                       // $E000..$FFFF
	bankIDAuxDisplayText1                // $0400..$07FF
	bankIDAuxDisplayHiRes1               // $2000..$3FFF
	bankIDIOSwitches                     // $C000..$C0FF
	bankIDSlotROM                        // $C100..$C7FF
	bankIDExpansionROM                   // $C800..$CFFF

	bankCount
)

// addrRange represents an address range.
type addrRange struct {
	start uint16
	end   uint16
}

// The access bit mask is used to indicate memory access: read and/or write.
type access uint8

const (
	read access = 1 << iota
	write
)

// A page is a 256-byte chunk of memory.
type page struct {
	read  memoryBank // memory bank used for this page's reads
	write memoryBank // memory bank used for this page's writes
}

// An MMU represents the Apple2 memory management unit. It manages multiple
// memory banks, each with different address ranges and access patterns.
type MMU struct {
	mainRAM       []byte // entire physical 64K main RAM address space
	auxRAM        []byte // entire physical 64K aux RAM address space
	systemROM     []byte // Holds 16K of Apple II CD/EF ROMs
	peripheralROM []byte // Holds 4K peripheral ROM

	banks [bankCount]memoryBank // memory banks
	pages [256]page             // virtual 64K address space broken into 256-byte pages
}

// NewMMU creates a new Apple2 memory management unit.
func NewMMU() *MMU {
	mainRAM := make([]byte, 64*1024)
	auxRAM := make([]byte, 64*1024)
	systemROM := make([]byte, 16*1024)
	peripheralROM := make([]byte, 4*1024)

	m := &MMU{
		mainRAM:       mainRAM,
		auxRAM:        auxRAM,
		systemROM:     systemROM,
		peripheralROM: peripheralROM,
	}

	// Create all Apple II memory banks.
	m.addBank(newROM(bankIDSystemROM, 0xc000, 0x0000, 0x4000, systemROM))
	m.addBank(newRAM(bankIDMainRAM, 0x0000, 0x0000, 0xc000, mainRAM))
	m.addBank(newRAM(bankIDMainRAMDBank1, 0xd000, 0xc800, 0x0800, mainRAM))
	m.addBank(newRAM(bankIDMainRAMDBank2, 0xd000, 0xd000, 0x0800, mainRAM))
	m.addBank(newRAM(bankIDMainRAMEF, 0xe000, 0xe000, 0x2000, mainRAM))
	m.addBank(newDisplayMemory(bankIDMainDisplayText1, 0x0400, 0x0400, 0x0400, mainRAM))
	m.addBank(newDisplayMemory(bankIDMainDisplayText2, 0x0800, 0x0800, 0x0400, mainRAM))
	m.addBank(newDisplayMemory(bankIDMainDisplayHiRes1, 0x2000, 0x2000, 0x2000, mainRAM))
	m.addBank(newDisplayMemory(bankIDMainDisplayHiRes2, 0x4000, 0x2000, 0x2000, mainRAM))
	m.addBank(newRAM(bankIDAuxRAM, 0x0000, 0x0000, 0xc000, auxRAM))
	m.addBank(newRAM(bankIDAuxRAMDBank1, 0xd000, 0xc800, 0x0800, auxRAM))
	m.addBank(newRAM(bankIDAuxRAMDBank2, 0xd000, 0xd000, 0x0800, auxRAM))
	m.addBank(newRAM(bankIDAuxRAMEF, 0xe000, 0xe000, 0x2000, auxRAM))
	m.addBank(newDisplayMemory(bankIDAuxDisplayText1, 0x0400, 0x0400, 0x0400, auxRAM))
	m.addBank(newDisplayMemory(bankIDAuxDisplayHiRes1, 0x2000, 0x2000, 0x2000, auxRAM))
	m.addBank(newROM(bankIDSlotROM, 0xc100, 0x0100, 0x0700, peripheralROM))
	m.addBank(newROM(bankIDExpansionROM, 0xc800, 0x0800, 0x0800, peripheralROM))
	m.addBank(newIO(bankIDIOSwitches, 0xc000, 0x0000))
	m.addBank(newIO(bankIDSlotROM, 0xc100, 0x0700))
	m.addBank(newIO(bankIDExpansionROM, 0xc800, 0x0800))

	// Activate initial memory banks.
	m.ActivateBank(bankIDMainRAM, addrRange{}, read|write)
	m.ActivateBank(bankIDSystemROM, addrRange{}, read|write)
	m.ActivateBank(bankIDMainDisplayText1, addrRange{}, read|write)
	m.ActivateBank(bankIDIOSwitches, addrRange{}, read|write)

	return m
}

// LoadByte loads a single byte from the address and returns it.
func (m *MMU) LoadByte(addr uint16) (byte, error) {
	b := m.pages[addr>>8].read
	if b == nil {
		return 0, errMemoryFault
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
		return 0, errMemoryFault
	}
	return b.LoadAddress(addr), nil
}

// StoreByte stores a byte to the requested address.
func (m *MMU) StoreByte(addr uint16, v byte) error {
	b := m.pages[addr>>8].write
	if b == nil {
		return errMemoryFault
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
func (m *MMU) StoreAddress(addr, v uint16) error {
	b := m.pages[addr>>8].write
	if b == nil {
		return errMemoryFault
	}
	b.StoreAddress(addr, v)
	return nil
}

// Add a memory bank to the manager.
func (m *MMU) addBank(b memoryBank) {
	m.banks[b.ID()] = b
}

// ActivateBank activates a range of virtual addresses within a memory bank so
// that all accesses to addresses within that range are handled by the bank.
// Read and write access may be configured independently.
func (m *MMU) ActivateBank(bankID uint8, r addrRange, access access) {
	b := m.banks[bankID]
	if r.start == 0 && r.end == 0 {
		r = b.AddressRange()
	}

	enableReads := (access & read) != 0
	enableWrites := (access & write) != 0
	for i, j := r.start>>8, r.end>>8; i < j; i++ {
		if enableReads {
			m.pages[i].read = b
		}
		if enableWrites {
			m.pages[i].write = b
		}
	}
}

// DeactivateBank deactivates a range of virtual addresses within a memory
// bank so that the bank no longer handles accesses to address within that
// range. Read and write access may be configured independently.
func (m *MMU) DeactivateBank(bankID uint8, r addrRange, access access) {
	b := m.banks[bankID]
	if r.start == 0 && r.end == 0 {
		r = b.AddressRange()
	}

	disableReads := (access & read) != 0
	disableWrites := (access & write) != 0
	for i, j := r.start>>8, r.end>>8; i < j; i++ {
		if disableReads {
			m.pages[i].read = b
		}
		if disableWrites {
			m.pages[i].write = b
		}
	}
}

//
// RAM bank
//

// ram represents a bank of random-access memory that can be read and written.
type ram struct {
	id    uint8
	size  uint16 // size of RAM bank in bytes
	vbase uint16 // virtual base address of bank
	pbase uint16 // physical base address of bank
	pmem  []byte // entire physical memory
}

// newRAM creates a new RAM memory bank of the requested size. Required
// parameters include the base virtual address of the memory bank, the
// base physical address corresponding to the virtual address, the size
// of the bank in bytes, and a reference to the entire physical memory.
func newRAM(id uint8, vbase, pbase, size uint16, pmem []byte) *ram {
	prange := addrRange{start: pbase, end: pbase + size}
	if int(prange.end) > len(pmem) {
		panic("RAM address exceeds 64K")
	}
	if (prange.end-prange.start)&0xff != 0 {
		panic("RAM size must be a multiple of the 256-byte page size")
	}

	return &ram{
		id:    id,
		size:  size,
		vbase: vbase,
		pbase: pbase,
		pmem:  pmem,
	}
}

// ID returns the RAM memory bank's bank id.
func (r *ram) ID() uint8 {
	return r.id
}

// AddressRange returns the range of virtual addresses covered by the bank.
func (r *ram) AddressRange() addrRange {
	return addrRange{start: r.vbase, end: r.vbase + r.size}
}

// LoadByte returns the value of a byte of memory at the requested address.
func (r *ram) LoadByte(addr uint16) byte {
	paddr := (addr - r.vbase) + r.pbase
	return r.pmem[paddr]
}

// LoadAddress loads a 16-bit address from the requested memory address.
func (r *ram) LoadAddress(addr uint16) uint16 {
	paddr := (addr - r.vbase) + r.pbase
	if (paddr & 0xff) == 0xff {
		return uint16(r.pmem[paddr]) | uint16(r.pmem[paddr-0xff])<<8
	}
	return uint16(r.pmem[paddr]) | uint16(r.pmem[paddr+1])<<8
}

// StoreByte stores a byte value at the requested address.
func (r *ram) StoreByte(addr uint16, b byte) {
	paddr := (addr - r.vbase) + r.pbase
	r.pmem[paddr] = b
}

// StoreAddress stores a 16-bit address 'v' at the requested address.
func (r *ram) StoreAddress(addr, v uint16) {
	paddr := (addr - r.vbase) + r.pbase
	r.pmem[paddr] = byte(v)
	if (paddr & 0xff) == 0xff {
		r.pmem[paddr-0xff] = byte(v >> 8)
	} else {
		r.pmem[paddr+1] = byte(v >> 8)
	}
}

//
// ROM bank
//

// rom represents a bank of read-only memory.
type rom struct {
	id    uint8
	size  uint16
	vbase uint16
	pbase uint16
	pmem  []byte
}

// newROM creates a new ROM memory bank of the requested size. Required
// parameters include the base virtual address of the memory bank, the
// base physical address corresponding to the virtual address, the size
// of the bank in bytes, and a reference to the entire physical memory.
func newROM(id uint8, vbase, pbase, size uint16, pmem []byte) *rom {
	prange := addrRange{start: pbase, end: pbase + size}
	if int(prange.end) > len(pmem) {
		panic("ROM address space exceeds 64K")
	}
	if (prange.end-prange.start)&0xff != 0 {
		panic("ROM size must be a multiple of the 256-byte page size")
	}

	r := &rom{
		id:    id,
		size:  size,
		vbase: vbase,
		pbase: pbase,
		pmem:  pmem,
	}
	return r
}

// ID returns the ROM memory bank's bank id.
func (r *rom) ID() uint8 {
	return r.id
}

// AddressRange returns the virtual address range covered by the ROM bank.
func (r *rom) AddressRange() addrRange {
	return addrRange{start: r.vbase, end: r.vbase + r.size}
}

// LoadByte returns the value of a byte of memory at the requested address.
func (r *rom) LoadByte(addr uint16) byte {
	paddr := (addr - r.vbase) + r.pbase
	return r.pmem[paddr]
}

// LoadAddress loads a 16-bit address from the requested memory address.
func (r *rom) LoadAddress(addr uint16) uint16 {
	paddr := (addr - r.vbase) + r.pbase
	if (paddr & 0xff) == 0xff {
		return uint16(r.pmem[paddr]) | uint16(r.pmem[paddr-0xff])<<8
	}
	return uint16(r.pmem[paddr]) | uint16(r.pmem[paddr+1])<<8
}

// StoreByte does nothing for ROM.
func (r *rom) StoreByte(addr uint16, b byte) {
	// Do nothing
}

// StoreAddress does nothing for ROM.
func (r *rom) StoreAddress(addr, v uint16) {
}

//
// display memory bank (placeholder implementation)
//

// displayMemory represents a bank of random-access memory that can be read and written.
type displayMemory struct {
	id    uint8
	size  uint16
	vbase uint16
	pbase uint16
	pmem  []byte
}

func newDisplayMemory(id uint8, vbase, pbase, size uint16, pmem []byte) *displayMemory {
	prange := addrRange{start: pbase, end: pbase + size}
	if int(prange.end) > len(pmem) {
		panic("RAM address exceeds 64K")
	}
	if (prange.end-prange.start)&0xff != 0 {
		panic("RAM size must be a multiple of the 256-byte page size")
	}

	return &displayMemory{
		id:    id,
		size:  size,
		vbase: vbase,
		pbase: pbase,
		pmem:  pmem,
	}
}

func (m *displayMemory) ID() uint8 {
	return m.id
}

func (m *displayMemory) AddressRange() addrRange {
	return addrRange{start: m.vbase, end: m.vbase + m.size}
}

func (m *displayMemory) LoadByte(addr uint16) byte {
	paddr := (addr - m.vbase) + m.pbase
	return m.pmem[paddr]
}

func (m *displayMemory) LoadAddress(addr uint16) uint16 {
	paddr := (addr - m.vbase) + m.pbase
	if (paddr & 0xff) == 0xff {
		return uint16(m.pmem[paddr]) | uint16(m.pmem[paddr-0xff])<<8
	}
	return uint16(m.pmem[paddr]) | uint16(m.pmem[paddr+1])<<8
}

func (m *displayMemory) StoreByte(addr uint16, b byte) {
	paddr := (addr - m.vbase) + m.pbase
	m.pmem[paddr] = b
}

func (m *displayMemory) StoreAddress(addr, v uint16) {
	paddr := (addr - m.vbase) + m.pbase
	m.pmem[paddr] = byte(v)
	if (paddr & 0xff) == 0xff {
		m.pmem[paddr-0xff] = byte(v >> 8)
	} else {
		m.pmem[paddr+1] = byte(v >> 8)
	}
}

//
// IO bank (placeholder)
//

type io struct {
	id    uint8
	size  uint16
	vbase uint16
}

func newIO(id uint8, vbase, size uint16) *io {
	return &io{
		id:    id,
		size:  size,
		vbase: vbase,
	}
}

// ID returns the IO bank's bank id.
func (io *io) ID() uint8 {
	return io.id
}

// AddressRange returns the virtual address range covered by the IO bank.
func (io *io) AddressRange() addrRange {
	return addrRange{start: io.vbase, end: io.vbase + io.size}
}

// LoadByte returns the value of a byte of memory at the requested address.
func (io *io) LoadByte(addr uint16) byte {
	// TODO: Write me
	return 0
}

// LoadAddress loads a 16-bit address from the requested memory address.
func (io *io) LoadAddress(addr uint16) uint16 {
	// TODO: Write me
	return 0
}

// StoreByte stores a byte to the requested memory address.
func (io *io) StoreByte(addr uint16, b byte) {
	// TODO: Write me
}

// StoreAddress does nothing for ROM.
func (io *io) StoreAddress(addr, v uint16) {
	// TODO: Write me
}
