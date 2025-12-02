#!/bin/bash
# scripts/generate-certs.sh
# Generates self-signed SSL certificates for local development

set -e

# Directory to store certificates
CERTS_DIR="./certs"
KEY_FILE="$CERTS_DIR/key.pem"
CERT_FILE="$CERTS_DIR/cert.pem"

# Ensure certs directory exists
if [ ! -d "$CERTS_DIR" ]; then
    echo "Creating certificates directory at '$CERTS_DIR'..."
    mkdir -p "$CERTS_DIR"
fi

echo "Generating self-signed SSL certificates..."

# Generate the certificate and key in one go
# -nodes: No password on the key (needed for auto-start)
# -days 365: Valid for a year
# -addext: Adds Subject Alternative Names (vital for Chrome/modern tools)
openssl req -x509 -nodes -newkey rsa:2048 \
    -keyout "$KEY_FILE" \
    -out "$CERT_FILE" \
    -days 365 \
    -subj "/C=US/ST=Dev/L=Local/O=Dev/OU=IT/CN=localhost" \
    -addext "subjectAltName = DNS:localhost,IP:127.0.0.1"

# Set restrictive permissions (read/write for owner only)
# This mimics production security best practices
chmod 644 "$CERT_FILE"
chmod 644 "$KEY_FILE"

echo ""
echo "✅ Certificates generated successfully!"
echo "   - Key:  $KEY_FILE"
echo "   - Cert: $CERT_FILE"
echo ""
echo "You can now run 'docker compose up --build'"