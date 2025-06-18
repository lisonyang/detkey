# DetKey - Deterministic SSH Key Generator

[English](README.md) | [中文](README_zh.md)

DetKey is a powerful command-line tool that allows you to deterministically generate SSH keys using a master password and context string. This means the same input will always produce the same key pair, enabling you to regenerate identical SSH keys anywhere without storing or transferring key files.

## Core Features

- **Deterministic Generation**: Same master password and context always generate identical key pairs
- **Zero Dependencies**: Compiled to a single executable with no external dependencies
- **Cross-Platform**: Supports Linux, macOS, Windows
- **Security-First Design**: Uses Argon2id for key stretching and HKDF for key derivation
- **Standard Format**: Outputs standard OpenSSH format keys

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

### Basic Usage

```bash
# Generate private key
./detkey --context "ssh/server-a/v1"

# Generate public key
./detkey --context "ssh/server-a/v1" --pub
```

### Real-World Use Cases

#### 1. Deploy Public Key to Server

```bash
# Generate public key for a specific server and add to authorized_keys
./detkey --context "ssh/prod-server/v1" --pub | ssh user@server "cat >> ~/.ssh/authorized_keys"
```

#### 2. Login Using Generated Private Key

```bash
# Use process substitution to login directly without saving private key to disk
ssh -i <(./detkey --context "ssh/prod-server/v1") user@server
```

#### 3. Create Convenient Aliases

Add to your `~/.bashrc` or `~/.zshrc`:

```bash
alias ssh-prod='ssh -i <(detkey --context "ssh/prod-server/v1") user@prod-server'
alias ssh-dev='ssh -i <(detkey --context "ssh/dev-server/v1") user@dev-server'
```

Then you can simply run:

```bash
ssh-prod  # Connect to production server
ssh-dev   # Connect to development server
```

## Context String Design

Context strings are used to distinguish different purposes. We recommend using hierarchical naming:

```
ssh/production/web-server-1/v1
ssh/staging/database/v1
ssh/personal/vps/v2
git/github/personal/v1
git/gitlab/work/v1
```

## Security Considerations

### Advantages

- **Key Stretching**: Uses Argon2id algorithm to make brute force attacks extremely costly
- **Isolation**: Different contexts generate completely independent keys
- **No Storage**: Keys are generated in memory and destroyed immediately after use
- **Deterministic**: No need to worry about key loss or backups

### Trade-offs

- **Master Password Strength**: The tool's security depends on your master password strength
- **Offline Attacks**: If an attacker obtains the tool and a known key pair, they might attempt to brute force the master password

### Best Practices

1. **Use Strong Master Password**: Recommended to use long passwords with uppercase, lowercase, numbers, and special characters
2. **Protect Tool Security**: Don't use in untrusted environments
3. **Context Version Control**: Change version number in context when keys need rotation
4. **Regular Rotation**: Periodically rotate keys for important services

## Technical Implementation

DetKey uses the following cryptographic components:

1. **Argon2id**: Converts user password to high-strength master seed
2. **HKDF**: Derives context-specific key seed from master seed
3. **Ed25519**: Generates SSH key pairs

### Key Generation Flow

```
Master Password → [Argon2id] → Master Seed → [HKDF + Context] → Final Seed → [Ed25519] → SSH Key Pair
```

## License

This project follows the same license as the repository.

## Contributing

Issues and pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.
