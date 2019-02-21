package deputynode

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto/sha3"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/rlp"
	"math/big"
	"net"
	"strconv"
	"strings"
	"sync"
)

const (
	TotalCount = 5
)

var (
	selfNodeKey *ecdsa.PrivateKey
)

func GetSelfNodeKey() *ecdsa.PrivateKey {
	return selfNodeKey
}

func GetSelfNodeID() []byte {
	return (crypto.FromECDSAPub(&selfNodeKey.PublicKey))[1:]
}

func SetSelfNodeKey(key *ecdsa.PrivateKey) {
	selfNodeKey = key
}

//go:generate gencodec -type DeputyNode --field-override deputyNodeMarshaling -out gen_deputy_node_json.go
//go:generate gencodec -type CandiDeputyNodeCoutdateNode --field-override candidateNodeMarshaling -out gen_candidate_node_json.go

// CandidateNode
type CandidateNode struct {
	IsCandidate  bool           `json:"isCandidate"      gencodec:"required"`
	MinerAddress common.Address `json:"minerAddress"        gencodec:"required"` // 候选节点挖矿收益地址
	NodeID       []byte         `json:"nodeID"         gencodec:"required"`
	Host         string         `json:"host"             gencodec:"required"` // ip或者域名
	Port         uint32         `json:"port"           gencodec:"required"`   // 端口
}

type candidateNodeMarshaling struct {
	NodeID hexutil.Bytes
	Port   hexutil.Uint32
}

// DeputyNode
type DeputyNode struct {
	MinerAddress common.Address `json:"minerAddress"   gencodec:"required"`
	NodeID       []byte         `json:"nodeID"         gencodec:"required"`
	IP           net.IP         `json:"ip"             gencodec:"required"` // ip
	Port         uint32         `json:"port"           gencodec:"required"` // 端口
	Rank         uint32         `json:"rank"           gencodec:"required"` // 排名 从0开始
	Votes        *big.Int       `json:"votes"          gencodec:"required"` // 得票数
}

func (d *DeputyNode) Hash() (h common.Hash) {
	data := []interface{}{
		d.MinerAddress,
		d.NodeID,
		d.IP,
		d.Port,
		d.Rank,
		d.Votes,
	}
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, data)
	hw.Sum(h[:0])
	return h
}

func (d *DeputyNode) Check() error {
	if len(d.NodeID) != 64 {
		log.Errorf("incorrect field: 'NodeID'.value: %s", common.ToHex(d.NodeID))
		return errors.New("incorrect field: 'NodeID'")
	}
	if d.Port > 65535 {
		log.Errorf("incorrect field: 'port'. value: %d", d.Port)
		return errors.New("max deputy node's port is 65535")
	}
	return nil
}

type DeputyNodes []*DeputyNode

type deputyNodeMarshaling struct {
	NodeID hexutil.Bytes
	IP     hexutil.IP
	Port   hexutil.Uint32
	Rank   hexutil.Uint32
	Votes  *hexutil.Big10
}

func (nodes DeputyNodes) String() string {
	if buf, err := json.Marshal(nodes); err == nil {
		return string(buf)
	}
	return ""
}

type DeputyNodesRecord struct {
	height uint32 // 0, 100W+1K, 200W+1K, 300W+1K, 400W+1K...
	nodes  DeputyNodes
}

// Manager 代理节点管理器
type Manager struct {
	DeputyNodesList []*DeputyNodesRecord // key：节点列表生效开始高度 value：节点列表
	lock            sync.Mutex
}

// Add 投票结束 统计结果通过add函数缓存起来
// height: next term start height. if snapshot:1000, period:100, result: 1101 2101 3101...
func (d *Manager) Add(height uint32, nodes DeputyNodes) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.DeputyNodesList = append(d.DeputyNodesList, &DeputyNodesRecord{height: height, nodes: nodes})
	log.Info("deputy node.add")
}

var deputyNodeManger *Manager
var once sync.Once

func Instance() *Manager {
	once.Do(func() {
		deputyNodeManger = &Manager{
			DeputyNodesList: make([]*DeputyNodesRecord, 0, 1),
		}
	})
	return deputyNodeManger
}

// GetDeputiesByHeight 通过height获取对应的节点列表
func (d *Manager) GetDeputiesByHeight(height uint32, total bool) DeputyNodes {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.DeputyNodesList == nil || len(d.DeputyNodesList) == 0 {
		panic("not set deputy nodes")
	} else if len(d.DeputyNodesList) == 1 {
		if !total && len(d.DeputyNodesList[0].nodes) > TotalCount {
			return d.DeputyNodesList[0].nodes[:TotalCount]
		}
		return d.DeputyNodesList[0].nodes
	}
	var nodes DeputyNodes
	for i := 0; i < len(d.DeputyNodesList)-1; i++ {
		if d.DeputyNodesList[i].height <= height && d.DeputyNodesList[i+1].height > height {
			if !total && len(d.DeputyNodesList[i].nodes) > TotalCount {
				nodes = d.DeputyNodesList[i].nodes[:TotalCount]
			} else {
				nodes = d.DeputyNodesList[i].nodes
			}
			break
		}
	}
	if nodes == nil {
		if !total && len(d.DeputyNodesList[len(d.DeputyNodesList)-1].nodes) > TotalCount {
			nodes = d.DeputyNodesList[len(d.DeputyNodesList)-1].nodes[:TotalCount]
		} else {
			nodes = d.DeputyNodesList[len(d.DeputyNodesList)-1].nodes
		}
	}
	return nodes
}

