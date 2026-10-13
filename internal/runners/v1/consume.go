package runner_v1

import (
	"bufio"
	"context"
	"fmt"

	"github.com/dejitarudemon/ignicula-go/internal/runners"
	table_v1 "github.com/dejitarudemon/ignicula-go/internal/table/v1"
	"github.com/dejitarudemon/ignicula-wire/v1/body"
	"github.com/dejitarudemon/ignicula-wire/v1/body/bodies"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/frame"
)

// Consume reads one v1 frame from reader and either completes a pending
// Handshake (while the version is not yet granted), replies to an unsolicited
// server Ping, or delivers an answer for a registered request.
//
// A decode or validation failure returns [runners.CloseConnection]. An unknown
// non-Ping request returns [runners.LogAndIgnore] wrapping
// [runners.ProtocolViolationByServer]. Failures that belong to a registered
// request are sent on [table_v1.RequestV1.ToCaller], the channel is closed,
// the row is terminated, and Consume returns [runners.LogAndIgnore]. A
// completed answer closes the caller channel; a partial answer leaves it open.
// A protocol [bodies.ErrorAnswer] is a valid reply to any registered command
// and is delivered as a normal response body.
//
// The registering caller must not close [table_v1.RequestV1.ToCaller]; Consume
// (or another consumer) closes it when the request is finished or failed.
func (r *Runner) Consume(ctx context.Context, reader *bufio.Reader) error {
	frm, err := r.decoder.DecodeFrame(reader)
	if err != nil {
		return runners.NewCloseConnection(err)
	}

	if err := frm.IsValid(); err != nil {
		return runners.NewCloseConnection(err)
	}

	if !r.table.IsGranted() {
		return r.consumeHandshake(frm)
	}

	row, ok := r.table.Find(frm.RequestID)
	if !ok {
		return r.consumeUnsolicited(ctx, frm)
	}

	return r.consumeAnswer(frm, row)
}

// consumeUnsolicited handles a frame whose RequestID is not in the table.
// Only a server Ping is accepted; any other command closes the connection.
func (r *Runner) consumeUnsolicited(ctx context.Context, frm frame.Frame) error {
	if frm.Body.Command() != fields.Ping {
		return runners.NewLogAndIgnore(
			runners.NewProtocolViolationByServer(
				fmt.Sprintf("server sent %v request", frm.Body.Command()),
			),
		)
	}

	return r.replyServerPing(ctx, frm.RequestID)
}

// replyServerPing builds a Ping answer, writes it to the outgoing requests
// channel, and terminates id afterward (including on failure paths via defer).
func (r *Runner) replyServerPing(ctx context.Context, id fields.RequestID) error {
	defer r.table.Terminate(id)

	response, err := r.builder.NewPingAnswer(id)
	if err != nil {
		return runners.NewCloseConnection(runners.NewInternalError(err))
	}

	raw, err := r.encodeFrame(response, nil)
	if err != nil {
		return runners.NewCloseConnection(runners.NewInternalError(err))
	}

	select {
	case <-ctx.Done():
		return runners.NewCloseConnection(ctx.Err())
	case <-r.close:
		return runners.NewCloseConnection(runners.ErrRunnerClosed)
	case r.requests <- raw:
		return nil
	}
}

// consumeAnswer delivers one answer frame for a registered request row.
func (r *Runner) consumeAnswer(frm frame.Frame, row table_v1.RequestV1) error {
	answer, ok := frm.Body.(body.Answer)
	if !ok {
		return r.failRequest(row, frm, runners.NewProtocolViolationByServer(
			fmt.Sprintf("server sends not an answer: %v", frm.Body.Command()),
		))
	}

	// ErrorAnswer.IsResponseTo is a stub ([fields.Answer]); it is a valid
	// failure reply to any registered request command.
	if _, isProtoErr := answer.(bodies.ErrorAnswer); !isProtoErr && row.Command != answer.IsResponseTo() {
		return r.failRequest(row, frm, runners.NewProtocolViolationByServer(
			fmt.Sprintf("server sends not an answer to expected: got %v, expected %v", answer.IsResponseTo(), row.Command),
		))
	}

	got, err := answerCount(answer)
	if err != nil {
		return r.failRequest(row, frm, runners.NewInternalError(err))
	}

	waitNext, err := r.table.RegisterResponse(frm.RequestID, uint32(got))
	if err != nil {
		return r.failRequest(row, frm, runners.NewInternalError(err))
	}

	return r.deliverToCaller(row, frm, !waitNext)
}

