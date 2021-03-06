package network

import (
	"errors"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/params"
	"github.com/LemoFoundationLtd/lemochain-core/chain/txpool"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"github.com/LemoFoundationLtd/lemochain-core/common/subscribe"
	"github.com/LemoFoundationLtd/lemochain-core/metrics"
	"github.com/LemoFoundationLtd/lemochain-core/network/p2p"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrNodeInvalid        = errors.New("discover node is invalid")
	ErrRequestBlocks      = errors.New("invalid request blocks' param")
	ErrHandleLstStatusMsg = errors.New("stable height can't > current height")
	ErrHandleGetBlocksMsg = errors.New("invalid request blocks'param")
)

var (
	handleBlocksMsgMeter                 = metrics.NewMeter(metrics.HandleBlocksMsg_meterName)                 // 统计调用handleBlocksMsg的频率
	handleGetBlocksMsgMeter              = metrics.NewMeter(metrics.HandleGetBlocksMsg_meterName)              // 统计调用handleGetBlocksMsg的频率
	handleBlockHashMsgMeter              = metrics.NewMeter(metrics.HandleBlockHashMsg_meterName)              // 统计调用handleBlockHashMsg的频率
	handleGetConfirmsMsgMeter            = metrics.NewMeter(metrics.HandleGetConfirmsMsg_meterName)            // 统计调用handleGetConfirmsMsg的频率
	handleConfirmMsgMeter                = metrics.NewMeter(metrics.HandleConfirmMsg_meterName)                // 统计调用handleConfirmMsg的频率
	handleGetBlocksWithChangeLogMsgMeter = metrics.NewMeter(metrics.HandleGetBlocksWithChangeLogMsg_meterName) // 统计调用handleGetBlocksWithChangeLogMsg的频率
	handleDiscoverReqMsgMeter            = metrics.NewMeter(metrics.HandleDiscoverReqMsg_meterName)            // 统计调用handleDiscoverReqMsg的频率
	handleDiscoverResMsgMeter            = metrics.NewMeter(metrics.HandleDiscoverResMsg_meterName)            // 统计调用handleDiscoverResMsg的频率
)

// just for test
const (
	testBroadcastTxs int = 1 + iota
	testBroadcastBlock
	testBroadcastConfirm
	testRcvBlocks
	testQueueTimer
	testStableBlock
	testAddPeer
	testRemovePeer
	testForceSync
	testDiscover
)

// var testRcvFlag = false   // for test

type rcvBlockObj struct {
	p      *peer
	blocks types.Blocks
}

type ProtocolManager struct {
	chainID     uint16
	nodeID      p2p.NodeID
	nodeVersion uint32

	chain             BlockChain
	dm                *deputynode.Manager
	discover          *p2p.DiscoverManager
	txPool            TxPool
	txGuard           *txpool.TxGuard
	maxDelayNodeCount int           // delay node is what not deputy node. They only receive stable blocks
	peers             *peerSet      // connected peers
	confirmsCache     *ConfirmCache // received confirm info before block, cache them
	blockCache        *BlockCache
	dataDir           string
	oldStableBlock    atomic.Value

	addPeerCh    chan p2p.IPeer
	removePeerCh chan p2p.IPeer

	txCh             chan *types.Transaction
	newMinedBlockCh  chan *types.Block
	stableBlockCh    chan *types.Block
	rcvBlocksCh      chan *rcvBlockObj
	confirmCh        chan *BlockConfirmData
	fetchConfirms    chan []GetConfirmInfo
	lastSyncTime     int64
	lastSyncToHeight uint32

	wg     sync.WaitGroup
	quitCh chan struct{}

	test       bool
	testOutput chan int
}

func NewProtocolManager(chainID uint16, nodeID p2p.NodeID, chain BlockChain, dm *deputynode.Manager, txPool TxPool, txGuard *txpool.TxGuard, discover *p2p.DiscoverManager, delayNodeLimit int, nodeVersion uint32, dataDir string) *ProtocolManager {
	pm := &ProtocolManager{
		chainID:           chainID,
		nodeID:            nodeID,
		nodeVersion:       nodeVersion,
		chain:             chain,
		dm:                dm,
		txPool:            txPool,
		txGuard:           txGuard,
		discover:          discover,
		maxDelayNodeCount: delayNodeLimit,
		peers:             NewPeerSet(discover, dm),
		confirmsCache:     NewConfirmCache(),
		blockCache:        NewBlockCache(),
		dataDir:           dataDir,
		addPeerCh:         make(chan p2p.IPeer),
		removePeerCh:      make(chan p2p.IPeer),

		txCh:            make(chan *types.Transaction, 10),
		newMinedBlockCh: make(chan *types.Block),
		stableBlockCh:   make(chan *types.Block, 10),
		rcvBlocksCh:     make(chan *rcvBlockObj, 10),
		confirmCh:       make(chan *BlockConfirmData, 10),
		fetchConfirms:   make(chan []GetConfirmInfo),

		quitCh: make(chan struct{}),
	}
	pm.sub()
	return pm
}

