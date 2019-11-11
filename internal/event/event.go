package event

import (
	"fmt"

	ui "github.com/gizak/termui/v3"
)

type Kind int

const (
	EventSlack = iota
	EventTUI
	EventDebug
)

type Event struct {

	// TODO
	slack interface{}

	// TODO make read only method
	Tui *ui.Event

	// TODO make read only method
	Debug string

	kind Kind
}

func FromTUI(tuiEvent *ui.Event) Event {
	return Event{
		Tui:  tuiEvent,
		kind: EventTUI,
	}
}

// TODO
// func FromSlack(slackEvent interface{}) Event {}

func FromDebug(debugEvent string) Event {
	return Event{
		Debug: debugEvent,
		kind:  EventDebug,
	}
}

func (event Event) ID() string {
	switch event.kind {
	case EventSlack:
		// TODO
		panic("not implemeneted")
	case EventTUI:
		return fmt.Sprintf("tui:%s", event.Tui.ID)
	case EventDebug:
		return "debug:"
	default:
		panic("unkown event.Event ID")
	}
}
