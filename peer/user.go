package peer

import "go.dedis.ch/cs438/datastructures"

// Define the capabilities a user of the MinoTor social network
type User interface {
	// Publish a publication to the network.
	// To be embedded in a rumor.
	Publish(content string) error

	// Like a given publication
	Like(p Publication) error

	// Dislike a given publication
	Dislike(p Publication) error

	// Follow the user with specified username (ie. address)
	Follow(user string) error

	// Fetch the reputation of the user (ideally stored in dedicated blockchain)
	GetReputation() (int, error)

	// Fetch the list of users followed by this user
	GetFollowed() datastructures.Set[string]
}

// Represents a publication (ie. a message posted publicly on the social network)
type Publication struct {
	// Unique ID, use xid.New().String() to generate
	ID string
	// Username of the author
	Author string
	// Content of the publication
	Content string
}
