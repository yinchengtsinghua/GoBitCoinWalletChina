
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package chain

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/gcs"
	"github.com/btcsuite/btcutil/gcs/builder"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/wtxmgr"
	"github.com/lightninglabs/neutrino"
)

//中微子客户端是btcwalet-chain.interface接口的一个实现。
type NeutrinoClient struct {
	CS *neutrino.ChainService

	chainParams *chaincfg.Params

//我们目前支持每个客户端一个重新扫描/通知goroutine
	rescan *neutrino.Rescan

	enqueueNotification chan interface{}
	dequeueNotification chan interface{}
	startTime           time.Time
	lastProgressSent    bool
	currentBlock        chan *waddrmgr.BlockStamp

	quit       chan struct{}
	rescanQuit chan struct{}
	rescanErr  <-chan error
	wg         sync.WaitGroup
	started    bool
	scanning   bool
	finished   bool
	isRescan   bool

	clientMtx sync.Mutex
}

//新中微子客户端创建一个新的中微子客户端结构
//链接服务。
func NewNeutrinoClient(chainParams *chaincfg.Params,
	chainService *neutrino.ChainService) *NeutrinoClient {

	return &NeutrinoClient{
		CS:          chainService,
		chainParams: chainParams,
	}
}

//后端返回驱动程序的名称。
func (s *NeutrinoClient) BackEnd() string {
	return "neutrino"
}

//Start复制RPC客户端的Start方法。
func (s *NeutrinoClient) Start() error {
	s.CS.Start()
	s.clientMtx.Lock()
	defer s.clientMtx.Unlock()
	if !s.started {
		s.enqueueNotification = make(chan interface{})
		s.dequeueNotification = make(chan interface{})
		s.currentBlock = make(chan *waddrmgr.BlockStamp)
		s.quit = make(chan struct{})
		s.started = true
		s.wg.Add(1)
		go func() {
			select {
			case s.enqueueNotification <- ClientConnected{}:
			case <-s.quit:
			}
		}()
		go s.notificationHandler()
	}
	return nil
}

//Stop replicates the RPC client's Stop method.
func (s *NeutrinoClient) Stop() {
	s.clientMtx.Lock()
	defer s.clientMtx.Unlock()
	if !s.started {
		return
	}
	close(s.quit)
	s.started = false
}

//WaitForShutdown复制RPC客户端的WaitForShutdown方法。
func (s *NeutrinoClient) WaitForShutdown() {
	s.wg.Wait()
}

//getblock复制RPC客户端的getblock命令。
func (s *NeutrinoClient) GetBlock(hash *chainhash.Hash) (*wire.MsgBlock, error) {
//TODO（roasbef）：添加块缓存？
//*哪些驱逐策略？取决于用例
//块缓存应该在中微子内部而不是btcwallet中吗？
	block, err := s.CS.GetBlock(*hash)
	if err != nil {
		return nil, err
	}
	return block.MsgBlock(), nil
}

//getBlockHeight根据块的哈希值获取块的高度。它作为一个
//钱包包使用getblockverbosetxanc的替代品
//因为我们不能返回FutureGetBlockVerboseResult，因为
//基础类型对RpcClient是私有的。
func (s *NeutrinoClient) GetBlockHeight(hash *chainhash.Hash) (int32, error) {
	return s.CS.GetBlockHeight(hash)
}

//GetBestBlock复制RPC客户端的GetBestBlock命令。
func (s *NeutrinoClient) GetBestBlock() (*chainhash.Hash, int32, error) {
	chainTip, err := s.CS.BestBlock()
	if err != nil {
		return nil, 0, err
	}

	return &chainTip.Hash, chainTip.Height, nil
}

//blockstamp返回客户端通知的最新块或错误
//如果客户端已关闭。
func (s *NeutrinoClient) BlockStamp() (*waddrmgr.BlockStamp, error) {
	select {
	case bs := <-s.currentBlock:
		return bs, nil
	case <-s.quit:
		return nil, errors.New("disconnected")
	}
}

