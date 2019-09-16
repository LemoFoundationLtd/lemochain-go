package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/miner"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/hexutil"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/subscribe"
	"github.com/LemoFoundationLtd/lemochain-core/network"
	"github.com/LemoFoundationLtd/lemochain-core/network/p2p"
	"math/big"
	"runtime"
	"strconv"
	"time"
)

var (
	ErrToName         = errors.New("the length of toName field in transaction is out of max length limit")
	ErrTxMessage      = errors.New("the length of message field in transaction is out of max length limit")
	ErrCreateContract = errors.New("the data of create contract transaction can't be null")
	ErrSpecialTx      = errors.New("the data of special transaction can't be null")
	ErrTxType         = errors.New("the transaction type does not exist")
	ErrLemoAddress    = errors.New("lemoAddress is incorrect")
	ErrAssetId        = errors.New("assetid is incorrect")
	ErrTxExpiration   = errors.New("tx expiration time is out of date")
	ErrNegativeValue  = errors.New("negative value")
	ErrTxChainID      = errors.New("tx chainID is incorrect")
	ErrInputParams    = errors.New("input params incorrect")
	ErrTxTo           = errors.New("transaction to is incorrect")
)

// Private
type PrivateAccountAPI struct {
	manager *account.Manager
}

// NewPrivateAccountAPI
func NewPrivateAccountAPI(m *account.Manager) *PrivateAccountAPI {
	return &PrivateAccountAPI{m}
}

//go:generate gencodec -type LemoAccount -out gen_lemo_account_json.go
type LemoAccount struct {
	Private string         `json:"private"`
	Address common.Address `json:"address"`
}

// NewAccount get lemo address api
func (a *PrivateAccountAPI) NewKeyPair() (*LemoAccount, error) {
	accountKey, err := crypto.GenerateAddress()
	if err != nil {
		return nil, err
	}
	acc := &LemoAccount{
		Private: accountKey.Private,
		Address: accountKey.Address,
	}
	return acc, nil
}

// PublicAccountAPI API for access to account information
type PublicAccountAPI struct {
	manager *account.Manager
}

// NewPublicAccountAPI
func NewPublicAccountAPI(m *account.Manager) *PublicAccountAPI {
	return &PublicAccountAPI{m}
}

// GetBalance get balance in mo
func (a *PublicAccountAPI) GetBalance(LemoAddress string) (string, error) {
	if !network.VerifyLemoAddress(LemoAddress) {
		log.Warnf("LemoAddress is incorrect. lemoAddress: %s", LemoAddress)
		return "", ErrLemoAddress
	}
	lemoAccount, err := a.GetAccount(LemoAddress)
	if err != nil {
		return "", err
	}
	balance := lemoAccount.GetBalance().String()

	return balance, nil
}

// GetAccount return the struct of the &AccountData{}
func (a *PublicAccountAPI) GetAccount(LemoAddress string) (types.AccountAccessor, error) {
	if !network.VerifyLemoAddress(LemoAddress) {
		log.Warnf("LemoAddress is incorrect. lemoAddress: %s", LemoAddress)
		return nil, ErrLemoAddress
	}
	address, err := common.StringToAddress(LemoAddress)
	if err != nil {
		return nil, err
	}
	accountData := a.manager.GetCanonicalAccount(address)
	// accountData := a.manager.GetAccount(address)
	return accountData, nil
}

// GetVoteFor
func (a *PublicAccountAPI) GetVoteFor(LemoAddress string) (string, error) {
	if !network.VerifyLemoAddress(LemoAddress) {
		log.Warnf("LemoAddress is incorrect. lemoAddress: %s", LemoAddress)
		return "", ErrLemoAddress
	}
	candiAccount, err := a.GetAccount(LemoAddress)
	if err != nil {
		return "", err
	}
	forAddress := candiAccount.GetVoteFor().String()
	return forAddress, nil
}