func (pm *ProtocolManager) setTest() {
	pm.test = true
	pm.testOutput = make(chan int)
}

// sub subscribe channel
func (pm *ProtocolManager) sub() {
	subscribe.Sub(subscribe.AddNewPeer, pm.addPeerCh)
	subscribe.Sub(subscribe.DeletePeer, pm.removePeerCh)
	subscribe.Sub(subscribe.NewMinedBlock, pm.newMinedBlockCh)
	subscribe.Sub(subscribe.NewStableBlock, pm.stableBlockCh)
	subscribe.Sub(subscribe.NewTx, pm.txCh)
	subscribe.Sub(subscribe.NewConfirm, pm.confirmCh)
	subscribe.Sub(subscribe.FetchConfirms, pm.fetchConfirms)
}

// unSub unsubscribe channel
func (pm *ProtocolManager) unSub() {
	subscribe.UnSub(subscribe.AddNewPeer, pm.addPeerCh)
	subscribe.UnSub(subscribe.DeletePeer, pm.removePeerCh)
	subscribe.UnSub(subscribe.NewMinedBlock, pm.newMinedBlockCh)
	subscribe.UnSub(subscribe.NewStableBlock, pm.stableBlockCh)
	subscribe.UnSub(subscribe.NewTx, pm.txCh)
	subscribe.UnSub(subscribe.NewConfirm, pm.confirmCh)
	subscribe.UnSub(subscribe.FetchConfirms, pm.fetchConfirms)
}

// Start
func (pm *ProtocolManager) Start() {
	go pm.txConfirmLoop()
	go pm.rcvBlockLoop()
	go pm.stableBlockLoop()
	go pm.peerLoop()
}

// Stop
func (pm *ProtocolManager) Stop() {
	pm.unSub()
	close(pm.quitCh)
	pm.wg.Wait()
	log.Info("ProtocolManager has stopped")
}

// txConfirmLoop receive transactions and confirm and then broadcast them. request fetch block confirm and send msg
func (pm *ProtocolManager) txConfirmLoop() {
	pm.wg.Add(1)
	defer func() {
		pm.wg.Done()
		log.Debugf("txConfirmLoop finished")
	}()

	for {
		select {
		case <-pm.quitCh:
			log.Info("txConfirmLoop finished")
			return
		case tx := <-pm.txCh:
			currentBlock := pm.chain.CurrentBlock()
			peers := pm.peers.NeedBroadcastTxsNodes(currentBlock.Height(), currentBlock.MinerAddress())
			// for point log
			nodesID := make([]string, 0, len(peers))
			for _, p := range peers {
				nodesID = append(nodesID, common.ToHex(p.conn.RNodeID()[:]))
			}
			log.Debugf("tx broadcast peer nodeId: %v", nodesID)
			go pm.broadcastTxs(peers, types.Transactions{tx})
		case info := <-pm.confirmCh:
			if pm.peers.LatestStableHeight() > info.Height {
				continue
			}
			curHeight := pm.chain.CurrentBlock().Height()
			peers := pm.peers.DeputyNodes(curHeight)
			go pm.broadcastConfirm(peers, info)
			log.Debugf("broadcast confirm, len(peers)=%d, height: %d", len(peers), info.Height)
		case infoList := <-pm.fetchConfirms:
			go pm.fetchConfirmFromRemote(infoList)
		}
	}
}

