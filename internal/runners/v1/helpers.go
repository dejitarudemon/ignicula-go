package runner_v1

import (
	"time"

	"github.com/dejitarudemon/ignicula-wire/v1/compressor"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

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

func calculateTimeout(base time.Duration, backoff time.Duration, multiplier int) time.Duration {
	return base + backoff*time.Duration(multiplier)
}
