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

func NoteTypeToProto(noteType db_models.NoteType) (warehouse_iface.TransactionNoteType, error) {
	var err error
	var txNoteType warehouse_iface.TransactionNoteType

	switch noteType {
	case db_models.NoteProblem:
		txNoteType = warehouse_iface.TransactionNoteType_TRANSACTION_NOTE_TYPE_PROBLEM
	case db_models.NoteCommon:
		txNoteType = warehouse_iface.TransactionNoteType_TRANSACTION_NOTE_TYPE_COMMON
	case db_models.NoteBroken:
		txNoteType = warehouse_iface.TransactionNoteType_TRANSACTION_NOTE_TYPE_BROKEN
	case db_models.NoteReturn:
		txNoteType = warehouse_iface.TransactionNoteType_TRANSACTION_NOTE_TYPE_RETURN
	case db_models.NoteCancel:
		txNoteType = warehouse_iface.TransactionNoteType_TRANSACTION_NOTE_TYPE_CANCEL
	default:
		err = errors.New("note type not supported")
	}

	return txNoteType, err
}

// WarehouseIDs implements warehouse_ifaceconnect.WarehouseServiceHandler.
func (w *warehouseServiceImpl) TransactionNoteList(
	ctx context.Context,
	req *connect.Request[warehouse_iface.TransactionNoteListRequest],
) (*connect.Response[warehouse_iface.TransactionNoteListResponse], error) {

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

	result := warehouse_iface.TransactionNoteListResponse{
		List: []*warehouse_iface.Note{},
	}

	err = db.Transaction(func(tx *gorm.DB) error {

		var invTx db_models.InvTransaction
		err = tx.
			Model(&db_models.InvTransaction{}).
			Where("team_id = ?", domainId).
			First(&invTx, pay.TxId).
			Error
		if err != nil {
			return err
		}

		noteTx := tx.Model(&invTx)
		if pay.OrderId != 0 {
			noteTx = noteTx.Where("order_id = ?", pay.OrderId)
		}

		var notes []db_models.InvNote
		err = noteTx.Association("InvNotes").Find(&notes)

		for _, note := range notes {
			notType, err := NoteTypeToProto(note.NoteType)
			if err != nil {
				return err
			}

			result.List = append(result.List, &warehouse_iface.Note{
				Type:     notType,
				NoteText: note.NoteText,
			})
		}

		return err
	})

	return connect.NewResponse(&result), err
}
