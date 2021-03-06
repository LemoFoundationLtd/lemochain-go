package consensus

import (
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/chain/account"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/transaction"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/crypto"
	"github.com/LemoFoundationLtd/lemochain-core/common/merkle"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

func TestNewBlockAssembler(t *testing.T) {
	dm := deputynode.NewManager(5, &testBlockLoader{})

	ba := NewBlockAssembler(nil, dm, &transaction.TxProcessor{}, createCandidateLoader())
	assert.Equal(t, dm, ba.dm)
}

func TestBlockAssembler_PrepareHeader(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	dm := initDeputyManager(5)

	ba := NewBlockAssembler(am, dm, &transaction.TxProcessor{}, createCandidateLoader())
	parentHeader := &types.Header{Height: 100, Time: 1001}

	// I'm a miner
	now := uint32(time.Now().Unix())
	newHeader, err := ba.PrepareHeader(parentHeader, "abc")
	assert.NoError(t, err)
	assert.Equal(t, parentHeader.Hash(), newHeader.ParentHash)
	assert.Equal(t, testDeputies[0].MinerAddress, newHeader.MinerAddress)
	assert.Equal(t, parentHeader.Height+1, newHeader.Height)
	assert.Equal(t, params.GenesisGasLimit, newHeader.GasLimit)
	assert.Equal(t, "abc", newHeader.Extra)
	assert.Equal(t, true, newHeader.Time >= now)

	// I'm not miner
	privateBackup := deputynode.GetSelfNodeKey()
	private, err := crypto.GenerateKey()
	assert.NoError(t, err)
	deputynode.SetSelfNodeKey(private)
	_, err = ba.PrepareHeader(parentHeader, "")
	assert.Equal(t, ErrNotDeputy, err)
	deputynode.SetSelfNodeKey(privateBackup)
}

func TestCalcGasLimit(t *testing.T) {
	parentHeader := &types.Header{GasLimit: 0, GasUsed: 0}
	limit := calcGasLimit(parentHeader)
	assert.Equal(t, true, limit > params.MinGasLimit)
	assert.Equal(t, params.TargetGasLimit, limit)

	parentHeader = &types.Header{GasLimit: params.TargetGasLimit + 100000000000, GasUsed: params.TargetGasLimit + 100000000000}
	limit = calcGasLimit(parentHeader)
	assert.Equal(t, true, limit > params.MinGasLimit)
	assert.Equal(t, true, limit > params.TargetGasLimit)

	parentHeader = &types.Header{GasLimit: params.TargetGasLimit + 100000000000, GasUsed: 0}
	limit = calcGasLimit(parentHeader)
	assert.Equal(t, true, limit > params.MinGasLimit)
	assert.Equal(t, true, limit > params.TargetGasLimit)
}

func TestBlockAssembler_Seal(t *testing.T) {
	canLoader := createCandidateLoader(0, 1, 3)
	ba := NewBlockAssembler(nil, nil, nil, canLoader)

	// not snapshot block
	header := &types.Header{Height: 100}
	txs := []*types.Transaction{
		types.NewTransaction(common.HexToAddress("0x123"), common.Address{}, common.Big1, 2000000, common.Big2, []byte{12}, 0, 100, 1538210391, "aa", "aaa"),
		{},
	}
	product := &account.TxsProduct{
		Txs:         txs,
		GasUsed:     123,
		ChangeLogs:  types.ChangeLogSlice{{LogType: 101}, {}},
		VersionRoot: common.HexToHash("0x124"),
	}
	confirms := []types.SignData{{0x12}, {0x34}}
	block01 := ba.Seal(header, product, confirms)
	assert.Equal(t, product.VersionRoot, block01.VersionRoot())
	assert.Equal(t, product.ChangeLogs.MerkleRootSha(), block01.LogRoot())
	assert.Equal(t, product.Txs.MerkleRootSha(), block01.TxRoot())
	assert.Equal(t, product.GasUsed, block01.GasUsed())
	assert.Equal(t, product.Txs, block01.Txs)
	assert.Equal(t, product.ChangeLogs, block01.ChangeLogs)
	assert.Equal(t, confirms, block01.Confirms)

	// snapshot block
	header = &types.Header{Height: params.TermDuration}
	product = &account.TxsProduct{}
	confirms = []types.SignData{}
	block02 := ba.Seal(header, product, confirms)
	assert.NotEqual(t, block01.Hash(), block02.Hash())
	deputies := types.DeputyNodes(canLoader)
	assert.Equal(t, deputies, block02.DeputyNodes)
	deputyRoot := deputies.MerkleRootSha()
	assert.Equal(t, deputyRoot[:], block02.DeputyRoot())
	assert.Equal(t, merkle.EmptyTrieHash, block02.LogRoot())
	assert.Equal(t, merkle.EmptyTrieHash, block02.TxRoot())
	assert.Equal(t, uint64(0), block02.GasUsed())
	assert.Empty(t, block02.Txs)
	assert.Empty(t, block02.ChangeLogs)
	assert.Empty(t, block02.Confirms)
}

// createAssembler clear account manager then make some new change logs
func createAssembler(db *store.ChainDatabase, fillSomeLogs bool) *BlockAssembler {
	am := account.NewManager(common.Hash{}, db)

	deputyCount := 5
	dm := deputynode.NewManager(deputyCount, db)
	deputies := generateDeputies(deputyCount).ToDeputyNodes()
	dm.SaveSnapshot(0, deputies)
	dm.SaveSnapshot(params.TermDuration, deputies)

	if fillSomeLogs {
		// This will make 5 CandidateLogs and 5 VotesLogs
		for _, deputy := range deputies {
			profile := types.Profile{
				types.CandidateKeyIsCandidate:   types.IsCandidateNode,
				types.CandidateKeyDepositAmount: "100",
			}
			deputyAccount := am.GetAccount(deputy.MinerAddress)
			deputyAccount.SetCandidate(profile)
			deputyAccount.SetVotes(deputy.Votes)
		}
		// set reward pool balance. It will make 1 BalanceLog
		am.GetAccount(params.DepositPoolAddress).SetBalance(big.NewInt(int64(100 * deputyCount)))

		// make balance logs. It will make 3 BalanceLogs, but they could be merged to 2 logs
		account1 := am.GetAccount(common.HexToAddress("0x123"))
		account1.SetBalance(big.NewInt(10000))
		account1.SetBalance(big.NewInt(9999))
		account2 := am.GetAccount(common.HexToAddress("234"))
		account2.SetBalance(big.NewInt(100))

		// set reward for term 0. It will make 1 StorageLog. And 1 StorageRootLog after accountManager.Finalise()
		rewardAccont := am.GetAccount(params.TermRewardContract)
		rewardsMap := params.RewardsMap{
			0: &params.Reward{Term: 0, Value: common.Lemo2Mo("10000")},
			1: &params.Reward{Term: 1, Value: common.Lemo2Mo("0")},
		}
		storageVal, err := json.Marshal(rewardsMap)
		if err != nil {
			panic(err)
		}
		err = rewardAccont.SetStorageState(params.TermRewardContract.Hash(), storageVal)
		if err != nil {
			panic(err)
		}

		// unregister first deputy It will make 1 CandidateStateLog
		deputy := deputies[0]
		am.GetAccount(deputy.MinerAddress).SetCandidateState(types.CandidateKeyIsCandidate, types.NotCandidateNode)
	}

	processor := transaction.NewTxProcessor(deputies[0].MinerAddress, 100, &parentLoader{db}, am, db, dm)
	return NewBlockAssembler(am, dm, processor, testCandidateLoader{deputies[0]})
}

func TestBlockAssembler_Finalize(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()

	finalizeAndCountLogs := func(ba *BlockAssembler, height uint32, logsCount int) {
		err := ba.Finalize(height) // genesis block
		assert.NoError(t, err)
		logs := ba.am.GetChangeLogs()
		assert.Equal(t, logsCount, len(logs))
	}
	checkReward := func(am *account.Manager, dm *deputynode.Manager) {
		deputy := dm.GetDeputiesByHeight(1, true)[1]
		deputyAccount := am.GetAccount(deputy.MinerAddress)
		assert.Equal(t, -1, big.NewInt(0).Cmp(deputyAccount.GetBalance()))
	}
	checkRefund := func(am *account.Manager, dm *deputynode.Manager) {
		deputy := dm.GetDeputiesByHeight(1, true)[0]
		deputyAccount := am.GetAccount(deputy.MinerAddress)
		assert.Equal(t, types.NotCandidateNode, deputyAccount.GetCandidateState(types.CandidateKeyIsCandidate))
		assert.Equal(t, "", deputyAccount.GetCandidateState(types.CandidateKeyDepositAmount))
		assert.Equal(t, -1, big.NewInt(0).Cmp(deputyAccount.GetVotes()))
	}

	// nothing to finalize and no reward
	ba := createAssembler(db, false)
	finalizeAndCountLogs(ba, 0, 0) // genesis block
	ba = createAssembler(db, false)
	finalizeAndCountLogs(ba, 1, 0) // normal block

	// many change logs
	ba = createAssembler(db, true)
	finalizeAndCountLogs(ba, 0, 16) // genesis block
	ba = createAssembler(db, true)
	finalizeAndCountLogs(ba, 1, 16) // normal block
	ba = createAssembler(db, true)
	finalizeAndCountLogs(ba, params.TermDuration+params.InterimDuration+1-params.RewardCheckHeight, 16) // check reward block
	ba = createAssembler(db, true)
	finalizeAndCountLogs(ba, params.TermDuration, 16) // snapshot block
	ba = createAssembler(db, true)
	finalizeAndCountLogs(ba, params.TermDuration+params.InterimDuration+1, 22) // reward and deposit refund block
	checkReward(ba.am, ba.dm)
	checkRefund(ba.am, ba.dm)
	ba = createAssembler(db, true)
	finalizeAndCountLogs(ba, params.TermDuration*2-100, 16) // 2st term normal block
	ba = createAssembler(db, true)
	finalizeAndCountLogs(ba, params.TermDuration*2+params.InterimDuration+1, 18) // 2st term reward block
	checkRefund(ba.am, ba.dm)
}

type errCanLoader struct {
}

func (cl errCanLoader) LoadTopCandidates(blockHash common.Hash) types.DeputyNodes {
	return types.DeputyNodes{}
}

func (cl errCanLoader) LoadRefundCandidates(height uint32) ([]common.Address, error) {
	return []common.Address{}, errors.New("refund error")
}

func TestBlockAssembler_Finalize2(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()

	// issue term reward failed
	ba := createAssembler(db, true)
	rewardAccont := ba.am.GetAccount(params.TermRewardContract)
	err := rewardAccont.SetStorageState(params.TermRewardContract.Hash(), []byte{0x12})
	assert.NoError(t, err)
	err = ba.Finalize(params.TermDuration + params.InterimDuration + 1)
	assert.EqualError(t, err, "invalid character '\\x12' looking for beginning of value")

	// refund deposit failed
	ba = createAssembler(db, true)
	ba.canLoader = errCanLoader{}
	err = ba.Finalize(params.TermDuration + params.InterimDuration + 1)
	assert.Equal(t, errors.New("refund error"), err)

	// account manager finalise failed
	ba = createAssembler(db, true)
	storageRootLog := &types.ChangeLog{
		Version: 1,
		Address: params.TermRewardContract,
		LogType: account.StorageRootLog,
		NewVal:  common.HexToHash("0x12"),
	}
	err = ba.am.Rebuild(params.TermRewardContract, types.ChangeLogSlice{storageRootLog}) // break the storage root
	assert.NoError(t, err)
	err = ba.Finalize(1)
	assert.Equal(t, types.ErrTrieFail, err)
}

func TestCheckTermReward(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	rewardAccont := am.GetAccount(params.TermRewardContract)

	// no reward and not the height for checking
	dm := deputynode.NewManager(5, db)
	ba := NewBlockAssembler(am, dm, &transaction.TxProcessor{}, testCandidateLoader{})
	assert.Equal(t, true, ba.checkTermReward(0))

	// no reward and the height for checking
	assert.Equal(t, false, ba.checkTermReward(params.TermDuration+params.InterimDuration+1-params.RewardCheckHeight))

	// set rewards
	rewardAmount := common.Lemo2Mo("100")
	rewardsMap := params.RewardsMap{0: &params.Reward{Term: 0, Value: rewardAmount}}
	storageVal, err := json.Marshal(rewardsMap)
	assert.NoError(t, err)
	err = rewardAccont.SetStorageState(params.TermRewardContract.Hash(), storageVal)
	assert.NoError(t, err)
	assert.Equal(t, true, ba.checkTermReward(params.TermDuration+params.InterimDuration+1-params.RewardCheckHeight))

	// ignore getTermRewards error
	// set invalid reward
	err = rewardAccont.SetStorageState(params.TermRewardContract.Hash(), []byte{0x12})
	assert.NoError(t, err)
	assert.Equal(t, true, ba.checkTermReward(params.TermDuration+params.InterimDuration+1-params.RewardCheckHeight))
	am.Reset(common.Hash{})
	rewardAccont = am.GetAccount(params.TermRewardContract)
}

func TestIssueTermReward(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()
	ba := createAssembler(db, false)
	firstTerm, err := ba.dm.GetTermByHeight(1, true)
	assert.NoError(t, err)
	rewardAmount := common.Lemo2Mo("100")
	// choose a miner
	minerAddress1 := firstTerm.Nodes[1].MinerAddress

	initRewardData := func() {
		ba.am.Reset(common.Hash{})
		rewardAccont := ba.am.GetAccount(params.TermRewardContract)
		rewardsMap := params.RewardsMap{0: &params.Reward{Term: 0, Value: rewardAmount}}
		storageVal, err := json.Marshal(rewardsMap)
		assert.NoError(t, err)
		err = rewardAccont.SetStorageState(params.TermRewardContract.Hash(), storageVal)
		assert.NoError(t, err)
	}

	// not reward block
	initRewardData()
	err = ba.issueTermReward(ba.am, params.TermDuration)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(0), ba.am.GetAccount(minerAddress1).GetBalance())

	// reward block
	initRewardData()
	err = ba.issueTermReward(ba.am, params.TermDuration+params.InterimDuration+1)
	assert.NoError(t, err)
	deputyCount := len(firstTerm.Nodes)
	assert.Equal(t, big.NewInt(0).Div(rewardAmount, big.NewInt(int64(deputyCount))), ba.am.GetAccount(minerAddress1).GetBalance())

	// not set reward yet
	initRewardData()
	err = ba.issueTermReward(ba.am, params.TermDuration*2+params.InterimDuration+1)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(0), ba.am.GetAccount(minerAddress1).GetBalance())
}