//GetBlockHash返回给定高度的块哈希，或者如果
//客户端已关闭，或者块高度处的哈希不存在，或者
//是未知的。
func (s *NeutrinoClient) GetBlockHash(height int64) (*chainhash.Hash, error) {
	return s.CS.GetBlockHash(height)
}

//GetBlockHeader返回给定块哈希的块头或错误
//如果客户端已关闭或哈希不存在或未知。
func (s *NeutrinoClient) GetBlockHeader(
	blockHash *chainhash.Hash) (*wire.BlockHeader, error) {
	return s.CS.GetBlockHeader(blockHash)
}

//sendrawtransaction复制RPC客户端的sendrawtransaction命令。
func (s *NeutrinoClient) SendRawTransaction(tx *wire.MsgTx, allowHighFees bool) (
	*chainhash.Hash, error) {
	err := s.CS.SendTransaction(tx)
	if err != nil {
		return nil, err
	}
	hash := tx.TxHash()
	return &hash, nil
}

//filterblocks扫描filterblocks请求中包含的块
//相关地址。对于每个请求的块，对应的压缩
//将首先检查筛选器是否匹配，跳过那些不报告的
//什么都行。如果过滤器返回一个postive匹配，则整个块将
//提取并过滤。此方法返回第一个
//包含匹配地址的块。如果在以下范围内找不到匹配项
//请求的块，返回的响应将为零。
func (s *NeutrinoClient) FilterBlocks(
	req *FilterBlocksRequest) (*FilterBlocksResponse, error) {

	blockFilterer := NewBlockFilterer(s.chainParams, req)

//使用包含的地址和输出点构造观察列表
//在筛选器块请求中。
	watchList, err := buildFilterBlocksWatchList(req)
	if err != nil {
		return nil, err
	}

//迭代请求的块，获取
//并将其与上面生成的监视列表进行匹配。如果
//过滤器返回一个正匹配，然后请求整个块
//并使用块过滤器扫描地址。
	for i, blk := range req.Blocks {
		filter, err := s.pollCFilter(&blk.Hash)
		if err != nil {
			return nil, err
		}

//跳过任何空过滤器。
		if filter == nil || filter.N() == 0 {
			continue
		}

		key := builder.DeriveKey(&blk.Hash)
		matched, err := filter.MatchAny(key, watchList)
		if err != nil {
			return nil, err
		} else if !matched {
			continue
		}

		log.Infof("Fetching block height=%d hash=%v",
			blk.Height, blk.Hash)

//todo（conner）：只能通过提取来优化带宽
//剥离块
		rawBlock, err := s.GetBlock(&blk.Hash)
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
			BlockMeta:          blk,
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

//buildFilterBlockSwatchList构造用于与
//C来自filterblocks请求的筛选器。监视列表将填充所有
//中包含的外部地址、内部地址和输出点
//请求。
func buildFilterBlocksWatchList(req *FilterBlocksRequest) ([][]byte, error) {
//构建一个包含所有脚本地址的监视列表
//请求的内部和外部地址，以及
//当前正在监视的一组输出点。
	watchListSize := len(req.ExternalAddrs) +
		len(req.InternalAddrs) +
		len(req.WatchedOutPoints)

	watchList := make([][]byte, 0, watchListSize)

	for _, addr := range req.ExternalAddrs {
		p2shAddr, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return nil, err
		}

		watchList = append(watchList, p2shAddr)
	}

	for _, addr := range req.InternalAddrs {
		p2shAddr, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return nil, err
		}

		watchList = append(watchList, p2shAddr)
	}

	for _, addr := range req.WatchedOutPoints {
		addr, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return nil, err
		}

		watchList = append(watchList, addr)
	}

	return watchList, nil
}