// blockLoop receive special type block event
func (pm *ProtocolManager) rcvBlockLoop() {
	pm.wg.Add(1)
	defer func() {
		pm.wg.Done()
		log.Debugf("RcvBlockLoop finished")
	}()

	proInterval := 500 * time.Millisecond
	queueTimer := time.NewTimer(proInterval)

	// 初始化区块黑名单
	bbc := InitBlockBlackCache(pm.dataDir)

	// just for test
	// testRcvTimer := time.NewTimer(8 * time.Second)

	for {
		select {
		case <-pm.quitCh:
			log.Info("BlockLoop finished")
			return
		case block := <-pm.newMinedBlockCh:
			log.Debugf("Current peers count: %d", len(pm.peers.peers))
			peers := pm.peers.DeputyNodes(block.Height())
			go pm.broadcastBlock(peers, block, true)
		case rcvMsg := <-pm.rcvBlocksCh:
			// for test
			// testRcvFlag = false
			// testRcvTimer.Reset(8 * time.Second)

			// peer's latest height
			pLstHeight := rcvMsg.p.LatestStatus().CurHeight
			log.Infof("Got %d blocks from peer: %#x", len(rcvMsg.blocks), rcvMsg.p.NodeID()[:4])

			for i, b := range rcvMsg.blocks {
				// 判断收到的区块是否为黑名单区块
				if bbc.IsBlackBlock(b.Hash(), b.ParentHash()) {
					log.Warnf("This block is black block. block: %s", b.String())
					continue
				}
				// update latest status
				if b.Height() > pLstHeight && rcvMsg.p != nil {
					rcvMsg.p.UpdateStatus(b.Height(), b.Hash())
				}
				// block is stale
				if b.Height() <= pm.chain.StableBlock().Height() || pm.chain.HasBlock(b.Hash()) {
					log.Debugf("This block has exist. blockHeight: %d", b.Height())
					continue
				}
				// The block mined by minerAddress in blackList
				if pm.chain.IsInBlackList(b) {
					log.Debug("This block minerAddress is in BlackList, black miner: %s, block height: %d, block hash: %s.", b.MinerAddress().String(), b.Height(), b.Hash().String())
					pm.blockCache.Remove(b)
					continue
				}
				// local chain has parent block or parent block will insert chain
				if pm.chain.HasBlock(b.ParentHash()) {
					log.Infof("Got a block %s from peer: %#x", b.ShortString(), rcvMsg.p.NodeID()[:4])
					if err := pm.insertBlock(b); err != nil {
						log.Warnf("block verify failed. ignore the rest %d blocks", len(rcvMsg.blocks)-1-i)
						break
					}
				} else if b.Height() <= 1 && pm.chain.Genesis() != nil {
					parentHash := b.ParentHash()
					log.Warnf("Different genesis! Got a block %x(parent:%s) from peer: %#x", parentHash[:8], b.ShortString(), rcvMsg.p.NodeID()[:4])
					rcvMsg.p.DifferentGenesisClose()
					pm.peers.UnRegister(rcvMsg.p)
					break
				} else {
					log.Infof("Got a block %s from peer: %#x, cache it", b.ShortString(), rcvMsg.p.NodeID()[:4])
					pm.blockCache.Add(b)
					if rcvMsg.p != nil {
						// request parent block
						go rcvMsg.p.RequestBlocks(b.Height()-1, b.Height()-1)
					}
				}
			}
			// for test
			if pm.test {
				pm.testOutput <- testRcvBlocks
			}
			// time to check cache. request parent block of the first block in cache
		case <-queueTimer.C:
			processBlock := func(block *types.Block) bool {
				// 检查缓存中的黑名单区块
				if bbc.IsBlackBlock(block.Hash(), block.ParentHash()) {
					return true
				}
				if pm.chain.HasBlock(block.ParentHash()) {
					go pm.insertBlock(block)
					return true
				}
				return false
			}
			pm.blockCache.Iterate(processBlock)
			queueTimer.Reset(proInterval)
			// output cache size
			cacheSize := pm.blockCache.Size()
			if cacheSize > 0 {
				p := pm.peers.BestToSync(pm.blockCache.FirstHeight())
				if p != nil {
					log.Debugf("BlockCache's size: %d. request the parent block of %d", cacheSize, pm.blockCache.FirstHeight())
					go p.RequestBlocks(pm.blockCache.FirstHeight()-1, pm.blockCache.FirstHeight()-1)
				}
			}
			// for test
			if pm.test {
				pm.testOutput <- testQueueTimer
			}
			// case <-testRcvTimer.C: // just for test
			// 	testRcvFlag = true
		}
	}
}

// insertBlock insert block
func (pm *ProtocolManager) insertBlock(b *types.Block) error {
	// pop the confirms which arrived before block
	pm.mergeConfirmsFromCache(b)
	return pm.chain.InsertBlock(b)
}