func TestRefundCandidateDeposit(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()
	ba := createAssembler(db, false)
	firstTerm, err := ba.dm.GetTermByHeight(1, true)
	assert.NoError(t, err)
	depositAmount := common.Lemo2Mo("100")
	// refund account is set in createAssembler
	refundAddress := firstTerm.Nodes[0].MinerAddress
	initDepositData := func() {
		ba.am.Reset(common.Hash{})
		ba.am.GetAccount(refundAddress).SetCandidateState(types.CandidateKeyDepositAmount, depositAmount.String())
		depositPoolAccount := ba.am.GetAccount(params.DepositPoolAddress)
		depositPoolAccount.SetBalance(depositAmount)
	}

	// not reward block
	initDepositData()
	err = ba.refundCandidateDeposit(ba.am, params.TermDuration)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(0), ba.am.GetAccount(refundAddress).GetBalance())

	// not set reward yet
	initDepositData()
	err = ba.refundCandidateDeposit(ba.am, params.TermDuration*2+params.InterimDuration+1)
	assert.NoError(t, err)
	assert.Equal(t, depositAmount, ba.am.GetAccount(refundAddress).GetBalance())

	// reward block
	initDepositData()
	err = ba.refundCandidateDeposit(ba.am, params.TermDuration+params.InterimDuration+1)
	assert.NoError(t, err)
	assert.Equal(t, depositAmount, ba.am.GetAccount(refundAddress).GetBalance())
}

