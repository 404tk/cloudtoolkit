package headless

import (
	"errors"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/runtime/vmexecspec"
	"github.com/404tk/cloudtoolkit/utils"
)

var actionSpecs = map[string]actionSpec{
	"ls": {
		payload: "cloudlist",
		minArgs: 0,
		maxArgs: 1,
		usage:   "ls [resource[,resource...]]",
		summary: "list cloud resources",
		build: func(args []string) string {
			if len(args) == 0 {
				return ""
			}
			return strings.TrimSpace(args[0])
		},
	},
	"useradd": {
		payload: "iam-user-check",
		minArgs: 2,
		maxArgs: 2,
		usage:   "useradd <username> <password>",
		summary: "create a validation IAM user",
		build: func(args []string) string {
			return "add " + args[0] + " " + args[1]
		},
	},
	"userdel": {
		payload: "iam-user-check",
		minArgs: 1,
		maxArgs: 1,
		usage:   "userdel <username>",
		summary: "remove a validation IAM user",
		build: func(args []string) string {
			return "del " + args[0]
		},
	},
	"bls": {
		payload: "bucket-check",
		minArgs: 0,
		maxArgs: 1,
		usage:   "bls [bucket]",
		summary: "list objects in bucket(s)",
		build: func(args []string) string {
			if len(args) == 0 {
				return "list all"
			}
			return "list " + args[0]
		},
	},
	"bcnt": {
		payload: "bucket-check",
		minArgs: 0,
		maxArgs: 1,
		usage:   "bcnt [bucket]",
		summary: "count objects in bucket(s)",
		build: func(args []string) string {
			if len(args) == 0 {
				return "total all"
			}
			return "total " + args[0]
		},
	},
	"shell": {
		payload: "instance-cmd-check",
		minArgs: 2,
		maxArgs: -1,
		usage:   "shell <instance-id> <cmd...> -r <region> (-sh | -cmd)",
		summary: "run validation on a single instance",
	},
	"rolels": {
		payload: "role-binding-check",
		minArgs: 0,
		maxArgs: 2,
		usage:   "rolels [principal] [scope]",
		summary: "list role bindings at a scope",
		build: func(args []string) string {
			parts := []string{"list"}
			parts = append(parts, args...)
			return strings.Join(parts, " ")
		},
	},
	"roleadd": {
		payload: "role-binding-check",
		minArgs: 2,
		maxArgs: 3,
		usage:   "roleadd <principal> <role> [scope]",
		summary: "bind a principal to a role at a scope",
		build: func(args []string) string {
			parts := []string{"add"}
			parts = append(parts, args...)
			return strings.Join(parts, " ")
		},
	},
	"roledel": {
		payload: "role-binding-check",
		minArgs: 2,
		maxArgs: 3,
		usage:   "roledel <principal> <role> [scope]",
		summary: "remove a principal/role binding at a scope",
		build: func(args []string) string {
			parts := []string{"del"}
			parts = append(parts, args...)
			return strings.Join(parts, " ")
		},
	},
	"bacl": {
		payload: "bucket-acl-check",
		minArgs: 0,
		maxArgs: 1,
		usage:   "bacl [container]",
		summary: "audit storage container public access",
		build: func(args []string) string {
			parts := []string{"audit"}
			parts = append(parts, args...)
			return strings.Join(parts, " ")
		},
	},
	"expose": {
		payload: "bucket-acl-check",
		minArgs: 1,
		maxArgs: 2,
		usage:   "expose <container> [level]",
		summary: "set public access on a storage container",
		build: func(args []string) string {
			parts := []string{"expose"}
			parts = append(parts, args...)
			return strings.Join(parts, " ")
		},
	},
	"unexpose": {
		payload: "bucket-acl-check",
		minArgs: 1,
		maxArgs: 1,
		usage:   "unexpose <container>",
		summary: "revert public access on a storage container",
		build: func(args []string) string {
			return "unexpose " + args[0]
		},
	},
	"keyls": {
		payload: "sa-key-check",
		minArgs: 1,
		maxArgs: 1,
		usage:   "keyls <service-account>",
		summary: "list service-account keys",
		build: func(args []string) string {
			return "list " + args[0]
		},
	},
	"keyadd": {
		payload: "sa-key-check",
		minArgs: 1,
		maxArgs: 1,
		usage:   "keyadd <service-account>",
		summary: "mint a validation service-account key",
		build: func(args []string) string {
			return "create " + args[0]
		},
	},
	"keydel": {
		payload: "sa-key-check",
		minArgs: 2,
		maxArgs: 2,
		usage:   "keydel <service-account> <key-id>",
		summary: "revoke a service-account key",
		build: func(args []string) string {
			return "delete " + args[0] + " " + args[1]
		},
	},
}