//pollcfilter尝试从中微子客户机获取一个cffilter。这是
//用来绕过过滤器头可能落后于
//已知的最高块头。
func (s *NeutrinoClient) pollCFilter(hash *chainhash.Hash) (*gcs.Filter, error) {
	var (
		filter *gcs.Filter
		err    error
		count  int
	)

	const maxFilterRetries = 50
	for count < maxFilterRetries {
		if count > 0 {
			time.Sleep(100 * time.Millisecond)
		}

		filter, err = s.CS.GetCFilter(*hash, wire.GCSFilterRegular)
		if err != nil {
			count++
			continue
		}

		return filter, nil
	}

	return nil, err
}

//rescan复制RPC客户端的rescan命令。
func (s *NeutrinoClient) Rescan(startHash *chainhash.Hash, addrs []btcutil.Address,
	outPoints map[wire.OutPoint]btcutil.Address) error {

	s.clientMtx.Lock()
	defer s.clientMtx.Unlock()
	if !s.started {
		return fmt.Errorf("can't do a rescan when the chain client " +
			"is not started")
	}
	if s.scanning {
//通过终止现有的重新扫描来重新启动重新扫描。
		close(s.rescanQuit)
		s.clientMtx.Unlock()
		s.rescan.WaitForShutdown()
		s.clientMtx.Lock()
		s.rescan = nil
		s.rescanErr = nil
	}
	s.rescanQuit = make(chan struct{})
	s.scanning = true
	s.finished = false
	s.lastProgressSent = false
	s.isRescan = true

	bestBlock, err := s.CS.BestBlock()
	if err != nil {
		return fmt.Errorf("Can't get chain service's best block: %s", err)
	}
	header, err := s.CS.GetBlockHeader(&bestBlock.Hash)
	if err != nil {
		return fmt.Errorf("Can't get block header for hash %v: %s",
			bestBlock.Hash, err)
	}

//如果钱包已经被完全套住，或者重新扫描已经开始
//如果状态显示“新”钱包，我们将发送一个
//指示重新扫描已“完成”的通知。
	if header.BlockHash() == *startHash {
		s.finished = true
		select {
		case s.enqueueNotification <- &RescanFinished{
			Hash:   startHash,
			Height: int32(bestBlock.Height),
			Time:   header.Timestamp,
		}:
		case <-s.quit:
			return nil
		case <-s.rescanQuit:
			return nil
		}
	}

	var inputsToWatch []neutrino.InputWithScript
	for op, addr := range outPoints {
		addrScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
		}

		inputsToWatch = append(inputsToWatch, neutrino.InputWithScript{
			OutPoint: op,
			PkScript: addrScript,
		})
	}

	newRescan := s.CS.NewRescan(
		neutrino.NotificationHandlers(rpcclient.NotificationHandlers{
			OnBlockConnected:         s.onBlockConnected,
			OnFilteredBlockConnected: s.onFilteredBlockConnected,
			OnBlockDisconnected:      s.onBlockDisconnected,
		}),
		neutrino.StartBlock(&waddrmgr.BlockStamp{Hash: *startHash}),
		neutrino.StartTime(s.startTime),
		neutrino.QuitChan(s.rescanQuit),
		neutrino.WatchAddrs(addrs...),
		neutrino.WatchInputs(inputsToWatch...),
	)
	s.rescan = newRescan
	s.rescanErr = s.rescan.Start()

	return nil
}

//notifyblocks复制RPC客户端的notifyblocks命令。
func (s *NeutrinoClient) NotifyBlocks() error {
	s.clientMtx.Lock()
//如果我们正在扫描，我们已经通知了块。否则，
//开始重新扫描而不查看任何地址。
	if !s.scanning {
		s.clientMtx.Unlock()
		return s.NotifyReceived([]btcutil.Address{})
	}
	s.clientMtx.Unlock()
	return nil
}

