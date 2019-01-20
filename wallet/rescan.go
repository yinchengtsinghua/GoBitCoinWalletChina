
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wallet

import (
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/chain"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/wtxmgr"
)

//rescanprogressmsg报告重新扫描的当前进度
//钱包地址集。
type RescanProgressMsg struct {
	Addresses    []btcutil.Address
	Notification *chain.RescanProgress
}

//RescanFinishedMSG报告当
//收到了重新扫描完成消息，正在重新扫描一批地址。
type RescanFinishedMsg struct {
	Addresses    []btcutil.Address
	Notification *chain.RescanFinished
}

//RescanJob是要由RescanManager处理的作业。这项工作包括
//一组钱包地址，开始重新扫描的起始高度，以及
//输出被认为未使用的地址所消耗的点数。后
//重新扫描完成，错误上发送重新扫描RPC的错误结果。
//通道。
type RescanJob struct {
	InitialSync bool
	Addrs       []btcutil.Address
	OutPoints   map[wire.OutPoint]btcutil.Address
	BlockStamp  waddrmgr.BlockStamp
	err         chan error
}

//Rescanbatch是合并的一个或多个RescanJobs的集合
//一起重新扫描。
type rescanBatch struct {
	initialSync bool
	addrs       []btcutil.Address
	outpoints   map[wire.OutPoint]btcutil.Address
	bs          waddrmgr.BlockStamp
	errChans    []chan error
}

//SubmitRescan向RescanManager提交RescanJob。通道是
//返回了最后一个重新扫描错误。通道被缓冲
//而且不需要读取来防止死锁。
func (w *Wallet) SubmitRescan(job *RescanJob) <-chan error {
	errChan := make(chan error, 1)
	job.err = errChan
	w.rescanAddJob <- job
	return errChan
}

//批处理为单个重新扫描作业创建重新扫描批处理。
func (job *RescanJob) batch() *rescanBatch {
	return &rescanBatch{
		initialSync: job.InitialSync,
		addrs:       job.Addrs,
		outpoints:   job.OutPoints,
		bs:          job.BlockStamp,
		errChans:    []chan error{job.err},
	}
}

//合并将k的工作合并为j，将起始高度设置为
//
//重复地址或输出点。
func (b *rescanBatch) merge(job *RescanJob) {
	if job.InitialSync {
		b.initialSync = true
	}
	b.addrs = append(b.addrs, job.Addrs...)

	for op, addr := range job.OutPoints {
		b.outpoints[op] = addr
	}

	if job.BlockStamp.Height < b.bs.Height {
		b.bs = job.BlockStamp
	}
	b.errChans = append(b.errChans, job.err)
}

//完成迭代所有错误通道，复制发送错误
//通知呼叫者重新扫描已完成（或到期无法完成）
//一个错误）。
func (b *rescanBatch) done(err error) {
	for _, c := range b.errChans {
		c <- err
	}
}

//RescanBatchHandler处理传入的重新扫描请求，序列化重新扫描
//提交，并可能将许多等待的请求集中在一起，以便
//可以在当前重新扫描完成后通过单个重新扫描进行处理。
func (w *Wallet) rescanBatchHandler() {
	var curBatch, nextBatch *rescanBatch
	quit := w.quitChan()

out:
	for {
		select {
		case job := <-w.rescanAddJob:
			if curBatch == nil {
//
//请求。
				curBatch = job.batch()
				w.rescanBatch <- curBatch
			} else {
//如果不存在，则创建下一批，或者
//合并作业。
				if nextBatch == nil {
					nextBatch = job.batch()
				} else {
					nextBatch.merge(job)
				}
			}

		case n := <-w.rescanNotifications:
			switch n := n.(type) {
			case *chain.RescanProgress:
				if curBatch == nil {
					log.Warnf("Received rescan progress " +
						"notification but no rescan " +
						"currently running")
					continue
				}
				w.rescanProgress <- &RescanProgressMsg{
					Addresses:    curBatch.addrs,
					Notification: n,
				}

			case *chain.RescanFinished:
				if curBatch == nil {
					log.Warnf("Received rescan finished " +
						"notification but no rescan " +
						"currently running")
					continue
				}
				w.rescanFinished <- &RescanFinishedMsg{
					Addresses:    curBatch.addrs,
					Notification: n,
				}

				curBatch, nextBatch = nextBatch, nil

				if curBatch != nil {
					w.rescanBatch <- curBatch
				}

			default:
//意外消息
				panic(n)
			}

		case <-quit:
			break out
		}
	}

	w.wg.Done()
}

