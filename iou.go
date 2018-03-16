package main

type ioSwitch uint8

const (
	ioSwitchAUXRAMRD   ioSwitch = iota // 1 = Aux RAM read enabled, 0 = main RAM read enabled
	ioSwitchAUXRAMWRT                  // 1 = Aux RAM write enabled, 0 = main ram write enabled
	ioSwitchALTCHARSET                 // 1 = Alt char set on, 0 = off
	ioSwitchTEXT                       // 1 = text mode on, 0 = off
	ioSwitchMIXED                      // 1 = mixed mode on, 0 = off
	ioSwitch80COL                      // 1 = 80 column mode on, 0 = off
	ioSwitch80STORE                    // 1 = $0200..$BFFF ignores RAMRD/RAMWRT, 0 = $0200..$BFFF controlled by RAMRD/RAMWRT
	ioSwitchPAGE2                      // If 80STORE is 1: 1 = Aux Display page 2 enabled, 0 = Main display page 2 enabled
	ioSwitchHIRES                      // If 80STORE is 1: 1 = Aux HiRes page enabled, 0 = Main HiRes page enabled
	ioSwitchDHIRES                     // 1 = double hires on, 0 = off
	ioSwitchIOUDIS                     // 1 = disable C058..C05F, enable DHIRES. 0=opposite
	ioSwitchALTZP                      // 1 = Aux ZP+stack, 0 = Main ZP+stack
	ioSwitchLCRAMRD                    // 1 = LC RAM read enabled, 0 = LC ROM read enabled
	ioSwitchLCRAMWRT                   // 1 = LC RAM write enabled, 0 = LC RAM write disabled
	ioSwitchLCBANK2                    // 1 = LC RAM bank 2 enabled, 0 = LC RAM bank 1 enabled
	ioSwitchVBL                        // read vertical blanking
)

const (
	updateSystemRAM uint32 = 1 << iota // update lower 48K memory banks (except ZPS)
	updateZPSRAM                       // update zero and stack pages
	updateLCRAM                        // update upper 16K memory  banks
)

var switchUpdates = []uint32{
	updateSystemRAM | updateLCRAM, // ioSwitchAUXRAMRD
	updateSystemRAM | updateLCRAM, // ioSwitchAUXRAMWRT
	0,               // ioSwitchALTCHARSET
	0,               // ioSwitchTEXT
	0,               // ioSwitchMIXED
	0,               // ioSwitch80COL
	updateSystemRAM, // ioSwitch80STORE
	updateSystemRAM, // ioSwitchPAGE2
	updateSystemRAM, // ioSwitchHIRES
	0,               // ioSwitchDHIRES
	0,               // ioSwitchIOUDIS
	updateZPSRAM,    // ioSwitchALTZP
	updateLCRAM,     // ioSwitchLCRAMRD
	updateLCRAM,     // ioSwitchLCRAMWRT
	updateLCRAM,     // ioSwitchLCBANK2
	0,               // ioSwitchVBL
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

func (iou *iou) getSoftSwitch(sw ioSwitch) byte {
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

func (iou *iou) readSoftSwitch(addr uint16) byte {
	var ret byte

	switch addr & 0xf0 {
	case 0x10:
		switch addr {
		case 0x13:
			ret = iou.getSoftSwitch(ioSwitchAUXRAMRD)
		case 0x14:
			ret = iou.getSoftSwitch(ioSwitchAUXRAMWRT)
		case 0x16:
			ret = iou.getSoftSwitch(ioSwitchALTZP)
		case 0x18:
			ret = iou.getSoftSwitch(ioSwitch80STORE)
		case 0x1c:
			ret = iou.getSoftSwitch(ioSwitchPAGE2)
		case 0x1d:
			ret = iou.getSoftSwitch(ioSwitchHIRES)
		}

		// Language card (LC) bank switching:
	case 0x80:
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
		ret = 0xa0
	}

	iou.applySwitchUpdates()
	return ret
}

func (iou *iou) writeSoftSwitch(addr uint16, v byte) {
	switch addr & 0xf0 {
	case 0x00:
		switch addr {
		case 0x00:
			iou.setSoftSwitch(ioSwitch80STORE, false)
		case 0x01:
			iou.setSoftSwitch(ioSwitch80STORE, true)
		case 0x02:
			iou.setSoftSwitch(ioSwitchAUXRAMRD, false)
		case 0x03:
			iou.setSoftSwitch(ioSwitchAUXRAMRD, true)
		case 0x04:
			iou.setSoftSwitch(ioSwitchAUXRAMWRT, false)
		case 0x05:
			iou.setSoftSwitch(ioSwitchAUXRAMWRT, true)
		case 0x08:
			iou.setSoftSwitch(ioSwitchALTZP, false)
		case 0x09:
			iou.setSoftSwitch(ioSwitchALTZP, true)
		}
	case 0x50:
		switch addr {
		case 0x54:
			iou.setSoftSwitch(ioSwitchPAGE2, false)
		case 0x55:
			iou.setSoftSwitch(ioSwitchPAGE2, true)
		case 0x56:
			iou.setSoftSwitch(ioSwitchHIRES, false)
		case 0x57:
			iou.setSoftSwitch(ioSwitchHIRES, true)
		}
	}

	iou.applySwitchUpdates()
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
	return a.iou.readSoftSwitch(addr)
}

func (a *ioSwitchBankAccessor) StoreByte(addr uint16, v byte) {
	a.iou.writeSoftSwitch(addr, v)
}

func (a *ioSwitchBankAccessor) CopyBytes(b []byte) {
	// Do nothing
}
