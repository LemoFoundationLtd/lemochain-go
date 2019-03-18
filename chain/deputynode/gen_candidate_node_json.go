// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package deputynode

import (
	"encoding/json"
	"errors"

	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
)

var _ = (*candidateNodeMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (c CandidateNode) MarshalJSON() ([]byte, error) {
	type CandidateNode struct {
		IsCandidate  bool           `json:"isCandidate"      gencodec:"required"`
		MinerAddress common.Address `json:"minerAddress"        gencodec:"required"`
		NodeID       hexutil.Bytes  `json:"nodeID"         gencodec:"required"`
		Host         string         `json:"host"             gencodec:"required"`
		Port         hexutil.Uint32 `json:"port"           gencodec:"required"`
	}
	var enc CandidateNode
	enc.IsCandidate = c.IsCandidate
	enc.MinerAddress = c.MinerAddress
	enc.NodeID = c.NodeID
	enc.Host = c.Host
	enc.Port = hexutil.Uint32(c.Port)
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (c *CandidateNode) UnmarshalJSON(input []byte) error {
	type CandidateNode struct {
		IsCandidate  *bool           `json:"isCandidate"      gencodec:"required"`
		MinerAddress *common.Address `json:"minerAddress"        gencodec:"required"`
		NodeID       *hexutil.Bytes  `json:"nodeID"         gencodec:"required"`
		Host         *string         `json:"host"             gencodec:"required"`
		Port         *hexutil.Uint32 `json:"port"           gencodec:"required"`
	}
	var dec CandidateNode
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.IsCandidate == nil {
		return errors.New("missing required field 'isCandidate' for CandidateNode")
	}
	c.IsCandidate = *dec.IsCandidate
	if dec.MinerAddress == nil {
		return errors.New("missing required field 'minerAddress' for CandidateNode")
	}
	c.MinerAddress = *dec.MinerAddress
	if dec.NodeID == nil {
		return errors.New("missing required field 'nodeID' for CandidateNode")
	}
	c.NodeID = *dec.NodeID
	if dec.Host == nil {
		return errors.New("missing required field 'host' for CandidateNode")
	}
	c.Host = *dec.Host
	if dec.Port == nil {
		return errors.New("missing required field 'port' for CandidateNode")
	}
	c.Port = uint32(*dec.Port)
	return nil
}
