package outbound

import (
	"github.com/pdcgo/shared/interfaces/authorization_iface"
	"gorm.io/gorm"
)

type outboundImpl struct {
	db   *gorm.DB
	auth authorization_iface.Authorization
}

func NewOutboundService(
	db *gorm.DB,
	auth authorization_iface.Authorization,
) *outboundImpl {
	return &outboundImpl{
		db:   db,
		auth: auth,
	}
}
