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

type IAMUserCheck struct{}

type IAMUserCheckResult struct {
	Provider  string `json:"provider"`
	Action    string `json:"action"`
	Username  string `json:"username"`
	Password  string `json:"password,omitempty"`
	Status    string `json:"status"`
	LoginURL  string `json:"login_url,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
}

type iamUserAction struct {
	Action   string
	Username string
	Password string
}

func (p IAMUserCheck) Run(ctx context.Context, config map[string]string) {
	result, err := p.Result(ctx, config)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	iamResult, ok := result.(IAMUserCheckResult)
	if !ok {
		logger.Error("Invalid result type")
		return
	}

	if iamResult.Status == "error" {
		logger.Error(iamResult.Error)
		return
	}

	if iamResult.LoginURL != "" {
		type loginRow struct {
			Username string `table:"Username"`
			Password string `table:"Password"`
			LoginURL string `table:"Login URL"`
		}
		table.Output([]loginRow{{
			Username: iamResult.Username,
			Password: iamResult.Password,
			LoginURL: iamResult.LoginURL,
		}})
	} else {
		logger.Warning(iamResult.Message)
	}
}

func (p IAMUserCheck) Result(ctx context.Context, config map[string]string) (any, error) {
	parsed, err := parseIAMUserAction(config["metadata"])
	if err != nil {
		return nil, err
	}

	i, err := inventoryFromConfig(config)
	if err != nil {
		return nil, err
	}

	mgr, ok := i.Providers.(schema.IAMManager)
	if !ok {
		return nil, fmt.Errorf("%s does not support user management", i.Providers.Name())
	}

	iamResult, err := mgr.UserManagement(parsed.Action, parsed.Username, parsed.Password)

	result := IAMUserCheckResult{
		Provider: i.Providers.Name(),
		Action:   parsed.Action,
		Username: parsed.Username,
	}

	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result, NewResultError(result, 4, err)
	}

	result.Password = iamResult.Password
	result.LoginURL = iamResult.LoginURL
	result.AccountID = iamResult.AccountID
	result.Message = iamResult.Message

	result.Status = "success"
	return result, nil
}

func (p IAMUserCheck) Desc() string {
	return "Provision or remove a test IAM user in an authorized environment to validate identity telemetry, alerting, and persistence detection coverage."
}

func (p IAMUserCheck) Capability() string {
	return "iam"
}

func (p IAMUserCheck) Help() HelpDoc {
	return HelpDoc{
		MetadataSyntax: []string{
			"set metadata <action> <username> <password>",
			"`action` is typically `add` or `del`.",
		},
		MetadataExamples: []string{
			"set metadata add demo-user 'TempPassw0rd!'",
			"set metadata del demo-user cleanup-placeholder",
		},
		MetadataSuggestions: []Suggestion{
			{Text: "add <username> <password>", Description: "create a validation IAM user"},
			{Text: "del <username>", Description: "remove a validation IAM user"},
		},
		SafetyNotes: []string{
			"Use dedicated test identities and remove them after validation.",
			"Validate only in environments where creating or deleting IAM users is explicitly approved.",
		},
	}
}

func (p IAMUserCheck) Sensitivity(metadata string) Sensitivity {
	data := argparse.Split(metadata)
	if len(data) < 2 {
		return Sensitivity{}
	}
	return Sensitivity{
		Level:      "destructive",
		ConfirmKey: "iam-user-check." + data[0],
		Resource:   data[1],
	}
}

func parseIAMUserAction(metadata string) (iamUserAction, error) {
	data := argparse.Split(metadata)
	if len(data) < 2 {
		return iamUserAction{}, errors.New("invalid metadata format: expected 'add <username> <password>' or 'del <username>'")
	}
	action := iamUserAction{
		Action:   data[0],
		Username: data[1],
	}
	if len(data) >= 3 {
		action.Password = data[2]
	}
	if action.Action == "add" && action.Password == "" {
		return iamUserAction{}, errors.New("invalid metadata format: expected 'add <username> <password>'")
	}
	return action, nil
}

func init() {
	registerPayload("iam-user-check", IAMUserCheck{})
}
