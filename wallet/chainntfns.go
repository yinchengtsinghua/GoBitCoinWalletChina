
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wallet

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/chain"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/btcsuite/btcwallet/wtxmgr"
)

const (
//BirthDayBlockDelta是我们
//搜索时的生日时间戳和生日块的时间戳
//
	birthdayBlockDelta = 2 * time.Hour
)

func (w *Wallet) handleChainNotifications() {
	defer w.wg.Done()

	chainClient, err := w.requireChainClient()
	if err != nil {
		log.Errorf("handleChainNotifications called without RPC client")
		return
	}

	sync := func(w *Wallet, birthdayStamp *waddrmgr.BlockStamp) {
//如果重新扫描失败，目前没有追索权。
//但是，由于某些原因，钱包将不会标记为“同步”
//很多方法都会很早就出错，因为钱包是已知的
//过时了。
		err := w.syncWithChain(birthdayStamp)
		if err != nil && !w.ShuttingDown() {
			log.Warnf("Unable to synchronize wallet to chain: %v", err)
		}
	}

	catchUpHashes := func(w *Wallet, client chain.Interface,
		height int32) error {
//托多（阿克塞尔罗德）：这里有一个比赛条件，它
//当REORG在
//重新扫描进度通知和最后一个GetBlockHash
//打电话。使用BTCD时的解决方案是制作BTCD
//向每个块发送blockconnected通知
//
//另一种方法是检查最后的哈希值，
//如果它与返回的原始哈希不匹配
//
//再扫描。
		log.Infof("Catching up block hashes to height %d, this"+
			" might take a while", height)
		err := walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) error {
			ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)

			startBlock := w.Manager.SyncedTo()

			for i := startBlock.Height + 1; i <= height; i++ {
				hash, err := client.GetBlockHash(int64(i))
				if err != nil {
					return err
				}
				header, err := chainClient.GetBlockHeader(hash)
				if err != nil {
					return err
				}

				bs := waddrmgr.BlockStamp{
					Height:    i,
					Hash:      *hash,
					Timestamp: header.Timestamp,
				}
				err = w.Manager.SetSyncedTo(ns, &bs)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Errorf("Failed to update address manager "+
				"sync state for height %d: %v", height, err)
		}

		log.Info("Done catching up block hashes")
		return err
	}

	for {
		select {
		case n, ok := <-chainClient.Notifications():
			if !ok {
				return
			}

			var notificationName string
			var err error
			switch n := n.(type) {
			case chain.ClientConnected:
//在尝试与后端同步之前，
//我们会确保我们的生日街区
//已正确设置为可能阻止
//缺少相关事件。
				birthdayStore := &walletBirthdayStore{
					db:      w.db,
					manager: w.Manager,
				}
				birthdayBlock, err := birthdaySanityCheck(
					chainClient, birthdayStore,
				)
				if err != nil && !waddrmgr.IsError(err, waddrmgr.ErrBirthdayBlockNotSet) {
					err := fmt.Errorf("unable to sanity "+
						"check wallet birthday block: %v",
						err)
					log.Error(err)
					panic(err)
				}

				go sync(w, birthdayBlock)
			case chain.BlockConnected:
				err = walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) error {
					return w.connectBlock(tx, wtxmgr.BlockMeta(n))
				})
				notificationName = "blockconnected"
			case chain.BlockDisconnected:
				err = walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) error {
					return w.disconnectBlock(tx, wtxmgr.BlockMeta(n))
				})
				notificationName = "blockdisconnected"
			case chain.RelevantTx:
				err = walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) error {
					return w.addRelevantTx(tx, n.TxRecord, n.Block)
				})
				notificationName = "recvtx/redeemingtx"
			case chain.FilteredBlockConnected:
//对整个块进行原子更新。
				if len(n.RelevantTxs) > 0 {
					err = walletdb.Update(w.db, func(
						tx walletdb.ReadWriteTx) error {
						var err error
						for _, rec := range n.RelevantTxs {
							err = w.addRelevantTx(tx, rec,
								n.Block)
							if err != nil {
								return err
							}
						}
						return nil
					})
				}
				notificationName = "filteredblockconnected"

//下面需要一些数据库维护，但是
//需要向钱包的Rescan Goroutine报告。
			case *chain.RescanProgress:
				err = catchUpHashes(w, chainClient, n.Height)
				notificationName = "rescanprogress"
				select {
				case w.rescanNotifications <- n:
				case <-w.quitChan():
					return
				}
			case *chain.RescanFinished:
				err = catchUpHashes(w, chainClient, n.Height)
				notificationName = "rescanprogress"
				w.SetChainSynced(true)
				select {
				case w.rescanNotifications <- n:
				case <-w.quitChan():
					return
				}
			}
			if err != nil {
//仅限于不同步的块连接通知
//发送调试消息。
				errStr := "Failed to process consensus server " +
					"notification (name: `%s`, detail: `%v`)"
				if notificationName == "blockconnected" &&
					strings.Contains(err.Error(),
						"couldn't get hash from database") {
					log.Debugf(errStr, notificationName, err)
				} else {
					log.Errorf(errStr, notificationName, err)
				}
			}
		case <-w.quit:
			return
		}
	}
}