// stableBlockLoop block has been stable
func (pm *ProtocolManager) stableBlockLoop() {
	pm.wg.Add(1)
	defer func() {
		pm.wg.Done()
		log.Debugf("StableBlockLoop finished")
	}()

	for {
		select {
		case <-pm.quitCh:
			return
		case block := <-pm.stableBlockCh:
			pm.oldStableBlock.Store(block)
			peers := pm.peers.DelayNodes(block.Height())
			if len(peers) > 0 {
				// for debug
				log.Debug("Broadcast stable block to delay node")
				go pm.broadcastBlock(peers, block, false)
			}
			go func() {
				pm.confirmsCache.Clear(block.Height())
				pm.blockCache.Clear(block.Height())
			}()
			// for test
			if pm.test {
				pm.testOutput <- testStableBlock
			}
		}
	}
}

// fetchConfirmFromRemote
func (pm *ProtocolManager) fetchConfirmFromRemote(infoList []GetConfirmInfo) {
	length := len(infoList)
	if length == 0 {
		return
	}

	// get maximal height for find the best peer
	var maxHeight uint32 = 0
	for i := 0; i < length; i++ {
		if infoList[i].Height > maxHeight {
			maxHeight = infoList[i].Height
		}
	}
	// find the best peer to fetch block confirms
	p := pm.peers.BestToFetchConfirms(maxHeight)
	if p == nil {
		return
	}

	for i := 0; i < length; i++ {
		err := p.SendGetConfirms(infoList[i].Height, infoList[i].Hash)
		if err != nil {
			log.Errorf("Send get confirms message error,errorCode = %d", err)
			return
		}
	}
}

// mergeConfirmsFromCache merge confirms into block from cache
func (pm *ProtocolManager) mergeConfirmsFromCache(block *types.Block) {
	confirms := pm.confirmsCache.Pop(block.Height(), block.Hash())
	for _, confirm := range confirms {
		// The duplicates will be remove by consensus validator
		block.Confirms = append(block.Confirms, confirm.SignInfo)
	}
	log.Debugf("Pop %d confirms from cache, now we have %d confirms", len(confirms), len(block.Confirms))
}

// peerLoop something about peer event
func (pm *ProtocolManager) peerLoop() {
	pm.wg.Add(1)
	defer func() {
		pm.wg.Done()
		log.Debugf("PeerLoop finished")
	}()

	forceSyncTimer := time.NewTimer(params.ForceSyncInternal)
	discoverTimer := time.NewTimer(params.DiscoverInternal)
	for {
		select {
		case <-pm.quitCh:
			log.Info("PeerLoop finished")
			return
		case rPeer := <-pm.addPeerCh: // new peer added
			if pm.checkConnectionLimit(rPeer) {
				p := newPeer(rPeer)
				go pm.handlePeer(p)
				// for test
				if pm.test {
					pm.testOutput <- testAddPeer
				}
			}
		case rPeer := <-pm.removePeerCh:
			p := newPeer(rPeer)
			pm.peers.UnRegister(p)
			log.Infof("Connection has dropped, nodeID: %s", p.NodeID().String()[:16])
			// for test
			if pm.test {
				pm.testOutput <- testRemovePeer
			}
		case <-forceSyncTimer.C: // time to synchronise block
			p := pm.peers.BestToSync(pm.chain.CurrentBlock().Height())
			if p != nil {
				go p.SendReqLatestStatus()
			}
			forceSyncTimer.Reset(params.ForceSyncInternal)
			// for test
			if pm.test {
				pm.testOutput <- testForceSync
			}
		case <-discoverTimer.C: // time to discover
			if len(pm.peers.peers) < params.LeastPeersToDiscover {
				p := pm.peers.BestToDiscover()
				if p != nil {
					go p.SendDiscover()
				}
			}
			discoverTimer.Reset(params.DiscoverInternal)
			// for test
			if pm.test {
				pm.testOutput <- testDiscover
			}
		}
	}
}

func (pm *ProtocolManager) checkConnectionLimit(p p2p.IPeer) bool {
	height := pm.chain.CurrentBlock().Height() + 1
	rNodeID := p.RNodeID()
	// deputy node
	if n := pm.dm.GetDeputyByNodeID(height, rNodeID[:]); n != nil {
		return true
	}
	// if node in white list
	if pm.discover.InWhiteList(rNodeID) {
		return true
	}
	// limit
	connected := pm.peers.DelayNodes(height)
	if len(connected) <= pm.maxDelayNodeCount {
		return true
	}
	return false
}

// broadcastTxs broadcast transaction
func (pm *ProtocolManager) broadcastTxs(peers []*peer, txs types.Transactions) {
	for _, p := range peers {
		p.SendTxs(txs)
	}
	if pm.test {
		pm.testOutput <- testBroadcastTxs
	}
}

