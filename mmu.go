package main

import "io"

type bankID byte

const (
	bankSystemCXROM    bankID = iota // $C100..$CFFF (Cx ROM)
	bankSystemDEFROM                 // $D000..$FFFF (DEF ROM)
	bankZeroStackRAM                 // $0000..$01FF (ZeroPage+Stack)
	bankMainRAM                      // $0200..$BFFF (Lower 48K)
	bankLangCardDX1RAM               // $D000..$DFFF (Dx Bank 1)
	bankLangCardDX2RAM               // $D000..$DFFF (Dx Bank 2)
	bankLangCardEFRAM                // $E000..$FFFF (EF RAM)
	bankDisplayPage1                 // $0400..$07FF (Text+LoRes page 1)
	bankDisplayPage2                 // $0800..$0BFF (Text+LoRes page 2)
	bankHiRes1                       // $2000..$3FFF (HiRes page 1)
	bankHiRes2                       // $4000..$5FFF (HiRes page 2)
	bankIOSwitches                   // $C000..$C0FF (IOU soft switches)
	bankSlotROM                      // $C100..$C7FF (Slot ROM)
	bankExpansionROM                 // $C800..$CFFF (Expansion ROM)

	bankIDs
)

type bankType byte

const (
	bankTypeMain bankType = iota
	bankTypeAux

	bankTypes
)

// A bank represents a switchable bank of memory.
type bank struct {
	id       bankID // bank ID
	size     uint16 // size of bank in bytes
	baseAddr uint16 // base virtual address
	mem      []byte // memory slice assigned to bank
	accessor bankAccessor
}

// A bankAccessor handles the reading and writing of bytes in a memory
// bank. This interface allows the different kinds of memory banks to
// abstract their read/write behavior in a way that is specific to the
// type of memory they represent.
type bankAccessor interface {
	LoadByte(addr uint16) byte
	StoreByte(addr uint16, v byte)
	CopyBytes(b []byte)
}

// Each memory page holds 256 bytes and can be mapped to a bank for reads
// and a bank for writes.
type page struct {
	read  *bank // memory bank used for this page's reads
	write *bank // memory bank used for this page's writes
}

// The access bit mask is used to indicate a type of memory access.
type access uint8

const (
	read access = 1 << iota
	write
)

// An mmu represents the Apple2 memory management unit. It manages multiple
// memory banks, each with different address ranges and access patterns.
type mmu struct {
	apple2 *apple2

	mainRAM   []byte // entire physical 64K main RAM address space
	auxRAM    []byte // entire physical 64K aux RAM address space
	systemROM []byte // Holds 16K of Apple II CD/EF ROMs

	banks [bankTypes][bankIDs]bank // all known memory banks
	pages [256]page                // virtual 64K address space broken into 256-byte pages
}

func newMMU(apple2 *apple2) *mmu {
	return &mmu{
		apple2: apple2,
	}
}

