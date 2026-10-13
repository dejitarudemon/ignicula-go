package configs

type AuthConfig struct {
	Login string
	Hash  [32]byte
}

// DefaultAuthConfig returns an empty auth config. Callers must set Login and
// Hash before activating a connection.
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{}
}