// getDeputyNodeCount 获取共识节点数量
func (d *Manager) GetDeputiesCount(height uint32) int {
	nodes := d.GetDeputiesByHeight(height, true)
	if len(nodes) < TotalCount {
		return len(nodes)
	}
	return TotalCount
}

// getNodeByAddress 获取address对应的节点
func (d *Manager) GetDeputyByAddress(height uint32, addr common.Address) *DeputyNode {
	nodes := d.GetDeputiesByHeight(height, false)
	if nodes == nil || len(nodes) == 0 {
		log.Warnf("GetDeputyByAddress: can't get deputy node, height: %d, addr: %s", height, addr.String())
		return nil
	}
	for _, node := range nodes {
		if node.MinerAddress == addr {
			return node
		}
	}
	return nil
}

// getNodeByNodeID 根据nodeid获取对应的节点
func (d *Manager) GetDeputyByNodeID(height uint32, nodeID []byte) *DeputyNode {
	nodes := d.GetDeputiesByHeight(height, false)
	for _, node := range nodes {
		if bytes.Compare(node.NodeID, nodeID) == 0 {
			return node
		}
	}
	return nil
}

// 获取最新块的出块者序号与本节点序号差
func (d *Manager) GetSlot(height uint32, firstAddress, nextAddress common.Address) int {
	firstNode := d.GetDeputyByAddress(height, firstAddress)
	nextNode := d.GetDeputyByAddress(height, nextAddress)
	// for test
	if height%params.SnapshotBlock > params.PeriodBlock && height%params.SnapshotBlock < params.PeriodBlock+20 {
		log.Debugf("GetSlot: height:%d. first: %s, next: %s", height, firstAddress.String(), nextAddress.String())
	}
	if ((height == 1) || (height > params.SnapshotBlock && height%params.SnapshotBlock == params.PeriodBlock+1)) && nextNode != nil {
		log.Debugf("GetSlot: change term. rank: %d", nextNode.Rank)
		return int(nextNode.Rank + 1)
	}
	if firstNode == nil || nextNode == nil {
		return -1
	}
	nodeCount := d.GetDeputiesCount(height)
	// 只有一个主节点
	if nodeCount == 1 {
		log.Debug("getSlot: only one star node")
		return 1
	}
	return (int(nextNode.Rank) - int(firstNode.Rank) + nodeCount) % nodeCount
}

func (d *Manager) GetNodeRankByNodeID(height uint32, nodeID []byte) int {
	if nextNode := d.GetDeputyByNodeID(height, nodeID); nextNode != nil {
		return int(nextNode.Rank)
	}
	return -1
}

func (d *Manager) GetNodeRankByAddress(height uint32, addr common.Address) int {
	if nextNode := d.GetDeputyByAddress(height, addr); nextNode != nil {
		return int(nextNode.Rank)
	}
	return -1
}

// TimeToHandOutRewards 是否该发出块奖励了
func (d *Manager) TimeToHandOutRewards(height uint32) bool {
	for i := 1; i < len(d.DeputyNodesList); i++ {
		if d.DeputyNodesList[i].height+params.PeriodBlock == height {
			return true
		}
	}
	return false
}

// IsSelfDeputyNode
func (d *Manager) IsSelfDeputyNode(height uint32) bool {
	node := d.GetDeputyByNodeID(height, GetSelfNodeID())
	return node != nil
}

// IsNodeDeputy
func (d *Manager) IsNodeDeputy(height uint32, nodeID []byte) bool {
	node := d.GetDeputyByNodeID(height, nodeID)
	return node != nil
}

func (d *Manager) Clear() {
	d.DeputyNodesList = make([]*DeputyNodesRecord, 0, 1)
}

func (d *Manager) GetLatestDeputies() []string {
	res := make([]string, 0)
	for _, n := range d.DeputyNodesList[len(d.DeputyNodesList)-1].nodes {
		builder := &strings.Builder{}
		builder.WriteString(common.ToHex(n.NodeID)[2:])
		builder.WriteString("@")
		builder.WriteString(n.IP.String())
		builder.WriteString(":")
		builder.WriteString(strconv.Itoa(int(n.Port)))
		res = append(res, builder.String())
	}
	return res
}