func TestCalculateSalary(t *testing.T) {
	tests := []struct {
		Expect, TotalSalary, DeputyVotes, TotalVotes, Precision, deputyNodesNum int64
	}{
		// total votes=100
		{0, 100, 0, 100, 1, 5},
		{1, 100, 1, 100, 1, 5},
		{2, 100, 2, 100, 1, 5},
		{100, 100, 100, 100, 1, 5},
		// total votes=100, precision=10
		{0, 100, 1, 100, 10, 5},
		{10, 100, 10, 100, 10, 5},
		{10, 100, 11, 100, 10, 5},
		// total votes=1000
		{0, 100, 1, 1000, 1, 5},
		{0, 100, 9, 1000, 1, 5},
		{1, 100, 10, 1000, 1, 5},
		{1, 100, 11, 1000, 1, 5},
		{100, 100, 1000, 1000, 1, 5},
		// total votes=1000, precision=10
		{10, 100, 100, 1000, 10, 5},
		{10, 100, 120, 1000, 10, 5},
		{20, 100, 280, 1000, 10, 5},
		// total votes=10
		{0, 100, 0, 10, 1, 5},
		{10, 100, 1, 10, 1, 5},
		{100, 100, 10, 10, 1, 5},
		// total votes=10, precision=10
		{10, 100, 1, 10, 10, 5},
		{100, 100, 10, 10, 10, 5},
		// total votes = 0
		{100, 100, 0, 0, 1, 1},
		{50, 100, 0, 0, 1, 2},
		{33, 100, 0, 0, 1, 3},
	}
	for _, test := range tests {
		expect := big.NewInt(test.Expect)
		totalSalary := big.NewInt(test.TotalSalary)
		deputyVotes := big.NewInt(test.DeputyVotes)
		totalVotes := big.NewInt(test.TotalVotes)
		precision := big.NewInt(test.Precision)
		nodesNum := int(test.deputyNodesNum)
		assert.Equalf(t, 0, calculateSalary(totalSalary, deputyVotes, totalVotes, precision, nodesNum).Cmp(expect), "calculateSalary(%v, %v, %v, %v)", totalSalary, deputyVotes, totalVotes, precision)
	}
}

