package main

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// SALT 是一个固定的、公开的字符串。更改它会导致所有密钥都变化。
// 理想情况下，每个用户都应该使用自己独一无二的盐。
const SALT = "a-unique-salt-for-detkey-v1"

func main() {
	// --- 1. 解析命令行参数 ---
	context := flag.String("context", "", "用于密钥衍生的上下文字符串 (例如 'ssh/server-a/v1' 或 'mtls/ca/v1') (必须)")
	isPublicKey := flag.Bool("pub", false, "如果指定, 则只输出公钥, 否则输出私钥。")
	keyType := flag.String("type", "ed25519", "要生成的密钥类型 (ed25519, rsa2048, rsa4096)")
	outputFormat := flag.String("format", "auto", "输出格式 (auto, ssh, pem). auto 会根据上下文自动选择")
	flag.Parse()

	if *context == "" {
		flag.Usage()
		log.Fatal("错误: --context 参数是必须的。")
	}

	// 验证密钥类型
	if !isValidKeyType(*keyType) {
		log.Fatalf("错误: 不支持的密钥类型 '%s'。支持的类型: ed25519, rsa2048, rsa4096", *keyType)
	}

	// --- 2. 安全地读取主密码 ---
	var password []byte
	var err error
	
	// 检查是否是终端环境
	if term.IsTerminal(int(os.Stdin.Fd())) {
		// 在终端环境中安全读取密码（不回显）
		fmt.Print("请输入您的主密码: ")
		password, err = term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println() // 读取后换行
		if err != nil {
			log.Fatalf("错误: 无法读取密码: %v", err)
		}
	} else {
		// 在非终端环境中（如管道、脚本）从标准输入读取
		var line string
		line, err = readLine(os.Stdin)
		if err != nil {
			log.Fatalf("错误: 无法读取密码: %v", err)
		}
		password = []byte(line)
	}
	
	if len(password) == 0 {
		log.Fatal("错误: 密码不能为空。")
	}

	// --- 3. 衍生密钥并生成 ---
	privateKey, err := deriveAndGenerateKey(password, []byte(SALT), *context, *keyType)
	if err != nil {
		log.Fatalf("错误: 密钥生成失败: %v", err)
	}

	// --- 4. 确定输出格式 ---
	format := determineOutputFormat(*outputFormat, *context, *keyType)

	// --- 5. 根据参数输出结果 ---
	if *isPublicKey {
		err = outputPublicKey(privateKey, format)
	} else {
		err = outputPrivateKey(privateKey, format)
	}
	
	if err != nil {
		log.Fatalf("错误: 输出失败: %v", err)
	}
}

// deriveAndGenerateKey 是核心逻辑函数，现在支持多种密钥类型
func deriveAndGenerateKey(password, salt []byte, context, keyType string) (crypto.PrivateKey, error) {
	// --- 核心步骤 A: 密钥延伸 (Key Stretching) ---
	// 使用 Argon2id 对用户输入的密码进行"慢哈希"，生成一个高强度的 32 字节主种子。
	// 这使得对主密码的离线暴力破解变得极其昂贵。
	// Argon2id 的参数可以调整，值越大越安全，但生成速度越慢。
	masterSeed := argon2.IDKey(password, salt, 1, 64*1024, 4, 32)

	// --- 核心步骤 B: 密钥衍生 (Key Derivation) ---
	// 使用 HKDF 从主种子和上下文中衍生出最终的、用于生成密钥的种子。
	// 使用 SHA256 作为哈希函数。
	hkdfReader := hkdf.New(sha256.New, masterSeed, salt, []byte(context))

	// --- 核心步骤 C: 根据类型生成密钥 ---
	var privateKey crypto.PrivateKey
	var err error

	switch keyType {
	case "rsa2048":
		// 为 RSA 密钥生成创建无限熵源
		deterministicReader := newDeterministicReader(hkdfReader)
		privateKey, err = rsa.GenerateKey(deterministicReader, 2048)
	case "rsa4096":
		deterministicReader := newDeterministicReader(hkdfReader)
		privateKey, err = rsa.GenerateKey(deterministicReader, 4096)
	case "ed25519":
		finalSeed := make([]byte, ed25519.SeedSize) // Ed25519 需要 32 字节的种子
		if _, err = io.ReadFull(hkdfReader, finalSeed); err != nil {
			return nil, fmt.Errorf("无法从 HKDF 读取最终种子: %w", err)
		}
		privateKey = ed25519.NewKeyFromSeed(finalSeed)
	default:
		return nil, fmt.Errorf("不支持的密钥类型: %s", keyType)
	}

	if err != nil {
		return nil, fmt.Errorf("无法生成 %s 密钥: %w", keyType, err)
	}

	return privateKey, nil
}

// deterministicReader 提供快速的确定性熵源
type deterministicReader struct {
	seed    [32]byte
	counter uint64
	buffer  []byte
	bufPos  int
}

// newDeterministicReader 创建一个新的高效确定性读取器
func newDeterministicReader(hkdf io.Reader) *deterministicReader {
	dr := &deterministicReader{
		buffer: make([]byte, 8192), // 8KB 缓冲区
		bufPos: 0,
	}
	
	// 从 HKDF 读取种子
	_, err := io.ReadFull(hkdf, dr.seed[:])
	if err != nil {
		// 如果读取失败，使用默认种子（不应该发生）
		copy(dr.seed[:], []byte("default-seed-for-rsa-generation"))
	}
	
	// 预填充缓冲区
	dr.refillBuffer()
	
	return dr
}

