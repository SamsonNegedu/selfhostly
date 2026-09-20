package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// eventsKeepalive keeps proxies from closing a stream that is quiet.
const eventsKeepalive = 25 * time.Second

// streamEvents holds the connection open and writes one server-sent event for every change to an app, job or
// node. An event says what changed and not the new values, so the browser asks for those the normal way.
func (s *Server) streamEvents(c *gin.Context) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Streaming unsupported", Details: "Response writer does not support flush"})
		return
	}

	// Listen before telling the client it is connected, so no change made after that can be missed.
	ch, stop := s.events.Subscribe()
	defer stop()

	h := c.Writer.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	fmt.Fprint(c.Writer, ": connected\n\n")
	flusher.Flush()

	keepalive := time.NewTicker(eventsKeepalive)
	defer keepalive.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-s.shutdownCtx.Done():
			return
		case e := <-ch:
			payload, err := json.Marshal(e)
			if err != nil {
				continue
			}
			fmt.Fprintf(c.Writer, "event: change\ndata: %s\n\n", payload)
			flusher.Flush()
		case <-keepalive.C:
			fmt.Fprint(c.Writer, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}
