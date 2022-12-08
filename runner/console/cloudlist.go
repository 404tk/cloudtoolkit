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
	inventory, err := inventory.New(schema.Options{config})
	if err != nil {
		log.Println(err)
		return
	}

	for _, provider := range inventory.Providers {
		resources, err := provider.Resources(context.Background())
		if err != nil {
			log.Println("[Failed]", err.Error())
			// return
		}
		pprint := func(len int, tag string, res interface{}) {
			if len > 0 {
				fmt.Println(fmt.Sprintf("%s results:\n%s", tag, table.Table(res)))
			}
		}

		pprint(len(resources.Hosts), "Hosts", resources.Hosts)
		pprint(len(resources.Storages), "Storages", resources.Storages)
		pprint(len(resources.Users), "Users", resources.Users)

		log.Println("[+] Done.")
	}
}
