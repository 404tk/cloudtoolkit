package replay

import demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"

func Enable(provider string) {
	demoreplay.Enable(provider)
}

func Disable() {
	demoreplay.Disable()
}

func IsActive() bool {
	return demoreplay.IsActive()
}

func ActiveProvider() string {
	return demoreplay.ActiveProvider()
}

func IsActiveForProvider(provider string) bool {
	return demoreplay.IsActiveForProvider(provider)
}
