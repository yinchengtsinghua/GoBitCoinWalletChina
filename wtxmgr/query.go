
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2017 BTCSuite开发者
//版权所有（c）2015-2016法令开发商
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wtxmgr

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/walletdb"
)

//CreditRecord包含有关已知
//交易。可以通过索引wire.msgtx.txout来查找更多详细信息。
//使用索引字段。
type CreditRecord struct {
	Amount btcutil.Amount
	Index  uint32
	Spent  bool
	Change bool
}

//Debitrecord包含有关已知的交易借记的元数据
//交易。可以通过索引Wire.msgtx.txin来查找更多详细信息。
//使用索引字段。
type DebitRecord struct {
	Amount btcutil.Amount
	Index  uint32
}

//txtdetails旨在为呼叫者提供访问丰富详细信息的权限。
//关于相关交易，哪些输入和输出是信贷或
//借方。
type TxDetails struct {
	TxRecord
	Block   BlockMeta
	Credits []CreditRecord
	Debits  []DebitRecord
}

//MinedTxDetails使用哈希获取挖掘事务的TxDetails
//TxHash和传递的Tx记录键和值。
func (s *Store) minedTxDetails(ns walletdb.ReadBucket, txHash *chainhash.Hash, recKey, recVal []byte) (*TxDetails, error) {
	var details TxDetails

//分析事务记录k/v，查找
//阻塞时间，读取所有匹配的贷记、借记。
	err := readRawTxRecord(txHash, recVal, &details.TxRecord)
	if err != nil {
		return nil, err
	}
	err = readRawTxRecordBlock(recKey, &details.Block.Block)
	if err != nil {
		return nil, err
	}
	details.Block.Time, err = fetchBlockTime(ns, details.Block.Height)
	if err != nil {
		return nil, err
	}

	credIter := makeReadCreditIterator(ns, recKey)
	for credIter.next() {
		if int(credIter.elem.Index) >= len(details.MsgTx.TxOut) {
			str := "saved credit index exceeds number of outputs"
			return nil, storeError(ErrData, str, nil)
		}

//信贷迭代器不记录此信贷是否
//被一个未链接的事务所花费，所以在这里检查一下。
		if !credIter.elem.Spent {
			k := canonicalOutPoint(txHash, credIter.elem.Index)
			spent := existsRawUnminedInput(ns, k) != nil
			credIter.elem.Spent = spent
		}
		details.Credits = append(details.Credits, credIter.elem)
	}
	if credIter.err != nil {
		return nil, credIter.err
	}

	debIter := makeReadDebitIterator(ns, recKey)
	for debIter.next() {
		if int(debIter.elem.Index) >= len(details.MsgTx.TxIn) {
			str := "saved debit index exceeds number of inputs"
			return nil, storeError(ErrData, str, nil)
		}

		details.Debits = append(details.Debits, debIter.elem)
	}
	return &details, debIter.err
}

//unlinedxdetails获取与
//哈希txshash和传递的未限定记录值。
func (s *Store) unminedTxDetails(ns walletdb.ReadBucket, txHash *chainhash.Hash, v []byte) (*TxDetails, error) {
	details := TxDetails{
		Block: BlockMeta{Block: Block{Height: -1}},
	}
	err := readRawTxRecord(txHash, v, &details.TxRecord)
	if err != nil {
		return nil, err
	}

	it := makeReadUnminedCreditIterator(ns, txHash)
	for it.next() {
		if int(it.elem.Index) >= len(details.MsgTx.TxOut) {
			str := "saved credit index exceeds number of outputs"
			return nil, storeError(ErrData, str, nil)
		}

//由于迭代器不执行此操作，因此请设置已用字段。
		it.elem.Spent = existsRawUnminedInput(ns, it.ck) != nil
		details.Credits = append(details.Credits, it.elem)
	}
	if it.err != nil {
		return nil, it.err
	}

//未关联交易记录不保存借方记录。相反，他们
//必须手动查找每个事务输入。有两个
//以前的信用证的种类，可由未经列账的
//事务：挖掘的未使用的输出（仍标记为未使用的偶数
//当由非关联交易使用时），以及来自其他非关联交易的贷项
//交易。必须同时考虑这两种情况。
	for i, output := range details.MsgTx.TxIn {
		opKey := canonicalOutPoint(&output.PreviousOutPoint.Hash,
			output.PreviousOutPoint.Index)
		credKey := existsRawUnspent(ns, opKey)
		if credKey != nil {
			v := existsRawCredit(ns, credKey)
			amount, err := fetchRawCreditAmount(v)
			if err != nil {
				return nil, err
			}

			details.Debits = append(details.Debits, DebitRecord{
				Amount: amount,
				Index:  uint32(i),
			})
			continue
		}

		v := existsRawUnminedCredit(ns, opKey)
		if v == nil {
			continue
		}

		amount, err := fetchRawCreditAmount(v)
		if err != nil {
			return nil, err
		}
		details.Debits = append(details.Debits, DebitRecord{
			Amount: amount,
			Index:  uint32(i),
		})
	}

	return &details, nil
}

