// Package versions tracks in-flight protocol v1 request IDs on one connection.
//
// [RequestsV1] records outgoing requests that still expect answers and incoming
// requests that were accepted from the peer. [RequestsV1.RegisterOutcomingRequest]
// allocates a free non-zero ID and returns it. [RequestsV1.RegisterIncomingRequest]
// stores a peer-chosen non-zero ID. [RequestsV1.RegisterResponse] counts answers
// for an outgoing request until the expected total is reached; the row stays
// until [RequestsV1.Terminate].
//
// A Handshake request is not registered here. Until an official Handshake
// succeeds, the table is not considered ready to register requests;
// [RequestsV1.Grant] / [RequestsV1.Activate] record that readiness afterward.
// Register and response methods require both flags and otherwise return
// [ErrVersionNeitherGrantedNorActivated]. Request ID 0 is reserved for
// Handshake on the wire and is rejected by this package for both outgoing
// allocation and incoming registration.
//
// Methods are safe for concurrent use. A RequestsV1 contains a mutex and must
// not be copied; share the pointer returned by [NewRequestsV1].
package versions
