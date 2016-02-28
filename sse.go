package sse

import ()

type Event interface {
	SSEEvent() string
}

type Msg struct {
	Event string
	Data  string
	Id    string
	Retry string
}

func (msg Msg) SSEEvent() string {
	message := ""
	if msg.Event != "" {
		message += "event: " + msg.Event + "\n"
	}
	if msg.Data != "" {
		message += "data: " + msg.Data + "\n"
	}
	if msg.Id != "" {
		message += "id: " + msg.Id + "\n"
	}
	if msg.Retry != "" {
		message += "retry: " + msg.Retry + "\n"
	}
	return message + "\n"
}

type Comment string

func (comment Comment) SSEEvent() string {
	return ": " + string(comment) + "\n\n"
}