//RescanProgressHandler处理部分和完全完成的通知
//rescans by marking each rescanned address as partially or fully synced.
func (w *Wallet) rescanProgressHandler() {
	quit := w.quitChan()
out:
	for {
//因为这两个通道都是
//未缓冲并从同一上下文（批处理）发送
//处理程序）。
		select {
		case msg := <-w.rescanProgress:
			n := msg.Notification
			log.Infof("Rescanned through block %v (height %d)",
				n.Hash, n.Height)

		case msg := <-w.rescanFinished:
			n := msg.Notification
			addrs := msg.Addresses
			noun := pickNoun(len(addrs), "address", "addresses")
			log.Infof("Finished rescan for %d %s (synced to block "+
				"%s, height %d)", len(addrs), noun, n.Hash,
				n.Height)

			go w.resendUnminedTxs()

		case <-quit:
			break out
		}
	}
	w.wg.Done()
}

//rescanrpchandler读取rescanbatchhandler发送的批处理作业并发送
//RPC请求执行重新扫描。只有重新扫描后才能读取新作业
//完成。
func (w *Wallet) rescanRPCHandler() {
	chainClient, err := w.requireChainClient()
	if err != nil {
		log.Errorf("rescanRPCHandler called without an RPC client")
		w.wg.Done()
		return
	}

	quit := w.quitChan()

out:
	for {
		select {
		case batch := <-w.rescanBatch:
//记录新开始的重新扫描。
			numAddrs := len(batch.addrs)
			noun := pickNoun(numAddrs, "address", "addresses")
			log.Infof("Started rescan from block %v (height %d) for %d %s",
				batch.bs.Hash, batch.bs.Height, numAddrs, noun)

			err := chainClient.Rescan(&batch.bs.Hash, batch.addrs,
				batch.outpoints)
			if err != nil {
				log.Errorf("Rescan for %d %s failed: %v", numAddrs,
					noun, err)
			}
			batch.done(err)
		case <-quit:
			break out
		}
	}

	w.wg.Done()
}

//重新扫描开始重新扫描的所有活动地址和未暂停的输出
//钱包它用于将钱包同步回
//当前主链中的最佳块，并被视为初始同步
//再扫描。
func (w *Wallet) Rescan(addrs []btcutil.Address, unspent []wtxmgr.Credit) error {
	return w.rescanWithTarget(addrs, unspent, nil)
}

//RescanWithTarget从可选的StartStamp开始执行重新扫描。如果
//没有提供，重新扫描将从管理器的同步提示开始。
func (w *Wallet) rescanWithTarget(addrs []btcutil.Address,
	unspent []wtxmgr.Credit, startStamp *waddrmgr.BlockStamp) error {

	outpoints := make(map[wire.OutPoint]btcutil.Address, len(unspent))
	for _, output := range unspent {
		_, outputAddrs, _, err := txscript.ExtractPkScriptAddrs(
			output.PkScript, w.chainParams,
		)
		if err != nil {
			return err
		}

		outpoints[output.OutPoint] = outputAddrs[0]
	}

//
//重新扫描的起始点。
	if startStamp == nil {
		startStamp = &waddrmgr.BlockStamp{}
		*startStamp = w.Manager.SyncedTo()
	}

	job := &RescanJob{
		InitialSync: true,
		Addrs:       addrs,
		OutPoints:   outpoints,
		BlockStamp:  *startStamp,
	}

//提交合并的作业并阻止，直到重新扫描完成。
	return <-w.SubmitRescan(job)
}
