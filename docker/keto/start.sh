#!/bin/ash
export KETO_READ_REMOTE=127.0.0.1:4466
export KETO_WRITE_REMOTE=127.0.0.1:4467
keto status --insecure-disable-transport-security
# keto relation-tuple create /home/ory/relation-tuple.json --insecure-disable-transport-security
# keto relation-tuple get --subject-set="User:user_1"  --insecure-disable-transport-security