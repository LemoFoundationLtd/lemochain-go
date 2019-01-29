package store

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/stretchr/testify/assert"
	"testing"
	// "github.com/LemoFoundationLtd/lemochain-go/common/log"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"strconv"
)

func TestCacheChain_SetBlock(t *testing.T) {
	ClearData()

	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	result, err := cacheChain.GetBlockByHeight(0)
	assert.Equal(t, err, ErrNotExist)

	//
	block0 := GetBlock0()
	err = cacheChain.SetBlock(block0.Hash(), block0)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(block0.Hash())
	assert.NoError(t, err)

	result, err = cacheChain.GetBlockByHash(block0.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block0.ParentHash(), result.ParentHash())

	//
	block1 := GetBlock1()
	err = cacheChain.SetBlock(block1.Hash(), block1)
	assert.NoError(t, err)

	result, err = cacheChain.GetBlockByHash(block1.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block1.ParentHash(), result.ParentHash())

	result, err = cacheChain.GetBlockByHeight(block1.Height())
	assert.Equal(t, err, ErrNotExist)

	block2 := GetBlock2()
	err = cacheChain.SetBlock(block2.Hash(), block2)
	assert.NoError(t, err)

	result, err = cacheChain.GetBlockByHash(block2.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block2.ParentHash(), result.ParentHash())

	result, err = cacheChain.GetBlockByHeight(block2.Height())
	assert.Equal(t, err, ErrNotExist)

	block3 := GetBlock3()
	err = cacheChain.SetBlock(block3.Hash(), block3)
	assert.NoError(t, err)

	result, err = cacheChain.GetBlockByHash(block3.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block3.ParentHash(), result.ParentHash())

	result, err = cacheChain.GetBlockByHeight(block3.Height())
	assert.Equal(t, err, ErrNotExist)

	//
	err = cacheChain.SetStableBlock(block2.Hash())
	assert.NoError(t, err)

	result, err = cacheChain.GetBlockByHash(block2.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block2.ParentHash(), result.ParentHash())

	result, err = cacheChain.GetBlockByHeight(block2.Height())
	assert.NoError(t, err)
	assert.Equal(t, block2.ParentHash(), result.ParentHash())

	result, err = cacheChain.GetBlockByHash(block3.Hash())
	assert.NoError(t, err)
	assert.Equal(t, block3.ParentHash(), result.ParentHash())

	result, err = cacheChain.GetBlockByHeight(block3.Height())
	assert.Equal(t, err, ErrNotExist)
}

func TestCacheChain_SetBlockError(t *testing.T) {
	ClearData()

	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	block1 := GetBlock1()

	assert.PanicsWithValue(t, "(database.LastConfirm.Block == nil) && (block.Height() != 0) && (block.ParentHash() != common.Hash{})", func() {
		cacheChain.SetBlock(block1.Hash(), block1)
	})

	block0 := GetBlock0()
	cacheChain.SetBlock(block0.Hash(), block0)
	cacheChain.SetStableBlock(block0.Hash())

	err := cacheChain.SetBlock(block0.Hash(), block0)
	assert.Equal(t, ErrExist, err)

	block1.Header.Height = 0
	assert.PanicsWithValue(t, "(block.Height() == 0) || (block.ParentHash() == common.Hash{})", func() {
		cacheChain.SetBlock(block1.Hash(), block1)
	})
	block1.Header.Height = 1
	cacheChain.SetBlock(block1.Hash(), block1)
	cacheChain.SetStableBlock(block1.Hash())

	block2 := GetBlock2()
	block2.Header.Height = 1
	assert.PanicsWithValue(t, "(database.LastConfirm.Block != nil) && (height < database.LastConfirm.Block.Height())", func() {
		cacheChain.SetBlock(block2.Hash(), block2)
	})

	block4 := GetBlock4()
	assert.PanicsWithValue(t, "parent block is not exist.", func() {
		cacheChain.SetBlock(block4.Hash(), block4)
	})
}

func TestCacheChain_IsExistByHash(t *testing.T) {
	ClearData()

	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	isExist, err := cacheChain.IsExistByHash(common.Hash{})
	assert.NoError(t, err)
	assert.Equal(t, false, isExist)

	parentBlock := GetBlock0()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)

	isExist, err = cacheChain.IsExistByHash(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, true, isExist)

	err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, true, isExist)
}

func TestCacheChain_WriteChain1(t *testing.T) {
	ClearData()

	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	parentBlock := GetBlock0()
	err := cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)

	childBlock := GetBlock1()
	err = cacheChain.SetBlock(childBlock.Hash(), childBlock)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(childBlock.Hash())
	assert.NoError(t, err)

	result, err := cacheChain.GetBlockByHash(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), parentBlock.Hash())

	result, err = cacheChain.GetBlockByHash(childBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), childBlock.Hash())
}

