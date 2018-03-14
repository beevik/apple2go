package main

import (
	"fmt"
	"os"

	"github.com/beevik/go6502"
)

type apple2 struct {
	mmu *mmu
	iou *iou
	cpu *go6502.CPU
}

func newApple2() *apple2 {
	mmu := newMMU()
	iou := newIOU(mmu)
	cpu := go6502.NewCPU(go6502.NMOS, mmu)

	return &apple2{
		mmu: mmu,
		iou: iou,
		cpu: cpu,
	}
}

func (a *apple2) LoadROM(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return a.mmu.LoadSystemROM(file)
}

func main() {
	apple := newApple2()

	err := apple.LoadROM("./resources/apple2e.rom")
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
