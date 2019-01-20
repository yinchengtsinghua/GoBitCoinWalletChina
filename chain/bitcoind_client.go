
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package chain

import (
	"container/list"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/wtxmgr"
)

var (
//ErrBitcoindClientsHuttingDown是我们尝试时返回的错误
//接收特定项目和比特币客户的通知
//正在关机。
	ErrBitcoindClientShuttingDown = errors.New("client is shutting down")
)

//比特币客户机表示与比特币服务器的持久客户端连接
//有关当前最佳区块链的信息。
type BitcoindClient struct {
started int32 //原子化使用。
stopped int32 //原子化使用。

//生日是我们最早开始扫描
//链。
	birthday time.Time

//chainParams是此客户端当前链的参数
//主动下。
	chainParams *chaincfg.Params

//ID是由支持比特币分配的此客户端的唯一ID
//连接。
	id uint64

//chainconn是我们的rescan客户机的备份客户机，其中包含
//rpc和zmq连接到比特节点。
	chainConn *BitcoindConn

//BestBlock跟踪当前最佳链的尖端。
	bestBlockMtx sync.RWMutex
	bestBlock    waddrmgr.BlockStamp

//notifyBlocks表示客户端是否正在发送块
//通知呼叫者。
	notifyBlocks uint32

//rescanupdate是一个频道将发送我们应该匹配的项目
//处理链时针对的事务重新扫描以确定
//它们与客户有关。
	rescanUpdate chan interface{}

//watchedAddresses、watchedOutpoints和watchedTxs是
//在处理一个链时，我们应该与事务匹配的项
//重新扫描以确定它们是否与客户端相关。
	watchMtx         sync.RWMutex
	watchedAddresses map[string]struct{}
	watchedOutPoints map[wire.OutPoint]struct{}
	watchedTxs       map[chainhash.Hash]struct{}

//mempool跟踪所有尚未
//证实。这用于快捷方式的筛选过程
//当新的已确认事务通知为
//收到。
//
//注意：这需要保持watchmtx。
	mempool map[chainhash.Hash]struct{}

//ExpiredEmpool跟踪一组已确认的交易
//它们被包含在一个块中的高度。这些
//在一段时间后，事务将从mempool中删除。
//288个街区。这样做是为了确保交易在
//在链中重新组合。
//
//注意：这需要保持watchmtx。
	expiredMempool map[int32]map[chainhash.Hash]struct{}

//notificationQueue是处理
//正在向此客户端的订阅服务器发送通知。
//
//TODO:而不是将此作为所有类型的
//通知，尝试删除那些稍后进入队列的通知
//可以完全使等待处理的无效。例如，
//块高度较大的块连接通知可以删除
//需要处理仍在等待处理的早期通知。
	notificationQueue *ConcurrentQueue

//zmqtxntfns是zmq事务事件将通过的通道
//从支持比特币连接中检索。
	zmqTxNtfns chan *wire.MsgTx

//zmqblockntfns是zmq块事件将通过的通道
//从支持比特币连接中检索。
	zmqBlockNtfns chan *wire.MsgBlock

	quit chan struct{}
	wg   sync.WaitGroup
}

//编译时检查以确保比特币客户满足
//链。接口接口。
var _ Interface = (*BitcoindClient)(nil)

//后端返回驱动程序的名称。
func (c *BitcoindClient) BackEnd() string {
	return "bitcoind"
}

//GetBestBlock返回比特币已知的最高块。
func (c *BitcoindClient) GetBestBlock() (*chainhash.Hash, int32, error) {
	bcinfo, err := c.chainConn.client.GetBlockChainInfo()
	if err != nil {
		return nil, 0, err
	}

	hash, err := chainhash.NewHashFromStr(bcinfo.BestBlockHash)
	if err != nil {
		return nil, 0, err
	}

	return hash, bcinfo.Blocks, nil
}

//GetBlockHeight返回哈希的高度（如果已知），或者返回
//错误。
func (c *BitcoindClient) GetBlockHeight(hash *chainhash.Hash) (int32, error) {
	header, err := c.chainConn.client.GetBlockHeaderVerbose(hash)
	if err != nil {
		return 0, err
	}

	return header.Height, nil
}

//GetBlock从哈希返回一个块。
func (c *BitcoindClient) GetBlock(hash *chainhash.Hash) (*wire.MsgBlock, error) {
	return c.chainConn.client.GetBlock(hash)
}

//GetBlockVerbose从哈希返回详细块。
func (c *BitcoindClient) GetBlockVerbose(
	hash *chainhash.Hash) (*btcjson.GetBlockVerboseResult, error) {

	return c.chainConn.client.GetBlockVerbose(hash)
}

//GetBlockHash从高度返回块哈希。
func (c *BitcoindClient) GetBlockHash(height int64) (*chainhash.Hash, error) {
	return c.chainConn.client.GetBlockHash(height)
}

