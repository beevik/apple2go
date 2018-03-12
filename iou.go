package main

import "github.com/beevik/go6502"

const (
	switchRAMRD    uint32 = 1 << iota // 1 = Aux RAM read enabled, 0 = main RAM read enabled
	switchRAMWRT                      // 1 = Aux RAM write enabled, 0 = main ram write enabled
	switch80STORE                     // 1 = $0200..$BFFF ignores RAMRD/RAMWRT, 0 = $0200..$BFFF controlled by RAMRD/RAMWRT
	switchPAGE2                       // If 80STORE is 1: 1 = Aux Display page 2 enabled, 0 = Main display page 2 enabled
	switchHIRES                       // If 80STORE is 1: 1 = Aux HiRes page enabled, 0 = Main HiRes page enabled
	switchALTZP                       // 1 = Aux ZP+stack, 0 = Main ZP+stack
	switchLCRAMRD                     // 1 = LC RAM read enabled, 0 = Dx ROM read enabled
	switchLCRAMWRT                    // 1 = LC RAM write enabled, 0 = LC RAM write disabled
	switchLCBANK2                     // 1 = LC RAM bank 2 enabled, 0 = LC RAM bank 1 enabled
)

type iou struct {
	mmu *mmu
	cpu *go6502.CPU
}

func newIOU(mmu *mmu, cpu *go6502.CPU) *iou {
	iou := &iou{
		mmu: mmu,
		cpu: cpu,
	}
	mmu.setBankAccessor(bankIDIOSwitches, &ioSwitchBankAccessor{iou: iou})
	return iou
}

func (iou *iou) readSoftSwitch(addr uint16) byte {
	return 0
}

func (iou *iou) writeSoftSwitch(addr uint16, v byte) {
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

// 00 = WriteRAM/0 ReadRAM/1 ReadROM/0
// 01 = WriteRAM/1 ReadRAM/0 ReadROM/1
// 10 = WriteRAM/0 ReadRAM/0 ReadROM/1
// 11 = WriteRAM/1 ReadRAM/1 ReadROM/0

// ReadROM = !ReadRAM

// WriteRAM = bit 0
// ReadRAM = !(bit0 ^ bit1)