func TestCacheChain_WriteChain2(t *testing.T) {
	ClearData()

	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	parentBlock := GetBlock0()
	err := cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)

	childBlock := GetBlock1()
	err = cacheChain.SetBlock(childBlock.Hash(), childBlock)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(childBlock.Hash())
	assert.NoError(t, err)

	result, err := cacheChain.GetBlockByHash(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), parentBlock.Hash())

	result, err = cacheChain.GetBlockByHash(childBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, result.Hash(), childBlock.Hash())
}

func TestCacheChain_repeat(t *testing.T) {
	ClearData()

	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	parentBlock := GetBlock0()
	err := cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)

	childBlock := GetBlock1()
	err = cacheChain.SetBlock(childBlock.Hash(), childBlock)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(childBlock.Hash())
	assert.NoError(t, err)

	//
	parentBlock = GetBlock0()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.Equal(t, err, ErrExist)

	childBlock = GetBlock1()
	err = cacheChain.SetBlock(childBlock.Hash(), childBlock)
	assert.Equal(t, err, ErrExist)
}

func TestCacheChain_Load4Hit(t *testing.T) {
	ClearData()

	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	block := GetBlock0()
	err := cacheChain.SetBlock(block.Hash(), block)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(block.Hash())
	assert.NoError(t, err)

	block = GetBlock1()
	err = cacheChain.SetBlock(block.Hash(), block)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(block.Hash())
	assert.NoError(t, err)

	cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	parentBlock := GetBlock0()

	result, err := cacheChain.GetBlockByHash(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, parentBlock.ParentHash(), result.ParentHash())

	//
	childBlock := GetBlock1()
	result, err = cacheChain.GetBlockByHash(childBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, childBlock.ParentHash(), parentBlock.Hash())
}

func TestChainDatabase_SetContractCode(t *testing.T) {
	ClearData()

	database := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	code := types.Code("this  is code")
	hash := common.HexToHash("this is code")

	err := database.SetContractCode(hash, code)
	assert.NoError(t, err)

	result, err := database.GetContractCode(hash)
	assert.NoError(t, err)
	assert.Equal(t, code, result)
}

func TestCacheChain_LoadLatestBlock1(t *testing.T) {
	ClearData()

	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	result, err := cacheChain.LoadLatestBlock()
	assert.Equal(t, err, ErrNotExist)
	assert.Nil(t, result)

	parentBlock := GetBlock0()
	childBlock := GetBlock1()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)
	err = cacheChain.SetBlock(childBlock.Hash(), childBlock)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.NoError(t, err)

	result, err = cacheChain.LoadLatestBlock()
	assert.NoError(t, err)
	assert.Equal(t, parentBlock.ParentHash(), result.ParentHash())

	err = cacheChain.SetStableBlock(childBlock.Hash())
	assert.NoError(t, err)

	result, err = cacheChain.LoadLatestBlock()
	assert.NoError(t, err)
	assert.Equal(t, childBlock.ParentHash(), result.ParentHash())

	cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	result, err = cacheChain.LoadLatestBlock()
	assert.NoError(t, err)
	assert.Equal(t, childBlock.ParentHash(), result.ParentHash())
}

func TestCacheChain_LoadLatestBlock2(t *testing.T) {
	ClearData()

	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	parentBlock := GetBlock0()
	childBlock := GetBlock1()
	err := cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)
	err = cacheChain.SetBlock(childBlock.Hash(), childBlock)
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(childBlock.Hash())
	assert.NoError(t, err)

	result, err := cacheChain.LoadLatestBlock()
	assert.NoError(t, err)
	assert.Equal(t, childBlock.ParentHash(), result.ParentHash())

	err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.Equal(t, err, ErrNotExist)

	result, err = cacheChain.LoadLatestBlock()
	assert.NoError(t, err)
	assert.Equal(t, childBlock.ParentHash(), result.ParentHash())
}

func TestCacheChain_SetConfirm1(t *testing.T) {
	ClearData()

	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	signs, err := CreateSign(16)
	assert.NoError(t, err)

	parentBlock := GetBlock0()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)
	err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.NoError(t, err)
	err = cacheChain.SetConfirms(parentBlock.Hash(), signs)
	assert.NoError(t, err)

	result, err := cacheChain.GetConfirms(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 16, len(result))
	assert.Equal(t, signs[0], result[0])
	assert.Equal(t, signs[1], result[1])
	assert.Equal(t, signs[2], result[2])
	assert.Equal(t, signs[3], result[3])

	//cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	result, err = cacheChain.GetConfirms(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 16, len(result))
	assert.Equal(t, signs[0], result[0])
	assert.Equal(t, signs[1], result[1])
	assert.Equal(t, signs[2], result[2])
	assert.Equal(t, signs[3], result[3])
}

