package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/cloud"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
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

type awsOriginalMessage struct {
	sqsService    *sqs.SQS
	queueUrl      *string
	receiptHandle *string
}

// Ack deletes the message from the queue after successful processing.
func (a awsOriginalMessage) Ack(ctx context.Context) error {
	_, err := a.sqsService.DeleteMessageWithContext(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      a.queueUrl,
		ReceiptHandle: a.receiptHandle,
	})

	return err
}

// Nack leaves the message in the queue. With requeue it becomes visible again
// immediately; otherwise it reappears after the visibility timeout and the
// queue's redrive policy routes it to the DLQ once maxReceiveCount is exceeded.
func (a awsOriginalMessage) Nack(ctx context.Context, requeue bool, _ error) error {
	if !requeue {
		return nil
	}

	_, err := a.sqsService.ChangeMessageVisibilityWithContext(ctx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          a.queueUrl,
		ReceiptHandle:     a.receiptHandle,
		VisibilityTimeout: aws.Int64(0),
	})

	return err
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
		Message: aws.String(msg.String()),
		TopicArn: aws.String(fmt.Sprintf("arn:%s:sns:%s:%s:%s",
			cloud.GetAwsARN().Partition,
			*cloud.GetAwsSession().Config.Region,
			cloud.GetAwsARN().AccountID,
			p.topic,
		)),
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
				continue
			}

			if len(msgs.Messages) > 0 {
				m.handleMessage(ctx, c, queueUrl, msgs.Messages[0], ch)
			}
		}
	}()

	return ch, nil
}

// handleMessage unwraps the SNS notification and delivers the message to the consumer
// channel with a real ack/nack bound to the SQS receipt handle. Malformed messages are
// skipped without deletion so they redeliver after the visibility timeout and the
// queue's redrive policy can route them to the DLQ.
func (m *awsMessaging) handleMessage(ctx context.Context, c *consumer, queueUrl *sqs.GetQueueUrlOutput, msg *sqs.Message, ch chan *ProviderMessage) {
	var n sqsNotification
	if err := json.Unmarshal([]byte(*msg.Body), &n); err != nil {
		logging.Error(ctx).Err(err).Msgf(couldNotReadMsgBody, *msg.MessageId, c.queue)
		return
	}

	var pm ProviderMessage
	if err := json.Unmarshal([]byte(n.Message), &pm); err != nil {
		logging.Error(ctx).Err(err).Msgf(couldNotReadMsgBody, *msg.MessageId, c.queue)
		return
	}

	pm.addOriginBrokerNotification(awsOriginalMessage{
		sqsService:    m.sqsService,
		queueUrl:      queueUrl.QueueUrl,
		receiptHandle: msg.ReceiptHandle,
	})
	ch <- &pm
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
