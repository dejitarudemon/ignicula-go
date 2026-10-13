package configs

// AuthConfig holds Handshake credentials.
//
// When UseAnonim is true, the client sends an empty login and a zero hash
// (anonymous connection). Otherwise Password is hashed with Argon2id into the
// 32-byte Handshake hash.
type AuthConfig struct {
	Login     string
	Password  string
	UseAnonim bool
}

// DefaultAuthConfig returns an empty auth config. Callers must set Login and
// Password (or UseAnonim) before activating a connection.
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{}
}
