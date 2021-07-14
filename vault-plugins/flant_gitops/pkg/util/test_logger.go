package util

import (
	"bufio"
	"bytes"
	"io"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/helper/logging"
)

func NewTestLogger() *TestLogger {
	logger := &TestLogger{}
	logger.VaultLogger = logging.NewVaultLoggerWithWriter(io.MultiWriter(&logger.BytesBuffer, hclog.DefaultOutput), hclog.Trace)
	return logger
}

type TestLogger struct {
	Lines       []string
	BytesBuffer bytes.Buffer

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

	return logger.Lines
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
