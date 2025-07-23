package warehouse_mutations

import (
	"errors"
	"fmt"
	"time"

	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/identity_iface"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type TransactionLogNewEntry interface {
	SetStatus(status db_models.InvTxStatus) TransactionLogNewEntry
	SetActionType(tipe db_models.ActionType) TransactionLogNewEntry
	SetTxID(txID uint) TransactionLogNewEntry
	SetBeforeUpdatedData(key db_models.ActionType, data any) TransactionLogNewEntry
	Data() *db_models.InvTimestamp
	Do() error
}

type txLogImpl struct {
	db    *gorm.DB
	agent identity_iface.Agent
	data  *db_models.InvTimestamp
}

// SetLogType implements TransactionLogNewEntry.
func (t *txLogImpl) SetActionType(action db_models.ActionType) TransactionLogNewEntry {
	t.data.ActionType = action
	return t
}

// SetTxID implements TransactionLogNewEntry.
func (t *txLogImpl) SetTxID(txID uint) TransactionLogNewEntry {
	t.data.TxID = txID
	return t
}

// Data implements TransactionLog.
func (t *txLogImpl) Data() *db_models.InvTimestamp {
	return t.data
}

// Do implements TransactionLog.
func (t *txLogImpl) Do() error {
	if t.data == nil {
		return errors.New("log data nil")
	}

	if t.data.ActionType == db_models.ActionEmpty {
		return errors.New("action_type empty not allowed")
	}

	if t.data.TxID == 0 {
		return errors.New("tx_id empty not allowed")
	}

	switch t.data.ActionType {
	case db_models.ActionEditPrice:
		if t.data.BeforeUpdated[string(db_models.ActionEditPrice)] == nil {
			return fmt.Errorf("before_updated of key %s is empty", db_models.ActionEditPrice)
		}

	}

	err := t.db.Save(t.data).Error
	return err
}

// LogStatus implements TransactionLog.
func (t *txLogImpl) SetStatus(status db_models.InvTxStatus) TransactionLogNewEntry {
	t.data.Status = status
	return t
}

// SetBeforeUpdatedData implements TransactionLog.
func (t *txLogImpl) SetBeforeUpdatedData(key db_models.ActionType, data any) TransactionLogNewEntry {
	t.data.BeforeUpdated[string(key)] = data
	return t
}

func NewTransactionLogNewEntry(db *gorm.DB, agent identity_iface.Agent) TransactionLogNewEntry {
	return &txLogImpl{
		db:    db,
		agent: agent,
		data: &db_models.InvTimestamp{
			UserID:        agent.GetUserID(),
			From:          agent.GetAgentType(),
			BeforeUpdated: datatypes.JSONMap{},
			Timestamp:     time.Now(),
		},
	}
}