//GetBlockHeader从哈希返回块头。
func (c *BitcoindClient) GetBlockHeader(
	hash *chainhash.Hash) (*wire.BlockHeader, error) {

	return c.chainConn.client.GetBlockHeader(hash)
}

//GetBlockHeaderbose从哈希返回块头。
func (c *BitcoindClient) GetBlockHeaderVerbose(
	hash *chainhash.Hash) (*btcjson.GetBlockHeaderVerboseResult, error) {

	return c.chainConn.client.GetBlockHeaderVerbose(hash)
}

//GetRawTransactionVerbose从Tx哈希返回事务。
func (c *BitcoindClient) GetRawTransactionVerbose(
	hash *chainhash.Hash) (*btcjson.TxRawResult, error) {

	return c.chainConn.client.GetRawTransactionVerbose(hash)
}

//gettxout从提供的输出点信息返回txout。
func (c *BitcoindClient) GetTxOut(txHash *chainhash.Hash, index uint32,
	mempool bool) (*btcjson.GetTxOutResult, error) {

	return c.chainConn.client.GetTxOut(txHash, index, mempool)
}

//sendrawtransaction通过比特币发送原始交易。
func (c *BitcoindClient) SendRawTransaction(tx *wire.MsgTx,
	allowHighFees bool) (*chainhash.Hash, error) {

	return c.chainConn.client.SendRawTransaction(tx, allowHighFees)
}

//通知返回从中检索通知的通道。
//
//注意：这是chain.interface接口的一部分。
func (c *BitcoindClient) Notifications() <-chan interface{} {
	return c.notificationQueue.ChanOut()
}

//notifyReceived允许链后端在
//交易支付到任何给定的地址。
//
//注意：这是chain.interface接口的一部分。
func (c *BitcoindClient) NotifyReceived(addrs []btcutil.Address) error {
	c.NotifyBlocks()

	select {
	case c.rescanUpdate <- addrs:
	case <-c.quit:
		return ErrBitcoindClientShuttingDown
	}

	return nil
}

//notifyspeed允许链后端在
//事务花费任何给定的输出点。
func (c *BitcoindClient) NotifySpent(outPoints []*wire.OutPoint) error {
	c.NotifyBlocks()

	select {
	case c.rescanUpdate <- outPoints:
	case <-c.quit:
		return ErrBitcoindClientShuttingDown
	}

	return nil
}

//notifytx允许链后端通知调用方
//给定的交易在链内确认。
func (c *BitcoindClient) NotifyTx(txids []chainhash.Hash) error {
	c.NotifyBlocks()

	select {
	case c.rescanUpdate <- txids:
	case <-c.quit:
		return ErrBitcoindClientShuttingDown
	}

	return nil
}

//notifyBlocks允许链后端在每次块
//连接或断开。
//
//注意：这是chain.interface接口的一部分。
func (c *BitcoindClient) NotifyBlocks() error {
	atomic.StoreUint32(&c.notifyBlocks, 1)
	return nil
}

//shouldnotifyBlocks确定客户端是否应发送块
//通知呼叫者。
func (c *BitcoindClient) shouldNotifyBlocks() bool {
	return atomic.LoadUint32(&c.notifyBlocks) == 1
}

//loadtxfilter使用给定的过滤器来匹配事务
//以确定它们是否与客户相关。重置参数
//用于重置当前筛选器。
//
//当前支持的筛选器类型如下：
//[]B计算机地址
//[线]外点
//[]*线输出点
//映射[Wire.Outpoint]btcutil.address
//[]chainhash.hash
//[]*链哈希.hash
func (c *BitcoindClient) LoadTxFilter(reset bool, filters ...interface{}) error {
	if reset {
		select {
		case c.rescanUpdate <- struct{}{}:
		case <-c.quit:
			return ErrBitcoindClientShuttingDown
		}
	}

	updateFilter := func(filter interface{}) error {
		select {
		case c.rescanUpdate <- filter:
		case <-c.quit:
			return ErrBitcoindClientShuttingDown
		}

		return nil
	}

//为了使这个操作成为原子操作，我们将迭代
//筛选两次：第一次确保不存在任何不受支持的
//过滤器类型，第二个更新过滤器。
	for _, filter := range filters {
		switch filter := filter.(type) {
		case []btcutil.Address, []wire.OutPoint, []*wire.OutPoint,
			map[wire.OutPoint]btcutil.Address, []chainhash.Hash,
			[]*chainhash.Hash:

//继续检查下一个过滤器类型。
		default:
			return fmt.Errorf("unsupported filter type %T", filter)
		}
	}

	for _, filter := range filters {
		if err := updateFilter(filter); err != nil {
			return err
		}
	}

	return nil
}

