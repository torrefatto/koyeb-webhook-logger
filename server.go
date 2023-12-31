package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var (
	upgrader = websocket.Upgrader{}
	//go:embed index.html
	indexHtml []byte
)

// server is an HTTP server that replies on three endpoints:
//   - "/" where it serves a static html app that connects to the websocket
//   - "/logs" where it serves the websocket
//   - "/webhook" where it receives the webhooks from external services
//
// All the browser clients connected to the websocket will receive the same
// messages ingested on `/webhook`.
type server struct {
	bearer   string
	log      *zerolog.Logger
	messages chan string
	fanOut   map[int64]chan string
	mu       sync.Mutex
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.log.Info().Stringer("url", r.URL).Msg("Received a request")

	go s.run()

	switch r.URL.Path {
	case "/":
		s.serveIndex(w, r)
	case "/logs":
		s.logOutput(w, r)
	case "/webhook":
		s.webhookReceiver(w, r)
	}
}

func (s *server) serveIndex(w http.ResponseWriter, r *http.Request) {
	s.log.Info().Msg("Serving the index")
	if r.Method != http.MethodGet {
		s.log.Debug().Msg("Received a non-GET request")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("idx")
	if err == nil {
		idx := parseCookie(cookie)
		if ch := s.getListener(idx); ch == nil {
			err = http.ErrNoCookie
		}
	}
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			idx, _ := s.addListener()
			s.log.Info().Int64("idx", idx).Msg("Setting cookie")
			http.SetCookie(w, &http.Cookie{
				Name:     "idx",
				Value:    fmt.Sprint(idx),
				HttpOnly: false,
			})
		} else {
			s.log.Error().Err(err).Msg("Failed to get the cookie")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write(indexHtml)
}

func (s *server) webhookReceiver(w http.ResponseWriter, r *http.Request) {
	s.log.Info().Msg("Received a webhook")
	if r.Method != http.MethodPost {
		s.log.Debug().Msg("Received a non-POST request")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if s.bearer != "" {
		auth := r.Header.Get("Authorization")
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" || parts[1] != s.bearer {
			s.log.Debug().Msg("Received a request without a bearer token")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	body := bytes.NewBufferString("")
	if _, err := io.Copy(body, r.Body); err != nil {
		s.log.Error().Err(err).Msg("Failed to copy the body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.messages <- body.String()
}

func (s *server) logOutput(w http.ResponseWriter, r *http.Request) {
	s.log.Info().Msg("Received a log output")
	if r.Method != http.MethodGet {
		s.log.Debug().Msg("Received a non-GET request")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var idx int64

	cookie, err := r.Cookie("idx")
	if err == nil {
		idx = parseCookie(cookie)
		if idx == 0 {
			err = http.ErrNoCookie
		}
	}
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			s.log.Warn().Msg("Unauthorized")
			w.WriteHeader(http.StatusUnauthorized)
			return
		} else {
			s.log.Error().Err(err).Msg("Failed to get the cookie")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	ch := s.getListener(idx)
	if ch == nil {
		s.log.Error().Err(err).Msg("Failed to get the listener")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to upgrade the connection")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer func() {
		conn.Close()
		s.removeListener(idx)
	}()

	for msg := range ch {
		if err := s.push(conn, msg); err != nil {
			break
		}
	}
}

func (s *server) push(conn *websocket.Conn, msg string) error {
	w, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get the writer")
		return err
	}

	if _, err := io.Copy(w, strings.NewReader(msg)); err != nil {
		s.log.Error().Err(err).Msg("Failed to copy the message")
		return err
	}

	if err := w.Close(); err != nil {
		s.log.Error().Err(err).Msg("Failed to close the writer")
		return err
	}

	return nil
}

func (s *server) run() {
	for {
		select {
		case msg := <-s.messages:
			s.mu.Lock()
			for idx, ch := range s.fanOut {
				s.log.Debug().Int64("idx", idx).Msg("Forwarding the message")
				ch <- msg
			}
			s.mu.Unlock()
		}
	}
}

func (s *server) addListener() (int64, chan string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := time.Now().UnixNano()
	s.log.Info().Int64("idx", idx).Msg("Adding a listener")
	ch := make(chan string, msgBuffer)
	s.fanOut[idx] = ch
	return idx, ch
}

func (s *server) getListener(idx int64) chan string {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.log.Info().Int64("idx", idx).Msg("Getting listener")
	return s.fanOut[idx]
}

func (s *server) removeListener(idx int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.log.Info().Int64("idx", idx).Msg("Removing a listener")
	delete(s.fanOut, idx)
}

func parseCookie(cookie *http.Cookie) int64 {
	if cookie == nil {
		return 0
	}

	idx, err := strconv.ParseInt(cookie.Value, 10, 64)
	if err != nil {
		return 0
	}

	return idx
}
