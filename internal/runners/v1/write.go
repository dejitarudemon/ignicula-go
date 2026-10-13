package runner_v1

import (
	"context"
	"fmt"
	"time"

	"github.com/dejitarudemon/ignicula-go/internal/runners"
	table_v1 "github.com/dejitarudemon/ignicula-go/internal/table/v1"
	"github.com/dejitarudemon/ignicula-go/values"
	"github.com/dejitarudemon/ignicula-wire/v1/body"
	"github.com/dejitarudemon/ignicula-wire/v1/body/bodies"
	protocolerrs "github.com/dejitarudemon/ignicula-wire/v1/err/errs"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

// Write stores value under key and waits for a Write answer or a protocol error
// answer. Transient server errors ([protocolerrs.ErrorRequestsConflict],
// [protocolerrs.ErrorInternalError]) are retried up to the configured attempt
// limit. The client value is converted through the runner's v1 adapter.
func (r *Runner) Write(ctx context.Context, key []byte, value values.Value) error {
	if r.activating.Load() {
		return runners.NewInternalError(runners.ErrActivating)
	}

	for attempt := 0; attempt < r.timeouts.MaxAttempts; attempt++ {
		retry, err := r.writeOnce(ctx, attempt, key, value)
		if err != nil {
			return err
		}
		if !retry {
			return nil
		}
	}

	return runners.NewInternalError(runners.ErrWriteExhausted)
}

// writeOnce registers, sends, and interprets one write attempt.
// retry is true when the server reported a transient error and another
// attempt is still allowed.
func (r *Runner) writeOnce(parent context.Context, attempt int, key []byte, value values.Value) (retry bool, err error) {
	backoff(time.Duration(attempt) * r.timeouts.Backoff)

	ctx, cancel := context.WithTimeout(parent, r.timeouts.Write)
	defer cancel()

	id, ch, err := r.sendWrite(ctx, key, value)
	if err != nil {
		return false, err
	}

	response, err := r.await(ctx, ch)
	if err != nil {
		r.table.Terminate(id)
		return false, err
	}

	transient, err := r.interpretWriteResponse(response)
	if err != nil {
		return false, err
	}
	if transient == nil {
		return false, nil
	}
	if attempt+1 < r.timeouts.MaxAttempts {
		return true, nil
	}
	return false, runners.NewServerError(transient)
}

// sendWrite converts value, registers a Write request, encodes the frame, and
// writes it to the outgoing requests channel. On failure before a successful
// send it terminates the request id and returns [runners.InternalError].
func (r *Runner) sendWrite(ctx context.Context, key []byte, value values.Value) (fields.RequestID, table_v1.ResponseV1Chan, error) {
	wireValue, err := r.adapter.To(value)
	if err != nil {
		return 0, nil, runners.NewInternalError(err)
	}

	ch := make(table_v1.ResponseV1Chan, 1)

	id, err := r.table.RegisterRequest(fields.Write, ch, 1)
	if err != nil {
		return 0, nil, runners.NewInternalError(err)
	}

	frm, err := r.builder.NewWrite(id, fields.Key(key), wireValue)
	if err != nil {
		r.table.Terminate(id)
		return 0, nil, runners.NewInternalError(err)
	}

	raw, err := r.encodeFrame(frm, nil)
	if err != nil {
		r.table.Terminate(id)
		return 0, nil, runners.NewInternalError(err)
	}

	if err := r.send(ctx, raw); err != nil {
		r.table.Terminate(id)
		return 0, nil, runners.NewInternalError(err)
	}

	return id, ch, nil
}

// interpretWriteResponse classifies a completed write response.
// On success both return values are nil. A transient protocol error
// ([protocolerrs.ErrorRequestsConflict], [protocolerrs.ErrorInternalError])
// is returned as transient with a nil error so the caller can retry or wrap
// it as [runners.ServerError]. Other failures are returned only as err.
func (r *Runner) interpretWriteResponse(response table_v1.ResponseV1) (transient error, err error) {
	if err := response.Err; err != nil {
		return nil, err
	}

	if response.Response.Body == nil {
		return nil, runners.NewProtocolViolationByServer("empty write response body")
	}

	if response.Response.Body.Command() != fields.Answer {
		return nil, runners.NewProtocolViolationByServer(
			fmt.Sprintf("server sent %v instead of Answer", response.Response.Body.Command()),
		)
	}

	answer, ok := response.Response.Body.(body.Answer)
	if !ok {
		return nil, runners.NewProtocolViolationByServer(
			fmt.Sprintf("server sent non-answer body %T", response.Response.Body),
		)
	}

	if answer.IsResponseTo() == fields.Write {
		if _, ok := answer.(bodies.WriteAnswer); !ok {
			return nil, runners.NewProtocolViolationByServer(
				fmt.Sprintf("server sent non-write answer %T", answer),
			)
		}
		return nil, nil
	}

	protoerr, ok := answer.(bodies.ErrorAnswer)
	if !ok || protoerr.Err == nil {
		return nil, runners.NewProtocolViolationByServer(
			fmt.Sprintf("server sent unexpected answer to %v", answer.IsResponseTo()),
		)
	}

	switch protoerr.Err.(type) {
	case protocolerrs.ErrorRequestsConflict, protocolerrs.ErrorInternalError:
		return protoerr.Err, nil
	}

	return nil, runners.NewServerError(protoerr.Err)
}
