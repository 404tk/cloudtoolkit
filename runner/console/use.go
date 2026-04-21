package console

import (
	"errors"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

var config map[string]string

func use(args []string) {
	if len(args) < 1 {
		logger.Error("Example: use alibaba")
		return
	}
	if err := loadModule(args[0]); err != nil {
		logger.Error(err)
	}
}

func loadModule(m string) error {
	defaultConfig, ok := defaultProviderConfig(m)
	if !ok {
		return errors.New("Unsupported provider: " + m)
	}
	config = defaultConfig

	config[utils.Provider] = m
	config[utils.Payload] = "cloudlist" // Default payload is cloud asset inventory
	config[utils.Metadata] = ""
	resetDemoReplay()
	startProviderConsole(m)
	return nil
}