// answerCount returns how many nested answers this payload counts as.
// Non-batch answers count as one. A batch answer counts as len(BatchAnswer).
func answerCount(answer body.Answer) (int, error) {
	if answer.IsResponseTo() != fields.Batch {
		return 1, nil
	}

	banswer, ok := answer.(bodies.BatchAnswer)
	if !ok {
		return 0, runners.ErrFailedCastBatchAnswer
	}

	return len(banswer), nil
}

// failRequest sends err to the caller, closes the channel, terminates the row,
// and returns [runners.LogAndIgnore]. If the runner is closed, it closes the
// caller channel and returns [runners.CloseConnection] with [runners.ErrRunnerClosed].
func (r *Runner) failRequest(row table_v1.RequestV1, frm frame.Frame, err error) error {
	select {
	case <-r.close:
		close(row.ToCaller)
		r.table.Terminate(frm.RequestID)
		return runners.NewCloseConnection(runners.ErrRunnerClosed)
	case row.ToCaller <- table_v1.ResponseV1{Response: frm, Err: err}:
		close(row.ToCaller)
		r.table.Terminate(frm.RequestID)
		return runners.NewLogAndIgnore(err)
	}
}

// deliverToCaller sends frm to the caller. When done is true it closes the
// channel and terminates the row; otherwise the request stays registered.
// The registering caller must not close ToCaller itself. If the runner is
// closed, the caller channel is closed and the error is [runners.CloseConnection]
// wrapping [runners.ErrRunnerClosed].
func (r *Runner) deliverToCaller(row table_v1.RequestV1, frm frame.Frame, done bool) error {
	select {
	case <-r.close:
		close(row.ToCaller)
		r.table.Terminate(frm.RequestID)
		return runners.NewCloseConnection(runners.ErrRunnerClosed)
	case row.ToCaller <- table_v1.ResponseV1{Response: frm}:
	}

	if done {
		close(row.ToCaller)
		r.table.Terminate(frm.RequestID)
	}

	return nil
}

// consumeHandshake handles frames received before the version is granted.
// A successful [bodies.HandshakeAnswer] stores compressions/versions in info,
// grants and activates the table, clears the activating flag, and closes
// [Runner.ready] so [Runner.WaitReady] unblocks. A protocol [bodies.ErrorAnswer]
// returns [runners.CloseConnection] wrapping [runners.ServerError]. Any other
// frame while activating is a protocol violation that closes the connection.
// Frames while not activating are logged and ignored.
func (r *Runner) consumeHandshake(frm frame.Frame) error {
	if !r.activating.Load() {
		return runners.NewLogAndIgnore(
			runners.NewProtocolViolationByServer("handshake frame while not activating"),
		)
	}

	if frm.RequestID != table_v1.PROHIBITED_REQUEST_ID {
		r.activating.Store(false)
		return runners.NewCloseConnection(
			runners.NewProtocolViolationByServer(
				fmt.Sprintf("handshake answer request id = %d, want 0", frm.RequestID),
			),
		)
	}

	answer, ok := frm.Body.(body.Answer)
	if !ok {
		r.activating.Store(false)
		return runners.NewCloseConnection(
			runners.NewProtocolViolationByServer(
				fmt.Sprintf("server sends not an answer: %v", frm.Body.Command()),
			),
		)
	}

	if answer.IsResponseTo() == fields.Handshake {
		hs, ok := answer.(bodies.HandshakeAnswer)
		if !ok {
			r.activating.Store(false)
			return runners.NewCloseConnection(
				runners.NewProtocolViolationByServer(
					fmt.Sprintf("server sent non-handshake answer %T", answer),
				),
			)
		}

		r.info.SetSupportedCompressions(hs.Compressions)
		r.info.SetSupportedVersions(hs.Versions)
		r.table.Grant(true)
		r.table.Activate(true)
		r.activating.Store(false)
		close(r.ready)
		return nil
	}

	protoerr, ok := answer.(bodies.ErrorAnswer)
	if !ok || protoerr.Err == nil {
		r.activating.Store(false)
		return runners.NewCloseConnection(
			runners.NewProtocolViolationByServer(
				fmt.Sprintf("server sent unexpected answer to %v", answer.IsResponseTo()),
			),
		)
	}

	r.activating.Store(false)
	return runners.NewCloseConnection(runners.NewServerError(protoerr.Err))
}
