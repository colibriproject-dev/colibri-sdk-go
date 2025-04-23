package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/cloud"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/logging"
)

type sqsNotification struct {
	Type             string `json:"Type"`
	TopicArn         string `json:"TopicArn"`
	MessageId        string `json:"MessageId"`
	Message          string `json:"Message"`
	Timestamp        string `json:"Timestamp"`
	SignatureVersion string `json:"SignatureVersion"`
	Signature        string `json:"Signature"`
	SigningCertURL   string `json:"SigningCertURL"`
	UnsubscribeURL   string `json:"UnsubscribeURL"`
}

type awsMessaging struct {
	snsService *sns.SNS
	sqsService *sqs.SQS
}

func newAwsMessaging() *awsMessaging {
	var m awsMessaging
	m.snsService = sns.New(cloud.GetAwsSession())
	m.sqsService = sqs.New(cloud.GetAwsSession())

	if _, err := m.snsService.ListTopics(nil); err != nil {
		logging.Fatal(context.Background()).Err(err).Msg(connectionError)
	}

	return &m
}

func (m *awsMessaging) producer(ctx context.Context, p *Producer, msg *ProviderMessage) error {
	_, err := m.snsService.PublishWithContext(ctx, &sns.PublishInput{
		Message:  aws.String(msg.String()),
		TopicArn: aws.String(fmt.Sprintf("arn:aws:sns:us-east-1:000000000000:%s", p.topic)),
	})

	return err
}

func (m *awsMessaging) consumer(ctx context.Context, c *consumer) (chan *ProviderMessage, error) {
	ch := make(chan *ProviderMessage, 1)
	queueUrl := m.getQueueUrl(ctx, c.queue)

	go func() {
		for {
			if c.isCanceled() {
				c.Done()
				return
			}

			msgs, err := m.readMessages(ctx, queueUrl)
			if err != nil {
				logging.Error(ctx).Err(err).Msgf(couldNotReceiveMsg, c.queue)
			}

			if len(msgs.Messages) > 0 {
				msg := msgs.Messages[0]

				var n sqsNotification
				if err = json.Unmarshal([]byte(*msg.Body), &n); err != nil {
					logging.Error(ctx).Err(err).Msgf(couldNotReadMsgBody, *msg.MessageId, c.queue)
				}

				var pm ProviderMessage
				if err = json.Unmarshal([]byte(n.Message), &pm); err != nil {
					logging.Error(ctx).Err(err).Msgf(couldNotReadMsgBody, *msg.MessageId, c.queue)
					return
				}

				pm.addOriginBrokerNotification(&n)
				ch <- &pm
				m.removeMessageFromQueue(ctx, queueUrl, msg)
			}
		}
	}()

	return ch, nil
}

func (m *awsMessaging) readMessages(ctx context.Context, queueResult *sqs.GetQueueUrlOutput) (*sqs.ReceiveMessageOutput, error) {
	var msgs, err = m.sqsService.ReceiveMessageWithContext(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:              queueResult.QueueUrl,
		MaxNumberOfMessages:   aws.Int64(1),
		WaitTimeSeconds:       aws.Int64(1),
		MessageAttributeNames: aws.StringSlice([]string{"All"}),
	})

	return msgs, err
}

func (m *awsMessaging) removeMessageFromQueue(ctx context.Context, queueResult *sqs.GetQueueUrlOutput, msg *sqs.Message) {
	if _, err := m.sqsService.DeleteMessageWithContext(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      queueResult.QueueUrl,
		ReceiptHandle: msg.ReceiptHandle,
	}); err != nil {
		logging.Error(ctx).Err(err).Msgf(couldNotDeleteMsg, *msg.MessageId, *queueResult.QueueUrl)
	}
}

func (m *awsMessaging) getQueueUrl(ctx context.Context, queue string) *sqs.GetQueueUrlOutput {
	queueResult, err := m.sqsService.GetQueueUrlWithContext(ctx, &sqs.GetQueueUrlInput{QueueName: aws.String(queue)})
	if err != nil {
		logging.Fatal(ctx).Err(err).Msgf(couldNotConnectQueue, queue)
	}

	if queueResult.QueueUrl == nil {
		logging.Fatal(ctx).Msgf(queueNotFound, queue)
	}

	return queueResult
}
