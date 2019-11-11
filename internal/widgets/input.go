package form

import (
	"image"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

// Input is the definition of an Input component
type Input struct {
	// TODO make private or embed this instead?
	Paragraph *widgets.Paragraph
}

var _ ui.Drawable = (*Input)(nil)

// NewInput creates a new Input
func NewInput() *Input {

	input := &Input{
		Paragraph: widgets.NewParagraph()}

	return input
}

func (input *Input) GetRect() image.Rectangle {
	return input.Paragraph.GetRect()
}

func (input *Input) SetRect(x1, y1, x2, y2 int) {
	input.Paragraph.SetRect(x1, y1, x2, y2)
	if input.Paragraph.Dy() < 3 {
		panic("Input can't be less than 3 high")
	}
}

func (input *Input) Draw(buffer *ui.Buffer) {
	input.Paragraph.Draw(buffer)
}

func (input *Input) Lock() {
	input.Paragraph.Lock()
}

func (input *Input) Unlock() {
	input.Paragraph.Unlock()
}

func (input *Input) Append(s string) {
	input.Paragraph.Text += s
}

func (input *Input) OnBackspace() {
	length := len(input.Paragraph.Text)
	if length == 0 {
		return
	}
	input.Paragraph.Text = input.Paragraph.Text[:length-1]
}

func (input *Input) Clear() {
	input.Paragraph.Text = ""
}
