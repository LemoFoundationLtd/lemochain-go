package store

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/rlp"
	"github.com/LemoFoundationLtd/lemochain-core/store/leveldb"
	"strconv"
)

//go:generate gencodec -type VTransaction --field-override vTransactionMarshaling -out gen_vTransaction_info_json.go
type VTransaction struct {
	Tx *types.Transaction `json:"tx" gencodec:"required"`
	St int64              `json:"time" gencodec:"required"`
}
type vTransactionMarshaling struct {
	St hexutil.Uint64
}

//go:generate gencodec -type VTransactionDetail --field-override vTransactionDetailMarshaling -out gen_vTransactionDetail_info_json.go
type VTransactionDetail struct {
	BlockHash common.Hash        `json:"blockHash" gencodec:"required"`
	Height    uint32             `json:"height" gencodec:"required"`
	Tx        *types.Transaction `json:"tx"  gencodec:"required"`
	St        int64              `json:"time" gencodec:"required"`
}

type vTransactionDetailMarshaling struct {
	Height hexutil.Uint32
	St     hexutil.Uint64
}

type BizDb interface {
	GetTxByHash(hash common.Hash) (*VTransactionDetail, error)

	GetTxByAddr(src common.Address, index int, size int) ([]*VTransaction, uint32, error)
}

type Reader interface {
	GetLastConfirm() *CBlock

	GetBlockByHash(hash common.Hash) (*types.Block, error)
}

type BizDatabase struct {
	Reader  Reader
	LevelDB *leveldb.LevelDBDatabase
}

func NewBizDatabase(reader Reader, levelDB *leveldb.LevelDBDatabase) *BizDatabase {
	return &BizDatabase{
		Reader:  reader,
		LevelDB: levelDB,
	}
}

func (db *BizDatabase) GetTxByHash(hash common.Hash) (*VTransactionDetail, error) {
	// blockHash, val, st, err := db.Database.TxGetByHash(hash.Hex())
	// if err != nil {
	// 	return nil, err
	// }
	//
	// block, err := db.Reader.GetBlockByHash(common.HexToHash(blockHash))
	// if err == ErrNotExist {
	// 	return nil, ErrNotExist
	// }
	//
	// if err != nil {
	// 	return nil, err
	// }
	//
	// var tx types.Transaction
	// err = rlp.DecodeBytes(val, &tx)
	// if err != nil {
	// 	return nil, err
	// }
	//
	// return &VTransactionDetail{
	// 	BlockHash: block.Hash(),
	// 	Height:    block.Height(),
	// 	Tx:        &tx,
	// 	St:        st,
	// }, nil
	return nil, nil
}

func (db *BizDatabase) GetTxByAddr(src common.Address, index int, size int) ([]*VTransaction, uint32, error) {
	// if (index < 0) || (size > 200) || (size <= 0) {
	// 	return nil, 0, errors.New("argment error.")
	// }
	//
	// confirm := db.Reader.GetLastConfirm()
	// trieDB := confirm.AccountTrieDB
	// if trieDB == nil {
	// 	return make([]*VTransaction, 0), 0, nil
	// }
	//
	// account, err := trieDB.Get(src)
	// if err != nil {
	// 	return nil, 0, err
	// }
	//
	// if account == nil {
	// 	return make([]*VTransaction, 0), 0, nil
	// }
	//
	// txCount := uint32(0)
	// // txCount := account.TxCount
	// // if uint32(index) > txCount {
	// // 	return make([]*VTransaction, 0), txCount, nil
	// // }
	//
	// _, vals, sts, err := db.Database.TxGetByAddr(src.Hex(), index, size)
	// if err != nil {
	// 	return nil, 0, err
	// }
	//
	// txs := make([]*VTransaction, len(vals))
	// for index := 0; index < len(vals); index++ {
	// 	var tx types.Transaction
	// 	err = rlp.DecodeBytes(vals[index], &tx)
	// 	if err != nil {
	// 		return nil, 0, err
	// 	}
	//
	// 	txs[index] = &VTransaction{
	// 		Tx: &tx,
	// 		St: sts[index],
	// 	}
	// }
	// return txs, txCount, nil
	return nil, 0, nil
}

func (db *BizDatabase) AfterCommit(flag uint, key []byte, val []byte) error {
	if flag == CACHE_FLG_BLOCK {
		return db.afterBlock(key, val)
	} else if flag == CACHE_FLG_BLOCK_HEIGHT {
		return nil
	} else if flag == CACHE_FLG_TRIE {
		return nil
	} else if flag == CACHE_FLG_ACT {
		return nil
	} else if flag == CACHE_FLG_TX_INDEX {
		return nil
	} else if flag == CACHE_FLG_CODE {
		return nil
	} else if flag == CACHE_FLG_KV {
		return nil
	} else {
		panic("unknown flag.flag = " + strconv.Itoa(int(flag)))
	}
}

func (db *BizDatabase) afterBlock(key []byte, val []byte) error {
	var block types.Block
	err := rlp.DecodeBytes(val, &block)
	ret := hexutil.Encode(val)
	fmt.Sprintf(ret)
	if err != nil {
		return err
	}

	txs := block.Txs
	if len(txs) <= 0 {
		return nil
	}

	// ver := time.Now().UnixNano()
	for index := 0; index < len(txs); index++ {
		tx := txs[index]
		// hash := tx.Hash()
		from, err := tx.From()
		if err != nil {
			return err
		}

		txType := tx.Type()
		if txType == params.CreateAssetTx {
			log.Info("insert account code: " + tx.Hash().Hex() + "|addr: " + from.Hex())
			err := leveldb.Set(db.LevelDB, leveldb.GetAssetCodeKey(tx.Hash()), from.Bytes())
			if err != nil {
				panic("insert account code err: " + err.Error())
			}
		} else if txType == params.IssueAssetTx {
			log.Info("insert account id: " + tx.Hash().Hex() + "|addr: " + from.Hex())
			err := leveldb.Set(db.LevelDB, leveldb.GetAssetIdKey(tx.Hash()), from.Bytes())
			if err != nil {
				panic("insert account id err: " + err.Error())
			}
		}

		//err = db.Database.TxSet(hashStr, common.ToHex(key), fromStr, toStr, val, ver+int64(index), int64(block.Header.Time))
		if err != nil {
			return err
		}
	}
	return nil
}
