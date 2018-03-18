package main

type ioSwitch uint8

const (
	ioSwitchAUXRAMRD     ioSwitch = iota // 1 = Aux RAM read enabled, 0 = main RAM read enabled
	ioSwitchAUXRAMWRT                    // 1 = Aux RAM write enabled, 0 = main ram write enabled
	ioSwitchALTCHARSET                   // 1 = Alt char set on, 0 = off
	ioSwitchTEXT                         // 1 = text mode on, 0 = off
	ioSwitchMIXED                        // 1 = mixed mode on, 0 = off
	ioSwitch80COL                        // 1 = 80 column mode on, 0 = off
	ioSwitch80STORE                      // 1 = $0200..$BFFF ignores RAMRD/RAMWRT, 0 = $0200..$BFFF controlled by RAMRD/RAMWRT
	ioSwitchPAGE2                        // If 80STORE is 1: 1 = Aux Display page 2 enabled, 0 = Main display page 2 enabled
	ioSwitchHIRES                        // If 80STORE is 1: 1 = Aux HiRes page enabled, 0 = Main HiRes page enabled
	ioSwitchDHIRES                       // 1 = double hires on, 0 = off
	ioSwitchIOUDIS                       // 1 = disable C058..C05F, enable DHIRES. 0=opposite
	ioSwitchALTZP                        // 1 = Aux ZP+stack, 0 = Main ZP+stack
	ioSwitchLCRAMRD                      // 1 = LC RAM read enabled, 0 = LC ROM read enabled
	ioSwitchLCRAMWRT                     // 1 = LC RAM write enabled, 0 = LC RAM write disabled
	ioSwitchLCBANK2                      // 1 = LC RAM bank 2 enabled, 0 = LC RAM bank 1 enabled
	ioSwitchCXROM                        // 1 = using internal slot ROM, 0 = not using
	ioSwitchC3ROM                        // 1 = using slot 3 ROM, 0 = not using
	ioSwitchVBLINT                       // read vertical blanking
	ioSwitchANNUNCIATOR0                 // 1 = hand control annunciator 0 on, 0 = off
	ioSwitchANNUNCIATOR1                 // 1 = hand control annunciator 1 on, 0 = off
	ioSwitchANNUNCIATOR2                 // 1 = hand control annunciator 2 on, 0 = off
	ioSwitchANNUNCIATOR3                 // 1 = hand control annunciator 3 on, 0 = off

	ioSwitches
)

const (
	updateSystemRAM uint32 = 1 << iota // update lower 48K memory banks (except ZPS)
	updateZPSRAM                       // update zero and stack pages
	updateLCRAM                        // update upper 16K memory banks
)

var switchUpdates = []uint32{
	/* ioSwitchAUXRAMRD     */ updateSystemRAM | updateLCRAM,
	/* ioSwitchAUXRAMWRT    */ updateSystemRAM | updateLCRAM,
	/* ioSwitchALTCHARSET   */ 0,
	/* ioSwitchTEXT         */ 0,
	/* ioSwitchMIXED        */ 0,
	/* ioSwitch80COL        */ 0,
	/* ioSwitch80STORE      */ updateSystemRAM,
	/* ioSwitchPAGE2   	    */ updateSystemRAM,
	/* ioSwitchHIRES        */ updateSystemRAM,
	/* ioSwitchDHIRES       */ 0,
	/* ioSwitchIOUDIS       */ 0,
	/* ioSwitchALTZP        */ updateZPSRAM,
	/* ioSwitchLCRAMRD      */ updateLCRAM,
	/* ioSwitchLCRAMWRT     */ updateLCRAM,
	/* ioSwitchLCBANK2      */ updateLCRAM,
	/* ioSwitchCXROM        */ 0,
	/* ioSwitchC3ROM        */ 0,
	/* ioSwitchVBLINT       */ 0,
	/* ioSwitchANNUNCIATOR0 */ 0,
	/* ioSwitchANNUNCIATOR1 */ 0,
	/* ioSwitchANNUNCIATOR2 */ 0,
	/* ioSwitchANNUNCIATOR3 */ 0,
}

type iou struct {
	mmu *mmu

	switches uint32 // bitmask of current switch settings
	updates  uint32 // pending updates required
}

