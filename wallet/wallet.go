
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wallet

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/btcsuite/btcwallet/chain"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
	"github.com/btcsuite/btcwallet/wallet/txrules"
	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/btcsuite/btcwallet/walletdb/migration"
	"github.com/btcsuite/btcwallet/wtxmgr"
	"github.com/davecgh/go-spew/spew"
)

const (
//
//
//
//
//
//
//注：在编写时，公共加密仅适用于公共
//waddrmgr命名空间中的数据。事务尚未加密。
	InsecurePubPassphrase = "public"

	walletDbWatchingOnlyName = "wowallet.db"

//recoverybatchsize是将
//由恢复管理器依次扫描，如果
//
	recoveryBatchSize = 2000
)

//
//
//
var ErrNotSynced = errors.New("wallet is not synchronized with the chain server")

//
var (
	waddrmgrNamespaceKey = []byte("waddrmgr")
	wtxmgrNamespaceKey   = []byte("wtxmgr")
)

//
//
//地址和钥匙）
type Wallet struct {
	publicPassphrase []byte

//数据存储
	db      walletdb.DB
	Manager *waddrmgr.Manager
	TxStore *wtxmgr.Store

	chainClient        chain.Interface
	chainClientLock    sync.Mutex
	chainClientSynced  bool
	chainClientSyncMtx sync.Mutex

	lockedOutpoints map[wire.OutPoint]struct{}

	recoveryWindow uint32

//用于重新扫描处理的通道。请求被添加并与合并
//任何等待请求，在发送到另一个goroutine之前
//调用重新扫描RPC。
	rescanAddJob        chan *RescanJob
	rescanBatch         chan *rescanBatch
rescanNotifications chan interface{} //来自链服务器
	rescanProgress      chan *RescanProgressMsg
	rescanFinished      chan *RescanFinishedMsg

//
	createTxRequests chan createTxRequest

//
	unlockRequests     chan unlockRequest
	lockRequests       chan struct{}
	holdUnlockRequests chan chan heldUnlock
	lockState          chan bool
	changePassphrase   chan changePassphraseRequest
	changePassphrases  chan changePassphrasesRequest

//
	reorganizingLock sync.Mutex
	reorganizeToHash chainhash.Hash
	reorganizing     bool

	NtfnServer *NotificationServer

	chainParams *chaincfg.Params
	wg          sync.WaitGroup

	started bool
	quit    chan struct{}
	quitMu  sync.Mutex
}

//
func (w *Wallet) Start() {
	w.quitMu.Lock()
	select {
	case <-w.quit:
//
		w.WaitForShutdown()
		w.quit = make(chan struct{})
	default:
//
		if w.started {
			w.quitMu.Unlock()
			return
		}
		w.started = true
	}
	w.quitMu.Unlock()

	w.wg.Add(2)
	go w.txCreator()
	go w.walletLocker()
}

//
//
//
//
//此方法不稳定，将在移动所有同步逻辑时删除。
//在钱包外。
func (w *Wallet) SynchronizeRPC(chainClient chain.Interface) {
	w.quitMu.Lock()
	select {
	case <-w.quit:
		w.quitMu.Unlock()
		return
	default:
	}
	w.quitMu.Unlock()

//TODO:在已设置新客户端的情况下忽略新客户端breaks调用方
//可能是在断开连接后，由谁来替换客户机。
	w.chainClientLock.Lock()
	if w.chainClient != nil {
		w.chainClientLock.Unlock()
		return
	}
	w.chainClient = chainClient

//
//
	switch cc := chainClient.(type) {
	case *chain.NeutrinoClient:
		cc.SetStartTime(w.Manager.Birthday())
	case *chain.BitcoindClient:
		cc.SetBirthday(w.Manager.Birthday())
	}
	w.chainClientLock.Unlock()

//
//
//
//
	w.wg.Add(4)
	go w.handleChainNotifications()
	go w.rescanBatchHandler()
	go w.rescanProgressHandler()
	go w.rescanRPCHandler()
}

//
//
//
//钱包。
func (w *Wallet) requireChainClient() (chain.Interface, error) {
	w.chainClientLock.Lock()
	chainClient := w.chainClient
	w.chainClientLock.Unlock()
	if chainClient == nil {
		return nil, errors.New("blockchain RPC is inactive")
	}
	return chainClient, nil
}

//chainclient返回与
//钱包。
//
//
//钱包。
func (w *Wallet) ChainClient() chain.Interface {
	w.chainClientLock.Lock()
	chainClient := w.chainClient
	w.chainClientLock.Unlock()
	return chainClient
}

//
func (w *Wallet) quitChan() <-chan struct{} {
	w.quitMu.Lock()
	c := w.quit
	w.quitMu.Unlock()
	return c
}

//
func (w *Wallet) Stop() {
	w.quitMu.Lock()
	quit := w.quit
	w.quitMu.Unlock()

	select {
	case <-quit:
	default:
		close(quit)
		w.chainClientLock.Lock()
		if w.chainClient != nil {
			w.chainClient.Stop()
			w.chainClient = nil
		}
		w.chainClientLock.Unlock()
	}
}

//
//
func (w *Wallet) ShuttingDown() bool {
	select {
	case <-w.quitChan():
		return true
	default:
		return false
	}
}

//
func (w *Wallet) WaitForShutdown() {
	w.chainClientLock.Lock()
	if w.chainClient != nil {
		w.chainClient.WaitForShutdown()
	}
	w.chainClientLock.Unlock()
	w.wg.Wait()
}

//
//with the Bitcoin network.
func (w *Wallet) SynchronizingToNetwork() bool {
//目前，RPC是唯一的同步方法。在
//将来，当添加SPV时，还需要单独检查，或
//
//创建钱包。
	w.chainClientSyncMtx.Lock()
	syncing := w.chainClient != nil
	w.chainClientSyncMtx.Unlock()
	return syncing
}

//chainsynced返回钱包是否已连接到链服务器
//并同步到主链上的最佳块。
func (w *Wallet) ChainSynced() bool {
	w.chainClientSyncMtx.Lock()
	synced := w.chainClientSynced
	w.chainClientSyncMtx.Unlock()
	return synced
}

//setchainsynced标记钱包是否连接到当前同步
//链服务器通知最新的块。
//
//注意：由于rpcclient的API限制，在
//客户端已断开连接（正在尝试重新连接）。这将是未知的
//在收到重新连接通知之前，此时钱包可以
//再次标记为不同步，直到下次重新扫描完成。
func (w *Wallet) SetChainSynced(synced bool) {
	w.chainClientSyncMtx.Lock()
	w.chainClientSynced = synced
	w.chainClientSyncMtx.Unlock()
}

//activedata返回当前活动的接收地址和所有未使用的
//输出。这主要是为了提供
//重新请求。
func (w *Wallet) activeData(dbtx walletdb.ReadTx) ([]btcutil.Address, []wtxmgr.Credit, error) {
	addrmgrNs := dbtx.ReadBucket(waddrmgrNamespaceKey)
	txmgrNs := dbtx.ReadBucket(wtxmgrNamespaceKey)

	var addrs []btcutil.Address
	err := w.Manager.ForEachActiveAddress(addrmgrNs, func(addr btcutil.Address) error {
		addrs = append(addrs, addr)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	unspent, err := w.TxStore.UnspentOutputs(txmgrNs)
	return addrs, unspent, err
}

//SyncWithChain使钱包与当前的Chain服务器保持同步
//连接。它创建一个重新扫描请求并阻止，直到重新扫描
//完成了。如果设置了生日块，可以传入以确保
//正确检测是否回滚。
func (w *Wallet) syncWithChain(birthdayStamp *waddrmgr.BlockStamp) error {
	chainClient, err := w.requireChainClient()
	if err != nil {
		return err
	}

//向所有钱包发送交易请求通知
//地址。
	var (
		addrs   []btcutil.Address
		unspent []wtxmgr.Credit
	)
	err = walletdb.View(w.db, func(dbtx walletdb.ReadTx) error {
		var err error
		addrs, unspent, err = w.activeData(dbtx)
		return err
	})
	if err != nil {
		return err
	}

	startHeight := w.Manager.SyncedTo().Height

//如果没有未使用的内容，我们会将此标记为第一次同步
//outputs as known by the wallet. This'll allow us to skip a full
//
	isInitialSync := len(unspent) == 0

	isRecovery := w.recoveryWindow > 0
	birthday := w.Manager.Birthday()

//TODO（JRICK）：该如何处理早于
//链服务器最好的块？

//当没有为钱包生成地址时，重新扫描可以
//跳过。
//
//TODO:这是正确的，因为上面的activedata返回所有
//曾经创建过的地址，包括那些不需要监视的地址
//不再。如果此假设为“否”，则应更新此代码。
//longer true, but worst case would result in an unnecessary rescan.
	if isInitialSync || isRecovery {
//找到最新检查站的高度。这让我们赶上
//至少那个检查点，因为我们正在从
//抓取，让我们避免一堆昂贵的数据库事务
//当我们使用bdb作为walletdb后端时，
//链的中微子。接口后端和链
//后端开始与钱包同步。
		_, bestHeight, err := chainClient.GetBestBlock()
		if err != nil {
			return err
		}

		checkHeight := bestHeight
		if len(w.chainParams.Checkpoints) > 0 {
			checkHeight = w.chainParams.Checkpoints[len(
				w.chainParams.Checkpoints)-1].Height
		}

		logHeight := checkHeight
		if bestHeight > logHeight {
			logHeight = bestHeight
		}

		log.Infof("Catching up block hashes to height %d, this will "+
			"take a while...", logHeight)

//初始化第一个数据库事务。
		tx, err := w.db.BeginReadWriteTx()
		if err != nil {
			return err
		}
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)

//只有在我们实际处于恢复状态时才分配recoveryMgr
//模式。
		var recoveryMgr *RecoveryManager
		if isRecovery {
			log.Infof("RECOVERY MODE ENABLED -- rescanning for "+
				"used addresses with recovery_window=%d",
				w.recoveryWindow)

//Initialize the recovery manager with a default batch
//大小为2000。
			recoveryMgr = NewRecoveryManager(
				w.recoveryWindow, recoveryBatchSize,
				w.chainParams,
			)

//如果恢复，我们
//
//数据库。对于基本恢复，我们只会在
//默认作用域。
			scopedMgrs, err := w.defaultScopeManagers()
			if err != nil {
				return err
			}

			txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)
			credits, err := w.TxStore.UnspentOutputs(txmgrNs)
			if err != nil {
				return err
			}

			err = recoveryMgr.Resurrect(ns, scopedMgrs, credits)
			if err != nil {
				return err
			}
		}

		for height := startHeight; height <= bestHeight; height++ {
			hash, err := chainClient.GetBlockHash(int64(height))
			if err != nil {
				tx.Rollback()
				return err
			}

//如果我们使用中微子后端，我们可以检查
//它是不是现在的。对于其他后端，我们假设
//如果最佳高度达到
//最后一个检查点。
			isCurrent := func(bestHeight int32) bool {
				switch c := chainClient.(type) {
				case *chain.NeutrinoClient:
					return c.CS.IsCurrent()
				}
				return bestHeight >= checkHeight
			}

//如果我们找到了后端知道的最佳高度
//
//等待。我们可以给它一点时间
//
//基于后端。一旦我们看到后端
//已经进步了，我们可以赶上。
			for height == bestHeight && !isCurrent(bestHeight) {
				time.Sleep(100 * time.Millisecond)
				_, bestHeight, err = chainClient.GetBestBlock()
				if err != nil {
					tx.Rollback()
					return err
				}
			}

			header, err := chainClient.GetBlockHeader(hash)
			if err != nil {
				return err
			}

//检查该头的时间戳是否已超过
//我们的生日，或者如果我们以前超过过一个。
			timestamp := header.Timestamp
			if timestamp.After(birthday) || birthdayStamp != nil {
//如果这是我们生日的第一个街区，
//记录块戳以便使用
//这是重新扫描的起点。
//这将确保我们不会错过交易
//在初始阶段发送到钱包的
//同步。
//
//注意：钱包里的生日是
//在钱包生日的前两天，
//
//时间戳。
				if birthdayStamp == nil {
					birthdayStamp = &waddrmgr.BlockStamp{
						Height:    height,
						Hash:      *hash,
						Timestamp: timestamp,
					}

					log.Debugf("Found birthday block: "+
						"height=%d, hash=%v",
						birthdayStamp.Height,
						birthdayStamp.Hash)

					err := w.Manager.SetBirthdayBlock(
						ns, *birthdayStamp, true,
					)
					if err != nil {
						tx.Rollback()
						return err
					}
				}

//如果我们在恢复模式和检查
//通过，我们将把这个块添加到
//扫描恢复地址的块。
				if isRecovery {
					recoveryMgr.AddToBlockBatch(
						hash, height, timestamp,
					)
				}
			}

			err = w.Manager.SetSyncedTo(ns, &waddrmgr.BlockStamp{
				Hash:      *hash,
				Height:    height,
				Timestamp: timestamp,
			})
			if err != nil {
				tx.Rollback()
				return err
			}

//如果我们处于恢复模式，尝试恢复
//已添加到恢复管理器的块
//到目前为止，阻止批处理。如果块批次为空，则此
//将是一个NOP。
			if isRecovery && height%recoveryBatchSize == 0 {
				err := w.recoverDefaultScopes(
					chainClient, tx, ns,
					recoveryMgr.BlockBatch(),
					recoveryMgr.State(),
				)
				if err != nil {
					tx.Rollback()
					return err
				}

//清除所有已处理块的批次。
				recoveryMgr.ResetBlockBatch()
			}

//每隔10公里，提交并启动一个新的数据库tx。
			if height%10000 == 0 {
				err = tx.Commit()
				if err != nil {
					tx.Rollback()
					return err
				}

				log.Infof("Caught up to height %d", height)

				tx, err = w.db.BeginReadWriteTx()
				if err != nil {
					return err
				}

				ns = tx.ReadWriteBucket(waddrmgrNamespaceKey)
			}
		}

//对所有块执行最后一次恢复尝试
//未按默认粒度2000块进行批处理。
		if isRecovery {
			err := w.recoverDefaultScopes(
				chainClient, tx, ns, recoveryMgr.BlockBatch(),
				recoveryMgr.State(),
			)
			if err != nil {
				tx.Rollback()
				return err
			}
		}

//提交（或回滚）最终的数据库事务。
		err = tx.Commit()
		if err != nil {
			tx.Rollback()
			return err
		}
		log.Info("Done catching up block hashes")

//因为我们花了一些时间来处理块哈希，所以我们
//可能有新地址等待我们请求
//在初始同步期间。确保我们之前有这些
//稍后请求重新扫描。
		err = walletdb.View(w.db, func(dbtx walletdb.ReadTx) error {
			var err error
			addrs, unspent, err = w.activeData(dbtx)
			return err
		})
		if err != nil {
			return err
		}
	}

//将以前看到的块与链服务器进行比较。如果有的话
//
//在重新扫描之前。
	rollback := false
	rollbackStamp := w.Manager.SyncedTo()
	err = walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		txmgrNs := tx.ReadWriteBucket(wtxmgrNamespaceKey)
		for height := rollbackStamp.Height; true; height-- {
			hash, err := w.Manager.BlockHash(addrmgrNs, height)
			if err != nil {
				return err
			}
			chainHash, err := chainClient.GetBlockHash(int64(height))
			if err != nil {
				return err
			}
			header, err := chainClient.GetBlockHeader(chainHash)
			if err != nil {
				return err
			}

			rollbackStamp.Hash = *chainHash
			rollbackStamp.Height = height
			rollbackStamp.Timestamp = header.Timestamp

			if bytes.Equal(hash[:], chainHash[:]) {
				break
			}
			rollback = true
		}

		if rollback {
			err := w.Manager.SetSyncedTo(addrmgrNs, &rollbackStamp)
			if err != nil {
				return err
			}
//在和之后回滚未确认的事务
//传递的高度，因此在新的“同步到高度”中添加一个
//以防止同步到块的未确认txs。
			err = w.TxStore.Rollback(txmgrNs, rollbackStamp.Height+1)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

//如果在初始同步和
//回滚会使我们还原它，更新生日戳以便
//指向新提示。
	birthdayRollback := false
	if birthdayStamp != nil && rollbackStamp.Height <= birthdayStamp.Height {
		birthdayStamp = &rollbackStamp
		birthdayRollback = true

		log.Debugf("Found new birthday block after rollback: "+
			"height=%d, hash=%v", birthdayStamp.Height,
			birthdayStamp.Hash)

		err := walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) error {
			ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
			return w.Manager.SetBirthdayBlock(
				ns, *birthdayStamp, true,
			)
		})
		if err != nil {
			return nil
		}
	}

