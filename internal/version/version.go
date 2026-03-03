package version

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"
)

// Version 版本信息
type Version struct {
    ID        string    `json:"id"`        // 版本ID (哈希)
    Timestamp time.Time `json:"timestamp"` // 创建时间
    Author    string    `json:"author"`    // 作者
    Message   string    `json:"message"`   // 提交信息
    Parent    string    `json:"parent"`    // 父版本ID
    FileHash  string    `json:"file_hash"` // 文件哈希
    FilePath  string    `json:"file_path"` // 文件路径
    Size      int64     `json:"size"`      // 文件大小
}

// VersionManager 版本管理器
type VersionManager struct {
    repoPath string
    versions map[string]*Version
    current  string
}

// NewVersionManager 创建版本管理器
func NewVersionManager(repoPath string) *VersionManager {
    return &VersionManager{
        repoPath: repoPath,
        versions: make(map[string]*Version),
    }
}

// Init 初始化版本库
func (vm *VersionManager) Init() error {
    // 创建目录结构
    dirs := []string{
        filepath.Join(vm.repoPath, "objects"), // 存储对象
        filepath.Join(vm.repoPath, "versions"), // 存储版本信息
        filepath.Join(vm.repoPath, "refs"),     // 存储引用
    }

    for _, dir := range dirs {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return err
        }
    }

    return nil
}

// Commit 提交新版本
func (vm *VersionManager) Commit(filePath, author, message, fileHash string) (*Version, error) {
    // 获取文件信息
    fileInfo, err := os.Stat(filePath)
    if err != nil {
        return nil, err
    }

    // 创建版本
    version := &Version{
        ID:        generateVersionID(fileHash, author, message),
        Timestamp: time.Now(),
        Author:    author,
        Message:   message,
        Parent:    vm.current,
        FileHash:  fileHash,
        FilePath:  filePath,
        Size:      fileInfo.Size(),
    }

    // 保存版本
    if err := vm.saveVersion(version); err != nil {
        return nil, err
    }

    vm.versions[version.ID] = version
    vm.current = version.ID

    // 更新当前引用
    vm.updateRef("HEAD", version.ID)

    return version, nil
}

// GetVersion 获取版本信息
func (vm *VersionManager) GetVersion(versionID string) (*Version, error) {
    // 先从缓存找
    if v, ok := vm.versions[versionID]; ok {
        return v, nil
    }

    // 从文件加载
    return vm.loadVersion(versionID)
}

// GetHistory 获取版本历史
func (vm *VersionManager) GetHistory(limit int) ([]*Version, error) {
    var history []*Version
    current := vm.current

    for current != "" && len(history) < limit {
        version, err := vm.GetVersion(current)
        if err != nil {
            return nil, err
        }
        history = append(history, version)
        current = version.Parent
    }

    return history, nil
}

// Diff 比较两个版本
func (vm *VersionManager) Diff(version1, version2 string) ([]string, error) {
    v1, err := vm.GetVersion(version1)
    if err != nil {
        return nil, err
    }

    v2, err := vm.GetVersion(version2)
    if err != nil {
        return nil, err
    }

    var diff []string
    diff = append(diff, fmt.Sprintf("版本 %s -> %s", version1[:8], version2[:8]))
    diff = append(diff, fmt.Sprintf("作者: %s -> %s", v1.Author, v2.Author))
    diff = append(diff, fmt.Sprintf("时间: %s -> %s", v1.Timestamp, v2.Timestamp))
    diff = append(diff, fmt.Sprintf("消息: %s -> %s", v1.Message, v2.Message))
    diff = append(diff, fmt.Sprintf("文件大小: %d -> %d 字节", v1.Size, v2.Size))

    if v1.FileHash != v2.FileHash {
        diff = append(diff, "⚠️ 文件内容已修改")
    } else {
        diff = append(diff, "✅ 文件内容未变")
    }

    return diff, nil
}

// saveVersion 保存版本到文件
func (vm *VersionManager) saveVersion(v *Version) error {
    data, err := json.MarshalIndent(v, "", "  ")
    if err != nil {
        return err
    }

    path := filepath.Join(vm.repoPath, "versions", v.ID+".json")
    return os.WriteFile(path, data, 0644)
}

// loadVersion 从文件加载版本
func (vm *VersionManager) loadVersion(versionID string) (*Version, error) {
    path := filepath.Join(vm.repoPath, "versions", versionID+".json")
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var v Version
    if err := json.Unmarshal(data, &v); err != nil {
        return nil, err
    }

    vm.versions[versionID] = &v
    return &v, nil
}

// updateRef 更新引用
func (vm *VersionManager) updateRef(refName, versionID string) error {
    refPath := filepath.Join(vm.repoPath, "refs", refName)
    return os.WriteFile(refPath, []byte(versionID), 0644)
}

// generateVersionID 生成版本ID
func generateVersionID(fileHash, author, message string) string {
    data := fmt.Sprintf("%s:%s:%s:%d", fileHash, author, message, time.Now().UnixNano())
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}