//txtdetails查找与某个交易有关的所有记录的详细信息
//搞砸。在哈希冲突的情况下，最新的事务
//返回匹配的哈希。
//
//找不到具有此哈希的事务不是错误。在这种情况下，
//返回nil txtdetails。
func (s *Store) TxDetails(ns walletdb.ReadBucket, txHash *chainhash.Hash) (*TxDetails, error) {
//首先，检查是否存在与此
//搞砸。如果找到就用它。
	v := existsRawUnmined(ns, txHash[:])
	if v != nil {
		return s.unminedTxDetails(ns, txHash, v)
	}

//否则，如果存在与此匹配的挖掘事务
//散列，跳到最新的并开始获取所有详细信息。
	k, v := latestTxRecord(ns, txHash)
	if v == nil {
//找不到
		return nil, nil
	}
	return s.minedTxDetails(ns, txHash, k, v)
}

//uniquetxdetails查找记录的交易的所有记录的详细信息
//在某个特定的块中挖掘，或者如果块为零，则为未链接的事务。
//
//从该块中找不到具有此哈希的事务不是错误。在
//这种情况下，返回零txtdetails。
func (s *Store) UniqueTxDetails(ns walletdb.ReadBucket, txHash *chainhash.Hash,
	block *Block) (*TxDetails, error) {

	if block == nil {
		v := existsRawUnmined(ns, txHash[:])
		if v == nil {
			return nil, nil
		}
		return s.unminedTxDetails(ns, txHash, v)
	}

	k, v := existsTxRecord(ns, txHash, block)
	if v == nil {
		return nil, nil
	}
	return s.minedTxDetails(ns, txHash, k, v)
}

//RangeUnlinedTransactions为每个
//未开采的交易。如果没有未被挖掘的事务存在，则不执行F。
//从F返回的错误（如果有）会优先于调用方。返回真
//（发出中断RangeTransactions的信号）iff f执行并返回
//真的。
func (s *Store) rangeUnminedTransactions(ns walletdb.ReadBucket, f func([]TxDetails) (bool, error)) (bool, error) {
	var details []TxDetails
	err := ns.NestedReadBucket(bucketUnmined).ForEach(func(k, v []byte) error {
		if len(k) < 32 {
			str := fmt.Sprintf("%s: short key (expected %d "+
				"bytes, read %d)", bucketUnmined, 32, len(k))
			return storeError(ErrData, str, nil)
		}

		var txHash chainhash.Hash
		copy(txHash[:], k)
		detail, err := s.unminedTxDetails(ns, &txHash, v)
		if err != nil {
			return err
		}

//因为密钥是在foreach覆盖
//水桶，它应该是永远不可能的
//成功返回nil details结构。
		details = append(details, *detail)
		return nil
	})
	if err == nil && len(details) > 0 {
		return f(details)
	}
	return false, err
}

//RangeBlockTransactions为每个块执行带有txtDetails的函数f
//在高度开始和结束之间（当结束>开始时颠倒顺序），直到F
//返回true，或者处理来自块的事务。返回真iff
//F执行并返回true。
func (s *Store) rangeBlockTransactions(ns walletdb.ReadBucket, begin, end int32,
	f func([]TxDetails) (bool, error)) (bool, error) {

//mempool高度被认为是一个上限。
	if begin < 0 {
		begin = int32(^uint32(0) >> 1)
	}
	if end < 0 {
		end = int32(^uint32(0) >> 1)
	}

	var blockIter blockIterator
	var advance func(*blockIterator) bool
	if begin < end {
//按向前顺序迭代
		blockIter = makeReadBlockIterator(ns, begin)
		advance = func(it *blockIterator) bool {
			if !it.next() {
				return false
			}
			return it.elem.Height <= end
		}
	} else {
//从begin->end反向迭代。
		blockIter = makeReadBlockIterator(ns, begin)
		advance = func(it *blockIterator) bool {
			if !it.prev() {
				return false
			}
			return end <= it.elem.Height
		}
	}

	var details []TxDetails
	for advance(&blockIter) {
		block := &blockIter.elem

		if cap(details) < len(block.transactions) {
			details = make([]TxDetails, 0, len(block.transactions))
		} else {
			details = details[:0]
		}

		for _, txHash := range block.transactions {
			k := keyTxRecord(&txHash, &block.Block)
			v := existsRawTxRecord(ns, k)
			if v == nil {
				str := fmt.Sprintf("missing transaction %v for "+
					"block %v", txHash, block.Height)
				return false, storeError(ErrData, str, nil)
			}
			detail := TxDetails{
				Block: BlockMeta{
					Block: block.Block,
					Time:  block.Time,
				},
			}
			err := readRawTxRecord(&txHash, v, &detail.TxRecord)
			if err != nil {
				return false, err
			}

			credIter := makeReadCreditIterator(ns, k)
			for credIter.next() {
				if int(credIter.elem.Index) >= len(detail.MsgTx.TxOut) {
					str := "saved credit index exceeds number of outputs"
					return false, storeError(ErrData, str, nil)
				}

//Credit迭代器不记录
//这个信用证是由一个无限制的
//交易，请在这里检查。
				if !credIter.elem.Spent {
					k := canonicalOutPoint(&txHash, credIter.elem.Index)
					spent := existsRawUnminedInput(ns, k) != nil
					credIter.elem.Spent = spent
				}
				detail.Credits = append(detail.Credits, credIter.elem)
			}
			if credIter.err != nil {
				return false, credIter.err
			}

			debIter := makeReadDebitIterator(ns, k)
			for debIter.next() {
				if int(debIter.elem.Index) >= len(detail.MsgTx.TxIn) {
					str := "saved debit index exceeds number of inputs"
					return false, storeError(ErrData, str, nil)
				}

				detail.Debits = append(detail.Debits, debIter.elem)
			}
			if debIter.err != nil {
				return false, debIter.err
			}

			details = append(details, detail)
		}

//每个块记录必须至少有一个事务，因此
//打F是安全的。
		brk, err := f(details)
		if err != nil || brk {
			return brk, err
		}
	}
	return false, blockIter.err
}

