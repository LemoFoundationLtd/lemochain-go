// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package types

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/hexutil"
)

var _ = (*txdataMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (t txdata) MarshalJSON() ([]byte, error) {
	type txdata struct {
		Recipient     *common.Address `json:"to" rlp:"nil"`
		RecipientName string          `json:"toName"`
		GasPrice      *hexutil.Big10  `json:"gasPrice" gencodec:"required"`
		GasLimit      hexutil.Uint64  `json:"gasLimit" gencodec:"required"`
		Amount        *hexutil.Big10  `json:"amount" gencodec:"required"`
		Data          hexutil.Bytes   `json:"data"`
		Expiration    hexutil.Uint64  `json:"expirationTime" gencodec:"required"`
		Message       string          `json:"message"`
		V             *hexutil.Big    `json:"v" gencodec:"required"`
		R             *hexutil.Big    `json:"r" gencodec:"required"`
		S             *hexutil.Big    `json:"s" gencodec:"required"`
		Hash          *common.Hash    `json:"hash" rlp:"-"`
		GasPayerSig   hexutil.Bytes   `json:"gasPayerSig"`
	}
	var enc txdata
	enc.Recipient = t.Recipient
	enc.RecipientName = t.RecipientName
	enc.GasPrice = (*hexutil.Big10)(t.GasPrice)
	enc.GasLimit = hexutil.Uint64(t.GasLimit)
	enc.Amount = (*hexutil.Big10)(t.Amount)
	enc.Data = t.Data
	enc.Expiration = hexutil.Uint64(t.Expiration)
	enc.Message = t.Message
	enc.V = (*hexutil.Big)(t.V)
	enc.R = (*hexutil.Big)(t.R)
	enc.S = (*hexutil.Big)(t.S)
	enc.Hash = t.Hash
	enc.GasPayerSig = t.GasPayerSig
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (t *txdata) UnmarshalJSON(input []byte) error {
	type txdata struct {
		Recipient     *common.Address `json:"to" rlp:"nil"`
		RecipientName *string         `json:"toName"`
		GasPrice      *hexutil.Big10  `json:"gasPrice" gencodec:"required"`
		GasLimit      *hexutil.Uint64 `json:"gasLimit" gencodec:"required"`
		Amount        *hexutil.Big10  `json:"amount" gencodec:"required"`
		Data          *hexutil.Bytes  `json:"data"`
		Expiration    *hexutil.Uint64 `json:"expirationTime" gencodec:"required"`
		Message       *string         `json:"message"`
		V             *hexutil.Big    `json:"v" gencodec:"required"`
		R             *hexutil.Big    `json:"r" gencodec:"required"`
		S             *hexutil.Big    `json:"s" gencodec:"required"`
		Hash          *common.Hash    `json:"hash" rlp:"-"`
		GasPayerSig   *hexutil.Bytes  `json:"gasPayerSig"`
	}
	var dec txdata
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Recipient != nil {
		t.Recipient = dec.Recipient
	}
	if dec.RecipientName != nil {
		t.RecipientName = *dec.RecipientName
	}
	if dec.GasPrice == nil {
		return errors.New("missing required field 'gasPrice' for txdata")
	}
	t.GasPrice = (*big.Int)(dec.GasPrice)
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for txdata")
	}
	t.GasLimit = uint64(*dec.GasLimit)
	if dec.Amount == nil {
		return errors.New("missing required field 'amount' for txdata")
	}
	t.Amount = (*big.Int)(dec.Amount)
	if dec.Data != nil {
		t.Data = *dec.Data
	}
	if dec.Expiration == nil {
		return errors.New("missing required field 'expirationTime' for txdata")
	}
	t.Expiration = uint64(*dec.Expiration)
	if dec.Message != nil {
		t.Message = *dec.Message
	}
	if dec.V == nil {
		return errors.New("missing required field 'v' for txdata")
	}
	t.V = (*big.Int)(dec.V)
	if dec.R == nil {
		return errors.New("missing required field 'r' for txdata")
	}
	t.R = (*big.Int)(dec.R)
	if dec.S == nil {
		return errors.New("missing required field 's' for txdata")
	}
	t.S = (*big.Int)(dec.S)
	if dec.Hash != nil {
		t.Hash = dec.Hash
	}
	if dec.GasPayerSig != nil {
		t.GasPayerSig = *dec.GasPayerSig
	}
	return nil
}
