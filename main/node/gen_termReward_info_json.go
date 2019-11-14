// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package node

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
)

var _ = (*termRewardInfoMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (t TermRewardInfo) MarshalJSON() ([]byte, error) {
	type TermRewardInfo struct {
		Term         hexutil.Uint32 `json:"term" gencodec:"required"`
		Value        *hexutil.Big10 `json:"value" gencodec:"required"`
		RewardHeight hexutil.Uint32 `json:"rewardHeight" gencodec:"required"`
	}
	var enc TermRewardInfo
	enc.Term = hexutil.Uint32(t.Term)
	enc.Value = (*hexutil.Big10)(t.Value)
	enc.RewardHeight = hexutil.Uint32(t.RewardHeight)
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (t *TermRewardInfo) UnmarshalJSON(input []byte) error {
	type TermRewardInfo struct {
		Term         *hexutil.Uint32 `json:"term" gencodec:"required"`
		Value        *hexutil.Big10  `json:"value" gencodec:"required"`
		RewardHeight *hexutil.Uint32 `json:"rewardHeight" gencodec:"required"`
	}
	var dec TermRewardInfo
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Term == nil {
		return errors.New("missing required field 'term' for TermRewardInfo")
	}
	t.Term = uint32(*dec.Term)
	if dec.Value == nil {
		return errors.New("missing required field 'value' for TermRewardInfo")
	}
	t.Value = (*big.Int)(dec.Value)
	if dec.RewardHeight == nil {
		return errors.New("missing required field 'rewardHeight' for TermRewardInfo")
	}
	t.RewardHeight = uint32(*dec.RewardHeight)
	return nil
}