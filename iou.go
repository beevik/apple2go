package main

const (
	switchAUXRAMRD  uint32 = 1 << iota // 1 = Aux RAM read enabled, 0 = main RAM read enabled
	switchAUXRAMWRT                    // 1 = Aux RAM write enabled, 0 = main ram write enabled
	switch80STORE                      // 1 = $0200..$BFFF ignores RAMRD/RAMWRT, 0 = $0200..$BFFF controlled by RAMRD/RAMWRT
	switchPAGE2                        // If 80STORE is 1: 1 = Aux Display page 2 enabled, 0 = Main display page 2 enabled
	switchHIRES                        // If 80STORE is 1: 1 = Aux HiRes page enabled, 0 = Main HiRes page enabled
	switchALTZP                        // 1 = Aux ZP+stack, 0 = Main ZP+stack
	switchLCRAMRD                      // 1 = LC RAM read enabled, 0 = LC ROM read enabled
	switchLCRAMWRT                     // 1 = LC RAM write enabled, 0 = LC RAM write disabled
	switchLCBANK2                      // 1 = LC RAM bank 2 enabled, 0 = LC RAM bank 1 enabled
)

type iou struct {
	mmu *mmu

	switches uint32
}

func newIOU(mmu *mmu) *iou {
	iou := &iou{
		mmu: mmu,
	}
	mmu.setBankAccessor(bankIDIOSwitches, &ioSwitchBankAccessor{iou: iou})
	return iou
}

func bitTest16(v uint16, mask uint16) bool {
	return (v & mask) != 0
}

func (iou *iou) testSoftSwitch(sw uint32) bool {
	return (iou.switches & sw) != 0
}

func (iou *iou) setSoftSwitch(sw uint32, v bool) {
	if v {
		iou.switches |= sw
	} else {
		iou.switches &^= sw
	}
}

func (iou *iou) readSoftSwitch(addr uint16) byte {
	switch addr & 0xf0 {
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
		iou.setSoftSwitch(switchLCRAMRD, !bitTest16(addr^(addr>>1), 1<<0))
		iou.setSoftSwitch(switchLCRAMWRT, bitTest16(addr, 1<<0))
		iou.setSoftSwitch(switchLCBANK2, !bitTest16(addr, 1<<3))
		iou.applyLCSwitches()
		return 0xa0
	}
	return 0
}

func (iou *iou) writeSoftSwitch(addr uint16, v byte) {
}

// Apply soft switches to activate or deactivate memory banks in the
// language card memory range ($D000..$FFFF).
func (iou *iou) applyLCSwitches() {
	mmu := iou.mmu

	if iou.testSoftSwitch(switchLCRAMRD) {
		if iou.testSoftSwitch(switchAUXRAMRD) {
			mmu.activateBank(bankIDAuxEFRAM, read)
			if iou.testSoftSwitch(switchLCBANK2) {
				mmu.activateBank(bankIDAuxDX2RAM, read)
			} else {
				mmu.activateBank(bankIDAuxDX1RAM, read)
			}
		} else {
			mmu.activateBank(bankIDMainEFRAM, read)
			if iou.testSoftSwitch(switchLCBANK2) {
				mmu.activateBank(bankIDMainDX2RAM, read)
			} else {
				mmu.activateBank(bankIDMainDX1RAM, read)
			}
		}
	} else {
		mmu.activateBank(bankIDSystemDEFROM, read)
	}

	if iou.testSoftSwitch(switchLCRAMWRT) {
		if iou.testSoftSwitch(switchAUXRAMWRT) {
			mmu.activateBank(bankIDAuxEFRAM, write)
			if iou.testSoftSwitch(switchLCBANK2) {
				mmu.activateBank(bankIDAuxDX2RAM, write)
			} else {
				mmu.activateBank(bankIDAuxDX1RAM, write)
			}
		} else {
			mmu.activateBank(bankIDMainEFRAM, write)
			if iou.testSoftSwitch(switchLCBANK2) {
				mmu.activateBank(bankIDMainDX2RAM, write)
			} else {
				mmu.activateBank(bankIDMainDX1RAM, write)
			}
		}
	} else {
		if iou.testSoftSwitch(switchAUXRAMWRT) {
			mmu.deactivateBank(bankIDAuxEFRAM, write)
			mmu.deactivateBank(bankIDAuxDX1RAM, write)
			mmu.deactivateBank(bankIDAuxDX2RAM, write)
		} else {
			mmu.deactivateBank(bankIDMainEFRAM, write)
			mmu.deactivateBank(bankIDMainDX1RAM, write)
			mmu.deactivateBank(bankIDMainDX2RAM, write)
		}
	}
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
