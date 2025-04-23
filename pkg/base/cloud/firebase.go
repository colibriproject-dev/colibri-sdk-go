package cloud

import (
	"context"

	firebase "firebase.google.com/go"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/logging"
	"google.golang.org/api/option"
)

func newFirebaseSession() *firebase.App {
	opt := option.WithCredentialsJSON([]byte(config.CLOUD_SECRET))
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		logging.Fatal(context.Background()).Msg("Firebase initialization error")
	}

	return app
}
