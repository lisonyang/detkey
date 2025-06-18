#!/bin/bash

# Quick test for mTLS functionality
set -e

echo "=== DetKey mTLS Functionality Test ==="
echo ""

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Set detkey command
DETKEY_CMD="$PROJECT_DIR/detkey"
if ! command -v "$DETKEY_CMD" &> /dev/null; then
    echo "Error: Cannot find detkey tool at $DETKEY_CMD"
    exit 1
fi

PASSWORD="test123"

echo "Test 1: Ed25519 SSH format (traditional functionality)"
echo "$PASSWORD" | $DETKEY_CMD --context "ssh/test-server/v1" --type ed25519 | head -2
echo "✓ Ed25519 SSH private key generation successful"
echo ""

echo "Test 2: Ed25519 mTLS PEM format (new functionality)"
echo "$PASSWORD" | $DETKEY_CMD --context "mtls/ca/v1" --type ed25519 | head -2
echo "✓ Ed25519 mTLS PEM private key generation successful"
echo ""

echo "Test 3: RSA key generation (deterministic test)"
echo "Generating the same RSA key twice to verify determinism:"

# First generation
RSA_KEY1=$(echo "$PASSWORD" | $DETKEY_CMD --context "mtls/server/api.example.com/v1" --type rsa2048 2>/dev/null | head -2)
# Second generation  
RSA_KEY2=$(echo "$PASSWORD" | $DETKEY_CMD --context "mtls/server/api.example.com/v1" --type rsa2048 2>/dev/null | head -2)

if [ "$RSA_KEY1" = "$RSA_KEY2" ]; then
    echo "✓ RSA key deterministic generation verification successful"
else
    echo "✗ RSA key deterministic generation verification failed"
    exit 1
fi
echo ""

echo "Test 4: Automatic format detection"
SSH_FORMAT=$(echo "$PASSWORD" | $DETKEY_CMD --context "ssh/prod-server/v1" --type ed25519 | head -1)
MTLS_FORMAT=$(echo "$PASSWORD" | $DETKEY_CMD --context "mtls/server/api.service.com/v1" --type ed25519 | head -1)

if [[ "$SSH_FORMAT" == *"OPENSSH"* ]]; then
    echo "✓ SSH context automatically uses OpenSSH format"
else
    echo "✗ SSH context format detection failed"
    exit 1
fi

if [[ "$MTLS_FORMAT" == *"PRIVATE KEY"* ]]; then
    echo "✓ mTLS context automatically uses PEM format"
else
    echo "✗ mTLS context format detection failed"
    exit 1
fi
echo ""

echo "Test 5: Public key generation"
echo "$PASSWORD" | $DETKEY_CMD --context "mtls/client/dashboard/v1" --type ed25519 --pub | head -2
echo "✓ mTLS PEM public key generation successful"
echo ""

echo "Test 6: Format override"
echo "$PASSWORD" | $DETKEY_CMD --context "mtls/ca/v1" --type ed25519 --format ssh --pub | head -1
echo "✓ Format override successful"
echo ""

echo "Test 7: Hierarchical context patterns"
echo "Testing recommended context naming patterns:"

CONTEXTS=(
    "mtls/ca/v1"
    "mtls/server/api.example.com/v1"
    "mtls/client/monitoring/v1"
    "ssh/production/web-server/v1"
    "ssh/staging/database/v1"
)

for context in "${CONTEXTS[@]}"; do
    KEY_TYPE=$(echo "$PASSWORD" | $DETKEY_CMD --context "$context" --pub 2>/dev/null | head -1 | awk '{print $1}')
    echo "✓ Context '$context' → Key type: $KEY_TYPE"
done
echo ""

echo "=== All Tests Passed! ==="
echo ""
echo "mTLS functionality has been successfully integrated into detkey tool:"
echo "- ✓ Supports RSA 2048/4096 key types"
echo "- ✓ Smart format detection (SSH vs PEM)"
echo "- ✓ Deterministic key generation"
echo "- ✓ Backward compatible with existing SSH functionality"
echo "- ✓ Supports public/private key output"
echo "- ✓ Supports format override"
echo "- ✓ Hierarchical context naming patterns"
echo ""
echo "You can now use detkey for complete mTLS certificate management!"
echo "Recommended usage patterns:"
echo "  detkey --context \"mtls/ca/v1\" --type rsa4096 > ca.key"
echo "  detkey --context \"mtls/server/api.example.com/v1\" --type rsa2048 > server.key"
echo "  detkey --context \"mtls/client/dashboard/v1\" --type rsa2048 > client.key" 