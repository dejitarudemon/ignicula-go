// Package table holds the per-connection request table for protocol versions
// used by the transport.
//
// [RequestsTable] currently supports only protocol version 1. Use
// [RequestsTable.Support] to check a version and [RequestsTable.V1] to reach
// the v1 request set. [RequestsTable.Close] clears every registered request
// and clears grant/active so the version must be granted and activated again.
package table
