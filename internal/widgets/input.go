package wid

import (
	"bytes"
	"image"

	ui "github.com/gizak/termui/v3"
)

// Input allows a user to enter text such as for chat
type Input struct {
	ui.Block
	buff bytes.Buffer
}

var _ ui.Drawable = (*Input)(nil)

// NewInput creates a new Input
func NewInput() *Input {

	input := &Input{
		Block: *ui.NewBlock()}

	return input
}

// Draw updates the shell portion of the Input object
func (input *Input) Draw(buffer *ui.Buffer) {
	input.Block.Draw(buffer)

	for i, s := range input.buff.String() {

		cell := ui.Cell{Rune: s,
			Style: ui.StyleClear}
		x := i % input.Inner.Dx()
		y := i / input.Inner.Dx()

		if y >= input.Inner.Dy() {
			return
		}

		buffer.SetCell(cell, image.Pt(x, y).Add(input.Inner.Min))
	}
}

// AppendChar adds one character to the Input
func (input *Input) AppendChar(s string) {
	if len(s) != 1 {
		// TODO complain with some error message
		return
	}
	_, _ = input.buff.WriteString(s)
}

// OnBackspace removes the last character from the input
func (input *Input) OnBackspace() {
	length := input.buff.Len()
	if length != 0 {
		input.buff.Truncate(length - 1)
	}
}

// Submit returns the input text and empties it
func (input *Input) Submit() string {
	text := input.buff.String()
	input.buff.Reset()
	return text
}
