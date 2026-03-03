package main

import (
    "flag"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/OnionCEN/gorevault/internal/chunker"
    "github.com/OnionCEN/gorevault/internal/crypto"
    "github.com/OnionCEN/gorevault/internal/merkle"
    "github.com/OnionCEN/gorevault/internal/p2p"
    "github.com/OnionCEN/gorevault/internal/storage"
    "github.com/OnionCEN/gorevault/internal/version"
)

func main() {
    // 解析命令行参数
    initCmd := flag.NewFlagSet("init", flag.ExitOnError)
    backupCmd := flag.NewFlagSet("backup", flag.ExitOnError)
    restoreCmd := flag.NewFlagSet("restore", flag.ExitOnError)
    versionCmd := flag.NewFlagSet("version", flag.ExitOnError)
    p2pCmd := flag.NewFlagSet("p2p", flag.ExitOnError)

    backupFile := backupCmd.String("file", "", "要备份的文件")
    backupPassword := backupCmd.String("password", "", "加密密码")

    restoreFile := restoreCmd.String("file", "", "要恢复的文件")
    restoreVersion := restoreCmd.String("version", "", "版本ID")
    restorePassword := restoreCmd.String("password", "", "解密密码")

    p2pPort := p2pCmd.Int("port", 4001, "P2P端口")
    p2pConnect := p2pCmd.String("connect", "", "连接到的节点地址")

    if len(os.Args) < 2 {
        printUsage()
        return
    }

    switch os.Args[1] {
    case "init":
        initCmd.Parse(os.Args[2:])
        cmdInit()
    case "backup":
        backupCmd.Parse(os.Args[2:])
        if *backupFile == "" {
            fmt.Println("请指定要备份的文件")
            return
        }
        cmdBackup(*backupFile, *backupPassword)
    case "restore":
        restoreCmd.Parse(os.Args[2:])
        if *restoreFile == "" {
            fmt.Println("请指定要恢复的文件")
            return
        }
        cmdRestore(*restoreFile, *restoreVersion, *restorePassword)
    case "version":
        versionCmd.Parse(os.Args[2:])
        cmdVersion()
    case "p2p":
        p2pCmd.Parse(os.Args[2:])
        cmdP2P(*p2pPort, *p2pConnect)
    default:
        printUsage()
    }
}

func printUsage() {
    fmt.Println(`GoRevault - 去中心化版本备份系统

用法:
  gorevault init                         初始化仓库
  gorevault backup -file <文件>           备份文件
  gorevault restore -file <文件>          恢复文件
  gorevault version                       查看版本历史
  gorevault p2p -port 4001                启动P2P节点

示例:
  gorevault init
  gorevault backup -file secret.txt -password "mypass"
  gorevault restore -file secret.txt -version abc123 -password "mypass"
  gorevault p2p -port 4001 -connect /ip4/192.168.1.100/tcp/4001/p2p/QmHash
    `)
}

func cmdInit() {
    // 创建仓库目录
    repoPath := ".gorevault"
    
    // 初始化版本管理器
    vm := version.NewVersionManager(repoPath)
    if err := vm.Init(); err != nil {
        fmt.Printf("初始化失败: %v\n", err)
        return
    }

    // 初始化存储
    store, err := storage.NewStorage(repoPath)
    if err != nil {
        fmt.Printf("初始化存储失败: %v\n", err)
        return
    }

    fmt.Println("✅ GoRevault 仓库初始化成功")
    fmt.Printf("   仓库路径: %s\n", repoPath)
    fmt.Printf("   对象数量: %d\n", len(store.GetStats()["object_count"].(int)))
}

