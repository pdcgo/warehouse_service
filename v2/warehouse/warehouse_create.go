package warehouse

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	role_base "github.com/pdcgo/schema/services/role_base/v1"
	warehouse_iface "github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/user_service/access_interceptors"
	"github.com/pdcgo/user_service/user_models"
	"gorm.io/gorm"
)

// warehouseCreateLogger adapts an slog handler onto the response stream: every log
// line is delivered to the client as a WarehouseCreateResponse.Message — the progress
// channel of the "long running task" RPC.
type warehouseCreateLogger struct {
	stream *connect.ServerStream[warehouse_iface.WarehouseCreateResponse]
}

// Write implements [io.Writer].
func (l *warehouseCreateLogger) Write(p []byte) (int, error) {
	err := l.stream.Send(&warehouse_iface.WarehouseCreateResponse{
		Message: string(p),
	})
	return len(p), err
}

// WarehouseCreate implements [warehouse_ifaceconnect.WarehouseServiceHandler].
//
// It is a server-streaming "long running task" (see docs/code-implementation-guideline.md):
// each step is logged to the response stream. The warehouse id is server-assigned — equal
// to a freshly created team's id — so creation runs three writes in one transaction:
//  1. create a warehouse team,
//  2. create the warehouse with primary key = team id,
//  3. add the caller as team owner.
//
// Auth (ROLE_ROOT/ROLE_ADMIN, per the request_policy) is enforced by the access interceptor
// — including for this streaming RPC — which also places the caller identity in ctx.
func (w *warehouseServiceImpl) WarehouseCreate(
	ctx context.Context,
	req *connect.Request[warehouse_iface.WarehouseCreateRequest],
	stream *connect.ServerStream[warehouse_iface.WarehouseCreateResponse],
) error {
	caller, err := access_interceptors.GetIdentityFromCtx(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}
	callerID := uint(caller.IdentityId)

	var logwriter io.Writer = &warehouseCreateLogger{stream: stream}
	logger := slog.New(slog.NewTextHandler(logwriter, nil))

	var newID uint
	err = w.db.
		WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {
			var txErr error
			newID, txErr = createWarehouseWithTeam(tx, logger, callerID, req.Msg)
			return txErr
		})
	if err != nil {
		logger.Error("warehouse create failed", "err", err)
		return err
	}

	return stream.Send(&warehouse_iface.WarehouseCreateResponse{
		Id:      uint64(newID),
		Message: "warehouse created",
	})
}

// createWarehouseWithTeam performs the three-step create inside the given transaction and
// returns the new warehouse/team id. Split from the streaming handler so the core logic is
// unit-testable without a connect stream. Each step is logged through logger (which, in the
// handler, streams progress to the client).
func createWarehouseWithTeam(
	tx *gorm.DB,
	logger *slog.Logger,
	callerID uint,
	pay *warehouse_iface.WarehouseCreateRequest,
) (uint, error) {
	// 1. create the warehouse team (its id becomes the warehouse id).
	logger.Info("creating team", "code", pay.TeamCode)
	team := &db_models.Team{
		Type:     db_models.WarehouseTeamType,
		Name:     pay.Name,
		TeamCode: db_models.TeamCode(strings.ToUpper(pay.TeamCode)),
	}
	err := tx.Create(team).Error
	if err != nil {
		return 0, err
	}
	logger.Info("team created", "teamID", team.ID)

	// 2. create the warehouse with primary key = team id.
	logger.Info("creating warehouse", "id", team.ID)
	wh := &db_models.Warehouse{
		ID:            team.ID,
		Name:          pay.Name,
		Desc:          pay.Desc,
		Address:       pay.Address,
		IsFull:        pay.IsFull,
		IsClosed:      pay.IsClosed,
		UseFixedFee:   pay.UseFixedFee,
		FeeFix:        pay.FeeFix,
		FeePercent:    pay.FeePercent,
		MaxFee:        pay.MaxFee,
		OpenTime:      parseHHMM(pay.OpenTime),
		CloseTime:     parseHHMM(pay.CloseTime),
		CloseOrder:    parseHHMM(pay.CloseOrder),
		Created:       time.Now(),
		WarehouseStat: &db_models.WarehouseStat{},
	}
	err = tx.Create(wh).Error
	if err != nil {
		return 0, err
	}
	logger.Info("warehouse created", "id", wh.ID)

	// 3. add the caller as team owner.
	logger.Info("assigning owner", "userID", callerID)
	owner := &user_models.UserTeamRole{
		TeamID:    team.ID,
		UserID:    callerID,
		Role:      role_base.Role_ROLE_WAREHOUSE_OWNER,
		Alias:     "own",
		CreatedAt: time.Now(),
	}
	err = tx.Create(owner).Error
	if err != nil {
		return 0, err
	}
	logger.Info("owner assigned", "teamID", team.ID, "userID", callerID)

	return team.ID, nil
}