// GetAllRewardValue get the value for each bonus
func (a *PublicAccountAPI) GetAllRewardValue() ([]*params.Reward, error) {
	address := params.TermRewardContract
	acc, err := a.GetAccount(address.String())
	if err != nil {
		return nil, err
	}
	key := address.Hash()
	value, err := acc.GetStorageState(key)
	rewardMap := make(params.RewardsMap)
	json.Unmarshal(value, &rewardMap)
	var result = make([]*params.Reward, 0)
	// var maxTerm uint32 = 0
	// for _, v := range rewardMap {
	// 	if v.Term == maxTerm {
	// 		result = append(result, v)
	// 		maxTerm++
	// 	}
	// }
	var i uint32
	for i = 0; ; i++ {
		if v, ok := rewardMap[i]; ok {
			result = append(result, v)
		} else {
			break
		}
	}

	return result, nil
}

// GetAssetEquity returns asset equity
func (a *PublicAccountAPI) GetAssetEquityByAssetId(LemoAddress string, assetId common.Hash) (*types.AssetEquity, error) {
	if !network.VerifyLemoAddress(LemoAddress) {
		log.Warnf("LemoAddress is incorrect. lemoAddress: %s", LemoAddress)
		return nil, ErrLemoAddress
	}
	acc, err := a.GetAccount(LemoAddress)
	if err != nil {
		return nil, err
	}
	return acc.GetEquityState(assetId)
}

//go:generate gencodec -type CandidateInfo -out gen_candidate_info_json.go
type CandidateInfo struct {
	CandidateAddress string            `json:"address" gencodec:"required"`
	Votes            string            `json:"votes" gencodec:"required"`
	Profile          map[string]string `json:"profile"  gencodec:"required"`
}

// ChainAPI
type PublicChainAPI struct {
	chain *chain.BlockChain
}

// NewChainAPI API for access to chain information
func NewPublicChainAPI(chain *chain.BlockChain) *PublicChainAPI {
	return &PublicChainAPI{chain}
}

// GetDeputyNodeList
func (c *PublicChainAPI) GetDeputyNodeList() []string {
	nodes := c.chain.DeputyManager().GetDeputiesByHeight(c.chain.CurrentBlock().Height())

	var result []string
	for _, n := range nodes {
		candidateAcc := c.chain.AccountManager().GetCanonicalAccount(n.MinerAddress)
		profile := candidateAcc.GetCandidate()
		host := profile[types.CandidateKeyHost]
		port := profile[types.CandidateKeyPort]
		nodeAddrString := fmt.Sprintf("%x@%s:%s; votes: %s, rank: %d", n.NodeID, host, port, n.Votes.String(), n.Rank)
		result = append(result, nodeAddrString)
	}
	return result
}

// GetCandidateTop30 get top 30 candidate node
func (c *PublicChainAPI) GetCandidateTop30() []*CandidateInfo {
	latestStableBlock := c.chain.StableBlock()
	stableBlockHash := latestStableBlock.Hash()
	storeInfos := c.chain.GetCandidatesTop(stableBlockHash)
	candidateList := make([]*CandidateInfo, 0, 30)
	for _, info := range storeInfos {
		candidateInfo := &CandidateInfo{
			Profile: make(map[string]string),
		}
		CandidateAddress := info.GetAddress()
		CandidateAccount := c.chain.AccountManager().GetCanonicalAccount(CandidateAddress)
		profile := CandidateAccount.GetCandidate()
		candidateInfo.Profile = profile
		candidateInfo.CandidateAddress = CandidateAddress.String()
		candidateInfo.Votes = info.GetTotal().String()
		candidateList = append(candidateList, candidateInfo)
	}
	return candidateList
}

// GetBlockByNumber get block information by height
func (c *PublicChainAPI) GetBlockByHeight(height uint32, withBody bool) *types.Block {
	if withBody {
		return c.chain.GetBlockByHeight(height)
	} else {
		block := c.chain.GetBlockByHeight(height)
		if block == nil {
			return nil
		}
		// copy only header
		onlyHeaderBlock := &types.Block{
			Header: block.Header,
		}
		return onlyHeaderBlock
	}
}

// GetBlockByHash get block information by hash
func (c *PublicChainAPI) GetBlockByHash(hash string, withBody bool) *types.Block {
	if len(common.FromHex(hash)) != common.HashLength {
		log.Warnf("Hash is incorrect, Hash: %s", hash)
		return nil
	}
	if withBody {
		return c.chain.GetBlockByHash(common.HexToHash(hash))
	} else {
		block := c.chain.GetBlockByHash(common.HexToHash(hash))
		if block == nil {
			return nil
		}
		// copy only header
		onlyHeaderBlock := &types.Block{
			Header: block.Header,
		}
		return onlyHeaderBlock
	}
}

