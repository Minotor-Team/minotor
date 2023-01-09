package performance

import (
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/types"
)

// A binary publication is a publication that has a truth value.
// It can either be a true publication (meaning that its content is
// correct) or a fake publication (meaning that its content are
// fake news)
type BinaryPublication struct {
	peer.Publication
	TruthValue bool
}

type Factory func(conf peer.Configuration) *PublicationReactorNode

// Return a new publication whose content is fake.
func NewFakePublication(p peer.Publication) BinaryPublication {
	return BinaryPublication{
		Publication: p,
		TruthValue:  false,
	}
}

// Return a new binary publication whose content is true.
func NewTruePublication(p peer.Publication) BinaryPublication {
	return BinaryPublication{
		Publication: p,
		TruthValue:  true,
	}
}

// Return true iff the content of the publication is fake.
func (p *BinaryPublication) isFake() bool {
	return !p.TruthValue
}

func publicationFromMessage(msg types.PublicationMessage) peer.Publication {
	return peer.Publication{
		ID:      msg.ID,
		Author:  msg.Author,
		Content: msg.Content,
	}
}

func binaryPublicationFromMessage(msg types.BinaryPublicationMessage) BinaryPublication {
	return BinaryPublication{
		Publication: publicationFromMessage(*msg.Message),
		TruthValue:  msg.TruthValue,
	}
}

func publicationToMessage(p peer.Publication) types.PublicationMessage {
	return types.PublicationMessage{
		ID:      p.ID,
		Author:  p.Author,
		Content: p.Content,
	}
}

func binaryPublicationToMessage(p BinaryPublication) types.BinaryPublicationMessage {
	pubMsg := publicationToMessage(p.Publication)
	return types.BinaryPublicationMessage{
		Message:    &pubMsg,
		TruthValue: p.TruthValue,
	}
}