//请求已连接和已断开连接的块的通知。
//
//TODO（JRICK）：要么只请求一次此通知，要么在
//rpcclient被修改为允许某些通知请求
//重新连接时自动重新发送，包括notifyblocks请求
//
//通知重新注册，在这种情况下，这里的代码应该是
//照原样走。
	if err := chainClient.NotifyBlocks(); err != nil {
		return err
	}

//如果这是我们的初始同步，我们将从种子中恢复，或者
//由于一次连锁重组，生日推迟了，我们将发送一个
//从生日栏重新扫描以确保检测到所有相关的
//从这一点开始的连锁事件。
	if isInitialSync || isRecovery || birthdayRollback {
		return w.rescanWithTarget(addrs, unspent, birthdayStamp)
	}

//否则，我们将从提示重新扫描。
	return w.rescanWithTarget(addrs, unspent, nil)
}

//DefaultScopeManager使用
//默认的键范围集。
func (w *Wallet) defaultScopeManagers() (
	map[waddrmgr.KeyScope]*waddrmgr.ScopedKeyManager, error) {

	scopedMgrs := make(map[waddrmgr.KeyScope]*waddrmgr.ScopedKeyManager)
	for _, scope := range waddrmgr.DefaultKeyScopes {
		scopedMgr, err := w.Manager.FetchScopedKeyManager(scope)
		if err != nil {
			return nil, err
		}

		scopedMgrs[scope] = scopedMgr
	}

	return scopedMgrs, nil
}

//recoverDefaultScopes尝试恢复属于
//钱包里知道的活跃范围的钥匙经理。恢复每个作用域
//默认帐户将对同一批块进行迭代。
//TODO（连接器）：并行化/管道/缓存中间网络请求
func (w *Wallet) recoverDefaultScopes(
	chainClient chain.Interface,
	tx walletdb.ReadWriteTx,
	ns walletdb.ReadWriteBucket,
	batch []wtxmgr.BlockMeta,
	recoveryState *RecoveryState) error {

	scopedMgrs, err := w.defaultScopeManagers()
	if err != nil {
		return err
	}

	return w.recoverScopedAddresses(
		chainClient, tx, ns, batch, recoveryState, scopedMgrs,
	)
}

//recoveraccountaddresses扫描一系列块以尝试恢复
//特定帐户派生路径以前使用的地址。处于高位
//级别，该算法的工作原理如下：
//1）确保内部和外部分支水平完全扩大。
//2）过滤整个范围的块，如果非零个数停止
//地址包含在特定的块中。
//3）记录块中找到的所有内部和外部地址。
//4）记录在该块中发现的任何应注意支出的输出点。
//5）调整块的范围，直到并包括报告加法器的块。
//6）如果范围内还有更多块，则从（1）重复。
func (w *Wallet) recoverScopedAddresses(
	chainClient chain.Interface,
	tx walletdb.ReadWriteTx,
	ns walletdb.ReadWriteBucket,
	batch []wtxmgr.BlockMeta,
	recoveryState *RecoveryState,
	scopedMgrs map[waddrmgr.KeyScope]*waddrmgr.ScopedKeyManager) error {

//如果批处理中没有块，我们就完成了。
	if len(batch) == 0 {
		return nil
	}

	log.Infof("Scanning %d blocks for recoverable addresses", len(batch))

expandHorizons:
	for scope, scopedMgr := range scopedMgrs {
		scopeState := recoveryState.StateForScope(scope)
		err := expandScopeHorizons(ns, scopedMgr, scopeState)
		if err != nil {
			return err
		}
	}

//随着内部和外部视野的适当扩大，我们现在
//构造筛选块请求。请求包括范围
//除了作用域index->addr之外，我们还打算扫描的块
//所有内部和外部分支的映射。
	filterReq := newFilterBlocksRequest(batch, scopedMgrs, recoveryState)

//使用我们的链后端启动过滤块请求。如果一个
//出现错误，我们无法继续恢复。
	filterResp, err := chainClient.FilterBlocks(filterReq)
	if err != nil {
		return err
	}

//如果过滤器响应为空，则此信号表示
//批处理已完成，未发现其他地址。作为一个
//结果，不需要进一步修改恢复状态。
//我们可以继续下一批。
	if filterResp == nil {
		return nil
	}

//否则，检索检测到
//地址匹配数非零。
	block := batch[filterResp.BatchIndex]

//记录地址或输出点的任何重要发现。
	logFilterBlocksResp(block, filterResp)

//报告由于
//适当的分支恢复状态。在上面添加索引
//最后找到的任何一个索引都将导致层位扩展
//在下一次迭代时。任何找到的地址也标记为已使用
//使用范围键管理器。
	err = extendFoundAddresses(ns, filterResp, scopedMgrs, recoveryState)
	if err != nil {
		return err
	}

//使用找到的任何输出点更新被监视输出点的全局集
//在街区。
	for outPoint, addr := range filterResp.FoundOutPoints {
		recoveryState.AddWatchedOutPoint(&outPoint, addr)
	}

//最后，记录所有返回的相关交易
//在过滤块响应中。这样可以确保这些事务
//并在执行最终重新扫描时跟踪其输出。
	for _, txn := range filterResp.RelevantTxns {
		txRecord, err := wtxmgr.NewTxRecordFromMsgTx(
			txn, filterResp.BlockMeta.Time,
		)
		if err != nil {
			return err
		}

		err = w.addRelevantTx(tx, txRecord, &filterResp.BlockMeta)
		if err != nil {
			return err
		}
	}

//
//返回的那个找到了地址。
	batch = batch[filterResp.BatchIndex+1:]

//如果这不是批处理中的最后一个块，我们将重复
//在扩展了我们的视野之后，再次过滤过程。
	if len(batch) > 0 {
		goto expandHorizons
	}

	return nil
}

//ExpandScopeHorizons确保ScopereCoveryState具有足够的
//对其内部和外部分支进行规模展望。钥匙
//这里派生的将添加到作用域的恢复状态，但不影响
//钱包的持久状态。如果检测到任何无效的子密钥，则
//地平线将适当延伸，这样我们的展望总是包括
//有效子密钥的正确数目。
func expandScopeHorizons(ns walletdb.ReadWriteBucket,
	scopedMgr *waddrmgr.ScopedKeyManager,
	scopeState *ScopeRecoveryState) error {

//计算当前外部范围和我们的地址数量
//必须派生以确保我们为
//外部分支。
	exHorizon, exWindow := scopeState.ExternalBranch.ExtendHorizon()
	count, childIndex := uint32(0), exHorizon
	for count < exWindow {
		keyPath := externalKeyPath(childIndex)
		addr, err := scopedMgr.DeriveFromKeyPath(ns, keyPath)
		switch {
		case err == hdkeychain.ErrInvalidChild:
//用
//外部分支的恢复状态。这也
//增加分支机构的范围，以便它能够
//对于此跳过的子索引。
			scopeState.ExternalBranch.MarkInvalidChild(childIndex)
			childIndex++
			continue

		case err != nil:
			return err
		}

//注册新生成的外部地址和子索引
//具有外部分支恢复状态。
		scopeState.ExternalBranch.AddAddr(childIndex, addr.Address())

		childIndex++
		count++
	}

//计算当前的内部范围和地址数
//必须派生以确保我们为
//内部分支机构。
	inHorizon, inWindow := scopeState.InternalBranch.ExtendHorizon()
	count, childIndex = 0, inHorizon
	for count < inWindow {
		keyPath := internalKeyPath(childIndex)
		addr, err := scopedMgr.DeriveFromKeyPath(ns, keyPath)
		switch {
		case err == hdkeychain.ErrInvalidChild:
//用
//内部分支机构的恢复状态。这也
//增加分支机构的范围，以便它能够
//对于此跳过的子索引。
			scopeState.InternalBranch.MarkInvalidChild(childIndex)
			childIndex++
			continue

		case err != nil:
			return err
		}

//注册新生成的内部地址和子索引
//具有内部分支恢复状态。
		scopeState.InternalBranch.AddAddr(childIndex, addr.Address())

		childIndex++
		count++
	}

	return nil
}

//ExternalKeyPath返回相对外部派生路径/0/0/索引。
func externalKeyPath(index uint32) waddrmgr.DerivationPath {
	return waddrmgr.DerivationPath{
		Account: waddrmgr.DefaultAccountNum,
		Branch:  waddrmgr.ExternalBranch,
		Index:   index,
	}
}

//InternalKeyPath返回相对的内部派生路径/0/1/索引。
func internalKeyPath(index uint32) waddrmgr.DerivationPath {
	return waddrmgr.DerivationPath{
		Account: waddrmgr.DefaultAccountNum,
		Branch:  waddrmgr.InternalBranch,
		Index:   index,
	}
}

