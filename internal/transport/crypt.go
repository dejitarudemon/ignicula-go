package transport

import "golang.org/x/crypto/argon2"

// Argon2id parameters for the Handshake hash. Parallelism is fixed so the same
// password always yields the same 32-byte digest on every machine.
const (
	argonMemory      = 64 * 1024
	argonIterations  = 1
	argonParallelism = 4
	argonKeyLen      = 32
)

// hash returns a 32-byte Argon2id digest of password (empty salt).
func hash(password string) [32]byte {
	key := argon2.IDKey(
		[]byte(password),
		nil,
		argonIterations,
		argonMemory,
		argonParallelism,
		argonKeyLen,
	)
	var out [32]byte
	copy(out[:], key)
	return out
}
