package console

import "fmt"

func providerPromptPrefix(provider string) string {
	return fmt.Sprintf("ctk > %s > ", provider)
}

func mockPromptPrefix(provider string) string {
	return fmt.Sprintf("ctk > %s[mock] > ", provider)
}

func shellPromptPrefix(instanceID string, mock bool) string {
	if mock {
		return fmt.Sprintf("[mock@%s ~]$ ", instanceID)
	}
	return fmt.Sprintf("[validation@%s ~]$ ", instanceID)
}
