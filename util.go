package main

func bitTest16(v, mask uint16) bool {
	return (v & mask) != 0
}