func TestGetDeputyIncomeAddress(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)

	// no set income address
	minerAddr := common.HexToAddress("0x123")
	deputy := &types.DeputyNode{MinerAddress: minerAddr}
	incomeAddr := getDeputyIncomeAddress(am, deputy)
	assert.Equal(t, minerAddr, incomeAddr)

	// invalid income address
	candidate := am.GetAccount(minerAddr)
	candidate.SetCandidateState(types.CandidateKeyIncomeAddress, "random text")
	incomeAddr = getDeputyIncomeAddress(am, deputy)
	assert.Equal(t, minerAddr, incomeAddr)

	// valid income address
	incomeAddrStr := "lemoqr"
	validIncomeAddr, _ := common.StringToAddress(incomeAddrStr)
	candidate.SetCandidateState(types.CandidateKeyIncomeAddress, incomeAddrStr)
	incomeAddr = getDeputyIncomeAddress(am, deputy)
	assert.Equal(t, validIncomeAddr, incomeAddr)
}

// TestDivideSalary test total salary with random data
func TestDivideSalary(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)

	r := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < 100; i++ {
		nodeCount := r.Intn(49) + 1 // [1, 50]
		nodes := generateDeputies(nodeCount).ToDeputyNodes()
		registerDeputies(nodes, am)
		for _, node := range nodes {
			node.Votes = randomBigInt(r)
		}

		totalSalary := randomBigInt(r)
		term := &deputynode.TermRecord{TermIndex: 0, Nodes: nodes}

		salaries := DivideSalary(totalSalary, am, term)
		assert.Len(t, salaries, nodeCount)

		// 验证income是否相同
		for j := 0; j < len(nodes); j++ {
			if getDeputyIncomeAddress(am, nodes[j]) != salaries[j].Address {
				panic("income address no equal")
			}
		}
		actualTotal := new(big.Int)
		for _, s := range salaries {
			actualTotal.Add(actualTotal, s.Salary)
		}
		totalVotes := new(big.Int)
		for _, v := range nodes {
			totalVotes.Add(totalVotes, v.Votes)
		}
		// 比较每个deputy node salary
		for k := 0; k < len(nodes); k++ {
			if salaries[k].Salary.Cmp(calculateSalary(totalSalary, nodes[k].Votes, totalVotes, params.MinRewardPrecision, len(nodes))) != 0 {
				panic("deputy node salary no equal")
			}
		}

		// errRange = nodeCount * minPrecision
		// actualTotal must be in range [totalSalary - errRange, totalSalary]
		errRange := new(big.Int).Mul(big.NewInt(int64(nodeCount)), params.MinRewardPrecision)
		assert.Equal(t, true, actualTotal.Cmp(new(big.Int).Sub(totalSalary, errRange)) >= 0)
		assert.Equal(t, true, actualTotal.Cmp(totalSalary) <= 0)
	}
	// 测试所有的votes都为0的情况
	nodes := generateDeputies(5).ToDeputyNodes()
	// 设置nodes的votes为0
	for _, node := range nodes {
		node.Votes = big.NewInt(0)
	}
	registerDeputies(nodes, am)
	term := &deputynode.TermRecord{TermIndex: 0, Nodes: nodes}
	salarySlice := DivideSalary(common.Lemo2Mo("10004"), am, term)
	for _, salary := range salarySlice {
		// 均分10004LEMO,奖励最小值限制为1LEMO，所以均分后的奖励为2000LEMO
		assert.Equal(t, salary.Salary, common.Lemo2Mo("2000"))
	}
}

