package headless

import (
	"flag"
	"strings"

	"github.com/404tk/cloudtoolkit/runner/payloads"
)

// Stable process exit contract. Values are part of the headless automation
// API and must not be reassigned.
const (
	ExitSuccess          = 0
	ExitPartial          = 2
	ExitApprovalRequired = 3
	ExitConfigError      = 4
	ExitUnsupported      = 5
	ExitExecutionError   = 6
	ExitDeadlineExceeded = 124
	ExitCanceled         = 130
)

const (
	exitSuccess          = ExitSuccess
	exitPartial          = ExitPartial
	exitApprovalRequired = ExitApprovalRequired
	exitConfigError      = ExitConfigError
	exitUnsupported      = ExitUnsupported
	exitExecutionError   = ExitExecutionError
	exitDeadlineExceeded = ExitDeadlineExceeded
	exitCanceled         = ExitCanceled
)

type commandFlags struct {
	JSON      bool
	Quiet     bool
	Describe  bool
	Stdin     bool
	Approval  bool
	ShellMode bool
	CmdMode   bool
	Profile   string
	CredsPath string
	Metadata  string

	providerValues map[string]string
}

func (f *commandFlags) setProviderOption(name, value string) {
	if f.providerValues == nil {
		f.providerValues = make(map[string]string)
	}
	f.providerValues[name] = value
}

func (f commandFlags) providerOption(name string) string {
	if len(f.providerValues) == 0 {
		return ""
	}
	return strings.TrimSpace(f.providerValues[name])
}

func (f commandFlags) providerOptions() map[string]string {
	if len(f.providerValues) == 0 {
		return nil
	}
	items := make(map[string]string, len(f.providerValues))
	for key, value := range f.providerValues {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		items[key] = value
	}
	return items
}

type codedError interface {
	error
	ErrorCode() payloads.ErrorCode
}

type headlessError struct {
	code    payloads.ErrorCode
	message string
}

type actionSpec struct {
	payload string
	minArgs int
	maxArgs int
	usage   string
	summary string
	build   func([]string) string
}

type flagKind int

const (
	flagBool flagKind = iota
	flagValue
)

type helpSection int

const (
	helpHidden helpSection = iota
	helpCommon
	helpProvider
)

type headlessFlagSpec struct {
	long      string
	short     string
	aliases   []string
	kind      flagKind
	valueName string
	help      string
	section   helpSection
	bind      func(*flag.FlagSet, *commandFlags)
}

func (e headlessError) Error() string {
	return e.message
}

func (e headlessError) ErrorCode() payloads.ErrorCode {
	return e.code
}
