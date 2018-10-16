package node

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/chain/account"
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/store"
	"github.com/LemoFoundationLtd/lemochain-go/store/protocol"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/LemoFoundationLtd/lemochain-go/chain"
	"github.com/LemoFoundationLtd/lemochain-go/chain/params"
	"github.com/LemoFoundationLtd/lemochain-go/common/crypto"
)

type blockInfo struct {
	hash        common.Hash
	parentHash  common.Hash
	height      uint32
	author      common.Address
	versionRoot common.Hash
	txRoot      common.Hash
	logsRoot    common.Hash
	txList      []*types.Transaction
	gasLimit    uint64
	time        *big.Int
}

var (
	chainID         uint16 = 200
	bigNumber, _           = new(big.Int).SetString("1000000000000000000000", 10) // 1 thousand
	testSigner             = types.DefaultSigner{}
	testPrivate, _         = crypto.HexToECDSA("432a86ab8765d82415a803e29864dcfc1ed93dac949abf6f95a583179f27e4bb") // secp256k1.V = 1
	testAddr               = crypto.PubkeyToAddress(testPrivate.PublicKey)                                         // 0x0107134B9CdD7D89F83eFa6175F9b3552F29094c
	defaultAccounts        = []common.Address{
		common.HexToAddress("0x10000"), common.HexToAddress("0x20000"), testAddr,
	}
	defaultBlocks     = make([]*types.Block, 0)
	defaultBlockInfos = []blockInfo{
		// genesis block must no transactions
		{
			hash:        common.HexToHash("0x16019ad7c4d4ecf5163906339048ac73a7aa7131b1154fefeb865c0d523d23f5"),
			height:      0,
			author:      defaultAccounts[0],
			versionRoot: common.HexToHash("0xaa4c649637a466c2879495969aac0403716ad1e8b62a9865bead851d99c6f895"),
			txRoot:      common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"), // empty merkle
			logsRoot:    common.HexToHash("0x189fc4c478582b56997a1808456a32285672109334e9f03654d3e4f181eb83a2"),
			time:        big.NewInt(1538209751),
		},
		// block 1 is stable block
		{
			hash:        common.HexToHash("0xd67857de0f447554c94712d9c0016a8d9e4974d6c3b14b9b062226637d968449"),
			height:      1,
			author:      common.HexToAddress("0x20000"),
			versionRoot: common.HexToHash("0xc4fa99a20b2db7026f80366e34213861457e240bf0411d57f7a22cbf5b7f2346"),
			txRoot:      common.HexToHash("0xf044cc436950ef7470aca61053eb3f1ed46b9dcd501a5210f3673dc657c4fc88"),
			logsRoot:    common.HexToHash("0xc287a6c955f3d6d4d240413a585f48009106a6ff5886b194cb506e87323786d7"),
			txList: []*types.Transaction{
				// testAddr -> defaultAccounts[0] 1
				signTransaction(types.NewTransaction(defaultAccounts[0], common.Big1, 2000000, common.Big2, []byte{12}, chainID, big.NewInt(1538210391), "aa", []byte{34}), testPrivate),
				// testAddr -> defaultAccounts[1] 1
				makeTransaction(testPrivate, defaultAccounts[1], common.Big1, common.Big2, big.NewInt(1538210491), 2000000),
			},
			gasLimit: 20000000,
			time:     big.NewInt(1538209755),
		},
		// block 2 is not stable block
		{
			hash:        common.HexToHash("0x5afb6907e01a243325ce7c6e56e463f777080f6e5277ba2ec83928329c8dce61"),
			height:      2,
			author:      defaultAccounts[0],
			versionRoot: common.HexToHash("0xb169fb0304ad3c989589b044afbfe6257d125222e5023004e418b4bdb5f154f9"),
			txRoot:      common.HexToHash("0x2501f471d13f0383a9a09ef7776dc94c6ccdc19ebd36127ac8a39cfef85dc412"),
			logsRoot:    common.HexToHash("0x6c24ab2c37e6bdd12c276361ba7aa1300ab389b4861ca364e1fdd480197803da"),
			txList: []*types.Transaction{
				// testAddr -> defaultAccounts[0] 2
				makeTransaction(testPrivate, defaultAccounts[0], bigNumber, common.Big2, big.NewInt(1538210395), 2000000),
			},
			time:     big.NewInt(1538209758),
			gasLimit: 20000000,
		},
		// block 3 is not store in db
		{
			hash:        common.HexToHash("0x1889ca33d2ea9bfe68b171258e19f3034e9518c47d15b1d484797458e96cfb96"),
			height:      3,
			author:      defaultAccounts[0],
			versionRoot: common.HexToHash("0xfe833c7d3aa89fc1031198374e6eda37425de579e02430e46cc2273ee7f94162"),
			txRoot:      common.HexToHash("0x883e9542653e2ad8bae93fa5567d8f5833543057af94ea0ed59c97a7034acac8"),
			logsRoot:    common.HexToHash("0xeff8eb5827967fe372182d8173b8be788c8367e7c04823c09bf49d40bf070982"),
			txList: []*types.Transaction{
				// testAddr -> defaultAccounts[0] 2
				makeTransaction(testPrivate, defaultAccounts[0], common.Big2, common.Big2, big.NewInt(1538210398), 30000),
				// testAddr -> defaultAccounts[1] 2
				makeTransaction(testPrivate, defaultAccounts[1], common.Big2, common.Big3, big.NewInt(1538210425), 30000),
			},
			time:     big.NewInt(1538209761),
			gasLimit: 20000000,
		},
	}
)

