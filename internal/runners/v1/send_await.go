package runner_v1

import (
	"context"

	"github.com/dejitarudemon/ignicula-go/internal/runners"
	table_v1 "github.com/dejitarudemon/ignicula-go/internal/table/v1"
)

func (r *Runner) send(ctx context.Context, request []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.close:
		return runners.ErrRunnerClosed
	case r.requests <- request:
		return nil
	}
}

func (r *Runner) await(ctx context.Context, ch table_v1.ResponseV1Chan) (table_v1.ResponseV1, error) {
	var response table_v1.ResponseV1

	for {
		select {
		case <-ctx.Done():
			return table_v1.ResponseV1{}, runners.NewInternalError(ctx.Err())
		case <-r.close:
			return table_v1.ResponseV1{}, runners.NewInternalError(runners.ErrRunnerClosed)
		case result, ok := <-ch:
			if !ok {
				return response, nil
			}
			response = result
		}
	}
}
