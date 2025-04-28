package cloud

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/google/uuid"
)

func newAwsSession() *session.Session {
	awsConfig := &aws.Config{
		Region:           aws.String(config.CLOUD_REGION),
		Endpoint:         aws.String(config.CLOUD_HOST),
		DisableSSL:       aws.Bool(config.CLOUD_DISABLE_SSL),
		Credentials:      credentials.NewStaticCredentials(uuid.NewString(), config.CLOUD_SECRET, config.CLOUD_TOKEN),
		S3ForcePathStyle: aws.Bool(true),
	}

	return session.Must(session.NewSession(awsConfig))
}
