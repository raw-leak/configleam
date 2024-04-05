package notify

import (
	"context"

	"github.com/raw-leak/configleam/config"
	"github.com/raw-leak/configleam/internal/app/notify/controller"
	"github.com/raw-leak/configleam/internal/app/notify/service"
)

type NotifSet struct {
	*controller.NotifyEndpoints
	*service.NotifyService
}

func Init(ctx context.Context, cfg *config.Config) (*NotifSet, error) {
	service := service.New(bool(cfg.EnableLeaderElection))
	endpoints := controller.New(service)

	return &NotifSet{
		endpoints,
		service,
	}, nil
}
