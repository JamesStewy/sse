package sse

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// Client represents one Server-Sent Events connection.
type Client struct {
	eventC chan Event
	done   chan struct{}
	w      http.ResponseWriter
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
		eventC: make(chan Event),
		done:   make(chan struct{}),
		w:      w,
	}, nil
}

// Run waits for, and then writes events.
// Run will only exit once the connection is closed either by the browser, or the provided context closes.
func (client *Client) Run(ctx context.Context) {
	defer close(client.done)

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
		case <-client.w.(http.CloseNotifier).CloseNotify():
			break loop
		case <-ctx.Done():
			break loop
		}
	}

	// Send close event
	closeMsg := Msg{
		Event: "close",
	}
	streamData(client.w, closeMsg.SSEEvent())
}

func streamData(w http.ResponseWriter, data ...interface{}) {
	// Write to the ResponseWriter
	fmt.Fprint(w, data...)
	// Flush the data immediatly instead of buffering it for later.
	w.(http.Flusher).Flush()
}

type key uint8

const (
	// ClientKey is used in Handler to set the client variable inside of the request context.
	// Do not set a value in a context with this key, only read.
	//
	// Example: client := r.Context().Value(sse.ClientKey).(*sse.Client)
	ClientKey key = iota
)

// Handler provides middleware for serving SSE requests.
// Handler sets the created sse client inside the request context using the key ClientKey.
// Handler will start serving events once the provided handler h exits.
// Handler exits when the client closes.
func Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client, err := ClientInit(w)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), ClientKey, client))
		h.ServeHTTP(w, r)

		client.Run(r.Context())
	})
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

// Done returns a channel which is closed when the client exits.
func (client *Client) Done() <-chan struct{} {
	return client.done
}
