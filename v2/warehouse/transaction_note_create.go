package warehouse

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/access_iface/v1"
	"github.com/pdcgo/schema/services/warehouse_iface/v1"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_models"
	"gorm.io/gorm"
)

func ProtoToNoteType(txNoteType warehouse_iface.TransactionNoteType) (db_models.NoteType, error) {
	var err error
	var noteType db_models.NoteType

	switch txNoteType {
	case warehouse_iface.TransactionNoteType_TRANSACTION_NOTE_TYPE_PROBLEM:
		noteType = db_models.NoteProblem
	case warehouse_iface.TransactionNoteType_TRANSACTION_NOTE_TYPE_COMMON:
		noteType = db_models.NoteCommon
	case warehouse_iface.TransactionNoteType_TRANSACTION_NOTE_TYPE_BROKEN:
		noteType = db_models.NoteBroken
	case warehouse_iface.TransactionNoteType_TRANSACTION_NOTE_TYPE_RETURN:
		noteType = db_models.NoteReturn
	case warehouse_iface.TransactionNoteType_TRANSACTION_NOTE_TYPE_CANCEL:
		noteType = db_models.NoteCancel
	default:
		err = errors.New("note type not supported")
	}

	return noteType, err
}

// WarehouseIDs implements warehouse_ifaceconnect.WarehouseServiceHandler.
func (w *warehouseServiceImpl) TransactionNoteCreate(
	ctx context.Context,
	req *connect.Request[warehouse_iface.TransactionNoteCreateRequest],
) (*connect.Response[warehouse_iface.TransactionNoteCreateResponse], error) {

	source, err := custom_connect.GetRequestSource(ctx)
	if err != nil {
		return nil, err
	}

	identity := w.auth.AuthIdentityFromHeader(req.Header())
	err = identity.Err()
	if err != nil {
		return nil, err
	}

	db := w.db.WithContext(ctx)
	pay := req.Msg

	var domainId = source.TeamId
	if source.RequestFrom == access_iface.RequestFrom_REQUEST_FROM_ADMIN {
		domainId = pay.TeamId
	}

	result := warehouse_iface.TransactionNoteCreateResponse{
		Ids: []uint64{},
	}

	var notes = []*db_models.InvNote{}
	err = db.Transaction(func(tx *gorm.DB) error {

		for _, note := range pay.Notes {

			if note.Type != warehouse_iface.TransactionNoteType_TRANSACTION_NOTE_TYPE_COMMON {
				continue
			}

			noteType, err := ProtoToNoteType(note.Type)
			if err != nil {
				return err
			}
			notes = append(notes, &db_models.InvNote{
				InvTransactionID: uint(pay.TxId),
				NoteType:         noteType,
				NoteText:         note.NoteText,
			})
		}

		var invTx db_models.InvTransaction
		err = tx.
			Model(&db_models.InvTransaction{}).
			Where("team_id = ?", domainId).
			First(&invTx, pay.TxId).
			Error
		if err != nil {
			return err
		}

		noteTx := tx.
			Model(&db_models.InvNote{}).
			Where("inv_transaction_id = ?", invTx.ID).
			Where("note_type = ?", db_models.NoteCommon)
		if pay.OrderId != 0 {
			noteTx = noteTx.Where("order_id = ?", pay.OrderId)
		}
		err = noteTx.Delete(&db_models.InvNote{}).Error
		if err != nil {
			return err
		}

		if len(notes) > 0 {
			return tx.Model(&db_models.InvNote{}).Create(notes).Error
		}
		return nil
	})

	for _, note := range notes {
		result.Ids = append(result.Ids, uint64(note.ID))
	}
	return connect.NewResponse(&result), err
}
