package slk

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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
	grid             *ui.Grid
	chatWidget       *wid.Chat
	channelsWidget   *widgets.Paragraph
	inputWidget      *wid.Input
	slackRTM         *slack.RTM
	events           chan event.Event
	handlers         map[string]func(event.Event)
	channels         map[string]slack.Channel
	activeChannelKey string
}

// NewTUI creates a new TUI object
func NewTUI(slackRTM *slack.RTM) (*TUI, error) {
	if err := ui.Init(); err != nil {
		return nil, errors.Wrap(err, "failed to initialize termui")
	}

	channelsWidget := widgets.NewParagraph()
	channelsWidget.Title = "Channels"

	chatWidget := wid.NewChat()
	chatWidget.Title = "Messages"

	inputWidget := wid.NewInput()

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	grid.Set(
		ui.NewCol(0.3, channelsWidget),
		ui.NewCol(0.7,
			ui.NewRow(0.8, chatWidget),
			ui.NewRow(0.2, inputWidget),
		),
	)

	tui := &TUI{
		grid:           grid,
		chatWidget:     chatWidget,
		inputWidget:    inputWidget,
		channelsWidget: channelsWidget,
		events:         make(chan event.Event),
		slackRTM:       slackRTM,
	}

	tui.handlers = map[string]func(event.Event){
		"debug:":               tui.OnDebug,
		"slack:connecting":     nil,
		"slack:hello":          nil,
		"slack:latency_report": nil,
		"slk:list_channels":    tui.OnListChannels,
		"slk:list_users":       nil,
		"tui:<Backspace>":      tui.OnBackSpace,
		"tui:<C-c>":            tui.Shutdown,
		"tui:<Escape>":         tui.Shutdown,
		"tui:<Enter>":          tui.OnEnter,
		"tui:<Resize>":         tui.OnResize,
		"tui:<Space>":          tui.OnSpace,
	}

	return tui, nil
}

// OnListChannels handles the event that the list of channels are loaded
func (tui *TUI) OnListChannels(e event.Event) {

	tui.channels = e.Data.(map[string]slack.Channel)
	var channelList bytes.Buffer

	for _, channel := range e.Data.(map[string]slack.Channel) {
		if channel.IsMember {
			_, _ = channelList.WriteString(fmt.Sprintf("#%s\n", channel.Name))
		}
	}

	tui.channelsWidget.Text = channelList.String()
}

// OnSpace handles the space key press event
func (tui *TUI) OnSpace(e event.Event) {
	tui.inputWidget.AppendChar(" ")
}

// OnBackSpace handles the backspace key press event
func (tui *TUI) OnBackSpace(e event.Event) {
	tui.inputWidget.OnBackspace()
}

// Shutdown shuts down the terminal UI
func (tui *TUI) Shutdown(e event.Event) {
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

// OnCommand handles any command entered in chat
// TODO this entire function is hacked up
func (tui *TUI) OnCommand(message string) {

	split := strings.Split(message, " ")
	command := split[0]
	args := split[1:]

	if command == "/join" {
		if len(args) != 1 {
			tui.Debugf("failed to process command, need 1 argument")
			return
		}
		for channelKey, channel := range tui.channels {
			tui.Debugf("Comparing %s %s", fmt.Sprintf("#%s", channel.Name), args[0])
			if fmt.Sprintf("#%s", channel.Name) == args[0] {
				tui.switchChannel(channelKey)
				return
			}
		}
		tui.Debugf("channel not found: #%s", args[0])
		return
	}

	tui.Debugf("unprocessed command: %s", message)
}

// switchChannel handles switching channels and loading history
func (tui *TUI) switchChannel(channelKey string) {

	channelName := tui.channels[channelKey].Name
	tui.Debugf("switching to channel #%s with key %s", channelName, channelKey)
	tui.activeChannelKey = channelKey

	// TODO use goroutine here
	history, err := tui.slackRTM.GetChannelHistory(channelKey, slack.NewHistoryParameters())
	if err != nil {
		tui.Debugf("could not load history: %s", err.Error())
		return
	}

	tui.chatWidget.Clear()

	for _, message := range history.Messages {
		timestampFloat, err := strconv.ParseFloat(message.Timestamp, 64)
		if err != nil {
			// TODO log conversion error
			continue
		}

		timestamp := time.Unix(int64(timestampFloat), 0)

		tui.chatWidget.AddMessage(wid.Message{
			Sender:    message.Username,
			Text:      message.Text,
			Timestamp: timestamp,
		})
	}

}

// OnEnter handles the enter key press event
func (tui *TUI) OnEnter(e event.Event) {
	message := tui.inputWidget.Submit()

	// discard empty messages
	if message == "" {
		return
	}

	if strings.HasPrefix(message, "/") {
		tui.OnCommand(message)
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
	message := wid.Message{
		Timestamp: time.Now(),
		Sender:    "debug",
		Text:      e.Data.(string),
	}

	tui.chatWidget.AddMessage(message)
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
