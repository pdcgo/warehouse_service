package warehouse_service

import (
	"context"

	"github.com/pdcgo/shared/interfaces/warehouse_iface"
)

type warehouseFinImpl struct {
	warehouse_iface.UnimplementedWarehouseFinanceServiceServer
}

// ExpenseAccountCreate implements warehouse_iface.WarehouseFinanceServiceServer.
func (w *warehouseFinImpl) ExpenseAccountCreate(context.Context, *warehouse_iface.ExpenseAccountCreateReq) (*warehouse_iface.ExpenseAccount, error) {
	panic("unimplemented")
}

// ExpenseAccountEdit implements warehouse_iface.WarehouseFinanceServiceServer.
func (w *warehouseFinImpl) ExpenseAccountEdit(context.Context, *warehouse_iface.ExpenseAccountEditReq) (*warehouse_iface.ExpenseAccount, error) {
	panic("unimplemented")
}

// ExpenseAccountGet implements warehouse_iface.WarehouseFinanceServiceServer.
func (w *warehouseFinImpl) ExpenseAccountGet(context.Context, *warehouse_iface.ExpenseAccountGetReq) (*warehouse_iface.ExpenseAccount, error) {
	panic("unimplemented")
}

// ExpenseAccountList implements warehouse_iface.WarehouseFinanceServiceServer.
func (w *warehouseFinImpl) ExpenseAccountList(context.Context, *warehouse_iface.ExpenseAccountListReq) (*warehouse_iface.ExpenseAccountListRes, error) {
	panic("unimplemented")
}

// ExpenseHistoryAdd implements warehouse_iface.WarehouseFinanceServiceServer.
func (w *warehouseFinImpl) ExpenseHistoryAdd(context.Context, *warehouse_iface.ExpenseHistoryAddReq) (*warehouse_iface.ExpenseHistoryAddRes, error) {
	panic("unimplemented")
}

// ExpenseHistoryList implements warehouse_iface.WarehouseFinanceServiceServer.
func (w *warehouseFinImpl) ExpenseHistoryList(context.Context, *warehouse_iface.ExpenseHistoryListReq) (*warehouse_iface.ExpenseHistoryListRes, error) {
	panic("unimplemented")
}

// ExpenseReportDaily implements warehouse_iface.WarehouseFinanceServiceServer.
func (w *warehouseFinImpl) ExpenseReportDaily(context.Context, *warehouse_iface.ExpenseReportDailyReq) (*warehouse_iface.ExpenseReportDailyRes, error) {
	panic("unimplemented")
}

func NewWarehouseFinanceService() warehouse_iface.WarehouseFinanceServiceServer {
	return &warehouseFinImpl{}
}
