package main

import (
	"crypto/ed25519"
	"crypto/sha256"
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
	context := flag.String("context", "", "用于密钥衍生的上下文字符串 (例如 'ssh/server-a/v1') (必须)")
	isPublicKey := flag.Bool("pub", false, "如果指定, 则只输出公钥, 否则输出私钥。")
	flag.Parse()

	if *context == "" {
		flag.Usage()
		log.Fatal("错误: --context 参数是必须的。")
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
	privateKey, err := deriveAndGenerateKey(password, []byte(SALT), *context)
	if err != nil {
		log.Fatalf("错误: 密钥生成失败: %v", err)
	}

	// --- 4. 根据参数输出结果 ---
	if *isPublicKey {
		// 输出 OpenSSH 格式的公钥
		sshPubKey, err := ssh.NewPublicKey(privateKey.Public())
		if err != nil {
			log.Fatalf("错误: 无法创建 SSH 公钥: %v", err)
		}
		fmt.Print(string(ssh.MarshalAuthorizedKey(sshPubKey)))
	} else {
		// 输出 PEM 格式的私钥
		pemBlock, err := ssh.MarshalPrivateKey(privateKey, "")
		if err != nil {
			log.Fatalf("错误: 无法序列化私钥: %v", err)
		}
		if err := pem.Encode(os.Stdout, pemBlock); err != nil {
			log.Fatalf("错误: 无法将私钥编码为 PEM 格式: %v", err)
		}
	}
}

// deriveAndGenerateKey 是核心逻辑函数
func deriveAndGenerateKey(password, salt []byte, context string) (ed25519.PrivateKey, error) {
	// --- 核心步骤 A: 密钥延伸 (Key Stretching) ---
	// 使用 Argon2id 对用户输入的密码进行"慢哈希"，生成一个高强度的 32 字节主种子。
	// 这使得对主密码的离线暴力破解变得极其昂贵。
	// Argon2id 的参数可以调整，值越大越安全，但生成速度越慢。
	masterSeed := argon2.IDKey(password, salt, 1, 64*1024, 4, 32)

	// --- 核心步骤 B: 密钥衍生 (Key Derivation) ---
	// 使用 HKDF 从主种子和上下文中衍生出最终的、用于生成 SSH 密钥的种子。
	// 使用 SHA256 作为哈希函数。
	hkdfReader := hkdf.New(sha256.New, masterSeed, salt, []byte(context))
	
	finalSeed := make([]byte, ed25519.SeedSize) // Ed25519 需要 32 字节的种子
	if _, err := io.ReadFull(hkdfReader, finalSeed); err != nil {
		return nil, fmt.Errorf("无法从 HKDF 读取最终种子: %w", err)
	}

	// --- 核心步骤 C: 密钥生成 (Key Generation) ---
	// 使用最终的确定性种子生成 Ed25519 私钥。
	privateKey := ed25519.NewKeyFromSeed(finalSeed)

	return privateKey, nil
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