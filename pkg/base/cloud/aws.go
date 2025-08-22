package cloud

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/google/uuid"
)

func newAwsSession() *session.Session {
	if config.IsCloudEnvironment() {
		return session.Must(session.NewSession(&aws.Config{
			S3ForcePathStyle: aws.Bool(true),
		}))
	}

	return session.Must(session.NewSession(&aws.Config{
		Region:           aws.String(config.CLOUD_REGION),
		Endpoint:         aws.String(config.CLOUD_HOST),
		DisableSSL:       aws.Bool(config.CLOUD_DISABLE_SSL),
		Credentials:      credentials.NewStaticCredentials(uuid.NewString(), config.CLOUD_SECRET, config.CLOUD_TOKEN),
		S3ForcePathStyle: aws.Bool(true),
	}))
}

func getAwsARN() *arn.ARN {
	if config.IsLocalEnvironment() {
		parsedArn, _ := arn.Parse("arn:aws:iam::000000000000:role/app-name")
		return &parsedArn
	}

	if config.CLOUD_AWS_ROLE_ARN == "" {
		logging.Warn(context.Background()).Msg("AWS_ROLE_ARN not defined")
	}

	parsedArn, err := arn.Parse(config.CLOUD_AWS_ROLE_ARN)
	if err != nil {
		logging.Error(context.Background()).Err(err).Msg("invalid AWS_ROLE_ARN")
	}

	return &parsedArn
}
