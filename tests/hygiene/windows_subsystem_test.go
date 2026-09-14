package hygiene_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseBuildsKeepGUIHostAndConsoleCLI(t *testing.T) {
	for _, name := range []string{"alpha-evidence.yml", "consumer-evidence.yml"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", name))
			if err != nil {
				t.Fatal(err)
			}
			hosts, clis := 0, 0
			for _, line := range strings.Split(string(data), "\n") {
				if !strings.Contains(line, "go build ") {
					continue
				}
				if strings.HasSuffix(strings.TrimSpace(line), "./cmd/velox-host") {
					hosts++
					if !strings.Contains(line, "-H windowsgui") {
						t.Errorf("host build must not allocate a console: %s", line)
					}
				} else if strings.HasSuffix(strings.TrimSpace(line), "./cmd/velox") {
					clis++
					if strings.Contains(line, "windowsgui") {
						t.Errorf("CLI must retain its console subsystem: %s", line)
					}
				}
			}
			if hosts == 0 || clis == 0 {
				t.Fatal("missing host or CLI build coverage")
			}
		})
	}
}
