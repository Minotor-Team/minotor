package types

import "fmt"

// -----------------------------------------------------------------------------
// PublicatioMessage

// NewEmpty implements types.Message.
func (s PublicationMessage) NewEmpty() Message {
	return &PublicationMessage{}
}

// Name implements types.Message.
func (s PublicationMessage) Name() string {
	return "publicationmessage"
}

// String implements types.Message.
func (s PublicationMessage) String() string {
	return fmt.Sprintf("publication{author=%v, content=%v}", s.Author, s.Content)
}

// HTML implements types.Message.
func (s PublicationMessage) HTML() string {
	return s.String()
}

// -----------------------------------------------------------------------------
// FollowRequest

// NewEmpty implements types.Message.
func (s FollowRequest) NewEmpty() Message {
	return &FollowRequest{}
}

// Name implements types.Message.
func (s FollowRequest) Name() string {
	return "followrequest"
}

// String implements types.Message.
func (s FollowRequest) String() string {
	return "follorequest"
}

// HTML implements types.Message.
func (s FollowRequest) HTML() string {
	return s.String()
}

// -----------------------------------------------------------------------------
// FollowResponse

// NewEmpty implements types.Message.
func (s FollowResponse) NewEmpty() Message {
	return &FollowResponse{}
}

// Name implements types.Message.
func (s FollowResponse) Name() string {
	return "followresponse"
}

// String implements types.Message.
func (s FollowResponse) String() string {
	return "folloresponse"
}

// HTML implements types.Message.
func (s FollowResponse) HTML() string {
	return s.String()
}
