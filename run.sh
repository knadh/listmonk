#!/bin/sh

# Run database installation only once
./listmonk --install --idempotent --yes

# Then start the actual application
./listmonk
