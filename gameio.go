package main

type gameIO struct {
	apple2 *apple2
}

func newGameIO(apple2 *apple2) *gameIO {
	return &gameIO{
		apple2: apple2,
	}
}

func (g *gameIO) Init() {
}

func (g *gameIO) GetStrobe() byte {
	return 0
}
