
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package chain

import (
	"errors"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/gcs"
	"github.com/btcsuite/btcutil/gcs/builder"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/wtxmgr"
)

//rpc client表示与比特币RPC服务器的持久客户端连接
//有关当前最佳区块链的信息。
type RPCClient struct {
	*rpcclient.Client
connConfig        *rpcclient.ConnConfig //在未变形的场地周围工作
	chainParams       *chaincfg.Params
	reconnectAttempts int

	enqueueNotification chan interface{}
	dequeueNotification chan interface{}
	currentBlock        chan *waddrmgr.BlockStamp

	quit    chan struct{}
	wg      sync.WaitGroup
	started bool
	quitMtx sync.Mutex
}

//newrpcclient创建到服务器的客户端连接，由
//连接字符串。如果disabletls为false，则远程RPC证书必须
//在certs切片中提供。连接没有立即建立，
//但必须使用start方法完成。如果远程服务器没有
//在同一比特币网络上操作，如通过的链所述。
//参数，连接将断开。
func NewRPCClient(chainParams *chaincfg.Params, connect, user, pass string, certs []byte,
	disableTLS bool, reconnectAttempts int) (*RPCClient, error) {

	if reconnectAttempts < 0 {
		return nil, errors.New("reconnectAttempts must be positive")
	}

	client := &RPCClient{
		connConfig: &rpcclient.ConnConfig{
			Host:                 connect,
			Endpoint:             "ws",
			User:                 user,
			Pass:                 pass,
			Certificates:         certs,
			DisableAutoReconnect: false,
			DisableConnectOnNew:  true,
			DisableTLS:           disableTLS,
		},
		chainParams:         chainParams,
		reconnectAttempts:   reconnectAttempts,
		enqueueNotification: make(chan interface{}),
		dequeueNotification: make(chan interface{}),
		currentBlock:        make(chan *waddrmgr.BlockStamp),
		quit:                make(chan struct{}),
	}
	ntfnCallbacks := &rpcclient.NotificationHandlers{
		OnClientConnected:   client.onClientConnect,
		OnBlockConnected:    client.onBlockConnected,
		OnBlockDisconnected: client.onBlockDisconnected,
		OnRecvTx:            client.onRecvTx,
		OnRedeemingTx:       client.onRedeemingTx,
		OnRescanFinished:    client.onRescanFinished,
		OnRescanProgress:    client.onRescanProgress,
	}
	rpcClient, err := rpcclient.New(client.connConfig, ntfnCallbacks)
	if err != nil {
		return nil, err
	}
	client.Client = rpcClient
	return client, nil
}

//后端返回驱动程序的名称。
func (c *RPCClient) BackEnd() string {
	return "btcd"
}

//开始尝试与远程服务器建立客户端连接。
//如果成功，将启动处理程序goroutine来处理通知
//由服务器发送。在有限的连接尝试之后，此
//函数放弃，因此不会永远阻塞等待
//要建立到可能不存在的服务器的连接。
func (c *RPCClient) Start() error {
	err := c.Connect(c.reconnectAttempts)
	if err != nil {
		return err
	}

//验证服务器是否在预期网络上运行。
	net, err := c.GetCurrentNet()
	if err != nil {
		c.Disconnect()
		return err
	}
	if net != c.chainParams.Net {
		c.Disconnect()
		return errors.New("mismatched networks")
	}

	c.quitMtx.Lock()
	c.started = true
	c.quitMtx.Unlock()

	c.wg.Add(1)
	go c.handler()
	return nil
}

//停止断开客户端连接，并发出关闭所有goroutine的信号
//从开始。
func (c *RPCClient) Stop() {
	c.quitMtx.Lock()
	select {
	case <-c.quit:
	default:
		close(c.quit)
		c.Client.Shutdown()

		if !c.started {
			close(c.dequeueNotification)
		}
	}
	c.quitMtx.Unlock()
}

//rescan用一个附加的参数包装普通的rescan命令，
//允许我们将一个联合点映射到它支付的链中的地址。
//这在使用bip 158过滤器时很有用，因为它们包括prev pkscript
//而不是全速前进。
func (c *RPCClient) Rescan(startHash *chainhash.Hash, addrs []btcutil.Address,
	outPoints map[wire.OutPoint]btcutil.Address) error {

	flatOutpoints := make([]*wire.OutPoint, 0, len(outPoints))
	for ops := range outPoints {
		flatOutpoints = append(flatOutpoints, &ops)
	}

	return c.Client.Rescan(startHash, addrs, flatOutpoints)
}

//WaitForShutdown块，直到两个客户端都完成断开连接
//所有处理程序都已退出。
func (c *RPCClient) WaitForShutdown() {
	c.Client.WaitForShutdown()
	c.wg.Wait()
}

//通知返回远程服务器发送的已分析通知的通道。
//比特币RPC服务器。此通道必须连续读取或处理
//可能因内存不足而中止，因为未读通知排队等待
//后来阅读。
func (c *RPCClient) Notifications() <-chan interface{} {
	return c.dequeueNotification
}