func newIOU(mmu *mmu) *iou {
	iou := &iou{
		mmu: mmu,
	}
	mmu.setBankAccessor(bankIOSwitches, bankTypeMain, &ioSwitchBankAccessor{iou: iou})
	return iou
}

func bitTest16(v uint16, mask uint16) bool {
	return (v & mask) != 0
}

func (iou *iou) testSoftSwitch(sw ioSwitch) bool {
	return (iou.switches & (1 << sw)) != 0
}

func (iou *iou) getSoftSwitchBit7(sw ioSwitch) byte {
	if (iou.switches & (1 << sw)) != 0 {
		return 0x80
	}
	return 0
}

func (iou *iou) setSoftSwitch(sw ioSwitch, v bool) {
	orig := iou.switches

	if v {
		iou.switches |= (1 << sw)
	} else {
		iou.switches &= ^(1 << sw)
	}

	if orig != iou.switches {
		iou.updates |= switchUpdates[sw]
	}
}

var switchBank = []struct {
	read  func(iou *iou, addr uint16) byte
	write func(iou *iou, addr uint16)
}{
	/* c00x */ {write: (*iou).onSwitchWriteC00x},
	/* c01x */ {read: (*iou).onSwitchReadC01x},
	/* c02x */ {},
	/* c03x */ {},
	/* c04x */ {},
	/* c05x */ {read: (*iou).onSwitchReadC05x, write: (*iou).onSwitchWriteC05x},
	/* c06x */ {},
	/* c07x */ {write: (*iou).onSwitchWriteC07x},
	/* c08x */ {read: (*iou).onSwitchReadC08x},
}

var switchWriteC00x = []ioSwitch{
	/* c000..c001 */ ioSwitch80STORE,
	/* c002..c003 */ ioSwitchAUXRAMRD,
	/* c004..c005 */ ioSwitchAUXRAMWRT,
	/* c006..c007 */ ioSwitchCXROM,
	/* c008..c009 */ ioSwitchALTZP,
	/* c00a..c00b */ ioSwitchC3ROM,
	/* c00c..c00d */ ioSwitch80COL,
	/* c00e..c00f */ ioSwitchALTCHARSET,
}

func (iou *iou) onSwitchWriteC00x(addr uint16) {
	// Sequence:
	//  addr0: switch1 OFF
	//  addr1: switch1 ON
	//  addr2: switch2 OFF
	//  addr3: switch2 ON
	//  ...etc.

	sw := switchWriteC00x[addr>>1]
	v := (addr & 1) == 1
	iou.setSoftSwitch(sw, v)
}

var switchReadC01x = []ioSwitch{
	ioSwitches,         // c010 (invalid)
	ioSwitchLCBANK2,    // c011
	ioSwitchLCRAMRD,    // c012
	ioSwitchAUXRAMRD,   // c013
	ioSwitchAUXRAMWRT,  // c014
	ioSwitchCXROM,      // c015
	ioSwitchALTZP,      // c016
	ioSwitchC3ROM,      // c017
	ioSwitch80STORE,    // c018
	ioSwitchVBLINT,     // c019
	ioSwitchTEXT,       // c01a
	ioSwitchMIXED,      // c01b
	ioSwitchPAGE2,      // c01c
	ioSwitchHIRES,      // c01d
	ioSwitchALTCHARSET, // c01e
	ioSwitch80COL,      // c01f
}

func (iou *iou) onSwitchReadC01x(addr uint16) byte {
	sw := switchReadC01x[addr-0x10]
	return iou.getSoftSwitchBit7(sw)
}