//rescanblocks重新扫描通过的任何块，只返回
//与[]btcjson.blockdetails匹配。
func (c *BitcoindClient) RescanBlocks(
	blockHashes []chainhash.Hash) ([]btcjson.RescannedBlock, error) {

	rescannedBlocks := make([]btcjson.RescannedBlock, 0, len(blockHashes))
	for _, hash := range blockHashes {
		header, err := c.GetBlockHeaderVerbose(&hash)
		if err != nil {
			log.Warnf("Unable to get header %s from bitcoind: %s",
				hash, err)
			continue
		}

		block, err := c.GetBlock(&hash)
		if err != nil {
			log.Warnf("Unable to get block %s from bitcoind: %s",
				hash, err)
			continue
		}

		relevantTxs, err := c.filterBlock(block, header.Height, false)
		if len(relevantTxs) > 0 {
			rescannedBlock := btcjson.RescannedBlock{
				Hash: hash.String(),
			}
			for _, tx := range relevantTxs {
				rescannedBlock.Transactions = append(
					rescannedBlock.Transactions,
					hex.EncodeToString(tx.SerializedTx),
				)
			}

			rescannedBlocks = append(rescannedBlocks, rescannedBlock)
		}
	}

	return rescannedBlocks, nil
}

//使用给定哈希重新扫描块，直到当前块，
//在将传递的地址和输出添加到客户机的监视列表之后。
func (c *BitcoindClient) Rescan(blockHash *chainhash.Hash,
	addresses []btcutil.Address, outPoints map[wire.OutPoint]btcutil.Address) error {

//需要使用块哈希作为重新扫描的起始点。
	if blockHash == nil {
		return errors.New("rescan requires a starting block hash")
	}

//然后我们将使用给定的输出点和地址更新过滤器。
	select {
	case c.rescanUpdate <- addresses:
	case <-c.quit:
		return ErrBitcoindClientShuttingDown
	}

	select {
	case c.rescanUpdate <- outPoints:
	case <-c.quit:
		return ErrBitcoindClientShuttingDown
	}

//更新过滤器后，我们可以开始重新扫描。
	select {
	case c.rescanUpdate <- *blockHash:
	case <-c.quit:
		return ErrBitcoindClientShuttingDown
	}

	return nil
}

//Start使用支持比特币初始化比特币重新扫描客户端
//连接并启动处理重新扫描所需的所有goroutine
//以及ZMQ通知。
//
//注意：这是chain.interface接口的一部分。
func (c *BitcoindClient) Start() error {
	if !atomic.CompareAndSwapInt32(&c.started, 0, 1) {
		return nil
	}

//启动通知队列并立即调度
//客户端已连接到调用方的通知。这是需要的一些
//呼叫者在继续之前需要此通知。
	c.notificationQueue.Start()
	c.notificationQueue.ChanIn() <- ClientConnected{}

//检索链的最佳块。
	bestHash, bestHeight, err := c.GetBestBlock()
	if err != nil {
		return fmt.Errorf("unable to retrieve best block: %v", err)
	}
	bestHeader, err := c.GetBlockHeaderVerbose(bestHash)
	if err != nil {
		return fmt.Errorf("unable to retrieve header for best block: "+
			"%v", err)
	}

	c.bestBlockMtx.Lock()
	c.bestBlock = waddrmgr.BlockStamp{
		Hash:      *bestHash,
		Height:    bestHeight,
		Timestamp: time.Unix(bestHeader.Time, 0),
	}
	c.bestBlockMtx.Unlock()

//一旦客户端成功启动，我们将把它包括在集合中。
//重新扫描支持比特币连接的客户端，以便
//收到ZMQ事件通知。
	c.chainConn.AddClient(c)

	c.wg.Add(2)
	go c.rescanHandler()
	go c.ntfnHandler()

	return nil
}

//停止阻止比特币重新扫描客户端处理重新扫描和ZMQ
//通知。
//
//注意：这是chain.interface接口的一部分。
func (c *BitcoindClient) Stop() {
	if !atomic.CompareAndSwapInt32(&c.stopped, 0, 1) {
		return
	}

	close(c.quit)

//从比特币连接中删除此客户端的引用
//停止后阻止向其发送通知。
	c.chainConn.RemoveClient(c.id)

	c.notificationQueue.Stop()
}

//WaitForShutdown块，直到客户端完成断开连接
//处理程序已退出。
//
//注意：这是chain.interface接口的一部分。
func (c *BitcoindClient) WaitForShutdown() {
	c.wg.Wait()
}

