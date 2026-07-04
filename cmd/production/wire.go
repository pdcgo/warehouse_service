//go:build wireinject
// +build wireinject

package main

import (
	"net/http"

	"github.com/google/wire"
	"github.com/pdcgo/event_source"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/warehouse_service/v2"
	"github.com/urfave/cli/v3"
)

func InitializeApp() (*cli.Command, error) {
	wire.Build(
		http.NewServeMux,
		configs.NewProductionConfig,
		custom_connect.NewDefaultInterceptor,
		custom_connect.NewRegisterReflect,
		NewDatabase,
		NewCache,
		NewCacheManager,
		NewAuthorization,
		event_source.NewPubSubDefaultClient,
		event_source.NewPubsubEventSender,
		warehouse_service.NewWarehousePushHandler,
		warehouse_service.NewWarehousePushHttpHandler,
		warehouse_service.NewRegister,
		NewServiceApi,
		NewPrepareStat,
		NewApp,
	)

	return &cli.Command{}, nil
}
