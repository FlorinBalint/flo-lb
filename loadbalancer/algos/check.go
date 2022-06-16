package algos

import (
	"context"
	"time"
)

type Checker struct {
	fn         func(context.Context, *Backend)
	period     time.Duration
	beSupplier func() []*Backend
}

func NewChecker(fn func(context.Context, *Backend), period time.Duration) *Checker {
	return &Checker{
		fn:     fn,
		period: period,
	}
}

func (chk *Checker) runInBackground(ctx context.Context) {
	go func() {
		t := time.NewTicker(chk.period)
		defer t.Stop()
		for true {
			select {
			case <-ctx.Done():
				break
			case <-t.C:
				// range makes a slice copy, so changing backends is safe
				for _, be := range chk.beSupplier() {
					go chk.fn(ctx, be)
				}
			}
		}
	}()
}