//RangeTransactions对以下所有事务详细信息运行函数f：
//在高度范围内的最佳链上的块[开始，结束]。特殊
//高度-1还可用于包括未限定的事务。如果结束
//高度在开始高度之前，块按相反顺序迭代
//首先处理未关联的事务（如果有）。
//
//函数f可能返回一个错误，如果不是nil，则该错误将传播到
//来电者。此外，布尔返回值允许退出函数
//如果为真，则在早期不读取任何附加事务。
//
//所有对f的调用都保证传递一个大于零的切片。
//元素。切片可以重用为多个块，因此不安全
//在获取循环迭代后使用它。
func (s *Store) RangeTransactions(ns walletdb.ReadBucket, begin, end int32,
	f func([]TxDetails) (bool, error)) error {

	var addedUnmined bool
	if begin < 0 {
		brk, err := s.rangeUnminedTransactions(ns, f)
		if err != nil || brk {
			return err
		}
		addedUnmined = true
	}

	brk, err := s.rangeBlockTransactions(ns, begin, end, f)
	if err == nil && !brk && !addedUnmined && end < 0 {
		_, err = s.rangeUnminedTransactions(ns, f)
	}
	return err
}

//previousupkscripts返回每个信贷的前一个输出脚本片段
//输出此交易记录的借方。
func (s *Store) PreviousPkScripts(ns walletdb.ReadBucket, rec *TxRecord, block *Block) ([][]byte, error) {
	var pkScripts [][]byte

	if block == nil {
		for _, input := range rec.MsgTx.TxIn {
			prevOut := &input.PreviousOutPoint

//输入可能会花费以前的未经编辑的输出，a
//开采产量（仍将标记
//或者两者都没有。

			v := existsRawUnmined(ns, prevOut.Hash[:])
			if v != nil {
//确保对此存在信用
//包括之前的非关联交易
//输出脚本。
				k := canonicalOutPoint(&prevOut.Hash, prevOut.Index)
				if existsRawUnminedCredit(ns, k) == nil {
					continue
				}

				pkScript, err := fetchRawTxRecordPkScript(
					prevOut.Hash[:], v, prevOut.Index)
				if err != nil {
					return nil, err
				}
				pkScripts = append(pkScripts, pkScript)
				continue
			}

			_, credKey := existsUnspent(ns, prevOut)
			if credKey != nil {
				k := extractRawCreditTxRecordKey(credKey)
				v = existsRawTxRecord(ns, k)
				pkScript, err := fetchRawTxRecordPkScript(k, v,
					prevOut.Index)
				if err != nil {
					return nil, err
				}
				pkScripts = append(pkScripts, pkScript)
				continue
			}
		}
		return pkScripts, nil
	}

	recKey := keyTxRecord(&rec.Hash, block)
	it := makeReadDebitIterator(ns, recKey)
	for it.next() {
		credKey := extractRawDebitCreditKey(it.cv)
		index := extractRawCreditIndex(credKey)
		k := extractRawCreditTxRecordKey(credKey)
		v := existsRawTxRecord(ns, k)
		pkScript, err := fetchRawTxRecordPkScript(k, v, index)
		if err != nil {
			return nil, err
		}
		pkScripts = append(pkScripts, pkScript)
	}
	if it.err != nil {
		return nil, it.err
	}

	return pkScripts, nil
}
