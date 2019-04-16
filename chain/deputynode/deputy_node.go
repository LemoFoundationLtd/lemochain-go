package deputynode

import (
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto/sha3"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/merkle"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"math/big"
)

//go:generate gencodec -type DeputyNode --field-override deputyNodeMarshaling -out gen_deputy_node_json.go

var (
	ErrMinerAddressInvalid = errors.New("incorrect field: 'MinerAddress'")
	ErrNodeIDInvalid       = errors.New("incorrect field: 'NodeID'")
	ErrPortInvalid         = errors.New("max deputy node's port is 65535")
	ErrRankInvalid         = errors.New("max deputy node's rank is 65535")
	ErrVotesInvalid        = errors.New("min deputy node's votes are 0")
)

// DeputyNode
type DeputyNode struct {
	MinerAddress common.Address `json:"minerAddress"   gencodec:"required"`
	NodeID       []byte         `json:"nodeID"         gencodec:"required"`
	Rank         uint32         `json:"rank"           gencodec:"required"` // 排名 从0开始
	Votes        *big.Int       `json:"votes"          gencodec:"required"` // 得票数
}

type deputyNodeMarshaling struct {
	NodeID hexutil.Bytes
	Rank   hexutil.Uint32
	Votes  *hexutil.Big10
}

func (d *DeputyNode) Hash() (h common.Hash) {
	data := []interface{}{
		d.MinerAddress,
		d.NodeID,
		d.Rank,
		d.Votes,
	}
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, data)
	hw.Sum(h[:0])
	return h
}

func (d *DeputyNode) Check() error {
	if d.MinerAddress == (common.Address{}) {
		log.Errorf("incorrect field: 'MinerAddress'. value: %s", d.MinerAddress.String())
		return ErrMinerAddressInvalid
	}
	if len(d.NodeID) != 64 {
		log.Errorf("incorrect field: 'NodeID'. value: %s", common.ToHex(d.NodeID))
		return ErrNodeIDInvalid
	}
	if d.Rank > 65535 {
		log.Errorf("incorrect field: 'rank'. value: %d", d.Rank)
		return ErrRankInvalid
	}
	if d.Votes.Cmp(new(big.Int)) < 0 {
		log.Errorf("incorrect field: 'votes'. value: %d", d.Votes)
		return ErrVotesInvalid
	}
	return nil
}

func (d *DeputyNode) Copy() *DeputyNode {
	result := &DeputyNode{
		MinerAddress: d.MinerAddress,
		NodeID:       d.NodeID,
		Rank:         d.Rank,
		Votes:        new(big.Int).Set(d.Votes),
	}

	return result
}

// GetIncomeAddress
func (d *DeputyNode) GetIncomeAddress(am *account.Manager) common.Address {
	minerAcc := am.GetAccount(d.MinerAddress)
	profile := minerAcc.GetCandidate()
	strIncomeAddress, ok := profile[types.CandidateKeyIncomeAddress]
	if !ok {
		log.Errorf("not exist income address; miner address = %s", d.MinerAddress)
		return common.Address{}
	}
	incomeAddress, err := common.StringToAddress(strIncomeAddress)
	if err != nil {
		log.Errorf("income address unavailability; incomeAddress = %s", strIncomeAddress)
		return common.Address{}
	}
	return incomeAddress
}

type DeputyNodes []*DeputyNode

func (nodes DeputyNodes) String() string {
	if buf, err := json.Marshal(nodes); err == nil {
		return string(buf)
	}
	return ""
}

// MerkleRootSha compute the root hash of deputy nodes merkle trie
func (nodes DeputyNodes) MerkleRootSha() common.Hash {
	leaves := make([]common.Hash, len(nodes))
	for i, item := range nodes {
		leaves[i] = item.Hash()
	}
	return merkle.New(leaves).Root()
}
