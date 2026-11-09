package domain

import "errors"

var ErrQuotaExceeded = errors.New("monthly AI quota exceeded")
var ErrIdempotencyConflict = errors.New("idempotency key reused with a different request")
