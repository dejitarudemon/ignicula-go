package runner_v1

import (
	"context"
	"fmt"

	"github.com/dejitarudemon/ignicula-go/internal/runners"
	table_v1 "github.com/dejitarudemon/ignicula-go/internal/table/v1"
	"github.com/dejitarudemon/ignicula-wire/v1/body"
	"github.com/dejitarudemon/ignicula-wire/v1/body/bodies"
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
	protocolerrs "github.com/dejitarudemon/ignicula-wire/v1/err/errs"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

// Ping sends a protocol ping and waits for a Ping answer or a protocol error
// answer. Transient server errors ([protocolerrs.ErrorRequestsConflict],
// [protocolerrs.ErrorInternalError]) are retried up to the configured attempt
// limit. A valid ErrorAnswer is returned as [runners.ServerError]. A response
// that breaks the protocol is [runners.ProtocolViolationByServer].
func (r *Runner) Ping(ctx context.Context) error {
	if r.activating.Load() {
		return runners.NewInternalError(runners.ErrActivating)
	}

	for attempt := 0; attempt < r.timeouts.MaxAttempts; attempt++ {
		retry, err := r.pingOnce(ctx, attempt)
		if err != nil {
			return err
		}
		if !retry {
			return nil
		}
	}

	// MaxAttempts is at least 1 after [NewRunner]; interpretPingResponse
	// returns ServerError on the last retryable attempt instead of retry.
	return runners.NewInternalError(runners.ErrPingExhausted)
}

// pingOnce registers, sends, and interprets one ping attempt.
// retry is true when the server reported a transient error and another
// attempt is still allowed.
func (r *Runner) pingOnce(parent context.Context, attempt int) (retry bool, err error) {
	timeout := calculateTimeout(r.timeouts.Ping, r.timeouts.Backoff, attempt)

	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	id, ch, err := r.sendPing(ctx)
	if err != nil {
		return false, err
	}

	response, err := r.await(ctx, ch)
	if err != nil {
		r.table.Terminate(id)
		return false, err
	}

	transient, err := interpretPingResponse(response)
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

// sendPing registers a Ping request, encodes the frame, and writes it to the
// outgoing requests channel. On failure before a successful send it terminates
// the request id and returns [runners.InternalError].
func (r *Runner) sendPing(ctx context.Context) (fields.RequestID, table_v1.ResponseV1Chan, error) {
	ch := make(table_v1.ResponseV1Chan, 1)

	id, err := r.table.RegisterRequest(fields.Ping, ch, 1)
	if err != nil {
		return 0, nil, runners.NewInternalError(err)
	}

	frm, err := r.builder.NewPing(id)
	if err != nil {
		r.table.Terminate(id)
		return 0, nil, runners.NewInternalError(err)
	}

	buf := buffer.Slice{}
	buf.Preallocate(frm.Size())
	if err := frm.Encode(&buf, nil); err != nil {
		r.table.Terminate(id)
		return 0, nil, runners.NewInternalError(err)
	}

	if err := r.send(ctx, buf.Bytes()); err != nil {
		r.table.Terminate(id)
		return 0, nil, runners.NewInternalError(err)
	}

	return id, ch, nil
}

// interpretPingResponse classifies a completed ping response.
// On success both return values are nil. A transient protocol error
// ([protocolerrs.ErrorRequestsConflict], [protocolerrs.ErrorInternalError])
// is returned as transient with a nil error so the caller can retry or wrap
// it as [runners.ServerError]. Other failures are returned only as err.
func interpretPingResponse(response table_v1.ResponseV1) (transient error, err error) {
	if err := response.Err; err != nil {
		return nil, err
	}

	if response.Response.Body == nil {
		return nil, runners.NewProtocolViolationByServer("empty ping response body")
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

	if answer.IsResponseTo() == fields.Ping {
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