//notifyreceived复制RPC客户端的notifyreceived命令。
func (s *NeutrinoClient) NotifyReceived(addrs []btcutil.Address) error {
	s.clientMtx.Lock()

//如果我们有一个重新扫描运行，我们只需要添加适当的
//监视列表的地址。
	if s.scanning {
		s.clientMtx.Unlock()
		return s.rescan.Update(neutrino.AddAddrs(addrs...))
	}

	s.rescanQuit = make(chan struct{})
	s.scanning = true

//不需要重新扫描完成或重新扫描进度通知。
	s.finished = true
	s.lastProgressSent = true

//仅使用指定的地址重新扫描。
	newRescan := s.CS.NewRescan(
		neutrino.NotificationHandlers(rpcclient.NotificationHandlers{
			OnBlockConnected:         s.onBlockConnected,
			OnFilteredBlockConnected: s.onFilteredBlockConnected,
			OnBlockDisconnected:      s.onBlockDisconnected,
		}),
		neutrino.StartTime(s.startTime),
		neutrino.QuitChan(s.rescanQuit),
		neutrino.WatchAddrs(addrs...),
	)
	s.rescan = newRescan
	s.rescanErr = s.rescan.Start()
	s.clientMtx.Unlock()
	return nil
}

//通知复制RPC客户端的通知方法。
func (s *NeutrinoClient) Notifications() <-chan interface{} {
	return s.dequeueNotification
}

//setStartTime是设置钱包生日的非接口方法
//使用此对象。因为当前只有一次重新扫描
//支持，只需设置一个生日。这不会完全重新启动
//正在运行重新扫描，因此不应在重新扫描运行时使用它来更新。
//TODO:在将每个中微子客户机分解为多个重新扫描时，添加一个
//每位客户的生日。
func (s *NeutrinoClient) SetStartTime(startTime time.Time) {
	s.clientMtx.Lock()
	defer s.clientMtx.Unlock()

	s.startTime = startTime
}

//OnFilteredBlockConnected向通知发送适当的通知
//通道。
func (s *NeutrinoClient) onFilteredBlockConnected(height int32,
	header *wire.BlockHeader, relevantTxs []*btcutil.Tx) {
	ntfn := FilteredBlockConnected{
		Block: &wtxmgr.BlockMeta{
			Block: wtxmgr.Block{
				Hash:   header.BlockHash(),
				Height: height,
			},
			Time: header.Timestamp,
		},
	}
	for _, tx := range relevantTxs {
		rec, err := wtxmgr.NewTxRecordFromMsgTx(tx.MsgTx(),
			header.Timestamp)
		if err != nil {
			log.Errorf("Cannot create transaction record for "+
				"relevant tx: %s", err)
//托多：回来了？
			continue
		}
		ntfn.RelevantTxs = append(ntfn.RelevantTxs, rec)
	}
	select {
	case s.enqueueNotification <- ntfn:
	case <-s.quit:
		return
	case <-s.rescanQuit:
		return
	}

//处理重新扫描完成的通知（如果需要）。
	bs, err := s.CS.BestBlock()
	if err != nil {
		log.Errorf("Can't get chain service's best block: %s", err)
		return
	}

	if bs.Hash == header.BlockHash() {
//只发送一次RescanFinished通知。
		s.clientMtx.Lock()
		if s.finished {
			s.clientMtx.Unlock()
			return
		}
//只发送一次重新扫描完成的通知
//底层的链服务将自身视为当前的。
		current := s.CS.IsCurrent() && s.lastProgressSent
		if current {
			s.finished = true
		}
		s.clientMtx.Unlock()
		if current {
			select {
			case s.enqueueNotification <- &RescanFinished{
				Hash:   &bs.Hash,
				Height: bs.Height,
				Time:   header.Timestamp,
			}:
			case <-s.quit:
				return
			case <-s.rescanQuit:
				return
			}
		}
	}
}

//OnBlockDisconnected向通知发送适当的通知
//通道。
func (s *NeutrinoClient) onBlockDisconnected(hash *chainhash.Hash, height int32,
	t time.Time) {
	select {
	case s.enqueueNotification <- BlockDisconnected{
		Block: wtxmgr.Block{
			Hash:   *hash,
			Height: height,
		},
		Time: t,
	}:
	case <-s.quit:
	case <-s.rescanQuit:
	}
}

