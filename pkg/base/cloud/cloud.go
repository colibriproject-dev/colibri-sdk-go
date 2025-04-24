package cloud

import (
	"context"

	firebase "firebase.google.com/go"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/logging"
)

// Cloud is a struct that contains the cloud settings.
type Cloud struct {
	aws      *session.Session
	firebase *firebase.App
}

var instance *Cloud

// Initialize loads the cloud settings according to the configured environment.
func Initialize() {
	instance = &Cloud{}

	switch config.CLOUD {
	case config.CLOUD_AWS:
		instance.aws = newAwsSession()
	case config.CLOUD_FIREBASE:
		instance.firebase = newFirebaseSession()
	case config.CLOUD_GCP:
		logging.Info(context.Background()).Msg("Initializing GCP")
	}

	logging.Info(context.Background()).Msg("Cloud provider connected")
}

// GetAwsSession returns the AWS session.
func GetAwsSession() *session.Session {
	return instance.aws
}

// GetFirebaseSession returns the Firebase session.
func GetFirebaseSession() *firebase.App {
	return instance.firebase
}
