package processbar

import (
	"fmt"
	"strings"
)

func RegionPrint(region string, count, prev int, flag bool) (int, bool) {
	progress := fmt.Sprintf("[%s] %d found.", region, count)
	if count == 0 {
		if flag {
			fmt.Print(progress)
		} else {
			progress += getBlock(prev, len(progress))
			fmt.Printf("\r%s", progress)
		}
		flag = false
	} else {
		if flag {
			fmt.Println(progress)
		} else {
			progress += getBlock(prev, len(progress))
			fmt.Printf("\r%s\n", progress)
		}
		flag = true
	}
	return len(progress), flag
}

func CountPrint(tag string, count, prev int) int {
	progress := fmt.Sprintf("[%s] %d found.", tag, count)
	progress += getBlock(prev, len(progress))
	fmt.Printf("\r%s", progress)
	return len(progress)
}

func getBlock(prev, index int) string {
	if prev > index {
		return strings.Repeat(" ", prev-index)
	}
	return ""
}
