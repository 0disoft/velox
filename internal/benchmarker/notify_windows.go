package benchmarker

import (
	"errors"
	"fmt"
	"os"
)

const PipeEnvironment = "VELOX_BENCH_PIPE"

func NotifyReady(phase string, browserProcessID uint32) error {
	if phase != "dom-2raf" {
		return errors.New("unexpected ready phase")
	}
	if browserProcessID == 0 {
		return errors.New("browser process ID is required")
	}
	return notify(fmt.Sprintf("ready dom-2raf %d\n", browserProcessID))
}

func NotifyPolicyAudit() error {
	return notify("ready security-ok\n")
}

func notify(marker string) error {
	pipePath := os.Getenv(PipeEnvironment)
	if pipePath == "" {
		return nil
	}

	pipe, err := os.OpenFile(pipePath, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("open benchmark pipe: %w", err)
	}
	defer pipe.Close()

	if _, err := pipe.WriteString(marker); err != nil {
		return fmt.Errorf("write benchmark marker: %w", err)
	}
	return nil
}