//RescanHandler处理调用方触发链所需的逻辑
//再扫描。
//
//注意：这必须称为goroutine。
func (c *BitcoindClient) rescanHandler() {
	defer c.wg.Done()

	for {
		select {
		case update := <-c.rescanUpdate:
			switch update := update.(type) {

//我们正在清理过滤器。
			case struct{}:
				c.watchMtx.Lock()
				c.watchedOutPoints = make(map[wire.OutPoint]struct{})
				c.watchedAddresses = make(map[string]struct{})
				c.watchedTxs = make(map[chainhash.Hash]struct{})
				c.watchMtx.Unlock()

//我们正在将地址添加到筛选器中。
			case []btcutil.Address:
				c.watchMtx.Lock()
				for _, addr := range update {
					c.watchedAddresses[addr.String()] = struct{}{}
				}
				c.watchMtx.Unlock()

//我们正在将输出点添加到过滤器中。
			case []wire.OutPoint:
				c.watchMtx.Lock()
				for _, op := range update {
					c.watchedOutPoints[op] = struct{}{}
				}
				c.watchMtx.Unlock()
			case []*wire.OutPoint:
				c.watchMtx.Lock()
				for _, op := range update {
					c.watchedOutPoints[*op] = struct{}{}
				}
				c.watchMtx.Unlock()

//我们正在添加映射到脚本的输出点
//我们应该扫描到我们的过滤器。
			case map[wire.OutPoint]btcutil.Address:
				c.watchMtx.Lock()
				for op := range update {
					c.watchedOutPoints[op] = struct{}{}
				}
				c.watchMtx.Unlock()

//我们正在将事务添加到过滤器中。
			case []chainhash.Hash:
				c.watchMtx.Lock()
				for _, txid := range update {
					c.watchedTxs[txid] = struct{}{}
				}
				c.watchMtx.Unlock()
			case []*chainhash.Hash:
				c.watchMtx.Lock()
				for _, txid := range update {
					c.watchedTxs[*txid] = struct{}{}
				}
				c.watchMtx.Unlock()

//我们正在从哈希开始重新扫描。
			case chainhash.Hash:
				if err := c.rescan(update); err != nil {
					log.Errorf("Unable to complete chain "+
						"rescan: %v", err)
				}
			default:
				log.Warnf("Received unexpected filter type %T",
					update)
			}
		case <-c.quit:
			return
		}
	}
}

//ntfnhandler处理从备份检索zmq通知的逻辑
//比特币连接。
//
//注意：这必须称为goroutine。
func (c *BitcoindClient) ntfnHandler() {
	defer c.wg.Done()

	for {
		select {
		case tx := <-c.zmqTxNtfns:
			if _, _, err := c.filterTx(tx, nil, true); err != nil {
				log.Errorf("Unable to filter transaction %v: %v",
					tx.TxHash(), err)
			}
		case newBlock := <-c.zmqBlockNtfns:
//如果新块的前一个哈希匹配最佳
//我们知道散列，那么新的块就是下一个
//继承人，因此我们将更新最佳块以反映
//并确定此新块是否与
//我们现有的过滤器。
			c.bestBlockMtx.Lock()
			bestBlock := c.bestBlock
			c.bestBlockMtx.Unlock()
			if newBlock.Header.PrevBlock == bestBlock.Hash {
				newBlockHeight := bestBlock.Height + 1
				_, err := c.filterBlock(
					newBlock, newBlockHeight, true,
				)
				if err != nil {
					log.Errorf("Unable to filter block %v: %v",
						newBlock.BlockHash(), err)
					continue
				}

//通过成功筛选块，我们将
//让它成为我们新的最佳街区。
				bestBlock.Hash = newBlock.BlockHash()
				bestBlock.Height = newBlockHeight
				bestBlock.Timestamp = newBlock.Header.Timestamp

				c.bestBlockMtx.Lock()
				c.bestBlock = bestBlock
				c.bestBlockMtx.Unlock()

				continue
			}

//否则，我们遇到了REORG。
			if err := c.reorg(bestBlock, newBlock); err != nil {
				log.Errorf("Unable to process chain reorg: %v",
					err)
			}
		case <-c.quit:
			return
		}
	}
}

//setBirthday设置比特币Rescan客户端的生日。
//
//注意：这应该在客户机启动之前完成，以便
//正确履行职责。
func (c *BitcoindClient) SetBirthday(t time.Time) {
	c.birthday = t
}

//blockstamp返回客户端通知的最新块或错误
//如果客户端已关闭。
func (c *BitcoindClient) BlockStamp() (*waddrmgr.BlockStamp, error) {
	c.bestBlockMtx.RLock()
	bestBlock := c.bestBlock
	c.bestBlockMtx.RUnlock()

	return &bestBlock, nil
}

//OnBlockConnected是一个回调，每当新块
//检测。这将把一个blockconnected通知排队发送给调用者。
func (c *BitcoindClient) onBlockConnected(hash *chainhash.Hash, height int32,
	timestamp time.Time) {

	if c.shouldNotifyBlocks() {
		select {
		case c.notificationQueue.ChanIn() <- BlockConnected{
			Block: wtxmgr.Block{
				Hash:   *hash,
				Height: height,
			},
			Time: timestamp,
		}:
		case <-c.quit:
		}
	}
}

//onfilteredBlockConnected是一个备用回调，每当
//已检测到新块。它的作用与
//OnBlockConnected，但它还包括相关事务的列表
//在正在连接的块中找到。这将排队
//向调用方发送FilteredBlockConnected通知。
func (c *BitcoindClient) onFilteredBlockConnected(height int32,
	header *wire.BlockHeader, relevantTxs []*wtxmgr.TxRecord) {

	if c.shouldNotifyBlocks() {
		select {
		case c.notificationQueue.ChanIn() <- FilteredBlockConnected{
			Block: &wtxmgr.BlockMeta{
				Block: wtxmgr.Block{
					Hash:   header.BlockHash(),
					Height: height,
				},
				Time: header.Timestamp,
			},
			RelevantTxs: relevantTxs,
		}:
		case <-c.quit:
		}
	}
}

