#!/bin/bash

export PUBSUB_EMULATOR_HOST=localhost:8686

gcloud pubsub topics create COLIBRI_PROJECT_USER_CREATE
gcloud pubsub topics create COLIBRI_PROJECT_FAIL_USER_CREATE
gcloud pubsub topics create COLIBRI_PROJECT_FAIL_USER_CREATE_DLT

gcloud pubsub subscriptions create COLIBRI_PROJECT_USER_CREATE_APP_CONSUMER --topic=COLIBRI_PROJECT_USER_CREATE
gcloud pubsub subscriptions create COLIBRI_PROJECT_FAIL_USER_CREATE_APP_CONSUMER --topic=COLIBRI_PROJECT_FAIL_USER_CREATE \
       --dead-letter-topic=projects/test-project/topics/COLIBRI_PROJECT_FAIL_USER_CREATE_DLT --max-delivery-attempts=5

gcloud pubsub subscriptions create COLIBRI_PROJECT_FAIL_USER_CREATE_APP_CONSUMER_DLQ --topic=COLIBRI_PROJECT_FAIL_USER_CREATE_DLT