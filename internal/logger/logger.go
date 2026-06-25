package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

var (
	InfoLogger  *log.Logger
	WarnLogger  *log.Logger
	ErrorLogger *log.Logger
	logFile     *os.File
)

func Init(logPath string) error {
	dir := filepath.Dir(logPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory %q: %w", dir, err)
		}
	}

	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %q: %w", logPath, err)
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	InfoLogger = log.New(multiWriter, "[INFO]  ", log.LstdFlags|log.Lshortfile)
	WarnLogger = log.New(multiWriter, "[WARN]  ", log.LstdFlags|log.Lshortfile)
	ErrorLogger = log.New(multiWriter, "[ERROR] ", log.LstdFlags|log.Lshortfile)

	return nil
}

func Close() error {
	if logFile != nil {
		return logFile.Close()
	}
	return nil
}

func Info(format string, v ...interface{}) {
	if InfoLogger != nil {
		InfoLogger.Output(2, fmt.Sprintf(format, v...))
	} else {
		log.Printf("[INFO] "+format, v...)
	}
}

func Warn(format string, v ...interface{}) {
	if WarnLogger != nil {
		WarnLogger.Output(2, fmt.Sprintf(format, v...))
	} else {
		log.Printf("[WARN] "+format, v...)
	}
}

func Error(format string, v ...interface{}) {
	if ErrorLogger != nil {
		ErrorLogger.Output(2, fmt.Sprintf(format, v...))
	} else {
		log.Printf("[ERROR] "+format, v...)
	}
}