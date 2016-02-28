package sse

import (
	"errors"
	"fmt"
	"net/http"
)

// Client represents one Server-Sent Events connection.
type Client struct {
	eventC  chan Event
	closeC  chan bool
	running bool
	w       http.ResponseWriter
}

// ClientInit initialises an HTTP connection for Server-Sent Events.
//
// ClientInit first checks that data streaming is supported on the connection.
// ClientInit then sets the appropriate HTTP headers for Server-Sent Events.
// Finaly ClientInit returns a Client that can later be run and have events sent too.
//
// Headers: Content-Type=text/event-stream, Cache-Control=no-cache, Connection=keep-alive.
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

// Run waits for, and then writes events.
// Run will only exit once the connection is closed either by the browser or by running client.Close().
func (client *Client) Run() {
	client.running = true

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

	client.running = false
	// Send close message back to indicate close completion
	client.closeC <- true
}

func streamData(w http.ResponseWriter, data ...interface{}) {
	// Write to the ResponseWriter
	fmt.Fprint(w, data...)
	// Flush the data immediatly instead of buffering it for later.
	w.(http.Flusher).Flush()
}

// Send an event to the client.
// Send will only exit once the event has been sent.
func (client *Client) Send(event Event) {
	client.eventC <- event
}

// Close the connection.
// Close will only exit once the connection is fully closed.
func (client *Client) Close() {
	client.closeC <- true
	// Wait until close has completed
	<-client.closeC
}

// Running will return whether or not the Client is currently accepting and writing events.
func (client *Client) Running() bool {
	return client.running
}
