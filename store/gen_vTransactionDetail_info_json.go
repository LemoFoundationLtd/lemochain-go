// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package store

import (
	"encoding/json"

	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
)

// MarshalJSON marshals as JSON.
func (v VTransactionDetail) MarshalJSON() ([]byte, error) {
	type VTransactionDetail struct {
		BlockHash common.Hash        `json:"blockHash"  	gencodec:"required"`
		Height    uint32             `json:"height"  	gencodec:"required"`
		Tx        *types.Transaction `json:"tx"  		gencodec:"required"`
		St        int64              `json:"time"  		gencodec:"required"`
	}
	var enc VTransactionDetail
	enc.BlockHash = v.BlockHash
	enc.Height = v.Height
	enc.Tx = v.Tx
	enc.St = v.St
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (v *VTransactionDetail) UnmarshalJSON(input []byte) error {
	type VTransactionDetail struct {
		BlockHash *common.Hash       `json:"blockHash"  	gencodec:"required"`
		Height    *uint32            `json:"height"  	gencodec:"required"`
		Tx        *types.Transaction `json:"tx"  		gencodec:"required"`
		St        *int64             `json:"time"  		gencodec:"required"`
	}
	var dec VTransactionDetail
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.BlockHash != nil {
		v.BlockHash = *dec.BlockHash
	}
	if dec.Height != nil {
		v.Height = *dec.Height
	}
	if dec.Tx != nil {
		v.Tx = dec.Tx
	}
	if dec.St != nil {
		v.St = *dec.St
	}
	return nil
}
