package slk

import (
	"fmt"
	"os"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/pkg/errors"
)

type customEvent struct {
	kind string
	data string
}

type TUI struct {
	grid           *ui.Grid
	messagesWidget *widgets.Paragraph
	slackEvents    chan customEvent
}

func NewTUI() (*TUI, error) {

	if err := ui.Init(); err != nil {
		return nil, errors.Wrap(err, "failed to initialize termui")
	}

	p := widgets.NewParagraph()
	p.Text = "channels\ngo\nhere\n"
	p.Title = "Channels"

	messagesWidget := widgets.NewParagraph()
	messagesWidget.Text = "messages\ngo\nhere\n"
	messagesWidget.Title = "Messages"

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	grid.Set(
		ui.NewCol(0.5, p),
		ui.NewCol(0.5, messagesWidget),
	)

	tui := &TUI{
		grid:           grid,
		messagesWidget: messagesWidget,
		slackEvents:    make(chan customEvent)}
	return tui, nil
}

func (tui *TUI) Run() {
	ui.Render(tui.grid)
	defer ui.Close()

	go func() {
		for {
			time.Sleep(30 * time.Millisecond)
			tui.slackEvents <- customEvent{kind: "message", data: "some message"}
		}
	}()

	events := ui.PollEvents()

	for {
		select {
		case event := <-events:
			switch event.ID {
			case "q", "<C-c>":
				ui.Close()

				// TODO remove
				os.Exit(0)

				return
			case "<Resize>":
				payload := event.Payload.(ui.Resize)
				tui.grid.SetRect(0, 0, payload.Width, payload.Height)
				ui.Clear()
				ui.Render(tui.grid)
			default:
				go func() {
					tui.slackEvents <- customEvent{kind: "message", data: fmt.Sprintf("Unhandled event: %+v", event)}
				}()
			}
		case event := <-tui.slackEvents:
			switch event.kind {
			case "message":
				tui.messagesWidget.Text += event.data + "\n"
				ui.Clear()
				ui.Render(tui.grid)
			}
		}
	}
}
