package bwlimit

import (
	"context"
)

type Limiter struct {
	Reads  *Operation
	Writes *Operation
}

func NewLimiter(ctx context.Context) *Limiter {
	return &Limiter{
		Reads:  NewOperation(ctx, true),
		Writes: NewOperation(ctx, false),
	}
}
