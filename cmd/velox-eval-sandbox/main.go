package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/0disoft/velox/internal/buildinfo"
	"github.com/0disoft/velox/internal/evalsandbox"
)

type rootsFlag []string

func (value *rootsFlag) String() string {
	return strings.Join(*value, string(os.PathListSeparator))
}

func (value *rootsFlag) Set(root string) error {
	if root == "" {
		return fmt.Errorf("tool root cannot be empty")
	}
	*value = append(*value, root)
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "velox-eval-sandbox: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("velox-eval-sandbox", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var toolRoots rootsFlag
	var passEnvironment rootsFlag
	trialID := flags.String("trial-id", "", "immutable trial identity")
	seriesID := flags.String("series-id", "", "immutable series identity")
	sequence := flags.Int("sequence", 0, "trial sequence from 1 to 3")
	trialRoot := flags.String("trial-root", "", "absolute writable trial root")
	receiptPath := flags.String("receipt", "", "exclusive output path outside the trial root")
	timeout := flags.Duration("timeout", 45*time.Minute, "maximum evaluator lifetime")
	jsonOutput := flags.Bool("json", false, "write a compact success result")
	sessionIDEnvironment := flags.String("session-id-env", "VELOX_HERMES_SESSION_ID", "environment variable containing the evaluator session ID")
	version := flags.Bool("version", false, "print the supervisor version")
	flags.Var(&toolRoots, "tool-root", "absolute read-execute tool root; repeatable")
	flags.Var(&passEnvironment, "pass-env", "explicitly forward one existing environment variable; repeatable")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *version {
		fmt.Println(buildinfo.Version)
		return nil
	}
	command := flags.Args()
	if len(command) == 0 {
		return fmt.Errorf("a command is required after --")
	}
	receipt, err := evalsandbox.Run(evalsandbox.Config{
		TrialID:              *trialID,
		SeriesID:             *seriesID,
		Sequence:             *sequence,
		TrialRoot:            *trialRoot,
		ToolRoots:            toolRoots,
		PassEnvironment:      passEnvironment,
		SessionIDEnvironment: *sessionIDEnvironment,
		ReceiptPath:          *receiptPath,
		Timeout:              *timeout,
		Command:              command,
	})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{
			"schemaVersion": receipt.SchemaVersion,
			"trialId":       receipt.TrialID,
			"sequence":      strconv.Itoa(receipt.Sequence),
			"status":        "passed",
		})
	}
	fmt.Println(evalsandbox.SafeSummary(receipt))
	return nil
}
