package txpool

import (
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"sync"
)

var TransactionExpiration = 30 * 60

type TxPool struct {
	/* 还未被打包进块的交易 */
	PendingTxs *TxQueue

	/* 最近1个小时的所有交易 */
	RecentTxs *RecentTx

	/* 从当前高度向后的3600个块 */
	BlockCache *BlocksTrie

	RW sync.RWMutex
}

func NewTxPool() *TxPool {
	return &TxPool{
		PendingTxs: NewTxQueue(),
		RecentTxs:  NewTxRecently(),
		BlockCache: NewBlocksTrie(),
	}
}

/* 本节点出块时，从交易池中取出交易进行打包，但并不从交易池中删除 */
func (pool *TxPool) Get(time uint32, size int) []*types.Transaction {
	pool.RW.Lock()
	defer pool.RW.Unlock()
	return pool.PendingTxs.Pop(time, size)
}

/* 本节点出块时，执行交易后，发现错误的交易通过该接口进行删除 */
func (pool *TxPool) DelInvalidTxs(txs []*types.Transaction) {
	pool.RW.Lock()
	defer pool.RW.Unlock()

	if len(txs) <= 0 {
		return
	}

	hashes := make([]common.Hash, 0, len(txs))
	for _, tx := range txs {
		hashes = append(hashes, tx.Hash())
	}
	pool.PendingTxs.DelBatch(hashes)
}

func (pool *TxPool) isInBlocks(hashes HashSet, blocks []*TrieNode) bool {
	if len(hashes) <= 0 || len(blocks) <= 0 {
		return false
	}

	for _, v := range blocks {
		if !hashes.Has(v.Header.Hash()) {
			continue
		} else {
			return true
		}
	}

	return false
}

/* 新收一个块时，验证块中的交易是否被同一条分叉上的其他块打包了 */
func (pool *TxPool) VerifyTxInBlock(block *types.Block) bool {
	pool.RW.Lock()
	defer pool.RW.Unlock()

	if block == nil {
		return false
	}
	if len(block.Txs) <= 0 {
		return true
	}

	traceByHash := pool.RecentTxs.GetTrace(block.Txs)
	minHeight, maxHeight, blocks := pool.distance(traceByHash)
	startHash := block.ParentHash()
	startHeight := block.Height() - 1

	nodes := pool.BlockCache.Path(startHash, startHeight, uint32(minHeight), uint32(maxHeight))
	return !pool.isInBlocks(blocks, nodes)
}

func (pool *TxPool) distance(traceByHash map[common.Hash]TxTrace) (int64, int64, HashSet) {
	minHeight := int64(^uint64(0) >> 1)
	maxHeight := int64(-1)
	blockSet := make(HashSet)
	for _, trace := range traceByHash {
		if len(trace) <= 0 {
			continue
		} else {
			minHeightTmp, maxHeightTmp := trace.heightRange()
			if minHeight > minHeightTmp {
				minHeight = minHeightTmp
			}

			if maxHeight < maxHeightTmp {
				maxHeight = maxHeightTmp
			}

			for blockHash, _ := range trace {
				blockSet.Add(blockHash)
			}
		}
	}

	return minHeight, maxHeight, blockSet
}

/* 新收到一个通过验证的新块（包括本节点出的块），需要从交易池中删除该块中已打包的交易 */
func (pool *TxPool) RecvBlock(block *types.Block) {
	pool.RW.Lock()
	defer pool.RW.Unlock()

	if block == nil {
		return
	}

	if len(block.Txs) > 0 {
		hashes := make([]common.Hash, 0, len(block.Txs))
		for _, v := range block.Txs {
			hashes = append(hashes, v.Hash())
		}

		pool.PendingTxs.DelBatch(hashes)
		pool.RecentTxs.RecvBlock(block.Hash(), int64(block.Height()), block.Txs)
	}

	pool.BlockCache.PushBlock(block)
}

/* 收到一笔新的交易 */
func (pool *TxPool) RecvTx(tx *types.Transaction) {
	pool.RW.Lock()
	defer pool.RW.Unlock()

	if tx == nil {
		return
	}

	pool.RecentTxs.RecvTx(tx)
	pool.PendingTxs.Push(tx)

}

func (pool *TxPool) RecvTxs(txs []*types.Transaction) {
	if len(txs) <= 0 {
		return
	}

	for _, v := range txs {
		pool.RecvTx(v)
	}
}

/* 对链进行剪枝，剪下的块中的交易需要回归交易池 */
func (pool *TxPool) PruneBlock(block *types.Block) {
	pool.RW.Lock()
	defer pool.RW.Unlock()

	if block == nil {
		return
	}

	if len(block.Txs) > 0 {
		pool.PendingTxs.PushBatch(block.Txs)
		pool.RecentTxs.PruneBlock(block.Hash(), int64(block.Height()), block.Txs)
	}

	pool.BlockCache.DelBlock(block)
}
