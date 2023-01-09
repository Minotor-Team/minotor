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
