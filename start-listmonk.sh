#!/usr/bin/env bash

# Source the DB variables from the env.
source ./env.sh

./listmonk --upgrade --yes
./listmonk
