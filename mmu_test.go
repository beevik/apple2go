package main

import "testing"

func TestWriteC00xSwitches(t *testing.T) {
	a := newApple2()

	cases := []struct {
		writeAddr uint16
		readAddr  uint16
		value     bool
	}{
		{0xc002, 0xc013, false},
		{0xc003, 0xc013, true},
		{0xc004, 0xc014, false},
		{0xc005, 0xc014, true},
		{0xc006, 0xc015, false},
		{0xc007, 0xc015, true},
		{0xc008, 0xc016, false},
		{0xc009, 0xc016, true},
		{0xc00a, 0xc017, false},
		{0xc00b, 0xc017, true},
		{0xc00c, 0xc01f, false},
		{0xc00d, 0xc01f, true},
		{0xc00e, 0xc01e, false},
		{0xc00f, 0xc01e, true},
	}

	for _, c := range cases {
		a.mmu.StoreByte(c.writeAddr, 0)
		v := a.mmu.LoadByte(c.readAddr)&0x80 == 0x80
		if v != c.value {
			t.Errorf("Wrote %04x, read %04x, expected %v, got %v\n", c.writeAddr, c.readAddr, c.value, v)
		}
	}
}

func TestReadC08xSwitches(t *testing.T) {
	a := newApple2()

	cases := []struct {
		setAddr uint16
		rdlcram bool
		rdbnk2  bool
		efram   access
		dx1ram  access
		dx2ram  access
		defrom  access
	}{
		{0xc080, true, true, read, 0, read, write},
		{0xc081, false, true, write, 0, write, read},
		{0xc082, false, true, 0, 0, 0, read | write},
		{0xc083, true, true, read | write, 0, read | write, 0},
		{0xc084, true, true, read, 0, read, write},
		{0xc085, false, true, write, 0, write, read},
		{0xc086, false, true, 0, 0, 0, read | write},
		{0xc087, true, true, read | write, 0, read | write, 0},
		{0xc088, true, false, read, read, 0, write},
		{0xc089, false, false, write, write, 0, read},
		{0xc08a, false, false, 0, 0, 0, read | write},
		{0xc08b, true, false, read | write, read | write, 0, 0},
		{0xc08c, true, false, read, read, 0, write},
		{0xc08d, false, false, write, write, 0, read},
		{0xc08e, false, false, 0, 0, 0, read | write},
		{0xc08f, true, false, read | write, read | write, 0, 0},
	}

	for _, c := range cases {
		a.mmu.LoadByte(c.setAddr)

		rdlcram := (a.iou.getSoftSwitch(ioSwitchLCRAMRD) & 0x80) != 0
		rdbnk2 := (a.iou.getSoftSwitch(ioSwitchLCBANK2) & 0x80) != 0

		if c.rdlcram != rdlcram {
			t.Errorf("Switch %04x: expected LCRAMRD to be %v\n", c.setAddr, c.rdlcram)
		}
		if c.rdbnk2 != rdbnk2 {
			t.Errorf("Switch %04x: expected LCBANK2 to be %v\n", c.setAddr, c.rdbnk2)
		}

		efram := a.mmu.getBankAccess(bankLangCardEFRAM, bankTypeMain)
		dx1ram := a.mmu.getBankAccess(bankLangCardDX1RAM, bankTypeMain)
		dx2ram := a.mmu.getBankAccess(bankLangCardDX2RAM, bankTypeMain)
		defrom := a.mmu.getBankAccess(bankSystemDEFROM, bankTypeMain)
		if efram != c.efram {
			t.Errorf("Switch %04x: expected EFRAM %d, got %d\n", c.setAddr, c.efram, efram)
		}
		if dx1ram != c.dx1ram {
			t.Errorf("Switch %04x: expected DX1RAM %d, got %d\n", c.setAddr, c.dx1ram, dx1ram)
		}
		if dx2ram != c.dx2ram {
			t.Errorf("Switch %04x: expected DX2RAM %d, got %d\n", c.setAddr, c.dx2ram, dx2ram)
		}
		if defrom != c.defrom {
			t.Errorf("Switch %04x: expected DEFROM %d, got %d\n", c.setAddr, c.defrom, defrom)
		}
	}
}
