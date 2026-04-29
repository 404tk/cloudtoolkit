package payloads

import (
	"context"
	"errors"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/argparse"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/table"
)

type RDSAccountCheck struct{}

type RDSAccountCheckResult struct {
	Provider   string `json:"provider"`
	Action     string `json:"action"`
	InstanceID string `json:"instance_id"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	Privilege  string `json:"privilege,omitempty"`
	Message    string `json:"message,omitempty"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

type rdsAction struct {
	Action     string
	InstanceID string
}

func (p RDSAccountCheck) Run(ctx context.Context, config map[string]string) {
	resultAny, err := p.Result(ctx, config)
	if err != nil && resultAny == nil {
		logger.Error(err.Error())
		return
	}

	result, ok := resultAny.(RDSAccountCheckResult)
	if !ok {
		logger.Error("Invalid result type")
		return
	}
	if result.Status == "error" {
		logger.Error(result.Error)
		return
	}

	if result.Username != "" {
		type accountRow struct {
			Username  string `table:"Username"`
			Password  string `table:"Password"`
			Privilege string `table:"Privilege"`
		}
		table.Output([]accountRow{{
			Username:  result.Username,
			Password:  result.Password,
			Privilege: result.Privilege,
		}})
	}
	if result.Message != "" {
		logger.Warning(result.Message)
	}
}

func (p RDSAccountCheck) Result(ctx context.Context, config map[string]string) (any, error) {
	parsed, err := parseRDSAction(config["metadata"])
	if err != nil {
		return nil, err
	}

	i, err := inventoryFromConfig(config)
	if err != nil {
		return nil, err
	}
	mgr, ok := i.Providers.(schema.DBManager)
	if !ok {
		return nil, fmt.Errorf("%s does not support rds-account-check", i.Providers.Name())
	}

	dbResult, err := mgr.DBManagement(ctx, parsed.Action, parsed.InstanceID)
	result := RDSAccountCheckResult{
		Provider:   i.Providers.Name(),
		Action:     parsed.Action,
		InstanceID: parsed.InstanceID,
		Username:   dbResult.Username,
		Password:   dbResult.Password,
		Privilege:  dbResult.Privilege,
		Message:    dbResult.Message,
	}
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result, NewResultError(result, 4, err)
	}
	result.Status = "success"
	return result, nil
}

func parseRDSAction(metadata string) (rdsAction, error) {
	data := argparse.Split(metadata)
	if len(data) < 2 {
		return rdsAction{}, errors.New("invalid metadata format: expected 'useradd <instance-id>' or 'userdel <instance-id>'")
	}
	return rdsAction{
		Action:     data[0],
		InstanceID: data[1],
	}, nil
}

func (p RDSAccountCheck) Help() HelpDoc {
	return HelpDoc{
		MetadataSyntax: []string{
			"set metadata <action> <instance-id>",
			"`action` is typically `useradd` or `userdel`.",
		},
		MetadataExamples: []string{
			"set metadata useradd rm-1234567890",
			"set metadata userdel rm-1234567890",
		},
		MetadataSuggestions: []Suggestion{
			{Text: "useradd <instance-id>", Description: "provision a validation database account"},
			{Text: "userdel <instance-id>", Description: "remove a validation database account"},
		},
		SafetyNotes: []string{
			"Run this only where creating validation database accounts is explicitly authorized.",
			"Remove temporary accounts after testing and confirm the expected database privilege scope before execution.",
		},
	}
}

func (p RDSAccountCheck) Desc() string {
	return "Provision a read-only test database account in an authorized environment to validate database telemetry, investigation readiness, and control coverage."
}

func (p RDSAccountCheck) Sensitivity(metadata string) Sensitivity {
	data := argparse.Split(metadata)
	if len(data) < 2 {
		return Sensitivity{}
	}
	return Sensitivity{
		Level:      "destructive",
		ConfirmKey: "rds-account-check." + data[0],
		Resource:   data[1],
	}
}

func init() {
	registerPayload("rds-account-check", RDSAccountCheck{})
}
