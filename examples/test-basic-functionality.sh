#!/bin/bash

# DetKey Basic Functionality Test Script
# This script demonstrates detkey's deterministic key generation capabilities

echo "=== DetKey Basic Functionality Test ==="
echo

# Ensure detkey executable exists
if [ ! -f "../detkey" ]; then
    echo "Error: detkey executable not found. Please run 'make build' to build the project first."
    exit 1
fi

DETKEY="../detkey"
TEST_PASSWORD="test-password-123"
TEST_CONTEXT="ssh/prod-server/v1"

echo "Test parameters:"
echo "- Password: $TEST_PASSWORD"
echo "- Context: $TEST_CONTEXT"
echo

echo "=== Test 1: Deterministic Generation Verification ==="
echo "Generating keys twice with the same context to verify consistency..."

# First generation
echo "First public key generation:"
PUBKEY1=$(echo "$TEST_PASSWORD" | $DETKEY --context "$TEST_CONTEXT" --pub)
echo "$PUBKEY1"
echo

# Second generation
echo "Second public key generation:"
PUBKEY2=$(echo "$TEST_PASSWORD" | $DETKEY --context "$TEST_CONTEXT" --pub)
echo "$PUBKEY2"
echo

# Compare results
if [ "$PUBKEY1" = "$PUBKEY2" ]; then
    echo "✅ Deterministic test passed: Same input produces same output"
else
    echo "❌ Deterministic test failed: Same input produces different output"
    exit 1
fi
echo

echo "=== Test 2: Different Contexts Generate Different Keys ==="
CONTEXT1="ssh/production/web-server/v1"
CONTEXT2="ssh/staging/database/v1"

echo "Context 1: $CONTEXT1"
PUBKEY_CTX1=$(echo "$TEST_PASSWORD" | $DETKEY --context "$CONTEXT1" --pub)
echo "$PUBKEY_CTX1"
echo

echo "Context 2: $CONTEXT2"
PUBKEY_CTX2=$(echo "$TEST_PASSWORD" | $DETKEY --context "$CONTEXT2" --pub)
echo "$PUBKEY_CTX2"
echo

if [ "$PUBKEY_CTX1" != "$PUBKEY_CTX2" ]; then
    echo "✅ Context isolation test passed: Different contexts produce different keys"
else
    echo "❌ Context isolation test failed: Different contexts produce same keys"
    exit 1
fi
echo

echo "=== Test 3: Private and Public Key Format Validation ==="

echo "Generating private key (PEM format):"
PRIVATE_KEY=$(echo "$TEST_PASSWORD" | $DETKEY --context "$TEST_CONTEXT")
echo "$PRIVATE_KEY" | head -5
echo "... (truncated display)"
echo "$PRIVATE_KEY" | tail -5
echo

# Validate private key format
if echo "$PRIVATE_KEY" | grep -q "BEGIN OPENSSH PRIVATE KEY"; then
    echo "✅ Private key format validation passed: Standard OpenSSH PEM format"
else
    echo "❌ Private key format validation failed: Non-standard format"
    exit 1
fi

# Validate public key format
if echo "$PUBKEY1" | grep -q "^ssh-ed25519 "; then
    echo "✅ Public key format validation passed: Standard SSH public key format"
else
    echo "❌ Public key format validation failed: Non-standard format"
    exit 1
fi
echo

echo "=== Test 4: Version Control Functionality ==="
VERSION_V1="ssh/prod-server/v1"
VERSION_V2="ssh/prod-server/v2"

echo "Version v1:"
PUBKEY_V1=$(echo "$TEST_PASSWORD" | $DETKEY --context "$VERSION_V1" --pub)
echo "$PUBKEY_V1"
echo

echo "Version v2:"
PUBKEY_V2=$(echo "$TEST_PASSWORD" | $DETKEY --context "$VERSION_V2" --pub)
echo "$PUBKEY_V2"
echo

if [ "$PUBKEY_V1" != "$PUBKEY_V2" ]; then
    echo "✅ Version control test passed: Different versions produce different keys"
else
    echo "❌ Version control test failed: Different versions produce same keys"
    exit 1
fi
echo

echo "=== Test 5: Shell Function Usage Demo ==="
echo "Demonstrating the recommended sshd shell function approach:"
echo

cat << 'EOF'
# Add this function to your ~/.zshrc or ~/.bashrc:
sshd() {
    if [ "$#" -lt 2 ]; then
        echo "Usage: sshd <context> <user@host> [additional ssh options...]"
        return 1
    fi

    local context="$1"
    shift

    local tmp_key_file
    tmp_key_file=$(mktemp)
    if [ -z "$tmp_key_file" ]; then
        echo "Error: Unable to create temporary file." >&2
        return 1
    fi

    trap 'rm -f "$tmp_key_file"' EXIT INT TERM

    if ! detkey --context "$context" > "$tmp_key_file"; then
        echo "Error: detkey private key generation failed." >&2
        return 1
    fi

    echo "Connecting using derived key..." >&2
    ssh -i "$tmp_key_file" "$@"
}

# Usage examples:
sshd "ssh/production/web-server/v1" user@prod-server.com
sshd "ssh/staging/database/v1" admin@staging-db.internal
EOF

echo
echo "To deploy public key (one-time setup):"
echo "detkey --context \"ssh/production/web-server/v1\" --pub | ssh user@server \"cat >> ~/.ssh/authorized_keys\""
echo

echo "=== All Tests Passed! ==="
echo "DetKey tool is working correctly and ready for safe use."
echo
echo "Next steps:"
echo "1. Run 'make install' to install the tool to system path"
echo "2. Start using your real master password and contexts"
echo "3. Add the sshd shell function to your shell configuration"
echo "4. Create context patterns for your servers using hierarchical naming" 