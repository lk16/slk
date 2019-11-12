package event

import (
	"fmt"

	ui "github.com/gizak/termui/v3"
	"github.com/nlopes/slack"
)


// Event is a very generic event
type Event struct {
	Data interface{}
}

// New creates a new Event
func New(i interface{}) Event{
	return Event{Data: i}
}

// ID returns a unique string per event type
func (event Event) ID() string {
	
	switch data := event.Data.(type) {
	case *slack.RTMEvent:
		return fmt.Sprintf("slack:%s", data.Type)
	case *ui.Event:
		return fmt.Sprintf("tui:%s", data.ID)
	case string:
		return "debug:"
	default:
		return fmt.Sprintf("unknown_event_type_%T", event)
	}
}
