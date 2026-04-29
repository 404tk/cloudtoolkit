package payloads

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/argparse"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type InstanceCmdCheck struct{}

type InstanceCmdCheckResult struct {
	Provider   string `json:"provider"`
	InstanceID string `json:"instance_id"`
	Command    string `json:"command"`
	Output     string `json:"output,omitempty"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

type instanceCommand struct {
	InstanceID string
	Command    string
}

func (p InstanceCmdCheck) Run(ctx context.Context, config map[string]string) {
	resultAny, err := p.Result(ctx, config)
	if err != nil && resultAny == nil {
		logger.Error(err.Error())
		return
	}

	result, ok := resultAny.(InstanceCmdCheckResult)
	if !ok {
		logger.Error("Invalid result type")
		return
	}
	if result.Status == "error" {
		logger.Error(result.Error)
		return
	}
	if result.Output == "" {
		return
	}
	if _, err := os.Stdout.WriteString(result.Output); err != nil {
		logger.Error(err.Error())
	}
}

func (p InstanceCmdCheck) Result(ctx context.Context, config map[string]string) (any, error) {
	parsed, err := parseInstanceCommand(config["metadata"])
	if err != nil {
		return nil, err
	}

	i, err := inventoryFromConfig(config)
	if err != nil {
		return nil, err
	}
	execer, ok := i.Providers.(schema.VMExecutor)
	if !ok {
		return nil, fmt.Errorf("%s does not support instance-cmd-check", i.Providers.Name())
	}

	commandResult, err := execer.ExecuteCloudVMCommand(ctx, parsed.InstanceID, parsed.Command)
	result := InstanceCmdCheckResult{
		Provider:   i.Providers.Name(),
		InstanceID: parsed.InstanceID,
		Command:    parsed.Command,
		Output:     commandResult.Output,
	}
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result, NewResultError(result, 4, err)
	}
	result.Status = "success"
	return result, nil
}

func parseInstanceCommand(metadata string) (instanceCommand, error) {
	data := argparse.SplitN(metadata, 2)
	if len(data) < 2 {
		return instanceCommand{}, errors.New("invalid metadata format: expected '<instance-id> <cmd>'")
	}
	return instanceCommand{
		InstanceID: data[0],
		Command:    data[1],
	}, nil
}

func (p InstanceCmdCheck) Help() HelpDoc {
	return HelpDoc{
		MetadataSyntax: []string{
			"set metadata <instance-id> <cmd>",
			"`shell <instance-id>` wraps this payload and forwards all non-local input as `<cmd>`.",
		},
		MetadataExamples: []string{
			"set metadata i-1234567890abcdef0 whoami",
			"set metadata i-1234567890abcdef0 'id && hostname'",
			"shell i-1234567890abcdef0",
		},
		MetadataSuggestions: []Suggestion{
			{Text: "<instance-id> <cmd>", Description: "run one validation command; prefer `shell <instance-id>` for interactive use"},
		},
		SafetyNotes: []string{
			"Use only on instances that are owned, lab-managed, or explicitly authorized for command validation.",
			"Remember that shell mode sends non-local input to the remote instance as a validation command.",
		},
	}
}

func (p InstanceCmdCheck) Desc() string {
	return "Run an authorized validation command on a cloud instance to generate telemetry for detection and investigation verification."
}

func (p InstanceCmdCheck) Capability() string {
	return "vm"
}

func (p InstanceCmdCheck) Sensitivity(metadata string) Sensitivity {
	data := argparse.SplitN(metadata, 2)
	if len(data) < 2 {
		return Sensitivity{}
	}
	return Sensitivity{
		Level:      "destructive",
		ConfirmKey: "instance-cmd-check.exec",
		Resource:   data[0],
	}
}

func init() {
	registerPayload("instance-cmd-check", InstanceCmdCheck{})
}