func init() {
	clearDB()
}

func clearDB() {
	filepath.Walk("../../db", func(path string, f os.FileInfo, err error) error {
		return os.RemoveAll(path)
	})
}

// newChain creates chain for test
func newChain() *chain.BlockChain {
	db := newDB()
	newBlockCh := make(chan *types.Block)
	bc, err := chain.NewBlockChain(uint64(chainID), chain.NewDpovp(10*1000, 3*1000), db, newBlockCh, map[string]string{})
	if err != nil {
		panic(err)
	}
	return bc
}

// newDB creates db for test chain module
func newDB() protocol.ChainDB {
	db, err := store.NewCacheChain("../../db")
	if err != nil {
		panic(err)
	}

	for i, _ := range defaultBlockInfos {
		if i > 0 {
			defaultBlockInfos[i].parentHash = defaultBlocks[i-1].Hash()
		}
		newBlock := makeBlock(db, defaultBlockInfos[i], i < 3)
		defaultBlocks = append(defaultBlocks, newBlock)
	}
	err = db.SetStableBlock(defaultBlockInfos[1].hash)
	if err != nil {
		panic(err)
	}
	return db
}

func makeBlock(db protocol.ChainDB, info blockInfo, save bool) *types.Block {
	manager := account.NewManager(info.parentHash, db)
	// sign transactions
	var err error
	var gasUsed uint64 = 0
	txRoot := types.DeriveTxsSha(info.txList)
	if txRoot != info.txRoot {
		if info.txRoot != (common.Hash{}) {
			fmt.Printf("%d txRoot hash error. except: %s, got: %s\n", info.height, info.txRoot.Hex(), txRoot.Hex())
		}
		info.txRoot = txRoot
	}
	// genesis coin
	if info.height == 0 {
		owner := manager.GetAccount(testAddr)
		// 1 million
		balance, _ := new(big.Int).SetString("1000000000000000000000000", 10)
		owner.SetBalance(balance)
	}
	// account
	salary := new(big.Int)
	for _, tx := range info.txList {
		gas := params.TxGas + params.TxDataNonZeroGas*uint64(len(tx.Data()))
		fromAddr, err := tx.From()
		if err != nil {
			panic(err)
		}
		from := manager.GetAccount(fromAddr)
		fee := new(big.Int).Mul(new(big.Int).SetUint64(gas), tx.GasPrice())
		cost := new(big.Int).Add(tx.Value(), fee)
		to := manager.GetAccount(*tx.To())
		// make sure the change log has right order
		if fromAddr.Hex() < tx.To().Hex() {
			from.SetBalance(new(big.Int).Sub(from.GetBalance(), cost))
			to.SetBalance(new(big.Int).Add(to.GetBalance(), tx.Value()))
		} else {
			to.SetBalance(new(big.Int).Add(to.GetBalance(), tx.Value()))
			from.SetBalance(new(big.Int).Sub(from.GetBalance(), cost))
		}
		gasUsed += gas
		salary.Add(salary, fee)
	}
	miner := manager.GetAccount(info.author)
	miner.SetBalance(new(big.Int).Add(miner.GetBalance(), salary))
	err = manager.Finalise()
	if err != nil {
		panic(err)
	}
	// header
	if manager.GetVersionRoot() != info.versionRoot {
		if info.versionRoot != (common.Hash{}) {
			fmt.Printf("%d version root error. except: %s, got: %s\n", info.height, info.versionRoot.Hex(), manager.GetVersionRoot().Hex())
		}
		info.versionRoot = manager.GetVersionRoot()
	}
	changeLogs := manager.GetChangeLogs()
	// fmt.Printf("%d changeLogs %v\n", info.height, changeLogs)
	logsRoot := types.DeriveChangeLogsSha(changeLogs)
	if logsRoot != info.logsRoot {
		if info.logsRoot != (common.Hash{}) {
			fmt.Printf("%d change logs root error. except: %s, got: %s\n", info.height, info.logsRoot.Hex(), logsRoot.Hex())
		}
		info.logsRoot = logsRoot
	}
	if info.time == nil {
		info.time = new(big.Int).SetUint64(uint64(time.Now().Unix()))
	}
	if info.gasLimit == 0 {
		info.gasLimit = 1000000
	}
	header := &types.Header{
		ParentHash:  info.parentHash,
		LemoBase:    info.author,
		VersionRoot: info.versionRoot,
		TxRoot:      info.txRoot,
		LogsRoot:    info.logsRoot,
		Bloom:       types.CreateBloom(nil),
		EventRoot:   common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"), // empty merkle
		Height:      info.height,
		GasLimit:    info.gasLimit,
		GasUsed:     gasUsed,
		Time:        info.time,
		Extra:       []byte{},
	}
	blockHash := header.Hash()
	if blockHash != info.hash {
		if info.hash != (common.Hash{}) {
			fmt.Printf("%d block hash error. except: %s, got: %s\n", info.height, info.hash.Hex(), blockHash.Hex())
		}
		info.hash = blockHash
	}
	// block
	block := &types.Block{
		Txs:       info.txList,
		ChangeLog: changeLogs,
	}
	block.SetHeader(header)
	if save {
		err = db.SetBlock(blockHash, block)
		if err != nil && err != store.ErrExist {
			panic(err)
		}
		err = manager.Save(blockHash)
		if err != nil {
			panic(err)
		}
	}
	return block
}