func (s *NeutrinoClient) onBlockConnected(hash *chainhash.Hash, height int32,
	time time.Time) {
//TODO:将此闭包移出并参数化？有用吗？
//在外面？
	sendRescanProgress := func() {
		select {
		case s.enqueueNotification <- &RescanProgress{
			Hash:   hash,
			Height: height,
			Time:   time,
		}:
		case <-s.quit:
		case <-s.rescanQuit:
		}
	}
//仅在处理块时发送blockconnected通知
//在生日之前。否则，我们只能使用
//重新扫描进度通知。
	if time.Before(s.startTime) {
//每10公里发送一次重新扫描进度通知。
		if height%10000 == 0 {
			s.clientMtx.Lock()
			shouldSend := s.isRescan && !s.finished
			s.clientMtx.Unlock()
			if shouldSend {
				sendRescanProgress()
			}
		}
	} else {
//如果我们正在检查，请发送重新扫描进度通知
//生日前和生日后街区之间的界限，
//注意我们已经发送了。
		s.clientMtx.Lock()
		if !s.lastProgressSent {
			shouldSend := s.isRescan && !s.finished
			if shouldSend {
				s.clientMtx.Unlock()
				sendRescanProgress()
				s.clientMtx.Lock()
				s.lastProgressSent = true
			}
		}
		s.clientMtx.Unlock()
		select {
		case s.enqueueNotification <- BlockConnected{
			Block: wtxmgr.Block{
				Hash:   *hash,
				Height: height,
			},
			Time: time,
		}:
		case <-s.quit:
		case <-s.rescanQuit:
		}
	}
}

//通知处理程序队列和出列通知。目前有
//队列上没有边界，因此应连续读取出列通道
//避免内存不足。
func (s *NeutrinoClient) notificationHandler() {
	hash, height, err := s.GetBestBlock()
	if err != nil {
		log.Errorf("Failed to get best block from chain service: %s",
			err)
		s.Stop()
		s.wg.Done()
		return
	}

	bs := &waddrmgr.BlockStamp{Hash: *hash, Height: height}

//TODO:而不是将此作为所有类型的
//通知，尝试删除那些稍后进入队列的通知
//可以完全使等待处理的无效。例如，
//块高度较大的块连接通知可以删除
//需要处理之前仍在等待的BlockConnected通知
//在这里。

	var notifications []interface{}
	enqueue := s.enqueueNotification
	var dequeue chan interface{}
	var next interface{}
out:
	for {
		s.clientMtx.Lock()
		rescanErr := s.rescanErr
		s.clientMtx.Unlock()
		select {
		case n, ok := <-enqueue:
			if !ok {
//如果没有通知排队等待处理，
//队列已完成。
				if len(notifications) == 0 {
					break out
				}
//无通道，因此无法再进行读取。
				enqueue = nil
				continue
			}
			if len(notifications) == 0 {
				next = n
				dequeue = s.dequeueNotification
			}
			notifications = append(notifications, n)

		case dequeue <- next:
			if n, ok := next.(BlockConnected); ok {
				bs = &waddrmgr.BlockStamp{
					Height: n.Height,
					Hash:   n.Hash,
				}
			}

			notifications[0] = nil
			notifications = notifications[1:]
			if len(notifications) != 0 {
				next = notifications[0]
			} else {
//如果无法将更多通知排队，则
//队列已完成。
				if enqueue == nil {
					break out
				}
				dequeue = nil
			}

		case err := <-rescanErr:
			if err != nil {
				log.Errorf("Neutrino rescan ended with error: %s", err)
			}

		case s.currentBlock <- bs:

		case <-s.quit:
			break out
		}
	}

	s.Stop()
	close(s.dequeueNotification)
	s.wg.Done()
}
