#!/bin/sh

set -e

export UID=${UID:-0}
export GID=${GID:-0}
export GROUP_NAME="app"
export USER_NAME="app"

# This function evaluates if the supplied GID is already in use
# if it is not in use, it creates the group with the GID
# if it is in use, it sets the GROUP_NAME to the existing group
create_group() {
  if ! getent group ${GID} > /dev/null 2>&1; then
    addgroup -g ${GID} ${GROUP_NAME}
  else
    existing_group=$(getent group ${GID} | cut -d: -f1)
    export GROUP_NAME=${existing_group}
  fi
}

# This function evaluates if the supplied UID is already in use
# if it is not in use, it creates the user with the UID and GID
create_user() {
  if ! getent passwd ${UID} > /dev/null 2>&1; then
    adduser -u ${UID} -G ${GROUP_NAME} -s /bin/sh -D ${USER_NAME}
  else
    existing_user=$(getent passwd ${UID} | cut -d: -f1)
    export USER_NAME=${existing_user}
  fi
}

# Run the needed functions to create the user and group
create_group
create_user

# Set the ownership of the app directory to the app user
chown -R ${UID}:${GID} /listmonk

echo "Launching listmonk with user=[${USER_NAME}] group=[${GROUP_NAME}] uid=[${UID}] gid=[${GID}]"

# If running as root and UID is not 0, then execute command as UID
# this allows us to run the container as a non-root user
if [ "$(id -u)" = "0" ] && [ "${UID}" != "0" ]; then
  su-exec ${UID}:${GID} "$@"
else
  exec "$@"
fi