//newfilterblocksrequest使用当前的
//block range, scoped managers, and recovery state.
func newFilterBlocksRequest(batch []wtxmgr.BlockMeta,
	scopedMgrs map[waddrmgr.KeyScope]*waddrmgr.ScopedKeyManager,
	recoveryState *RecoveryState) *chain.FilterBlocksRequest {

	filterReq := &chain.FilterBlocksRequest{
		Blocks:           batch,
		ExternalAddrs:    make(map[waddrmgr.ScopedIndex]btcutil.Address),
		InternalAddrs:    make(map[waddrmgr.ScopedIndex]btcutil.Address),
		WatchedOutPoints: recoveryState.WatchedOutPoints(),
	}

//通过合并地址填充外部和内部地址
//集合属于所有当前跟踪的作用域。
	for scope := range scopedMgrs {
		scopeState := recoveryState.StateForScope(scope)
		for index, addr := range scopeState.ExternalBranch.Addrs() {
			scopedIndex := waddrmgr.ScopedIndex{
				Scope: scope,
				Index: index,
			}
			filterReq.ExternalAddrs[scopedIndex] = addr
		}
		for index, addr := range scopeState.InternalBranch.Addrs() {
			scopedIndex := waddrmgr.ScopedIndex{
				Scope: scope,
				Index: index,
			}
			filterReq.InternalAddrs[scopedIndex] = addr
		}
	}

	return filterReq
}

//extendFoundAddresses接受包含地址的筛选器块响应
//在链上找到，并将所有相关派生路径的状态提升到
//匹配每个分支找到的最高子索引。
func extendFoundAddresses(ns walletdb.ReadWriteBucket,
	filterResp *chain.FilterBlocksResponse,
	scopedMgrs map[waddrmgr.KeyScope]*waddrmgr.ScopedKeyManager,
	recoveryState *RecoveryState) error {

//将所有恢复的外部地址标记为已使用。只能这样做
//对于在中报告非零外部地址数的作用域
//这个街区。
	for scope, indexes := range filterResp.FoundExternalAddrs {
//首先，报告为此找到的所有外部子索引
//范围。这样可以确保外部最后找到的索引将
//更新以包括迄今为止看到的最大子索引。
		scopeState := recoveryState.StateForScope(scope)
		for index := range indexes {
			scopeState.ExternalBranch.ReportFound(index)
		}

		scopedMgr := scopedMgrs[scope]

//现在，报告所有找到的地址，派生并扩展所有
//最新找到的外部地址（包括最新找到的外部地址）
//此范围的索引。
		exNextUnfound := scopeState.ExternalBranch.NextUnfound()

		exLastFound := exNextUnfound
		if exLastFound > 0 {
			exLastFound--
		}

		err := scopedMgr.ExtendExternalAddresses(
			ns, waddrmgr.DefaultAccountNum, exLastFound,
		)
		if err != nil {
			return err
		}

//最后，扩展作用域的地址后，我们标记
//在块中找到的外部地址和
//
		for index := range indexes {
			addr := scopeState.ExternalBranch.GetAddr(index)
			err := scopedMgr.MarkUsed(ns, addr)
			if err != nil {
				return err
			}
		}
	}

//将所有恢复的内部地址标记为已使用。只能这样做
//对于在中报告非零个内部地址的作用域
//这个街区。
	for scope, indexes := range filterResp.FoundInternalAddrs {
//首先，报告为此找到的所有内部子索引
//范围。这样可以确保内部最后找到的索引
//更新以包括迄今为止看到的最大子索引。
		scopeState := recoveryState.StateForScope(scope)
		for index := range indexes {
			scopeState.InternalBranch.ReportFound(index)
		}

		scopedMgr := scopedMgrs[scope]

//现在，报告所有找到的地址，派生并扩展所有
//内部地址最多包括当前最后找到的
//此范围的索引。
		inNextUnfound := scopeState.InternalBranch.NextUnfound()

		inLastFound := inNextUnfound
		if inLastFound > 0 {
			inLastFound--
		}
		err := scopedMgr.ExtendInternalAddresses(
			ns, waddrmgr.DefaultAccountNum, inLastFound,
		)
		if err != nil {
			return err
		}

//最后，扩展作用域的地址后，我们标记
//在块中找到的内部地址属于
//达到这个范围。
		for index := range indexes {
			addr := scopeState.InternalBranch.GetAddr(index)
			err := scopedMgr.MarkUsed(ns, addr)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

//logfilterblocksresp在筛选时提供有用的日志信息
//已成功查找相关交易。
func logFilterBlocksResp(block wtxmgr.BlockMeta,
	resp *chain.FilterBlocksResponse) {

//记录在此块中找到的外部地址数。
	var nFoundExternal int
	for _, indexes := range resp.FoundExternalAddrs {
		nFoundExternal += len(indexes)
	}
	if nFoundExternal > 0 {
		log.Infof("Recovered %d external addrs at height=%d hash=%v",
			nFoundExternal, block.Height, block.Hash)
	}

//记录在此块中找到的内部地址数。
	var nFoundInternal int
	for _, indexes := range resp.FoundInternalAddrs {
		nFoundInternal += len(indexes)
	}
	if nFoundInternal > 0 {
		log.Infof("Recovered %d internal addrs at height=%d hash=%v",
			nFoundInternal, block.Height, block.Hash)
	}

//记录在此块中找到的输出点数。
	nFoundOutPoints := len(resp.FoundOutPoints)
	if nFoundOutPoints > 0 {
		log.Infof("Found %d spends from watched outpoints at "+
			"height=%d hash=%v",
			nFoundOutPoints, block.Height, block.Hash)
	}
}

type (
	createTxRequest struct {
		account     uint32
		outputs     []*wire.TxOut
		minconf     int32
		feeSatPerKB btcutil.Amount
		resp        chan createTxResponse
	}
	createTxResponse struct {
		tx  *txauthor.AuthoredTx
		err error
	}
)

//txcreator负责输入选择和创建
//交易。这些功能是这个方法的责任
//（设计为作为自己的goroutine运行）因为输入选择必须
//序列化，否则可以通过选择
//多个事务的相同输入。与输入选择一起，
//方法还负责签署交易，因为我们没有
//
//正在创建交易记录。在这种情况下，就有可能
//两个请求，而不是一个请求，由于可用性不足而失败
//输入。
func (w *Wallet) txCreator() {
	quit := w.quitChan()
out:
	for {
		select {
		case txr := <-w.createTxRequests:
			heldUnlock, err := w.holdUnlock()
			if err != nil {
				txr.resp <- createTxResponse{nil, err}
				continue
			}
			tx, err := w.txToOutputs(txr.outputs, txr.account,
				txr.minconf, txr.feeSatPerKB)
			heldUnlock.release()
			txr.resp <- createTxResponse{tx, err}
		case <-quit:
			break out
		}
	}
	w.wg.Done()
}

//CreateSimpleTX创建一个新的已签名事务，该事务花费未使用的p2pkh
//输出与at laest minconf确认支出到任意数量
//地址/金额对。变更和适当的交易费用
//必要时自动包含。通过此创建所有事务
//函数被序列化以防止创建
//花费相同的产出。
func (w *Wallet) CreateSimpleTx(account uint32, outputs []*wire.TxOut,
	minconf int32, satPerKb btcutil.Amount) (*txauthor.AuthoredTx, error) {

	req := createTxRequest{
		account:     account,
		outputs:     outputs,
		minconf:     minconf,
		feeSatPerKB: satPerKb,
		resp:        make(chan createTxResponse),
	}
	w.createTxRequests <- req
	resp := <-req.resp
	return resp.tx, resp.err
}

type (
	unlockRequest struct {
		passphrase []byte
lockAfter  <-chan time.Time //nil防止超时。
		err        chan error
	}

	changePassphraseRequest struct {
		old, new []byte
		private  bool
		err      chan error
	}

	changePassphrasesRequest struct {
		publicOld, publicNew   []byte
		privateOld, privateNew []byte
		err                    chan error
	}

//Heldunlock是一种防止钱包自动
//
//未锁定的钱包已完成。任何捕获的heldunlock
//*必须*释放（最好是延迟）或钱包
//将永远保持解锁。
	heldUnlock chan struct{}
)

//WalletLocker管理钱包的锁定/解锁状态。
func (w *Wallet) walletLocker() {
	var timeout <-chan time.Time
	holdChan := make(heldUnlock)
	quit := w.quitChan()
out:
	for {
		select {
		case req := <-w.unlockRequests:
			err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
				addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
				return w.Manager.Unlock(addrmgrNs, req.passphrase)
			})
			if err != nil {
				req.err <- err
				continue
			}
			timeout = req.lockAfter
			if timeout == nil {
				log.Info("The wallet has been unlocked without a time limit")
			} else {
				log.Info("The wallet has been temporarily unlocked")
			}
			req.err <- nil
			continue

		case req := <-w.changePassphrase:
			err := walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) error {
				addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
				return w.Manager.ChangePassphrase(
					addrmgrNs, req.old, req.new, req.private,
					&waddrmgr.DefaultScryptOptions,
				)
			})
			req.err <- err
			continue

		case req := <-w.changePassphrases:
			err := walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) error {
				addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
				err := w.Manager.ChangePassphrase(
					addrmgrNs, req.publicOld, req.publicNew,
					false, &waddrmgr.DefaultScryptOptions,
				)
				if err != nil {
					return err
				}

				return w.Manager.ChangePassphrase(
					addrmgrNs, req.privateOld, req.privateNew,
					true, &waddrmgr.DefaultScryptOptions,
				)
			})
			req.err <- err
			continue

		case req := <-w.holdUnlockRequests:
			if w.Manager.IsLocked() {
				close(req)
				continue
			}

			req <- holdChan
<-holdChan //锁止，直到锁松开。

//如果，在握住未锁定的钱包
//时间，超时已经过期，现在锁定它
//希望下次顶级的时候能解锁
//选择运行。
			select {
			case <-timeout:
//让顶层选择Fallthrough，以便
//钱包被锁住了。
			default:
				continue
			}

		case w.lockState <- w.Manager.IsLocked():
			continue

		case <-quit:
			break out

		case <-w.lockRequests:
		case <-timeout:
		}

//select语句被显式锁或
//计时器即将过期。把经理锁在这里。
		timeout = nil
		err := w.Manager.Lock()
		if err != nil && !waddrmgr.IsError(err, waddrmgr.ErrLocked) {
			log.Errorf("Could not lock wallet: %v", err)
		} else {
			log.Info("The wallet has been locked")
		}
	}
	w.wg.Done()
}

//解锁解锁钱包的地址管理器，并在超时后重新锁定。
//期满。如果钱包已经解锁，新密码是
//正确，当前超时将替换为新超时。钱包会
//如果密码不正确或在
//解锁。
func (w *Wallet) Unlock(passphrase []byte, lock <-chan time.Time) error {
	err := make(chan error, 1)
	w.unlockRequests <- unlockRequest{
		passphrase: passphrase,
		lockAfter:  lock,
		err:        err,
	}
	return <-err
}

//
func (w *Wallet) Lock() {
	w.lockRequests <- struct{}{}
}

//锁定返回钱包的帐户管理器是否锁定。
func (w *Wallet) Locked() bool {
	return <-w.lockState
}

//保持解锁可防止钱包被锁定。Heldunlock对象
//
//
//TODO:为了防止出现上述情况，可能应该传递闭包
//到walletlocker goroutine，不允许呼叫者从
//操作锁止机构。
func (w *Wallet) holdUnlock() (heldUnlock, error) {
	req := make(chan heldUnlock)
	w.holdUnlockRequests <- req
	hl, ok := <-req
	if !ok {
//TODO（Davec）：应该定义它并从中导出
//WADRMGR.
		return nil, waddrmgr.ManagerError{
			ErrorCode:   waddrmgr.ErrLocked,
			Description: "address manager is locked",
		}
	}
	return hl, nil
}

//释放释放钱包解锁状态的保持，并允许
//钱包要再锁上。如果锁定超时已过期，则
//钱包一被释放就被再次锁定。
func (c heldUnlock) release() {
	c <- struct{}{}
}

//changeprivatepassphrase尝试将钱包的密码从
//旧到新。更改密码将与所有其他地址同步
//管理器锁定和解锁。锁定状态将与以前相同
//密码更改前。
func (w *Wallet) ChangePrivatePassphrase(old, new []byte) error {
	err := make(chan error, 1)
	w.changePassphrase <- changePassphraseRequest{
		old:     old,
		new:     new,
		private: true,
		err:     err,
	}
	return <-err
}

