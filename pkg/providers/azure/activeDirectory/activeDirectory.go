package activeDirectory

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

type ADProvider struct {
	Config auth.ClientCredentialsConfig
}

func (d *ADProvider) GetActiveDirectory(ctx context.Context) ([]*schema.User, error) {
	list := schema.NewResources().Users
	log.Println("[*] Start enumerating Active Directory ...")
	usersClient := graphrbac.NewUsersClient(d.Config.TenantID)
	d.Config.Resource = azure.PublicCloud.GraphEndpoint
	auth, _ := d.Config.Authorizer()
	usersClient.Authorizer = auth
	users, err := usersClient.List(ctx, "", "")
	if err != nil {
		log.Println("[-] List Active Directory failed.")
		return list, err
	}

	for _, user := range users.Values() {
		_user := &schema.User{
			UserName:    *user.DisplayName,
			UserId:      *user.ObjectID,
			EnableLogin: *user.AccountEnabled,
		}

		list = append(list, _user)
	}

	return list, nil
}