// ChainID get chain id
func (c *PublicChainAPI) ChainID() uint16 {
	return c.chain.ChainID()
}

// Genesis get the creation block
func (c *PublicChainAPI) Genesis() *types.Block {
	return c.chain.Genesis()
}

// CurrentBlock get the latest block. It may not be confirmed by enough deputy nodes
func (c *PublicChainAPI) UnstableBlock(withBody bool) *types.Block {
	block := c.chain.CurrentBlock()
	if !withBody && block != nil {
		// copy only header
		block = &types.Block{
			Header: block.Header,
		}
	}
	return block
}

// CurrentBlock get the latest block.
func (c *PublicChainAPI) CurrentBlock(withBody bool) *types.Block {
	block := c.chain.StableBlock()
	if !withBody && block != nil {
		// copy only header
		block = &types.Block{
			Header: block.Header,
		}
	}
	return block
}

// UnstableHeight
func (c *PublicChainAPI) UnstableHeight() uint32 {
	return c.chain.CurrentBlock().Height()
}

// CurrentHeight
func (c *PublicChainAPI) CurrentHeight() uint32 {
	return c.chain.StableBlock().Height()
}

// NodeVersion
func (n *PublicChainAPI) NodeVersion() string {
	return params.Version
}

// PrivateMineAPI
type PrivateMineAPI struct {
	miner *miner.Miner
}

// NewPrivateMinerAPI
func NewPrivateMinerAPI(miner *miner.Miner) *PrivateMineAPI {
	return &PrivateMineAPI{miner}
}

// MineStart
func (m *PrivateMineAPI) MineStart() {
	m.miner.Start()
}

// MineStop
func (m *PrivateMineAPI) MineStop() {
	m.miner.Stop()
}

// SetLeastGasPrice
func (m *PrivateMineAPI) SetLeastGasPrice(price *big.Int) {
	params.MinGasPrice = price
}

// PublicMineAPI
type PublicMineAPI struct {
	miner *miner.Miner
}

// NewPublicMineAPI
func NewPublicMineAPI(miner *miner.Miner) *PublicMineAPI {
	return &PublicMineAPI{miner}
}

// GetLeastGasPrice
func (m *PublicMineAPI) GetLeastGasPrice() string {
	return params.MinGasPrice.String()
}

// IsMining
func (m *PublicMineAPI) IsMining() bool {
	return m.miner.IsMining()
}

// Miner
func (m *PublicMineAPI) Miner() string {
	address := m.miner.GetMinerAddress()
	return address.String()
}

// PrivateNetAPI
type PrivateNetAPI struct {
	node *Node
}

// NewPrivateNetAPI
func NewPrivateNetAPI(node *Node) *PrivateNetAPI {
	return &PrivateNetAPI{node}
}

// Connect (node = nodeID@IP:Port)
func (n *PrivateNetAPI) Connect(node string) {
	if !network.VerifyNode(node) {
		log.Errorf("The node is incorrect, node: %s", node)
		return
	}
	n.node.server.Connect(node)
}

// Disconnect
func (n *PrivateNetAPI) Disconnect(node string) bool {
	if !network.VerifyNode(node) {
		log.Errorf("The node is incorrect, node: %s", node)
		return false
	}
	return n.node.server.Disconnect(node)
}

// Connections
func (n *PrivateNetAPI) Connections() []p2p.PeerConnInfo {
	return n.node.server.Connections()
}

// PublicNetAPI
type PublicNetAPI struct {
	node *Node
}

// NewPublicNetAPI
func NewPublicNetAPI(node *Node) *PublicNetAPI {
	return &PublicNetAPI{node}
}

// PeersCount return peers number
func (n *PublicNetAPI) PeersCount() string {
	count := strconv.Itoa(len(n.node.server.Connections()))
	return count
}

//go:generate gencodec -type NetInfo --field-override netInfoMarshaling -out gen_net_info_json.go