//changepublicpassphrase修改钱包的公共密码。
func (w *Wallet) ChangePublicPassphrase(old, new []byte) error {
	err := make(chan error, 1)
	w.changePassphrase <- changePassphraseRequest{
		old:     old,
		new:     new,
		private: false,
		err:     err,
	}
	return <-err
}

//changepassphrases修改钱包的公共和私人密码
//原子性的
func (w *Wallet) ChangePassphrases(publicOld, publicNew, privateOld,
	privateNew []byte) error {

	err := make(chan error, 1)
	w.changePassphrases <- changePassphrasesRequest{
		publicOld:  publicOld,
		publicNew:  publicNew,
		privateOld: privateOld,
		privateNew: privateNew,
		err:        err,
	}
	return <-err
}

//accountused返回是否有任何记录的交易支出
//一个给定的帐户。如果帐户中至少有一个地址是
//如果帐户中没有使用地址，则使用和否。
func (w *Wallet) accountUsed(addrmgrNs walletdb.ReadWriteBucket, account uint32) (bool, error) {
	var used bool
	err := w.Manager.ForEachAccountAddress(addrmgrNs, account,
		func(maddr waddrmgr.ManagedAddress) error {
			used = maddr.Used(addrmgrNs)
			if used {
				return waddrmgr.Break
			}
			return nil
		})
	if err == waddrmgr.Break {
		err = nil
	}
	return used, err
}

//accountAddresses返回
//帐户。
func (w *Wallet) AccountAddresses(account uint32) (addrs []btcutil.Address, err error) {
	err = walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		return w.Manager.ForEachAccountAddress(addrmgrNs, account, func(maddr waddrmgr.ManagedAddress) error {
			addrs = append(addrs, maddr.Address())
			return nil
		})
	})
	return
}

//计算余额合计所有未占用交易的金额
//输出到钱包地址并返回余额。
//
//如果确认为0，则所有的utxo，甚至不存在于
//块（高度-1），将用于获得平衡。否则，
//utxo必须在块中。如果确认为1或更大，
//余额将根据多少块计算
//包括一个UTXO。
func (w *Wallet) CalculateBalance(confirms int32) (btcutil.Amount, error) {
	var balance btcutil.Amount
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)
		var err error
		blk := w.Manager.SyncedTo()
		balance, err = w.TxStore.Balance(txmgrNs, confirms, blk.Height)
		return err
	})
	return balance, err
}

//
//奖励余额。
type Balances struct {
	Total          btcutil.Amount
	Spendable      btcutil.Amount
	ImmatureReward btcutil.Amount
}

//计算账户余额合计所有未用交易的金额
//输出到钱包的指定帐户并返回余额。
//
//由于事务输出，此函数比需要的慢得多
//
//必须迭代输出。
func (w *Wallet) CalculateAccountBalances(account uint32, confirms int32) (Balances, error) {
	var bals Balances
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

//获取当前块。用于计算的块高度
//Tx确认的数目。
		syncBlock := w.Manager.SyncedTo()

		unspent, err := w.TxStore.UnspentOutputs(txmgrNs)
		if err != nil {
			return err
		}
		for i := range unspent {
			output := &unspent[i]

			var outputAcct uint32
			_, addrs, _, err := txscript.ExtractPkScriptAddrs(
				output.PkScript, w.chainParams)
			if err == nil && len(addrs) > 0 {
				_, outputAcct, err = w.Manager.AddrAccount(addrmgrNs, addrs[0])
			}
			if err != nil || outputAcct != account {
				continue
			}

			bals.Total += output.Amount
			if output.FromCoinBase && !confirmed(int32(w.chainParams.CoinbaseMaturity),
				output.Height, syncBlock.Height) {
				bals.ImmatureReward += output.Amount
			} else if confirmed(confirms, output.Height, syncBlock.Height) {
				bals.Spendable += output.Amount
			}
		}
		return nil
	})
	return bals, err
}

//当前地址获取最近请求的比特币支付地址
//从钱包中找到一个特定的钥匙链范围。如果地址已经
//已使用（在
//区块链或btcd mempool），返回下一个链接地址。
func (w *Wallet) CurrentAddress(account uint32, scope waddrmgr.KeyScope) (btcutil.Address, error) {
	chainClient, err := w.requireChainClient()
	if err != nil {
		return nil, err
	}

	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return nil, err
	}

	var (
		addr  btcutil.Address
		props *waddrmgr.AccountProperties
	)
	err = walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		maddr, err := manager.LastExternalAddress(addrmgrNs, account)
		if err != nil {
//如果还不存在地址，请创建第一个外部地址
//地址。
			if waddrmgr.IsError(err, waddrmgr.ErrAddressNotFound) {
				addr, props, err = w.newAddress(
					addrmgrNs, account, scope,
				)
			}
			return err
		}

//
//使用。
		if maddr.Used(addrmgrNs) {
			addr, props, err = w.newAddress(
				addrmgrNs, account, scope,
			)
			return err
		}

		addr = maddr.Address()
		return nil
	})
	if err != nil {
		return nil, err
	}

//如果道具是最初的，那么我们必须创建一个新地址
//to satisfy the query. Notify the rpc server about the new address.
	if props != nil {
		err = chainClient.NotifyReceived([]btcutil.Address{addr})
		if err != nil {
			return nil, err
		}

		w.NtfnServer.notifyAccountProperties(props)
	}

	return addr, nil
}

//PubKeyForAddress looks up the associated public key for a P2PKH address.
func (w *Wallet) PubKeyForAddress(a btcutil.Address) (*btcec.PublicKey, error) {
	var pubKey *btcec.PublicKey
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		managedAddr, err := w.Manager.Address(addrmgrNs, a)
		if err != nil {
			return err
		}
		managedPubKeyAddr, ok := managedAddr.(waddrmgr.ManagedPubKeyAddress)
		if !ok {
			return errors.New("address does not have an associated public key")
		}
		pubKey = managedPubKeyAddr.PubKey()
		return nil
	})
	return pubKey, err
}

//PrivKeyForAddress looks up the associated private key for a P2PKH or P2PK
//地址。
func (w *Wallet) PrivKeyForAddress(a btcutil.Address) (*btcec.PrivateKey, error) {
	var privKey *btcec.PrivateKey
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		managedAddr, err := w.Manager.Address(addrmgrNs, a)
		if err != nil {
			return err
		}
		managedPubKeyAddr, ok := managedAddr.(waddrmgr.ManagedPubKeyAddress)
		if !ok {
			return errors.New("address does not have an associated private key")
		}
		privKey, err = managedPubKeyAddr.PrivKey()
		return err
	})
	return privKey, err
}

//HaveAddress返回钱包是否为地址A的所有者。
func (w *Wallet) HaveAddress(a btcutil.Address) (bool, error) {
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		_, err := w.Manager.Address(addrmgrNs, a)
		return err
	})
	if err == nil {
		return true, nil
	}
	if waddrmgr.IsError(err, waddrmgr.ErrAddressNotFound) {
		return false, nil
	}
	return false, err
}

//AccountOfAddress查找与地址关联的帐户。
func (w *Wallet) AccountOfAddress(a btcutil.Address) (uint32, error) {
	var account uint32
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		var err error
		_, account, err = w.Manager.AddrAccount(addrmgrNs, a)
		return err
	})
	return account, err
}

//AddressInfo返回有关钱包地址的详细信息。
func (w *Wallet) AddressInfo(a btcutil.Address) (waddrmgr.ManagedAddress, error) {
	var managedAddress waddrmgr.ManagedAddress
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		var err error
		managedAddress, err = w.Manager.Address(addrmgrNs, a)
		return err
	})
	return managedAddress, err
}

//account number返回帐户名在
//特定的关键范围。
func (w *Wallet) AccountNumber(scope waddrmgr.KeyScope, accountName string) (uint32, error) {
	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return 0, err
	}

	var account uint32
	err = walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		var err error
		account, err = manager.LookupAccount(addrmgrNs, accountName)
		return err
	})
	return account, err
}

//account name返回帐户名。
func (w *Wallet) AccountName(scope waddrmgr.KeyScope, accountNumber uint32) (string, error) {
	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return "", err
	}

	var accountName string
	err = walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		var err error
		accountName, err = manager.AccountName(addrmgrNs, accountNumber)
		return err
	})
	return accountName, err
}

//account properties返回帐户的属性，包括地址
//indexes and name. It first fetches the desynced information from the address
//管理器，然后根据地址池更新索引。
func (w *Wallet) AccountProperties(scope waddrmgr.KeyScope, acct uint32) (*waddrmgr.AccountProperties, error) {
	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return nil, err
	}

	var props *waddrmgr.AccountProperties
	err = walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		waddrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		var err error
		props, err = manager.AccountProperties(waddrmgrNs, acct)
		return err
	})
	return props, err
}

//renameaccount将帐号的名称设置为newname。
func (w *Wallet) RenameAccount(scope waddrmgr.KeyScope, account uint32, newName string) error {
	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return err
	}

	var props *waddrmgr.AccountProperties
	err = walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		err := manager.RenameAccount(addrmgrNs, account, newName)
		if err != nil {
			return err
		}
		props, err = manager.AccountProperties(addrmgrNs, account)
		return err
	})
	if err == nil {
		w.NtfnServer.notifyAccountProperties(props)
	}
	return err
}

const maxEmptyAccounts = 100

//NextAccount创建下一个帐户并返回其帐号。这个
//名称必须对帐户唯一。为了支持自动播种
//
//账户没有交易历史记录（这是与bip0044的偏差
//spec, which allows no unused account gaps).
func (w *Wallet) NextAccount(scope waddrmgr.KeyScope, name string) (uint32, error) {
	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return 0, err
	}

	var (
		account uint32
		props   *waddrmgr.AccountProperties
	)
	err = walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		var err error
		account, err = manager.NewAccount(addrmgrNs, name)
		if err != nil {
			return err
		}
		props, err = manager.AccountProperties(addrmgrNs, account)
		return err
	})
	if err != nil {
		log.Errorf("Cannot fetch new account properties for notification "+
			"after account creation: %v", err)
	} else {
		w.NtfnServer.notifyAccountProperties(props)
	}
	return account, err
}

//CreditCategory描述钱包交易输出的类型。范畴
//“已发送的交易”（借方）的值始终为“发送”，且不表示为
//这种类型。
//
//TODO: This is a requirement of the RPC server and should be moved.
type CreditCategory byte

//这些常量定义了可能的信用类别。
const (
	CreditReceive CreditCategory = iota
	CreditGenerate
	CreditImmature
)

//String将该类别作为字符串返回。此字符串可以用作
//JSON string for categories as part of listtransactions and gettransaction
//RPC响应。
func (c CreditCategory) String() string {
	switch c {
	case CreditReceive:
		return "receive"
	case CreditGenerate:
		return "generate"
	case CreditImmature:
		return "immature"
	default:
		return "unknown"
	}
}

//recvcategory返回从
//交易记录。通过的区块链高度用于区分
//从成熟的coinbase产出来看是不成熟的。
//
//TODO:这是由RPC服务器使用的，应将其移出
//这个包裹在以后的时间。
func RecvCategory(details *wtxmgr.TxDetails, syncHeight int32, net *chaincfg.Params) CreditCategory {
	if blockchain.IsCoinBaseTx(&details.MsgTx) {
		if confirmed(int32(net.CoinbaseMaturity), details.Block.Height,
			syncHeight) {
			return CreditGenerate
		}
		return CreditImmature
	}
	return CreditReceive
}

//ListTransactions创建一个对象，该对象可能被封送到响应结果
//对于ListTransactions RPC。
//
//TODO:应该将其移到legacyrpc包中。
func listTransactions(tx walletdb.ReadTx, details *wtxmgr.TxDetails, addrMgr *waddrmgr.Manager,
	syncHeight int32, net *chaincfg.Params) []btcjson.ListTransactionsResult {

	addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)

	var (
		blockHashStr  string
		blockTime     int64
		confirmations int64
	)
	if details.Block.Height != -1 {
		blockHashStr = details.Block.Hash.String()
		blockTime = details.Block.Time.Unix()
		confirmations = int64(confirms(details.Block.Height, syncHeight))
	}

	results := []btcjson.ListTransactionsResult{}
	txHashStr := details.Hash.String()
	received := details.Received.Unix()
	generated := blockchain.IsCoinBaseTx(&details.MsgTx)
	recvCat := RecvCategory(details, syncHeight, net).String()

	send := len(details.Debits) != 0

