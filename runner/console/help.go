package console

import (
	"fmt"
	"strings"
)

func help(args []string) {
	args = normalizeHelpArgs(args)
	ctx := currentHelpContext()
	if len(args) == 0 {
		renderContextHelp(ctx)
		return
	}

	switch args[0] {
	case "payload":
		if len(args) == 1 {
			renderPayloadCatalogHelp(ctx)
			return
		}
		renderPayloadHelp(ctx, args[1], false)
		return
	case "metadata":
		if len(args) == 1 {
			renderMetadataOverviewHelp(ctx)
			return
		}
		renderPayloadHelp(ctx, args[1], true)
		return
	}

	if topic, ok := helpTopics[args[0]]; ok {
		renderTopicHelp(ctx, topic)
		return
	}

	fmt.Printf("No help available for %q.\n", strings.Join(args, " "))
	fmt.Println("Try `help`, `help payload`, or `help metadata`.")
}

func normalizeHelpArgs(args []string) []string {
	normalized := make([]string, 0, len(args))
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg != "" {
			normalized = append(normalized, arg)
		}
	}
	return normalized
}
