package payloads

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/argparse"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/table"
)

type EventCheck struct{}

type EventCheckResult struct {
	Provider string         `json:"provider"`
	Action   string         `json:"action"`
	Scope    string         `json:"scope,omitempty"`
	Events   []schema.Event `json:"events,omitempty"`
	TaskID   int64          `json:"task_id,omitempty"`
	Message  string         `json:"message,omitempty"`
	Status   string         `json:"status"`
	Error    string         `json:"error,omitempty"`
}

type eventAction struct {
	Action string
	Scope  string
}

func (p EventCheck) Run(ctx context.Context, config map[string]string) {
	resultAny, err := p.Result(ctx, config)
	if err != nil && resultAny == nil {
		logger.Error(err.Error())
		return
	}

	result, ok := resultAny.(EventCheckResult)
	if !ok {
		logger.Error("Invalid result type")
		return
	}
	if result.Status == "error" {
		logger.Error(result.Error)
		return
	}

	if len(result.Events) > 0 {
		table.Output(result.Events)
		if e := env.From(ctx); e.LogEnable {
			filename := time.Now().Format("20060102150405.log")
			path := fmt.Sprintf("%s/%s_eventdump_%s", e.LogDir, result.Provider, filename)
			table.FileOutput(path, result.Events)
			logger.Info(fmt.Sprintf("Output written to [%s]", path))
		}
	}
	if result.Message != "" {
		logger.Warning(result.Message)
	}
}

func (p EventCheck) Result(ctx context.Context, config map[string]string) (any, error) {
	parsed, err := parseEventAction(config["metadata"])
	if err != nil {
		return nil, err
	}

	i, err := inventoryFromConfig(config)
	if err != nil {
		return nil, err
	}
	reader, ok := i.Providers.(schema.EventReader)
	if !ok {
		return nil, fmt.Errorf("%s does not support event-check", i.Providers.Name())
	}

	eventResult, err := reader.EventDump(ctx, parsed.Action, parsed.Scope)
	result := EventCheckResult{
		Provider: i.Providers.Name(),
		Action:   parsed.Action,
		Scope:    parsed.Scope,
		Events:   eventResult.Events,
		TaskID:   eventResult.TaskID,
		Message:  eventResult.Message,
	}
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result, NewResultError(result, 4, err)
	}
	if result.Message == "" {
		switch {
		case result.TaskID > 0:
			result.Message = fmt.Sprintf("event handling task submitted: %d", result.TaskID)
		case parsed.Action == "dump" && len(result.Events) == 0:
			result.Message = "no events found"
		}
	}
	result.Status = "success"
	return result, nil
}

func parseEventAction(metadata string) (eventAction, error) {
	data := argparse.Split(metadata)
	if len(data) < 2 {
		return eventAction{}, errors.New("invalid metadata format: expected 'dump <source-ip|all>' or 'whitelist <security-event-id>'")
	}
	return eventAction{
		Action: data[0],
		Scope:  data[1],
	}, nil
}

func (p EventCheck) Desc() string {
	return "Review cloud security events from an authorized environment to validate alert context and investigation workflows."
}

func (p EventCheck) Capability() string {
	return "event"
}

func (p EventCheck) Help() HelpDoc {
	return HelpDoc{
		MetadataSyntax: []string{
			"set metadata dump <source-ip|all>",
			"set metadata whitelist <security-event-id>",
		},
		MetadataExamples: []string{
			"set metadata dump all",
			"set metadata dump 198.51.100.24",
			"set metadata whitelist 1234567890",
		},
		MetadataSuggestions: []Suggestion{
			{Text: "dump all", Description: "review all relevant events"},
			{Text: "dump <source-ip>", Description: "review events for one source IP"},
			{Text: "whitelist <security-event-id>", Description: "adjust one provider event handling rule where explicitly approved"},
		},
		SafetyNotes: []string{
			"Use event review in environments where log access is approved.",
			"Treat event output as investigative data and handle it according to local retention and access policies.",
			"Whitelist-style actions mutate provider-side handling and should only run with explicit approval.",
		},
	}
}

func (p EventCheck) Sensitivity(metadata string) Sensitivity {
	data := argparse.Split(metadata)
	if len(data) < 1 || data[0] != "whitelist" {
		return Sensitivity{}
	}
	resource := ""
	if len(data) >= 2 {
		resource = data[1]
	}
	return Sensitivity{
		Level:      "mutate",
		ConfirmKey: "event-check.whitelist",
		Resource:   resource,
	}
}

func init() {
	registerPayload("event-check", EventCheck{})
}
