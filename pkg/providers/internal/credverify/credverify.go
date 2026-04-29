package credverify

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Result struct {
	Summary     string
	SessionUser string
}

func ForCloudlist(options schema.Options, provider any, skipCredentialCache bool, probe func(context.Context) (Result, error)) error {
	if strings.TrimSpace(options[utils.Payload]) != "cloudlist" {
		return nil
	}

	result, err := probe(context.Background())
	if err != nil {
		return err
	}
	if !skipCredentialCache {
		cache.Cfg.CredInsert(result.SessionUser, provider, options)
	}
	if strings.TrimSpace(result.Summary) != "" {
		logger.Warning(result.Summary)
	}
	return nil
}
