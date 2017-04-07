// Copyright 2017 Ross Peoples
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mydis

import (
	"bufio"
	"context"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

// WebsocketProxy attempts to expose the underlying handler as a bidirectional websocket stream with
// newline-delimited JSON as the content encoding.
//
// The HTTP Authorization header is populated from the Sec-Websocket-Protocol field.
//
// example:
//   Sec-Websocket-Protocol: Bearer, foobar
// is converted to:
//   Authorization: Bearer foobar
func WebsocketProxy(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !websocket.IsWebSocketUpgrade(r) {
			h.ServeHTTP(w, r)
			return
		}
		websocketProxy(w, r, h)
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func websocketProxy(w http.ResponseWriter, r *http.Request, h http.Handler) {
	var responseHeader http.Header
	// If Sec-WebSocket-Protocol starts with "Bearer", respond in kind.
	if strings.HasPrefix(r.Header.Get("Sec-WebSocket-Protocol"), "Bearer") {
		responseHeader = http.Header{
			"Sec-WebSocket-Protocol": []string{"Bearer"},
		}
	}
	conn, err := upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		log.Println("error upgrading websocket:", err)
		return
	}
	defer conn.Close()

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	requestBodyR, requestBodyW := io.Pipe()
	request, err := http.NewRequest("POST", r.URL.String(), requestBodyR)
	if err != nil {
		log.Println("error preparing request:", err)
		return
	}
	if swsp := r.Header.Get("Sec-WebSocket-Protocol"); swsp != "" {
		request.Header.Set("Authorization", strings.Replace(swsp, "Bearer, ", "Bearer ", 1))
	}
	if m := r.URL.Query().Get("method"); m != "" {
		request.Method = m
	}

	responseBodyR, responseBodyW := io.Pipe()
	go func() {
		<-ctx.Done()
		log.Println("closing pipes")
		requestBodyW.CloseWithError(io.EOF)
		responseBodyW.CloseWithError(io.EOF)
	}()

	response := newResponseWriter(responseBodyW)
	go func() {
		defer cancelFn()
		h.ServeHTTP(response, request)
	}()

	// read loop -- take messages from websocket and write to http request
	go func() {
		defer func() {
			cancelFn()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_, p, err := conn.ReadMessage()
			if err != nil {
				log.Println("error reading websocket message:", err)
				return
			}

			_, err = requestBodyW.Write(p)
			requestBodyW.Write([]byte("\n"))

			if err != nil {
				log.Println("[read] error writing message to upstream http server:", err)
				return
			}
		}
	}()

	// write loop -- take messages from response and write to websocket
	scanner := bufio.NewScanner(responseBodyR)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			log.Println("[write] empty scan", scanner.Err())
			continue
		}

		if err = conn.WriteMessage(websocket.TextMessage, scanner.Bytes()); err != nil {
			log.Println("[write] error writing websocket message:", err)
			return
		}
	}

	if err := scanner.Err(); err != nil {
		log.Println("scanner err:", err)
	}
}

type responseWriter struct {
	io.Writer
	header http.Header
	code   int
}

func newResponseWriter(w io.Writer) *responseWriter {
	return &responseWriter{
		Writer: w,
		header: http.Header{},
	}
}

func (w *responseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}
func (w *responseWriter) Header() http.Header {
	return w.header
}
func (w *responseWriter) WriteHeader(code int) {
	w.code = code
}
func (w *responseWriter) Flush() {}
