package impl

import (
	"context"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/rs/zerolog/log"
	"golang.org/x/xerrors"
	"google.golang.org/api/option"
)

func newAuthentication() *authentication {
	ctx := context.Background()
	opt := option.WithCredentialsFile("path/to/serviceAccountKey.json")
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Err(err).Msg("failed to create app")
	}
	client, err := app.Auth(ctx)
	if err != nil {
		log.Err(err).Msg("failed to create client")
	}
	return &authentication{
		ctx:    ctx,
		client: client,
	}
}

type authentication struct {
	ctx    context.Context
	client *auth.Client
}

func (authentication *authentication) SignUp(email, name string) (*auth.UserRecord, error) {
	// Create a new user with the email.
	params := (&auth.UserToCreate{}).
		Email(email).
		DisplayName(name)
	user, err := authentication.client.CreateUser(authentication.ctx, params)
	if err != nil {
		return nil, xerrors.Errorf("error creating user: %v", err)
	}

	return user, nil
}

func (authentication *authentication) SignIn(email, uid string) (*auth.UserRecord, error) {
	// Get user by email.
	user, err := authentication.client.GetUserByEmail(context.Background(), email)
	if err != nil {
		return nil, xerrors.Errorf("error getting user with this email address: %v", err)
	}

	if user.UID != uid {
		return nil, xerrors.Errorf("invalid uid for this email address")
	}

	return user, nil
}
