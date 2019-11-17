package wid

import (
	"fmt"
	"image"
	"time"

	ui "github.com/gizak/termui/v3"
)

// Message represents a chat text entry
type Message struct {
	Timestamp time.Time
	Sender    string
	Text      string
}

// Chat shows chat history
type Chat struct {
	ui.Block
	messages []Message
}

var _ ui.Drawable = (*Chat)(nil)

// NewChat creates a new Chat
func NewChat() *Chat {

	chat := &Chat{
		Block: *ui.NewBlock()}

	return chat
}

// Draw updates the shell portion of the chat object
func (chat *Chat) Draw(buffer *ui.Buffer) {
	chat.Block.Draw(buffer)

	// TODO we assume every message fits on one line

	yOffset := chat.Inner.Dy() - len(chat.messages)
	messageOffset := 0

	if yOffset < 0 {
		yOffset = 0
		messageOffset = len(chat.messages) - chat.Inner.Dy()
	}

	offset := image.Pt(0, yOffset)

	for y, message := range chat.messages[messageOffset:] {

		if y > chat.Inner.Dy() {
			break
		}

		line := fmt.Sprintf("%s %s: %s", message.Timestamp.Format("02/01 15:04"), message.Sender, message.Text)

		for x, s := range line {

			// TODO wrap around
			if x >= chat.Inner.Dx() {
				break
			}

			cell := ui.Cell{Rune: s,
				Style: ui.StyleClear}

			buffer.SetCell(cell, image.Pt(x, y).Add(chat.Inner.Min).Add(offset))
		}
	}
}

// Clear removes all chat messages from the UI
func (chat *Chat) Clear() {
	chat.messages = []Message{}
}

// AddMessage adds a message at the bottom of the chat
func (chat *Chat) AddMessage(message Message) {
	chat.messages = append(chat.messages, message)
}
