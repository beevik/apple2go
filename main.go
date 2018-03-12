package main

import "github.com/beevik/go6502"

type apple2 struct {
	mmu *mmu
	cpu *go6502.CPU
	iou *iou
}

func newApple2() *apple2 {
	mmu := newMMU()
	cpu := go6502.NewCPU(go6502.NMOS, mmu)
	iou := newIOU(mmu, cpu)

	return &apple2{
		mmu: mmu,
		cpu: cpu,
		iou: iou,
	}
}

func main() {
	apple := newApple2()
	_ = apple
}