//blockstamp返回客户端通知的最新块或错误
//如果客户端已关闭。
func (c *RPCClient) BlockStamp() (*waddrmgr.BlockStamp, error) {
	select {
	case bs := <-c.currentBlock:
		return bs, nil
	case <-c.quit:
		return nil, errors.New("disconnected")
	}
}

//filterblocks扫描filterblocks请求中包含的块
//相关地址。对于每个请求的块，对应的压缩
//将首先检查筛选器是否匹配，跳过那些不报告的
//什么都行。如果过滤器返回一个postive匹配，则整个块将
//提取并过滤。此方法返回第一个
//包含匹配地址的块。如果在以下范围内找不到匹配项
//请求的块，返回的响应将为零。
func (c *RPCClient) FilterBlocks(
	req *FilterBlocksRequest) (*FilterBlocksResponse, error) {

	blockFilterer := NewBlockFilterer(c.chainParams, req)

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
		rawFilter, err := c.GetCFilter(&blk.Hash, wire.GCSFilterRegular)
		if err != nil {
			return nil, err
		}

//确保过滤器足够大，可以反序列化。
		if len(rawFilter.Data) < 4 {
			continue
		}

		filter, err := gcs.FromNBytes(
			builder.DefaultP, builder.DefaultM, rawFilter.Data,
		)
		if err != nil {
			return nil, err
		}

//跳过任何空过滤器。
		if filter.N() == 0 {
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

		rawBlock, err := c.GetBlock(&blk.Hash)
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

//parseBlock解析块的btcws定义，将Tx挖掘到
//wtxmgr包的块结构和块索引。这样做了
//这里是因为rpcclient不能很好地为我们分析这个问题。
func parseBlock(block *btcjson.BlockDetails) (*wtxmgr.BlockMeta, error) {
	if block == nil {
		return nil, nil
	}
	blkHash, err := chainhash.NewHashFromStr(block.Hash)
	if err != nil {
		return nil, err
	}
	blk := &wtxmgr.BlockMeta{
		Block: wtxmgr.Block{
			Height: block.Height,
			Hash:   *blkHash,
		},
		Time: time.Unix(block.Time, 0),
	}
	return blk, nil
}

func (c *RPCClient) onClientConnect() {
	select {
	case c.enqueueNotification <- ClientConnected{}:
	case <-c.quit:
	}
}

func (c *RPCClient) onBlockConnected(hash *chainhash.Hash, height int32, time time.Time) {
	select {
	case c.enqueueNotification <- BlockConnected{
		Block: wtxmgr.Block{
			Hash:   *hash,
			Height: height,
		},
		Time: time,
	}:
	case <-c.quit:
	}
}

func (c *RPCClient) onBlockDisconnected(hash *chainhash.Hash, height int32, time time.Time) {
	select {
	case c.enqueueNotification <- BlockDisconnected{
		Block: wtxmgr.Block{
			Hash:   *hash,
			Height: height,
		},
		Time: time,
	}:
	case <-c.quit:
	}
}

func (c *RPCClient) onRecvTx(tx *btcutil.Tx, block *btcjson.BlockDetails) {
	blk, err := parseBlock(block)
	if err != nil {
//记录并删除不正确的通知。
		log.Errorf("recvtx notification bad block: %v", err)
		return
	}

	rec, err := wtxmgr.NewTxRecordFromMsgTx(tx.MsgTx(), time.Now())
	if err != nil {
		log.Errorf("Cannot create transaction record for relevant "+
			"tx: %v", err)
		return
	}
	select {
	case c.enqueueNotification <- RelevantTx{rec, blk}:
	case <-c.quit:
	}
}

func (c *RPCClient) onRedeemingTx(tx *btcutil.Tx, block *btcjson.BlockDetails) {
//处理方式与recvtx通知完全相同。
	c.onRecvTx(tx, block)
}

func (c *RPCClient) onRescanProgress(hash *chainhash.Hash, height int32, blkTime time.Time) {
	select {
	case c.enqueueNotification <- &RescanProgress{hash, height, blkTime}:
	case <-c.quit:
	}
}

func (c *RPCClient) onRescanFinished(hash *chainhash.Hash, height int32, blkTime time.Time) {
	select {
	case c.enqueueNotification <- &RescanFinished{hash, height, blkTime}:
	case <-c.quit:
	}

}

//处理程序维护通知队列和当前状态（最佳
//链条的）挡块。
func (c *RPCClient) handler() {
	hash, height, err := c.GetBestBlock()
	if err != nil {
		log.Errorf("Failed to receive best block from chain server: %v", err)
		c.Stop()
		c.wg.Done()
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
	enqueue := c.enqueueNotification
	var dequeue chan interface{}
	var next interface{}
out:
	for {
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
				dequeue = c.dequeueNotification
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

		case c.currentBlock <- bs:

		case <-c.quit:
			break out
		}
	}

	c.Stop()
	close(c.dequeueNotification)
	c.wg.Done()
}

//Postclient创建等效的http post rpcclient.client。
func (c *RPCClient) POSTClient() (*rpcclient.Client, error) {
	configCopy := *c.connConfig
	configCopy.HTTPPostMode = true
	return rpcclient.New(&configCopy, nil)
}
