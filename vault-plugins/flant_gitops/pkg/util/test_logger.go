package util

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/helper/logging"
)

// Buffer is a goroutine safe bytes.Buffer
type Buffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (b *Buffer) Read(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Read(p)
}

func (b *Buffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Write(p)
}

func (b *Buffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.String()
}

func (b *Buffer) Reset() {
	b.m.Lock()
	defer b.m.Unlock()
	b.b.Reset()
}

func NewTestLogger() *TestLogger {
	logger := &TestLogger{}
	logger.VaultLogger = logging.NewVaultLoggerWithWriter(io.MultiWriter(&logger.BytesBuffer, hclog.DefaultOutput), hclog.Trace)
	return logger
}

type TestLogger struct {
	Lines       []string
	BytesBuffer Buffer

	VaultLogger hclog.Logger
}

func (logger *TestLogger) Reset() {
	logger.BytesBuffer.Reset()
	logger.Lines = nil
}

func (logger *TestLogger) GetLines() []string {
	scanner := bufio.NewScanner(&logger.BytesBuffer)
	scanner.Split(bufio.ScanLines)

	var newLines []string
	for scanner.Scan() {
		newLines = append(newLines, scanner.Text())
	}
	logger.Lines = append(logger.Lines, newLines...)
	var result []string
	result = append(result, logger.Lines...)
	return result
}

func (logger *TestLogger) Grep(text string) (bool, []string) {
	var matchedLines []string

	for _, line := range logger.GetLines() {
		if strings.Contains(line, text) {
			matchedLines = append(matchedLines, line)
		}
	}

	return len(matchedLines) > 0, matchedLines
}

func (logger *TestLogger) GetDataByMarkers(beginMark, endMark string) (bool, []byte) {
	var matchedLines []string

	state := ""
	for _, line := range logger.GetLines() {
		switch state {
		case "":
			if strings.Contains(line, beginMark) {
				state = "beginMark"
			}
		case "beginMark":
			if strings.Contains(line, endMark) {
				return true, []byte(strings.Join(matchedLines, "\n"))
			}
			matchedLines = append(matchedLines, line)
		default:
			panic("unexpected")
		}
	}
	return false, nil
}
