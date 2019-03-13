package middleware

import (
	"x/src/middleware/notify"
	"x/src/middleware/types"
)

func InitMiddleware() error {

	types.InitSerialzation()
	notify.BUS = notify.NewBus()

	return nil
}
