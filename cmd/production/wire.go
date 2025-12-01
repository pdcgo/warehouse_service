//go:build wireinject
// +build wireinject

package main

import (
	"net/http"

	"github.com/google/wire"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/warehouse_service/v2"
)

func InitializeApp() (*App, error) {
	wire.Build(
		http.NewServeMux,
		configs.NewProductionConfig,
		custom_connect.NewDefaultInterceptor,
		NewDatabase,
		NewCache,
		NewAuthorization,

		warehouse_service.NewRegister,
		NewApp,
	)

	return &App{}, nil
}
