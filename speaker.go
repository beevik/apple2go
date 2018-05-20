package main

type speaker struct {
	apple2 *apple2
}

func newSpeaker(apple2 *apple2) *speaker {
	return &speaker{
		apple2: apple2,
	}
}

func (s *speaker) Init() {
}

func (s *speaker) Toggle() {
	// toggle speaker diaphragm
}
