package merkle

import (
    "crypto/sha256"
    "encoding/hex"
    "hash"
)

// MerkleTree 默克尔树结构
type MerkleTree struct {
    Root     *Node
    Leaves   []*Node
    hashFunc hash.Hash
}

// Node 树节点
type Node struct {
    Hash  string
    Left  *Node
    Right *Node
    Data  []byte // 只在叶子节点有数据
}

// NewMerkleTree 创建新的默克尔树
func NewMerkleTree(data [][]byte) *MerkleTree {
    if len(data) == 0 {
        return nil
    }

    tree := &MerkleTree{
        hashFunc: sha256.New(),
    }

    // 创建叶子节点
    var leaves []*Node
    for _, d := range data {
        hash := tree.calculateHash(d)
        leaves = append(leaves, &Node{
            Hash: hash,
            Data: d,
        })
    }
    tree.Leaves = leaves
    tree.Root = tree.buildTree(leaves)
    
    return tree
}

// buildTree 递归构建树
func (t *MerkleTree) buildTree(nodes []*Node) *Node {
    if len(nodes) == 1 {
        return nodes[0]
    }

    var parents []*Node
    for i := 0; i < len(nodes); i += 2 {
        if i+1 < len(nodes) {
            parent := t.combineNodes(nodes[i], nodes[i+1])
            parents = append(parents, parent)
        } else {
            // 奇数节点，复制一份
            parent := t.combineNodes(nodes[i], nodes[i])
            parents = append(parents, parent)
        }
    }
    
    return t.buildTree(parents)
}

// combineNodes 合并两个节点
func (t *MerkleTree) combineNodes(left, right *Node) *Node {
    combined := append([]byte(left.Hash), []byte(right.Hash)...)
    hash := t.calculateHash(combined)
    
    return &Node{
        Hash:  hash,
        Left:  left,
        Right: right,
    }
}

// calculateHash 计算哈希
func (t *MerkleTree) calculateHash(data []byte) string {
    t.hashFunc.Reset()
    t.hashFunc.Write(data)
    return hex.EncodeToString(t.hashFunc.Sum(nil))
}

// Verify 验证数据是否在树中
func (t *MerkleTree) Verify(data []byte) bool {
    hash := t.calculateHash(data)
    
    // 查找叶子节点
    var target *Node
    for _, leaf := range t.Leaves {
        if leaf.Hash == hash {
            target = leaf
            break }
    }
    
    if target == nil {
        return false
    }
    
    // 验证路径
    return t.verifyPath(target, t.Root)
}

// verifyPath 递归验证路径
func (t *MerkleTree) verifyPath(node, root *Node) bool {
    if node == root {
        return true
    }
    
    // 向上查找父节点
    return t.findParent(node, root)
}

// findParent 查找父节点
func (t *MerkleTree) findParent(node, current *Node) bool {
    if current.Left == nil && current.Right == nil {
        return false
    }
    
    if current.Left == node || current.Right == node {
        // 验证当前节点的哈希是否正确
        if current.Left != nil && current.Right != nil {
            combined := append([]byte(current.Left.Hash), []byte(current.Right.Hash)...)
            expected := t.calculateHash(combined)
            return expected == current.Hash
        }
        return true
    }
    
    // 继续向下查找
    found := false
    if current.Left != nil {
        found = t.findParent(node, current.Left)
    }
    if !found && current.Right != nil {
        found = t.findParent(node, current.Right)
    }
    return found
}