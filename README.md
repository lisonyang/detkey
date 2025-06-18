# DetKey - Deterministic SSH Key Generator

[English](README.md) | [中文](README_zh.md)

DetKey is a powerful command-line tool that allows you to deterministically generate SSH keys and mTLS certificates using a master password and context string. This means the same input will always produce the same key pair, enabling you to regenerate identical keys anywhere without storing or transferring key files.

## Core Features

- **Deterministic Generation**: Same master password and context always generate identical key pairs
- **Multiple Key Types**: Supports Ed25519, RSA 2048-bit, and RSA 4096-bit keys
- **mTLS Support**: Generate RSA keys for mutual TLS authentication scenarios
- **Zero Dependencies**: Compiled to a single executable with no external dependencies
- **Cross-Platform**: Supports Linux, macOS, Windows
- **Security-First Design**: Uses Argon2id for key stretching and HKDF for key derivation
- **Multiple Output Formats**: Standard OpenSSH format and PEM format
- **Smart Format Detection**: Automatically chooses appropriate format based on context

## Installation

### One-Click Install (Recommended)

You can automatically download and install the latest version with:

```bash
curl -sfL https://raw.githubusercontent.com/lisonyang/detkey/main/install.sh | sh
```

This script will automatically:
- Detect your operating system and CPU architecture
- Download the corresponding binary from GitHub Releases
- Install to `/usr/local/bin` directory (may require sudo)

### Manual Installation