func makeTx(fromPrivate *ecdsa.PrivateKey, to common.Address, amount *big.Int) *types.Transaction {
	return makeTransaction(fromPrivate, to, amount, common.Big1, new(big.Int).SetUint64(uint64(time.Now().Unix()+300)), 1000000)
}

func makeTransaction(fromPrivate *ecdsa.PrivateKey, to common.Address, amount, gasPrice, expiration *big.Int, gasLimit uint64) *types.Transaction {
	tx := types.NewTransaction(to, amount, gasLimit, gasPrice, []byte{}, chainID, expiration, "", []byte{})
	return signTransaction(tx, fromPrivate)
}

func signTransaction(tx *types.Transaction, private *ecdsa.PrivateKey) *types.Transaction {
	tx, err := types.SignTx(tx, testSigner, private)
	if err != nil {
		panic(err)
	}
	return tx
}

// h returns hash for test
func h(i int64) common.Hash { return common.HexToHash(fmt.Sprintf("0xa%x", i)) }

// b returns block hash for test
func b(i int64) common.Hash { return common.HexToHash(fmt.Sprintf("0xb%x", i)) }

// c returns code hash for test
func c(i int64) common.Hash { return common.HexToHash(fmt.Sprintf("0xc%x", i)) }

// k returns storage key hash for test
func k(i int64) common.Hash { return common.HexToHash(fmt.Sprintf("0xd%x", i)) }

// t returns transaction hash for test
func th(i int64) common.Hash { return common.HexToHash(fmt.Sprintf("0xe%x", i)) }