//onBlockDisconnected是一个回调，每当一个块
//断开的。这会将blockdisconnected通知排队发送给调用者
//断开块的详细信息。
func (c *BitcoindClient) onBlockDisconnected(hash *chainhash.Hash, height int32,
	timestamp time.Time) {

	if c.shouldNotifyBlocks() {
		select {
		case c.notificationQueue.ChanIn() <- BlockDisconnected{
			Block: wtxmgr.Block{
				Hash:   *hash,
				Height: height,
			},
			Time: timestamp,
		}:
		case <-c.quit:
		}
	}
}

//OnRelevantTx是一个回调，在事务相关时执行。
//给呼叫者。这意味着事务与
//客户端的不同筛选器。这会将相关的TTX通知排队到
//来电者。
func (c *BitcoindClient) onRelevantTx(tx *wtxmgr.TxRecord,
	blockDetails *btcjson.BlockDetails) {

	block, err := parseBlock(blockDetails)
	if err != nil {
		log.Errorf("Unable to send onRelevantTx notification, failed "+
			"parse block: %v", err)
		return
	}

	select {
	case c.notificationQueue.ChanIn() <- RelevantTx{
		TxRecord: tx,
		Block:    block,
	}:
	case <-c.quit:
	}
}

//OnRescanProgress是一个回调，每当Rescan处于
//进展。这将向调用者排队重新扫描进度通知
//当前重新扫描进度详细信息。
func (c *BitcoindClient) onRescanProgress(hash *chainhash.Hash, height int32,
	timestamp time.Time) {

	select {
	case c.notificationQueue.ChanIn() <- &RescanProgress{
		Hash:   hash,
		Height: height,
		Time:   timestamp,
	}:
	case <-c.quit:
	}
}

//OnRescanFinished是每当重新扫描
//完成了。这将对调用方的重新扫描完成通知进行排队
//重新扫描范围内最后一个块的详细信息。
func (c *BitcoindClient) onRescanFinished(hash *chainhash.Hash, height int32,
	timestamp time.Time) {

	log.Infof("Rescan finished at %d (%s)", height, hash)

	select {
	case c.notificationQueue.ChanIn() <- &RescanFinished{
		Hash:   hash,
		Height: height,
		Time:   timestamp,
	}:
	case <-c.quit:
	}
}

//REORG在链同步期间处理重组。这是
//与Rescan对REORG的处理不同。这个会倒回去直到它
//找到一个共同的祖先并通知此后所有的新块。
func (c *BitcoindClient) reorg(currentBlock waddrmgr.BlockStamp,
	reorgBlock *wire.MsgBlock) error {

	log.Debugf("Possible reorg at block %s", reorgBlock.BlockHash())

//根据导致
//ReRog这样，我们就可以保留我们需要的区块链
//检索。
	bestHash := reorgBlock.BlockHash()
	bestHeight, err := c.GetBlockHeight(&bestHash)
	if err != nil {
		return err
	}

	if bestHeight < currentBlock.Height {
		log.Debug("Detected multiple reorgs")
		return nil
	}

//现在我们将跟踪*链*已知的所有块，从
//从我们所知道的最好的木块到链条上最好的木块。
//这将使我们在未来的任何重组中快速前进。
	blocksToNotify := list.New()
	blocksToNotify.PushFront(reorgBlock)
	previousBlock := reorgBlock.Header.PrevBlock
	for i := bestHeight - 1; i >= currentBlock.Height; i-- {
		block, err := c.GetBlock(&previousBlock)
		if err != nil {
			return err
		}
		blocksToNotify.PushFront(block)
		previousBlock = block.Header.PrevBlock
	}

//使用上一个
//阻止来自每个头的哈希以避免任何竞争条件。如果我们
//遇到更多的重组，他们会排队，我们会重复这个循环。
//
//我们将从检索头到已知的最佳块开始。
	currentHeader, err := c.GetBlockHeader(&currentBlock.Hash)
	if err != nil {
		return err
	}

//然后，我们将在链中向后走，直到找到我们的共同点
//祖先。
	for previousBlock != currentHeader.PrevBlock {
//由于上一个哈希不匹配，当前块具有
//我们应该发送一个
//阻止它的断开连接通知。
		log.Debugf("Disconnecting block %d (%v)", currentBlock.Height,
			currentBlock.Hash)

		c.onBlockDisconnected(
			&currentBlock.Hash, currentBlock.Height,
			currentBlock.Timestamp,
		)

//我们当前的块现在应该反映上一个块
//继续共同祖先搜索。
		currentHeader, err = c.GetBlockHeader(&currentHeader.PrevBlock)
		if err != nil {
			return err
		}

		currentBlock.Height--
		currentBlock.Hash = currentHeader.PrevBlock
		currentBlock.Timestamp = currentHeader.Timestamp

//将正确的块存储在我们的列表中以便通知它
//一旦我们找到了我们共同的祖先。
		block, err := c.GetBlock(&previousBlock)
		if err != nil {
			return err
		}
		blocksToNotify.PushFront(block)
		previousBlock = block.Header.PrevBlock
	}

//从旧链条上断开最后一个挡块。从上一个
//旧链条和新链条之间的挡块保持不变，尖端将
//现在是最后一个共同的祖先。
	log.Debugf("Disconnecting block %d (%v)", currentBlock.Height,
		currentBlock.Hash)

	c.onBlockDisconnected(
		&currentBlock.Hash, currentBlock.Height, currentHeader.Timestamp,
	)

	currentBlock.Height--

//现在我们快进到新的街区，沿途通知。
	for blocksToNotify.Front() != nil {
		nextBlock := blocksToNotify.Front().Value.(*wire.MsgBlock)
		nextHeight := currentBlock.Height + 1
		nextHash := nextBlock.BlockHash()
		nextHeader, err := c.GetBlockHeader(&nextHash)
		if err != nil {
			return err
		}

		_, err = c.filterBlock(nextBlock, nextHeight, true)
		if err != nil {
			return err
		}

		currentBlock.Height = nextHeight
		currentBlock.Hash = nextHash
		currentBlock.Timestamp = nextHeader.Timestamp

		blocksToNotify.Remove(blocksToNotify.Front())
	}

	c.bestBlockMtx.Lock()
	c.bestBlock = currentBlock
	c.bestBlockMtx.Unlock()

	return nil
}

