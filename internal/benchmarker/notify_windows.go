package benchmarker

import (
	"errors"
	"fmt"
	"os"
)

const PipeEnvironment = "VELOX_BENCH_PIPE"

func NotifyReady(phase string) error {
	pipePath := os.Getenv(PipeEnvironment)
	if pipePath == "" {
		return nil
	}
	if phase != "dom-2raf" {
		return errors.New("unexpected ready phase")
	}

	pipe, err := os.OpenFile(pipePath, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("open benchmark pipe: %w", err)
	}
	defer pipe.Close()

	if _, err := pipe.WriteString("ready dom-2raf\n"); err != nil {
		return fmt.Errorf("write benchmark marker: %w", err)
	}
	return nil
}
