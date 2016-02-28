package sse

import (
	"errors"
	"fmt"
	"net/http"
)

type Client struct {
	eventC  chan Event
	closeC  chan bool
	running bool
	w       http.ResponseWriter
}

func ClientInit(w http.ResponseWriter) (*Client, error) {
	// Make sure that the writer supports flushing.
	if _, ok := w.(http.Flusher); !ok {
		return nil, errors.New("Streaming unsupported")
	}

	// Set Server Sent Event headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create client
	client := &Client{
		running: false,
		eventC:  make(chan Event),
		closeC:  make(chan bool),
		w:       w,
	}

	if c, ok := w.(http.CloseNotifier); ok {
		// Wait for user to close connection and close the client
		notify := c.CloseNotify()
		go func() {
			<-notify
			client.Close()
		}()
	} else {
		return nil, errors.New("Unable to start close handler")
	}

	return client, nil
}

func (client *Client) Run() {
	client.running = true // Change client run status

	// Send open event
	go client.Send(Msg{
		Event: "open",
	})

	// Wait on channels and send messages until quit
loop:
	for {
		select {
		case event := <-client.eventC:
			// Stream event
			streamData(client.w, event.SSEEvent())
		case <-client.closeC:
			// Send close event
			closeMsg := Msg{
				Event: "close",
			}
			streamData(client.w, closeMsg.SSEEvent())

			// Exit loop
			break loop
		}
	}

	client.running = false // Change client run status
}

func streamData(w http.ResponseWriter, data ...interface{}) {
	// Write to the ResponseWriter
	fmt.Fprint(w, data...)
	// Flush the data immediatly instead of buffering it for later.
	w.(http.Flusher).Flush()
}

func (client *Client) Send(event Event) {
	client.eventC <- event
}

func (client *Client) Close() {
	client.closeC <- true
}

func (client *Client) Running() bool {
	return client.running
}