// broadcastConfirm broadcast confirm info to deputy nodes
func (pm *ProtocolManager) broadcastConfirm(peers []*peer, confirmInfo *BlockConfirmData) {
	for _, p := range peers {
		p.SendConfirmInfo(confirmInfo)
	}
	if pm.test {
		pm.testOutput <- testBroadcastConfirm
	}
}

// broadcastBlock broadcast block
func (pm *ProtocolManager) broadcastBlock(peers []*peer, block *types.Block, withBody bool) {
	for _, p := range peers {
		if withBody {
			p.SendBlocks([]*types.Block{block})
		} else {
			p.SendBlockHash(block.Height(), block.Hash())
		}
	}
	if pm.test {
		pm.testOutput <- testBroadcastBlock
	}
}

// handlePeer handle about peer
func (pm *ProtocolManager) handlePeer(p *peer) {
	// handshake
	rStatus, err := pm.handshake(p)
	if err != nil {
		log.Warnf("protocol handshake failed: %v", err)
		if err = pm.discover.SetConnectResult(p.NodeID(), false); err != nil {
			log.Debugf("handlePeer: %v", err)
		}
		p.FailedHandshakeClose()
		return
	}
	// register peer to set
	pm.peers.Register(p)
	// synchronise block
	if pm.chain.CurrentBlock().Height() < rStatus.LatestStatus.CurHeight {
		from, err := pm.findSyncFrom(&rStatus.LatestStatus)
		if err != nil {
			log.Warnf("Find sync from error: %v", err)
			if err = pm.discover.SetConnectResult(p.NodeID(), false); err != nil {
				log.Debugf("HandlePeer: %v", err)
			}
			p.HardForkClose()
			return
		}
		log.Infof("sync status from the new peer %s", p.NodeID().String()[:16])
		pm.syncBlocks(p, from, rStatus.LatestStatus.CurHeight)
	}
	// set connect result
	if err = pm.discover.SetConnectResult(p.NodeID(), true); err != nil {
		log.Debugf("HandlePeer set connect result: %v", err)
	}

	for {
		// handle peer net message
		if err := pm.handleMsg(p); err != nil {
			log.Debugf("Handle message failed: %v", err)
			if err != io.EOF {
				p.conn.Close()
			}
			return
		}
	}
}

// handshake protocol handshake
func (pm *ProtocolManager) handshake(p *peer) (*ProtocolHandshake, error) {
	phs := &ProtocolHandshake{
		ChainID:     pm.chainID,
		GenesisHash: pm.chain.Genesis().Hash(),
		NodeVersion: pm.nodeVersion,
		LatestStatus: LatestStatus{
			CurHash:   pm.chain.CurrentBlock().Hash(),
			CurHeight: pm.chain.CurrentBlock().Height(),
			StaHash:   pm.chain.StableBlock().Hash(),
			StaHeight: pm.chain.StableBlock().Height(),
		},
	}
	content := phs.Bytes()
	if content == nil {
		return nil, errors.New("rlp encode error")
	}
	remoteStatus, err := p.Handshake(content)
	if err != nil {
		return nil, err
	}
	return remoteStatus, nil
}

// forceSyncBlock force to sync block
func (pm *ProtocolManager) forceSyncBlock(status *LatestStatus, p *peer) {
	if status.CurHeight <= pm.chain.CurrentBlock().Height() {
		return
	}

	from, err := pm.findSyncFrom(status)
	if err != nil {
		log.Warnf("forceSyncBlock. Find sync from error: %v", err)
		p.HardForkClose()
		pm.peers.UnRegister(p)
		return
	}
	log.Infof("sync status from the best peer %s", p.NodeID().String()[:16])
	pm.syncBlocks(p, from, status.CurHeight)
}

