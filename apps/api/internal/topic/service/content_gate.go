package service

import (
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/errors"
)

func errContentBlocked() *errors.AppError {
	return gate.ErrContentBlocked()
}