//只有当每个输入都是借方时，才能确定费用。
	var feeF64 float64
	if len(details.Debits) == len(details.MsgTx.TxIn) {
		var debitTotal btcutil.Amount
		for _, deb := range details.Debits {
			debitTotal += deb.Amount
		}
		var outputTotal btcutil.Amount
		for _, output := range details.MsgTx.TxOut {
			outputTotal += btcutil.Amount(output.Value)
		}
//注：实际费用为借记-输出合计。然而，
//此RPC报告费用的负数，因此
//计算。
		feeF64 = (outputTotal - debitTotal).ToBTC()
	}

outputs:
	for i, output := range details.MsgTx.TxOut {
//Determine if this output is a credit, and if so, determine
//它的温和。
		var isCredit bool
		var spentCredit bool
		for _, cred := range details.Credits {
			if cred.Index == uint32(i) {
//更改输出被忽略。
				if cred.Change {
					continue outputs
				}

				isCredit = true
				spentCredit = cred.Spent
				break
			}
		}

		var address string
		var accountName string
		_, addrs, _, _ := txscript.ExtractPkScriptAddrs(output.PkScript, net)
		if len(addrs) == 1 {
			addr := addrs[0]
			address = addr.EncodeAddress()
			mgr, account, err := addrMgr.AddrAccount(addrmgrNs, addrs[0])
			if err == nil {
				accountName, err = mgr.AccountName(addrmgrNs, account)
				if err != nil {
					accountName = ""
				}
			}
		}

		amountF64 := btcutil.Amount(output.Value).ToBTC()
		result := btcjson.ListTransactionsResult{
//字段左零：
//仅限InvolvesWatchOnly
//阻滞剂
//
//字段设置如下：
//帐户（仅适用于非“发送”类别）
//类别
//数量
//费用
			Address:         address,
			Vout:            uint32(i),
			Confirmations:   confirmations,
			Generated:       generated,
			BlockHash:       blockHashStr,
			BlockTime:       blockTime,
			TxID:            txHashStr,
			WalletConflicts: []string{},
			Time:            received,
			TimeReceived:    received,
		}

//如果这是信用证，则添加已收到/生成/未到期的结果。
//如果输出被占用，则在
//发送与输出量相反的类别。它是
//因此，单个输出可能包括在
//结果设置为零、一或两次。
//
//因为没有为输出保存学分
//由该钱包控制，所有非交易信用
//带借项分组在发送类别下。

		if send || spentCredit {
			result.Category = "send"
			result.Amount = -amountF64
			result.Fee = &feeF64
			results = append(results, result)
		}
		if isCredit {
			result.Account = accountName
			result.Category = recvCat
			result.Amount = amountF64
			result.Fee = nil
			results = append(results, result)
		}
	}
	return results
}

//
//
//这将用于listsincoblock rpc应答。
func (w *Wallet) ListSinceBlock(start, end, syncHeight int32) ([]btcjson.ListTransactionsResult, error) {
	txList := []btcjson.ListTransactionsResult{}
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

		rangeFn := func(details []wtxmgr.TxDetails) (bool, error) {
			for _, detail := range details {
				jsonResults := listTransactions(tx, &detail,
					w.Manager, syncHeight, w.chainParams)
				txList = append(txList, jsonResults...)
			}
			return false, nil
		}

		return w.TxStore.RangeTransactions(txmgrNs, start, end, rangeFn)
	})
	return txList, err
}

//ListTransactions返回一个对象切片，其中包含有关记录的
//
//回答。
func (w *Wallet) ListTransactions(from, count int) ([]btcjson.ListTransactionsResult, error) {
	txList := []btcjson.ListTransactionsResult{}

	err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

//获取当前块。用于计算的块高度
//Tx确认的数目。
		syncBlock := w.Manager.SyncedTo()

//需要跳过事务中的第一个，然后，仅
//包括下一个盘点交易记录。
		skipped := 0
		n := 0

		rangeFn := func(details []wtxmgr.TxDetails) (bool, error) {
//以相反的顺序在这个高度迭代事务。
//这对未关联的交易没有任何作用，因为
//未排序，但它将在
//相反的顺序，它们被标记为“地雷”。
			for i := len(details) - 1; i >= 0; i-- {
				if from > skipped {
					skipped++
					continue
				}

				n++
				if n > count {
					return true, nil
				}

				jsonResults := listTransactions(tx, &details[i],
					w.Manager, syncBlock.Height, w.chainParams)
				txList = append(txList, jsonResults...)

				if len(jsonResults) > 0 {
					n++
				}
			}

			return false, nil
		}

//首先返回更新的结果，方法是从mempool高度开始并工作
//一直到创世纪街区。
		return w.TxStore.RangeTransactions(txmgrNs, -1, 0, rangeFn)
	})
	return txList, err
}

//ListAddressTransactions返回对象切片，其中包含有关
//在属于一个集合的任何地址之间记录的交易。这是
//用于ListAddressTransactions RPC答复。
func (w *Wallet) ListAddressTransactions(pkHashes map[string]struct{}) ([]btcjson.ListTransactionsResult, error) {
	txList := []btcjson.ListTransactionsResult{}
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

//获取当前块。用于计算的块高度
//Tx确认的数目。
		syncBlock := w.Manager.SyncedTo()
		rangeFn := func(details []wtxmgr.TxDetails) (bool, error) {
		loopDetails:
			for i := range details {
				detail := &details[i]

				for _, cred := range detail.Credits {
					pkScript := detail.MsgTx.TxOut[cred.Index].PkScript
					_, addrs, _, err := txscript.ExtractPkScriptAddrs(
						pkScript, w.chainParams)
					if err != nil || len(addrs) != 1 {
						continue
					}
					apkh, ok := addrs[0].(*btcutil.AddressPubKeyHash)
					if !ok {
						continue
					}
					_, ok = pkHashes[string(apkh.ScriptAddress())]
					if !ok {
						continue
					}

					jsonResults := listTransactions(tx, detail,
						w.Manager, syncBlock.Height, w.chainParams)
					if err != nil {
						return false, err
					}
					txList = append(txList, jsonResults...)
					continue loopDetails
				}
			}
			return false, nil
		}

		return w.TxStore.RangeTransactions(txmgrNs, 0, -1, rangeFn)
	})
	return txList, err
}

//ListAllTransactions返回一个对象切片，其中包含有关记录的
//
//回答。
func (w *Wallet) ListAllTransactions() ([]btcjson.ListTransactionsResult, error) {
	txList := []btcjson.ListTransactionsResult{}
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

//获取当前块。用于计算的块高度
//Tx确认的数目。
		syncBlock := w.Manager.SyncedTo()

		rangeFn := func(details []wtxmgr.TxDetails) (bool, error) {
//以相反的顺序在这个高度迭代事务。
//这对未关联的交易没有任何作用，因为
//未排序，但它将在
//相反的顺序，它们被标记为“地雷”。
			for i := len(details) - 1; i >= 0; i-- {
				jsonResults := listTransactions(tx, &details[i], w.Manager,
					syncBlock.Height, w.chainParams)
				txList = append(txList, jsonResults...)
			}
			return false, nil
		}

//首先从mempool高度开始返回更新的结果，然后
//一直到Genesis区块。
		return w.TxStore.RangeTransactions(txmgrNs, -1, 0, rangeFn)
	})
	return txList, err
}

//块标识符通过高度或哈希来标识块。
type BlockIdentifier struct {
	height int32
	hash   *chainhash.Hash
}

//NewBlockIdentifierFromHeight为块高度构造块标识符。
func NewBlockIdentifierFromHeight(height int32) *BlockIdentifier {
	return &BlockIdentifier{height: height}
}

//
func NewBlockIdentifierFromHash(hash *chainhash.Hash) *BlockIdentifier {
	return &BlockIdentifier{hash: hash}
}

//
//有关详细信息，请参阅GetTransactions。
type GetTransactionsResult struct {
	MinedTransactions   []Block
	UnminedTransactions []TransactionSummary
}

//GetTransactions返回开始和结束之间的事务结果
//块。块范围内的块可以由高度或
//搞砸。
//
//因为这可能是一个较长的操作，所以提供了一个取消通道。
//取消任务。如果这个通道被阻塞，那么到目前为止所产生的结果
//将被退回。
//
//事务结果按块按升序组织，不进行链接。
//未指定顺序的交易。挖掘的事务保存在
//记录块属性的块结构。
func (w *Wallet) GetTransactions(startBlock, endBlock *BlockIdentifier, cancel <-chan struct{}) (*GetTransactionsResult, error) {
	var start, end int32 = 0, -1

	w.chainClientLock.Lock()
	chainClient := w.chainClient
	w.chainClientLock.Unlock()

//托多：通过他们的哈希值获取块的高度天生就很有活力。
//因为不是所有的块头都保存了，但是当它们是针对SPV时，
//数据库可以直接查询而不需要这个。
	var startResp, endResp rpcclient.FutureGetBlockVerboseResult
	if startBlock != nil {
		if startBlock.hash == nil {
			start = startBlock.height
		} else {
			if chainClient == nil {
				return nil, errors.New("no chain server client")
			}
			switch client := chainClient.(type) {
			case *chain.RPCClient:
				startResp = client.GetBlockVerboseTxAsync(startBlock.hash)
			case *chain.BitcoindClient:
				var err error
				start, err = client.GetBlockHeight(startBlock.hash)
				if err != nil {
					return nil, err
				}
			case *chain.NeutrinoClient:
				var err error
				start, err = client.GetBlockHeight(startBlock.hash)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	if endBlock != nil {
		if endBlock.hash == nil {
			end = endBlock.height
		} else {
			if chainClient == nil {
				return nil, errors.New("no chain server client")
			}
			switch client := chainClient.(type) {
			case *chain.RPCClient:
				endResp = client.GetBlockVerboseTxAsync(endBlock.hash)
			case *chain.NeutrinoClient:
				var err error
				end, err = client.GetBlockHeight(endBlock.hash)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	if startResp != nil {
		resp, err := startResp.Receive()
		if err != nil {
			return nil, err
		}
		start = int32(resp.Height)
	}
	if endResp != nil {
		resp, err := endResp.Receive()
		if err != nil {
			return nil, err
		}
		end = int32(resp.Height)
	}

	var res GetTransactionsResult
	err := walletdb.View(w.db, func(dbtx walletdb.ReadTx) error {
		txmgrNs := dbtx.ReadBucket(wtxmgrNamespaceKey)

		rangeFn := func(details []wtxmgr.TxDetails) (bool, error) {
//TODO:可能应该使RangeTransactions不重用
//详细信息备份阵列内存。
			dets := make([]wtxmgr.TxDetails, len(details))
			copy(dets, details)
			details = dets

			txs := make([]TransactionSummary, 0, len(details))
			for i := range details {
				txs = append(txs, makeTxSummary(dbtx, w, &details[i]))
			}

			if details[0].Block.Height != -1 {
				blockHash := details[0].Block.Hash
				res.MinedTransactions = append(res.MinedTransactions, Block{
					Hash:         &blockHash,
					Height:       details[0].Block.Height,
					Timestamp:    details[0].Block.Time.Unix(),
					Transactions: txs,
				})
			} else {
				res.UnminedTransactions = txs
			}

			select {
			case <-cancel:
				return true, nil
			default:
				return false, nil
			}
		}

		return w.TxStore.RangeTransactions(txmgrNs, start, end, rangeFn)
	})
	return &res, err
}

//account result是accountsresult类型的单个帐户结果。
type AccountResult struct {
	waddrmgr.AccountProperties
	TotalBalance btcutil.Amount
}

//账户结果是钱包账户方法的结果。看到那个
//
type AccountsResult struct {
	Accounts           []AccountResult
	CurrentBlockHash   *chainhash.Hash
	CurrentBlockHeight int32
}

//帐户返回所有帐户的当前名称、数字和总余额
//钱包中的帐户仅限于特定的密钥范围。电流
//
//
//TODO（JRICK）：是否确实需要链端，因为只有总余额
//包括在内吗？
func (w *Wallet) Accounts(scope waddrmgr.KeyScope) (*AccountsResult, error) {
	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return nil, err
	}

	var (
		accounts        []AccountResult
		syncBlockHash   *chainhash.Hash
		syncBlockHeight int32
	)
	err = walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

		syncBlock := w.Manager.SyncedTo()
		syncBlockHash = &syncBlock.Hash
		syncBlockHeight = syncBlock.Height
		unspent, err := w.TxStore.UnspentOutputs(txmgrNs)
		if err != nil {
			return err
		}
		err = manager.ForEachAccount(addrmgrNs, func(acct uint32) error {
			props, err := manager.AccountProperties(addrmgrNs, acct)
			if err != nil {
				return err
			}
			accounts = append(accounts, AccountResult{
				AccountProperties: *props,
//
			})
			return nil
		})
		if err != nil {
			return err
		}
		m := make(map[uint32]*btcutil.Amount)
		for i := range accounts {
			a := &accounts[i]
			m[a.AccountNumber] = &a.TotalBalance
		}
		for i := range unspent {
			output := unspent[i]
			var outputAcct uint32
			_, addrs, _, err := txscript.ExtractPkScriptAddrs(output.PkScript, w.chainParams)
			if err == nil && len(addrs) > 0 {
				_, outputAcct, err = w.Manager.AddrAccount(addrmgrNs, addrs[0])
			}
			if err == nil {
				amt, ok := m[outputAcct]
				if ok {
					*amt += output.Amount
				}
			}
		}
		return nil
	})
	return &AccountsResult{
		Accounts:           accounts,
		CurrentBlockHash:   syncBlockHash,
		CurrentBlockHeight: syncBlockHeight,
	}, err
}

//accountBalanceResult是wallet.accountBalances方法的单个结果。
type AccountBalanceResult struct {
	AccountNumber  uint32
	AccountName    string
	AccountBalance btcutil.Amount
}

//账户余额返回钱包中的所有账户及其余额。
//余额通过排除未满足的交易来确定。
//要求确认。
func (w *Wallet) AccountBalances(scope waddrmgr.KeyScope,
	requiredConfs int32) ([]AccountBalanceResult, error) {

	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return nil, err
	}

	var results []AccountBalanceResult
	err = walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

		syncBlock := w.Manager.SyncedTo()

//填写除余额以外的所有账户信息。
		lastAcct, err := manager.LastAccount(addrmgrNs)
		if err != nil {
			return err
		}
		results = make([]AccountBalanceResult, lastAcct+2)
		for i := range results[:len(results)-1] {
			accountName, err := manager.AccountName(addrmgrNs, uint32(i))
			if err != nil {
				return err
			}
			results[i].AccountNumber = uint32(i)
			results[i].AccountName = accountName
		}
		results[len(results)-1].AccountNumber = waddrmgr.ImportedAddrAccount
		results[len(results)-1].AccountName = waddrmgr.ImportedAddrAccountName

//获取所有未使用的输出，并对其进行迭代，计算每个输出
//输出脚本支付到帐户地址的帐户余额
//并且满足所需的确认数量。
		unspentOutputs, err := w.TxStore.UnspentOutputs(txmgrNs)
		if err != nil {
			return err
		}
		for i := range unspentOutputs {
			output := &unspentOutputs[i]
			if !confirmed(requiredConfs, output.Height, syncBlock.Height) {
				continue
			}
			if output.FromCoinBase && !confirmed(int32(w.ChainParams().CoinbaseMaturity),
				output.Height, syncBlock.Height) {
				continue
			}
			_, addrs, _, err := txscript.ExtractPkScriptAddrs(output.PkScript, w.chainParams)
			if err != nil || len(addrs) == 0 {
				continue
			}
			outputAcct, err := manager.AddrAccount(addrmgrNs, addrs[0])
			if err != nil {
				continue
			}
			switch {
			case outputAcct == waddrmgr.ImportedAddrAccount:
				results[len(results)-1].AccountBalance += output.Amount
			case outputAcct > lastAcct:
				return errors.New("waddrmgr.Manager.AddrAccount returned account " +
					"beyond recorded last account")
			default:
				results[outputAcct].AccountBalance += output.Amount
			}
		}
		return nil
	})
	return results, err
}

