package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/cloud"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

// releaseTimeout bounds the call that makes a message visible again. It is short on purpose:
// the release runs on the way out of a consumer, and a broker that stops responding must not
// hold that exit open.
const releaseTimeout = 5 * time.Second

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
	snsService *sns.Client
	sqsService *sqs.Client
}

type awsOriginalMessage struct {
	sqsService    *sqs.Client
	queueUrl      *string
	receiptHandle *string
}

// Ack deletes the message from the queue after successful processing.
func (a awsOriginalMessage) Ack(ctx context.Context) error {
	_, err := a.sqsService.DeleteMessage(ctx, &sqs.DeleteMessageInput{
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

	_, err := a.sqsService.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          a.queueUrl,
		ReceiptHandle:     a.receiptHandle,
		VisibilityTimeout: 0,
	})

	return err
}

func newAwsMessaging() *awsMessaging {
	var m awsMessaging
	m.snsService = sns.NewFromConfig(cloud.GetAwsConfig())
	m.sqsService = sqs.NewFromConfig(cloud.GetAwsConfig())

	if _, err := m.snsService.ListTopics(context.Background(), &sns.ListTopicsInput{}); err != nil {
		logging.Fatal(context.Background()).Err(err).Msg(connectionError)
	}

	return &m
}

func (m *awsMessaging) producer(ctx context.Context, p *Producer, msg *ProviderMessage) error {
	_, err := m.snsService.Publish(ctx, &sns.PublishInput{
		Message: aws.String(msg.String()),
		TopicArn: aws.String(fmt.Sprintf("arn:%s:sns:%s:%s:%s",
			cloud.GetAwsARN().Partition,
			cloud.GetAwsConfig().Region,
			cloud.GetAwsARN().AccountID,
			p.topic,
		)),
	})

	return err
}

func (m *awsMessaging) consumer(ctx context.Context, c *consumer) (chan *ProviderMessage, error) {
	ch := make(chan *ProviderMessage, 1)
	queueUrl := m.getQueueUrl(ctx, c.queue)

	// the poll is part of what Close waits for. A poll left running behind a closed consumer
	// keeps receiving from the queue, and every message it takes is one the next consumer on
	// that queue never sees
	c.Add(1)

	go func() {
		defer c.Done()

		for {
			if c.isCanceled() {
				return
			}

			// the receive in flight is deliberately not canceled when the consumer closes.
			// SQS keeps serving a long poll whose caller has gone away, so a message it
			// hands back would leave the queue with no receipt handle left to release it by.
			// Waiting for the response costs at most WaitTimeSeconds and keeps that handle
			// in reach
			msgs, err := m.readMessages(ctx, queueUrl)
			if err != nil {
				// the module context ends the poll on shutdown, which is the expected end of
				// the loop and not a failure to report
				if ctx.Err() != nil {
					return
				}

				logging.Error(ctx).Err(err).Msgf(couldNotReceiveMsg, c.queue)
				continue
			}

			if len(msgs.Messages) == 0 {
				continue
			}

			// closed while the receive was in flight: hand the message straight back so the
			// next consumer on this queue gets it instead of waiting out the visibility
			// timeout
			if c.isCanceled() {
				m.releaseMessage(ctx, c, queueUrl, &msgs.Messages[0])
				return
			}

			m.handleMessage(ctx, c, queueUrl, &msgs.Messages[0], ch)
		}
	}()

	return ch, nil
}

// releaseMessage makes a message visible again straight away when the consumer that received
// it is no longer able to process it. Without it the message stays in flight until the
// queue's visibility timeout expires (30s by default), which for a queue whose consumer is
// replaced by another one reads as a message that never arrives. It runs on a context of its
// own because the one it is given is the module context, already canceled when the release
// follows a shutdown.
func (m *awsMessaging) releaseMessage(ctx context.Context, c *consumer, queueUrl *sqs.GetQueueUrlOutput, msg *sqstypes.Message) {
	releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), releaseTimeout)
	defer cancel()

	if _, err := m.sqsService.ChangeMessageVisibility(releaseCtx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          queueUrl.QueueUrl,
		ReceiptHandle:     msg.ReceiptHandle,
		VisibilityTimeout: 0,
	}); err != nil {
		logging.Error(ctx).Err(err).Msgf(couldNotReleaseMsg, *msg.MessageId, c.queue)
	}
}

// handleMessage unwraps the SNS notification and delivers the message to the consumer
// channel with a real ack/nack bound to the SQS receipt handle. Malformed messages are
// skipped without deletion so they redeliver after the visibility timeout and the
// queue's redrive policy can route them to the DLQ.
func (m *awsMessaging) handleMessage(ctx context.Context, c *consumer, queueUrl *sqs.GetQueueUrlOutput, msg *sqstypes.Message, ch chan *ProviderMessage) {
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
	pm.setReceiptMetadata(awsAttributes(msg), awsDeliveryAttempt(msg))

	select {
	case ch <- &pm:
	case <-c.done:
		// nobody is listening anymore, so hand the message back rather than block this
		// goroutine forever. At most one message per consumer is in flight, so releasing it
		// redelivers a single message and not the whole queue at once
		m.releaseMessage(ctx, c, queueUrl, msg)
	}
}

// awsAttributes flattens the SQS system attributes and the user message attributes
// into a single string map so consumers can read broker metadata uniformly.
func awsAttributes(msg *sqstypes.Message) map[string]string {
	attrs := make(map[string]string, len(msg.Attributes)+len(msg.MessageAttributes))
	for k, v := range msg.Attributes {
		attrs[k] = v
	}
	for k, v := range msg.MessageAttributes {
		if v.StringValue != nil {
			attrs[k] = *v.StringValue
		}
	}
	if len(attrs) == 0 {
		return nil
	}
	return attrs
}

// awsDeliveryAttempt derives the delivery count from the SQS ApproximateReceiveCount
// system attribute when present.
func awsDeliveryAttempt(msg *sqstypes.Message) *int {
	v, ok := msg.Attributes["ApproximateReceiveCount"]
	if !ok {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return nil
	}
	return &n
}

func (m *awsMessaging) readMessages(ctx context.Context, queueResult *sqs.GetQueueUrlOutput) (*sqs.ReceiveMessageOutput, error) {
	var msgs, err = m.sqsService.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:                    queueResult.QueueUrl,
		MaxNumberOfMessages:         1,
		WaitTimeSeconds:             1,
		MessageSystemAttributeNames: []sqstypes.MessageSystemAttributeName{sqstypes.MessageSystemAttributeNameAll},
		MessageAttributeNames:       []string{"All"},
	})

	return msgs, err
}

func (m *awsMessaging) getQueueUrl(ctx context.Context, queue string) *sqs.GetQueueUrlOutput {
	queueResult, err := m.sqsService.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: aws.String(queue)})
	if err != nil {
		logging.Fatal(ctx).Err(err).Msgf(couldNotConnectQueue, queue)
	}

	if queueResult.QueueUrl == nil {
		logging.Fatal(ctx).Msgf(queueNotFound, queue)
	}

	return queueResult
}