//filterblocks扫描filterblocks请求中包含的块
//相关地址。每个块将按顺序获取和过滤，
//返回包含匹配项的第一个块的filterBlocks响应
//地址。如果在请求的块范围内找不到匹配项，则
//返回的响应将为零。
//
//注意：这是chain.interface接口的一部分。
func (c *BitcoindClient) FilterBlocks(
	req *FilterBlocksRequest) (*FilterBlocksResponse, error) {

	blockFilterer := NewBlockFilterer(c.chainParams, req)

//迭代请求的块，从RPC客户端获取每个块。
//每个块将使用生成的反向地址索引进行扫描
//上面，如果找到任何地址，请提前破译。
	for i, block := range req.Blocks {
//托多（康纳）：添加预取，因为我们已经知道我们将
//提取*每个*块
		rawBlock, err := c.GetBlock(&block.Hash)
		if err != nil {
			return nil, err
		}

		if !blockFilterer.FilterBlock(rawBlock) {
			continue
		}

//如果在此中检测到任何外部或内部地址
//阻止，我们将它们返回给调用方以便重新扫描
//窗口可以用后续地址加宽。这个
//返回“batchindex”，以便调用方可以计算
//*下一个*块，从中重新开始。
		resp := &FilterBlocksResponse{
			BatchIndex:         uint32(i),
			BlockMeta:          block,
			FoundExternalAddrs: blockFilterer.FoundExternal,
			FoundInternalAddrs: blockFilterer.FoundInternal,
			FoundOutPoints:     blockFilterer.FoundOutPoints,
			RelevantTxns:       blockFilterer.RelevantTxns,
		}

		return resp, nil
	}

//找不到此范围的地址。
	return nil, nil
}

