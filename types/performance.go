package types

import "fmt"

// -----------------------------------------------------------------------------
// BinaryPublicationMessage

// NewEmpty implements types.Message.
func (p BinaryPublicationMessage) NewEmpty() Message {
	return &BinaryPublicationMessage{}
}

// Name implements types.Message.
func (p BinaryPublicationMessage) Name() string {
	return "binarypublication"
}

// String implements types.Message.
func (p BinaryPublicationMessage) String() string {
	return fmt.Sprintf("binarypublication{publication=%v true=%v}", p.Message.String(), p.TruthValue)
}

// HTML implements types.Message.
func (p BinaryPublicationMessage) HTML() string {
	return fmt.Sprintf("binarypublication (id=%v, true=%v)", p.Message.ID, p.TruthValue)
}
