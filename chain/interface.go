
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package chain

import (
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/wtxmgr"
)

//back ends返回可用后端的列表。
//TODO:将每个重构为一个驱动程序并使用动态注册。
func BackEnds() []string {
	return []string{
		"bitcoind",
		"btcd",
		"neutrino",
	}
}

//接口允许多个支持区块链源，例如
//btcd rpc chain server或SPV库，只要我们为
//它。
type Interface interface {
	Start() error
	Stop()
	WaitForShutdown()
	GetBestBlock() (*chainhash.Hash, int32, error)
	GetBlock(*chainhash.Hash) (*wire.MsgBlock, error)
	GetBlockHash(int64) (*chainhash.Hash, error)
	GetBlockHeader(*chainhash.Hash) (*wire.BlockHeader, error)
	FilterBlocks(*FilterBlocksRequest) (*FilterBlocksResponse, error)
	BlockStamp() (*waddrmgr.BlockStamp, error)
	SendRawTransaction(*wire.MsgTx, bool) (*chainhash.Hash, error)
	Rescan(*chainhash.Hash, []btcutil.Address, map[wire.OutPoint]btcutil.Address) error
	NotifyReceived([]btcutil.Address) error
	NotifyBlocks() error
	Notifications() <-chan interface{}
	BackEnd() string
}

//
//为避免直接在
//rpcclient回调，这不是很正常，不允许
//阻止客户端调用。
type (
//ClientConnected是当客户端连接
//打开或重新建立到链服务器。
	ClientConnected struct{}

//BlockConnected是一个新附加到
//最佳链。
	BlockConnected wtxmgr.BlockMeta

//filteredblockconnected是一个包含
//一个结构中同时包含块和相关事务信息，其中
//允许原子更新。
	FilteredBlockConnected struct {
		Block       *wtxmgr.BlockMeta
		RelevantTxs []*wtxmgr.TxRecord
	}

//filterBlocksRequest指定块的范围和
//相关的内部和外部地址，通过相应的索引
//子地址的作用域索引。一组被监视的全球输出点
//还包括监控支出。
	FilterBlocksRequest struct {
		Blocks           []wtxmgr.BlockMeta
		ExternalAddrs    map[waddrmgr.ScopedIndex]btcutil.Address
		InternalAddrs    map[waddrmgr.ScopedIndex]btcutil.Address
		WatchedOutPoints map[wire.OutPoint]btcutil.Address
	}

//筛选块响应报告所有内部和外部
//响应filterblockrequest找到的地址，任何输出点
//发现与这些地址以及相关的
//可以修改钱包余额的交易。的索引
//将返回filterBlocksRequest中的块，以便
//调用方可以在
//更新感兴趣的地址。
	FilterBlocksResponse struct {
		BatchIndex         uint32
		BlockMeta          wtxmgr.BlockMeta
		FoundExternalAddrs map[waddrmgr.KeyScope]map[uint32]struct{}
		FoundInternalAddrs map[waddrmgr.KeyScope]map[uint32]struct{}
		FoundOutPoints     map[wire.OutPoint]btcutil.Address
		RelevantTxns       []*wire.MsgTx
	}

//blockdisconnected是块所描述的
//blockstamp是从最佳链中重新组织出来的。
	BlockDisconnected wtxmgr.BlockMeta

//relevanttx是一个用于支付钱包的交易的通知。
//输入或支付到被监视的地址。
	RelevantTx struct {
		TxRecord *wtxmgr.TxRecord
Block    *wtxmgr.BlockMeta //未开采的
	}

//RescanProgress是描述当前状态的通知
//正在重新扫描。
	RescanProgress struct {
		Hash   *chainhash.Hash
		Height int32
		Time   time.Time
	}

//RescanFinished是以前的重新扫描请求的通知
//已经完成了。
	RescanFinished struct {
		Hash   *chainhash.Hash
		Height int32
		Time   time.Time
	}
)