func (iou *iou) onSwitchReadC05x(addr uint16) byte {
	switch addr {
	case 0x50:
		iou.setSoftSwitch(ioSwitchTEXT, false)
	case 0x51:
		iou.setSoftSwitch(ioSwitchTEXT, true)
	case 0x52:
		iou.setSoftSwitch(ioSwitchMIXED, false)
	case 0x53:
		iou.setSoftSwitch(ioSwitchMIXED, true)
	case 0x54:
		iou.setSoftSwitch(ioSwitchPAGE2, false)
	case 0x55:
		iou.setSoftSwitch(ioSwitchPAGE2, true)
	case 0x56:
		iou.setSoftSwitch(ioSwitchHIRES, false)
	case 0x57:
		iou.setSoftSwitch(ioSwitchHIRES, true)
	case 0x58:
		if !iou.testSoftSwitch(ioSwitchIOUDIS) {
			iou.setSoftSwitch(ioSwitchANNUNCIATOR0, false)
		}
	case 0x59:
		if !iou.testSoftSwitch(ioSwitchIOUDIS) {
			iou.setSoftSwitch(ioSwitchANNUNCIATOR0, true)
		}
	case 0x5a:
		if !iou.testSoftSwitch(ioSwitchIOUDIS) {
			iou.setSoftSwitch(ioSwitchANNUNCIATOR1, false)
		}
	case 0x5b:
		if !iou.testSoftSwitch(ioSwitchIOUDIS) {
			iou.setSoftSwitch(ioSwitchANNUNCIATOR1, true)
		}
	case 0x5c:
		if !iou.testSoftSwitch(ioSwitchIOUDIS) {
			iou.setSoftSwitch(ioSwitchANNUNCIATOR2, false)
		}
	case 0x5d:
		if !iou.testSoftSwitch(ioSwitchIOUDIS) {
			iou.setSoftSwitch(ioSwitchANNUNCIATOR2, true)
		}
	case 0x5e:
		if iou.testSoftSwitch(ioSwitchIOUDIS) {
			iou.setSoftSwitch(ioSwitchDHIRES, true)
		} else {
			iou.setSoftSwitch(ioSwitchANNUNCIATOR3, false)
		}
	case 0x5f:
		if iou.testSoftSwitch(ioSwitchIOUDIS) {
			iou.setSoftSwitch(ioSwitchDHIRES, false)
		} else {
			iou.setSoftSwitch(ioSwitchANNUNCIATOR3, true)
		}
	}
	return 0xa0
}

func (iou *iou) onSwitchWriteC05x(addr uint16) {
	// write does the same as read for the c05x bank of switches.
	_ = iou.onSwitchReadC05x(addr)
}

func (iou *iou) onSwitchReadC07x(addr uint16) byte {
	var ret byte

	switch addr {
	case 0x7e:
		ret = iou.getSoftSwitchBit7(ioSwitchIOUDIS)
	case 0x7f:
		ret = iou.getSoftSwitchBit7(ioSwitchDHIRES)
	}

	iou.setSoftSwitch(ioSwitchVBLINT, false)

	return ret
}

func (iou *iou) onSwitchWriteC07x(addr uint16) {
	switch addr {
	case 0x7e:
		iou.setSoftSwitch(ioSwitchIOUDIS, false)
	case 0x7f:
		iou.setSoftSwitch(ioSwitchIOUDIS, true)
	}
}

func (iou *iou) onSwitchReadC08x(addr uint16) byte {
	// addr (least significant 4 bits, ignore 'z' bit)
	// 0z00 = LCRAMRD=1 LCRAMWRT=0 LCBANK2=1
	// 0z01 = LCRAMRD=0 LCRAMWRT=1 LCBANK2=1
	// 0z10 = LCRAMRD=0 LCRAMWRT=0 LCBANK2=1
	// 0z11 = LCRAMRD=1 LCRAMWRT=1 LCBANK2=1 (RR)
	// 1z00 = LCRAMRD=1 LCRAMWRT=0 LCBANK2=0
	// 1z01 = LCRAMRD=0 LCRAMWRT=1 LCBANK2=0
	// 1z10 = LCRAMRD=0 LCRAMWRT=0 LCBANK2=0
	// 1z11 = LCRAMRD=1 LCRAMWRT=1 LCBANK2=0 (RR)
	// ----
	// LCRAMRD  = !(bit0 ^ bit1)
	// LCRAMWRT = bit 0
	// LCBANK2  = !(bit 3)

	iou.setSoftSwitch(ioSwitchLCRAMRD, !bitTest16(addr^(addr>>1), 1<<0))
	iou.setSoftSwitch(ioSwitchLCRAMWRT, bitTest16(addr, 1<<0))
	iou.setSoftSwitch(ioSwitchLCBANK2, !bitTest16(addr, 1<<3))

	return 0xa0
}

