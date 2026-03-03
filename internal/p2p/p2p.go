package p2p

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p/core/host"
    "github.com/libp2p/go-libp2p/core/network"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/libp2p/go-libp2p/core/peerstore"
    "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

// MessageType 消息类型
type MessageType int

const (
    MsgDiscover MessageType = iota // 发现节点
    MsgChunk                        // 传输块
    MsgVersion                      // 传输版本
    MsgRequest                       // 请求数据
)

// Message P2P消息
type Message struct {
    Type    MessageType `json:"type"`
    From    string      `json:"from"`
    Payload []byte      `json:"payload"`
}

// ChunkPayload 块传输负载
type ChunkPayload struct {
    FilePath string `json:"file_path"`
    Index    int    `json:"index"`
    Hash     string `json:"hash"`
    Data     []byte `json:"data"`
}

// Node P2P节点
type Node struct {
    host     host.Host
    ctx      context.Context
    peers    map[peer.ID]bool
    mu       sync.RWMutex
    handlers map[MessageType]func(Message, network.Stream)
}

// NewNode 创建新节点
func NewNode(listenPort int) (*Node, error) {
    ctx := context.Background()

    // 创建host
    h, err := libp2p.New(
        libp2p.ListenAddrStrings(
            fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort),
        ),
    )
    if err != nil {
        return nil, err
    }

    node := &Node{
        host:     h,
        ctx:      ctx,
        peers:    make(map[peer.ID]bool),
        handlers: make(map[MessageType]func(Message, network.Stream)),
    }

    // 设置流处理器
    h.SetStreamHandler("/gorevault/1.0.0", node.handleStream)

    return node, nil
}

// Start 启动节点
func (n *Node) Start() error {
    // 启动mdns发现
    service := &mdnsService{
        node:       n,
        serviceTag: "gorevault",
    }
    
    discoverer := mdns.NewMdnsService(n.host, "gorevault", service)
    if err := discoverer.Start(); err != nil {
        return err
    }

    fmt.Printf("✅ P2P节点启动: %s\n", n.host.ID().String())
    fmt.Printf("   地址: %s\n", n.host.Addrs())

    return nil
}

// Connect 连接到其他节点
func (n *Node) Connect(addr string) error {
    // 解析地址
    peerinfo, err := peer.AddrInfoFromString(addr)
    if err != nil {
        return err
    }

    // 添加到peerstore
    n.host.Peerstore().AddAddrs(peerinfo.ID, peerinfo.Addrs, peerstore.PermanentAddrTTL)

    // 连接
    if err := n.host.Connect(n.ctx, *peerinfo); err != nil {
        return err
    }

    n.mu.Lock()
    n.peers[peerinfo.ID] = true
    n.mu.Unlock()

    fmt.Printf("✅ 连接到节点: %s\n", peerinfo.ID.String())
    return nil
}

// BroadcastChunk 广播块到所有节点
func (n *Node) BroadcastChunk(chunk *chunker.Chunk) error {
    payload := ChunkPayload{
        FilePath: chunk.FilePath,
        Index:    chunk.Index,
        Hash:     chunk.Hash,
        Data:     chunk.Data,
    }

    data, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    msg := Message{
        Type:    MsgChunk,
        From:    n.host.ID().String(),
        Payload: data,
    }

    // 广播给所有peers
    n.mu.RLock()
    defer n.mu.RUnlock()

    for peerID := range n.peers {
        go n.sendMessage(peerID, msg)
    }

    return nil
}

// sendMessage 发送消息到指定节点
func (n *Node) sendMessage(peerID peer.ID, msg Message) error {
    // 打开流
    stream, err := n.host.NewStream(n.ctx, peerID, "/gorevault/1.0.0")
    if err != nil {
        return err
    }
    defer stream.Close()

    // 编码并发送
    encoder := json.NewEncoder(stream)
    if err := encoder.Encode(msg); err != nil {
        return err
    }

    return nil
}

// handleStream 处理传入流
func (n *Node) handleStream(stream network.Stream) {
    defer stream.Close()

    // 解码消息
    var msg Message
    decoder := json.NewDecoder(stream)
    if err := decoder.Decode(&msg); err != nil {
        fmt.Printf("解码消息失败: %v\n", err)
        return
    }

    // 查找处理器
    if handler, ok := n.handlers[msg.Type]; ok {
        handler(msg, stream)
    }
}

// RegisterHandler 注册消息处理器
func (n *Node) RegisterHandler(msgType MessageType, handler func(Message, network.Stream)) {
    n.handlers[msgType] = handler
}

// mdns服务
type mdnsService struct {
    node       *Node
    serviceTag string
}

func (s *mdnsService) HandlePeerFound(info peer.AddrInfo) {
    if info.ID == s.node.host.ID() {
        return // 忽略自己
    }

    s.node.mu.Lock()
    defer s.node.mu.Unlock()

    if _, ok := s.node.peers[info.ID]; !ok {
        s.node.peers[info.ID] = true
        fmt.Printf("🔍 发现新节点: %s\n", info.ID.String())
    }
}