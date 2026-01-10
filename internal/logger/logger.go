package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type Logger interface {
	Info(msg string)
	Infof(format string, args ...any)
	Warn(msg string)
	Warnf(format string, args ...any)
}

type StdoutLogger struct{}

func (l *StdoutLogger) Info(msg string) {
	fmt.Println(msg)
}

func (l *StdoutLogger) Infof(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func (l *StdoutLogger) Warn(msg string) {
	fmt.Fprintln(os.Stderr, "Warning: "+msg)
}

func (l *StdoutLogger) Warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Warning: "+format+"\n", args...)
}

type SilentLogger struct{}

func (l *SilentLogger) Info(msg string)                  {}
func (l *SilentLogger) Infof(format string, args ...any) {}
func (l *SilentLogger) Warn(msg string)                  {}
func (l *SilentLogger) Warnf(format string, args ...any) {}

type WriterLogger struct {
	Out io.Writer
	Err io.Writer
}

func (l *WriterLogger) Info(msg string) {
	fmt.Fprintln(l.Out, msg)
}

func (l *WriterLogger) Infof(format string, args ...any) {
	fmt.Fprintf(l.Out, format+"\n", args...)
}

func (l *WriterLogger) Warn(msg string) {
	fmt.Fprintln(l.Err, "Warning: "+msg)
}

func (l *WriterLogger) Warnf(format string, args ...any) {
	fmt.Fprintf(l.Err, "Warning: "+format+"\n", args...)
}

type FileLogger struct {
	file  *os.File
	start time.Time
}

func NewFileLogger(path string) (*FileLogger, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileLogger{file: f, start: time.Now()}, nil
}

func (l *FileLogger) Log(format string, args ...any) {
	if l.file == nil {
		return
	}
	elapsed := time.Since(l.start)
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.file, "[%s] [+%v] %s\n", time.Now().Format("15:04:05.000"), elapsed.Round(time.Microsecond), msg)
}

func (l *FileLogger) Struct(label string, v any) {
	if l.file == nil {
		return
	}
	data, err := json.MarshalIndent(v, "  ", "  ")
	if err != nil {
		l.Log("%s: (marshal error: %v)", label, err)
		return
	}
	l.Log("%s: %s", label, string(data))
}

func (l *FileLogger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}
