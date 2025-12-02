#!/bin/sh
set -e

# Define source and destination paths
SRC_CERT="/etc/ssl/certs/cert.pem"
SRC_KEY="/etc/ssl/certs/key.pem"

# The data directory is the standard location for postgres data
PG_DATA_DIR="/var/lib/postgresql/data/pgdata"
DEST_CERT="${PG_DATA_DIR}/server.crt"
DEST_KEY="${PG_DATA_DIR}/server.key"
CONF_FILE="${PG_DATA_DIR}/postgresql.conf"

echo "Copying certificates to a secure location..."
# Copy certs from the insecure mount to the secure data volume
cp "$SRC_CERT" "$DEST_CERT"
cp "$SRC_KEY" "$DEST_KEY"

echo "Setting correct ownership and permissions..."
# Set correct permissions on the newly copied key
chown postgres:postgres "$DEST_KEY"
chmod 600 "$DEST_KEY"

# --- THIS IS THE KEY ---
# Append the SSL configuration to postgresql.conf
echo "Enabling SSL in postgresql.conf..."
{
  echo ""
  echo "# --- SSL Settings Added by init script ---"
  echo "ssl = on"
  echo "ssl_cert_file = '${DEST_CERT}'"
  echo "ssl_key_file = '${DEST_KEY}'"
  echo "# --- End SSL Settings ---"
} >> "$CONF_FILE"

echo "SSL configuration and permission fix complete."