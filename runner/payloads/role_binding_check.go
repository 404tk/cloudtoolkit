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

type RoleBindingCheck struct{}

type RoleBindingCheckResult struct {
	Provider     string                `json:"provider"`
	Action       string                `json:"action"`
	Principal    string                `json:"principal,omitempty"`
	Role         string                `json:"role,omitempty"`
	Scope        string                `json:"scope,omitempty"`
	AssignmentID string                `json:"assignment_id,omitempty"`
	Bindings     []roleBindingRowJSON  `json:"bindings,omitempty"`
	Message      string                `json:"message,omitempty"`
	Status       string                `json:"status"`
	Error        string                `json:"error,omitempty"`
}

type roleBindingRowJSON struct {
	Principal    string `json:"principal"`
	Role         string `json:"role"`
	Scope        string `json:"scope,omitempty"`
	AssignmentID string `json:"assignment_id,omitempty"`
}

type roleBindingAction struct {
	Action    string
	Principal string
	Role      string
	Scope     string
}

func (p RoleBindingCheck) Run(ctx context.Context, config map[string]string) {
	resultAny, err := p.Result(ctx, config)
	if err != nil && resultAny == nil {
		logger.Error(err.Error())
		return
	}

	result, ok := resultAny.(RoleBindingCheckResult)
	if !ok {
		logger.Error("Invalid result type")
		return
	}
	if result.Status == "error" {
		logger.Error(result.Error)
		return
	}

	if len(result.Bindings) > 0 {
		type bindingRow struct {
			Principal    string `table:"Principal"`
			Role         string `table:"Role"`
			Scope        string `table:"Scope"`
			AssignmentID string `table:"Assignment ID"`
		}
		rows := make([]bindingRow, 0, len(result.Bindings))
		for _, b := range result.Bindings {
			rows = append(rows, bindingRow{
				Principal:    b.Principal,
				Role:         b.Role,
				Scope:        b.Scope,
				AssignmentID: b.AssignmentID,
			})
		}
		table.Output(rows)
	}
	if result.Message != "" {
		logger.Warning(result.Message)
	}
}

func (p RoleBindingCheck) Result(ctx context.Context, config map[string]string) (any, error) {
	parsed, err := parseRoleBindingAction(config["metadata"])
	if err != nil {
		return nil, err
	}

	i, err := inventoryFromConfig(config)
	if err != nil {
		return nil, err
	}

	mgr, ok := i.Providers.(schema.RoleBindingManager)
	if !ok {
		return nil, fmt.Errorf("%s does not support role-binding-check", i.Providers.Name())
	}

	bindingResult, err := mgr.RoleBinding(ctx, parsed.Action, parsed.Principal, parsed.Role, parsed.Scope)

	result := RoleBindingCheckResult{
		Provider:  i.Providers.Name(),
		Action:    parsed.Action,
		Principal: parsed.Principal,
		Role:      parsed.Role,
		Scope:     parsed.Scope,
	}

	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result, NewResultError(result, 4, err)
	}

	if bindingResult.Scope != "" {
		result.Scope = bindingResult.Scope
	}
	result.AssignmentID = bindingResult.AssignmentID
	result.Message = bindingResult.Message
	for _, b := range bindingResult.Bindings {
		result.Bindings = append(result.Bindings, roleBindingRowJSON{
			Principal:    b.Principal,
			Role:         b.Role,
			Scope:        b.Scope,
			AssignmentID: b.AssignmentID,
		})
	}
	result.Status = "success"
	return result, nil
}

func (p RoleBindingCheck) Desc() string {
	return "Bind or unbind a test principal against an authorized role at a chosen scope to validate role-assignment telemetry, alerting, and audit trail coverage."
}

func (p RoleBindingCheck) Capability() string {
	return "iam-role"
}

func (p RoleBindingCheck) Help() HelpDoc {
	return HelpDoc{
		MetadataSyntax: []string{
			"set metadata <action> [principal] [role] [scope]",
			"`action` is typically `add`, `del`, or `list`.",
			"`scope` is optional: defaults to the provider's primary scope (Azure subscription / GCP project).",
		},
		MetadataExamples: []string{
			"set metadata list",
			"set metadata add 11111111-2222-3333-4444-555555555555 Reader",
			"set metadata add user:demo@example.com roles/viewer projects/ctk-demo",
			"set metadata del 11111111-2222-3333-4444-555555555555 Reader",
		},
		MetadataSuggestions: []Suggestion{
			{Text: "list", Description: "enumerate role bindings at the default scope"},
			{Text: "list <principal>", Description: "filter role bindings by principal"},
			{Text: "add <principal> <role>", Description: "bind a principal to a role at the default scope"},
			{Text: "add <principal> <role> <scope>", Description: "bind a principal to a role at an explicit scope"},
			{Text: "del <principal> <role>", Description: "remove a principal/role binding at the default scope"},
			{Text: "del <principal> <role> <scope>", Description: "remove a principal/role binding at an explicit scope"},
		},
		SafetyNotes: []string{
			"Use a dedicated test principal and remove the binding immediately after detection coverage is validated.",
			"Run only where modifying RBAC / IAM policy is explicitly approved.",
		},
	}
}

func (p RoleBindingCheck) Sensitivity(metadata string) Sensitivity {
	data := argparse.Split(metadata)
	if len(data) == 0 {
		return Sensitivity{}
	}
	switch data[0] {
	case "add", "del":
	default:
		return Sensitivity{}
	}
	resource := ""
	if len(data) >= 2 {
		resource = data[1]
	}
	if len(data) >= 3 {
		resource = resource + "@" + data[2]
	}
	return Sensitivity{
		Level:      "destructive",
		ConfirmKey: "role-binding-check." + data[0],
		Resource:   resource,
	}
}

func parseRoleBindingAction(metadata string) (roleBindingAction, error) {
	data := argparse.Split(metadata)
	if len(data) == 0 {
		return roleBindingAction{}, errors.New("invalid metadata format: expected 'list', 'add <principal> <role> [scope]' or 'del <principal> <role> [scope]'")
	}
	action := roleBindingAction{Action: data[0]}
	switch action.Action {
	case "list":
		if len(data) >= 2 {
			action.Principal = data[1]
		}
		if len(data) >= 3 {
			action.Scope = data[2]
		}
	case "add", "del":
		if len(data) < 3 {
			return roleBindingAction{}, fmt.Errorf("invalid metadata format: expected '%s <principal> <role> [scope]'", action.Action)
		}
		action.Principal = data[1]
		action.Role = data[2]
		if len(data) >= 4 {
			action.Scope = data[3]
		}
	default:
		return roleBindingAction{}, fmt.Errorf("unsupported action %q: expected list / add / del", action.Action)
	}
	return action, nil
}

func init() {
	registerPayload("role-binding-check", RoleBindingCheck{})
}
