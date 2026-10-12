package table

import (
	"github.com/dejitarudemon/ignicula-go/internal/transport/table/versions"
	"github.com/dejitarudemon/ignicula-wire/v0/fields"
)

// Version1 is the only protocol version [RequestsTable] currently supports.
const Version1 fields.Version = 1

// RequestsTable holds per-version request state for one connection.
//
// Only [Version1] is available for now: [RequestsTable.Support] reports true
// solely for that value, and [RequestsTable.V1] returns the v1 set. Later
// versions may add more accessors without changing this layout.
type RequestsTable struct {
	v1 *versions.RequestsV1
}

// NewRequestsTable returns a table with an empty v1 request set.
func NewRequestsTable() *RequestsTable {
	return &RequestsTable{
		v1: versions.NewRequestsV1(),
	}
}

// Close terminates every registered v1 request and clears grant and active
// on the v1 set. After Close, register and response methods on [RequestsTable.V1]
// return [versions.ErrVersionNeitherGrantedNorActivated] until the version is
// granted and activated again.
func (r *RequestsTable) Close() {
	r.v1.TerminateAll()
	r.v1.Activate(false)
	r.v1.Grant(false)
}

// Support reports whether version is available in this table.
// Today that is only [Version1].
func (r *RequestsTable) Support(version fields.Version) bool {
	return version == Version1
}

// V1 returns the protocol v1 request set.
// The pointer is shared; callers must not replace it.
func (r *RequestsTable) V1() *versions.RequestsV1 {
	return r.v1
}
