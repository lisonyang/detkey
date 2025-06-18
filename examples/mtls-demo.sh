#!/bin/bash

# mTLS Demo Script
# Demonstrates how to use the detkey tool to generate all certificates and keys required for mTLS

set -e

echo "=== DetKey mTLS Demo ==="
echo "This script demonstrates how to use the detkey tool to generate an mTLS environment"
echo ""

# Check if detkey exists
if ! command -v ./detkey &> /dev/null && ! command -v detkey &> /dev/null; then
    echo "Error: Cannot find detkey tool. Please build or install detkey first."
    exit 1
fi

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Set detkey command
DETKEY_CMD="$PROJECT_DIR/detkey"
if ! command -v "$DETKEY_CMD" &> /dev/null; then
    if command -v ./detkey &> /dev/null; then
        DETKEY_CMD="./detkey"
    elif command -v detkey &> /dev/null; then
        DETKEY_CMD="detkey"
    else
        echo "Error: Cannot find detkey tool. Please build or install detkey first."
        exit 1
    fi
fi

# Set password (read from stdin or use default password)
if [ -t 0 ]; then
    # If terminal, use default password for demo
    PASSWORD="demo123"
    echo "Using default demo password"
else
    # If piped input, read password
    read -r PASSWORD
    echo "Using input password"
fi

# Create temporary directory
TEMP_DIR=$(mktemp -d)
echo "Created temporary directory: $TEMP_DIR"
cd "$TEMP_DIR"

echo ""
echo "=== Step 1: Generate CA private key and certificate ==="

# Generate CA private key (RSA 4096-bit for production-grade security)
echo "Generating CA private key..."
echo "$PASSWORD" | $DETKEY_CMD --context "mtls/ca/v1" --type rsa4096 > ca.key
echo "✓ CA private key generated (ca.key)"

# Create self-signed CA certificate
echo "Creating CA certificate..."
openssl req -x509 -new -nodes -key ca.key -sha256 -days 1825 -out ca.crt \
    -subj "/C=US/ST=California/L=San Francisco/O=DetKey Demo/OU=CA/CN=DetKey Demo CA"
echo "✓ CA certificate created (ca.crt)"

echo ""
echo "=== Step 2: Generate server private key and certificate ==="

# Generate server private key
echo "Generating server private key..."
echo "$PASSWORD" | $DETKEY_CMD --context "mtls/server/api.example.com/v1" --type rsa2048 > server.key
echo "✓ Server private key generated (server.key)"

# Create server certificate signing request (CSR)
echo "Creating server CSR..."
openssl req -new -key server.key -out server.csr \
    -subj "/C=US/ST=California/L=San Francisco/O=DetKey Demo/OU=Server/CN=api.example.com"
echo "✓ Server CSR created (server.csr)"

# Create server certificate extensions file for SAN
cat > server.ext << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = api.example.com
DNS.2 = localhost
IP.1 = 127.0.0.1
EOF

# Sign server certificate with CA
echo "Signing server certificate..."
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out server.crt -days 365 -sha256 -extfile server.ext
echo "✓ Server certificate signed (server.crt)"

echo ""
echo "=== Step 3: Generate client private key and certificate ==="

# Generate client private key
echo "Generating client private key..."
echo "$PASSWORD" | $DETKEY_CMD --context "mtls/client/dashboard/v1" --type rsa2048 > client.key
echo "✓ Client private key generated (client.key)"

# Create client certificate signing request (CSR)
echo "Creating client CSR..."
openssl req -new -key client.key -out client.csr \
    -subj "/C=US/ST=California/L=San Francisco/O=DetKey Demo/OU=Client/CN=dashboard-client"
echo "✓ Client CSR created (client.csr)"

# Create client certificate extensions file
cat > client.ext << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = clientAuth
EOF

# Sign client certificate with CA
echo "Signing client certificate..."
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out client.crt -days 365 -sha256 -extfile client.ext
echo "✓ Client certificate signed (client.crt)"

echo ""
echo "=== Step 4: Verify certificates ==="

echo "Verifying CA certificate..."
openssl x509 -in ca.crt -text -noout | grep -A 2 "Subject:"

echo ""
echo "Verifying server certificate..."
openssl verify -CAfile ca.crt server.crt
openssl x509 -in server.crt -text -noout | grep -A 2 "Subject:"

echo ""
echo "Verifying client certificate..."
openssl verify -CAfile ca.crt client.crt
openssl x509 -in client.crt -text -noout | grep -A 2 "Subject:"

echo ""
echo "=== Generated files list ==="
ls -la *.crt *.key

echo ""
echo "=== mTLS setup complete! ==="
echo ""
echo "Generated files location: $TEMP_DIR"
echo ""
echo "File descriptions:"
echo "  ca.crt + ca.key         - CA certificate and private key"
echo "  server.crt + server.key - Server certificate and private key"
echo "  client.crt + client.key - Client certificate and private key"
echo ""

echo "=== Key Management Benefits ==="
echo "✅ All private keys can be regenerated using detkey and your master password"
echo "✅ Only certificate files (*.crt) need to be backed up"
echo "✅ Private keys can be regenerated anytime, anywhere"
echo "✅ For key rotation, simply change context version (v1 → v2)"
echo "✅ No need to store or sync private key files"
echo ""

echo "=== Usage Examples ==="
echo "Test mTLS connection (requires mTLS-enabled server):"
echo "curl --cert $TEMP_DIR/client.crt --key $TEMP_DIR/client.key --cacert $TEMP_DIR/ca.crt https://api.example.com"
echo ""

echo "Configure Nginx with generated certificates:"
cat << EOF
server {
    listen 443 ssl;
    server_name api.example.com;
    
    ssl_certificate     $TEMP_DIR/server.crt;
    ssl_certificate_key $TEMP_DIR/server.key;
    ssl_client_certificate $TEMP_DIR/ca.crt;
    ssl_verify_client on;
    
    location / {
        # Your application
    }
}
EOF
echo ""

echo "=== DetKey mTLS Command Reference ==="
echo "Regenerate CA private key:      echo 'your-password' | $DETKEY_CMD --context \"mtls/ca/v1\" --type rsa4096"
echo "Regenerate server private key:  echo 'your-password' | $DETKEY_CMD --context \"mtls/server/api.example.com/v1\" --type rsa2048"
echo "Regenerate client private key:  echo 'your-password' | $DETKEY_CMD --context \"mtls/client/dashboard/v1\" --type rsa2048"
echo ""

echo "=== Context Naming Best Practices ==="
echo "Use hierarchical naming for organization:"
echo "  mtls/ca/v1                           - Certificate Authority"
echo "  mtls/server/api.example.com/v1       - API server"
echo "  mtls/server/web.example.com/v1       - Web server"
echo "  mtls/client/monitoring/v1            - Monitoring client"
echo "  mtls/client/backup-service/v1        - Backup service client"
echo ""

echo "Demo completed. Temporary files saved at: $TEMP_DIR"
echo "To clean up, run: rm -rf $TEMP_DIR" 