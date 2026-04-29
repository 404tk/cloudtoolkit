package headless

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func writeHelp() int {
	var b strings.Builder
	b.WriteString("Usage:\n")
	b.WriteString("  ctk                      start REPL\n")
	b.WriteString("  ctk -v                   print version\n")
	b.WriteString("  ctk -h | --help          show this help\n")
	b.WriteString("  ctk <provider> <action> [args] [flags]\n")
	b.WriteString("  ctk <action> [args] (-P <profile> | --creds <file> | --stdin) [flags]\n")

	writeHelpActions(&b)
	writeHelpFlags(&b, "Common flags:", helpCommon)
	writeHelpFlags(&b, "Provider flags:", helpProvider)

	if _, err := fmt.Fprint(os.Stdout, b.String()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return exitConfigError
	}
	return exitSuccess
}

func writeHelpActions(b *strings.Builder) {
	type actionHelp struct {
		usage   string
		summary string
	}

	actions := make([]actionHelp, 0, len(actionSpecs))
	width := 0
	for _, spec := range actionSpecs {
		usage := strings.TrimSpace(spec.usage)
		if usage == "" {
			continue
		}
		actions = append(actions, actionHelp{
			usage:   usage,
			summary: strings.TrimSpace(spec.summary),
		})
		if len(usage) > width {
			width = len(usage)
		}
	}
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].usage < actions[j].usage
	})

	b.WriteString("\nActions:\n")
	for _, action := range actions {
		if action.summary == "" {
			fmt.Fprintf(b, "  %s\n", action.usage)
			continue
		}
		fmt.Fprintf(b, "  %-*s  %s\n", width, action.usage, action.summary)
	}
}

func writeHelpFlags(b *strings.Builder, title string, section helpSection) {
	specs := helpSpecsFor(section)
	if len(specs) == 0 {
		return
	}

	labels := make([]string, 0, len(specs))
	width := 0
	for _, spec := range specs {
		label := helpLabel(spec)
		labels = append(labels, label)
		if len(label) > width {
			width = len(label)
		}
	}

	fmt.Fprintf(b, "\n%s\n", title)
	for i, spec := range specs {
		fmt.Fprintf(b, "  %-*s  %s\n", width, labels[i], spec.help)
	}
}

func helpSpecsFor(section helpSection) []headlessFlagSpec {
	out := make([]headlessFlagSpec, 0)
	for _, spec := range headlessFlagSpecs {
		if spec.section != section {
			continue
		}
		if strings.TrimSpace(spec.help) == "" {
			continue
		}
		out = append(out, spec)
	}
	return out
}

func helpLabel(spec headlessFlagSpec) string {
	parts := make([]string, 0, 2+len(spec.aliases))
	if spec.short != "" {
		parts = append(parts, "-"+spec.short)
	}
	for _, alias := range spec.aliases {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}
		parts = append(parts, "-"+alias)
	}
	if spec.long != "" {
		parts = append(parts, "--"+spec.long)
	}
	label := strings.Join(parts, ", ")
	if spec.kind == flagValue && spec.valueName != "" {
		label += " <" + spec.valueName + ">"
	}
	return label
}
