#!/bin/ash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

if [ -z "$SMQ_MQTT_CLUSTER" ]
then
      envsubst '${SMQ_MQTT_BROKER_TYPE} ${SMQ_FLUXMQ_MQTT_PORT}' < /etc/nginx/snippets/mqtt-upstream-single.conf > /etc/nginx/snippets/mqtt-upstream.conf
      envsubst '${SMQ_MQTT_BROKER_TYPE} ${SMQ_FLUXMQ_WS_PORT}' < /etc/nginx/snippets/mqtt-ws-upstream-single.conf > /etc/nginx/snippets/mqtt-ws-upstream.conf
      envsubst '${SMQ_MQTT_BROKER_TYPE} ${SMQ_FLUXMQ_HTTP_PORT}' < /etc/nginx/snippets/fluxmq-http-upstream-single.conf > /etc/nginx/snippets/fluxmq-http-upstream.conf
else
      envsubst '${SMQ_FLUXMQ_MQTT_PORT}' < /etc/nginx/snippets/mqtt-upstream-cluster.conf > /etc/nginx/snippets/mqtt-upstream.conf
      envsubst '${SMQ_FLUXMQ_WS_PORT}' < /etc/nginx/snippets/mqtt-ws-upstream-cluster.conf > /etc/nginx/snippets/mqtt-ws-upstream.conf
      envsubst '${SMQ_FLUXMQ_HTTP_PORT}' < /etc/nginx/snippets/fluxmq-http-upstream-cluster.conf > /etc/nginx/snippets/fluxmq-http-upstream.conf
fi

envsubst '
    ${SMQ_NGINX_SERVER_NAME}
    ${SMQ_AUTH_HTTP_PORT}
    ${SMQ_DOMAINS_HTTP_PORT}
    ${SMQ_GROUPS_HTTP_PORT}
    ${SMQ_USERS_HTTP_PORT}
    ${SMQ_CLIENTS_HTTP_PORT}
    ${SMQ_CLIENTS_AUTH_HTTP_PORT}
    ${SMQ_CHANNELS_HTTP_PORT}
    ${SMQ_NGINX_MQTT_PORT}
    ${SMQ_NGINX_MQTTS_PORT}' < /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf

exec nginx -g "daemon off;"
