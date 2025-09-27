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

load_secret_files() {
  # Save and restore IFS
  old_ifs="$IFS"
  IFS='
'
  # Capture all env variables starting with LISTMONK_ and ending with _FILE.
  # It's value is assumed to be a file path with its actual value.
  for line in $(env | grep '^LISTMONK_.*_FILE='); do
    var="${line%%=*}"
    fpath="${line#*=}"

    # If it's a valid file, read its contents and assign it to the var
    # without the _FILE suffix.
    # Eg: LISTMONK_DB_USER_FILE=/run/secrets/user -> LISTMONK_DB_USER=$(contents of /run/secrets/user)
    if [ -f "$fpath" ]; then
      new_var="${var%_FILE}"
      export "$new_var"="$(cat "$fpath")"
    fi
  done
  IFS="$old_ifs"
}

# Load env variables from files if LISTMONK_*_FILE variables are set.
load_secret_files

# Parse DATABASE_URL if it exists and set listmonk database environment variables
if [ -n "$DATABASE_URL" ]; then
  echo "DATABASE_URL found, parsing database configuration..."
  
  # Extract components from DATABASE_URL
  # Format: postgres://user:password@host:port/database?options
  
  # Remove postgres:// prefix
  db_url_no_prefix="${DATABASE_URL#postgres://}"
  
  # Extract user:password@host:port/database?options (remove query params)
  user_pass_host_port_db="${db_url_no_prefix%%\?*}"
  
  # Extract user:password part
  user_pass="${user_pass_host_port_db%%@*}"
  db_user="${user_pass%%:*}"
  db_password="${user_pass#*:}"
  
  # Extract host:port/database part
  host_port_db="${user_pass_host_port_db#*@}"
  
  # Extract host:port
  host_port="${host_port_db%%/*}"
  db_host="${host_port%%:*}"
  db_port="${host_port#*:}"
  
  # Extract database name
  db_name="${host_port_db#*/}"
  
  # Set environment variables for listmonk
  export LISTMONK_db__host="$db_host"
  export LISTMONK_db__port="$db_port"
  export LISTMONK_db__user="$db_user"
  export LISTMONK_db__password="$db_password"
  export LISTMONK_db__database="$db_name"
  export LISTMONK_db__ssl_mode="disable"
  
  echo "Database configuration parsed successfully:"
  echo "  Host: $db_host"
  echo "  Port: $db_port"
  echo "  Database: $db_name"
  echo "  User: $db_user"
  echo "  SSL Mode: disable"
else
  echo "No DATABASE_URL found, using config.toml settings"
fi

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
