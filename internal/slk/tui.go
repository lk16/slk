package slk

import (
	"fmt"
	"log"
	"time"

	"github.com/marcusolsson/tui-go"
)

type terminalUI struct {
	handlers map[string]func(*terminalUI)
}

func NewTerminalUI() (*terminalUI, error) {
	sidebar := tui.NewVBox()
	sidebar.Append(tui.NewLabel("CHANNELS"))
	sidebar.Append(tui.NewLabel(""))
	sidebar.Append(tui.NewLabel("DIRECT MESSAGES"))
	sidebar.Append(tui.NewLabel(""))
	sidebar.SetBorder(true)
	sidebar.SetSizePolicy(tui.Minimum, tui.Maximum)

	history := tui.NewVBox()
	historyScroll := tui.NewScrollArea(history)
	historyScroll.SetAutoscrollToBottom(true)

	historyBox := tui.NewVBox(historyScroll)
	historyBox.SetBorder(true)

	input := tui.NewEntry()
	input.SetFocused(true)
	input.SetSizePolicy(tui.Expanding, tui.Maximum)

	inputBox := tui.NewHBox(input)
	inputBox.SetBorder(true)
	inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)

	chat := tui.NewVBox(historyBox, inputBox)
	chat.SetSizePolicy(tui.Expanding, tui.Expanding)

	input.OnSubmit(func(e *tui.Entry) {
		history.Append(tui.NewHBox(
			tui.NewLabel(time.Now().Format("15:04")),
			tui.NewPadder(1, 0, tui.NewLabel(fmt.Sprintf("<%s>", "john"))),
			tui.NewLabel(e.Text()),
			tui.NewSpacer(),
		))
		input.SetText("")
	})

	root := tui.NewHBox(sidebar, chat)

	ui, err := tui.New(root)
	if err != nil {
		log.Fatal(err)
	}

	ui.SetKeybinding("Esc", func() { ui.Quit() })

	if err := ui.Run(); err != nil {
		log.Fatal(err)
	}

	return nil, nil
}

/*
TODO
- init tui with empty stuff
- handle updates as they come in over channel

*/
