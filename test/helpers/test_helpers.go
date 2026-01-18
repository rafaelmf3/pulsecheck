package helpers

import (
	"context"
	"os/exec"
	"time"
)

// WaitForContainerHealth waits for a container to be healthy using docker commands
func WaitForContainerHealth(ctx context.Context, containerName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			cmd := exec.CommandContext(ctx, "docker", "ps", "--filter", "name="+containerName, "--format", "{{.Status}}")
			output, err := cmd.Output()
			if err != nil {
				continue
			}

			if len(output) > 0 {
				// Container is running
				return nil
			}
		}
	}

	return nil
}

// GetContainerLogs retrieves logs from a container using docker commands
func GetContainerLogs(ctx context.Context, containerName string, lines int) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "logs", containerName, "--tail", string(rune(lines)))
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}