func registerDeputies(deputies types.DeputyNodes, am *account.Manager) {
	for _, node := range deputies {
		profile := make(map[string]string)
		// 设置deputy node 的income address
		minerAcc := am.GetAccount(node.MinerAddress)
		// 设置income address 为minerAddress
		profile[types.CandidateKeyIncomeAddress] = node.MinerAddress.String()
		minerAcc.SetCandidate(profile)
	}
}

func randomBigInt(r *rand.Rand) *big.Int {
	return new(big.Int).Mul(big.NewInt(r.Int63()), big.NewInt(r.Int63()))
}

func TestGetTermRewards(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()
	am := account.NewManager(common.Hash{}, db)
	rewardAccont := am.GetAccount(params.TermRewardContract)

	// no reward
	result, err := getTermRewards(am, 0)
	assert.NoError(t, err)
	assert.Equal(t, *big.NewInt(0), *result)

	// load fail (account storage root is invalid)
	storageRootLog := &types.ChangeLog{
		Version: 1,
		Address: params.TermRewardContract,
		LogType: account.StorageRootLog,
		NewVal:  common.HexToHash("0x12"),
	}
	err = am.Rebuild(params.TermRewardContract, types.ChangeLogSlice{storageRootLog})
	assert.NoError(t, err)
	_, err = getTermRewards(am, 0)
	assert.Equal(t, types.ErrTrieFail, err)
	am.Reset(common.Hash{})
	rewardAccont = am.GetAccount(params.TermRewardContract)

	// set invalid reward
	err = rewardAccont.SetStorageState(params.TermRewardContract.Hash(), []byte{0x12})
	assert.NoError(t, err)
	result, err = getTermRewards(am, 0)
	assert.EqualError(t, err, "invalid character '\\x12' looking for beginning of value")
	am.Reset(common.Hash{})
	rewardAccont = am.GetAccount(params.TermRewardContract)

	// set rewards
	rewardAmount := common.Lemo2Mo("100")
	rewardsMap := params.RewardsMap{0: &params.Reward{Term: 0, Value: rewardAmount}}
	storageVal, err := json.Marshal(rewardsMap)
	assert.NoError(t, err)
	err = rewardAccont.SetStorageState(params.TermRewardContract.Hash(), storageVal)
	assert.NoError(t, err)
	result, err = getTermRewards(am, 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, rewardAmount.Cmp(result))
	// set rewards but not the term
	result, err = getTermRewards(am, 1)
	assert.NoError(t, err)
	assert.Equal(t, 0, big.NewInt(0).Cmp(result))
	am.Reset(common.Hash{})
	rewardAccont = am.GetAccount(params.TermRewardContract)

	// set empty rewards
	storageVal, err = json.Marshal(params.RewardsMap{})
	assert.NoError(t, err)
	err = rewardAccont.SetStorageState(params.TermRewardContract.Hash(), storageVal)
	assert.NoError(t, err)
	result, err = getTermRewards(am, 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, big.NewInt(0).Cmp(result))
}

