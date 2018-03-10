package main

type apple2 struct {
	mmu *MMU
}

func newApple2() *apple2 {

	return &apple2{
		mmu: NewMMU(),
	}
}

func main() {
}
