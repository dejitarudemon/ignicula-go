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

// Read sends a protocol read for key and waits for a Read answer or a protocol
// error answer. Transient server errors ([protocolerrs.ErrorRequestsConflict],
// [protocolerrs.ErrorInternalError]) are retried up to the configured attempt
// limit. A successful answer is converted through the runner's v1 adapter.
func (r *Runner) Read(ctx context.Context, key []byte) (values.Value, error) {
	if r.activating.Load() {
		return nil, runners.NewInternalError(runners.ErrActivating)
	}

	for attempt := 0; attempt < r.timeouts.MaxAttempts; attempt++ {
		val, retry, err := r.readOnce(ctx, attempt, key)
		if err != nil {
			return nil, err
		}
		if !retry {
			return val, nil
		}
	}

	return nil, runners.NewInternalError(runners.ErrReadExhausted)
}

// readOnce registers, sends, and interprets one read attempt.
// retry is true when the server reported a transient error and another
// attempt is still allowed.
func (r *Runner) readOnce(parent context.Context, attempt int, key []byte) (values.Value, bool, error) {
	backoff(time.Duration(attempt) * r.timeouts.Backoff)

	ctx, cancel := context.WithTimeout(parent, r.timeouts.Read)
	defer cancel()

	id, ch, err := r.sendRead(ctx, key)
	if err != nil {
		return nil, false, err
	}

	response, err := r.await(ctx, ch)
	if err != nil {
		r.table.Terminate(id)
		return nil, false, err
	}

	val, transient, err := r.interpretReadResponse(response)
	if err != nil {
		return nil, false, err
	}
	if transient == nil {
		return val, false, nil
	}
	if attempt+1 < r.timeouts.MaxAttempts {
		return nil, true, nil
	}
	return nil, false, runners.NewServerError(transient)
}

// sendRead registers a Read request, encodes the frame, and writes it to the
// outgoing requests channel. On failure before a successful send it terminates
// the request id and returns [runners.InternalError].
func (r *Runner) sendRead(ctx context.Context, key []byte) (fields.RequestID, table_v1.ResponseV1Chan, error) {
	ch := make(table_v1.ResponseV1Chan, 1)

	id, err := r.table.RegisterRequest(fields.Read, ch, 1)
	if err != nil {
		return 0, nil, runners.NewInternalError(err)
	}

	frm, err := r.builder.NewRead(id, fields.Key(key))
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

// interpretReadResponse classifies a completed read response.
// On success the adapted value is returned with both errors nil (value may be
// nil when the server sent an empty ReadAnswer). A transient protocol error
// ([protocolerrs.ErrorRequestsConflict], [protocolerrs.ErrorInternalError])
// is returned as the middle error so the caller can retry or wrap it as
// [runners.ServerError]. Other failures are returned only as the last error.
func (r *Runner) interpretReadResponse(response table_v1.ResponseV1) (values.Value, error, error) {
	if err := response.Err; err != nil {
		return nil, nil, err
	}

	if response.Response.Body == nil {
		return nil, nil, runners.NewProtocolViolationByServer("empty read response body")
	}

	if response.Response.Body.Command() != fields.Answer {
		return nil, nil, runners.NewProtocolViolationByServer(
			fmt.Sprintf("server sent %v instead of Answer", response.Response.Body.Command()),
		)
	}

	answer, ok := response.Response.Body.(body.Answer)
	if !ok {
		return nil, nil, runners.NewProtocolViolationByServer(
			fmt.Sprintf("server sent non-answer body %T", response.Response.Body),
		)
	}

	if answer.IsResponseTo() == fields.Read {
		ra, ok := answer.(bodies.ReadAnswer)
		if !ok {
			return nil, nil, runners.NewProtocolViolationByServer(
				fmt.Sprintf("server sent non-read answer %T", answer),
			)
		}
		if ra.Value == nil {
			return nil, nil, nil
		}
		val, err := r.adapter.From(ra.Value)
		if err != nil {
			return nil, nil, runners.NewInternalError(err)
		}
		return val, nil, nil
	}

	protoerr, ok := answer.(bodies.ErrorAnswer)
	if !ok || protoerr.Err == nil {
		return nil, nil, runners.NewProtocolViolationByServer(
			fmt.Sprintf("server sent unexpected answer to %v", answer.IsResponseTo()),
		)
	}

	switch protoerr.Err.(type) {
	case protocolerrs.ErrorRequestsConflict, protocolerrs.ErrorInternalError:
		return nil, protoerr.Err, nil
	}

	return nil, nil, runners.NewServerError(protoerr.Err)
}
