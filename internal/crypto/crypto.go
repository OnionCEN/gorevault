package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "errors"
    "io"
)

// Encryptor 加密器
type Encryptor struct {
    key []byte
}

// NewEncryptor 创建新的加密器
func NewEncryptor(password string) *Encryptor {
    // 从密码派生密钥
    hash := sha256.Sum256([]byte(password))
    return &Encryptor{
        key: hash[:],
    }
}

// Encrypt 加密数据
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return nil, err
    }

    // 创建GCM模式
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    // 创建随机nonce
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }

    // 加密并附加nonce
    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    return ciphertext, nil
}

// Decrypt 解密数据
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    if len(ciphertext) < gcm.NonceSize() {
        return nil, errors.New("密文太短")
    }

    // 提取nonce
    nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]

    // 解密
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, err
    }

    return plaintext, nil
}

// EncryptFile 加密文件
func (e *Encryptor) EncryptFile(inputPath, outputPath string) error {
    // 读取文件
    data, err := os.ReadFile(inputPath)
    if err != nil {
        return err
    }

    // 加密
    encrypted, err := e.Encrypt(data)
    if err != nil {
        return err
    }

    // 写入文件
    return os.WriteFile(outputPath, encrypted, 0644)
}

// DecryptFile 解密文件
func (e *Encryptor) DecryptFile(inputPath, outputPath string) error {
    // 读取加密文件
    data, err := os.ReadFile(inputPath)
    if err != nil {
        return err
    }

    // 解密
    decrypted, err := e.Decrypt(data)
    if err != nil {
        return err
    }

    // 写入文件
    return os.WriteFile(outputPath, decrypted, 0644)
}

// GenerateKey 生成随机密钥
func GenerateKey() (string, error) {
    key := make([]byte, 32) // 256位
    _, err := rand.Read(key)
    if err != nil {
        return "", err
    }
    return hex.EncodeToString(key), nil
}