// findSyncFrom find height of which sync from
func (pm *ProtocolManager) findSyncFrom(rStatus *LatestStatus) (uint32, error) {
	var from uint32
	staBlock := pm.chain.StableBlock()
	// 1. 本地稳定块高度大于等待远程稳定块高度的情况。本地稳定块高度大于远程稳定块默认为本地的网络环境好于远程节点，从而本地的区块多余远程节点，所以不去同步区块。
	if staBlock.Height() > rStatus.StaHeight {
		// 判断如果远程稳定块不在本地的当前链上，则表示链硬分叉了
		if !pm.chain.HasBlock(rStatus.StaHash) {
			return 0, errors.New("error: CHAIN FORK")
		}
		return rStatus.CurHeight + 1, nil
	}
	// 2. 本地稳定块高度小于等于远程稳定块高度的情况
	if staBlock.Height() <= rStatus.StaHeight {
		// 2.1 首先判断远程current是否已经存在本地链,存在则没必要去同步了
		if pm.chain.HasBlock(rStatus.CurHash) {
			return rStatus.CurHeight + 1, nil
		}
		// 2.2 判断远程稳定区块是否存在本地链上，如果存在则从远程稳定块高度开始同步，如果不存在则从本地稳定块高度开始同步
		if pm.chain.HasBlock(rStatus.StaHash) {
			// 从远程稳定块高度开始同步
			from = rStatus.StaHeight + 1
		} else {
			// 从本地区块高度开始同步
			from = staBlock.Height() + 1
		}
	}
	return from, nil
}

// syncBlocks sync blocks with throttle algorithm
func (pm *ProtocolManager) syncBlocks(p *peer, from, to uint32) bool {
	// to avoid sync blocks from different peer at same time
	var timeDelay int64
	if from != to {
		// multi blocks sync on be happened once a minute
		timeDelay = 60
	} else if to == pm.lastSyncToHeight {
		// synchronised this block before
		timeDelay = 10
	}

	now := time.Now().Unix()
	if now-pm.lastSyncTime > timeDelay {
		p.RequestBlocks(from, to)
		pm.lastSyncTime = now
		pm.lastSyncToHeight = to
		return true
	} else {
		log.Debug("stop sync blocks cause we just sync a few seconds ago")
		return false
	}
}

// work return handle msg error
func (pm *ProtocolManager) work(msg *p2p.Msg, p *peer) error {
	// log.Debug("Process received message", "Code", msg.Code, "NodeID", common.ToHex(p.NodeID()[:4]))
	switch msg.Code {
	case p2p.LstStatusMsg:
		return pm.handleLstStatusMsg(msg, p)
	case p2p.GetLstStatusMsg:
		return pm.handleGetLstStatusMsg(msg, p)
	case p2p.BlockHashMsg:
		return pm.handleBlockHashMsg(msg, p)
	case p2p.TxsMsg:
		return pm.handleTxsMsg(msg)
	case p2p.BlocksMsg:
		return pm.handleBlocksMsg(msg, p)
	case p2p.GetBlocksMsg:
		return pm.handleGetBlocksMsg(msg, p)
	case p2p.GetConfirmsMsg:
		return pm.handleGetConfirmsMsg(msg, p)
	case p2p.ConfirmsMsg:
		return pm.handleConfirmsMsg(msg)
	case p2p.ConfirmMsg:
		return pm.handleConfirmMsg(msg)
	case p2p.DiscoverReqMsg:
		return pm.handleDiscoverReqMsg(msg, p)
	case p2p.DiscoverResMsg:
		return pm.handleDiscoverResMsg(msg)
	case p2p.GetBlocksWithChangeLogMsg:
		return pm.handleGetBlocksWithChangeLogMsg(msg, p)
	default:
		log.Debugf("invalid code: %s, from: %s", msg.Code, common.ToHex(p.NodeID()[:4]))
		return ErrInvalidCode
	}
}

// handleMsg handle net received message
func (pm *ProtocolManager) handleMsg(p *peer) error {
	msgCache := NewMsgCache()
	errCh := make(chan error, 1)   // 返回error的管道
	closeCh := make(chan struct{}) // 监听退出ReadMsg协程的管道
	// read msg
	go func() {
		for {
			select {
			case <-closeCh:
				return
			default:
			}

			msg, err := p.ReadMsg()
			if err != nil {
				errCh <- err
				return
			}
			// cache msg
			msgCache.Push(msg)
		}
	}()

	for {
		// listen ReadMsg error
		if len(errCh) != 0 {
			err := <-errCh
			return err
		}

		msg := msgCache.Pop()
		err := pm.work(msg, p)
		if err != nil {
			close(closeCh)
			return err
		}
	}

}

// handleLstStatusMsg handle latest remote status message
func (pm *ProtocolManager) handleLstStatusMsg(msg *p2p.Msg, p *peer) error {
	var status LatestStatus
	if err := msg.Decode(&status); err != nil {
		return fmt.Errorf("handleLstStatusMsg error: %v", err)
	}

	if status.StaHeight > status.CurHeight {
		return ErrHandleLstStatusMsg
	}

	go pm.forceSyncBlock(&status, p)
	return nil
}

