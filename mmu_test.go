package main

import "testing"

func TestLCSwitches(t *testing.T) {
	a := newApple2()

	cases := []struct {
		addr   uint16
		efram  access
		dx1ram access
		dx2ram access
		defrom access
	}{
		{0xc080, read, 0, read, 0},
		{0xc081, write, 0, write, read},
		{0xc082, 0, 0, 0, read},
		{0xc083, read | write, 0, read | write, 0},
		{0xc084, read, 0, read, 0},
		{0xc085, write, 0, write, read},
		{0xc086, 0, 0, 0, read},
		{0xc087, read | write, 0, read | write, 0},
		{0xc088, read, read, 0, 0},
		{0xc089, write, write, 0, read},
		{0xc08a, 0, 0, 0, read},
		{0xc08b, read | write, read | write, 0, 0},
		{0xc08c, read, read, 0, 0},
		{0xc08d, write, write, 0, read},
		{0xc08e, 0, 0, 0, read},
		{0xc08f, read | write, read | write, 0, 0},
	}

	for i, c := range cases {
		a.mmu.LoadByte(c.addr)
		efram := a.mmu.getBankAccess(bankIDMainEFRAM)
		dx1ram := a.mmu.getBankAccess(bankIDMainDX1RAM)
		dx2ram := a.mmu.getBankAccess(bankIDMainDX2RAM)
		defrom := a.mmu.getBankAccess(bankIDSystemDEFROM)
		if efram != c.efram {
			t.Errorf("Case %d expected EFRAM %d, got %d\n", i, c.efram, efram)
		}
		if dx1ram != c.dx1ram {
			t.Errorf("Case %d expected DX1RAM %d, got %d\n", i, c.dx1ram, dx1ram)
		}
		if dx2ram != c.dx2ram {
			t.Errorf("Case %d expected DX2RAM %d, got %d\n", i, c.dx2ram, dx2ram)
		}
		if defrom != c.defrom {
			t.Errorf("Case %d expected DEFROM %d, got %d\n", i, c.defrom, defrom)
		}
	}
}
