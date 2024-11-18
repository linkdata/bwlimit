package bwlimit

import (
	"context"
)

type Limiter struct {
	Reads  *Operation
	Writes *Operation
}

const secparts = 100

func NewLimiter(ctx context.Context) *Limiter {
	return &Limiter{
		Reads:  NewOperation(ctx, true),
		Writes: NewOperation(ctx, false),
	}
}
