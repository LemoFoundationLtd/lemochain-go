package types

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto/sha3"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/merkle"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"io"
	"strings"
	"sync/atomic"
)

//go:generate gencodec -type Header -field-override headerMarshaling -out gen_header_json.go
//go:generate gencodec -type Block -out gen_block_json.go

type Header struct {
	ParentHash   common.Hash    `json:"parentHash"       gencodec:"required"`
	MinerAddress common.Address `json:"miner"            gencodec:"required"`
	VersionRoot  common.Hash    `json:"versionRoot"      gencodec:"required"`
	TxRoot       common.Hash    `json:"transactionRoot"  gencodec:"required"`
	LogRoot      common.Hash    `json:"changeLogRoot"    gencodec:"required"`
	Height       uint32         `json:"height"           gencodec:"required"`
	GasLimit     uint64         `json:"gasLimit"         gencodec:"required"`
	GasUsed      uint64         `json:"gasUsed"          gencodec:"required"`
	Time         uint32         `json:"timestamp"        gencodec:"required"` // seconds
	SignData     []byte         `json:"signData"         gencodec:"required"`
	DeputyRoot   []byte         `json:"deputyRoot"`
	Extra        string         `json:"extraData"` // max length is 256 bytes
	signerNodeID atomic.Value   // it is a cache
}

type headerMarshaling struct {
	Height     hexutil.Uint32
	GasLimit   hexutil.Uint64
	GasUsed    hexutil.Uint64
	Time       hexutil.Uint32
	SignData   hexutil.Bytes
	DeputyRoot hexutil.Bytes
	Hash       common.Hash `json:"hash"`
}

// 签名信息
type SignData [65]byte

func (sd SignData) RecoverNodeID(hash common.Hash) ([]byte, error) {
	pubKey, err := crypto.Ecrecover(hash[:], sd[:])
	if err != nil {
		log.Warn("recover fail", "err", err)
		return nil, ErrInvalidSig
	}
	return pubKey[1:], nil
}

func (sd SignData) MarshalText() ([]byte, error) {
	str := common.ToHex(sd[:])
	return []byte(str), nil
}

func (sd SignData) String() string {
	return common.ToHex(sd[:])
}

func BytesToSignData(bytes []byte) SignData {
	sd := SignData{}
	copy(sd[:], bytes)
	return sd
}

// Block
type Block struct {
	Header      *Header        `json:"header"        gencodec:"required"`
	Txs         Transactions   `json:"transactions"  gencodec:"required"`
	ChangeLogs  ChangeLogSlice `json:"changeLogs"    gencodec:"required"`
	Confirms    []SignData     `json:"confirms"` // no miner's signature inside
	DeputyNodes DeputyNodes    `json:"deputyNodes"`
}

func NewBlock(header *Header, txs []*Transaction, changeLog []*ChangeLog) *Block {
	return &Block{
		Header:     header,
		Txs:        txs,
		ChangeLogs: changeLog,
	}
}

type Blocks []*Block

// Hash 块hash 排除 SignData字段
func (h *Header) Hash() common.Hash {
	return rlpHash([]interface{}{
		h.ParentHash,
		h.MinerAddress,
		h.VersionRoot,
		h.TxRoot,
		h.LogRoot,
		h.Height,
		h.GasLimit,
		h.GasUsed,
		h.Time,
		h.DeputyRoot,
		h.Extra,
	})
}

func (h *Header) SignerNodeID() ([]byte, error) {
	if cache := h.signerNodeID.Load(); cache != nil {
		return cache.([]byte), nil
	}

	hash := h.Hash()
	pubKey, err := crypto.Ecrecover(hash[:], h.SignData)
	if err != nil {
		return nil, err
	}
	nodeID := pubKey[1:]

	h.signerNodeID.Store(nodeID)
	return nodeID, nil
}

// Copy 拷贝一份头
func (h *Header) Copy() *Header {
	cpy := *h
	if len(h.SignData) > 0 {
		cpy.SignData = make([]byte, len(h.SignData))
		copy(cpy.SignData, h.SignData)
	}
	if len(h.DeputyRoot) > 0 {
		cpy.DeputyRoot = make([]byte, len(h.DeputyRoot))
		copy(cpy.DeputyRoot, h.DeputyRoot)
	}
	if len(h.Extra) > 0 {
		cpy.Extra = h.Extra
	}
	return &cpy
}

// rlpHash 数据rlp编码后求hash
func rlpHash(data interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	if err := rlp.Encode(hw, data); err != nil {
		log.Error("hash block fail", "err", err)
	}
	hw.Sum(h[:0])
	return h
}

func (h *Header) String() string {
	set := []string{
		fmt.Sprintf("ParentHash: %s", h.ParentHash.Hex()),
		fmt.Sprintf("MinerAddress: %s", h.MinerAddress.String()),
		fmt.Sprintf("VersionRoot: %s", h.VersionRoot.Hex()),
		fmt.Sprintf("TxRoot: %s", h.TxRoot.Hex()),
		fmt.Sprintf("LogRoot: %s", h.LogRoot.Hex()),
		fmt.Sprintf("Height: %d", h.Height),
		fmt.Sprintf("GasLimit: %d", h.GasLimit),
		fmt.Sprintf("GasUsed: %d", h.GasUsed),
		fmt.Sprintf("Time: %d", h.Time),
		fmt.Sprintf("SignData: %s", common.ToHex(h.SignData[:])),
		fmt.Sprintf("DeputyNodes: %s", common.ToHex(h.DeputyRoot)),
	}
	if len(h.Extra) > 0 {
		set = append(set, fmt.Sprintf("Extra: %s", h.Extra))
	}

	return fmt.Sprintf("{%s}", strings.Join(set, ", "))
}

