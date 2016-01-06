#!/bin/bash

PREFIX=dev
cat <<EOF
#
# General redis container
#
redis image redis:latest
redis hostname ${PREFIX}_redis
redis publish 6379
redis hook after.run sleep 2
redis hook after.start sleep 2

#
# General mongodb container
#
mongo image mongo:latest
mongo command mongod --smallfiles
mongo hostname ${PREFIX}_mongo
mongo publish 27017

#
# General nats container
#
nats image nats:latest
nats hostname ${PREFIX}_nats
nats publish 4222
EOF