func (m *mmu) Init() {
	m.mainRAM = make([]byte, 64*1024)
	m.auxRAM = make([]byte, 64*1024)
	m.systemROM = make([]byte, 16*1024)

	m.addIOBank(bankIOSwitches, 0x0100, 0xc000)
	m.addIOBank(bankSlotROM, 0x0700, 0xc100)
	m.addIOBank(bankExpansionROM, 0x800, 0xc800)

	m.addROMBank(bankSystemCXROM, m.systemROM[0x0100:0x1000], 0xc100)
	m.addROMBank(bankSystemDEFROM, m.systemROM[0x1000:0x4000], 0xd000)

	m.addRAMBank(bankZeroStackRAM, bankTypeMain, m.mainRAM[0x0000:0x0200], 0x0000)
	m.addRAMBank(bankMainRAM, bankTypeMain, m.mainRAM[0x0200:0xc000], 0x0200)
	m.addRAMBank(bankDisplayPage1, bankTypeMain, m.mainRAM[0x0400:0x0800], 0x0400)
	m.addRAMBank(bankDisplayPage2, bankTypeMain, m.mainRAM[0x0800:0x0c00], 0x0800)
	m.addRAMBank(bankHiRes1, bankTypeMain, m.mainRAM[0x2000:0x4000], 0x2000)
	m.addRAMBank(bankHiRes2, bankTypeMain, m.mainRAM[0x4000:0x8000], 0x4000)
	m.addRAMBank(bankLangCardDX1RAM, bankTypeMain, m.mainRAM[0xc000:0xd000], 0xd000)
	m.addRAMBank(bankLangCardDX2RAM, bankTypeMain, m.mainRAM[0xd000:0xe000], 0xd000)
	m.addRAMBank(bankLangCardEFRAM, bankTypeMain, m.mainRAM[0xe000:], 0xe000)

	m.addRAMBank(bankZeroStackRAM, bankTypeAux, m.mainRAM[0x0000:0x0200], 0x0000)
	m.addRAMBank(bankMainRAM, bankTypeAux, m.auxRAM[0x0200:0xc000], 0x0200)
	m.addRAMBank(bankDisplayPage1, bankTypeAux, m.auxRAM[0x0400:0x0800], 0x0400)
	m.addRAMBank(bankHiRes1, bankTypeAux, m.auxRAM[0x2000:0x4000], 0x2000)
	m.addRAMBank(bankLangCardDX1RAM, bankTypeAux, m.auxRAM[0xc000:0xd000], 0xd000)
	m.addRAMBank(bankLangCardDX2RAM, bankTypeAux, m.auxRAM[0xd000:0xe000], 0xd000)
	m.addRAMBank(bankLangCardEFRAM, bankTypeAux, m.auxRAM[0xe000:], 0xe000)

	// Activate initial memory banks.
	m.ActivateBank(bankZeroStackRAM, bankTypeMain, read|write)
	m.ActivateBank(bankMainRAM, bankTypeMain, read|write)
	m.ActivateBank(bankDisplayPage1, bankTypeMain, read|write)
	m.ActivateBank(bankSystemCXROM, bankTypeMain, read)
	m.ActivateBank(bankSystemDEFROM, bankTypeMain, read)
	m.ActivateBank(bankIOSwitches, bankTypeMain, read|write)
}

// LoadSystemROM loads the system ROM memory from a reader.
func (m *mmu) LoadSystemROM(r io.Reader) error {
	_, err := io.ReadFull(r, m.systemROM)
	return err
}

// LoadByte loads a byte from the provided address.
func (m *mmu) LoadByte(addr uint16) byte {
	b := m.pages[addr>>8].read
	if b == nil {
		return 0
	}

	paddr := addr - b.baseAddr
	return b.accessor.LoadByte(paddr)
}

// LoadBytes loads a group of bytes from the provided address into the
// provided slice.
func (m *mmu) LoadBytes(addr uint16, b []byte) {
	for i, n := 0, len(b); i < n; i++ {
		b[i] = m.LoadByte(addr + uint16(i))
	}
}

// LoadAddress loads a 16-bit address from the provided address.
func (m *mmu) LoadAddress(addr uint16) uint16 {
	b := m.pages[addr>>8].read
	if b == nil {
		return 0
	}

	paddr := addr - b.baseAddr
	var lo, hi uint8
	lo = b.accessor.LoadByte(paddr)
	if (paddr & 0xff) == 0xff {
		hi = b.accessor.LoadByte(paddr - 0xff)
	} else {
		hi = b.accessor.LoadByte(paddr + 1)
	}
	return uint16(lo) | uint16(hi)<<8
}

// StoreByte stores a single byte to the provided address.
func (m *mmu) StoreByte(addr uint16, v byte) {
	b := m.pages[addr>>8].write
	if b == nil {
		return
	}

	paddr := addr - b.baseAddr
	b.accessor.StoreByte(paddr, v)
}

// StoreByte stores a group of bytes to the provided address.
func (m *mmu) StoreBytes(addr uint16, b []byte) {
	for i, n := 0, len(b); i < n; i++ {
		m.StoreByte(addr+uint16(i), b[i])
	}
}

// StoreAddress stores a 16-bit address to the memory starting at
// the provided address.
func (m *mmu) StoreAddress(addr, v uint16) {
	b := m.pages[addr>>8].write
	if b == nil {
		return
	}

	paddr := addr - b.baseAddr
	b.accessor.StoreByte(paddr, byte(v))
	if (paddr & 0xff) == 0xff {
		b.accessor.StoreByte(paddr-0xff, byte(v>>8))
	} else {
		b.accessor.StoreByte(paddr+1, byte(v>>8))
	}
}

