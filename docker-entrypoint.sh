#!/bin/sh

set -e

export PUID=${PUID:-0}
export PGID=${PGID:-0}
export GROUP_NAME="app"
export USER_NAME="app"

# This function evaluates if the supplied PGID is already in use
# if it is not in use, it creates the group with the PGID
# if it is in use, it sets the GROUP_NAME to the existing group
create_group() {
  if ! getent group ${PGID} > /dev/null 2>&1; then
    addgroup -g ${PGID} ${GROUP_NAME}
  else
    existing_group=$(getent group ${PGID} | cut -d: -f1)
    export GROUP_NAME=${existing_group}
  fi
}

# This function evaluates if the supplied PUID is already in use
# if it is not in use, it creates the user with the PUID and PGID
create_user() {
  if ! getent passwd ${PUID} > /dev/null 2>&1; then
    adduser -u ${PUID} -G ${GROUP_NAME} -s /bin/sh -D ${USER_NAME}
  else
    existing_user=$(getent passwd ${PUID} | cut -d: -f1)
    export USER_NAME=${existing_user}
  fi
}

# Run the needed functions to create the user and group
create_group
create_user

load_env_from_file() {
  VAR_NAME=$1
  eval "value=\${$VAR_NAME:-}"
  if [ -z "$value" ]; then
    eval "value_FILE=\${${VAR_NAME}_FILE:-}"
    if [ -n "$value_FILE" ]; then
      TEMP=$(cat "$value_FILE")
      eval "export $VAR_NAME=\"$TEMP\""
      echo "Loaded $VAR_NAME"
    fi
  fi
}

load_env_from_file LISTMONK_db__user
load_env_from_file LISTMONK_db__password
load_env_from_file LISTMONK_db__database
load_env_from_file LISTMONK_ADMIN_USER
load_env_from_file LISTMONK_ADMIN_PASSWORD

# Try to set the ownership of the app directory to the app user.
if ! chown -R ${PUID}:${PGID} /listmonk 2>/dev/null; then
  echo "Warning: Failed to change ownership of /listmonk. Readonly volume?"
fi

echo "Launching listmonk with user=[${USER_NAME}] group=[${GROUP_NAME}] PUID=[${PUID}] PGID=[${PGID}]"

# If running as root and PUID is not 0, then execute command as PUID
# this allows us to run the container as a non-root user
if [ "$(id -u)" = "0" ] && [ "${PUID}" != "0" ]; then
  su-exec ${PUID}:${PGID} "$@"
else
  exec "$@"
fi
