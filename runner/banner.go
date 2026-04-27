package runner

import "fmt"

const banner = `
  ▄▄▄ ▄▄▄▄▄▄▄ ▄    ▄
▄▀   ▀   █    █  ▄▀
█        █    █▄█
█        █    █  █▄
 ▀▄▄▄▀   █    █   ▀▄
                      v%s
`

// version is the current version of cloudtoolkit
const version = `0.2.7`

// showBanner is used to show the banner to the user
func ShowBanner() {
	fmt.Printf(banner, version)
}
