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

**⚠️ Important: Reliable SSH Key Deployment**

The commonly suggested one-line pipe command can cause password input conflicts when both `detkey` and `ssh` try to read from the terminal simultaneously. For reliable deployment, use the **three-step file-based method** below:

**Step 1: Generate public key to temporary file**
```bash
# Generate public key and save to temporary file
./detkey --context "ssh/prod-server/v1" --pub > /tmp/prod-server.pub
```

**Step 2: Deploy using ssh-copy-id (recommended)**
```bash
# Use OpenSSH's official deployment tool
ssh-copy-id -i /tmp/prod-server.pub user@server
```

**Step 3: Clean up temporary file**
```bash
# Remove temporary file
rm /tmp/prod-server.pub
```

**Alternative deployment method (if ssh-copy-id is not available):**
```bash
# Step 1: Generate public key to temporary file
./detkey --context "ssh/prod-server/v1" --pub > /tmp/prod-server.pub

# Step 2: Deploy manually
cat /tmp/prod-server.pub | ssh user@server "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && chmod 700 ~/.ssh"

# Step 3: Clean up
rm /tmp/prod-server.pub
```

**After deployment, use the reliable SSH function method described in the SSH Workflow section below.**

## Examples

### SSH Workflow

**⚠️ Important: Reliable SSH Login Method**

The commonly suggested one-line commands like `ssh -i <(./detkey ...)` can cause terminal control conflicts where both `detkey` and `ssh` try to interact with the terminal simultaneously. This results in password input being scrambled. 

**For completely reliable SSH login, use the following shell function approach:**

#### Step 1: Add the SSH Helper Function

Add the following function to your shell configuration file (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
#
# detkey_ssh - Secure and reliable SSH login function
#
# This function uses a temporary private key file to resolve terminal control conflicts,
# ensuring stable operation in any environment.
#
detkey_ssh() {
    # Check parameters
    if [ "$#" -lt 2 ]; then
        echo "Usage: detkey_ssh <context> <user@host> [additional ssh options...]"
        return 1
    fi

    local context="$1"
    shift # Remove first parameter (context), rest are ssh command parameters

    # Create a secure temporary file for the private key
    # mktemp creates a file readable/writable only by the current user
    local tmp_key_file
    tmp_key_file=$(mktemp)
    if [ -z "$tmp_key_file" ]; then
        echo "Error: Unable to create temporary file." >&2
        return 1
    fi

    # Set a trap to ensure the temporary file is deleted automatically
    # no matter how the function exits (success, failure, interruption).
    # This is a critical security measure!
    trap 'rm -f "$tmp_key_file"' EXIT INT TERM

    # --- Step 1: Generate private key independently and write to temp file ---
    # This process runs independently without any programs competing for terminal access.
    # We redirect detkey's output to the temporary file.
    if ! detkey --context "$context" > "$tmp_key_file"; then
        echo "Error: detkey failed to generate private key." >&2
        # trap will trigger here and automatically delete the file
        return 1
    fi
    # At this point you'll be prompted for your master password - enter it here.

    # --- Step 2: Use the generated temporary private key file for SSH login ---
    # Now ssh reads the private key from a static file, it will properly gain terminal control
    # without conflicting with any other processes.
    echo "Connecting using derived key..." >&2
    ssh -i "$tmp_key_file" "$@"
    
    # After ssh command completes, function exits and trap automatically deletes temp file.
}
```

**Note**: Replace `detkey` with the actual path to your detkey executable if it's not in your PATH.

#### Step 2: Reload Your Shell Configuration

```bash
source ~/.bashrc  # or ~/.zshrc
```

#### Step 3: Use the Reliable SSH Login

Now your login workflow becomes:

```bash
# Connect to production server
detkey_ssh "ssh/prod-server/v1" user@server

# Connect with additional SSH options
detkey_ssh "ssh/prod-server/v1" user@server -p 2222

# Connect with port forwarding
detkey_ssh "ssh/prod-server/v1" user@server -L 8080:localhost:80
```

**Your login experience:**
1. You run: `detkey_ssh "ssh/prod-server/v1" user@server`
2. You see: `Enter your master password:` (enter your **master password**)
3. You see: `Connecting using derived key...`
4. You're logged in: `user@server:~$`

No more password conflicts, no more confusion - just reliable, secure SSH access.

### mTLS Certificate Generation

DetKey can generate RSA private keys for mutual TLS (mTLS) authentication, perfect for microservices, API authentication, and secure inter-service communication.

#### Quick mTLS Setup

Generate all required private keys and certificates for a complete mTLS setup:

```bash
# Generate CA private key and certificate
./detkey --context "mtls/ca/v1" --type rsa4096 --action create-ca-cert --subj "/CN=My Internal CA" > ca.crt

# Generate server private key and certificate
./detkey --context "mtls/server/api.example.com/v1" --type rsa4096 --action sign-cert --ca-context "mtls/ca/v1" --subj "/CN=api.example.com" < ca.crt > server.crt

# Generate client private key and certificate
./detkey --context "mtls/client/dashboard/v1" --type rsa4096 --action sign-cert --ca-context "mtls/ca/v1" --subj "/CN=dashboard-client" < ca.crt > client.crt
```


#### mTLS Demo Script

For a complete mTLS setup example, see `./examples/mtls-demo.sh`.

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

### Password Input Conflicts

If you encounter the error "input password will be scrambled" or similar, it means two programs are trying to read from the terminal simultaneously. This is exactly why we recommend the three-step file method instead of pipe commands.
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

## License

This project follows the same license as the repository.

## Contributing

Issues and pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.