func resolveRunRequest(command string, args []string, flags commandFlags) (string, string, error) {
	command = strings.TrimSpace(command)
	metadata := strings.TrimSpace(flags.Metadata)
	if command == "shell" {
		return resolveShellAction(args, flags)
	}
	if spec, ok := actionSpecs[command]; ok {
		if metadata != "" {
			return "", "", fmt.Errorf("%s does not accept --metadata; use `%s`", command, spec.usage)
		}
		if len(args) < spec.minArgs {
			return "", "", fmt.Errorf("usage: %s", spec.usage)
		}
		if spec.maxArgs >= 0 && len(args) > spec.maxArgs {
			return "", "", fmt.Errorf("usage: %s", spec.usage)
		}
		if spec.build == nil {
			return "", "", fmt.Errorf("usage: %s", spec.usage)
		}
		return spec.payload, spec.build(args), nil
	}
	return "", "", fmt.Errorf("unsupported: %s", command)
}

func resolveShellAction(args []string, flags commandFlags) (string, string, error) {
	metadata := strings.TrimSpace(flags.Metadata)
	if metadata != "" {
		return "", "", errors.New("shell does not accept --metadata")
	}
	if len(args) < 2 {
		return "", "", errors.New("usage: shell <instance-id> <cmd...> -r <region> (-sh | -cmd)")
	}
	region := flags.providerOption(utils.Region)
	if region == "" || strings.EqualFold(region, "all") {
		return "", "", errors.New("headless shell requires explicit -r <region>; region=all is not supported")
	}
	if flags.ShellMode == flags.CmdMode {
		return "", "", errors.New("headless shell requires exactly one of -sh or -cmd")
	}

	instanceID := strings.TrimSpace(args[0])
	command := strings.TrimSpace(strings.Join(args[1:], " "))
	if instanceID == "" || command == "" {
		return "", "", errors.New("usage: shell <instance-id> <cmd...> -r <region> (-sh | -cmd)")
	}

	if flags.ShellMode {
		return "instance-cmd-check", instanceID + " " + vmexecspec.BuildLinux(command), nil
	}
	return "instance-cmd-check", instanceID + " " + vmexecspec.BuildWindows(command), nil
}

func isHeadlessCommand(command string) bool {
	command = strings.TrimSpace(command)
	_, ok := actionSpecs[command]
	return ok
}

func resolveCloudlistSelection(defaults []string, selection string) ([]string, error) {
	selection = strings.TrimSpace(selection)
	if selection == "" || strings.EqualFold(selection, "all") {
		return append([]string(nil), defaults...), nil
	}

	aliases := map[string]string{
		"all":      "all",
		"balance":  "balance",
		"amt":      "balance",
		"host":     "host",
		"vm":       "host",
		"user":     "account",
		"account":  "account",
		"iam":      "account",
		"bucket":   "bucket",
		"s3":       "bucket",
		"database": "database",
		"db":       "database",
		"rds":      "database",
		"domain":   "domain",
		"dns":      "domain",
		"sms":      "sms",
		"log":      "log",
		"sls":      "log",
	}

	items := make([]string, 0)
	seen := make(map[string]struct{})
	for _, raw := range strings.Split(selection, ",") {
		key := strings.ToLower(strings.TrimSpace(raw))
		if key == "" {
			continue
		}
		mapped, ok := aliases[key]
		if !ok {
			return nil, fmt.Errorf("unknown cloudlist resource type: %s", raw)
		}
		if mapped == "all" {
			return append([]string(nil), defaults...), nil
		}
		if _, ok := seen[mapped]; ok {
			continue
		}
		seen[mapped] = struct{}{}
		items = append(items, mapped)
	}
	if len(items) == 0 {
		return append([]string(nil), defaults...), nil
	}
	return items, nil
}
