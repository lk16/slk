package slk

import (
	"fmt"
	"os"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	wid "github.com/lk16/slk/internal/widgets"
	"github.com/pkg/errors"
)

type customEvent struct {
	kind string
	data string
}

type TUI struct {
	grid           *ui.Grid
	messagesWidget *widgets.Paragraph
	inputWidget    *wid.Input
	slackEvents    chan customEvent
	handlers       map[string]func(ui.Event)
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

	inputWidget := wid.NewInput()

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	grid.Set(
		ui.NewCol(0.5, p),
		ui.NewCol(0.5,
			ui.NewRow(0.8, messagesWidget),
			ui.NewRow(0.2, inputWidget),
		),
	)

	tui := &TUI{
		grid:           grid,
		messagesWidget: messagesWidget,
		inputWidget:    inputWidget,
		slackEvents:    make(chan customEvent)}

	tui.handlers = map[string]func(ui.Event){
		"<C-c>":       tui.OnInterrupt,
		"<Resize>":    tui.OnResize,
		"<Enter>":     tui.OnEnter,
		"<Backspace>": tui.OnBackSpace,
	}

	return tui, nil
}

func (tui *TUI) OnBackSpace(event ui.Event) {
	tui.inputWidget.OnBackspace()
}

func (tui *TUI) OnInterrupt(event ui.Event) {
	ui.Close()

	// TODO remove
	os.Exit(0)
}

func (tui *TUI) OnResize(event ui.Event) {
	payload := event.Payload.(ui.Resize)
	tui.grid.SetRect(0, 0, payload.Width, payload.Height)

	// TODO remove:
	ui.Clear()
	ui.Render(tui.grid)
}

func (tui *TUI) OnEnter(event ui.Event) {
	message := tui.inputWidget.Paragraph.Text
	tui.inputWidget.Clear()
	go func() {
		tui.slackEvents <- customEvent{
			kind: "message",
			data: message,
		}
	}()

}

func (tui *TUI) DefaultHandler(event ui.Event) {

	go func() {
		tui.slackEvents <- customEvent{
			kind: "debug",
			data: fmt.Sprintf("TUI: unhandled event with ID \"%s\"", event.ID)}
	}()
}

func (tui *TUI) HandleEvent(event ui.Event) {
	if handler, ok := tui.handlers[event.ID]; ok {
		handler(event)
		return
	}
	if !strings.HasPrefix(event.ID, "<") {
		tui.inputWidget.Append(event.ID)
		return
	}
	tui.DefaultHandler(event)
}

func (tui *TUI) Run() {
	ui.Render(tui.grid)
	defer ui.Close()

	events := ui.PollEvents()

	// TODO merge ui and slack event streams
	for {
		select {
		case event := <-events:
			tui.HandleEvent(event)
		case event := <-tui.slackEvents:
			tui.messagesWidget.Text += fmt.Sprintf("%s: %s\n", event.kind, event.data)
		}
		ui.Clear()
		ui.Render(tui.grid)
	}
}
