
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package chain

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightninglabs/gozmq"
)

//bitcoindconn表示与bitcoind节点的持久客户端连接
//它监听从ZMQ连接读取的事件。
type BitcoindConn struct {
started int32 //原子化使用。
stopped int32 //原子化使用。

//ResanclientCounter是一个原子计数器，它将唯一ID分配给
//每个新的比特币使用当前的比特币重新扫描客户端
//连接。
	rescanClientCounter uint64

//chainParams标识比特节点所在的当前网络
//继续运行。
	chainParams *chaincfg.Params

//客户端是指向比特节点的RPC客户端。
	client *rpcclient.Client

//zmqblockhost是侦听zmq连接的主机，
//负责交付原始事务事件。
	zmqBlockHost string

//zmqtxhost是侦听zmq连接的主机，
//负责交付原始事务事件。
	zmqTxHost string

//zmqPollInterval是我们尝试检索
//ZMQ连接的事件。
	zmqPollInterval time.Duration

//ResancClients是一组活跃的比特币重新扫描客户机，其
//ZMQ事件通知将发送到。
	rescanClientsMtx sync.Mutex
	rescanClients    map[uint64]*BitcoindClient

	quit chan struct{}
	wg   sync.WaitGroup
}

//newbitcoindconn创建到主机描述的节点的客户端连接
//字符串。未立即建立连接，但必须使用
//开始方法。如果远程节点不在同一比特币上操作
//按所传递的链参数描述的网络，连接将
//断开的。
func NewBitcoindConn(chainParams *chaincfg.Params,
	host, user, pass, zmqBlockHost, zmqTxHost string,
	zmqPollInterval time.Duration) (*BitcoindConn, error) {

	clientCfg := &rpcclient.ConnConfig{
		Host:                 host,
		User:                 user,
		Pass:                 pass,
		DisableAutoReconnect: false,
		DisableConnectOnNew:  true,
		DisableTLS:           true,
		HTTPPostMode:         true,
	}

	client, err := rpcclient.New(clientCfg, nil)
	if err != nil {
		return nil, err
	}

	conn := &BitcoindConn{
		chainParams:     chainParams,
		client:          client,
		zmqBlockHost:    zmqBlockHost,
		zmqTxHost:       zmqTxHost,
		zmqPollInterval: zmqPollInterval,
		rescanClients:   make(map[uint64]*BitcoindClient),
		quit:            make(chan struct{}),
	}

	return conn, nil
}

//开始尝试建立到比特节点的RPC和ZMQ连接。如果
//成功后，将生成一个goroutine来从zmq连接读取事件。
//由于连接数量有限，此功能可能会失败
//尝试。这样做是为了防止永远等待连接
//在节点关闭的情况下建立。
func (c *BitcoindConn) Start() error {
	if !atomic.CompareAndSwapInt32(&c.started, 0, 1) {
		return nil
	}

//验证节点是否在预期网络上运行。
	net, err := c.getCurrentNet()
	if err != nil {
		c.client.Disconnect()
		return err
	}
	if net != c.chainParams.Net {
		c.client.Disconnect()
		return fmt.Errorf("expected network %v, got %v",
			c.chainParams.Net, net)
	}

//与比特币建立两个不同的ZMQ连接以检索区块
//和事务事件通知。我们用两个来分隔
//确保一种类型的事件不会从连接中删除的问题
//由于另一种类型的事件而排队。
	zmqBlockConn, err := gozmq.Subscribe(
		c.zmqBlockHost, []string{"rawblock"}, c.zmqPollInterval,
	)
	if err != nil {
		c.client.Disconnect()
		return fmt.Errorf("unable to subscribe for zmq block events: "+
			"%v", err)
	}

	zmqTxConn, err := gozmq.Subscribe(
		c.zmqTxHost, []string{"rawtx"}, c.zmqPollInterval,
	)
	if err != nil {
		c.client.Disconnect()
		return fmt.Errorf("unable to subscribe for zmq tx events: %v",
			err)
	}

	c.wg.Add(2)
	go c.blockEventHandler(zmqBlockConn)
	go c.txEventHandler(zmqTxConn)

	return nil
}

//stop终止与比特节点的rpc和zmq连接，并删除任何
//活动的重新扫描客户端。
func (c *BitcoindConn) Stop() {
	if !atomic.CompareAndSwapInt32(&c.stopped, 0, 1) {
		return
	}

	for _, client := range c.rescanClients {
		client.Stop()
	}

	close(c.quit)
	c.client.Shutdown()

	c.client.WaitForShutdown()
	c.wg.Wait()
}

//BlockEventHandler从ZMQ块套接字读取原始块事件，并
//将它们转发到当前的重新扫描客户端。
//
//注意：这必须作为goroutine运行。
func (c *BitcoindConn) blockEventHandler(conn *gozmq.Conn) {
	defer c.wg.Done()
	defer conn.Close()

	log.Info("Started listening for bitcoind block notifications via ZMQ "+
		"on", c.zmqBlockHost)

	for {
//在试图读取ZMQ套接字之前，我们将
//一定要检查我们是否被要求关闭。
		select {
		case <-c.quit:
			return
		default:
		}

//从ZMQ套接字中轮询一个事件。
		msgBytes, err := conn.Receive()
		if err != nil {
//有可能连接到插座
//连续超时，因此我们将阻止记录此
//防止垃圾发送日志时出错。
			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() {
				continue
			}

			log.Errorf("Unable to receive ZMQ rawblock message: %v",
				err)
			continue
		}

//我们有一个活动！我们现在将确保这是一个阻止事件，
//反序列化它，并报告给不同的重新扫描
//客户。
		eventType := string(msgBytes[0])
		switch eventType {
		case "rawblock":
			block := &wire.MsgBlock{}
			r := bytes.NewReader(msgBytes[1])
			if err := block.Deserialize(r); err != nil {
				log.Errorf("Unable to deserialize block: %v",
					err)
				continue
			}

			c.rescanClientsMtx.Lock()
			for _, client := range c.rescanClients {
				select {
				case client.zmqBlockNtfns <- block:
				case <-client.quit:
				case <-c.quit:
					c.rescanClientsMtx.Unlock()
					return
				}
			}
			c.rescanClientsMtx.Unlock()
		default:
//如果
//比特币关闭，将导致无法读取
//事件类型。为了防止日志记录，我们将
//当然，它符合ASCII标准。
			if eventType == "" || !isASCII(eventType) {
				continue
			}

			log.Warnf("Received unexpected event type from "+
				"rawblock subscription: %v", eventType)
		}
	}
}

