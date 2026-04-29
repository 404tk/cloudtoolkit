package catalog

import "strings"

var payloadSpecs = map[string]PayloadSpec{
	"cloudlist": {
		Capability:  "cloudlist",
		Sensitivity: "read",
	},
	"iam-user-check": {
		Capability:  "iam",
		Sensitivity: "destructive",
	},
	"bucket-check": {
		Capability:  "bucket",
		Sensitivity: "read",
	},
	"event-check": {
		Capability:  "event",
		Sensitivity: "mixed",
	},
	"rds-account-check": {
		Capability:  "database",
		Sensitivity: "destructive",
	},
	"instance-cmd-check": {
		Capability:  "vm",
		Sensitivity: "destructive",
	},
}

func PayloadSpecFor(name string) (PayloadSpec, bool) {
	name = strings.TrimSpace(name)
	spec, ok := payloadSpecs[name]
	return spec, ok
}

func PayloadCapability(name string) string {
	spec, ok := PayloadSpecFor(name)
	if !ok {
		return ""
	}
	return spec.Capability
}

func PayloadSensitivity(name string) string {
	spec, ok := PayloadSpecFor(name)
	if !ok {
		return ""
	}
	return spec.Sensitivity
}
