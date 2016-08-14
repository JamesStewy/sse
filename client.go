package sse

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Client represents one Server-Sent Events connection.
type Client struct {
	eventC    chan Event
	done      chan struct{}
	close     chan struct{}
	timeout   time.Duration
	w         http.ResponseWriter
}

// ClientInit initialises an HTTP connection for Server-Sent Events.
//
// Timeout is how long the connection will stay open for.
// Timeout starts when client is run.
// Set timeout to 0 to keep the connection open indefinitely.
//
// ClientInit first checks that data streaming is supported on the connection.
// ClientInit then sets the appropriate HTTP headers for Server-Sent Events.
// Finaly ClientInit returns a Client that can later be run and have events sent too.
//
// Headers: Content-Type=text/event-stream, Cache-Control=no-cache, Connection=keep-alive.
func ClientInit(w http.ResponseWriter, timeout time.Duration) (*Client, error) {
	// Make sure that the writer supports flushing.
	if _, ok := w.(http.Flusher); !ok {
		return nil, errors.New("streaming not supported")
	}

	// Make sure that the writer supports close notifiers.
	if _, ok := w.(http.CloseNotifier); !ok {
		return nil, errors.New("unable to start close handler")
	}

	// Set Server Sent Event headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create client
	return &Client{
		eventC:  make(chan Event),
		done:    make(chan struct{}),
		close:   make(chan struct{}),
		timeout: timeout,
		w:       w,
	}, nil
}

// Run waits for, and then writes events.
// Run will only exit once the connection is closed either by the browser, by running client.Close() or the timeout is reached.
func (client *Client) Run() {
	timeoutC := time.After(client.timeout)
	if client.timeout <= time.Duration(0) {
		timeoutC = nil
	}

	// Send open event
	openMsg := Msg{
		Event: "open",
	}
	streamData(client.w, openMsg.SSEEvent())

	// Wait on channels and send messages until quit
loop:
	for {
		select {
		case event := <-client.eventC:
			streamData(client.w, event.SSEEvent()) // Stream event
		case <-client.close:
			break loop
		case <-client.w.(http.CloseNotifier).CloseNotify():
			break loop
		case <-timeoutC:
			break loop
		}
	}

	// Send close event
	closeMsg := Msg{
		Event: "close",
	}
	streamData(client.w, closeMsg.SSEEvent())

	// Close done channel to indicate that the client is closed
	close(client.done)
}

func streamData(w http.ResponseWriter, data ...interface{}) {
	// Write to the ResponseWriter
	fmt.Fprint(w, data...)
	// Flush the data immediatly instead of buffering it for later.
	w.(http.Flusher).Flush()
}

// Send an event to the client.
// Send will only exit once the event has been sent.
func (client *Client) Send(event Event) error {
	select {
	case client.eventC <- event:
	case <-client.done:
		return errors.New("message sent on closed client")
	}
	return nil
}

// Close the connection.
// Close will only exit once the connection is fully closed.
func (client *Client) Close() {
	for {
		select {
		case client.close <- struct{}{}:
		case <-client.done:
			return
		}
	}
}
