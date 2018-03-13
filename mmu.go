package main

// A bank represents a switchable bank of memory.
type bank struct {
	id       uint8  // bank ID
	size     uint16 // size of bank in bytes
	baseAddr uint16 // base virtual address
	accessor bankAccessor
}

// A bankAccessor handles the reading and writing of bytes in a memory
// bank. This interface allows the different kinds of memory banks to
// abstract their read/write behavior in a way that is specific to the
// type of memory they represent.
type bankAccessor interface {
	LoadByte(addr uint16) byte
	StoreByte(addr uint16, v byte)
}

// Memory bank identifiers
const (
	bankIDSystemCXROM      uint8 = iota // $C100..$CFFF (Cx ROM)
	bankIDSystemDEFROM                  // $D000..$FFFF (DEF ROM)
	bankIDMainZPS                       // $0000..$01FF (ZeroPage+Stack)
	bankIDMainRAM                       // $0200..$BFFF (Lower 48K)
	bankIDMainDX1RAM                    // $D000..$DFFF (Dx Bank 1)
	bankIDMainDX2RAM                    // $D000..$DFFF (Dx Bank 2)
	bankIDMainEFRAM                     // $E000..$FFFF (EF RAM)
	bankIDMainDisplayPage1              // $0400..$07FF (Text+LoRes page 1)
	bankIDMainDisplayPage2              // $0800..$0BFF (Text+LoRes page 2)
	bankIDMainHiRes1                    // $2000..$3FFF (HiRes page 1)
	bankIDMainHiRes2                    // $4000..$5FFF (HiRes page 2)
	bankIDAuxZPS                        // $0000..$01FF (ZeroPage+Stack)
	bankIDAuxRAM                        // $0200..$BFFF (Lower 48K)
	bankIDAuxDX1RAM                     // $D000..$DFFF (Dx Bank 1)
	bankIDAuxDX2RAM                     // $D000..$DFFF (Dx Bank 2)
	bankIDAuxEFRAM                      // $E000..$FFFF (EF RAM)
	bankIDAuxDisplayPage1               // $0400..$07FF (Text+LoRes page 1)
	bankIDAuxHiRes1                     // $2000..$3FFF (HiRes page 1)
	bankIDIOSwitches                    // $C000..$C0FF (IOU soft switches)
	bankIDSlotROM                       // $C100..$C7FF (Slot ROM)
	bankIDExpansionROM                  // $C800..$CFFF (Expansion ROM)

	bankCount
)

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
	mainRAM       []byte // entire physical 64K main RAM address space
	auxRAM        []byte // entire physical 64K aux RAM address space
	systemROM     []byte // Holds 16K of Apple II CD/EF ROMs
	peripheralROM []byte // Holds 4K peripheral ROM

	banks [bankCount]bank // all known memory banks
	pages [256]page       // virtual 64K address space broken into 256-byte pages
}

func newMMU() *mmu {
	mainRAM := make([]byte, 64*1024)
	auxRAM := make([]byte, 64*1024)
	systemROM := make([]byte, 16*1024)
	peripheralROM := make([]byte, 4*1024)

	m := &mmu{
		mainRAM:       mainRAM,
		auxRAM:        auxRAM,
		systemROM:     systemROM,
		peripheralROM: peripheralROM,
	}

	// Create all possible memory banks.
	m.addRAMBank(bankIDMainZPS, mainRAM[0x0000:0x0200], 0x0000)
	m.addRAMBank(bankIDMainRAM, mainRAM[0x0200:0xc000], 0x0200)
	m.addRAMBank(bankIDMainDX1RAM, mainRAM[0xc000:0xd000], 0xd000)
	m.addRAMBank(bankIDMainDX2RAM, mainRAM[0xd000:0xe000], 0xd000)
	m.addRAMBank(bankIDMainEFRAM, mainRAM[0xe000:], 0xe000)
	m.addDisplayBank(bankIDMainDisplayPage1, mainRAM[0x0400:0x0800], 0x0400)
	m.addDisplayBank(bankIDMainDisplayPage2, mainRAM[0x0800:0x0c00], 0x0800)
	m.addHiResBank(bankIDMainHiRes1, mainRAM[0x2000:0x4000], 0x2000)
	m.addHiResBank(bankIDMainHiRes2, mainRAM[0x4000:0x8000], 0x4000)
	m.addROMBank(bankIDSystemCXROM, systemROM[0x0100:0x1000], 0xc100)
	m.addROMBank(bankIDSystemDEFROM, systemROM[0x1000:0x4000], 0xd000)
	m.addIOSwitchBank(bankIDIOSwitches, 0x0100, 0xc000)
	m.addIOSlotROMBank(bankIDSlotROM, 0x0700, 0xc100)
	m.addIOExpansionROMBank(bankIDExpansionROM, 0x800, 0xc800)
	m.addRAMBank(bankIDAuxZPS, mainRAM[0x0000:0x0200], 0x0000)
	m.addRAMBank(bankIDAuxRAM, auxRAM[0x0200:0xc000], 0x0200)
	m.addRAMBank(bankIDAuxDX1RAM, auxRAM[0xc000:0xd000], 0xd000)
	m.addRAMBank(bankIDAuxDX2RAM, auxRAM[0xd000:0xe000], 0xd000)
	m.addRAMBank(bankIDAuxEFRAM, auxRAM[0xe000:], 0xe000)
	m.addDisplayBank(bankIDAuxDisplayPage1, auxRAM[0x0400:0x0800], 0x0400)
	m.addHiResBank(bankIDAuxHiRes1, auxRAM[0x2000:0x4000], 0x2000)

	// Activate initial memory banks.
	m.activateBank(bankIDMainZPS, read|write)
	m.activateBank(bankIDMainRAM, read|write)
	m.activateBank(bankIDMainDisplayPage1, read|write)
	m.activateBank(bankIDSystemCXROM, read)
	m.activateBank(bankIDSystemDEFROM, read)
	m.activateBank(bankIDIOSwitches, read|write)

	return m
}