// handleGetLstStatusMsg handle request of latest status
func (pm *ProtocolManager) handleGetLstStatusMsg(msg *p2p.Msg, p *peer) error {
	var req GetLatestStatus
	if err := msg.Decode(&req); err != nil {
		return fmt.Errorf("handleGetLstStatusMsg error: %v", err)
	}
	status := &LatestStatus{
		CurHeight: pm.chain.CurrentBlock().Height(),
		CurHash:   pm.chain.CurrentBlock().Hash(),
		StaHeight: pm.chain.StableBlock().Height(),
		StaHash:   pm.chain.StableBlock().Hash(),
	}
	go p.SendLstStatus(status)
	return nil
}

// handleBlockHashMsg handle receiving block's hash message
func (pm *ProtocolManager) handleBlockHashMsg(msg *p2p.Msg, p *peer) error {
	defer handleBlockHashMsgMeter.Mark(1)
	var hashMsg BlockHashData
	if err := msg.Decode(&hashMsg); err != nil {
		return fmt.Errorf("handleBlockHashMsg error: %v", err)
	}

	if pm.chain.HasBlock(hashMsg.Hash) {
		return nil
	}

	if pm.chain.StableBlock().Height() >= hashMsg.Height {
		return nil
	}

	// update status
	p.UpdateStatus(hashMsg.Height, hashMsg.Hash)
	go pm.syncBlocks(p, hashMsg.Height, hashMsg.Height)
	return nil
}

// handleTxsMsg handle transactions message
func (pm *ProtocolManager) handleTxsMsg(msg *p2p.Msg) error {
	var txs types.Transactions
	if err := msg.Decode(&txs); err != nil {
		return fmt.Errorf("handleTxsMsg error: %v", err)
	}
	// verify tx expiration time
	nowTime := uint64(time.Now().Unix())
	for _, tx := range txs {
		if err := tx.VerifyTxBody(pm.chainID, nowTime, false); err != nil {
			continue
		}

		go func() {
			// 判断接收到的交易是否在本分支已经存在
			currentBlock := pm.chain.CurrentBlock()
			isExist := pm.txGuard.ExistTx(currentBlock.Hash(), tx)
			if !isExist {
				if err := pm.txPool.AddTx(tx); err == nil { // 加入交易池
					// 广播交易
					subscribe.Send(subscribe.NewTx, tx)
				}
			}
		}()
	}
	return nil
}

// handleBlocksMsg handle receiving blocks message
func (pm *ProtocolManager) handleBlocksMsg(msg *p2p.Msg, p *peer) error {
	defer handleBlocksMsgMeter.Mark(1)
	var blocks types.Blocks
	if err := msg.Decode(&blocks); err != nil {
		return fmt.Errorf("handleBlocksMsg error: %v", err)
	}
	rcvMsg := &rcvBlockObj{
		p:      p,
		blocks: blocks,
	}
	pm.rcvBlocksCh <- rcvMsg
	return nil
}

// handleGetBlocksMsg handle get blocks message
func (pm *ProtocolManager) handleGetBlocksMsg(msg *p2p.Msg, p *peer) error {
	defer handleGetBlocksMsgMeter.Mark(1)
	var query GetBlocksData
	if err := msg.Decode(&query); err != nil {
		return fmt.Errorf("handleGetBlocksMsg error: %v", err)
	}
	if query.From > query.To {
		return ErrHandleGetBlocksMsg
	}
	if query.From > pm.chain.CurrentBlock().Height() {
		return nil
	}
	go pm.respBlocks(query.From, query.To, p, false)
	return nil
}

// respBlocks response blocks to remote peer
func (pm *ProtocolManager) respBlocks(from, to uint32, p *peer, hasChangeLog bool) {
	log.Info("response blocks", "peer", p.NodeID().String()[:16], "fromHeight", from, "toHeight", to)
	if from == to {
		b := pm.chain.GetBlockByHeight(from)
		if b == nil {
			log.Warnf("Can't get a block of height %d", from)
			return
		}
		if !hasChangeLog {
			b = b.ShallowCopy()
		}
		if b != nil && p != nil {
			p.SendBlocks([]*types.Block{b})
		}
		return
	}

	const eachSize = 10
	total := to - from + 1
	var count uint32
	if total%eachSize == 0 {
		count = total / eachSize
	} else {
		count = total/eachSize + 1
	}
	height := from
	for i := uint32(0); i < count; i++ {
		blocks := make(types.Blocks, 0, eachSize)
		for j := 0; j < eachSize; j++ {
			b := pm.chain.GetBlockByHeight(height)
			if b == nil {
				log.Warnf("Can't get a block of height %d", height)
				break
			}
			if !hasChangeLog {
				b = b.ShallowCopy()
			}
			blocks = append(blocks, b)
			height++
			if height > to {
				break
			}
		}
		if p != nil && len(blocks) != 0 {
			p.SendBlocks(blocks)
		}
	}
}

