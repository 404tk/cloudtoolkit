package activeDirectory

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

type Driver struct {
	Config auth.ClientCredentialsConfig
}

func (d *Driver) GetActiveDirectory(ctx context.Context) ([]schema.User, error) {
	list := []schema.User{}
	logger.Info("Start enumerating Active Directory ...")
	usersClient := graphrbac.NewUsersClient(d.Config.TenantID)
	d.Config.Resource = azure.PublicCloud.GraphEndpoint
	auth, _ := d.Config.Authorizer()
	usersClient.Authorizer = auth
	users, err := usersClient.List(ctx, "", "")
	if err != nil {
		logger.Error("List Active Directory failed.")
		return list, err
	}

	for _, user := range users.Values() {
		_user := schema.User{
			UserName:    *user.DisplayName,
			UserId:      *user.ObjectID,
			EnableLogin: *user.AccountEnabled,
		}

		list = append(list, _user)
	}

	return list, nil
}
