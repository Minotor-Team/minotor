package impl

import (
	"context"
	"os"
	"strings"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/rs/zerolog/log"
	"golang.org/x/xerrors"
	"google.golang.org/api/option"
)

func newAuthentication() *authentication {
	ctx := context.Background()
	f, _ := os.Getwd()
	f = strings.Split(f, "minotor")[0]
	f += "minotor/credentials/key.json"
	opt := option.WithCredentialsFile(f)
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

func (authentication *authentication) SignIn(email, uid string) error {
	// Get user by email.
	user, err := authentication.client.GetUserByEmail(context.Background(), email)
	if err != nil {
		return xerrors.Errorf("error getting user with this email address: %v", err)
	}

	if user.UID != uid {
		return xerrors.Errorf("invalid uid for this email address")
	}

	return nil
}
