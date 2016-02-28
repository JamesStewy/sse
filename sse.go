/*
Package sse provides Server-Sent Events.

Server-Sent Events on MDN: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events.

Example

This example binds the root of localhost:8080 to a Server-Sent Event stream that sends the time every two seconds.

	package main

	import (
		"github.com/JamesStewy/sse"
		"net/http"
		"time"
	)

	var clients map[*sse.Client]bool

	func main() {
		clients = make(map[*sse.Client]bool)

		// Asynchronously send event to all clients every two seconds
		ticker := time.NewTicker(time.Second * 2)
		go func() {
			for t := range ticker.C {
				// Create new message called 'time' with data containing the current time
				msg := sse.Msg{
					Event: "time",
					Data:  t.String(),
				}

				for client := range clients {
					// Send the message to this client
					client.Send(msg)
				}
			}
		}()

		// Start HTTP server
		http.HandleFunc("/", eventHandler)
		http.ListenAndServe("localhost:8080", nil)
	}

	func eventHandler(w http.ResponseWriter, req *http.Request) {
		// Initialise (REQUIRED)
		client, err := sse.ClientInit(w)

		// Return error if unable to initialise Server-Sent Events
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Add client to external variable for later use
		clients[client] = true
		// Remove client from external variable on exit
		defer delete(clients, client)

		// Run Client (REQUIRED)
		client.Run()
	}
*/
package sse

import ()

// Event represents a single event that can be sent to a SSE Client.
type Event interface {
	SSEEvent() string
}

// Msg is a Server-Sent event message.
// More information on MDN: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events#Event_stream_format.
//
// If a field is left as an empty string, that field will not be sent to the Client.
type Msg struct {
	// The Event field is the name of the message.
	// If specified this message will be despatched to a listener for this name.
	Event string

	// The Data field is the payload of the message.
	Data  string

	// The Id field sets the last event ID value in the EventSource object.
	Id    string

	Retry string
}

// Convert the message into a Server-Sent Event formatted string that can be written to a connection.
// All empty fields are not included.
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

// Comment is a Server-Sent event comment.
// Comments are not interpreted by the EventSource object in the browser.
type Comment string

// Convert the comment into a Server-Sent Event formatted string that can be written to a connection.
func (comment Comment) SSEEvent() string {
	return ": " + string(comment) + "\n\n"
}
