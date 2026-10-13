// Package table_v1 tracks in-flight protocol v1 request IDs on one connection.
//
// [RequestsV1] records client requests that still expect answers from the
// server. [RequestsV1.RegisterRequest] allocates a free non-zero ID, stores the
// command and a non-nil [ResponseV1Chan] for the caller, and returns that ID.
// The registering caller must not close that channel; the consumer closes it
// when the request ends. [RequestsV1.RegisterResponse] counts answers until the
// expected total is reached; the row stays until [RequestsV1.Terminate].
//
// A Handshake request is not registered here. Until an official Handshake
// succeeds, the table is not considered ready to register requests;
// [RequestsV1.Grant] / [RequestsV1.Activate] record that readiness afterward.
// Register and response methods require both flags and otherwise return
// [ErrVersionNeitherGrantedNorActivated]. Request ID 0 is reserved for
// Handshake on the wire and is never allocated by [RequestsV1.RegisterRequest].
//
// Methods are safe for concurrent use. A RequestsV1 contains a mutex and must
// not be copied; share the pointer returned by [NewRequestsV1].
package table_v1
