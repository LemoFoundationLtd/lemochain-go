package network

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"sync"
)

// blockConfirms same block's confirm data set
type blockConfirms map[common.Hash][]*BlockConfirmData

// ConfirmCache record block confirm data which block doesn't exist in local
type ConfirmCache struct {
	cache map[uint32]blockConfirms

	lock sync.Mutex
}

func NewConfirmCache() *ConfirmCache {
	return &ConfirmCache{
		cache: make(map[uint32]blockConfirms),
	}
}

// Push push block confirm data to cache
func (c *ConfirmCache) Push(data *BlockConfirmData) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if _, ok := c.cache[data.Height]; !ok {
		c.cache[data.Height] = make(map[common.Hash][]*BlockConfirmData)
		c.cache[data.Height][data.Hash] = make([]*BlockConfirmData, 0, 2)
	}
	c.cache[data.Height][data.Hash] = append(c.cache[data.Height][data.Hash], data)
}

// Pop get special confirm data by height and hash and then delete from cache
func (c *ConfirmCache) Pop(height uint32, hash common.Hash) []*BlockConfirmData {
	c.lock.Lock()
	defer c.lock.Unlock()

	if _, ok := c.cache[height]; !ok {
		return nil
	}
	if _, ok := c.cache[height][hash]; !ok {
		return nil
	}
	res := make([]*BlockConfirmData, 0, len(c.cache[height][hash]))
	for _, v := range c.cache[height][hash] {
		res = append(res, v)
	}
	delete(c.cache[height], hash)
	// if len(c.cache[height]) == 0{
	// 	delete(c.cache, height)
	// }
	return res
}

// Clear clear dirty data form cache
func (c *ConfirmCache) Clear(height uint32) {
	c.lock.Lock()
	defer c.lock.Unlock()

	for h, _ := range c.cache {
		if h <= height {
			delete(c.cache, h)
		}
	}
}

type blocksSameHeight struct {
	Height uint32
	Blocks types.Blocks
}

type BlockCache struct {
	cache []*blocksSameHeight
	lock  sync.Mutex
}

func NewBlockCache() *BlockCache {
	return &BlockCache{
		cache: make([]*blocksSameHeight, 0, 100),
	}
}

func (c *BlockCache) Add(block *types.Block) {
	c.lock.Lock()
	defer c.lock.Unlock()

	height := block.Height()
	bsh := &blocksSameHeight{
		Height: block.Height(),
		Blocks: []*types.Block{block},
	}
	length := len(c.cache)
	if length == 0 || height < c.cache[0].Height { // not exist or at first position
		c.cache = append([]*blocksSameHeight{bsh}, c.cache...)
	} else if height > c.cache[length-1].Height { // at last position
		c.cache = append(c.cache, bsh)
	} else {
		for i := 0; i < len(c.cache)-1; i++ {
			if c.cache[i].Height == height || i == len(c.cache)-1 { // already exist
				c.cache[i].Blocks = append(c.cache[i].Blocks, block)
				break
			} else if c.cache[i].Height > height && height < c.cache[i+1].Height { // not exist
				tmp := append(c.cache[:i+1], bsh)
				c.cache = append(tmp, c.cache[i+1:]...)
				break
			}
		}
	}
}

func (c *BlockCache) Iterate(callback func(*types.Block) bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, blocks := range c.cache {
		for j := 0; j < len(blocks.Blocks); {
			if callback(blocks.Blocks[j]) {
				blocks.Blocks = append(blocks.Blocks[:j], blocks.Blocks[j+1:]...)
			} else {
				j++
			}
		}
	}
}

func (c *BlockCache) Clear(height uint32) {
	c.lock.Lock()
	defer c.lock.Unlock()

	index := -1
	for i, item := range c.cache {
		if item.Height <= height {
			index = i
		} else if item.Height > height {
			break
		}
	}
	c.cache = c.cache[index+1:]
}