// rlpHeader
type rlpHeader struct {
	ParentHash   common.Hash
	MinerAddress common.Address
	VersionRoot  common.Hash
	TxRoot       []byte //
	LogRoot      []byte //
	Height       uint32
	GasLimit     uint64
	GasUsed      uint64
	Time         uint32
	SignData     []byte
	DeputyRoot   []byte
	Extra        string
}

// EncodeRLP implements rlp.Encoder.
func (h *Header) EncodeRLP(w io.Writer) error {
	var (
		txRoot  []byte
		logRoot []byte
	)
	if h.TxRoot != merkle.EmptyTrieHash {
		txRoot = h.TxRoot.Bytes()
	}

	if h.LogRoot != merkle.EmptyTrieHash {
		logRoot = h.LogRoot.Bytes()
	}

	return rlp.Encode(w, rlpHeader{
		ParentHash:   h.ParentHash,
		MinerAddress: h.MinerAddress,
		VersionRoot:  h.VersionRoot,
		TxRoot:       txRoot,
		LogRoot:      logRoot,
		Height:       h.Height,
		GasLimit:     h.GasLimit,
		GasUsed:      h.GasUsed,
		Time:         h.Time,
		SignData:     h.SignData,
		DeputyRoot:   h.DeputyRoot,
		Extra:        h.Extra,
	})
}

// DecodeRLP implements rlp.Decoder.
func (h *Header) DecodeRLP(s *rlp.Stream) error {
	var dec rlpHeader

	err := s.Decode(&dec)
	if err == nil {
		h.ParentHash, h.MinerAddress, h.VersionRoot, h.Height, h.GasLimit, h.GasUsed, h.Time, h.SignData, h.DeputyRoot, h.Extra =
			dec.ParentHash, dec.MinerAddress, dec.VersionRoot, dec.Height, dec.GasLimit, dec.GasUsed, dec.Time, dec.SignData, dec.DeputyRoot, dec.Extra

		if len(dec.TxRoot) > 0 {
			h.TxRoot = common.BytesToHash(dec.TxRoot)
		} else {
			h.TxRoot = merkle.EmptyTrieHash
		}
		if len(dec.LogRoot) > 0 {
			h.LogRoot = common.BytesToHash(dec.LogRoot)
		} else {
			h.LogRoot = merkle.EmptyTrieHash
		}
	}
	return err
}

func (b *Block) Hash() common.Hash            { return b.Header.Hash() }
func (b *Block) Height() uint32               { return b.Header.Height }
func (b *Block) ParentHash() common.Hash      { return b.Header.ParentHash }
func (b *Block) MinerAddress() common.Address { return b.Header.MinerAddress }
func (b *Block) VersionRoot() common.Hash     { return b.Header.VersionRoot }
func (b *Block) TxRoot() common.Hash          { return b.Header.TxRoot }
func (b *Block) LogRoot() common.Hash         { return b.Header.LogRoot }
func (b *Block) DeputyRoot() []byte           { return b.Header.DeputyRoot }

func (b *Block) GasLimit() uint64              { return b.Header.GasLimit }
func (b *Block) GasUsed() uint64               { return b.Header.GasUsed }
func (b *Block) Time() uint32                  { return b.Header.Time }
func (b *Block) SignData() []byte              { return b.Header.SignData }
func (b *Block) Extra() string                 { return b.Header.Extra }
func (b *Block) SignerNodeID() ([]byte, error) { return b.Header.SignerNodeID() }

func (b *Block) SetHeader(header *Header)        { b.Header = header }
func (b *Block) SetTxs(txs []*Transaction)       { b.Txs = txs }
func (b *Block) SetConfirms(confirms []SignData) { b.Confirms = confirms }
func (b *Block) SetChangeLogs(logs []*ChangeLog) { b.ChangeLogs = logs }

func (b *Block) SetDeputyNodes(deputyNodes DeputyNodes) { b.DeputyNodes = deputyNodes }

// IsConfirmExist test if the signature is exist in Header.SignData or Confirms
func (b *Block) IsConfirmExist(sig SignData) bool {
	if bytes.Compare(b.Header.SignData, sig[:]) == 0 {
		return true
	}
	for _, oldConfirm := range b.Confirms {
		if bytes.Compare(oldConfirm[:], sig[:]) == 0 {
			return true
		}
	}
	return false
}

func (b *Block) Size() int {
	return binary.Size(b)
}

func (b *Block) String() string {
	set := []string{
		fmt.Sprintf("Header: %v", b.Header),
		fmt.Sprintf("Txs: %v", b.Txs),
		fmt.Sprintf("ChangeLogs: %v", b.ChangeLogs),
		fmt.Sprintf("Confirms: %v", b.Confirms),
		fmt.Sprintf("DeputyNodes: %v", b.DeputyNodes),
	}

	return fmt.Sprintf("{%s}", strings.Join(set, ", "))
}

func (b *Block) ShortString() string {
	hash := b.Hash()
	return fmt.Sprintf("[%d]%x", b.Height(), hash[:8])
}

func (b *Block) Json() string {
	buf, err := json.Marshal(b)
	if err != nil {
		log.Error("Block's marshal failed", "error", err)
		return ""
	}
	return string(buf)
}

func (b *Block) ShallowCopy() *Block {
	return &Block{
		Header:      b.Header,
		Txs:         b.Txs,
		ChangeLogs:  nil,
		Confirms:    b.Confirms,
		DeputyNodes: b.DeputyNodes,
	}
}
