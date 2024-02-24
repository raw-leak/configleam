package configleamsecrets

import (
	"context"

	"github.com/raw-leak/configleam/config"
	"github.com/raw-leak/configleam/internal/app/configleam-dashboard/controller"
	"github.com/raw-leak/configleam/internal/app/configleam-dashboard/service"
	"github.com/raw-leak/configleam/internal/app/configleam-dashboard/templates"
)

type ConfigleamDashboardSet struct {
	*controller.ConfigleamDashboardEndpoints
}

func Init(ctx context.Context, cfg *config.Config, accessService service.AccessService, configService service.ConfigService) (*ConfigleamDashboardSet, error) {
	service := service.New(accessService, configService)
	templates := templates.New()
	endpoints := controller.New(service, templates)

	return &ConfigleamDashboardSet{
		endpoints,
	}, nil
}
