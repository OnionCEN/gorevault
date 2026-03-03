package storage

import (
    "encoding/json"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "sync"
)

// Object 存储对象
type Object struct {
    Hash     string `json:"hash"`
    Size     int64  `json:"size"`
    Path     string `json:"path"`
    RefCount int    `json:"ref_count"` // 引用计数
}

// Storage 存储管理器
type Storage struct {
    rootPath string
    objects  map[string]*Object
    mu       sync.RWMutex
}

// NewStorage 创建新的存储
func NewStorage(rootPath string) (*Storage, error) {
    s := &Storage{
        rootPath: rootPath,
        objects:  make(map[string]*Object),
    }

    // 创建目录
    dirs := []string{
        filepath.Join(rootPath, "objects"),
        filepath.Join(rootPath, "tmp"),
        filepath.Join(rootPath, "index"),
    }

    for _, dir := range dirs {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return nil, err
        }
    }

    // 加载已有对象
    if err := s.loadIndex(); err != nil {
        return nil, err
    }

    return s, nil
}

// Store 存储数据
func (s *Storage) Store(data []byte) (string, error) {
    // 计算哈希
    hash := fmt.Sprintf("%x", sha256.Sum256(data))
    
    s.mu.Lock()
    defer s.mu.Unlock()

    // 如果已存在，增加引用计数
    if obj, exists := s.objects[hash]; exists {
        obj.RefCount++
        s.saveIndex()
        return hash, nil
    }

    // 存储文件
    objPath := filepath.Join(s.rootPath, "objects", hash[:2], hash)
    if err := os.MkdirAll(filepath.Dir(objPath), 0755); err != nil {
        return "", err
    }

    if err := os.WriteFile(objPath, data, 0644); err != nil {
        return "", err
    }

    // 创建对象记录
    obj := &Object{
        Hash:     hash,
        Size:     int64(len(data)),
        Path:     objPath,
        RefCount: 1,
    }

    s.objects[hash] = obj
    s.saveIndex()

    return hash, nil
}

// Read 读取数据
func (s *Storage) Read(hash string) ([]byte, error) {
    s.mu.RLock()
    obj, exists := s.objects[hash]
    s.mu.RUnlock()

    if !exists {
        return nil, fmt.Errorf("对象不存在: %s", hash)
    }

    return os.ReadFile(obj.Path)
}

// Delete 删除对象
func (s *Storage) Delete(hash string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    obj, exists := s.objects[hash]
    if !exists {
        return nil
    }

    obj.RefCount--
    if obj.RefCount <= 0 {
        // 真正删除文件
        if err := os.Remove(obj.Path); err != nil {
            return err
        }
        delete(s.objects, hash)
        
        // 尝试删除空目录
        os.Remove(filepath.Dir(obj.Path))
    }

    return s.saveIndex()
}

// GetStats 获取存储统计
func (s *Storage) GetStats() map[string]interface{} {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var totalSize int64
    for _, obj := range s.objects {
        totalSize += obj.Size
    }

    return map[string]interface{}{
        "object_count": len(s.objects),
        "total_size":   totalSize,
        "root_path":    s.rootPath,
    }
}

// GC 垃圾回收
func (s *Storage) GC() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    // 遍历所有对象文件
    objectsDir := filepath.Join(s.rootPath, "objects")
    
    err := filepath.Walk(objectsDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() {
            return nil
        }

        // 检查是否在索引中
        hash := filepath.Base(path)
        if _, exists := s.objects[hash]; !exists {
            // 不在索引中，删除
            fmt.Printf("删除孤立对象: %s\n", path)
            return os.Remove(path)
        }
        
        return nil
    })

    return err
}

// loadIndex 加载索引
func (s *Storage) loadIndex() error {
    indexFile := filepath.Join(s.rootPath, "index", "objects.json")
    data, err := os.ReadFile(indexFile)
    if err != nil {
        if os.IsNotExist(err) {
            return nil
        }
        return err
    }

    var objects map[string]*Object
    if err := json.Unmarshal(data, &objects); err != nil {
        return err
    }

    s.objects = objects
    return nil
}

// saveIndex 保存索引
func (s *Storage) saveIndex() error {
    data, err := json.MarshalIndent(s.objects, "", "  ")
    if err != nil {
        return err
    }

    indexFile := filepath.Join(s.rootPath, "index", "objects.json")
    return os.WriteFile(indexFile, data, 0644)
}

// Compact 压缩存储（合并小对象）
func (s *Storage) Compact(threshold int) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    var smallObjects []*Object
    for _, obj := range s.objects {
        if obj.Size < int64(threshold) {
            smallObjects = append(smallObjects, obj)
        }
    }

    if len(smallObjects) < 2 {
        return nil
    }

    // 合并小对象
    var combined []byte
    for _, obj := range smallObjects {
        data, err := os.ReadFile(obj.Path)
        if err != nil {
            return err
        }
        combined = append(combined, data...)
    }

    // 存储合并后的对象
    combinedHash := fmt.Sprintf("%x", sha256.Sum256(combined))
    combinedPath := filepath.Join(s.rootPath, "objects", combinedHash[:2], combinedHash)
    
    if err := os.MkdirAll(filepath.Dir(combinedPath), 0755); err != nil {
        return err
    }
    
    if err := os.WriteFile(combinedPath, combined, 0644); err != nil {
        return err
    }

    // 创建索引记录
    combinedObj := &Object{
        Hash:     combinedHash,
        Size:     int64(len(combined)),
        Path:     combinedPath,
        RefCount: 1,
    }

    // 删除小对象
    for _, obj := range smallObjects {
        if err := os.Remove(obj.Path); err != nil {
            return err
        }
        delete(s.objects, obj.Hash)
    }

    s.objects[combinedHash] = combinedObj
    return s.saveIndex()
}