package peer

type IdentityVerifier interface {
	// Return all the users whose identity has been verified.
	// Incoming packets from unverified users should not be accepted.
	GetVerifiedUsers() []string
	// Returns the username of the user -- by default username == address
	GetName() string
}

type Identity struct {
	Username    string
	Email       string
	PhoneNumber string
}