// handleConfirmsMsg handle received block's confirm package message
func (pm *ProtocolManager) handleConfirmsMsg(msg *p2p.Msg) error {
	var info BlockConfirms
	if err := msg.Decode(&info); err != nil {
		return fmt.Errorf("handleConfirmsMsg error: %v", err)
	}
	go pm.chain.InsertConfirms(info.Height, info.Hash, info.Pack)
	return nil
}

// handleGetConfirmsMsg handle remote request of block's confirm package message
func (pm *ProtocolManager) handleGetConfirmsMsg(msg *p2p.Msg, p *peer) error {
	defer handleGetConfirmsMsgMeter.Mark(1)
	var condition GetConfirmInfo
	if err := msg.Decode(&condition); err != nil {
		return fmt.Errorf("handleGetConfirmsMsg error: %v", err)
	}
	var block *types.Block
	if condition.Hash != (common.Hash{}) {
		block = pm.chain.GetBlockByHash(condition.Hash)
	} else {
		block = pm.chain.GetBlockByHeight(condition.Height)
	}
	resMsg := &BlockConfirms{
		Height: condition.Height,
		Hash:   condition.Hash,
	}
	if block != nil {
		resMsg.Pack = block.Confirms
	}
	go p.SendConfirms(resMsg)
	return nil
}

// handleConfirmMsg handle confirm broadcast info
func (pm *ProtocolManager) handleConfirmMsg(msg *p2p.Msg) error {
	defer handleConfirmMsgMeter.Mark(1)
	confirm := new(BlockConfirmData)
	if err := msg.Decode(confirm); err != nil {
		return fmt.Errorf("handleConfirmMsg error: %v", err)
	}
	if pm.chain.HasBlock(confirm.Hash) {
		go pm.chain.InsertConfirms(confirm.Height, confirm.Hash, []types.SignData{confirm.SignInfo})
	} else {
		pm.confirmsCache.Push(confirm)
		if pm.confirmsCache.Size() > 100 {
			log.Debugf("confirmsCache's size: %d", pm.confirmsCache.Size())
		}
	}
	return nil
}

// handleDiscoverReqMsg handle discover nodes request
func (pm *ProtocolManager) handleDiscoverReqMsg(msg *p2p.Msg, p *peer) error {
	defer handleDiscoverReqMsgMeter.Mark(1)
	var condition DiscoverReqData
	if err := msg.Decode(&condition); err != nil {
		return fmt.Errorf("handleDiscoverReqMsg error: %v", err)
	}
	res := new(DiscoverResData)
	res.Sequence = condition.Sequence
	res.Nodes = pm.discover.GetNodesForDiscover(res.Sequence, p.NodeID().String())
	go p.SendDiscoverResp(res)
	return nil
}

// handleDiscoverResMsg handle discover nodes response
func (pm *ProtocolManager) handleDiscoverResMsg(msg *p2p.Msg) error {
	defer handleDiscoverResMsgMeter.Mark(1)
	var disRes DiscoverResData
	if err := msg.Decode(&disRes); err != nil {
		return fmt.Errorf("handleDiscoverResMsg error: %v", err)
	}
	// verify nodes
	for _, node := range disRes.Nodes {
		if err := VerifyNode(node); err != nil {
			log.Errorf("HandleDiscoverResMsg exists invalid node. error node: %s", node)
			return ErrNodeInvalid
		}
	}
	pm.discover.AddNewList(disRes.Nodes)
	return nil
}

// handleGetBlocksWithChangeLogMsg for
func (pm *ProtocolManager) handleGetBlocksWithChangeLogMsg(msg *p2p.Msg, p *peer) error {
	defer handleGetBlocksWithChangeLogMsgMeter.Mark(1)
	var query GetBlocksData
	if err := msg.Decode(&query); err != nil {
		return fmt.Errorf("handleGetBlocksMsg error: %v", err)
	}
	if query.From > query.To {
		return ErrRequestBlocks
	}
	if query.From > pm.chain.CurrentBlock().Height() {
		return nil
	}
	go pm.respBlocks(query.From, query.To, p, true)
	return nil
}
