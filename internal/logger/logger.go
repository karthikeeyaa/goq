package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"goq/internal/config"
)

type Logger struct {
	file *os.File
	writer io.Writer
}

func Init(cfg *config.Config) (*Logger, error) {
	logPath := cfg.LogFile

	dir := filepath.Dir(logPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory %q: %w", dir, err)
		}
	}

	var err error
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %q: %w", logPath, err)
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	return &Logger{file: logFile, writer: multiWriter}, nil
}

func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) Info(format string, v ...interface{}) {
	if l.writer != nil {
		fmt.Fprintf(l.writer, "[INFO] "+format+"\n", v...)
	}
}

func (l *Logger) Warn(format string, v ...interface{}) {
	if l.writer != nil {
		fmt.Fprintf(l.writer, "[WARN] "+format+"\n", v...)
	}
}

func (l *Logger) Error(format string, v ...interface{}) {
	if l.writer != nil {
		fmt.Fprintf(l.writer, "[ERROR] "+format+"\n", v...)
	}
}