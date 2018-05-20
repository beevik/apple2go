package main

import (
	"fmt"
	"os"

	"github.com/beevik/go6502/cpu"
)

type apple2 struct {
	mmu *mmu
	iou *iou
	kb  *keyboard
	sp  *speaker
	gi  *gameIO
	cpu *cpu.CPU
}

func newApple2() *apple2 {
	apple2 := &apple2{}

	apple2.mmu = newMMU(apple2)
	apple2.iou = newIOU(apple2)
	apple2.kb = newKeyboard(apple2)
	apple2.sp = newSpeaker(apple2)
	apple2.gi = newGameIO(apple2)
	apple2.cpu = cpu.NewCPU(cpu.NMOS, apple2.mmu)

	apple2.mmu.Init()
	apple2.iou.Init()
	apple2.kb.Init()
	apple2.sp.Init()
	apple2.gi.Init()

	return apple2
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
