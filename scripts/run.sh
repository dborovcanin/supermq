#!/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

###
# Runs all SuperMQ microservices (must be previously built and installed).
#
# Expects that PostgreSQL and needed messaging DB are alredy running.
# Additionally, core services depend on external infrastructure (DB and NATS).
#
###

BUILD_DIR=../build

# Kill all supermq-* stuff
function cleanup {
    pkill supermq
    pkill nats
}

###
# NATS
###
nats-server &
counter=1
until fuser 4222/tcp 1>/dev/null 2>&1;
do
    sleep 0.5
    ((counter++))
    if [ ${counter} -gt 10 ]
    then
        echo "NATS failed to start in 5 sec, exiting"
        exit 1
    fi
    echo "Waiting for NATS server"
done

###
# Users
###
SMQ_USERS_LOG_LEVEL=info SMQ_USERS_HTTP_PORT=9002 SMQ_USERS_GRPC_PORT=7001 SMQ_USERS_ADMIN_EMAIL=admin@supermq.com SMQ_USERS_ADMIN_PASSWORD=12345678 SMQ_USERS_ADMIN_USERNAME=admin SMQ_PASSWORD_RESET_URL_PREFIX=http://localhost:9002/password/reset SMQ_PASSWORD_RESET_EMAIL_TEMPLATE=../docker/templates/reset-password-email.tmpl SMQ_VERIFICATION_URL_PREFIX=http://localhost:9002/users/verify-email SMQ_VERIFICATION_EMAIL_TEMPLATE=../docker/templates/verification-email.tmpl $BUILD_DIR/supermq-users &

###
# Clients
###
SMQ_CLIENTS_LOG_LEVEL=info SMQ_CLIENTS_HTTP_PORT=9000 SMQ_CLIENTS_GRPC_PORT=7000 SMQ_CLIENTS_AUTH_HTTP_PORT=9002 $BUILD_DIR/supermq-clients &

trap cleanup EXIT

while : ; do sleep 1 ; done
