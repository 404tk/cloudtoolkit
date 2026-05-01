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

type IAMCredentialCheck struct{}

type IAMCredentialCheckResult struct {
	Provider       string                 `json:"provider"`
	Action         string                 `json:"action"`
	Principal      string                 `json:"principal,omitempty"`
	CredentialID   string                 `json:"credential_id,omitempty"`
	CredentialData string                 `json:"credential_data,omitempty"`
	Credentials    []iamCredentialRowJSON `json:"credentials,omitempty"`
	Message        string                 `json:"message,omitempty"`
	Status         string                 `json:"status"`
	Error          string                 `json:"error,omitempty"`
}

type iamCredentialRowJSON struct {
	CredentialID   string `json:"credential_id"`
	CredentialType string `json:"credential_type,omitempty"`
	ValidAfter     string `json:"valid_after,omitempty"`
	ValidBefore    string `json:"valid_before,omitempty"`
}

type iamCredentialAction struct {
	Action       string
	Principal    string
	CredentialID string
}

func (p IAMCredentialCheck) Run(ctx context.Context, config map[string]string) {
	resultAny, err := p.Result(ctx, config)
	if err != nil && resultAny == nil {
		logger.Error(err.Error())
		return
	}
	result, ok := resultAny.(IAMCredentialCheckResult)
	if !ok {
		logger.Error("Invalid result type")
		return
	}
	if result.Status == "error" {
		logger.Error(result.Error)
		return
	}

	if result.CredentialData != "" {
		type keyRow struct {
			Principal      string `table:"Principal"`
			CredentialID   string `table:"Credential ID"`
			CredentialData string `table:"Credential Data"`
		}
		table.Output([]keyRow{{
			Principal:      result.Principal,
			CredentialID:   result.CredentialID,
			CredentialData: result.CredentialData,
		}})
	} else if len(result.Credentials) > 0 {
		type keyRow struct {
			CredentialID   string `table:"Credential ID"`
			CredentialType string `table:"Type"`
			ValidAfter     string `table:"Valid After"`
			ValidBefore    string `table:"Valid Before"`
		}
		rows := make([]keyRow, 0, len(result.Credentials))
		for _, k := range result.Credentials {
			rows = append(rows, keyRow{
				CredentialID:   k.CredentialID,
				CredentialType: k.CredentialType,
				ValidAfter:     k.ValidAfter,
				ValidBefore:    k.ValidBefore,
			})
		}
		table.Output(rows)
	}
	if result.Message != "" {
		logger.Warning(result.Message)
	}
}

func (p IAMCredentialCheck) Result(ctx context.Context, config map[string]string) (any, error) {
	parsed, err := parseIAMCredentialAction(config["metadata"])
	if err != nil {
		return nil, err
	}

	i, err := inventoryFromConfig(config)
	if err != nil {
		return nil, err
	}

	mgr, ok := i.Providers.(schema.IAMCredentialManager)
	if !ok {
		return nil, fmt.Errorf("%s does not support iam-credential-check", i.Providers.Name())
	}

	credResult, err := mgr.IAMCredential(ctx, parsed.Action, parsed.Principal, parsed.CredentialID)

	result := IAMCredentialCheckResult{
		Provider:     i.Providers.Name(),
		Action:       parsed.Action,
		Principal:    parsed.Principal,
		CredentialID: parsed.CredentialID,
	}
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result, NewResultError(result, 4, err)
	}

	if credResult.Principal != "" {
		result.Principal = credResult.Principal
	}
	if credResult.CredentialID != "" {
		result.CredentialID = credResult.CredentialID
	}
	result.CredentialData = credResult.CredentialData
	result.Message = credResult.Message
	for _, k := range credResult.Credentials {
		result.Credentials = append(result.Credentials, iamCredentialRowJSON{
			CredentialID:   k.CredentialID,
			CredentialType: k.CredentialType,
			ValidAfter:     k.ValidAfter,
			ValidBefore:    k.ValidBefore,
		})
	}
	result.Status = "success"
	return result, nil
}

func (p IAMCredentialCheck) Desc() string {
	return "Create or revoke a long-lived IAM credential in an authorized environment to validate detection coverage for credential lifecycle abuse."
}

func (p IAMCredentialCheck) Capability() string {
	return "iam-credential"
}

func (p IAMCredentialCheck) Help() HelpDoc {
	return HelpDoc{
		MetadataSyntax: []string{
			"set metadata <action> <principal> [credential-id]",
			"`action` is typically `list`, `create`, or `delete`.",
		},
		MetadataExamples: []string{
			"set metadata list ctk-demo@ctk-demo-project.iam.gserviceaccount.com",
			"set metadata create ctk-demo@ctk-demo-project.iam.gserviceaccount.com",
			"set metadata delete ctk-demo@ctk-demo-project.iam.gserviceaccount.com 123abc",
		},
		MetadataSuggestions: []Suggestion{
			{Text: "list <principal>", Description: "enumerate long-lived credentials for one principal"},
			{Text: "create <principal>", Description: "mint a validation credential for one principal"},
			{Text: "delete <principal> <credential-id>", Description: "revoke a validation credential"},
		},
		SafetyNotes: []string{
			"Long-lived IAM credentials are sensitive; revoke or rotate any validation material after use.",
			"Principal syntax is provider-specific. Today GCP uses a service-account email address.",
			"Run only where minting IAM credentials is explicitly authorized.",
		},
	}
}

func (p IAMCredentialCheck) Sensitivity(metadata string) Sensitivity {
	data := argparse.Split(metadata)
	if len(data) == 0 {
		return Sensitivity{}
	}
	switch data[0] {
	case "create", "delete":
	default:
		return Sensitivity{}
	}
	resource := ""
	if len(data) >= 2 {
		resource = data[1]
	}
	return Sensitivity{
		Level:      "destructive",
		ConfirmKey: "iam-credential-check." + data[0],
		Resource:   resource,
	}
}

func parseIAMCredentialAction(metadata string) (iamCredentialAction, error) {
	data := argparse.Split(metadata)
	if len(data) == 0 {
		return iamCredentialAction{}, errors.New("invalid metadata format: expected 'list <principal>', 'create <principal>', or 'delete <principal> <credential-id>'")
	}
	action := iamCredentialAction{Action: data[0]}
	switch action.Action {
	case "list":
		if len(data) < 2 {
			return iamCredentialAction{}, errors.New("invalid metadata format: expected 'list <principal>'")
		}
		action.Principal = data[1]
	case "create":
		if len(data) < 2 {
			return iamCredentialAction{}, errors.New("invalid metadata format: expected 'create <principal>'")
		}
		action.Principal = data[1]
	case "delete":
		if len(data) < 3 {
			return iamCredentialAction{}, errors.New("invalid metadata format: expected 'delete <principal> <credential-id>'")
		}
		action.Principal = data[1]
		action.CredentialID = data[2]
	default:
		return iamCredentialAction{}, fmt.Errorf("unsupported action %q: expected list / create / delete", action.Action)
	}
	return action, nil
}

func init() {
	registerPayload("iam-credential-check", IAMCredentialCheck{})
}
