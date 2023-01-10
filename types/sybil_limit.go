package types

import "fmt"

// -----------------------------------------------------------------------------
// ForwardRouteMessage

// NewEmpty implements types.Message.
func (m RouteMessage) NewEmpty() Message {
	return &RouteMessage{}
}

// Name implements types.Message.
func (RouteMessage) Name() string {
	return "route"
}

// String implements types.Message.
func (m RouteMessage) String() string {
	return fmt.Sprintf("RouteMessage{length: %d, id: %d, budget: %d, key: %s, mac: %s>", m.Length, m.ID, m.Budget, m.Data, m.MAC)
}

// HTML implements types.Message.
func (m RouteMessage) HTML() string {
	return m.String()
}

// SuspectRouteProtocolDone

// NewEmpty implements types.Message.
func (m SuspectRouteProtocolDone) NewEmpty() Message {
	return &RouteMessage{}
}

// Name implements types.Message.
func (SuspectRouteProtocolDone) Name() string {
	return "route"
}

// String implements types.Message.
func (m SuspectRouteProtocolDone) String() string {
	return fmt.Sprintf("SuspectRouteProtocolDone{tails: %v}", m.Tails)
}

// HTML implements types.Message.
func (m SuspectRouteProtocolDone) HTML() string {
	return m.String()
}

// VerifierRegistrationQuery

// NewEmpty implements types.Message.
func (m VerifierRegistrationQuery) NewEmpty() Message {
	return &RouteMessage{}
}

// Name implements types.Message.
func (VerifierRegistrationQuery) Name() string {
	return "route"
}

// String implements types.Message.
func (m VerifierRegistrationQuery) String() string {
	return fmt.Sprintf("VerifierRegistrationQuery{Suspect: %s, Tail: %v}", m.Suspect, m.Tail)
}

// HTML implements types.Message.
func (m VerifierRegistrationQuery) HTML() string {
	return m.String()
}

// VerifierRegistrationAnswer

// NewEmpty implements types.Message.
func (m VerifierRegistrationAnswer) NewEmpty() Message {
	return &RouteMessage{}
}

// Name implements types.Message.
func (VerifierRegistrationAnswer) Name() string {
	return "route"
}

// String implements types.Message.
func (m VerifierRegistrationAnswer) String() string {
	return fmt.Sprintf("VerifierRegistrationAnswer{Suspect: %s, Tail: %v, IsRegistered: %v}", m.Suspect, m.Tail, m.IsRegistered)
}

// HTML implements types.Message.
func (m VerifierRegistrationAnswer) HTML() string {
	return m.String()
}
