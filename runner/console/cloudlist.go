package console

import (
	"context"
	"fmt"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/modood/table"
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

		if len(resources.Hosts) > 0 {
			fmt.Println("Host results: ")
			table.Output(resources.Hosts)
		} else {
			log.Println("No host result found.")
		}
	}
}
