package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	logPathEnvironment = "IMOOGI_ANKI_LOG"
	maxLogSize         = 5 << 20
)

type persistentLogger struct {
	mu     sync.Mutex
	writer io.WriteCloser
}

func openPersistentLogger() *persistentLogger {
	path := os.Getenv(logPathEnvironment)
	if path == "" {
		return &persistentLogger{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return &persistentLogger{}
	}
	if info, err := os.Stat(path); err == nil && info.Size() >= maxLogSize {
		_ = os.Remove(path + ".1")
		_ = os.Rename(path, path+".1")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return &persistentLogger{}
	}
	_ = file.Chmod(0o600)
	return &persistentLogger{writer: file}
}

func (l *persistentLogger) Close() {
	if l.writer != nil {
		_ = l.writer.Close()
	}
}

func (l *persistentLogger) Event(action string, fields map[string]any) {
	if l.writer == nil {
		return
	}
	record := make(map[string]any, len(fields)+2)
	record["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	record["event"] = action
	for key, value := range fields {
		record[key] = value
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.writer.Write(append(encoded, '\n'))
}
