package cloud

import (
	"context"

	firebase "firebase.google.com/go"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

// Cloud is a struct that contains the cloud settings.
type Cloud struct {
	awsSession *session.Session
	awsARN     *arn.ARN

	firebase *firebase.App
}

var instance *Cloud

// Initialize loads the cloud settings according to the configured environment.
func Initialize() {
	instance = &Cloud{}

	switch config.CLOUD {
	case config.CLOUD_AWS:
		instance.awsSession = newAwsSession()
		instance.awsARN = getAwsARN()
	case config.CLOUD_FIREBASE:
		instance.firebase = newFirebaseSession()
	case config.CLOUD_GCP:
		logging.Info(context.Background()).Msg("Initializing GCP")
	}

	logging.Info(context.Background()).Msg("Cloud provider connected")
}

// GetAwsSession returns the AWS session.
func GetAwsSession() *session.Session {
	return instance.awsSession
}

// GetAwsARN returns the AWS ARN.
func GetAwsARN() *arn.ARN {
	return instance.awsARN
}

// GetFirebaseSession returns the Firebase session.
func GetFirebaseSession() *firebase.App {
	return instance.firebase
}