func TestBlockAssembler_RunBlock(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()
	ba := createAssembler(db, false)
	firstTerm, err := ba.dm.GetTermByHeight(1, true)
	assert.NoError(t, err)
	tx := MakeTx(deputynode.GetSelfNodeKey(), common.HexToAddress("0x88"), common.Lemo2Mo("100"), uint64(time.Now().Unix()+300))
	tx.SetGasUsed(21204)

	// genesis block
	rawBlock := &types.Block{Header: &types.Header{Height: 0}, Txs: types.Transactions{}}
	assert.PanicsWithValue(t, transaction.ErrInvalidGenesis, func() {
		_, _ = ba.RunBlock(rawBlock)
	})

	// prepare a genesis block (and balance) for test
	genesisBlock := initGenesis(db, ba.am, firstTerm.Nodes)

	// process block fail
	rawBlock = &types.Block{Header: &types.Header{Height: 1}, Txs: types.Transactions{tx}}
	_, err = ba.RunBlock(rawBlock)
	assert.Equal(t, transaction.ErrInvalidTxInBlock, err)

	// process block success
	rawBlock = &types.Block{Header: &types.Header{Height: 1, ParentHash: genesisBlock.Hash(), GasLimit: 10000000, MinerAddress: firstTerm.Nodes[0].MinerAddress}, Txs: types.Transactions{tx}}
	newBlock, err := ba.RunBlock(rawBlock)
	assert.NoError(t, err)
	assert.Equal(t, rawBlock.Txs, newBlock.Txs)
	assert.Equal(t, rawBlock.MinerAddress(), newBlock.MinerAddress())
	assert.Equal(t, uint64(21204), newBlock.GasUsed())
	assert.NotEqual(t, 0, len(newBlock.ChangeLogs))
}

func TestBlockAssembler_MineBlock(t *testing.T) {
	ClearData()
	db := store.NewChainDataBase(GetStorePath())
	defer db.Close()
	ba := createAssembler(db, false)
	firstTerm, err := ba.dm.GetTermByHeight(1, true)
	assert.NoError(t, err)
	genesisBlock := initGenesis(db, ba.am, firstTerm.Nodes)

	tx := MakeTx(deputynode.GetSelfNodeKey(), common.HexToAddress("0x88"), common.Lemo2Mo("100"), uint64(time.Now().Unix()+300))
	invalidTx := MakeTx(deputynode.GetSelfNodeKey(), common.HexToAddress("0x88"), common.Lemo2Mo("100000000000000"), uint64(time.Now().Unix()+300))

	header := &types.Header{Height: 1, ParentHash: genesisBlock.Hash(), GasLimit: 10000000, MinerAddress: firstTerm.Nodes[0].MinerAddress}
	txs := types.Transactions{tx, invalidTx}
	newBlock, invalidTxs, err := ba.MineBlock(header, txs, 1000)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(newBlock.Txs))
	assert.Equal(t, 1, len(invalidTxs))
	assert.NotEqual(t, nil, newBlock.Header.SignData)
}
