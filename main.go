package main

type apple2 struct {
	mmu *mmu
}

func newApple2() *apple2 {
	mmu := newMMU()

	return &apple2{
		mmu: mmu,
	}
}

func main() {
	apple := newApple2()
	_ = apple
}
