package runner_v1

import (
	"time"

	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
	"github.com/dejitarudemon/ignicula-wire/v1/compressor"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
	"github.com/dejitarudemon/ignicula-wire/v1/frame"
)

// encodeFrame encodes frm using a pooled [buffer.Slice] and returns a copy of
// the bytes. The buffer is returned to the pool before encodeFrame returns.
func (r *Runner) encodeFrame(frm frame.Frame, c compressor.Compressor) ([]byte, error) {
	buf := r.bufPool.Get().(*buffer.Slice)
	buf.Preallocate(frm.Size())
	buf.Clean()
	defer r.bufPool.Put(buf)

	if err := frm.Encode(buf, c); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func toCompressionsMap(compressors []compressor.Compressor) map[fields.Compression]compressor.Compressor {
	result := make(map[fields.Compression]compressor.Compressor)

	for _, compressor := range compressors {
		if _, ok := result[compressor.Code()]; !ok && compressor != nil && compressor.Code() != fields.None {
			result[compressor.Code()] = compressor
		}
	}

	return result
}

func toCompressionsList(compressors map[fields.Compression]compressor.Compressor) []fields.Compression {
	result := make([]fields.Compression, 0, len(compressors))

	for code := range compressors {
		result = append(result, code)
	}

	return result
}

func backoff(backoff time.Duration) {
	if backoff != 0 {
		time.Sleep(backoff)
	}
}