//ConnectBlock通过标记钱包处理链服务器通知
//当前与正在同步的链服务器同步
//经过的街区。
func (w *Wallet) connectBlock(dbtx walletdb.ReadWriteTx, b wtxmgr.BlockMeta) error {
	addrmgrNs := dbtx.ReadWriteBucket(waddrmgrNamespaceKey)

	bs := waddrmgr.BlockStamp{
		Height:    b.Height,
		Hash:      b.Hash,
		Timestamp: b.Time,
	}
	err := w.Manager.SetSyncedTo(addrmgrNs, &bs)
	if err != nil {
		return err
	}

//将所连接的块通知相关客户端。
//
//TODO:将所有通知移出数据库事务。
	w.NtfnServer.notifyAttachedBlock(dbtx, &b)
	return nil
}

//disconnectBlock通过回滚所有
//阻止钱包重新排列块中与链同步的历史记录
//服务器。
func (w *Wallet) disconnectBlock(dbtx walletdb.ReadWriteTx, b wtxmgr.BlockMeta) error {
	addrmgrNs := dbtx.ReadWriteBucket(waddrmgrNamespaceKey)
	txmgrNs := dbtx.ReadWriteBucket(wtxmgrNamespaceKey)

	if !w.ChainSynced() {
		return nil
	}

//如果我们知道，断开拆下的块和其后的所有块
//断开的块。否则，这个街区就在未来。
	if b.Height <= w.Manager.SyncedTo().Height {
		hash, err := w.Manager.BlockHash(addrmgrNs, b.Height)
		if err != nil {
			return err
		}
		if bytes.Equal(hash[:], b.Hash[:]) {
			bs := waddrmgr.BlockStamp{
				Height: b.Height - 1,
			}
			hash, err = w.Manager.BlockHash(addrmgrNs, bs.Height)
			if err != nil {
				return err
			}
			b.Hash = *hash

			client := w.ChainClient()
			header, err := client.GetBlockHeader(hash)
			if err != nil {
				return err
			}

			bs.Timestamp = header.Timestamp
			err = w.Manager.SetSyncedTo(addrmgrNs, &bs)
			if err != nil {
				return err
			}

			err = w.TxStore.Rollback(txmgrNs, b.Height)
			if err != nil {
				return err
			}
		}
	}

//将断开连接的块通知相关客户端。
	w.NtfnServer.notifyDetachedBlock(&b.Hash)

	return nil
}

func (w *Wallet) addRelevantTx(dbtx walletdb.ReadWriteTx, rec *wtxmgr.TxRecord, block *wtxmgr.BlockMeta) error {
	addrmgrNs := dbtx.ReadWriteBucket(waddrmgrNamespaceKey)
	txmgrNs := dbtx.ReadWriteBucket(wtxmgrNamespaceKey)

//目前，所有已通知的交易均假定为
//相关的。当SPV支持为
//添加了，但在此之前，只需插入事务，因为
//应该是一个或多个相关的输入或输出。
	err := w.TxStore.InsertTx(txmgrNs, rec, block)
	if err != nil {
		return err
	}

//检查每个输出以确定是否由钱包控制
//关键。如果是，将输出标记为信用。
	for i, output := range rec.MsgTx.TxOut {
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(output.PkScript,
			w.chainParams)
		if err != nil {
//跳过非标准输出。
			continue
		}
		for _, addr := range addrs {
			ma, err := w.Manager.Address(addrmgrNs, addr)
			if err == nil {
//TODO:应将学分与
//他们所属的帐户，因此wtxmgr能够
//跟踪每个帐户的余额。
				err = w.TxStore.AddCredit(txmgrNs, rec, block, uint32(i),
					ma.Internal())
				if err != nil {
					return err
				}
				err = w.Manager.MarkUsed(addrmgrNs, addr)
				if err != nil {
					return err
				}
				log.Debugf("Marked address %v used", addr)
				continue
			}

//跳过丢失的地址。其他错误应该
//传播。
			if !waddrmgr.IsError(err, waddrmgr.ErrAddressNotFound) {
				return err
			}
		}
	}

//向任何有兴趣的人发送已开采或未开采交易的通知
//客户。
//
//TODO:避免额外的数据库命中。
	if block == nil {
		details, err := w.TxStore.UniqueTxDetails(txmgrNs, &rec.Hash, nil)
		if err != nil {
			log.Errorf("Cannot query transaction details for notification: %v", err)
		}

//可能在
//钱包的一组未确认的交易已经到期
//确认后，我们将避免通知。
//
//托多（威尔默）：理想情况下，我们应该找出我们为什么
//接收另一个未确认的链。相关的ttx
//来自链后端的通知。
		if details != nil {
			w.NtfnServer.notifyUnminedTransaction(dbtx, details)
		}
	} else {
		details, err := w.TxStore.UniqueTxDetails(txmgrNs, &rec.Hash, &block.Block)
		if err != nil {
			log.Errorf("Cannot query transaction details for notification: %v", err)
		}

//只有在
//钱包的已确认交易集。
		if details != nil {
			w.NtfnServer.notifyMinedTransaction(dbtx, details, block)
		}
	}

	return nil
}