type NetInfo struct {
	Port     uint32 `json:"port"        gencodec:"required"`
	NodeName string `json:"nodeName"    gencodec:"required"`
	Version  string `json:"nodeVersion" gencodec:"required"`
	OS       string `json:"os"          gencodec:"required"`
	Go       string `json:"runtime"     gencodec:"required"`
}

type netInfoMarshaling struct {
	Port hexutil.Uint32
}

// Info
func (n *PublicNetAPI) Info() *NetInfo {
	return &NetInfo{
		Port:     uint32(n.node.server.Port),
		NodeName: n.node.config.NodeName(),
		Version:  n.node.config.Version,
		OS:       runtime.GOOS + "-" + runtime.GOARCH,
		Go:       runtime.Version(),
	}
}

// NodeID
func (n *PublicNetAPI) NodeID() string {
	return common.ToHex(deputynode.GetSelfNodeID())
}

// TXAPI
type PublicTxAPI struct {
	// txpool *chain.TxPool
	node *Node
}

// NewTxAPI API for send a transaction
func NewPublicTxAPI(node *Node) *PublicTxAPI {
	return &PublicTxAPI{node}
}

// Send send a transaction
func (t *PublicTxAPI) SendTx(tx *types.Transaction) (common.Hash, error) {
	if err := tx.VerifyTxBody(t.node.ChainID(), uint64(time.Now().Unix()), false); err != nil {
		log.Errorf("VerifyTxBody error: %s", err)
		return common.Hash{}, err
	}
	if t.node.txPool.RecvTx(tx) {
		// 广播交易
		go subscribe.Send(subscribe.NewTx, tx)
	}
	return tx.Hash(), nil
}

// PendingTx
func (t *PublicTxAPI) PendingTx(size int) []*types.Transaction {
	return t.node.txPool.Get(uint32(time.Now().Unix()), size)
}

// ReadContract read variables in a contract includes the return value of a function.
func (t *PublicTxAPI) ReadContract(to *common.Address, data hexutil.Bytes) (string, error) {
	if to == nil {
		return "", ErrInputParams
	}
	ctx := context.Background()
	accM := account.NewReadOnlyManager(t.node.Db(), true)
	result, _, err := t.doCallTransaction(ctx, to, accM, params.OrdinaryTx, data, 5*time.Second)
	return common.ToHex(result), err
}

// EstimateGas returns an estimate of the amount of gas needed to execute the given transaction.
func (t *PublicTxAPI) EstimateGas(to *common.Address, txType uint16, data hexutil.Bytes) (string, error) {
	if !types.IsToExist(txType, to) {
		return "", types.ErrToExist
	}
	var costGas uint64
	var err error
	ctx := context.Background()
	accM := account.NewReadOnlyManager(t.node.Db(), false)
	_, costGas, err = t.doCallTransaction(ctx, to, accM, txType, data, 5*time.Second)
	strCostGas := strconv.FormatUint(costGas, 10)
	return strCostGas, err
}

// EstimateContractGas returns an estimate of the amount of gas needed to create a smart contract.
// Todo will delete
func (t *PublicTxAPI) EstimateCreateContractGas(data hexutil.Bytes) (uint64, error) {
	ctx := context.Background()
	accM := account.NewReadOnlyManager(t.node.Db(), false)
	_, costGas, err := t.doCallTransaction(ctx, nil, accM, params.CreateContractTx, data, 5*time.Second)
	return costGas, err
}

// doCallTransaction
func (t *PublicTxAPI) doCallTransaction(ctx context.Context, to *common.Address, accM *account.ReadOnlyManager, txType uint16, data hexutil.Bytes, timeout time.Duration) ([]byte, uint64, error) {
	t.node.lock.Lock()
	defer t.node.lock.Unlock()

	defer func(start time.Time) { log.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())
	// get current stableBlock
	currentBlock := t.node.chain.CurrentBlock()
	log.Infof("Current block height = %v", currentBlock.Height())
	currentHeader := currentBlock.Header
	p := t.node.chain.TxProcessor()
	ret, costGas, err := p.PreExecutionTransaction(ctx, accM, currentHeader, to, txType, data, common.Hash{}, timeout)

	return ret, costGas, err
}