// GetBank returns a pointer to the requested memory bank.
func (m *mmu) GetBank(id bankID, typ bankType) *bank {
	return &m.banks[typ][id]
}

// GetBankAccess returns the current access (read and/or write) allowed to the
// requested bank.
func (m *mmu) GetBankAccess(id bankID, typ bankType) access {
	b := &m.banks[typ][id]
	p := m.pages[b.baseAddr>>8]
	var a access
	if p.read == b {
		a |= read
	}
	if p.write == b {
		a |= write
	}
	return a
}

// ActivateBank activates all the pages within a bank's range of virtual
// addresses so that accesses to addresses within that range are handled
// by the bank's accessor. Read and write access may be activated
// independently.
func (m *mmu) ActivateBank(id bankID, typ bankType, access access) {
	if m.GetBankAccess(id, typ) == access {
		return
	}

	enableReads := (access & read) != 0
	enableWrites := (access & write) != 0

	b := &m.banks[typ][id]
	p0 := b.baseAddr >> 8
	pn := p0 + b.size>>8
	for p := p0; p < pn; p++ {
		page := &m.pages[p]
		if enableReads {
			page.read = b
		}
		if enableWrites {
			page.write = b
		}
	}
}

// DeactivateBank deactivates all the pages within a bank's range of virtual
// addresses so that accesses to addresses within that range are no longer
// handled by the bank. Read and write access may be deactivated independently.
func (m *mmu) DeactivateBank(id bankID, typ bankType, access access) {
	if m.GetBankAccess(id, typ) == ^access {
		return
	}

	disableReads := (access & read) != 0
	disableWrites := (access & write) != 0

	b := &m.banks[typ][id]
	p0 := b.baseAddr >> 8
	pn := p0 + b.size>>8
	for p := p0; p < pn; p++ {
		page := &m.pages[p]
		if disableReads && page.read == b {
			page.read = nil
		}
		if disableWrites && page.write == b {
			page.write = nil
		}
	}
}

// addRAMBank is a helper function that initializes a RAM memory bank and
// creates an accessor for it.
func (m *mmu) addRAMBank(id bankID, typ bankType, mem []byte, baseAddr uint16) {
	m.banks[typ][id] = bank{
		id:       id,
		size:     uint16(len(mem)),
		baseAddr: baseAddr,
		mem:      mem,
		accessor: &ramBankAccessor{mem: mem},
	}
}

// addROMBank is a helper function that initializes a ROM memory bank and
// creates an accessor for it.
func (m *mmu) addROMBank(id bankID, mem []byte, baseAddr uint16) {
	b := bank{
		id:       id,
		size:     uint16(len(mem)),
		baseAddr: baseAddr,
		mem:      mem,
		accessor: &romBankAccessor{mem: mem},
	}
	m.banks[bankTypeMain][id] = b
	m.banks[bankTypeAux][id] = b
}

// addIOBank is a helper function that initializes an IO bank and
// creates an accessor for it. IO banks do not have any system RAM or ROM
// associated with them.
func (m *mmu) addIOBank(id bankID, size, baseAddr uint16) {
	b := bank{
		id:       id,
		size:     size,
		baseAddr: baseAddr,
		mem:      nil,
	}
	m.banks[bankTypeMain][id] = b
	m.banks[bankTypeAux][id] = b
}

type ramBankAccessor struct {
	mem []byte
}

func (a *ramBankAccessor) LoadByte(addr uint16) byte {
	return a.mem[addr]
}

func (a *ramBankAccessor) StoreByte(addr uint16, v byte) {
	a.mem[addr] = v
}

func (a *ramBankAccessor) CopyBytes(b []byte) {
	copy(a.mem, b)
}

type romBankAccessor struct {
	mem []byte
}

func (a *romBankAccessor) LoadByte(addr uint16) byte {
	return a.mem[addr]
}

func (a *romBankAccessor) StoreByte(addr uint16, v byte) {
	// Do nothing
}

func (a *romBankAccessor) CopyBytes(b []byte) {
	copy(a.mem, b)
}
