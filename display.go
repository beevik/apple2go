package main

type displayBankAccessor struct {
	mem []byte
}

func (a *displayBankAccessor) LoadByte(addr uint16) byte {
	return a.mem[addr]
}

func (a *displayBankAccessor) StoreByte(addr uint16, v byte) {
	a.mem[addr] = v
}

func (a *displayBankAccessor) CopyBytes(b []byte) {
	copy(a.mem, b)
}

type hiResBankAccessor struct {
	mem []byte
}

func (a *hiResBankAccessor) LoadByte(addr uint16) byte {
	return a.mem[addr]
}

func (a *hiResBankAccessor) StoreByte(addr uint16, v byte) {
	a.mem[addr] = v
}

func (a *hiResBankAccessor) CopyBytes(b []byte) {
	copy(a.mem, b)
}