//Rescan使用比特币后端执行链的重新扫描，从
//指定哈希到最著名的哈希，同时注意
//在重新扫描期间发生。它使用被跟踪的地址和输出
//监视列表中的客户端。这只在队列处理中调用
//循环。
func (c *BitcoindClient) rescan(start chainhash.Hash) error {
	log.Infof("Starting rescan from block %s", start)

//我们首先得到已经处理过的最好的块。我们只使用
//高度，因为散列在重组过程中可以改变，我们
//通过测试从已知块到上一个块的连接来捕获
//块。
	bestHash, bestHeight, err := c.GetBestBlock()
	if err != nil {
		return err
	}
	bestHeader, err := c.GetBlockHeaderVerbose(bestHash)
	if err != nil {
		return err
	}
	bestBlock := waddrmgr.BlockStamp{
		Hash:      *bestHash,
		Height:    bestHeight,
		Timestamp: time.Unix(bestHeader.Time, 0),
	}

//创建按前进顺序排序的标题列表。我们将在
//由于链重新排序而需要回溯的事件。
	headers := list.New()
	previousHeader, err := c.GetBlockHeaderVerbose(&start)
	if err != nil {
		return err
	}
	previousHash, err := chainhash.NewHashFromStr(previousHeader.Hash)
	if err != nil {
		return err
	}
	headers.PushBack(previousHeader)

//用最后一个块将重新扫描完成的通知排队给调用方
//完成后在整个重新扫描过程中处理。
	defer c.onRescanFinished(
		previousHash, previousHeader.Height,
		time.Unix(previousHeader.Time, 0),
	)

//循环通过所有已知的比特币区块，注意
//ReOrgS
	for i := previousHeader.Height + 1; i <= bestBlock.Height; i++ {
		hash, err := c.GetBlockHash(int64(i))
		if err != nil {
			return err
		}

//如果前一个标题在钱包生日之前，请获取
//当前头并构造一个虚拟块，而不是
//正在获取整个块本身。当我们
//当我们知道它不可能的时候，就不必再去取整个街区了
//匹配我们的任何过滤器。
		var block *wire.MsgBlock
		afterBirthday := previousHeader.Time >= c.birthday.Unix()
		if !afterBirthday {
			header, err := c.GetBlockHeader(hash)
			if err != nil {
				return err
			}
			block = &wire.MsgBlock{
				Header: *header,
			}

			afterBirthday = c.birthday.Before(header.Timestamp)
			if afterBirthday {
				c.onRescanProgress(
					previousHash, i,
					block.Header.Timestamp,
				)
			}
		}

		if afterBirthday {
			block, err = c.GetBlock(hash)
			if err != nil {
				return err
			}
		}

		for block.Header.PrevBlock.String() != previousHeader.Hash {
//如果我们在这个for循环中，看起来我们已经
//重新组织。我们现在回到公共场所
//最佳链和已知链之间的祖先。
//
//首先，我们发出一个断开连接的块的信号来倒带
//重新扫描状态。
			c.onBlockDisconnected(
				previousHash, previousHeader.Height,
				time.Unix(previousHeader.Time, 0),
			)

//获取上一个最佳链块。
			hash, err := c.GetBlockHash(int64(i - 1))
			if err != nil {
				return err
			}
			block, err = c.GetBlock(hash)
			if err != nil {
				return err
			}

//那么，我们将得到前一个的标题
//块。
			if headers.Back() != nil {
//如果它已经在标题列表中，我们可以
//从那里拿出来然后把
//当前散列。
				headers.Remove(headers.Back())
				if headers.Back() != nil {
					previousHeader = headers.Back().
						Value.(*btcjson.GetBlockHeaderVerboseResult)
					previousHash, err = chainhash.NewHashFromStr(
						previousHeader.Hash,
					)
					if err != nil {
						return err
					}
				}
			} else {
//否则，我们就从比特币中得到。
				previousHash, err = chainhash.NewHashFromStr(
					previousHeader.PreviousHash,
				)
				if err != nil {
					return err
				}
				previousHeader, err = c.GetBlockHeaderVerbose(
					previousHash,
				)
				if err != nil {
					return err
				}
			}
		}

//既然我们已经确保没有遇到重组，我们将
//将当前块头添加到头列表中。
		blockHash := block.BlockHash()
		previousHash = &blockHash
		previousHeader = &btcjson.GetBlockHeaderVerboseResult{
			Hash:         blockHash.String(),
			Height:       i,
			PreviousHash: block.Header.PrevBlock.String(),
			Time:         block.Header.Timestamp.Unix(),
		}
		headers.PushBack(previousHeader)

//通知区块及其任何相关交易。
		if _, err = c.filterBlock(block, i, true); err != nil {
			return err
		}

		if i%10000 == 0 {
			c.onRescanProgress(
				previousHash, i, block.Header.Timestamp,
			)
		}

//如果我们已到达以前最著名的块，请检查
//确保基础节点没有额外同步
//阻碍。如果有，请更新已知块并继续
//重新扫描到那个点。
		if i == bestBlock.Height {
			bestHash, bestHeight, err = c.GetBestBlock()
			if err != nil {
				return err
			}
			bestHeader, err = c.GetBlockHeaderVerbose(bestHash)
			if err != nil {
				return err
			}

			bestBlock.Hash = *bestHash
			bestBlock.Height = bestHeight
			bestBlock.Timestamp = time.Unix(bestHeader.Time, 0)
		}
	}

	return nil
}

