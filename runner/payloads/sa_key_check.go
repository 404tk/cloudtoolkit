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

type SAKeyCheck struct{}

type SAKeyCheckResult struct {
	Provider       string                `json:"provider"`
	Action         string                `json:"action"`
	ServiceAccount string                `json:"service_account,omitempty"`
	KeyID          string                `json:"key_id,omitempty"`
	PrivateKeyData string                `json:"private_key_data,omitempty"`
	Keys           []saKeyRowJSON        `json:"keys,omitempty"`
	Message        string                `json:"message,omitempty"`
	Status         string                `json:"status"`
	Error          string                `json:"error,omitempty"`
}

type saKeyRowJSON struct {
	KeyID       string `json:"key_id"`
	KeyType     string `json:"key_type,omitempty"`
	ValidAfter  string `json:"valid_after,omitempty"`
	ValidBefore string `json:"valid_before,omitempty"`
}

type saKeyAction struct {
	Action         string
	ServiceAccount string
	KeyID          string
}

func (p SAKeyCheck) Run(ctx context.Context, config map[string]string) {
	resultAny, err := p.Result(ctx, config)
	if err != nil && resultAny == nil {
		logger.Error(err.Error())
		return
	}
	result, ok := resultAny.(SAKeyCheckResult)
	if !ok {
		logger.Error("Invalid result type")
		return
	}
	if result.Status == "error" {
		logger.Error(result.Error)
		return
	}

	if result.PrivateKeyData != "" {
		type keyRow struct {
			ServiceAccount string `table:"Service Account"`
			KeyID          string `table:"Key ID"`
			PrivateKeyData string `table:"Private Key (base64)"`
		}
		table.Output([]keyRow{{
			ServiceAccount: result.ServiceAccount,
			KeyID:          result.KeyID,
			PrivateKeyData: result.PrivateKeyData,
		}})
	} else if len(result.Keys) > 0 {
		type keyRow struct {
			KeyID       string `table:"Key ID"`
			KeyType     string `table:"Type"`
			ValidAfter  string `table:"Valid After"`
			ValidBefore string `table:"Valid Before"`
		}
		rows := make([]keyRow, 0, len(result.Keys))
		for _, k := range result.Keys {
			rows = append(rows, keyRow{
				KeyID:       k.KeyID,
				KeyType:     k.KeyType,
				ValidAfter:  k.ValidAfter,
				ValidBefore: k.ValidBefore,
			})
		}
		table.Output(rows)
	}
	if result.Message != "" {
		logger.Warning(result.Message)
	}
}

func (p SAKeyCheck) Result(ctx context.Context, config map[string]string) (any, error) {
	parsed, err := parseSAKeyAction(config["metadata"])
	if err != nil {
		return nil, err
	}

	i, err := inventoryFromConfig(config)
	if err != nil {
		return nil, err
	}

	mgr, ok := i.Providers.(schema.ServiceAccountKeyManager)
	if !ok {
		return nil, fmt.Errorf("%s does not support sa-key-check", i.Providers.Name())
	}

	saResult, err := mgr.ServiceAccountKey(ctx, parsed.Action, parsed.ServiceAccount, parsed.KeyID)

	result := SAKeyCheckResult{
		Provider:       i.Providers.Name(),
		Action:         parsed.Action,
		ServiceAccount: parsed.ServiceAccount,
		KeyID:          parsed.KeyID,
	}
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result, NewResultError(result, 4, err)
	}

	if saResult.ServiceAccount != "" {
		result.ServiceAccount = saResult.ServiceAccount
	}
	if saResult.KeyID != "" {
		result.KeyID = saResult.KeyID
	}
	result.PrivateKeyData = saResult.PrivateKeyData
	result.Message = saResult.Message
	for _, k := range saResult.Keys {
		result.Keys = append(result.Keys, saKeyRowJSON{
			KeyID:       k.KeyID,
			KeyType:     k.KeyType,
			ValidAfter:  k.ValidAfter,
			ValidBefore: k.ValidBefore,
		})
	}
	result.Status = "success"
	return result, nil
}

func (p SAKeyCheck) Desc() string {
	return "Mint or revoke a service-account key in an authorized environment to validate detection coverage for credential lifecycle abuse."
}

func (p SAKeyCheck) Capability() string {
	return "iam-sa-key"
}

func (p SAKeyCheck) Help() HelpDoc {
	return HelpDoc{
		MetadataSyntax: []string{
			"set metadata <action> <service-account> [key-id]",
			"`action` is typically `list`, `create`, or `delete`.",
		},
		MetadataExamples: []string{
			"set metadata list ctk-demo@ctk-demo-project.iam.gserviceaccount.com",
			"set metadata create ctk-demo@ctk-demo-project.iam.gserviceaccount.com",
			"set metadata delete ctk-demo@ctk-demo-project.iam.gserviceaccount.com 123abc",
		},
		MetadataSuggestions: []Suggestion{
			{Text: "list <service-account>", Description: "enumerate keys for a service account"},
			{Text: "create <service-account>", Description: "mint a validation key for a service account"},
			{Text: "delete <service-account> <key-id>", Description: "revoke a validation key"},
		},
		SafetyNotes: []string{
			"Service-account keys are long-lived credentials; treat created key material as sensitive and revoke it after validation.",
			"Run only where minting service-account keys is explicitly authorized.",
		},
	}
}

func (p SAKeyCheck) Sensitivity(metadata string) Sensitivity {
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
		ConfirmKey: "sa-key-check." + data[0],
		Resource:   resource,
	}
}

func parseSAKeyAction(metadata string) (saKeyAction, error) {
	data := argparse.Split(metadata)
	if len(data) == 0 {
		return saKeyAction{}, errors.New("invalid metadata format: expected 'list <sa>', 'create <sa>', or 'delete <sa> <key-id>'")
	}
	action := saKeyAction{Action: data[0]}
	switch action.Action {
	case "list":
		if len(data) < 2 {
			return saKeyAction{}, errors.New("invalid metadata format: expected 'list <service-account>'")
		}
		action.ServiceAccount = data[1]
	case "create":
		if len(data) < 2 {
			return saKeyAction{}, errors.New("invalid metadata format: expected 'create <service-account>'")
		}
		action.ServiceAccount = data[1]
	case "delete":
		if len(data) < 3 {
			return saKeyAction{}, errors.New("invalid metadata format: expected 'delete <service-account> <key-id>'")
		}
		action.ServiceAccount = data[1]
		action.KeyID = data[2]
	default:
		return saKeyAction{}, fmt.Errorf("unsupported action %q: expected list / create / delete", action.Action)
	}
	return action, nil
}

func init() {
	registerPayload("sa-key-check", SAKeyCheck{})
}