func (iou *iou) applySwitchUpdates() {
	if iou.updates == 0 {
		return
	}

	if (iou.updates & updateZPSRAM) != 0 {
		iou.applyZPSRAMSwitches()
	}
	if (iou.updates & updateSystemRAM) != 0 {
		iou.applySystemRAMSwitches()
	}
	if (iou.updates & updateLCRAM) != 0 {
		iou.applyLCRAMSwitches()
	}

	iou.updates = 0
}

func (iou *iou) applyZPSRAMSwitches() {
	mmu := iou.mmu

	if iou.testSoftSwitch(ioSwitchALTZP) {
		mmu.activateBank(bankZeroStackRAM, bankTypeAux, read|write)
	} else {
		mmu.activateBank(bankZeroStackRAM, bankTypeMain, read|write)
	}
}

func (iou *iou) applySystemRAMSwitches() {
	mmu := iou.mmu

	btr := iou.selectBankType(ioSwitchAUXRAMRD, bankTypeAux, bankTypeMain)
	btw := iou.selectBankType(ioSwitchAUXRAMWRT, bankTypeAux, bankTypeMain)

	mmu.activateBank(bankMainRAM, btr, read)
	mmu.activateBank(bankMainRAM, btw, write)

	if iou.testSoftSwitch(ioSwitch80STORE) {
		bt := iou.selectBankType(ioSwitchPAGE2, bankTypeAux, bankTypeMain)
		mmu.activateBank(bankDisplayPage1, bt, read|write)
		if iou.testSoftSwitch(ioSwitchHIRES) {
			mmu.activateBank(bankHiRes1, bt, read|write)
		}
	} else {
		dp := iou.selectBank(ioSwitchPAGE2, bankDisplayPage2, bankDisplayPage1)
		mmu.activateBank(dp, bankTypeMain, read|write)
		if iou.testSoftSwitch(ioSwitchHIRES) {
			hi := iou.selectBank(ioSwitchPAGE2, bankHiRes2, bankHiRes1)
			mmu.activateBank(hi, bankTypeMain, read|write)
		}
	}
}

func (iou *iou) applyLCRAMSwitches() {
	mmu := iou.mmu

	btr := iou.selectBankType(ioSwitchAUXRAMRD, bankTypeAux, bankTypeMain)
	btw := iou.selectBankType(ioSwitchAUXRAMWRT, bankTypeAux, bankTypeMain)
	lcbank := iou.selectBank(ioSwitchLCBANK2, bankLangCardDX2RAM, bankLangCardDX1RAM)

	if iou.testSoftSwitch(ioSwitchLCRAMRD) {
		mmu.activateBank(bankLangCardEFRAM, btr, read)
		mmu.activateBank(lcbank, btr, read)
	} else {
		mmu.activateBank(bankSystemDEFROM, bankTypeMain, read)
	}

	if iou.testSoftSwitch(ioSwitchLCRAMWRT) {
		mmu.activateBank(bankLangCardEFRAM, btw, write)
		mmu.activateBank(lcbank, btw, write)
	} else {
		mmu.activateBank(bankSystemDEFROM, bankTypeMain, write)
	}
}

func (iou *iou) selectBankType(sw ioSwitch, onResult, offResult bankType) bankType {
	if iou.testSoftSwitch(sw) {
		return onResult
	}
	return offResult
}

func (iou *iou) selectBank(sw ioSwitch, onResult, offResult bankID) bankID {
	if iou.testSoftSwitch(sw) {
		return onResult
	}
	return offResult
}

type ioSwitchBankAccessor struct {
	iou *iou
}

func (a *ioSwitchBankAccessor) LoadByte(addr uint16) byte {
	index := addr >> 4
	if index > 8 {
		return 0
	}

	fn := switchBank[index].read
	if fn == nil {
		return 0
	}

	ret := fn(a.iou, addr)
	a.iou.applySwitchUpdates()
	return ret
}

func (a *ioSwitchBankAccessor) StoreByte(addr uint16, v byte) {
	index := addr >> 4
	if index > 8 {
		return
	}

	fn := switchBank[index].write
	if fn == nil {
		return
	}

	fn(a.iou, addr)
	a.iou.applySwitchUpdates()
}

func (a *ioSwitchBankAccessor) CopyBytes(b []byte) {
	// Do nothing
}
