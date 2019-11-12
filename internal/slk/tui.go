package slk

import (
	"fmt"
	"os"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/lk16/slk/internal/event"
	wid "github.com/lk16/slk/internal/widgets"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
)

type customEvent struct {
	kind string
	data string
}

// TUI represents the terminal UI state
type TUI struct {
	grid           *ui.Grid
	messagesWidget *widgets.Paragraph
	inputWidget    *wid.Input
	slackRTM       *slack.RTM
	events         chan event.Event
	handlers       map[string]func(event.Event)
}

// NewTUI creates a new TUI object
func NewTUI(slackRTM *slack.RTM) (*TUI, error) {
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
		ui.NewCol(0.3, p),
		ui.NewCol(0.7,
			ui.NewRow(0.8, messagesWidget),
			ui.NewRow(0.2, inputWidget),
		),
	)

	tui := &TUI{
		grid:           grid,
		messagesWidget: messagesWidget,
		inputWidget:    inputWidget,
		events:         make(chan event.Event),
		slackRTM:       slackRTM}

	tui.handlers = map[string]func(event.Event){
		"debug:":               tui.OnDebug,
		"slack:connecting":     nil,
		"slack:hello":          nil,
		"slack:latency_report": nil,
		"tui:<Backspace>":      tui.OnBackSpace,
		"tui:<C-c>":            tui.OnInterrupt,
		"tui:<Enter>":          tui.OnEnter,
		"tui:<Resize>":         tui.OnResize,
		"tui:<Space>":          tui.OnSpace,
	}

	return tui, nil
}

// OnSpace handles the space key press event
func (tui *TUI) OnSpace(e event.Event) {
	tui.inputWidget.AppendChar(" ")
}

// OnBackSpace handles the backspace key press event
func (tui *TUI) OnBackSpace(e event.Event) {
	tui.inputWidget.OnBackspace()
}

// OnInterrupt handles the ctrl+C key press event
func (tui *TUI) OnInterrupt(e event.Event) {
	ui.Close()

	// TODO remove
	os.Exit(0)
}

// OnResize handles the resize event
func (tui *TUI) OnResize(e event.Event) {
	payload := e.Data.(*ui.Event).Payload.(ui.Resize)
	tui.grid.SetRect(0, 0, payload.Width, payload.Height)

	// TODO remove:
	ui.Clear()
	ui.Render(tui.grid)
}

// OnEnter handles the enter key press event
func (tui *TUI) OnEnter(e event.Event) {
	message := tui.inputWidget.Submit()

	// discard empty messages
	if message == "" {
		return
	}

	tui.Debugf("message: %s", message)
}

// OnUnhandledEvent handles any event that is not handled by other handlers
func (tui *TUI) OnUnhandledEvent(e event.Event) {
	tui.Debugf("unhandled event %s", e.ID())
}

// SendEvent generates an event to be handled by this terminal UI
func (tui *TUI) SendEvent(e event.Event) {
	go func() {
		tui.events <- e
	}()
}

// Debugf formats and sends a debug message
func (tui *TUI) Debugf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	e := event.New(message)
	tui.SendEvent(e)
}

// OnDebug displays a debug message in the terminal UI
func (tui *TUI) OnDebug(e event.Event) {
	tui.messagesWidget.Text += fmt.Sprintf("debug: %s\n", e.Data.(string))
}

// HandleEvent handles any event
func (tui *TUI) HandleEvent(e event.Event) {
	if handler, ok := tui.handlers[e.ID()]; ok {
		if handler != nil {
			handler(e)
		}
		return
	}
	if strings.HasPrefix(e.ID(), "tui:") && !strings.HasPrefix(e.ID(), "tui:<") {
		// TODO this is a hack
		tui.inputWidget.AppendChar(e.ID()[4:])
		return
	}
	tui.OnUnhandledEvent(e)
}

// Run is the entry point of the terminal UI
func (tui *TUI) Run() {
	ui.Render(tui.grid)
	defer ui.Close()

	go func() {
		for tuiEvent := range ui.PollEvents() {
			tui.events <- event.New(&tuiEvent) 
		}
	}()

	go func() {
		for slackEvent := range tui.slackRTM.IncomingEvents {
			tui.events <- event.New(&slackEvent)
		}
	}()

	for e := range tui.events {
		tui.HandleEvent(e)
		ui.Clear()
		ui.Render(tui.grid)
	}
}