// refillBuffer 使用快速哈希算法重新填充缓冲区
func (dr *deterministicReader) refillBuffer() {
	hasher := sha256.New()
	for i := 0; i < len(dr.buffer)/32; i++ {
		hasher.Reset()
		hasher.Write(dr.seed[:])
		hasher.Write([]byte{byte(dr.counter), byte(dr.counter >> 8), byte(dr.counter >> 16), byte(dr.counter >> 24),
			byte(dr.counter >> 32), byte(dr.counter >> 40), byte(dr.counter >> 48), byte(dr.counter >> 56)})
		chunk := hasher.Sum(nil)
		copy(dr.buffer[i*32:(i+1)*32], chunk)
		dr.counter++
	}
	dr.bufPos = 0
}

// Read 实现 io.Reader 接口，提供快速的确定性熵
func (dr *deterministicReader) Read(p []byte) (n int, err error) {
	totalRead := 0
	
	for totalRead < len(p) {
		// 如果缓冲区耗尽，重新填充
		if dr.bufPos >= len(dr.buffer) {
			dr.refillBuffer()
		}
		
		// 从缓冲区复制数据
		toCopy := len(p) - totalRead
		remaining := len(dr.buffer) - dr.bufPos
		if toCopy > remaining {
			toCopy = remaining
		}
		
		copy(p[totalRead:totalRead+toCopy], dr.buffer[dr.bufPos:dr.bufPos+toCopy])
		dr.bufPos += toCopy
		totalRead += toCopy
	}
	
	return totalRead, nil
}

// isValidKeyType 检查密钥类型是否有效
func isValidKeyType(keyType string) bool {
	validTypes := []string{"ed25519", "rsa2048", "rsa4096"}
	for _, t := range validTypes {
		if t == keyType {
			return true
		}
	}
	return false
}

// determineOutputFormat 根据上下文和参数确定输出格式
func determineOutputFormat(format, context, keyType string) string {
	if format != "auto" {
		return format
	}
	
	// 如果上下文包含 "mtls"，默认使用 PEM 格式
	if containsString(context, "mtls") {
		return "pem"
	}
	
	// 如果上下文包含 "ssh"，默认使用 SSH 格式
	if containsString(context, "ssh") {
		return "ssh"
	}
	
	// 对于 RSA 密钥，在没有明确上下文时，默认使用 PEM 格式
	if keyType == "rsa2048" || keyType == "rsa4096" {
		return "pem"
	}
	
	// 默认使用 SSH 格式
	return "ssh"
}

// containsString 检查字符串是否包含子字符串
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && (s[:len(substr)+1] == substr+"/" || 
		     s[len(s)-len(substr)-1:] == "/"+substr || 
		     containsSubstring(s, "/"+substr+"/"))))
}

func containsSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// outputPublicKey 输出公钥
func outputPublicKey(privateKey crypto.PrivateKey, format string) error {
	switch format {
	case "ssh":
		return outputSSHPublicKey(privateKey)
	case "pem":
		return outputPEMPublicKey(privateKey)
	default:
		return fmt.Errorf("不支持的公钥格式: %s", format)
	}
}

// outputPrivateKey 输出私钥
func outputPrivateKey(privateKey crypto.PrivateKey, format string) error {
	switch format {
	case "ssh":
		return outputSSHPrivateKey(privateKey)
	case "pem":
		return outputPEMPrivateKey(privateKey)
	default:
		return fmt.Errorf("不支持的私钥格式: %s", format)
	}
}

// outputSSHPublicKey 输出 SSH 格式的公钥
func outputSSHPublicKey(privateKey crypto.PrivateKey) error {
	publicKey := privateKey.(interface{ Public() crypto.PublicKey }).Public()
	sshPubKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("无法创建 SSH 公钥: %w", err)
	}
	fmt.Print(string(ssh.MarshalAuthorizedKey(sshPubKey)))
	return nil
}

// outputPEMPublicKey 输出 PEM 格式的公钥
func outputPEMPublicKey(privateKey crypto.PrivateKey) error {
	publicKey := privateKey.(interface{ Public() crypto.PublicKey }).Public()
	
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("无法序列化公钥: %w", err)
	}
	
	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}
	
	return pem.Encode(os.Stdout, pemBlock)
}

// outputSSHPrivateKey 输出 SSH 格式的私钥
func outputSSHPrivateKey(privateKey crypto.PrivateKey) error {
	pemBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return fmt.Errorf("无法序列化私钥: %w", err)
	}
	return pem.Encode(os.Stdout, pemBlock)
}

// outputPEMPrivateKey 输出 PEM 格式的私钥
func outputPEMPrivateKey(privateKey crypto.PrivateKey) error {
	var pemBlock *pem.Block
	
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		keyBytes := x509.MarshalPKCS1PrivateKey(key)
		pemBlock = &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: keyBytes,
		}
	case ed25519.PrivateKey:
		keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			return fmt.Errorf("无法序列化 Ed25519 私钥: %w", err)
		}
		pemBlock = &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: keyBytes,
		}
	default:
		return fmt.Errorf("不支持的私钥类型")
	}
	
	return pem.Encode(os.Stdout, pemBlock)
}

// readLine 从给定的 io.Reader 中读取一行文本
// 用于在非终端环境中读取密码
func readLine(reader io.Reader) (string, error) {
	var line []byte
	buffer := make([]byte, 1)
	
	for {
		n, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				break
			}
			return "", err
		}
		if n > 0 {
			if buffer[0] == '\n' {
				break
			}
			if buffer[0] != '\r' { // 忽略回车符
				line = append(line, buffer[0])
			}
		}
	}
	
	return string(line), nil
} 