1. Visit the [Releases page](https://github.com/lisonyang/detkey/releases)
2. Download the archive for your system
3. Extract and move the `detkey` executable to a directory in your PATH

### Build from Source

Ensure you have Go 1.21 or higher installed, then run:

```bash
go mod tidy
go build -o detkey
```

#### Cross-Platform Compilation

To compile for different platforms:

```bash
# Linux (AMD64)
GOOS=linux GOARCH=amd64 go build -o detkey-linux

# Windows (AMD64)
GOOS=windows GOARCH=amd64 go build -o detkey.exe

# macOS (ARM64)
GOOS=darwin GOARCH=arm64 go build -o detkey-darwin-arm64
```

## Usage

### Command-Line Options

```bash
detkey [options]

Options:
  --context string    Context string for key derivation (required)
                     Examples: 'ssh/server-a/v1', 'mtls/ca/v1'
  --type string      Key type to generate (default "ed25519")
                     Options: ed25519, rsa2048, rsa4096
  --format string    Output format (default "auto")
                     Options: auto, ssh, pem
  --pub             Output public key instead of private key
```

### SSH Key Generation

#### Basic SSH Usage

```bash
# Generate Ed25519 private key (default)
./detkey --context "ssh/server-a/v1"

# Generate Ed25519 public key
./detkey --context "ssh/server-a/v1" --pub

# Generate RSA private key
./detkey --context "ssh/server-b/v1" --type rsa2048
```

#### Real-World SSH Use Cases

```bash
# Deploy public key to server
./detkey --context "ssh/prod-server/v1" --pub | ssh user@server "cat >> ~/.ssh/authorized_keys"

# Login using generated private key
ssh -i <(./detkey --context "ssh/prod-server/v1") user@server

# Create convenient aliases
alias ssh-prod='ssh -i <(detkey --context "ssh/prod-server/v1") user@prod-server'
```

### mTLS Certificate Generation

DetKey can generate RSA private keys for mutual TLS (mTLS) authentication, perfect for microservices, API authentication, and secure inter-service communication.

#### Quick mTLS Setup

Generate all required private keys for a complete mTLS setup:

```bash
# Generate CA private key
./detkey --context "mtls/ca/v1" --type rsa4096 > ca.key

# Generate server private key
./detkey --context "mtls/server/api.example.com/v1" --type rsa4096 > server.key

# Generate client private key  
./detkey --context "mtls/client/dashboard/v1" --type rsa4096 > client.key
```

#### Complete mTLS Certificate Chain

After generating private keys, create the certificate chain using OpenSSL:

```bash
# 1. Create self-signed CA certificate
openssl req -x509 -new -nodes -key ca.key -sha256 -days 1024 -out ca.crt \
    -subj "/CN=My Internal CA"

# 2. Create server certificate
openssl req -new -key server.key -out server.csr \
    -subj "/CN=api.example.com"
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out server.crt -days 365 -sha256

# 3. Create client certificate
openssl req -new -key client.key -out client.csr \
    -subj "/CN=dashboard-client"
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out client.crt -days 365 -sha256
```

#### mTLS Demo Script

Run the included demonstration script to see the complete mTLS setup process:

```bash
./examples/mtls-demo.sh
```

This script will:
- Generate all required private keys using DetKey
- Create a complete certificate chain
- Validate all certificates
- Provide usage examples

### Output Formats

DetKey supports multiple output formats and can automatically detect the appropriate format:

- **SSH Format**: Standard OpenSSH private/public key format
- **PEM Format**: PKCS#1 (RSA) or PKCS#8 (Ed25519) format for TLS/SSL use

Format is automatically selected based on context:
- `ssh/*` contexts → SSH format
- `mtls/*` contexts → PEM format  
- RSA keys → PEM format (when context is ambiguous)
- Ed25519 keys → SSH format (when context is ambiguous)

## Context String Design

Context strings are used to distinguish different purposes and ensure key isolation. We recommend using hierarchical naming:

### SSH Contexts
```
ssh/production/web-server-1/v1
ssh/staging/database/v1
ssh/personal/vps/v2
git/github/personal/v1
git/gitlab/work/v1
```

### mTLS Contexts
```
mtls/ca/v1                           # Certificate Authority
mtls/server/api.example.com/v1       # Server certificates
mtls/server/internal.service/v1      # Internal service
mtls/client/dashboard/v1             # Client certificates
mtls/client/monitoring/v1            # Monitoring client
```

### Version Control

Change the version number when keys need rotation:
```
mtls/ca/v1    → mtls/ca/v2           # CA key rotation
ssh/prod/v1   → ssh/prod/v2          # SSH key rotation
```

## Key Types and Use Cases

| Key Type | Bits | Use Cases | Performance |
|----------|------|-----------|-------------|
| `ed25519` | 256 | SSH authentication, Git signing | Fastest |
| `rsa2048` | 2048 | Legacy TLS, older systems | Moderate |
| `rsa4096` | 4096 | High-security TLS, CA keys | Slower |

**Recommendations:**
- **SSH**: Use `ed25519` (default) for best performance and security
- **mTLS**: Use `rsa4096` for CA keys, `rsa2048` for client/server keys
- **Legacy Systems**: Use `rsa2048` when `ed25519` is not supported

## Security Considerations

### Advantages

- **Key Stretching**: Uses Argon2id algorithm to make brute force attacks extremely costly
- **Isolation**: Different contexts generate completely independent keys
- **No Storage**: Keys are generated in memory and destroyed immediately after use
- **Deterministic**: No need to worry about key loss or backups
- **Perfect Forward Secrecy**: Keys can be rotated by changing version numbers

### Trade-offs

- **Master Password Strength**: The tool's security depends on your master password strength
- **Offline Attacks**: If an attacker obtains the tool and a known key pair, they might attempt to brute force the master password

### Best Practices

1. **Use Strong Master Password**: Recommended to use long passwords with uppercase, lowercase, numbers, and special characters
2. **Protect Tool Security**: Don't use in untrusted environments
3. **Context Version Control**: Change version number in context when keys need rotation
4. **Regular Rotation**: Periodically rotate keys for important services
5. **Backup Certificates**: While private keys are deterministic, back up your certificates (.crt files)

## mTLS Benefits with DetKey

### Traditional mTLS Pain Points
- **Key Management**: Storing and distributing private keys securely
- **Backup & Recovery**: Risk of losing private keys
- **Key Rotation**: Complex process of replacing keys everywhere

### DetKey Solutions
- **Deterministic Generation**: Regenerate identical keys anywhere with just the master password
- **No Key Storage**: Private keys don't need to be stored on disk
- **Easy Rotation**: Change version in context string to rotate keys
- **Simplified Distribution**: Only certificates need to be distributed, not private keys

## Technical Implementation

DetKey uses the following cryptographic components:

1. **Argon2id**: Converts user password to high-strength master seed
2. **HKDF**: Derives context-specific key seed from master seed
3. **Key Generation**: 
   - **Ed25519**: Deterministic key generation from seed
   - **RSA**: Uses HKDF output as entropy source for deterministic generation

### Key Generation Flow

```
Master Password → [Argon2id] → Master Seed → [HKDF + Context] → Entropy → [Algorithm] → Key Pair
```

The same master password and context will always produce identical keys across different machines and time periods.

## Examples

### SSH Workflow
```bash
# Generate and deploy SSH key for production server
./detkey --context "ssh/prod-web/v1" --pub | ssh user@prod-server "cat >> ~/.ssh/authorized_keys"

# Login to server (key generated on-demand)
ssh -i <(./detkey --context "ssh/prod-web/v1") user@prod-server
```

### mTLS Workflow
```bash
# Generate all private keys for microservice mTLS
./detkey --context "mtls/ca/v1" --type rsa4096 > ca.key
./detkey --context "mtls/server/auth-service/v1" --type rsa2048 > auth-server.key
./detkey --context "mtls/client/web-frontend/v1" --type rsa2048 > web-client.key

# Create certificate chain (certificates need to be created once and distributed)
# ... (OpenSSL commands as shown above)

# Later, regenerate any private key when needed
./detkey --context "mtls/server/auth-service/v1" --type rsa2048 > auth-server.key
```

## License

This project follows the same license as the repository.

## Contributing

Issues and pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.