//creditslice满足sort.interface接口以提供排序
//从最旧到最新的交易信用。同一收货信用证
//同一区块的时间和开采量不保证按顺序排序。
//他们出现在街区里。来自同一笔交易的贷记按
//输出指数。
type creditSlice []wtxmgr.Credit

func (s creditSlice) Len() int {
	return len(s)
}

func (s creditSlice) Less(i, j int) bool {
	switch {
//如果两个信用证来自同一个Tx，则按输出索引排序。
	case s[i].OutPoint.Hash == s[j].OutPoint.Hash:
		return s[i].OutPoint.Index < s[j].OutPoint.Index

//如果这两个交易都没有关联，则按接收日期排序。
	case s[i].Height == -1 && s[j].Height == -1:
		return s[i].Received.Before(s[j].Received)

//未链接（更新）的TXS总是排在最后。
	case s[i].Height == -1:
		return false
	case s[j].Height == -1:
		return true

//如果两个TXS都开采在不同的区块中，则按区块高度排序。
	default:
		return s[i].Height < s[j].Height
	}
}

func (s creditSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

//listunspent返回代表未使用钱包的对象切片
//符合给定条件的事务。确认将超过
//minconf，小于maxconf，如果只填充地址，则只填充地址
//将考虑其中包含的内容。如果我们不知道
//事务将返回空数组。
func (w *Wallet) ListUnspent(minconf, maxconf int32,
	addresses map[string]struct{}) ([]*btcjson.ListUnspentResult, error) {

	var results []*btcjson.ListUnspentResult
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

		syncBlock := w.Manager.SyncedTo()

		filter := len(addresses) != 0
		unspent, err := w.TxStore.UnspentOutputs(txmgrNs)
		if err != nil {
			return err
		}
		sort.Sort(sort.Reverse(creditSlice(unspent)))

		defaultAccountName := "default"

		results = make([]*btcjson.ListUnspentResult, 0, len(unspent))
		for i := range unspent {
			output := unspent[i]

//确认数少于最小值或更多的输出
//不包括超过最大值的配置。
			confs := confirms(output.Height, syncBlock.Height)
			if confs < minconf || confs > maxconf {
				continue
			}

//仅包括成熟的Coinbase输出。
			if output.FromCoinBase {
				target := int32(w.ChainParams().CoinbaseMaturity)
				if !confirmed(target, output.Height, syncBlock.Height) {
					continue
				}
			}

//从结果集中排除锁定的输出。
			if w.LockedOutpoint(output.OutPoint) {
				continue
			}

//查找输出的关联帐户。使用
//没有关联帐户时的默认帐户名
//出于某种原因，尽管这不应该发生。
//
//一旦事务和输出
//分组在数据库中的关联帐户下。
			acctName := defaultAccountName
			sc, addrs, _, err := txscript.ExtractPkScriptAddrs(
				output.PkScript, w.chainParams)
			if err != nil {
				continue
			}
			if len(addrs) > 0 {
				smgr, acct, err := w.Manager.AddrAccount(addrmgrNs, addrs[0])
				if err == nil {
					s, err := smgr.AccountName(addrmgrNs, acct)
					if err == nil {
						acctName = s
					}
				}
			}

			if filter {
				for _, addr := range addrs {
					_, ok := addresses[addr.EncodeAddress()]
					if ok {
						goto include
					}
				}
				continue
			}

		include:
//目前不支持只监视地址，因此
//记录的非多信号输出是“可消耗的”。
//如果所有键都是
//由这个钱包控制。
//
//TODO:当只监视addr时，每个案例都需要更新
//添加。对于p2pk、p2pkh和p2sh，地址必须为
//抬头看，而不是只看。对于multisig，所有
//PubKeys必须属于具有关联的
//私钥（目前只检查pubkey
//存在，因为此时需要私钥）。
			var spendable bool
		scSwitch:
			switch sc {
			case txscript.PubKeyHashTy:
				spendable = true
			case txscript.PubKeyTy:
				spendable = true
			case txscript.WitnessV0ScriptHashTy:
				spendable = true
			case txscript.WitnessV0PubKeyHashTy:
				spendable = true
			case txscript.MultiSigTy:
				for _, a := range addrs {
					_, err := w.Manager.Address(addrmgrNs, a)
					if err == nil {
						continue
					}
					if waddrmgr.IsError(err, waddrmgr.ErrAddressNotFound) {
						break scSwitch
					}
					return err
				}
				spendable = true
			}

			result := &btcjson.ListUnspentResult{
				TxID:          output.OutPoint.Hash.String(),
				Vout:          output.OutPoint.Index,
				Account:       acctName,
				ScriptPubKey:  hex.EncodeToString(output.PkScript),
				Amount:        output.Amount.ToBTC(),
				Confirmations: int64(confs),
				Spendable:     spendable,
			}

//bug：这应该是一个JSON数组，因此
//可以包括或删除地址（以及
//调用者从pkscript中提取地址）。
			if len(addrs) > 0 {
				result.Address = addrs[0].EncodeAddress()
			}

			results = append(results, result)
		}
		return nil
	})
	return results, err
}

//dumpprivkeys返回WIF编码的所有地址的私钥
//钱包里的私人钥匙。
func (w *Wallet) DumpPrivKeys() ([]string, error) {
	var privkeys []string
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
//循环访问每个活动地址，将私钥附加到
//私钥。
		return w.Manager.ForEachActiveAddress(addrmgrNs, func(addr btcutil.Address) error {
			ma, err := w.Manager.Address(addrmgrNs, addr)
			if err != nil {
				return err
			}

//只需要那些带密钥的地址。
			pka, ok := ma.(waddrmgr.ManagedPubKeyAddress)
			if !ok {
				return nil
			}

			wif, err := pka.ExportPrivKey()
			if err != nil {
//最好把这里的数组调零。然而，
//
//我想我们不能控制打电话的人。：（
				return err
			}
			privkeys = append(privkeys, wif.String())
			return nil
		})
	})
	return privkeys, err
}

//dumpwifprivatekey返回
//单个钱包地址。
func (w *Wallet) DumpWIFPrivateKey(addr btcutil.Address) (string, error) {
	var maddr waddrmgr.ManagedAddress
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		waddrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
//如果钱包里有私人钥匙，就从钱包里拿出来。
		var err error
		maddr, err = w.Manager.Address(waddrmgrNs, addr)
		return err
	})
	if err != nil {
		return "", err
	}

	pka, ok := maddr.(waddrmgr.ManagedPubKeyAddress)
	if !ok {
		return "", fmt.Errorf("address %s is not a key type", addr)
	}

	wif, err := pka.ExportPrivKey()
	if err != nil {
		return "", err
	}
	return wif.String(), nil
}

//importprivatekey将私钥导入钱包并写入新的
//钱包到磁盘。
//
//注：如果未提供盖戳，则钱包的生日为
//设置到相应链的Genesis块。
func (w *Wallet) ImportPrivateKey(scope waddrmgr.KeyScope, wif *btcutil.WIF,
	bs *waddrmgr.BlockStamp, rescan bool) (string, error) {

	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return "", err
	}

//钥匙的起始块是Genesis块，除非另有说明。
//明确规定。
	if bs == nil {
		bs = &waddrmgr.BlockStamp{
			Hash:      *w.chainParams.GenesisHash,
			Height:    0,
			Timestamp: w.chainParams.GenesisBlock.Header.Timestamp,
		}
	} else if bs.Timestamp.IsZero() {
//如果仅使用默认值，只更新新生日时间
//实际上在头中有时间戳信息。
		header, err := w.chainClient.GetBlockHeader(&bs.Hash)
		if err == nil {
			bs.Timestamp = header.Timestamp
		}
	}

//尝试将私钥导入钱包。
	var addr btcutil.Address
	var props *waddrmgr.AccountProperties
	err = walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		maddr, err := manager.ImportPrivateKey(addrmgrNs, wif, bs)
		if err != nil {
			return err
		}
		addr = maddr.Address()
		props, err = manager.AccountProperties(
			addrmgrNs, waddrmgr.ImportedAddrAccount,
		)
		if err != nil {
			return err
		}

//如果是的话，我们只会用新的生日更新我们的生日。
//在我们现在的那个之前。否则，如果我们这样做，我们可以
//可能无法检测到
//在重新扫描时在它们之间发生。
		birthdayBlock, _, err := w.Manager.BirthdayBlock(addrmgrNs)
		if err != nil {
			return err
		}
		if bs.Height >= birthdayBlock.Height {
			return nil
		}

		err = w.Manager.SetBirthday(addrmgrNs, bs.Timestamp)
		if err != nil {
			return err
		}

//为了确保这个生日块是正确的，我们将它标记为
//未经验证，在下次重新启动时提示进行健全性检查
//确保它是正确的，因为它是由呼叫方提供的。
		return w.Manager.SetBirthdayBlock(addrmgrNs, *bs, false)
	})
	if err != nil {
		return "", err
	}

//重新扫描区块链以进行交易，txout脚本支付给
//导入地址。
	if rescan {
		job := &RescanJob{
			Addrs:      []btcutil.Address{addr},
			OutPoints:  nil,
			BlockStamp: *bs,
		}

//导入完成后提交重新扫描作业和日志。
//完成重新扫描时不要阻塞。重新扫描成功
//或者故障记录在其他地方，而通道没有
//required to be read, so discard the return value.
		_ = w.SubmitRescan(job)
	} else {
		err := w.chainClient.NotifyReceived([]btcutil.Address{addr})
		if err != nil {
			return "", fmt.Errorf("Failed to subscribe for address ntfns for "+
				"address %s: %s", addr.EncodeAddress(), err)
		}
	}

	addrStr := addr.EncodeAddress()
	log.Infof("Imported payment address %s", addrStr)

	w.NtfnServer.notifyAccountProperties(props)

