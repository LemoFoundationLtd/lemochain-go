// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package deputynode

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
)

var _ = (*deputyNodeMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (d DeputyNode) MarshalJSON() ([]byte, error) {
	type DeputyNode struct {
		MinerAddress common.Address `json:"minerAddress"   gencodec:"required"`
		NodeID       hexutil.Bytes  `json:"nodeID"         gencodec:"required"`
		Rank         hexutil.Uint32 `json:"rank"           gencodec:"required"`
		Votes        *hexutil.Big10 `json:"votes"          gencodec:"required"`
	}
	var enc DeputyNode
	enc.MinerAddress = d.MinerAddress
	enc.NodeID = d.NodeID
	enc.Rank = hexutil.Uint32(d.Rank)
	enc.Votes = (*hexutil.Big10)(d.Votes)
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (d *DeputyNode) UnmarshalJSON(input []byte) error {
	type DeputyNode struct {
		MinerAddress *common.Address `json:"minerAddress"   gencodec:"required"`
		NodeID       *hexutil.Bytes  `json:"nodeID"         gencodec:"required"`
		Rank         *hexutil.Uint32 `json:"rank"           gencodec:"required"`
		Votes        *hexutil.Big10  `json:"votes"          gencodec:"required"`
	}
	var dec DeputyNode
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.MinerAddress == nil {
		return errors.New("missing required field 'minerAddress' for DeputyNode")
	}
	d.MinerAddress = *dec.MinerAddress
	if dec.NodeID == nil {
		return errors.New("missing required field 'nodeID' for DeputyNode")
	}
	d.NodeID = *dec.NodeID
	if dec.Rank == nil {
		return errors.New("missing required field 'rank' for DeputyNode")
	}
	d.Rank = uint32(*dec.Rank)
	if dec.Votes == nil {
		return errors.New("missing required field 'votes' for DeputyNode")
	}
	d.Votes = (*big.Int)(dec.Votes)
	return nil
}