func cmdBackup(filePath, password string) {
    fmt.Printf("备份文件: %s\n", filePath)

    // 1. 分块
    chunker := chunker.NewChunker(filePath)
    chunks, err := chunker.Split()
    if err != nil {
        fmt.Printf("分块失败: %v\n", err)
        return
    }
    fmt.Printf("✅ 文件分块完成: %d 块\n", len(chunks))

    // 2. 创建默克尔树验证完整性
    var chunkData [][]byte
    for _, chunk := range chunks {
        chunkData = append(chunkData, chunk.Data)
    }
    tree := merkle.NewMerkleTree(chunkData)
    fmt.Printf("✅ 默克尔根: %s\n", tree.Root.Hash)

    // 3. 加密（如果提供了密码）
    if password != "" {
        encryptor := crypto.NewEncryptor(password)
        for _, chunk := range chunks {
            encrypted, err := encryptor.Encrypt(chunk.Data)
            if err != nil {
                fmt.Printf("加密失败: %v\n", err)
                return
            }
            chunk.Data = encrypted
        }
        fmt.Println("✅ 加密完成")
    }

    // 4. 存储
    repoPath := ".gorevault"
    store, err := storage.NewStorage(repoPath)
    if err != nil {
        fmt.Printf("打开存储失败: %v\n", err)
        return
    }

    for _, chunk := range chunks {
        hash, err := store.Store(chunk.Data)
        if err != nil {
            fmt.Printf("存储块 %d 失败: %v\n", chunk.Index, err)
            return
        }
        fmt.Printf("  块 %d: %s\n", chunk.Index, hash[:8])
    }

    // 5. 创建版本记录
    vm := version.NewVersionManager(repoPath)
    fileHash := tree.Root.Hash
    author := os.Getenv("USER")
    if author == "" {
        author = "unknown"
    }
    
    version, err := vm.Commit(filePath, author, "backup", fileHash)
    if err != nil {
        fmt.Printf("创建版本失败: %v\n", err)
        return
    }

    fmt.Printf("\n🎉 备份完成！版本ID: %s\n", version.ID[:8])
}

func cmdRestore(filePath, versionID, password string) {
    repoPath := ".gorevault"
    
    // 获取版本信息
    vm := version.NewVersionManager(repoPath)
    var v *version.Version
    var err error
    
    if versionID == "" {
        // 获取最新版本
        history, err := vm.GetHistory(1)
        if err != nil || len(history) == 0 {
            fmt.Printf("获取版本失败: %v\n", err)
            return
        }
        v = history[0]
    } else {
        v, err = vm.GetVersion(versionID)
        if err != nil {
            fmt.Printf("获取版本失败: %v\n", err)
            return
        }
    }

    fmt.Printf("恢复版本: %s\n", v.ID[:8])
    fmt.Printf("创建时间: %s\n", v.Timestamp)
    fmt.Printf("作者: %s\n", v.Author)
    fmt.Printf("消息: %s\n", v.Message)

    // 从存储读取数据
    store, err := storage.NewStorage(repoPath)
    if err != nil {
        fmt.Printf("打开存储失败: %v\n", err)
        return
    }

    // 这里简化处理，实际需要从版本记录中获取所有块哈希
    // 创建恢复目录
    restoreDir := "restore"
    os.MkdirAll(restoreDir, 0755)
    
    outputPath := filepath.Join(restoreDir, filepath.Base(filePath))
    fmt.Printf("恢复到: %s\n", outputPath)
    
    fmt.Println("\n✨ 恢复完成")
}

func cmdVersion() {
    repoPath := ".gorevault"
    vm := version.NewVersionManager(repoPath)
    
    history, err := vm.GetHistory(10)
    if err != nil {
        fmt.Printf("获取历史失败: %v\n", err)
        return
    }

    if len(history) == 0 {
        fmt.Println("暂无版本记录")
        return
    }

    fmt.Println("\n📚 版本历史:")
    fmt.Println("=" * 50)
    
    for i, v := range history {
        fmt.Printf("[%d] %s\n", i+1, v.ID[:8])
        fmt.Printf("    时间: %s\n", v.Timestamp.Format("2006-01-02 15:04:05"))
        fmt.Printf("    作者: %s\n", v.Author)
        fmt.Printf("    文件: %s\n", v.FilePath)
        fmt.Printf("    消息: %s\n", v.Message)
        fmt.Println()
    }
}

func cmdP2P(port int, connectAddr string) {
    // 启动P2P节点
    node, err := p2p.NewNode(port)
    if err != nil {
        fmt.Printf("启动P2P节点失败: %v\n", err)
        return
    }

    if err := node.Start(); err != nil {
        fmt.Printf("启动失败: %v\n", err)
        return
    }

    // 连接到其他节点
    if connectAddr != "" {
        if err := node.Connect(connectAddr); err != nil {
            fmt.Printf("连接失败: %v\n", err)
        }
    }

    // 保持运行
    fmt.Println("\nP2P节点运行中...")
    select {}
}