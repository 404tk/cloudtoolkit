package payloads

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/table"
)

// RunStructured is the compatibility entry point used by the interactive
// runner. It shares the same one-shot Result execution and renderer as
// headless mode.
func RunStructured(ctx context.Context, config map[string]string, producer ResultProducer) {
	result := Execute(ctx, config, producer)
	if result.Value != nil {
		if err := Render(ctx, result.Value); err != nil {
			logger.Error(err.Error())
		}
	}
	if result.Err != nil {
		logger.Error(result.Err.Error())
	}
}

// Render converts a structured payload result to the human-readable CLI
// representation without executing the provider operation again.
func Render(ctx context.Context, value any) error {
	switch result := value.(type) {
	case *CloudListResult:
		return renderCloudList(ctx, result)
	case CloudListResult:
		return renderCloudList(ctx, &result)
	case []BucketCheckResult:
		renderBucketCheck(result)
	case BucketACLCheckResult:
		renderBucketACL(result)
	case EventCheckResult:
		renderEventCheck(ctx, result)
	case IAMCredentialCheckResult:
		renderIAMCredential(result)
	case IAMUserCheckResult:
		renderIAMUser(result)
	case InstanceCmdCheckResult:
		if result.Status != "error" && result.Output != "" {
			_, err := os.Stdout.WriteString(result.Output)
			return err
		}
	case RDSAccountCheckResult:
		renderRDSAccount(result)
	case RoleBindingCheckResult:
		renderRoleBinding(result)
	default:
		return fmt.Errorf("unsupported structured result type %T", value)
	}
	return nil
}

func renderCloudList(ctx context.Context, result *CloudListResult) error {
	if result == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	e := env.From(ctx)
	path := ""
	if len(result.OutputFiles) > 0 {
		path = result.OutputFiles[0]
	}
	printGroup := func(tag string, items any) {
		logger.Warning(tag + " results:")
		table.Output(items)
		if e.LogEnable && path != "" {
			utils.WriteLog(path, tag+" results:")
			table.FileOutput(path, items)
		}
	}

	if len(result.Hosts) > 0 {
		printGroup("Hosts", result.Hosts)
	}
	if len(result.Storages) > 0 {
		printGroup("Storages", result.Storages)
	}
	if len(result.Users) > 0 {
		printGroup("Users", result.Users)
	}
	if len(result.Databases) > 0 {
		printGroup("Databases", result.Databases)
	}
	for _, domain := range result.Domains {
		if len(domain.Records) > 0 {
			printGroup("Domain "+domain.DomainName, domain.Records)
		}
	}
	if len(result.Logs) > 0 {
		printGroup("Log Service", result.Logs)
	}
	if len(result.SMS.Signs) > 0 {
		printGroup("SMS Signs", result.SMS.Signs)
	}
	if len(result.SMS.Templates) > 0 {
		printGroup("SMS Templates", result.SMS.Templates)
	}
	if result.SMS.DailySize > 0 {
		logger.Info(fmt.Sprintf("The total number of SMS messages sent today is %v.", result.SMS.DailySize))
	}
	for _, item := range result.Errors {
		logger.Error(fmt.Sprintf("%s failed: %s", item.Scope, item.Message))
	}
	if e.LogEnable && path != "" {
		logger.Info(fmt.Sprintf("Output written to [%s]", path))
	}
	return nil
}

func renderBucketCheck(results []BucketCheckResult) {
	for _, result := range results {
		if result.Status == "error" {
			continue
		}
		switch result.Action {
		case "list":
			if result.BucketName != "" {
				logger.Warning(fmt.Sprintf("%d objects found in %s.", result.ObjectCount, result.BucketName))
			} else if result.Message != "" {
				logger.Warning(result.Message)
			}
			type objectRow struct {
				Key  string `table:"Key"`
				Size string `table:"Size"`
			}
			rows := make([]objectRow, 0, len(result.Objects))
			for _, obj := range result.Objects {
				label := obj.Key
				if result.BucketName == "" && obj.BucketName != "" {
					label = obj.BucketName + "/" + obj.Key
				}
				rows = append(rows, objectRow{Key: label, Size: utils.ParseBytes(obj.Size)})
			}
			if len(rows) > 0 {
				table.Output(rows)
			}
		case "total":
			if result.BucketName != "" {
				logger.Warning(fmt.Sprintf("%s has %d objects.", result.BucketName, result.ObjectCount))
			} else if result.Message != "" {
				logger.Warning(result.Message)
			}
		default:
			if result.Message != "" {
				logger.Warning(result.Message)
			}
		}
	}
}

func renderBucketACL(result BucketACLCheckResult) {
	if result.Status == "error" {
		return
	}
	type aclRow struct {
		Account   string `table:"Account"`
		Container string `table:"Container"`
		Level     string `table:"Public Access"`
	}
	rows := make([]aclRow, 0, len(result.Containers))
	for _, entry := range result.Containers {
		rows = append(rows, aclRow{Account: entry.Account, Container: entry.Container, Level: entry.Level})
	}
	if len(rows) > 0 {
		table.Output(rows)
	}
	if result.Message != "" {
		logger.Warning(result.Message)
	}
}

func renderEventCheck(ctx context.Context, result EventCheckResult) {
	if result.Status == "error" {
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

func renderIAMCredential(result IAMCredentialCheckResult) {
	if result.Status == "error" {
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
		for _, key := range result.Credentials {
			rows = append(rows, keyRow{
				CredentialID:   key.CredentialID,
				CredentialType: key.CredentialType,
				ValidAfter:     key.ValidAfter,
				ValidBefore:    key.ValidBefore,
			})
		}
		table.Output(rows)
	}
	if result.Message != "" {
		logger.Warning(result.Message)
	}
}

func renderIAMUser(result IAMUserCheckResult) {
	if result.Status == "error" {
		return
	}
	if result.LoginURL != "" {
		type loginRow struct {
			Username string `table:"Username"`
			Password string `table:"Password"`
			LoginURL string `table:"Login URL"`
		}
		table.Output([]loginRow{{
			Username: result.Username,
			Password: result.Password,
			LoginURL: result.LoginURL,
		}})
	} else if result.Message != "" {
		logger.Warning(result.Message)
	}
}

func renderRDSAccount(result RDSAccountCheckResult) {
	if result.Status == "error" {
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

func renderRoleBinding(result RoleBindingCheckResult) {
	if result.Status == "error" {
		return
	}
	type bindingRow struct {
		Principal    string `table:"Principal"`
		Role         string `table:"Role"`
		Scope        string `table:"Scope"`
		AssignmentID string `table:"Assignment ID"`
	}
	rows := make([]bindingRow, 0, len(result.Bindings))
	for _, binding := range result.Bindings {
		rows = append(rows, bindingRow{
			Principal:    binding.Principal,
			Role:         binding.Role,
			Scope:        binding.Scope,
			AssignmentID: binding.AssignmentID,
		})
	}
	if len(rows) > 0 {
		table.Output(rows)
	}
	if result.Message != "" {
		logger.Warning(result.Message)
	}
}
