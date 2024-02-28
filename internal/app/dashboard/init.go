package dashboard

import (
	"context"

	"github.com/raw-leak/configleam/config"
	"github.com/raw-leak/configleam/internal/app/dashboard/controller"
	"github.com/raw-leak/configleam/internal/app/dashboard/service"
	"github.com/raw-leak/configleam/internal/app/dashboard/templates"
)

type DashboardSet struct {
	*controller.DashboardEndpoints
}

func Init(ctx context.Context, cfg *config.Config, accessService service.AccessService, configService service.ConfigurationService) (*DashboardSet, error) {
	service := service.New(accessService, configService)
	templates := templates.New()
	endpoints := controller.New(service, templates)

	return &DashboardSet{
		endpoints,
	}, nil
}
