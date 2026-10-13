package transport

import (
	"bufio"
	"fmt"
	"slices"
	"time"

	"github.com/dejitarudemon/ignicula-wire/v0/body/bodies"
	"github.com/dejitarudemon/ignicula-wire/v0/builder"
	"github.com/dejitarudemon/ignicula-wire/v0/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
)

const limitV0 = 263

func (c *Connection) hello(timeout time.Duration) error {
	frame, err := builder.NewFrameBuilder(limitV0).NewHello(c.versions)
	if err != nil {
		return fmt.Errorf("frame building error: %w", err)
	}

	buf := buffer.Slice{}
	buf.Preallocate(frame.Size())

	if err := frame.Encode(&buf); err != nil {
		return fmt.Errorf("frame encoding error: %w", err)
	}

	writer := bufio.NewWriter(c.inner)
	reader := bufio.NewReader(c.inner)

	if err := c.inner.SetDeadline(deadline(timeout)); err != nil {
		return fmt.Errorf("read timeout error: %w", err)
	}

	if _, err := writer.Write(buf.Raw()); err != nil {
		return fmt.Errorf("connection writing error: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("connection writing error: %w", err)
	}

	version, err := c.decoder.DecodePreamble(reader)
	if err != nil {
		return fmt.Errorf("response reading error: %w", err)
	}

	if version != fields.CurrentVersion {
		return fmt.Errorf("unexpected %v version for Hello", version)
	}

	response, err := c.decoder.DecodeFrame(reader)
	if err != nil {
		return fmt.Errorf("response reading error: %w", err)
	}

	if err := response.IsValid(); err != nil {
		return fmt.Errorf("response validation error: %w", err)
	}

	body, ok := response.Body.(bodies.Hello)
	if !ok {
		return fmt.Errorf("unexpected hello body %T", response.Body)
	}

	for _, version := range c.versions {
		if !slices.Contains(body.Versions, version) {
			return fmt.Errorf("server doesn't support %v version", version)
		}
	}

	if err := c.inner.SetDeadline(time.Time{}); err != nil {
		return fmt.Errorf("clear deadline error: %w", err)
	}

	return nil
}
