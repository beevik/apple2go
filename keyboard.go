package main

type keyboard struct {
	apple2  *apple2
	keydata byte
	keydown bool
}

const (
	keyStrobe byte = 0x80
)

func newKeyboard(apple2 *apple2) *keyboard {
	return &keyboard{
		apple2: apple2,
	}
}

func (kb *keyboard) Init() {
}

func (kb *keyboard) IsKeyDown() bool {
	return kb.keydown
}

func (kb *keyboard) GetKeyData() byte {
	return kb.keydata
}

func (kb *keyboard) SetKey(v byte) {
	kb.keydata = v | keyStrobe
}

func (kb *keyboard) ResetKeyStrobe() {
	kb.keydata &= ^keyStrobe
}