func TestCacheChain_SetConfirm2(t *testing.T) {
	ClearData()

	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	signs, err := CreateSign(16)
	assert.NoError(t, err)

	parentBlock := GetBlock0()
	err = cacheChain.SetConfirm(parentBlock.Hash(), signs[0])
	assert.Equal(t, err, ErrNotExist)

	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)
	err = cacheChain.SetConfirm(parentBlock.Hash(), signs[0])
	assert.NoError(t, err)
	err = cacheChain.SetConfirm(parentBlock.Hash(), signs[1])
	assert.NoError(t, err)
	err = cacheChain.SetConfirm(parentBlock.Hash(), signs[2])
	assert.NoError(t, err)
	err = cacheChain.SetConfirm(parentBlock.Hash(), signs[3])
	assert.NoError(t, err)

	result, err := cacheChain.GetConfirms(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 4, len(result))
	assert.Equal(t, signs[0], result[0])
	assert.Equal(t, signs[1], result[1])
	assert.Equal(t, signs[2], result[2])
	assert.Equal(t, signs[3], result[3])
	assert.Equal(t, signs[3], result[3])
	assert.Equal(t, signs[3], result[3])
	assert.Equal(t, signs[3], result[3])

	err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.NoError(t, err)
	result, err = cacheChain.GetConfirms(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 4, len(result))
	assert.Equal(t, signs[0], result[0])
	assert.Equal(t, signs[1], result[1])
	assert.Equal(t, signs[2], result[2])
	assert.Equal(t, signs[3], result[3])

	//cacheChain = NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	result, err = cacheChain.GetConfirms(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, 4, len(result))
	assert.Equal(t, signs[0], result[0])
	assert.Equal(t, signs[1], result[1])
	assert.Equal(t, signs[2], result[2])
	assert.Equal(t, signs[3], result[3])
}

func TestCacheChain_AppendConfirm(t *testing.T) {
	ClearData()

	cacheChain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)

	signs, err := CreateSign(16)
	assert.NoError(t, err)

	parentBlock := GetBlock0()
	err = cacheChain.SetBlock(parentBlock.Hash(), parentBlock)
	assert.NoError(t, err)

	err = cacheChain.SetConfirm(parentBlock.Hash(), signs[0])
	assert.NoError(t, err)

	err = cacheChain.SetStableBlock(parentBlock.Hash())
	assert.NoError(t, err)

	block, err := cacheChain.GetBlockByHash(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, len(block.Confirms), 1)
	assert.Equal(t, block.Confirms[0], signs[0])

	err = cacheChain.SetConfirm(parentBlock.Hash(), signs[1])
	assert.NoError(t, err)

	block, err = cacheChain.GetBlockByHash(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, len(block.Confirms), 2)
	assert.Equal(t, block.Confirms[0], signs[0])
	assert.Equal(t, block.Confirms[1], signs[1])

	err = cacheChain.SetConfirms(parentBlock.Hash(), signs)
	assert.NoError(t, err)

	block, err = cacheChain.GetBlockByHash(parentBlock.Hash())
	assert.NoError(t, err)
	assert.Equal(t, len(block.Confirms), 16)
	assert.Equal(t, block.Confirms[0], signs[0])
	assert.Equal(t, block.Confirms[1], signs[1])
	assert.Equal(t, block.Confirms[14], signs[14])
	assert.Equal(t, block.Confirms[15], signs[15])
}

func TestChainDatabase_Commit(t *testing.T) {
	ClearData()

	chain := NewChainDataBase(GetStorePath(), DRIVER_MYSQL, DNS_MYSQL)
	// rand.Seed(time.Now().Unix())
	blocks := NewBlockBatch(10)
	chain.SetBlock(blocks[0].Hash(), blocks[0])
	err := chain.SetStableBlock(blocks[0].Hash())
	assert.NoError(t, err)
	for index := 1; index < 10; index++ {
		chain.SetBlock(blocks[index].Hash(), blocks[index])
		err = chain.SetStableBlock(blocks[index].Hash())
		assert.NoError(t, err)
		if index > 10 {
			val, err := chain.GetBlockByHeight(uint32(index - 7))
			assert.NoError(t, err)
			if val == nil {
				panic("val :" + strconv.Itoa(index))
			}
		}
		log.Errorf("index:" + strconv.Itoa(index))
	}
}

func TestNewChainDataBase(t *testing.T) {
	for index := 0; index < 10; index++ {
		// offset := uint32(index) & 0xffffff00
		CURINDEX := uint32(index) & 0xff
		if CURINDEX == 129 {
			// panic("INDEX:129, INDEX:" + strconv.Itoa(int(index)))
			log.Errorf("INDEX:" + strconv.Itoa(index))
		}
	}
}
