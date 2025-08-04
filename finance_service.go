package warehouse_service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"github.com/pdcgo/shared/interfaces/warehouse_iface"
	"github.com/pdcgo/warehouse_service/models"
	"github.com/pdcgo/warehouse_service/warehouse_mutations"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func NewWarehouseFinanceService(db *gorm.DB, auth authorization_iface.Authorization) warehouse_iface.WarehouseFinanceServiceServer {
	return &warehouseFinImpl{
		db:   db,
		auth: auth,
	}
}

type warehouseFinImpl struct {
	warehouse_iface.UnimplementedWarehouseFinanceServiceServer
	db   *gorm.DB
	auth authorization_iface.Authorization
}

// ExpenseAccountCreate implements warehouse_iface.WarehouseFinanceServiceServer.
func (w *warehouseFinImpl) ExpenseAccountCreate(ctx context.Context, payload *warehouse_iface.ExpenseAccountCreateReq) (*warehouse_iface.WarehouseExpenseAccount, error) {
	identity := ctx.Value("identity").(*authorization.JwtIdentity)
	err := w.auth.HasPermission(identity, authorization_iface.CheckPermissionGroup{
		&models.WareExpenseAccount{}: &authorization_iface.CheckPermission{
			DomainID: uint(payload.DomainId),
			Actions:  []authorization_iface.Action{authorization_iface.Create},
		},
	})
	if err != nil {
		return nil, err
	}

	var result *warehouse_iface.WarehouseExpenseAccount

	name := strings.Trim(payload.Name, " ")
	numberId := strings.Trim(payload.NumberId, " ")

	db := w.db.WithContext(ctx)
	err = db.Transaction(func(tx *gorm.DB) error {
		createAccount := warehouse_mutations.NewCreateWarehouseExpenseAccount(tx, identity)
		data, err := createAccount.
			Create(uint(payload.WarehouseId), uint(payload.AccountTypeId), name, numberId, payload.IsOpsAccount)
		if err != nil {
			return err
		}

		result = &warehouse_iface.WarehouseExpenseAccount{
			Id:            uint64(data.AccountID),
			WarehouseId:   uint64(data.WarehouseID),
			AccountTypeId: uint64(data.Account.AccountTypeID),
			Name:          data.Account.Name,
			NumberId:      data.Account.NumberID,
			IsOpsAccount:  data.IsOpsAccount,
			Disabled:      data.Account.Disabled,
			CreatedAt:     timestamppb.New(data.Account.CreatedAt),
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ExpenseAccountEdit implements warehouse_iface.WarehouseFinanceServiceServer.
func (w *warehouseFinImpl) ExpenseAccountEdit(ctx context.Context, payload *warehouse_iface.ExpenseAccountEditReq) (*warehouse_iface.WarehouseExpenseAccount, error) {
	identity := ctx.Value("identity").(*authorization.JwtIdentity)
	err := w.auth.HasPermission(identity, authorization_iface.CheckPermissionGroup{
		&models.WareExpenseAccount{}: &authorization_iface.CheckPermission{
			DomainID: uint(payload.DomainId),
			Actions:  []authorization_iface.Action{authorization_iface.Update},
		},
	})
	if err != nil {
		return nil, err
	}

	name := strings.Trim(payload.Name, " ")
	numberId := strings.Trim(payload.NumberId, " ")

	var result *warehouse_iface.WarehouseExpenseAccount
	db := w.db.WithContext(ctx)
	err = db.Transaction(func(tx *gorm.DB) error {
		accountService := warehouse_mutations.NewExpenseAccountService(tx, uint(payload.WarehouseId))
		data, err := accountService.
			GetByQuery(true, func(tx *gorm.DB) *gorm.DB {
				return tx.Where("ware_expense_accounts.id = ?", payload.AccountId)
			})
		if err != nil {
			return err
		}

		err = accountService.Update(uint(payload.AccountTypeId), name, numberId)
		if err != nil {
			return err
		}

		result = &warehouse_iface.WarehouseExpenseAccount{
			Id:            uint64(data.Account.ID),
			WarehouseId:   uint64(payload.WarehouseId),
			AccountTypeId: uint64(data.Account.AccountTypeID),
			Disabled:      data.Account.Disabled,
			Name:          data.Account.Name,
			NumberId:      data.Account.NumberID,
			IsOpsAccount:  data.IsOpsAccount,
			CreatedAt:     timestamppb.New(data.Account.CreatedAt),
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ExpenseAccountGet implements warehouse_iface.WarehouseFinanceServiceServer.
func (w *warehouseFinImpl) ExpenseAccountGet(ctx context.Context, query *warehouse_iface.ExpenseAccountGetReq) (*warehouse_iface.WarehouseExpenseAccount, error) {

	var result *warehouse_iface.WarehouseExpenseAccount
	err := w.db.Transaction(func(tx *gorm.DB) error {
		accountService := warehouse_mutations.NewExpenseAccountService(tx, uint(query.WarehouseId))
		data, err := accountService.
			GetByQuery(false, func(tx *gorm.DB) *gorm.DB {
				return tx.
					Where("ware_expense_accounts.id = ?", query.Id).
					Where("ware_expense_account_warehouses.warehouse_id = ?", query.WarehouseId).
					Where("ware_expense_account_warehouses.is_ops_account = ?", query.IsOpsAccount)
			})
		if err != nil {
			return err
		}

		result = &warehouse_iface.WarehouseExpenseAccount{
			Id:           uint64(data.Account.ID),
			NumberId:     data.Account.NumberID,
			Name:         data.Account.Name,
			IsOpsAccount: data.IsOpsAccount,
			WarehouseId:  uint64(query.WarehouseId),
			CreatedAt:    timestamppb.New(data.Account.CreatedAt),
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ExpenseAccountList implements warehouse_iface.WarehouseFinanceServiceServer.
func (w *warehouseFinImpl) ExpenseAccountList(ctx context.Context, query *warehouse_iface.ExpenseAccountListReq) (*warehouse_iface.ExpenseAccountListRes, error) {
	data := []*models.WareExpenseAccountWarehouse{}

	db := w.db.WithContext(ctx)
	sqlQuery := db.Model(&models.WareExpenseAccountWarehouse{}).
		Joins("JOIN ware_expense_accounts ON ware_expense_accounts.id = ware_expense_account_warehouses.account_id")
	if query.WarehouseId != 0 {
		sqlQuery = sqlQuery.Where("ware_expense_account_warehouses.warehouse_id = ?", query.WarehouseId)
	}
	if query.NumberId != "" {
		sqlQuery = sqlQuery.Where("ware_expense_accounts.number_id LIKE ?", "%"+query.NumberId+"%")
	}
	if query.Name != "" {
		sqlQuery = sqlQuery.Where("LOWER(ware_expense_accounts.name) LIKE ?", "%"+query.Name+"%")
	}
	if query.IsOpsAccount {
		sqlQuery = sqlQuery.Where("ware_expense_account_warehouses.is_ops_account = ?", query.IsOpsAccount)
	}

	err := sqlQuery.
		Preload("Account").
		Find(&data).Error
	if err != nil {
		return nil, err
	}

	results := warehouse_iface.ExpenseAccountListRes{
		Data: make([]*warehouse_iface.WarehouseExpenseAccount, len(data)),
	}
	for i, v := range data {
		if v.Account == nil {
			return nil, errors.New("account not found")
		}

		results.Data[i] = &warehouse_iface.WarehouseExpenseAccount{
			Id:            uint64(v.AccountID),
			WarehouseId:   uint64(v.WarehouseID),
			AccountTypeId: uint64(v.Account.AccountTypeID),
			Name:          v.Account.Name,
			NumberId:      v.Account.NumberID,
			Disabled:      v.Account.Disabled,
			IsOpsAccount:  v.IsOpsAccount,
			CreatedAt:     timestamppb.New(v.Account.CreatedAt),
		}
	}

	return &results, err
}

// ExpenseHistoryAdd implements warehouse_iface.WarehouseFinanceServiceServer.
func (w *warehouseFinImpl) ExpenseHistoryAdd(ctx context.Context, payload *warehouse_iface.ExpenseHistoryAddReq) (*warehouse_iface.ExpenseHistoryAddRes, error) {
	identity := ctx.Value("identity").(*authorization.JwtIdentity)

	db := w.db.WithContext(ctx)
	err := db.Transaction(func(tx *gorm.DB) error {
		histService := warehouse_mutations.NewExpenseHistService(tx, identity)

		_, err := histService.GetAccount(uint(payload.AccountId), uint(payload.WarehouseId))
		if err != nil {
			return err
		}

		return histService.
			Create(identity.From, &warehouse_mutations.CreateExpensePayload{
				ExpenseType: models.ExpenseType(payload.ExpenseType),
				At:          payload.At.AsTime(),
				Amount:      payload.Amount,
				Note:        payload.Note,
			})
	})
	if err != nil {
		return nil, err
	}

	return &warehouse_iface.ExpenseHistoryAddRes{}, nil
}

// ExpenseHistoryAdd implements warehouse_iface.WarehouseFinanceServiceServer.
func (w *warehouseFinImpl) ExpenseHistoryEdit(ctx context.Context, payload *warehouse_iface.ExpenseHistoryEditReq) (*warehouse_iface.ExpenseHistoryEditRes, error) {
	identity := ctx.Value("identity").(*authorization.JwtIdentity)

	db := w.db.WithContext(ctx)
	err := db.Transaction(func(tx *gorm.DB) error {
		histService := warehouse_mutations.NewExpenseHistService(tx, identity)

		_, err := histService.GetExpense(uint(payload.HistId))
		if err != nil {
			return err
		}

		return histService.
			Update(identity.From, &warehouse_mutations.UpdateWareExpenseHistPayload{
				WarehouseID: uint(payload.WarehouseId),
				AccountID:   uint(payload.AccountId),
				ExpenseType: models.ExpenseType(payload.ExpenseType),
				Amount:      payload.Amount,
				At:          payload.At.AsTime(),
				Note:        payload.Note,
			})
	})
	if err != nil {
		return nil, err
	}

	return &warehouse_iface.ExpenseHistoryEditRes{}, nil
}

// ExpenseHistoryList implements warehouse_iface.WarehouseFinanceServiceServer.
func (w *warehouseFinImpl) ExpenseHistoryList(ctx context.Context, query *warehouse_iface.ExpenseHistoryListReq) (*warehouse_iface.ExpenseHistoryListRes, error) {
	result := warehouse_iface.ExpenseHistoryListRes{}

	db := w.db.WithContext(ctx)
	sqlQuery := db.Model(&models.WareExpenseHistory{}).
		Joins("JOIN ware_expense_account_warehouses ON ware_expense_account_warehouses.account_id = ware_expense_histories.account_id AND ware_expense_account_warehouses.warehouse_id = ware_expense_histories").
		Where("ware_expense_account_warehouses.is_ops_account = ?", query.IsOpsAccount)
	if query.WarehouseId != 0 {
		sqlQuery = sqlQuery.Where("ware_expense_histories.warehouse_id = ?", query.WarehouseId)
	}
	if query.AccountId != 0 {
		sqlQuery = sqlQuery.Where("ware_expense_histories.account_id = ?", query.AccountId)
	}
	if query.ExpenseType != "" {
		sqlQuery = sqlQuery.Where("ware_expense_histories.expense_type = ?", query.ExpenseType)
	}
	if query.StartDate != 0 {
		unixMilli := time.UnixMilli(query.StartDate)
		startDay := time.Date(unixMilli.Year(), unixMilli.Month(), unixMilli.Day(), 0, 0, 0, 0, unixMilli.Location())
		sqlQuery = sqlQuery.Where("ware_expense_histories.created_at >= ?", startDay)
	}
	if query.EndDate != 0 {
		unixMilli := time.UnixMilli(query.EndDate)
		endDay := time.Date(unixMilli.Year(), unixMilli.Month(), unixMilli.Day()+1, 0, 0, 0, -1, unixMilli.Location())
		sqlQuery = sqlQuery.Where("ware_expense_histories.created_at <= ?", endDay)
	}

	data := []*models.WareExpenseHistory{}
	err := sqlQuery.
		Find(&data).Error
	if err != nil {
		return nil, err
	}

	result.Data = make([]*warehouse_iface.WarehouseExpenseHistory, len(data))
	for i, v := range data {
		result.Data[i] = &warehouse_iface.WarehouseExpenseHistory{
			Id:          uint64(v.ID),
			AccountId:   uint64(v.AccountID),
			WarehouseId: uint64(v.WarehouseID),
			ExpenseType: string(v.ExpenseType),
			Amount:      v.Amount,
		}
	}

	return &result, nil
}

// ExpenseReportDaily implements warehouse_iface.WarehouseFinanceServiceServer.
func (w *warehouseFinImpl) ExpenseReportDaily(context.Context, *warehouse_iface.ExpenseReportDailyReq) (*warehouse_iface.ExpenseReportDailyRes, error) {
	panic("unimplemented")
}
