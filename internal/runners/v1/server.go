package runner_v1

import (
	"slices"
	"sync"

	"github.com/dejitarudemon/ignicula-wire/v1/fields"
)

// ServerInfo holds server capabilities learned while the runner works.
//
// It is not a construction config: the runner fills body/batch limits and the
// supported compressions and versions from server answers (for example after
// Handshake). Methods are safe for concurrent use. A ServerInfo contains a
// mutex and must not be copied; share the pointer from [NewServerInfo].
type ServerInfo struct {
	bodyLimit             fields.BodyLimit
	batchLimit            fields.BatchLimit
	supportedCompressions []fields.Compression
	supportedVersions     []fields.Version

	mx sync.RWMutex
}

// NewServerInfo returns an empty ServerInfo with zero limits and empty support lists.
func NewServerInfo() *ServerInfo {
	return &ServerInfo{}
}

// BodyLimit returns the last body limit recorded for the server.
func (s *ServerInfo) BodyLimit() fields.BodyLimit {
	s.mx.RLock()
	defer s.mx.RUnlock()

	return s.bodyLimit
}

// BatchLimit returns the last batch limit recorded for the server.
func (s *ServerInfo) BatchLimit() fields.BatchLimit {
	s.mx.RLock()
	defer s.mx.RUnlock()

	return s.batchLimit
}

// SupportedCompressions returns a snapshot of compressions the server reported.
// Changing the slice does not change ServerInfo.
func (s *ServerInfo) SupportedCompressions() []fields.Compression {
	s.mx.RLock()
	defer s.mx.RUnlock()

	return slices.Clone(s.supportedCompressions)
}

// SupportedVersions returns a snapshot of versions the server reported.
// Changing the slice does not change ServerInfo.
func (s *ServerInfo) SupportedVersions() []fields.Version {
	s.mx.RLock()
	defer s.mx.RUnlock()

	return slices.Clone(s.supportedVersions)
}

// SetBodyLimit records the server body limit.
func (s *ServerInfo) SetBodyLimit(limit fields.BodyLimit) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.bodyLimit = limit
}

// SetBatchLimit records the server batch limit.
func (s *ServerInfo) SetBatchLimit(limit fields.BatchLimit) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.batchLimit = limit
}

// SetSupportedCompressions replaces the supported compression list with a copy
// of compressions. A nil or empty list means no compressions are recorded.
func (s *ServerInfo) SetSupportedCompressions(compressions []fields.Compression) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.supportedCompressions = slices.Clone(compressions)
}

// SetSupportedVersions replaces the supported version list with a copy of
// versions. A nil or empty list means no versions are recorded.
func (s *ServerInfo) SetSupportedVersions(versions []fields.Version) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.supportedVersions = slices.Clone(versions)
}
