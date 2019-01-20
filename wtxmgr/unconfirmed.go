
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//版权所有（c）2015-2016法令开发商
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wtxmgr

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/walletdb"
)

//insertmempooltx插入未限定的事务记录。它也标志着
//以前的输出被输入引用为已用。
func (s *Store) insertMemPoolTx(ns walletdb.ReadWriteBucket, rec *TxRecord) error {
//检查事务是否已添加到
//未确认的桶。
	if existsRawUnmined(ns, rec.Hash[:]) != nil {
//TODO:比较序列化的TXS以确保它不是哈希
//碰撞？
		return nil
	}

//因为存储中的事务记录是由它们的
//事务\和块确认，我们将迭代
//交易的输出，以确定我们是否已经看到它们
//防止将此事务添加到未确认的存储桶。
	for i := range rec.MsgTx.TxOut {
		k := canonicalOutPoint(&rec.Hash, uint32(i))
		if existsRawUnspent(ns, k) != nil {
			return nil
		}
	}

	log.Infof("Inserting unconfirmed transaction %v", rec.Hash)
	v, err := valueTxRecord(rec)
	if err != nil {
		return err
	}
	err = putRawUnmined(ns, rec.Hash[:], v)
	if err != nil {
		return err
	}

	for _, input := range rec.MsgTx.TxIn {
		prevOut := &input.PreviousOutPoint
		k := canonicalOutPoint(&prevOut.Hash, prevOut.Index)
		err = putRawUnminedInput(ns, k, rec.Hash[:])
		if err != nil {
			return err
		}
	}

//TODO:为每个信用证增加信用证金额（但这些金额未知
//目前在这里）。

	return nil
}

//可删除的问题挂起检查将导致
//如果将Tx添加到商店（无论是确认的还是未确认的），则为双倍消费
//事务处理）。每个冲突的事务和所有花费
//它是递归删除的。
func (s *Store) removeDoubleSpends(ns walletdb.ReadWriteBucket, rec *TxRecord) error {
	for _, input := range rec.MsgTx.TxIn {
		prevOut := &input.PreviousOutPoint
		prevOutKey := canonicalOutPoint(&prevOut.Hash, prevOut.Index)

		doubleSpendHashes := fetchUnminedInputSpendTxHashes(ns, prevOutKey)
		for _, doubleSpendHash := range doubleSpendHashes {
			doubleSpendVal := existsRawUnmined(ns, doubleSpendHash[:])

//如果支出交易花费多个输出
//在同一笔交易中，我们会发现重复的
//商店内的条目，因此我们可能
//如果冲突已经发生，则无法找到它。
//在上一次迭代中删除。
			if doubleSpendVal == nil {
				continue
			}

			var doubleSpend TxRecord
			doubleSpend.Hash = doubleSpendHash
			err := readRawTxRecord(
				&doubleSpend.Hash, doubleSpendVal, &doubleSpend,
			)
			if err != nil {
				return err
			}

			log.Debugf("Removing double spending transaction %v",
				doubleSpend.Hash)
			if err := s.removeConflict(ns, &doubleSpend); err != nil {
				return err
			}
		}
	}

	return nil
}

//removeconflict删除未链接的事务记录和所有支出链
//从商店衍生而来。这是为了删除事务
//否则，如果留在商店，将导致双倍消费冲突，
//并删除在REORG上花费coinbase事务的事务。
func (s *Store) removeConflict(ns walletdb.ReadWriteBucket, rec *TxRecord) error {
//对于本记录的每个潜在信用，每个消费者（如果有）必须
//也可以递归删除。一旦消费者被移除，
//credit is deleted.
	for i := range rec.MsgTx.TxOut {
		k := canonicalOutPoint(&rec.Hash, uint32(i))
		spenderHashes := fetchUnminedInputSpendTxHashes(ns, k)
		for _, spenderHash := range spenderHashes {
			spenderVal := existsRawUnmined(ns, spenderHash[:])

//如果支出交易花费多个输出
//在同一笔交易中，我们会发现重复的
//商店内的条目，因此我们可能
//如果冲突已经发生，则无法找到它。
//在上一次迭代中删除。
			if spenderVal == nil {
				continue
			}

			var spender TxRecord
			spender.Hash = spenderHash
			err := readRawTxRecord(&spender.Hash, spenderVal, &spender)
			if err != nil {
				return err
			}

			log.Debugf("Transaction %v is part of a removed conflict "+
				"chain -- removing as well", spender.Hash)
			if err := s.removeConflict(ns, &spender); err != nil {
				return err
			}
		}
		if err := deleteRawUnminedCredit(ns, k); err != nil {
			return err
		}
	}

//如果此Tx使用任何以前的信用证（开采或未开采），则设置
//每个未花。挖掘的事务仅通过
//output in the unmined inputs bucket.
	for _, input := range rec.MsgTx.TxIn {
		prevOut := &input.PreviousOutPoint
		k := canonicalOutPoint(&prevOut.Hash, prevOut.Index)
		if err := deleteRawUnminedInput(ns, k); err != nil {
			return err
		}
	}

	return deleteRawUnmined(ns, rec.Hash[:])
}

//unlinedtx返回所有未链接事务的基础事务
//目前还不知道是在一个街区内开采的。交易是
//guaranteed to be sorted by their dependency order.
func (s *Store) UnminedTxs(ns walletdb.ReadBucket) ([]*wire.MsgTx, error) {
	recSet, err := s.unminedTxRecords(ns)
	if err != nil {
		return nil, err
	}

	recs := dependencySort(recSet)
	txs := make([]*wire.MsgTx, 0, len(recs))
	for _, rec := range recs {
		txs = append(txs, &rec.MsgTx)
	}
	return txs, nil
}

func (s *Store) unminedTxRecords(ns walletdb.ReadBucket) (map[chainhash.Hash]*TxRecord, error) {
	unmined := make(map[chainhash.Hash]*TxRecord)
	err := ns.NestedReadBucket(bucketUnmined).ForEach(func(k, v []byte) error {
		var txHash chainhash.Hash
		err := readRawUnminedHash(k, &txHash)
		if err != nil {
			return err
		}

		rec := new(TxRecord)
		err = readRawTxRecord(&txHash, v, rec)
		if err != nil {
			return err
		}
		unmined[rec.Hash] = rec
		return nil
	})
	return unmined, err
}

//unlinedXhash返回未知的所有事务的哈希值。
//分块开采。
func (s *Store) UnminedTxHashes(ns walletdb.ReadBucket) ([]*chainhash.Hash, error) {
	return s.unminedTxHashes(ns)
}

func (s *Store) unminedTxHashes(ns walletdb.ReadBucket) ([]*chainhash.Hash, error) {
	var hashes []*chainhash.Hash
	err := ns.NestedReadBucket(bucketUnmined).ForEach(func(k, v []byte) error {
		hash := new(chainhash.Hash)
		err := readRawUnminedHash(k, hash)
		if err == nil {
			hashes = append(hashes, hash)
		}
		return err
	})
	return hashes, err
}
