package main

type apple2 struct {
	mmu *MMU
}

func newApple2() *apple2 {
	mmu := NewMMU()

	return &apple2{
		mmu: mmu,
	}
}

func main() {
	apple := newApple2()
	_ = apple
}