//返回导入的私钥的付款地址字符串。
	return addrStr, nil
}

//lockedOutpoint返回是否已将输出点标记为已锁定和
//不应用作已创建交易记录的输入。
func (w *Wallet) LockedOutpoint(op wire.OutPoint) bool {
	_, locked := w.lockedOutpoints[op]
	return locked
}

//lockoutpoint将一个输出点标记为已锁定，即不应将其用作
//新创建的交易记录的输入。
func (w *Wallet) LockOutpoint(op wire.OutPoint) {
	w.lockedOutpoints[op] = struct{}{}
}

//unlockoutpoint将一个输出点标记为unlocked，也就是说，它可以用作
//
func (w *Wallet) UnlockOutpoint(op wire.OutPoint) {
	delete(w.lockedOutpoints, op)
}

//ResetLockedOutPoints重置锁定的输出点集，以便可以使用所有输出点
//作为新交易的输入。
func (w *Wallet) ResetLockedOutpoints() {
	w.lockedOutpoints = map[wire.OutPoint]struct{}{}
}

//LockedOutpoints返回当前锁定的输出点的切片。这是
//用于将结果封送为
//
func (w *Wallet) LockedOutpoints() []btcjson.TransactionInput {
	locked := make([]btcjson.TransactionInput, len(w.lockedOutpoints))
	i := 0
	for op := range w.lockedOutpoints {
		locked[i] = btcjson.TransactionInput{
			Txid: op.Hash.String(),
			Vout: op.Index,
		}
		i++
	}
	return locked
}

//ResendUnminedTXS迭代从Wallet支出的所有交易
//不知道被挖掘成一个区块的学分，以及尝试
//发送到链服务器进行中继。
func (w *Wallet) resendUnminedTxs() {
	chainClient, err := w.requireChainClient()
	if err != nil {
		log.Errorf("No chain server available to resend unmined transactions")
		return
	}

	var txs []*wire.MsgTx
	err = walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)
		var err error
		txs, err = w.TxStore.UnminedTxs(txmgrNs)
		return err
	})
	if err != nil {
		log.Errorf("Cannot load unmined transactions for resending: %v", err)
		return
	}

	for _, tx := range txs {
		resp, err := chainClient.SendRawTransaction(tx, false)
		if err != nil {
//如果交易已被接受
//mempool，我们可以继续而不记录错误。
			switch {
			case strings.Contains(err.Error(), "already have transaction"):
				fallthrough
			case strings.Contains(err.Error(), "txn-already-known"):
				continue
			}

			log.Debugf("Could not resend transaction %v: %v",
				tx.TxHash(), err)

//只有当我们
//检测到输出已完全耗尽，
//是孤儿，或与其他人冲突
//交易。
//
//TODO（roasbef）：sendrawtransaction需要返回
//具体的错误类型，不需要字符串匹配
			switch {
//以下是从BTCD返回的错误
//内存池。
			case strings.Contains(err.Error(), "spent"):
			case strings.Contains(err.Error(), "orphan"):
			case strings.Contains(err.Error(), "conflict"):
			case strings.Contains(err.Error(), "already exists"):
			case strings.Contains(err.Error(), "negative"):

//以下错误从比特币返回
//内存池。
			case strings.Contains(err.Error(), "Missing inputs"):
			case strings.Contains(err.Error(), "already in block chain"):
			case strings.Contains(err.Error(), "fee not met"):

			default:
				continue
			}

//由于交易被拒绝，我们将尝试
//同时删除未链接的事务。
//否则，我们会继续尝试重播这个，
//我们可能计算不正确的余额，如果
//这项交易记入我们的贷方或借方。
//
//
//桶-需要确定确认块。
			err := walletdb.Update(w.db, func(dbTx walletdb.ReadWriteTx) error {
				txmgrNs := dbTx.ReadWriteBucket(wtxmgrNamespaceKey)

				txRec, err := wtxmgr.NewTxRecordFromMsgTx(
					tx, time.Now(),
				)
				if err != nil {
					return err
				}

				return w.TxStore.RemoveUnminedTx(txmgrNs, txRec)
			})
			if err != nil {
				log.Warnf("unable to remove conflicting "+
					"tx %v: %v", tx.TxHash(), err)
				continue
			}

			log.Infof("Removed conflicting tx: %v", spew.Sdump(tx))

			continue
		}
		log.Debugf("Resent unmined transaction %v", resp)
	}
}

//SortedActivePaymentAddresses返回所有活动付款的一部分
//钱包里的地址。
func (w *Wallet) SortedActivePaymentAddresses() ([]string, error) {
	var addrStrs []string
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		return w.Manager.ForEachActiveAddress(addrmgrNs, func(addr btcutil.Address) error {
			addrStrs = append(addrStrs, addr.EncodeAddress())
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	sort.Sort(sort.StringSlice(addrStrs))
	return addrStrs, nil
}

//
func (w *Wallet) NewAddress(account uint32,
	scope waddrmgr.KeyScope) (btcutil.Address, error) {

	chainClient, err := w.requireChainClient()
	if err != nil {
		return nil, err
	}

	var (
		addr  btcutil.Address
		props *waddrmgr.AccountProperties
	)
	err = walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		var err error
		addr, props, err = w.newAddress(addrmgrNs, account, scope)
		return err
	})
	if err != nil {
		return nil, err
	}

//通知RPC服务器新创建的地址。
	err = chainClient.NotifyReceived([]btcutil.Address{addr})
	if err != nil {
		return nil, err
	}

	w.NtfnServer.notifyAccountProperties(props)

	return addr, nil
}

func (w *Wallet) newAddress(addrmgrNs walletdb.ReadWriteBucket, account uint32,
	scope waddrmgr.KeyScope) (btcutil.Address, *waddrmgr.AccountProperties, error) {

	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return nil, nil, err
	}

//从钱包里取下一个地址。
	addrs, err := manager.NextExternalAddresses(addrmgrNs, account, 1)
	if err != nil {
		return nil, nil, err
	}

	props, err := manager.AccountProperties(addrmgrNs, account)
	if err != nil {
		log.Errorf("Cannot fetch account properties for notification "+
			"after deriving next external address: %v", err)
		return nil, nil, err
	}

	return addrs[0].Address(), props, nil
}

//new change address返回钱包的新更改地址。
func (w *Wallet) NewChangeAddress(account uint32,
	scope waddrmgr.KeyScope) (btcutil.Address, error) {

	chainClient, err := w.requireChainClient()
	if err != nil {
		return nil, err
	}

	var addr btcutil.Address
	err = walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		var err error
		addr, err = w.newChangeAddress(addrmgrNs, account)
		return err
	})
	if err != nil {
		return nil, err
	}

//通知RPC服务器新创建的地址。
	err = chainClient.NotifyReceived([]btcutil.Address{addr})
	if err != nil {
		return nil, err
	}

	return addr, nil
}

//new change address返回钱包的新更改地址。
//
//注意：此方法要求调用方使用后端的notifyReceived
//用于检测链上交易何时支付到地址的方法
//正在创造。
func (w *Wallet) newChangeAddress(addrmgrNs walletdb.ReadWriteBucket,
	account uint32) (btcutil.Address, error) {

//当我们更改地址时，我们会找到经理的类型
//这可以使p2wkh输出，因为它们是最有效的。
	scopes := w.Manager.ScopesForExternalAddrType(
		waddrmgr.WitnessPubKey,
	)
	manager, err := w.Manager.FetchScopedKeyManager(scopes[0])
	if err != nil {
		return nil, err
	}

//从钱包中获取下一个链接的帐户更改地址。
	addrs, err := manager.NextInternalAddresses(addrmgrNs, account, 1)
	if err != nil {
		return nil, err
	}

	return addrs[0].Address(), nil
}

//已确认检查高度txheight处的事务是否符合minconf
//Height Curheight区块链的确认。
func confirmed(minconf, txHeight, curHeight int32) bool {
	return confirms(txHeight, curHeight) >= minconf
}

//确认返回块中某个事务的确认数
//高度tx高度（或未确认tx的-1）给定链条高度
//光洁度。
func confirms(txHeight, curHeight int32) int32 {
	switch {
	case txHeight == -1, txHeight > curHeight:
		return 0
	default:
		return curHeight - txHeight + 1
	}
}

//AccountTotalReceivedResult是
//wallet.totalReceivedForAccounts方法。
type AccountTotalReceivedResult struct {
	AccountNumber    uint32
	AccountName      string
	TotalReceived    btcutil.Amount
	LastConfirmation int32
}

//totalReceivedForaccounts迭代钱包的交易历史记录，
//返回所有账户收到的比特币总额。
func (w *Wallet) TotalReceivedForAccounts(scope waddrmgr.KeyScope,
	minConf int32) ([]AccountTotalReceivedResult, error) {

	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return nil, err
	}

	var results []AccountTotalReceivedResult
	err = walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

		syncBlock := w.Manager.SyncedTo()

		err := manager.ForEachAccount(addrmgrNs, func(account uint32) error {
			accountName, err := manager.AccountName(addrmgrNs, account)
			if err != nil {
				return err
			}
			results = append(results, AccountTotalReceivedResult{
				AccountNumber: account,
				AccountName:   accountName,
			})
			return nil
		})
		if err != nil {
			return err
		}

		var stopHeight int32

		if minConf > 0 {
			stopHeight = syncBlock.Height - minConf + 1
		} else {
			stopHeight = -1
		}

		rangeFn := func(details []wtxmgr.TxDetails) (bool, error) {
			for i := range details {
				detail := &details[i]
				for _, cred := range detail.Credits {
					pkScript := detail.MsgTx.TxOut[cred.Index].PkScript
					var outputAcct uint32
					_, addrs, _, err := txscript.ExtractPkScriptAddrs(pkScript, w.chainParams)
					if err == nil && len(addrs) > 0 {
						_, outputAcct, err = w.Manager.AddrAccount(addrmgrNs, addrs[0])
					}
					if err == nil {
						acctIndex := int(outputAcct)
						if outputAcct == waddrmgr.ImportedAddrAccount {
							acctIndex = len(results) - 1
						}
						res := &results[acctIndex]
						res.TotalReceived += cred.Amount
						res.LastConfirmation = confirms(
							detail.Block.Height, syncBlock.Height)
					}
				}
			}
			return false, nil
		}
		return w.TxStore.RangeTransactions(txmgrNs, 0, stopHeight, rangeFn)
	})
	return results, err
}

//totalReceivedForAddr迭代钱包的交易历史记录，
//返回单个钱包收到的比特币总额
//地址。
func (w *Wallet) TotalReceivedForAddr(addr btcutil.Address, minConf int32) (btcutil.Amount, error) {
	var amount btcutil.Amount
	err := walletdb.View(w.db, func(tx walletdb.ReadTx) error {
		txmgrNs := tx.ReadBucket(wtxmgrNamespaceKey)

		syncBlock := w.Manager.SyncedTo()

		var (
			addrStr    = addr.EncodeAddress()
			stopHeight int32
		)

		if minConf > 0 {
			stopHeight = syncBlock.Height - minConf + 1
		} else {
			stopHeight = -1
		}
		rangeFn := func(details []wtxmgr.TxDetails) (bool, error) {
			for i := range details {
				detail := &details[i]
				for _, cred := range detail.Credits {
					pkScript := detail.MsgTx.TxOut[cred.Index].PkScript
					_, addrs, _, err := txscript.ExtractPkScriptAddrs(pkScript,
						w.chainParams)
//仅从输出脚本创建地址时出错
//表示非标准脚本，因此忽略此信用。
					if err != nil {
						continue
					}
					for _, a := range addrs {
						if addrStr == a.EncodeAddress() {
							amount += cred.Amount
							break
						}
					}
				}
			}
			return false, nil
		}
		return w.TxStore.RangeTransactions(txmgrNs, 0, stopHeight, rangeFn)
	})
	return amount, err
}