//chainconn是一个抽象所需链连接逻辑的接口。
//执行钱包的生日检查。
type chainConn interface {
//GetBestBlock返回已知最佳块的哈希和高度
//后端。
	GetBestBlock() (*chainhash.Hash, int32, error)

//GetBlockHash返回具有给定高度的块的哈希。
	GetBlockHash(int64) (*chainhash.Hash, error)

//GetBlockHeader返回具有给定哈希的块的头。
	GetBlockHeader(*chainhash.Hash) (*wire.BlockHeader, error)
}

//
//执行生日块健全性检查所需的信息。
type birthdayStore interface {
//生日返回钱包的生日时间戳。
	Birthday() time.Time

//BirthDayBlock返回钱包的生日块。布尔函数
//返回时应显示钱包是否已验证
//它的生日块是正确的。
	BirthdayBlock() (waddrmgr.BlockStamp, bool, error)

//setbirthdayblock将钱包的生日块更新为
//给定块。布尔值可用于指示此块是否
//下次钱包启动时应该检查一下是否正常。
//
//
//生日街区。这将允许钱包从此处重新扫描
//检测任何可能错过的事件。
	SetBirthdayBlock(waddrmgr.BlockStamp) error
}

//WalletBirthDayStore是钱包数据库和地址的包装。
//满足生日存储接口的管理器。
type walletBirthdayStore struct {
	db      walletdb.DB
	manager *waddrmgr.Manager
}

var _ birthdayStore = (*walletBirthdayStore)(nil)

//生日返回钱包的生日时间戳。
func (s *walletBirthdayStore) Birthday() time.Time {
	return s.manager.Birthday()
}

//BirthDayBlock返回钱包的生日块。
func (s *walletBirthdayStore) BirthdayBlock() (waddrmgr.BlockStamp, bool, error) {
	var (
		birthdayBlock         waddrmgr.BlockStamp
		birthdayBlockVerified bool
	)

	err := walletdb.View(s.db, func(tx walletdb.ReadTx) error {
		var err error
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		birthdayBlock, birthdayBlockVerified, err = s.manager.BirthdayBlock(ns)
		return err
	})

	return birthdayBlock, birthdayBlockVerified, err
}

//setbirthdayblock将钱包的生日块更新为
//给定块。布尔值可用于指示此块是否
//下次钱包启动时应该检查一下是否正常。
//
//注意：这还应设置钱包的同步提示以反映新的
//生日街区。这将允许钱包从此处重新扫描
//检测任何可能错过的事件。
func (s *walletBirthdayStore) SetBirthdayBlock(block waddrmgr.BlockStamp) error {
	return walletdb.Update(s.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		err := s.manager.SetBirthdayBlock(ns, block, true)
		if err != nil {
			return err
		}
		return s.manager.SetSyncedTo(ns, &block)
	})
}