func (m *mmu) LoadByte(addr uint16) byte {
	b := m.pages[addr>>8].read
	if b == nil {
		return 0
	}

	paddr := addr - b.baseAddr
	return b.accessor.LoadByte(paddr)
}

func (m *mmu) LoadBytes(addr uint16, b []byte) {
	for i, n := 0, len(b); i < n; i++ {
		b[i] = m.LoadByte(addr + uint16(i))
	}
}

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

func (m *mmu) StoreByte(addr uint16, v byte) {
	b := m.pages[addr>>8].write
	if b == nil {
		return
	}

	paddr := addr - b.baseAddr
	b.accessor.StoreByte(paddr, v)
}

func (m *mmu) StoreBytes(addr uint16, b []byte) {
	for i, n := 0, len(b); i < n; i++ {
		m.StoreByte(addr+uint16(i), b[i])
	}
}

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

func (m *mmu) addRAMBank(id uint8, mem []byte, baseAddr uint16) {
	m.banks[id] = bank{
		id:       id,
		size:     uint16(len(mem)),
		baseAddr: baseAddr,
		accessor: &ramBankAccessor{mem: mem},
	}
}

func (m *mmu) addROMBank(id uint8, mem []byte, baseAddr uint16) {
	m.banks[id] = bank{
		id:       id,
		size:     uint16(len(mem)),
		baseAddr: baseAddr,
		accessor: &romBankAccessor{mem: mem},
	}
}

func (m *mmu) addDisplayBank(id uint8, mem []byte, baseAddr uint16) {
	m.banks[id] = bank{
		id:       id,
		size:     uint16(len(mem)),
		baseAddr: baseAddr,
		accessor: &displayBankAccessor{mem: mem},
	}
}

func (m *mmu) addHiResBank(id uint8, mem []byte, baseAddr uint16) {
	m.banks[id] = bank{
		id:       id,
		size:     uint16(len(mem)),
		baseAddr: baseAddr,
		accessor: &hiResBankAccessor{mem: mem},
	}
}

func (m *mmu) addIOSwitchBank(id uint8, size, baseAddr uint16) {
	m.banks[id] = bank{
		id:       id,
		size:     size,
		baseAddr: baseAddr,
	}
}

func (m *mmu) addIOSlotROMBank(id uint8, size, baseAddr uint16) {
	m.banks[id] = bank{
		id:       id,
		size:     size,
		baseAddr: baseAddr,
	}
}

func (m *mmu) addIOExpansionROMBank(id uint8, size, baseAddr uint16) {
	m.banks[id] = bank{
		id:       id,
		size:     size,
		baseAddr: baseAddr,
	}
}

func (m *mmu) setBankAccessor(bankID uint8, a bankAccessor) {
	m.banks[bankID].accessor = a
}

// getBankAccess returns the current access (read and/or write) allowed to the
// requested bank.
func (m *mmu) getBankAccess(bankID uint8) access {
	b := &m.banks[bankID]
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

// activateBank activates all the pages within a bank's range of virtual
// addresses so that accesses to addresses within that range are handled
// by the bank's accessor. Read and write access may be activated
// independently.
func (m *mmu) activateBank(bankID uint8, access access) {
	b := &m.banks[bankID]

	enableReads := (access & read) != 0
	enableWrites := (access & write) != 0

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

// deactivateBank deactivates all the pages within a bank's range of virtual
// addresses so that accesses to addresses within that range are no longer
// handled by the bank. Read and write access may be deactivated independently.
func (m *mmu) deactivateBank(bankID uint8, access access) {
	b := &m.banks[bankID]

	disableReads := (access & read) != 0
	disableWrites := (access & write) != 0

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

type ramBankAccessor struct {
	mem []byte
}

func (a *ramBankAccessor) LoadByte(addr uint16) byte {
	return a.mem[addr]
}

func (a *ramBankAccessor) StoreByte(addr uint16, v byte) {
	a.mem[addr] = v
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

type displayBankAccessor struct {
	mem []byte
}

func (a *displayBankAccessor) LoadByte(addr uint16) byte {
	return a.mem[addr]
}

func (a *displayBankAccessor) StoreByte(addr uint16, v byte) {
	a.mem[addr] = v
}

type hiResBankAccessor struct {
	mem []byte
}

func (a *hiResBankAccessor) LoadByte(addr uint16) byte {
	return a.mem[addr]
}

func (a *hiResBankAccessor) StoreByte(addr uint16, v byte) {
	a.mem[addr] = v
}
