// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package types

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
)

var _ = (*accountDataMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (a AccountData) MarshalJSON() ([]byte, error) {
	type AccountData struct {
		Address       common.Address                  `json:"address" gencodec:"required"`
		Balance       *hexutil.Big10                  `json:"balance" gencodec:"required"`
		CodeHash      common.Hash                     `json:"codeHash" gencodec:"required"`
		StorageRoot   common.Hash                     `json:"root" gencodec:"required"`
		NewestRecords map[ChangeLogType]VersionRecord `json:"records" gencodec:"required"`
		VoteFor       common.Address                  `json:"voteFor"`
		Candidate     Candidate                       `json:"candidate"`
		TxCount       hexutil.Uint32                  `json:"txCount"`
	}
	var enc AccountData
	enc.Address = a.Address
	enc.Balance = (*hexutil.Big10)(a.Balance)
	enc.CodeHash = a.CodeHash
	enc.StorageRoot = a.StorageRoot
	enc.NewestRecords = a.NewestRecords
	enc.VoteFor = a.VoteFor
	enc.Candidate = a.Candidate
	enc.TxCount = hexutil.Uint32(a.TxCount)
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (a *AccountData) UnmarshalJSON(input []byte) error {
	type AccountData struct {
		Address       *common.Address                 `json:"address" gencodec:"required"`
		Balance       *hexutil.Big10                  `json:"balance" gencodec:"required"`
		CodeHash      *common.Hash                    `json:"codeHash" gencodec:"required"`
		StorageRoot   *common.Hash                    `json:"root" gencodec:"required"`
		NewestRecords map[ChangeLogType]VersionRecord `json:"records" gencodec:"required"`
		VoteFor       *common.Address                 `json:"voteFor"`
		Candidate     *Candidate                      `json:"candidate"`
		TxCount       *hexutil.Uint32                 `json:"txCount"`
	}
	var dec AccountData
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Address == nil {
		return errors.New("missing required field 'address' for AccountData")
	}
	a.Address = *dec.Address
	if dec.Balance == nil {
		return errors.New("missing required field 'balance' for AccountData")
	}
	a.Balance = (*big.Int)(dec.Balance)
	if dec.CodeHash == nil {
		return errors.New("missing required field 'codeHash' for AccountData")
	}
	a.CodeHash = *dec.CodeHash
	if dec.StorageRoot == nil {
		return errors.New("missing required field 'root' for AccountData")
	}
	a.StorageRoot = *dec.StorageRoot
	if dec.NewestRecords == nil {
		return errors.New("missing required field 'records' for AccountData")
	}
	a.NewestRecords = dec.NewestRecords
	if dec.VoteFor != nil {
		a.VoteFor = *dec.VoteFor
	}
	if dec.Candidate != nil {
		a.Candidate = *dec.Candidate
	}
	if dec.TxCount != nil {
		a.TxCount = uint32(*dec.TxCount)
	}
	return nil
}