//birthdayssanitycheck是一个助手函数，用于确保生日块
//在合理的时间戳内正确反映生日时间戳
//三角洲。它是在钱包建立连接后运行的
//与后端，但在开始同步之前。这是第二次
//钱包地址管理器迁移的一部分，我们在那里填充生日
//阻止以确保在整个重新扫描过程中不会错过任何相关事件。
//
//已经成立了。
func birthdaySanityCheck(chainConn chainConn,
	birthdayStore birthdayStore) (*waddrmgr.BlockStamp, error) {

//
	birthdayTimestamp := birthdayStore.Birthday()
	birthdayBlock, birthdayBlockVerified, err := birthdayStore.BirthdayBlock()
	if err != nil {
		return nil, err
	}

//如果生日卡已经被确认是正确的，我们可以
//退出我们的健全检查，以防止潜在的获取更好的
//候选者。
	if birthdayBlockVerified {
		log.Debugf("Birthday block has already been verified: "+
			"height=%d, hash=%v", birthdayBlock.Height,
			birthdayBlock.Hash)

		return &birthdayBlock, nil
	}

	log.Debugf("Starting sanity check for the wallet's birthday block "+
		"from: height=%d, hash=%v", birthdayBlock.Height,
		birthdayBlock.Hash)

//现在，我们需要确定我们的块是否正确反映了
//时间戳。为此，我们将获取块头并检查其
//如果生日块的时间戳不是
//设置（如果是通过迁移设置的，这是可能的，因为我们
//不要存储块时间戳）。
	candidate := birthdayBlock
	header, err := chainConn.GetBlockHeader(&candidate.Hash)
	if err != nil {
		return nil, fmt.Errorf("unable to get header for block hash "+
			"%v: %v", candidate.Hash, err)
	}
	candidate.Timestamp = header.Timestamp

//
//阻止其时间戳低于我们的生日时间戳。
	heightDelta := int32(144)
	for birthdayTimestamp.Before(candidate.Timestamp) {
//如果生日街区到了创世纪，我们就可以离开了
//我们的搜索，因为在此之前没有数据。
		if candidate.Height == 0 {
			break
		}

//为了防止请求块超出范围，我们将使用
//链中第一个块的绑定。
		newCandidateHeight := int64(candidate.Height - heightDelta)
		if newCandidateHeight < 0 {
			newCandidateHeight = 0
		}

//然后，我们将获取当前候选人的哈希和头
//
		hash, err := chainConn.GetBlockHash(newCandidateHeight)
		if err != nil {
			return nil, fmt.Errorf("unable to get block hash for "+
				"height %d: %v", candidate.Height, err)
		}
		header, err := chainConn.GetBlockHeader(hash)
		if err != nil {
			return nil, fmt.Errorf("unable to get header for "+
				"block hash %v: %v", candidate.Hash, err)
		}

		candidate.Hash = *hash
		candidate.Height = int32(newCandidateHeight)
		candidate.Timestamp = header.Timestamp

		log.Debugf("Checking next birthday block candidate: "+
			"height=%d, hash=%v, timestamp=%v",
			candidate.Height, candidate.Hash,
			candidate.Timestamp)
	}

//
//尊重我们的生日时间戳，它在合理的增量内。
//生日已经调整为过去两天
//实际生日，所以我们将使我们预期的delta在两个以内
//计算网络调整时间和预防
//获取更多不必要的块。
	_, bestHeight, err := chainConn.GetBestBlock()
	if err != nil {
		return nil, err
	}
	timestampDelta := birthdayTimestamp.Sub(candidate.Timestamp)
	for timestampDelta > birthdayBlockDelta {
//我们将确定下一位候选人的身高，并
//当然不超出范围。如果是的话，我们会降低高度
//直到在范围内找到一个高度。
		newHeight := candidate.Height + heightDelta
		if newHeight > bestHeight {
			heightDelta /= 2

//
//再高一点，我们就可以假设现在的生日了
//布洛克是我们最好的估计。
			if heightDelta == 0 {
				break
			}

			continue
		}

//我们将获取下一个候选人的标题并比较
//时间戳。
		hash, err := chainConn.GetBlockHash(int64(newHeight))
		if err != nil {
			return nil, fmt.Errorf("unable to get block hash for "+
				"height %d: %v", candidate.Height, err)
		}
		header, err := chainConn.GetBlockHeader(hash)
		if err != nil {
			return nil, fmt.Errorf("unable to get header for "+
				"block hash %v: %v", hash, err)
		}

		log.Debugf("Checking next birthday block candidate: "+
			"height=%d, hash=%v, timestamp=%v", newHeight, hash,
			header.Timestamp)

//如果此块超过了我们的生日时间戳，我们将查找
//对于下一个具有较低高度增量的候选人。
		if birthdayTimestamp.Before(header.Timestamp) {
			heightDelta /= 2

//如果我们在
//再高一点，我们就可以假设现在的生日了
//布洛克是我们最好的估计。
			if heightDelta == 0 {
				break
			}

			continue
		}

//
//如果它符合我们期望的时间戳delta。
		candidate.Hash = *hash
		candidate.Height = newHeight
		candidate.Timestamp = header.Timestamp
		timestampDelta = birthdayTimestamp.Sub(header.Timestamp)
	}

//现在，我们找到了一个更好的新候选人，所以我们会写下来
//到磁盘。
	log.Debugf("Found a new valid wallet birthday block: height=%d, hash=%v",
		candidate.Height, candidate.Hash)

	if err := birthdayStore.SetBirthdayBlock(candidate); err != nil {
		return nil, err
	}

	return &candidate, nil
}
