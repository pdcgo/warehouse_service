package main

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/schema/services/warehouse_iface/v1/warehouse_ifaceconnect"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

type PrepareStatFunc cli.ActionFunc

func NewPrepareStat(db *gorm.DB, cfg *configs.AppConfig) PrepareStatFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		cancel, err := custom_connect.InitTracer("warehouse-service")
		if err != nil {
			return err
		}

		defer cancel(ctx)

		skuChan := make(chan []string, 10)

		go func() {
			slog.Info("running producer..")
			defer close(skuChan)

			ctx, span := otel.Tracer("").Start(ctx, "prepare-statistic-producer")
			defer span.End()

			db = db.WithContext(ctx)

			query := db.
				Raw(`
				select 
					distinct sku_id
				from invertory_histories ih 
				where ih.tx_id is null
			`)

			rows, err := query.Rows()

			if err != nil {
				slog.Error(err.Error())
				span.SetAttributes(
					attribute.String("producer.error", err.Error()),
				)
				span.SetStatus(codes.Error, err.Error())
				return
			}

			defer rows.Close()

			result := []string{}

			batchCount := 100

			for rows.Next() {
				var skuID string
				err = rows.Scan(&skuID)

				if err != nil {
					slog.Error(err.Error())
					span.SetAttributes(
						attribute.String("producer.parsing.error", err.Error()),
					)
					span.SetStatus(codes.Error, err.Error())
					return
				}

				result = append(result, skuID)

				if len(result) > batchCount {
					skuChan <- result
					result = []string{}
				}
			}

			if len(result) > 0 {
				skuChan <- result
			}
		}()

		// creating client
		inventoryServiceClient := warehouse_ifaceconnect.NewInventoryServiceClient(
			http.DefaultClient,
			cfg.InventoryService.Endpoint,
			connect.WithGRPC(),
		)

		ctx, span := otel.Tracer("").Start(ctx, "prepare-statistic-consumer")
		defer span.End()

		for skuIds := range skuChan {
			slog.Info("preparing", "count", len(skuIds))
			_, err = inventoryServiceClient.PrepareSkus(ctx, &connect.Request[warehouse_iface.PrepareSkusRequest]{
				Msg: &warehouse_iface.PrepareSkusRequest{
					SkuIds: skuIds,
				},
			})

			if err != nil {
				slog.Error(err.Error())
				span.SetAttributes(
					attribute.String("consumer.error", err.Error()),
				)
				span.SetStatus(codes.Error, err.Error())
			}
		}

		return nil
	}
}