//sendOutputs创建并发送付款事务。它返回
//交易成功。
func (w *Wallet) SendOutputs(outputs []*wire.TxOut, account uint32,
	minconf int32, satPerKb btcutil.Amount) (*wire.MsgTx, error) {

//确保要创建的输出符合网络的共识
//规则。
	for _, output := range outputs {
		if err := txrules.CheckOutput(output, satPerKb); err != nil {
			return nil, err
		}
	}

//创建事务并将其广播到网络。这个
//事务将添加到数据库中，以确保
//重新启动时继续重新广播事务，直到
//得到证实。
	createdTx, err := w.CreateSimpleTx(account, outputs, minconf, satPerKb)
	if err != nil {
		return nil, err
	}

	txHash, err := w.publishTransaction(createdTx.Tx)
	if err != nil {
		return nil, err
	}

//对返回的Tx哈希进行健全性检查。
	if *txHash != createdTx.Tx.TxHash() {
		return nil, errors.New("tx hash mismatch")
	}

	return createdTx.Tx, nil
}

//SignatureError records the underlying error when validating a transaction
//输入签名。
type SignatureError struct {
	InputIndex uint32
	Error      error
}

//SignTransaction使用钱包的秘密以及其他秘密
//由调用方传入，以创建输入签名并将其添加到事务。
//
//事务输入脚本验证用于确认所有签名
//是有效的。对于任何无效的输入，将在返回中添加SignatureError。
//最后的错误返回是为意外或致命错误保留的，例如
//无法确定要兑现的上一个输出脚本。
//
//Tx指向的事务由该函数修改。
func (w *Wallet) SignTransaction(tx *wire.MsgTx, hashType txscript.SigHashType,
	additionalPrevScripts map[wire.OutPoint][]byte,
	additionalKeysByAddress map[string]*btcutil.WIF,
	p2shRedeemScriptsByAddress map[string][]byte) ([]SignatureError, error) {

	var signErrors []SignatureError
	err := walletdb.View(w.db, func(dbtx walletdb.ReadTx) error {
		addrmgrNs := dbtx.ReadBucket(waddrmgrNamespaceKey)
		txmgrNs := dbtx.ReadBucket(wtxmgrNamespaceKey)

		for i, txIn := range tx.TxIn {
			prevOutScript, ok := additionalPrevScripts[txIn.PreviousOutPoint]
			if !ok {
				prevHash := &txIn.PreviousOutPoint.Hash
				prevIndex := txIn.PreviousOutPoint.Index
				txDetails, err := w.TxStore.TxDetails(txmgrNs, prevHash)
				if err != nil {
					return fmt.Errorf("cannot query previous transaction "+
						"details for %v: %v", txIn.PreviousOutPoint, err)
				}
				if txDetails == nil {
					return fmt.Errorf("%v not found",
						txIn.PreviousOutPoint)
				}
				prevOutScript = txDetails.MsgTx.TxOut[prevIndex].PkScript
			}

//设置我们传递给txscript的回调，以便它
//按地址查找适当的键和脚本。
			getKey := txscript.KeyClosure(func(addr btcutil.Address) (*btcec.PrivateKey, bool, error) {
				if len(additionalKeysByAddress) != 0 {
					addrStr := addr.EncodeAddress()
					wif, ok := additionalKeysByAddress[addrStr]
					if !ok {
						return nil, false,
							errors.New("no key for address")
					}
					return wif.PrivKey, wif.CompressPubKey, nil
				}
				address, err := w.Manager.Address(addrmgrNs, addr)
				if err != nil {
					return nil, false, err
				}

				pka, ok := address.(waddrmgr.ManagedPubKeyAddress)
				if !ok {
					return nil, false, fmt.Errorf("address %v is not "+
						"a pubkey address", address.Address().EncodeAddress())
				}

				key, err := pka.PrivKey()
				if err != nil {
					return nil, false, err
				}

				return key, pka.Compressed(), nil
			})
			getScript := txscript.ScriptClosure(func(addr btcutil.Address) ([]byte, error) {
//如果提供了密钥，那么我们只能使用
//我们的输入也提供了脚本。
				if len(additionalKeysByAddress) != 0 {
					addrStr := addr.EncodeAddress()
					script, ok := p2shRedeemScriptsByAddress[addrStr]
					if !ok {
						return nil, errors.New("no script for address")
					}
					return script, nil
				}
				address, err := w.Manager.Address(addrmgrNs, addr)
				if err != nil {
					return nil, err
				}
				sa, ok := address.(waddrmgr.ManagedScriptAddress)
				if !ok {
					return nil, errors.New("address is not a script" +
						" address")
				}

				return sa.Script()
			})

//只有在
//相应的输出。然而，这可能已经签署，
//所以我们总是验证输出。
			if (hashType&txscript.SigHashSingle) !=
				txscript.SigHashSingle || i < len(tx.TxOut) {

				script, err := txscript.SignTxOutput(w.ChainParams(),
					tx, i, prevOutScript, hashType, getKey,
					getScript, txIn.SignatureScript)
//
//TX没有完成。
				if err != nil {
					signErrors = append(signErrors, SignatureError{
						InputIndex: uint32(i),
						Error:      err,
					})
					continue
				}
				txIn.SignatureScript = script
			}

//
//找出它是完全满足还是仍然需要更多。
			vm, err := txscript.NewEngine(prevOutScript, tx, i,
				txscript.StandardVerifyFlags, nil, nil, 0)
			if err == nil {
				err = vm.Execute()
			}
			if err != nil {
				signErrors = append(signErrors, SignatureError{
					InputIndex: uint32(i),
					Error:      err,
				})
			}
		}
		return nil
	})
	return signErrors, err
}

//PublishTransaction将事务发送到协商一致的RPC服务器，因此
//
//
//此函数不稳定，一旦移出同步代码，将删除此函数
//钱包里的
func (w *Wallet) PublishTransaction(tx *wire.MsgTx) error {
	_, err := w.publishTransaction(tx)
	return err
}

//PublishTransaction是PublishTransaction的私有版本，它
//包含发布事务、更新所需的主逻辑
//相关的数据库状态，最后可能删除事务
//从数据库（以及清除所有使用的输入和输出
//已创建）如果事务被后端拒绝。
func (w *Wallet) publishTransaction(tx *wire.MsgTx) (*chainhash.Hash, error) {
	chainClient, err := w.requireChainClient()
	if err != nil {
		return nil, err
	}

//我们的目标是成为通用可靠的事务广播API，
//
//
//记录集。
	txRec, err := wtxmgr.NewTxRecordFromMsgTx(tx, time.Now())
	if err != nil {
		return nil, err
	}
	err = walletdb.Update(w.db, func(dbTx walletdb.ReadWriteTx) error {
		return w.addRelevantTx(dbTx, txRec, nil)
	})
	if err != nil {
		return nil, err
	}

//We'll also ask to be notified of the transaction once it confirms
//链上。This is done outside of the database transaction to prevent
//backend interaction within it.
//
//NOTE: In some cases, it's possible that the transaction to be
//broadcast is not directly relevant to the user's wallet, e.g.,
//多西格在这两种情况下，我们都会要求通知它什么时候。
//确认保持一致性。
//
//TODO（WILMER）：如果地址不为外部，则将脚本作为外部脚本导入
//属于重启时处理conf的钱包吗？
	for _, txOut := range tx.TxOut {
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(
			txOut.PkScript, w.chainParams,
		)
		if err != nil {
//可以安全地跳过非标准输出，因为
//钱包不支持它们。
			continue
		}

		if err := chainClient.NotifyReceived(addrs); err != nil {
			return nil, err
		}
	}

	txid, err := chainClient.SendRawTransaction(tx, false)
	switch {
	case err == nil:
		return txid, nil

//以下是从BTCD的内存池返回的错误。
	case strings.Contains(err.Error(), "spent"):
		fallthrough
	case strings.Contains(err.Error(), "orphan"):
		fallthrough
	case strings.Contains(err.Error(), "conflict"):
		fallthrough

//以下错误从比特币的内存池返回。
	case strings.Contains(err.Error(), "fee not met"):
		fallthrough
	case strings.Contains(err.Error(), "Missing inputs"):
		fallthrough
	case strings.Contains(err.Error(), "already in block chain"):
//If the transaction was rejected, then we'll remove it from
//TxStore，否则，我们将继续尝试
//重新播放，钱包的utxo状态将不会
//准确。
		dbErr := walletdb.Update(w.db, func(dbTx walletdb.ReadWriteTx) error {
			txmgrNs := dbTx.ReadWriteBucket(wtxmgrNamespaceKey)
			return w.TxStore.RemoveUnminedTx(txmgrNs, txRec)
		})
		if dbErr != nil {
			return nil, fmt.Errorf("unable to broadcast tx: %v, "+
				"unable to remove invalid tx: %v", err, dbErr)
		}

		return nil, err

	default:
		return nil, err
	}
}

//chainParams返回区块链钱包的网络参数
//属于。
func (w *Wallet) ChainParams() *chaincfg.Params {
	return w.chainParams
}

//数据库返回基础的walletdb数据库。提供此方法
//in order to allow applications wrapping btcwallet to store app-specific data
//with the wallet's database.
func (w *Wallet) Database() walletdb.DB {
	return w.db
}

//创建创建一个新钱包，将其写入空数据库。如果通过
//种子是非零的，它被使用。否则，一个安全的随机种子
//生成建议长度。
func Create(db walletdb.DB, pubPass, privPass, seed []byte, params *chaincfg.Params,
	birthday time.Time) error {

//如果提供了种子，请确保其长度有效。否则，
//我们用推荐的种子为钱包生成随机种子
//长度。
	if seed == nil {
		hdSeed, err := hdkeychain.GenerateSeed(
			hdkeychain.RecommendedSeedLen)
		if err != nil {
			return err
		}
		seed = hdSeed
	}
	if len(seed) < hdkeychain.MinSeedBytes ||
		len(seed) > hdkeychain.MaxSeedBytes {
		return hdkeychain.ErrInvalidSeedLen
	}

	return walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs, err := tx.CreateTopLevelBucket(waddrmgrNamespaceKey)
		if err != nil {
			return err
		}
		txmgrNs, err := tx.CreateTopLevelBucket(wtxmgrNamespaceKey)
		if err != nil {
			return err
		}

		err = waddrmgr.Create(
			addrmgrNs, seed, pubPass, privPass, params, nil,
			birthday,
		)
		if err != nil {
			return err
		}
		return wtxmgr.Create(txmgrNs)
	})
}

//open从传递的数据库和名称空间加载已创建的钱包。
func Open(db walletdb.DB, pubPass []byte, cbs *waddrmgr.OpenCallbacks,
	params *chaincfg.Params, recoveryWindow uint32) (*Wallet, error) {

	var (
		addrMgr *waddrmgr.Manager
		txMgr   *wtxmgr.Store
	)

//在打开钱包之前，我们先检查一下有没有
//数据库升级供我们继续。我们还将创建引用
//地址和事务管理器，因为它们由
//数据库。
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		addrMgrBucket := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		if addrMgrBucket == nil {
			return errors.New("missing address manager namespace")
		}
		txMgrBucket := tx.ReadWriteBucket(wtxmgrNamespaceKey)
		if txMgrBucket == nil {
			return errors.New("missing transaction manager namespace")
		}

		addrMgrUpgrader := waddrmgr.NewMigrationManager(addrMgrBucket)
		txMgrUpgrader := wtxmgr.NewMigrationManager(txMgrBucket)
		err := migration.Upgrade(txMgrUpgrader, addrMgrUpgrader)
		if err != nil {
			return err
		}

		addrMgr, err = waddrmgr.Open(addrMgrBucket, pubPass, params)
		if err != nil {
			return err
		}
		txMgr, err = wtxmgr.Open(txMgrBucket, params)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

log.Infof("Opened wallet") //托多：原木平衡？上次同步高度？

	w := &Wallet{
		publicPassphrase:    pubPass,
		db:                  db,
		Manager:             addrMgr,
		TxStore:             txMgr,
		lockedOutpoints:     map[wire.OutPoint]struct{}{},
		recoveryWindow:      recoveryWindow,
		rescanAddJob:        make(chan *RescanJob),
		rescanBatch:         make(chan *rescanBatch),
		rescanNotifications: make(chan interface{}),
		rescanProgress:      make(chan *RescanProgressMsg),
		rescanFinished:      make(chan *RescanFinishedMsg),
		createTxRequests:    make(chan createTxRequest),
		unlockRequests:      make(chan unlockRequest),
		lockRequests:        make(chan struct{}),
		holdUnlockRequests:  make(chan chan heldUnlock),
		lockState:           make(chan bool),
		changePassphrase:    make(chan changePassphraseRequest),
		changePassphrases:   make(chan changePassphrasesRequest),
		chainParams:         params,
		quit:                make(chan struct{}),
	}

	w.NtfnServer = newNotificationServer(w)
	w.TxStore.NotifyUnspent = func(hash *chainhash.Hash, index uint32) {
		w.NtfnServer.notifyUnspentOutput(0, hash, index)
	}

	return w, nil
}
