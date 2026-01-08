package operations

import (
	"fmt"
	"io"
	"os"
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

func (l *SilentLogger) Info(msg string)                    {}
func (l *SilentLogger) Infof(format string, args ...any)   {}
func (l *SilentLogger) Warn(msg string)                    {}
func (l *SilentLogger) Warnf(format string, args ...any)   {}

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
