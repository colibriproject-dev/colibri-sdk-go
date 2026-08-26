package cloud

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/google/uuid"
)

func newAwsConfig() *aws.Config {
	ctx := context.Background()

	if config.IsCloudEnvironment() {
		return loadAwsConfig(ctx)
	}

	// outside a cloud environment the endpoint points at an emulator (LocalStack), and the
	// credentials are the placeholders it accepts. CLOUD_HOST already carries the scheme, so
	// there is nothing left for the former DisableSSL flag to decide
	return loadAwsConfig(ctx,
		awsconfig.WithRegion(config.CLOUD_REGION),
		awsconfig.WithBaseEndpoint(config.CLOUD_HOST),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			uuid.NewString(), config.CLOUD_SECRET, config.CLOUD_TOKEN,
		)),
	)
}

func loadAwsConfig(ctx context.Context, optFns ...func(*awsconfig.LoadOptions) error) *aws.Config {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		logging.Fatal(ctx).Err(err).Msg("could not load the AWS configuration")
	}

	return &cfg
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
