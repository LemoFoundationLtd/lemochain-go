// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package types

import (
	"encoding/json"
	"errors"

	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
)

var _ = (*headerMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (h Header) MarshalJSON() ([]byte, error) {
	type Header struct {
		ParentHash   common.Hash    `json:"parentHash"       gencodec:"required"`
		MinerAddress common.Address `json:"miner"            gencodec:"required"`
		VersionRoot  common.Hash    `json:"versionRoot"      gencodec:"required"`
		TxRoot       common.Hash    `json:"transactionRoot"  gencodec:"required"`
		LogRoot      common.Hash    `json:"changeLogRoot"    gencodec:"required"`
		Height       hexutil.Uint32 `json:"height"           gencodec:"required"`
		GasLimit     hexutil.Uint64 `json:"gasLimit"         gencodec:"required"`
		GasUsed      hexutil.Uint64 `json:"gasUsed"          gencodec:"required"`
		Time         hexutil.Uint32 `json:"timestamp"        gencodec:"required"`
		SignData     hexutil.Bytes  `json:"signData"         gencodec:"required"`
		DeputyRoot   hexutil.Bytes  `json:"deputyRoot"`
		Extra        string         `json:"extraData"`
		Hash         common.Hash    `json:"hash"`
	}
	var enc Header
	enc.ParentHash = h.ParentHash
	enc.MinerAddress = h.MinerAddress
	enc.VersionRoot = h.VersionRoot
	enc.TxRoot = h.TxRoot
	enc.LogRoot = h.LogRoot
	enc.Height = hexutil.Uint32(h.Height)
	enc.GasLimit = hexutil.Uint64(h.GasLimit)
	enc.GasUsed = hexutil.Uint64(h.GasUsed)
	enc.Time = hexutil.Uint32(h.Time)
	enc.SignData = h.SignData
	enc.DeputyRoot = h.DeputyRoot
	enc.Extra = h.Extra
	enc.Hash = h.Hash()
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (h *Header) UnmarshalJSON(input []byte) error {
	type Header struct {
		ParentHash   *common.Hash    `json:"parentHash"       gencodec:"required"`
		MinerAddress *common.Address `json:"miner"            gencodec:"required"`
		VersionRoot  *common.Hash    `json:"versionRoot"      gencodec:"required"`
		TxRoot       *common.Hash    `json:"transactionRoot"  gencodec:"required"`
		LogRoot      *common.Hash    `json:"changeLogRoot"    gencodec:"required"`
		Height       *hexutil.Uint32 `json:"height"           gencodec:"required"`
		GasLimit     *hexutil.Uint64 `json:"gasLimit"         gencodec:"required"`
		GasUsed      *hexutil.Uint64 `json:"gasUsed"          gencodec:"required"`
		Time         *hexutil.Uint32 `json:"timestamp"        gencodec:"required"`
		SignData     *hexutil.Bytes  `json:"signData"         gencodec:"required"`
		DeputyRoot   *hexutil.Bytes  `json:"deputyRoot"`
		Extra        *string         `json:"extraData"`
	}
	var dec Header
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.ParentHash == nil {
		return errors.New("missing required field 'parentHash' for Header")
	}
	h.ParentHash = *dec.ParentHash
	if dec.MinerAddress == nil {
		return errors.New("missing required field 'miner' for Header")
	}
	h.MinerAddress = *dec.MinerAddress
	if dec.VersionRoot == nil {
		return errors.New("missing required field 'versionRoot' for Header")
	}
	h.VersionRoot = *dec.VersionRoot
	if dec.TxRoot == nil {
		return errors.New("missing required field 'transactionRoot' for Header")
	}
	h.TxRoot = *dec.TxRoot
	if dec.LogRoot == nil {
		return errors.New("missing required field 'changeLogRoot' for Header")
	}
	h.LogRoot = *dec.LogRoot
	if dec.Height == nil {
		return errors.New("missing required field 'height' for Header")
	}
	h.Height = uint32(*dec.Height)
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for Header")
	}
	h.GasLimit = uint64(*dec.GasLimit)
	if dec.GasUsed == nil {
		return errors.New("missing required field 'gasUsed' for Header")
	}
	h.GasUsed = uint64(*dec.GasUsed)
	if dec.Time == nil {
		return errors.New("missing required field 'timestamp' for Header")
	}
	h.Time = uint32(*dec.Time)
	if dec.SignData == nil {
		return errors.New("missing required field 'signData' for Header")
	}
	h.SignData = *dec.SignData
	if dec.DeputyRoot != nil {
		h.DeputyRoot = *dec.DeputyRoot
	}
	if dec.Extra != nil {
		h.Extra = *dec.Extra
	}
	return nil
}
