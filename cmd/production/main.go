package main

import (
	"context"
	"os"

	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/pkg/cloud_logging"
	"github.com/pdcgo/shared/pkg/ware_cache"
	"github.com/urfave/cli/v3"
	"gorm.io/gorm"
)

func NewCache(cfg *configs.AppConfig) (ware_cache.Cache, error) {
	return ware_cache.NewCustomCache(cfg.CacheService.Endpoint), nil
}

func NewAuthorization(
	cfg *configs.AppConfig,
	db *gorm.DB,
	cache ware_cache.Cache,
) authorization_iface.Authorization {

	return authorization.NewAuthorization(cache, db, cfg.JwtSecret)
}

func NewDatabase(cfg *configs.AppConfig) (*gorm.DB, error) {
	return db_connect.NewProductionDatabase("warehouse_service", &cfg.Database)
}

func NewApp(
	serviceFunc ServiceApiFunc,
	prepareStatFunc PrepareStatFunc,
) *cli.Command {
	return &cli.Command{
		Name:   "Warehouse Service",
		Action: cli.ActionFunc(serviceFunc),
		Commands: []*cli.Command{
			&cli.Command{
				Name:   "prepare-stat",
				Action: cli.ActionFunc(prepareStatFunc),
			},
		},
	}
}

func main() {
	if os.Getenv("DISABLE_CLOUD_LOGGING") == "" {
		cloud_logging.SetCloudLoggingDefault()
	}

	app, err := InitializeApp()
	if err != nil {
		panic(err)
	}

	err = app.Run(context.Background(), os.Args)
	if err != nil {
		panic(err)
	}
}
