package tests

import (
	"bufio"
	"bytes"
	"fmt"

	"github.com/docker/docker/pkg/stdcopy"
)

func ParseDockerOutput(reader *bufio.Reader) ([]byte, error) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	_, err := stdcopy.StdCopy(&out, &errOut, reader)
	if err != nil {
		return nil, fmt.Errorf("reading docker exec output: %w", err)
	}
	if errOut.Len() > 0 {
		return nil, fmt.Errorf("docker exec: %s", errOut.String())
	}
	return out.Bytes(), nil
}
