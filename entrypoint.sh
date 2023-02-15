#!/bin/sh

envvars="/minu/env/ocs-messaging-secrets"
. $envvars   

if [ "$1" = "start" ]
then
  # starting golang application
  exec /minu/ocs-messaging
else
  # executing command supressed in the command line
  exec "$@"
fi