//filterblock过滤被监视的输出点和地址的块，并返回
//任何匹配的事务，一路发送通知。
func (c *BitcoindClient) filterBlock(block *wire.MsgBlock, height int32,
	notify bool) ([]*wtxmgr.TxRecord, error) {

//如果这个街区发生在客户生日之前，那么我们将跳过
//完全是这样。
	if block.Header.Timestamp.Before(c.birthday) {
		return nil, nil
	}

	if c.shouldNotifyBlocks() {
		log.Debugf("Filtering block %d (%s) with %d transactions",
			height, block.BlockHash(), len(block.Transactions))
	}

//创建一个块详细信息模板，用于所有已确认的
//在此块中找到的事务。
	blockHash := block.BlockHash()
	blockDetails := &btcjson.BlockDetails{
		Hash:   blockHash.String(),
		Height: height,
		Time:   block.Header.Timestamp.Unix(),
	}

//现在，我们将完成块中所有事务的跟踪
//与来电者有关的信息。
	var relevantTxs []*wtxmgr.TxRecord
	confirmedTxs := make(map[chainhash.Hash]struct{})
	for i, tx := range block.Transactions {
//使用此的索引更新块详细信息中的索引
//交易。
		blockDetails.Index = i
		isRelevant, rec, err := c.filterTx(tx, blockDetails, notify)
		if err != nil {
			log.Warnf("Unable to filter transaction %v: %v",
				tx.TxHash(), err)
			continue
		}

		if isRelevant {
			relevantTxs = append(relevantTxs, rec)
			confirmedTxs[tx.TxHash()] = struct{}{}
		}
	}

//通过设置块的“已确认”来更新到期映射
//事务和删除已确认的mempool中的任何
//超过288个街区以前。
	c.watchMtx.Lock()
	c.expiredMempool[height] = confirmedTxs
	if oldBlock, ok := c.expiredMempool[height-288]; ok {
		for txHash := range oldBlock {
			delete(c.mempool, txHash)
		}
		delete(c.expiredMempool, height-288)
	}
	c.watchMtx.Unlock()

	if notify {
		c.onFilteredBlockConnected(height, &block.Header, relevantTxs)
		c.onBlockConnected(&blockHash, height, block.Header.Timestamp)
	}

	return relevantTxs, nil
}

//filtertx通过以下方式确定事务是否与客户端相关
//检查客户端的不同筛选器。
func (c *BitcoindClient) filterTx(tx *wire.MsgTx,
	blockDetails *btcjson.BlockDetails,
	notify bool) (bool, *wtxmgr.TxRecord, error) {

	txDetails := btcutil.NewTx(tx)
	if blockDetails != nil {
		txDetails.SetIndex(blockDetails.Index)
	}

	rec, err := wtxmgr.NewTxRecordFromMsgTx(txDetails.MsgTx(), time.Now())
	if err != nil {
		log.Errorf("Cannot create transaction record for relevant "+
			"tx: %v", err)
		return false, nil, err
	}
	if blockDetails != nil {
		rec.Received = time.Unix(blockDetails.Time, 0)
	}

//我们将通过持有锁开始过滤过程，以确保
//与过滤器中的当前内容完全匹配。
	c.watchMtx.Lock()
	defer c.watchMtx.Unlock()

//如果我们已经看到这个交易，现在已经确认了，
//然后我们将通过立即发送
//通知调用者筛选器匹配。
	if _, ok := c.mempool[tx.TxHash()]; ok {
		if notify && blockDetails != nil {
			c.onRelevantTx(rec, blockDetails)
		}
		return true, rec, nil
	}

//否则，这是一个新的交易，我们还没有看到。我们需要
//以确定此事务是否与调用方相关。
	var isRelevant bool

//我们将首先检查所有输入并确定它是否花费
//一个现有的输出点或一个pkscript在我们的监视中被编码为地址
//名单。
	for _, txIn := range tx.TxIn {
//如果它与监视列表中的输出点匹配，我们可以退出
//早循环。
		if _, ok := c.watchedOutPoints[txIn.PreviousOutPoint]; ok {
			isRelevant = true
			break
		}

//否则，我们将检查它是否与
//作为地址编码的监视列表。为此，我们将
//输入要花费的输出的pkscript。
		pkScript, err := txscript.ComputePkScript(
			txIn.SignatureScript, txIn.Witness,
		)
		if err != nil {
//可以安全跳过非标准输出。
			continue
		}
		addr, err := pkScript.Address(c.chainParams)
		if err != nil {
//可以安全跳过非标准输出。
			continue
		}
		if _, ok := c.watchedAddresses[addr.String()]; ok {
			isRelevant = true
			break
		}
	}

//我们还将循环查看它的输出，以确定它是否值得
//任何当前被监视的地址。如果输出匹配，我们将
//把它加入我们的观察名单。
	for i, txOut := range tx.TxOut {
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(
			txOut.PkScript, c.chainParams,
		)
		if err != nil {
//可以安全跳过非标准输出。
			continue
		}

		for _, addr := range addrs {
			if _, ok := c.watchedAddresses[addr.String()]; ok {
				isRelevant = true
				op := wire.OutPoint{
					Hash:  tx.TxHash(),
					Index: uint32(i),
				}
				c.watchedOutPoints[op] = struct{}{}
			}
		}
	}

//如果这笔交易没有支付到我们的任何监视地址，我们将
//检查我们当前是否正在监视此事务的哈希。
	if !isRelevant {
		if _, ok := c.watchedTxs[tx.TxHash()]; ok {
			isRelevant = true
		}
	}

//如果交易与我们无关，我们可以直接退出。
	if !isRelevant {
		return false, rec, nil
	}

//否则，事务与我们的过滤器匹配，因此我们应该
//通知。如果还未确认，我们会把它包括在
//我们的mempool，以便它也可以作为
//一旦确认，就进行筛选阻止连接。
	if blockDetails == nil {
		c.mempool[tx.TxHash()] = struct{}{}
	}

	c.onRelevantTx(rec, blockDetails)

	return true, rec, nil
}
