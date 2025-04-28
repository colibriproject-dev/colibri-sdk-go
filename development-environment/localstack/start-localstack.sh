#!/bin/bash

awslocal sns create-topic --name COLIBRI_PROJECT_USER_CREATE
awslocal sqs create-queue --queue-name COLIBRI_PROJECT_USER_CREATE_APP_CONSUMER
awslocal sns subscribe --topic-arn arn:aws:sns:us-east-1:000000000000:COLIBRI_PROJECT_USER_CREATE \
         --protocol sqs \
         --notification-endpoint arn:aws:sqs:us-east-1:000000000000:COLIBRI_PROJECT_USER_CREATE_APP_CONSUMER

awslocal sns create-topic --name COLIBRI_PROJECT_FAIL_USER_CREATE
awslocal sqs create-queue --queue-name COLIBRI_PROJECT_FAIL_USER_CREATE_APP_CONSUMER
awslocal sqs create-queue --queue-name COLIBRI_PROJECT_FAIL_USER_CREATE_APP_CONSUMER_DLQ
awslocal sns subscribe --topic-arn arn:aws:sns:us-east-1:000000000000:COLIBRI_PROJECT_FAIL_USER_CREATE \
         --protocol sqs \
         --notification-endpoint arn:aws:sqs:us-east-1:000000000000:COLIBRI_PROJECT_FAIL_USER_CREATE_APP_CONSUMER

awslocal sns create-topic --name COLIBRI_PROJECT_PRODUCER_TEST_APP_CONSUMER
awslocal sqs create-queue --queue-name COLIBRI_PROJECT_PRODUCER_TEST_APP_CONSUMER
awslocal sns subscribe --topic-arn arn:aws:sns:us-east-1:000000000000:COLIBRI_PROJECT_PRODUCER_TEST_APP_CONSUMER \
         --protocol sqs \
         --notification-endpoint arn:aws:sqs:us-east-1:000000000000:COLIBRI_PROJECT_PRODUCER_TEST_APP_CONSUMER

awslocal s3api create-bucket --bucket my-bucket --acl public-read

echo "localstack emulator started"
