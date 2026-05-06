package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/warehouse_service/v2"
	"github.com/urfave/cli/v3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type ServiceApiFunc cli.ActionFunc

func NewServiceApi(
	mux *http.ServeMux,
	warehouseRegister warehouse_service.RegisterHandler,
	reflectorRegister custom_connect.RegisterReflectFunc,
	// cache ware_cache.Cache
	// auth authorization_iface.Authorization,
) ServiceApiFunc {

	return func(ctx context.Context, cmd *cli.Command) error {
		cancel, err := custom_connect.InitTracer("warehouse-service")
		if err != nil {
			return err
		}

		defer cancel(ctx)

		warehouseReflect := warehouseRegister()
		reflectorRegister(warehouseReflect)

		port := os.Getenv("PORT")
		if port == "" {
			port = "8084"
		}

		host := os.Getenv("HOST")
		listen := fmt.Sprintf("%s:%s", host, port)
		log.Println("listening on", listen)

		http.ListenAndServe(
			listen,
			// Use h2c so we can serve HTTP/2 without TLS.
			h2c.NewHandler(
				custom_connect.WithCORS(mux),
				&http2.Server{}),
		)

		return nil
	}
}