//txEventHandler从zmq块套接字读取原始块事件并转发
//以及当前的重新扫描客户端。
//
//注意：这必须作为goroutine运行。
func (c *BitcoindConn) txEventHandler(conn *gozmq.Conn) {
	defer c.wg.Done()
	defer conn.Close()

	log.Info("Started listening for bitcoind transaction notifications "+
		"via ZMQ on", c.zmqTxHost)

	for {
//在试图读取ZMQ套接字之前，我们将
//一定要检查我们是否被要求关闭。
		select {
		case <-c.quit:
			return
		default:
		}

//从ZMQ套接字中轮询一个事件。
		msgBytes, err := conn.Receive()
		if err != nil {
//有可能连接到插座
//连续超时，因此我们将阻止记录此
//防止垃圾发送日志时出错。
			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() {
				continue
			}

			log.Errorf("Unable to receive ZMQ rawtx message: %v",
				err)
			continue
		}

//我们有一个活动！我们现在将确保它是一个事务事件，
//反序列化它，并报告给不同的重新扫描
//客户。
		eventType := string(msgBytes[0])
		switch eventType {
		case "rawtx":
			tx := &wire.MsgTx{}
			r := bytes.NewReader(msgBytes[1])
			if err := tx.Deserialize(r); err != nil {
				log.Errorf("Unable to deserialize "+
					"transaction: %v", err)
				continue
			}

			c.rescanClientsMtx.Lock()
			for _, client := range c.rescanClients {
				select {
				case client.zmqTxNtfns <- tx:
				case <-client.quit:
				case <-c.quit:
					c.rescanClientsMtx.Unlock()
					return
				}
			}
			c.rescanClientsMtx.Unlock()
		default:
//如果
//比特币关闭，将导致无法读取
//事件类型。为了防止日志记录，我们将
//当然，它符合ASCII标准。
			if eventType == "" || !isASCII(eventType) {
				continue
			}

			log.Warnf("Received unexpected event type from rawtx "+
				"subscription: %v", eventType)
		}
	}
}

//getcurrentnet返回比特节点运行的网络。
func (c *BitcoindConn) getCurrentNet() (wire.BitcoinNet, error) {
	hash, err := c.client.GetBlockHash(0)
	if err != nil {
		return 0, err
	}

	switch *hash {
	case *chaincfg.TestNet3Params.GenesisHash:
		return chaincfg.TestNet3Params.Net, nil
	case *chaincfg.RegressionNetParams.GenesisHash:
		return chaincfg.RegressionNetParams.Net, nil
	case *chaincfg.MainNetParams.GenesisHash:
		return chaincfg.MainNetParams.Net, nil
	default:
		return 0, fmt.Errorf("unknown network with genesis hash %v", hash)
	}
}

//newbitcoindclient返回使用当前比特币的比特币客户机
//连接。这允许我们使用多个
//客户。
func (c *BitcoindConn) NewBitcoindClient() *BitcoindClient {
	return &BitcoindClient{
		quit: make(chan struct{}),

		id: atomic.AddUint64(&c.rescanClientCounter, 1),

		chainParams: c.chainParams,
		chainConn:   c,

		rescanUpdate:     make(chan interface{}),
		watchedAddresses: make(map[string]struct{}),
		watchedOutPoints: make(map[wire.OutPoint]struct{}),
		watchedTxs:       make(map[chainhash.Hash]struct{}),

		notificationQueue: NewConcurrentQueue(20),
		zmqTxNtfns:        make(chan *wire.MsgTx),
		zmqBlockNtfns:     make(chan *wire.MsgBlock),

		mempool:        make(map[chainhash.Hash]struct{}),
		expiredMempool: make(map[int32]map[chainhash.Hash]struct{}),
	}
}

//addclient将客户端添加到当前的活动重新扫描客户端集
//链条连接。这允许连接包括指定的客户端
//在它的通知传递中。
//
//注意：此函数对于并发访问是安全的。
func (c *BitcoindConn) AddClient(client *BitcoindClient) {
	c.rescanClientsMtx.Lock()
	defer c.rescanClientsMtx.Unlock()

	c.rescanClients[client.id] = client
}

//removeclient从活动集中删除具有给定ID的客户端
//重新扫描客户端。一旦删除，客户端将不再接收块和
//来自链连接的事务通知。
//
//注意：此函数对于并发访问是安全的。
func (c *BitcoindConn) RemoveClient(id uint64) {
	c.rescanClientsMtx.Lock()
	defer c.rescanClientsMtx.Unlock()

	delete(c.rescanClients, id)
}

//isascii是一个助手方法，用于检查“data”中的所有字节是否
//如果解释为字符串，则为可打印的ASCII字符。
func isASCII(s string) bool {
	for _, c := range s {
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}
