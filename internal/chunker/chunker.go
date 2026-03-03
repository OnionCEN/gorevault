package chunker

import (
    "crypto/sha256"
    "encoding/hex"
    "io"
    "os"
)

const (
    // ChunkSize 每个块的大小 (1MB)
    ChunkSize = 1024 * 1024
)

// Chunk 文件块
type Chunk struct {
    Index    int
    Hash     string
    Data     []byte
    FilePath string
}

// Chunker 分块器
type Chunker struct {
    filePath string
    chunks   []*Chunk
}

// NewChunker 创建新的分块器
func NewChunker(filePath string) *Chunker {
    return &Chunker{
        filePath: filePath,
        chunks:   make([]*Chunk, 0),
    }
}

// Split 分割文件
func (c *Chunker) Split() ([]*Chunk, error) {
    file, err := os.Open(c.filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    buffer := make([]byte, ChunkSize)
    index := 0

    for {
        n, err := file.Read(buffer)
        if err != nil && err != io.EOF {
            return nil, err
        }
        if n == 0 {
            break
        }

        // 复制实际读取的数据
        data := make([]byte, n)
        copy(data, buffer[:n])

        // 计算哈希
        hash := sha256.Sum256(data)
        chunk := &Chunk{
            Index:    index,
            Hash:     hex.EncodeToString(hash[:]),
            Data:     data,
            FilePath: c.filePath,
        }

        c.chunks = append(c.chunks, chunk)
        index++

        if err == io.EOF {
            break
        }
    }

    return c.chunks, nil
}

// Merge 合并文件块
func (c *Chunker) Merge(chunks []*Chunk, outputPath string) error {
    file, err := os.Create(outputPath)
    if err != nil {
        return err
    }
    defer file.Close()

    for i := 0; i < len(chunks); i++ {
        // 按索引顺序写入
        for _, chunk := range chunks {
            if chunk.Index == i {
                _, err := file.Write(chunk.Data)
                if err != nil {
                    return err
                }
                break
            }
        }
    }

    return nil
}

// VerifyChunk 验证块完整性
func VerifyChunk(chunk *Chunk) bool {
    hash := sha256.Sum256(chunk.Data)
    return hex.EncodeToString(hash[:]) == chunk.Hash
}