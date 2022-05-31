package console

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func cloudlist() {
	inventory, err := inventory.New(schema.Options{
		config,
	})
	if err != nil {
		log.Println(err)
		return
	}

	for _, provider := range inventory.Providers {
		resources, err := provider.Resources(context.Background())
		if err != nil {
			log.Println(err)
			return
		}

		resp, _ := json.MarshalIndent(resources, "", "\t")
		fmt.Printf("%s\n", resp)
	}
}
