package cloud

import (
	"context"

	firebase "firebase.google.com/go/v4"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

// Cloud is a struct that contains the cloud settings.
type Cloud struct {
	awsConfig *aws.Config
	awsARN    *arn.ARN

	firebase *firebase.App
}

var instance *Cloud

// Initialize loads the cloud settings according to the configured environment.
func Initialize() {
	instance = &Cloud{}

	switch config.CLOUD {
	case config.CLOUD_AWS:
		instance.awsConfig = newAwsConfig()
		instance.awsARN = getAwsARN()
	case config.CLOUD_FIREBASE:
		instance.firebase = newFirebaseSession()
	case config.CLOUD_GCP:
		logging.Info(context.Background()).Msg("Initializing GCP")
	}

	logging.Info(context.Background()).Msg("Cloud provider connected")
}

// GetAwsConfig returns the AWS configuration used to build service clients.
func GetAwsConfig() aws.Config {
	return *instance.awsConfig
}

// GetAwsARN returns the AWS ARN.
func GetAwsARN() *arn.ARN {
	return instance.awsARN
}

// GetFirebaseSession returns the Firebase session.
func GetFirebaseSession() *firebase.App {
	return instance.firebase
}
