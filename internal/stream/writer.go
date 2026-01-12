package stream

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type LogMessage struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Stream string `json:"stream"`
	Data   string `json:"data"`
}

type CompleteMessage struct {
	Type        string       `json:"type"`
	Success     bool         `json:"success"`
	Environment *Environment `json:"environment,omitempty"`
	Error       string       `json:"error,omitempty"`
}

type Environment struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Branch string `json:"branch"`
	Path   string `json:"path"`
	Mode   string `json:"mode"`
	Status string `json:"status"`
}

type StreamWriter struct {
	conn       *websocket.Conn
	source     string
	stream     string
	mu         sync.Mutex
	buf        bytes.Buffer
	writeDelay time.Duration
	tee        io.Writer
}

func NewStreamWriter(conn *websocket.Conn, source, stream string) *StreamWriter {
	return &StreamWriter{
		conn:       conn,
		source:     source,
		stream:     stream,
		writeDelay: 5 * time.Second,
	}
}

func (w *StreamWriter) WithTee(tee io.Writer) *StreamWriter {
	w.tee = tee
	return w
}

func (w *StreamWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.tee != nil {
		w.tee.Write(p)
	}

	w.buf.Write(p)

	for {
		line, err := w.buf.ReadBytes('\n')
		if err != nil {
			w.buf.Write(line)
			break
		}
		if sendErr := w.sendLine(string(line)); sendErr != nil {
			return len(p), nil
		}
	}

	return len(p), nil
}

func (w *StreamWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.buf.Len() == 0 {
		return nil
	}

	remaining := w.buf.String()
	w.buf.Reset()
	return w.sendLine(remaining)
}

func (w *StreamWriter) sendLine(data string) error {
	if data == "" {
		return nil
	}

	msg := LogMessage{
		Type:   "log",
		Source: w.source,
		Stream: w.stream,
		Data:   data,
	}

	encoded, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	w.conn.SetWriteDeadline(time.Now().Add(w.writeDelay))
	return w.conn.WriteMessage(websocket.TextMessage, encoded)
}

func SendComplete(conn *websocket.Conn, env *Environment) error {
	msg := CompleteMessage{
		Type:        "complete",
		Success:     true,
		Environment: env,
	}
	encoded, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return conn.WriteMessage(websocket.TextMessage, encoded)
}

func SendError(conn *websocket.Conn, errMsg string) error {
	msg := CompleteMessage{
		Type:    "complete",
		Success: false,
		Error:   errMsg,
	}
	encoded, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return conn.WriteMessage(websocket.TextMessage, encoded)
}

type MultiWriter struct {
	writers []io.Writer
}

func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{writers: writers}
}

func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range mw.writers {
		w.Write(p)
	}
	return len(p), nil
}

type WriterFactory struct {
	conn *websocket.Conn
	tee  io.Writer
}

func NewWriterFactory(conn *websocket.Conn, tee io.Writer) *WriterFactory {
	return &WriterFactory{conn: conn, tee: tee}
}

func (f *WriterFactory) NewWriter(source, stream string) *StreamWriter {
	sw := NewStreamWriter(f.conn, source, stream)
	if f.tee != nil {
		sw.WithTee(f.tee)
	}
	return sw
}

func (f *WriterFactory) Git() (stdout, stderr *StreamWriter) {
	return f.NewWriter("git", "stdout"), f.NewWriter("git", "stderr")
}

func (f *WriterFactory) Docker() (stdout, stderr *StreamWriter) {
	return f.NewWriter("docker", "stdout"), f.NewWriter("docker", "stderr")
}

func (f *WriterFactory) Prepare() (stdout, stderr *StreamWriter) {
	return f.NewWriter("script:prepare", "stdout"), f.NewWriter("script:prepare", "stderr")
}

func (f *WriterFactory) Setup() (stdout, stderr *StreamWriter) {
	return f.NewWriter("script:setup", "stdout"), f.NewWriter("script:setup", "stderr")
}

func (f *WriterFactory) Piko() *StreamWriter {
	return f.NewWriter("piko", "stdout")
}
