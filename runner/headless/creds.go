package headless

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/registry"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
)

func buildRunConfig(provider, payload, metadataOverride string, flags commandFlags) (map[string]string, error) {
	sourceData, err := credentialDataFromFlags(flags)
	if err != nil {
		return nil, err
	}
	sourceProvider := strings.TrimSpace(sourceData[utils.Provider])
	resolvedProvider := strings.TrimSpace(provider)
	metadata := strings.TrimSpace(metadataOverride)
	flagMetadata := strings.TrimSpace(flags.Metadata)
	if resolvedProvider == "" {
		resolvedProvider = sourceProvider
	}
	if resolvedProvider == "" {
		return nil, errors.New("provider is required unless supplied by the selected credential source")
	}
	if sourceProvider != "" && sourceProvider != resolvedProvider {
		return nil, fmt.Errorf("provider mismatch: command selected %s but credential source is for %s", resolvedProvider, sourceProvider)
	}
	if _, ok := registry.Lookup(resolvedProvider); !ok {
		return nil, fmt.Errorf("unsupported provider: %s", resolvedProvider)
	}
	config, _ := registry.DefaultConfig(resolvedProvider)
	config[utils.Provider] = resolvedProvider
	config[utils.Payload] = payload
	config[utils.Metadata] = metadata

	mergeConfig(config, sourceData)
	mergeConfig(config, flags.providerOptions())

	config[utils.Provider] = resolvedProvider
	config[utils.Payload] = payload
	if metadata != "" {
		config[utils.Metadata] = metadata
	} else if flagMetadata != "" {
		config[utils.Metadata] = flagMetadata
	}
	return config, nil
}

func credentialDataFromFlags(flags commandFlags) (map[string]string, error) {
	profile := strings.TrimSpace(flags.Profile)
	credsPath := strings.TrimSpace(flags.CredsPath)
	sourceCount := 0
	if profile != "" {
		sourceCount++
	}
	if credsPath != "" {
		sourceCount++
	}
	if flags.Stdin {
		sourceCount++
	}
	if sourceCount > 1 {
		return nil, errors.New("credential sources are mutually exclusive: choose one of --profile, --creds, or --stdin")
	}

	switch {
	case profile != "":
		return loadProfile(profile)
	case credsPath != "":
		return loadCredentialFile(credsPath)
	case flags.Stdin:
		return loadCredentialStdin()
	default:
		return nil, nil
	}
}

func mergeConfig(dst, src map[string]string) {
	for key, value := range src {
		if strings.TrimSpace(value) == "" {
			continue
		}
		dst[key] = value
	}
}

func loadCredentialFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decodeCredentialJSON(data)
}

func loadCredentialStdin() (map[string]string, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	return decodeCredentialJSON(data)
}

func loadProfile(profile string) (map[string]string, error) {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return nil, errors.New("empty profile name")
	}

	if id, err := findProfileID(profile); err == nil {
		return decodeSessionJSON(cache.Cfg.CredSelect(id))
	}
	return decodeSessionJSON(cache.Cfg.CredSelect(profile))
}

func findProfileID(profile string) (string, error) {
	for _, cred := range cache.Cfg.Snapshot() {
		if cred.Note == profile || cred.UUID == profile {
			return cred.UUID, nil
		}
	}
	return "", fmt.Errorf("profile not found: %s", profile)
}

func decodeSessionJSON(data string) (map[string]string, error) {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil, errors.New("empty cached session")
	}
	return decodeCredentialJSON([]byte(data))
}

func decodeCredentialJSON(data []byte) (map[string]string, error) {
	items := make(map[string]string)
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}
