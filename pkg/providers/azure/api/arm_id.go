package api

import (
	"fmt"
	"strings"
)

type ResourceID struct {
	SubscriptionID string
	ResourceGroup  string
	Provider       string
	ResourceType   string
	ResourceName   string
}

func ParseResourceID(id string) (ResourceID, error) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(id), "/"), "/")
	if len(parts) < 8 {
		return ResourceID{}, fmt.Errorf("azure arm id: invalid resource id %q", id)
	}

	resource := ResourceID{}
	for i := 0; i < len(parts)-1; i++ {
		switch {
		case strings.EqualFold(parts[i], "subscriptions") && i+1 < len(parts):
			resource.SubscriptionID = parts[i+1]
			i++
		case strings.EqualFold(parts[i], "resourceGroups") && i+1 < len(parts):
			resource.ResourceGroup = parts[i+1]
			i++
		case strings.EqualFold(parts[i], "providers") && i+1 < len(parts):
			resource.Provider = parts[i+1]
			remaining := parts[i+2:]
			if len(remaining) >= 2 {
				resource.ResourceType = strings.Join(remaining[:len(remaining)-1], "/")
				resource.ResourceName = remaining[len(remaining)-1]
			}
			i = len(parts)
		}
	}

	if resource.SubscriptionID == "" || resource.ResourceGroup == "" || resource.Provider == "" || resource.ResourceName == "" {
		return ResourceID{}, fmt.Errorf("azure arm id: invalid resource id %q", id)
	}
	return resource, nil
}
