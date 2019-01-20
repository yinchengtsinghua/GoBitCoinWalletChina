
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
	"bytes"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/walletdb"
)

//块包含唯一标识块的最小数据量
//最好的或侧链。
type Block struct {
	Hash   chainhash.Hash
	Height int32
}

//blockmeta包含块和任何元数据的唯一标识
//用于修饰或说明该块。目前，这个附加的元数据只是
//包括来自块头的块时间。
type BlockMeta struct {
	Block
	Time time.Time
}

//BlockRecord是保存在
//数据库。
type blockRecord struct {
	Block
	Time         time.Time
	transactions []chainhash.Hash
}

//关联记录挖掘事务的块散列和区块链高度。
//因为仅事务哈希不足以唯一标识挖掘的
//事务（允许重复的事务散列），使用关联
//相反。
type incidence struct {
	txHash chainhash.Hash
	block  Block
}

//索引事件记录事务发生率和输入或输出
//索引。
type indexedIncidence struct {
	incidence
	index uint32
}

//借记记录交易记录从以前的钱包中借记的金额。
//交易信用。
type debit struct {
	txHash chainhash.Hash
	index  uint32
	amount btcutil.Amount
	spends indexedIncidence
}

//信用描述的是一个交易输出，它是或是可消费的钱包。
type credit struct {
	outPoint wire.OutPoint
	block    Block
	amount   btcutil.Amount
	change   bool
spentBy  indexedIncidence //index=^uint32（0）如果未使用
}

//TxRecord表示由存储管理的事务。
type TxRecord struct {
	MsgTx        wire.MsgTx
	Hash         chainhash.Hash
	Received     time.Time
SerializedTx []byte //可选：可以为零
}

//newtxrecord创建可以插入到
//商店。它使用memoization保存事务哈希和序列化
//交易。
func NewTxRecord(serializedTx []byte, received time.Time) (*TxRecord, error) {
	rec := &TxRecord{
		Received:     received,
		SerializedTx: serializedTx,
	}
	err := rec.MsgTx.Deserialize(bytes.NewReader(serializedTx))
	if err != nil {
		str := "failed to deserialize transaction"
		return nil, storeError(ErrInput, str, err)
	}
	copy(rec.Hash[:], chainhash.DoubleHashB(serializedTx))
	return rec, nil
}

//NewTxRecordFromMsgTx creates a new transaction record that may be inserted
//进入商店。
func NewTxRecordFromMsgTx(msgTx *wire.MsgTx, received time.Time) (*TxRecord, error) {
	buf := bytes.NewBuffer(make([]byte, 0, msgTx.SerializeSize()))
	err := msgTx.Serialize(buf)
	if err != nil {
		str := "failed to serialize transaction"
		return nil, storeError(ErrInput, str, err)
	}
	rec := &TxRecord{
		MsgTx:        *msgTx,
		Received:     received,
		SerializedTx: buf.Bytes(),
		Hash:         msgTx.TxHash(),
	}

	return rec, nil
}

//Credit是表示已花费或
//仍然可以用钱包消费。UTXO是一种未使用的信贷，但并非全部
//学分是UTXO。
type Credit struct {
	wire.OutPoint
	BlockMeta
	Amount       btcutil.Amount
	PkScript     []byte
	Received     time.Time
	FromCoinBase bool
}

//商店实现用于存储和管理钱包的交易商店
//交易。
type Store struct {
	chainParams *chaincfg.Params

//事件回调。它们与wtxmgr在同一个goroutine中执行
//来电者。
	NotifyUnspent func(hash *chainhash.Hash, index uint32)
}

//打开从walletdb名称空间打开wallet事务存储。如果
//存储不存在，返回errnoexist。
func Open(ns walletdb.ReadBucket, chainParams *chaincfg.Params) (*Store, error) {
//打开商店。
	err := openStore(ns)
	if err != nil {
		return nil, err
	}
s := &Store{chainParams, nil} //TODO:设置回调
	return s, nil
}

//创建在walletdb命名空间中创建新的持久事务存储。
//在此命名空间中已存在存储时创建存储将出错
//错误已存在。
func Create(ns walletdb.ReadWriteBucket) error {
	return createStore(ns)
}

//updateMedibalance更新存储中的已挖掘余额，如果更改，
//在处理给定的事务记录之后。
func (s *Store) updateMinedBalance(ns walletdb.ReadWriteBucket, rec *TxRecord,
	block *BlockMeta) error {

//如果需要更新，请提取已开采余额。
	minedBalance, err := fetchMinedBalance(ns)
	if err != nil {
		return err
	}

//为此交易记录花费的每个未使用的贷方添加借方记录。
//索引在下面的每个迭代中设置。
	spender := indexedIncidence{
		incidence: incidence{
			txHash: rec.Hash,
			block:  block.Block,
		},
	}

	newMinedBalance := minedBalance
	for i, input := range rec.MsgTx.TxIn {
		unspentKey, credKey := existsUnspent(ns, &input.PreviousOutPoint)
		if credKey == nil {
//非关联交易的借记没有明确规定
//跟踪。相反，任何
//未链接的事务将添加到映射中，以便快速
//当必须检查是否
//输出是否未暂停。
//
//跟踪未冻结交易的各个借方
//可以稍后添加以简化（并增加
//性能）确定需要的某些细节
//以前的输出（例如确定费用），但是
//the moment that is not done (and a db lookup is used
//对于这些情况）。还有一个很好的
//所有未链接的事务处理将
//完全移动到数据库，而不是在数据库中处理
//因为原子性的原因，所以
//目前正在使用实现。
			continue
		}

//如果这个输出与我们有关，我们将把它标记为已用
//从商店里取出它的数量。
		spender.index = uint32(i)
		amt, err := spendCredit(ns, credKey, &spender)
		if err != nil {
			return err
		}
		err = putDebit(
			ns, &rec.Hash, uint32(i), amt, &block.Block, credKey,
		)
		if err != nil {
			return err
		}
		if err := deleteRawUnspent(ns, unspentKey); err != nil {
			return err
		}

		newMinedBalance -= amt
	}

//对于标记为信用证的记录的每个输出，如果
//输出被未确认的存储标记为信用，请删除
//在数据库中将输出标记为贷记。
//
//移动的学分将作为未使用的学分添加，即使还有其他学分
//未经确认的支出交易。
	cred := credit{
		outPoint: wire.OutPoint{Hash: rec.Hash},
		block:    block.Block,
		spentBy:  indexedIncidence{index: ^uint32(0)},
	}

	it := makeUnminedCreditIterator(ns, &rec.Hash)
	for it.next() {
//TODO:这应该使用原始API。信用价值（IT.CV）
//可以直接从无链接移动到学分桶。
//键需要修改以包括块
//高度/散列。
		index, err := fetchRawUnminedCreditIndex(it.ck)
		if err != nil {
			return err
		}
		amount, change, err := fetchRawUnminedCreditAmountChange(it.cv)
		if err != nil {
			return err
		}

		cred.outPoint.Index = index
		cred.amount = amount
		cred.change = change

		if err := putUnspentCredit(ns, &cred); err != nil {
			return err
		}
		err = putUnspent(ns, &cred.outPoint, &block.Block)
		if err != nil {
			return err
		}

		newMinedBalance += amount
	}
	if it.err != nil {
		return it.err
	}

//如果余额已更改，则更新余额。
	if newMinedBalance != minedBalance {
		return putMinedBalance(ns, newMinedBalance)
	}

	return nil
}

//DeleteUnlinedX从存储中删除一个未链接的事务。
//
//注意：只有在挖掘事务后才应使用此选项。
func (s *Store) deleteUnminedTx(ns walletdb.ReadWriteBucket, rec *TxRecord) error {
	for i := range rec.MsgTx.TxOut {
		k := canonicalOutPoint(&rec.Hash, uint32(i))
		if err := deleteRawUnminedCredit(ns, k); err != nil {
			return err
		}
	}

	return deleteRawUnmined(ns, rec.Hash[:])
}

//inserttx记录属于钱包交易的交易
//历史。如果block为nil，则该事务被视为未暂停，并且
//必须取消设置事务的索引。
func (s *Store) InsertTx(ns walletdb.ReadWriteBucket, rec *TxRecord, block *BlockMeta) error {
	if block == nil {
		return s.insertMemPoolTx(ns, rec)
	}
	return s.insertMinedTx(ns, rec, block)
}

//removeUnlinedX尝试从
//事务存储。这将在以下情况中使用：
//我们试图重播，结果却花了我们的
//现有输入。此函数用于删除冲突事务
//由Tx记录标识，并递归地删除所有事务
//这取决于它。
func (s *Store) RemoveUnminedTx(ns walletdb.ReadWriteBucket, rec *TxRecord) error {
//因为我们已经有了一个Tx记录，我们可以直接调用
//removeconflict方法。这将完成递归删除的任务
//这是一个没有关联的事务，以及任何依赖它的事务。
	return s.removeConflict(ns, rec)
}

//insertminedtx将挖掘的事务的新事务记录插入到
//已确认存储桶下的数据库。它保证，如果
//传输之前未确认，然后它将负责清理
//未确认的状态。所有其他未确认的双倍消费尝试将
//也被移除了。
func (s *Store) insertMinedTx(ns walletdb.ReadWriteBucket, rec *TxRecord,
	block *BlockMeta) error {

//如果此哈希和块的事务记录已经存在，则我们
//可以提前退出。
	if _, v := existsTxRecord(ns, &rec.Hash, &block.Block); v != nil {
		return nil
	}

//如果对于来自此的任何事务，块记录尚不存在
//块，先插入块记录。否则，通过添加
//来自此块的事务集的事务哈希。
	var err error
	blockKey, blockValue := existsBlockRecord(ns, block.Height)
	if blockValue == nil {
		err = putBlockRecord(ns, block, &rec.Hash)
	} else {
		blockValue, err = appendRawBlockRecord(blockValue, &rec.Hash)
		if err != nil {
			return err
		}
		err = putRawBlockRecord(ns, blockKey, blockValue)
	}
	if err != nil {
		return err
	}
	if err := putTxRecord(ns, rec, &block.Block); err != nil {
		return err
	}

//确定此交易是否影响了我们的余额，如果影响了，
//更新它。
	if err := s.updateMinedBalance(ns, rec, block); err != nil {
		return err
	}

//如果此事务以前在存储区中以未经链接的形式存在，
//我们需要把它从没有链子的桶里拿出来。
	if v := existsRawUnmined(ns, rec.Hash[:]); v != nil {
		log.Infof("Marking unconfirmed transaction %v mined in block %d",
			&rec.Hash, block.Height)

		if err := s.deleteUnminedTx(ns, rec); err != nil {
			return err
		}
	}

//因为可能有未经确认的交易会因此失效。
//事务（重复或双倍开销），删除它们
//从未确认的集合。这还处理删除未确认的
//如果任何其他未确认的交易支出
//已删除的重复支出的输出。
	return s.removeDoubleSpends(ns, rec)
}

//addcredit将事务记录标记为包含事务输出
//可以用钱包消费。输出未经消耗而添加，并标记为已用
//当使用输出的新事务插入存储时。
//
//托多（JRICK）：这是不必要的。相反，传递索引
//当交易或merkleblock
//插入商店。
func (s *Store) AddCredit(ns walletdb.ReadWriteBucket, rec *TxRecord, block *BlockMeta, index uint32, change bool) error {
	if int(index) >= len(rec.MsgTx.TxOut) {
		str := "transaction output does not exist"
		return storeError(ErrInput, str, nil)
	}

	isNew, err := s.addCredit(ns, rec, block, index, change)
	if err == nil && isNew && s.NotifyUnspent != nil {
		s.NotifyUnspent(&rec.Hash, index)
	}
	return err
}

//addcredit是在更新事务中运行的addcredit帮助程序。这个
//bool return指定未暂停的输出是新添加的（true）还是
//重复（错误）。
func (s *Store) addCredit(ns walletdb.ReadWriteBucket, rec *TxRecord, block *BlockMeta, index uint32, change bool) (bool, error) {
	if block == nil {
//如果我们应该标记为信用的前哨已经存在
//在店内，无论是未确认还是确认，然后我们
//无事可做，可以离开。
		k := canonicalOutPoint(&rec.Hash, index)
		if existsRawUnminedCredit(ns, k) != nil {
			return false, nil
		}
		if existsRawUnspent(ns, k) != nil {
			return false, nil
		}
		v := valueUnminedCredit(btcutil.Amount(rec.MsgTx.TxOut[index].Value), change)
		return true, putRawUnminedCredit(ns, k, v)
	}

	k, v := existsCredit(ns, &rec.Hash, index, &block.Block)
	if v != nil {
		return false, nil
	}

	txOutAmt := btcutil.Amount(rec.MsgTx.TxOut[index].Value)
	log.Debugf("Marking transaction %v output %d (%v) spendable",
		rec.Hash, index, txOutAmt)

	cred := credit{
		outPoint: wire.OutPoint{
			Hash:  rec.Hash,
			Index: index,
		},
		block:   block.Block,
		amount:  txOutAmt,
		change:  change,
		spentBy: indexedIncidence{index: ^uint32(0)},
	}
	v = valueUnspentCredit(&cred)
	err := putRawCredit(ns, k, v)
	if err != nil {
		return false, err
	}

	minedBalance, err := fetchMinedBalance(ns)
	if err != nil {
		return false, err
	}
	err = putMinedBalance(ns, minedBalance+txOutAmt)
	if err != nil {
		return false, err
	}

	return true, putUnspent(ns, &cred.outPoint, &block.Block)
}

//回滚将删除高度上的所有块，并在其中移动任何事务
//到未确认池的每个块。
func (s *Store) Rollback(ns walletdb.ReadWriteBucket, height int32) error {
	return s.rollback(ns, height)
}

func (s *Store) rollback(ns walletdb.ReadWriteBucket, height int32) error {
	minedBalance, err := fetchMinedBalance(ns)
	if err != nil {
		return err
	}

//跟踪从CoinBase中删除的所有信用卡
//交易。分离所有块后，如果有任何事务记录
//存在于花掉这些输出、删除它们和它们的
//花钱连锁店。
//
//有必要将它们保存在内存中并修复未链接的
//之后的事务，因为块是按递增顺序删除的。
	var coinBaseCredits []wire.OutPoint
	var heightsToRemove []int32

	it := makeReverseBlockIterator(ns)
	for it.prev() {
		b := &it.elem
		if it.elem.Height < height {
			break
		}

		heightsToRemove = append(heightsToRemove, it.elem.Height)

		log.Infof("Rolling back %d transactions from block %v height %d",
			len(b.transactions), b.Hash, b.Height)

		for i := range b.transactions {
			txHash := &b.transactions[i]

			recKey := keyTxRecord(txHash, &b.Block)
			recVal := existsRawTxRecord(ns, recKey)
			var rec TxRecord
			err = readRawTxRecord(txHash, recVal, &rec)
			if err != nil {
				return err
			}

			err = deleteTxRecord(ns, txHash, &b.Block)
			if err != nil {
				return err
			}

//特别处理CoinBase事务，因为它们是
//未移动到未确认的存储区。coinbase不能
//包含任何借方，但应删除所有贷方
//采出的矿藏储量减少。
			if blockchain.IsCoinBaseTx(&rec.MsgTx) {
				op := wire.OutPoint{Hash: rec.Hash}
				for i, output := range rec.MsgTx.TxOut {
					k, v := existsCredit(ns, &rec.Hash,
						uint32(i), &b.Block)
					if v == nil {
						continue
					}
					op.Index = uint32(i)

					coinBaseCredits = append(coinBaseCredits, op)

					unspentKey, credKey := existsUnspent(ns, &op)
					if credKey != nil {
						minedBalance -= btcutil.Amount(output.Value)
						err = deleteRawUnspent(ns, unspentKey)
						if err != nil {
							return err
						}
					}
					err = deleteRawCredit(ns, k)
					if err != nil {
						return err
					}
				}

				continue
			}

			err = putRawUnmined(ns, txHash[:], recVal)
			if err != nil {
				return err
			}

//对于为此交易记录的每个借方，标记
//它花费的信贷是未使用的（只要它仍然
//存在）并删除借方。上一个输出是
//记录在未确认的存储中
//输出，而不仅仅是借记。
			for i, input := range rec.MsgTx.TxIn {
				prevOut := &input.PreviousOutPoint
				prevOutKey := canonicalOutPoint(&prevOut.Hash,
					prevOut.Index)
				err = putRawUnminedInput(ns, prevOutKey, rec.Hash[:])
				if err != nil {
					return err
				}

//如果此输入是借项，请删除借项
//记录并标记其作为
//未开采，增加开采平衡。
				debKey, credKey, err := existsDebit(ns,
					&rec.Hash, uint32(i), &b.Block)
				if err != nil {
					return err
				}
				if debKey == nil {
					continue
				}

//UnspendrawCredit不会出错，如果
//这个钥匙没有信用，但是这个
//行为正确。因为积木是
//按递增顺序删除，此信用证
//可能已从
//以前在中删除的事务记录
//这个回滚。
				var amt btcutil.Amount
				amt, err = unspendRawCredit(ns, credKey)
				if err != nil {
					return err
				}
				err = deleteRawDebit(ns, debKey)
				if err != nil {
					return err
				}

//如果信用证之前在
//回滚，贷方金额为零。只有
//将以前使用过的信贷标记为未使用
//如果它仍然存在。
				if amt == 0 {
					continue
				}
				unspentVal, err := fetchRawCreditUnspentValue(credKey)
				if err != nil {
					return err
				}
				minedBalance += amt
				err = putRawUnspent(ns, prevOutKey, unspentVal)
				if err != nil {
					return err
				}
			}

//对于每个独立的非CoinBase信贷，移动
//信用输出到未经许可。如果信用证有标记
//未使用时，它将从UTXO集合中移除，并且
//已开采余额减少。
//
//TODO:使用信用迭代器
			for i, output := range rec.MsgTx.TxOut {
				k, v := existsCredit(ns, &rec.Hash, uint32(i),
					&b.Block)
				if v == nil {
					continue
				}

				amt, change, err := fetchRawCreditAmountChange(v)
				if err != nil {
					return err
				}
				outPointKey := canonicalOutPoint(&rec.Hash, uint32(i))
				unminedCredVal := valueUnminedCredit(amt, change)
				err = putRawUnminedCredit(ns, outPointKey, unminedCredVal)
				if err != nil {
					return err
				}

				err = deleteRawCredit(ns, k)
				if err != nil {
					return err
				}

				credKey := existsRawUnspent(ns, outPointKey)
				if credKey != nil {
					minedBalance -= btcutil.Amount(output.Value)
					err = deleteRawUnspent(ns, outPointKey)
					if err != nil {
						return err
					}
				}
			}
		}

//删除此k/v对并前进到
//以前的。
		it.reposition(it.elem.Height)

//避免删除光标，直到解决螺栓问题620。
//err=it.delete（）。
//如果犯错！= nIL{
//返回错误
//}
	}
	if it.err != nil {
		return it.err
	}

//删除光标后删除迭代外的块记录
//被打破了。
	for _, h := range heightsToRemove {
		err = deleteBlockRecord(ns, h)
		if err != nil {
			return err
		}
	}

	for _, op := range coinBaseCredits {
		opKey := canonicalOutPoint(&op.Hash, op.Index)
		unminedSpendTxHashKeys := fetchUnminedInputSpendTxHashes(ns, opKey)
		for _, unminedSpendTxHashKey := range unminedSpendTxHashKeys {
			unminedVal := existsRawUnmined(ns, unminedSpendTxHashKey[:])

//如果支出交易花费多个输出
//在同一笔交易中，我们会发现重复的
//商店内的条目，因此我们可能
//如果冲突已经发生，则无法找到它。
//在上一次迭代中删除。
			if unminedVal == nil {
				continue
			}

			var unminedRec TxRecord
			unminedRec.Hash = unminedSpendTxHashKey
			err = readRawTxRecord(&unminedRec.Hash, unminedVal, &unminedRec)
			if err != nil {
				return err
			}

			log.Debugf("Transaction %v spends a removed coinbase "+
				"output -- removing as well", unminedRec.Hash)
			err = s.removeConflict(ns, &unminedRec)
			if err != nil {
				return err
			}
		}
	}

	return putMinedBalance(ns, minedBalance)
}

//未暂停的输出返回所有未暂停的已接收事务输出。
//顺序未定义。
func (s *Store) UnspentOutputs(ns walletdb.ReadBucket) ([]Credit, error) {
	var unspent []Credit

	var op wire.OutPoint
	var block Block
	err := ns.NestedReadBucket(bucketUnspent).ForEach(func(k, v []byte) error {
		err := readCanonicalOutPoint(k, &op)
		if err != nil {
			return err
		}
		if existsRawUnminedInput(ns, k) != nil {
//输出由未链接的事务使用。
//跳过这个k/v对。
			return nil
		}
		err = readUnspentBlock(v, &block)
		if err != nil {
			return err
		}

		blockTime, err := fetchBlockTime(ns, block.Height)
		if err != nil {
			return err
		}
//TODO（JRICK）：读取整个事务应该
//可以避免。创建信用证只需要
//输出量和pkscript。
		rec, err := fetchTxRecord(ns, &op.Hash, &block)
		if err != nil {
			return err
		}
		txOut := rec.MsgTx.TxOut[op.Index]
		cred := Credit{
			OutPoint: op,
			BlockMeta: BlockMeta{
				Block: block,
				Time:  blockTime,
			},
			Amount:       btcutil.Amount(txOut.Value),
			PkScript:     txOut.PkScript,
			Received:     rec.Received,
			FromCoinBase: blockchain.IsCoinBaseTx(&rec.MsgTx),
		}
		unspent = append(unspent, cred)
		return nil
	})
	if err != nil {
		if _, ok := err.(Error); ok {
			return nil, err
		}
		str := "failed iterating unspent bucket"
		return nil, storeError(ErrDatabase, str, err)
	}

	err = ns.NestedReadBucket(bucketUnminedCredits).ForEach(func(k, v []byte) error {
		if existsRawUnminedInput(ns, k) != nil {
//输出由未链接的事务使用。
//跳到下一个无限制信用证。
			return nil
		}

		err := readCanonicalOutPoint(k, &op)
		if err != nil {
			return err
		}

//TODO（JRICK）：读取/分析整个事务记录
//只需输出量和脚本就可以避免。
		recVal := existsRawUnmined(ns, op.Hash[:])
		var rec TxRecord
		err = readRawTxRecord(&op.Hash, recVal, &rec)
		if err != nil {
			return err
		}

		txOut := rec.MsgTx.TxOut[op.Index]
		cred := Credit{
			OutPoint: op,
			BlockMeta: BlockMeta{
				Block: Block{Height: -1},
			},
			Amount:       btcutil.Amount(txOut.Value),
			PkScript:     txOut.PkScript,
			Received:     rec.Received,
			FromCoinBase: blockchain.IsCoinBaseTx(&rec.MsgTx),
		}
		unspent = append(unspent, cred)
		return nil
	})
	if err != nil {
		if _, ok := err.(Error); ok {
			return nil, err
		}
		str := "failed iterating unmined credits bucket"
		return nil, storeError(ErrDatabase, str, err)
	}

	return unspent, nil
}

//余额返回可消费钱包余额（所有未使用的总价值
//交易输出），至少给出minconf确认，计算
//当前链高度为curheight。仅包括CoinBase输出
//在余额中，如果到期。
//
//如果同步度低于块，则平衡可能返回意外结果。
//存储中最近挖掘的事务的高度。
func (s *Store) Balance(ns walletdb.ReadBucket, minConf int32, syncHeight int32) (btcutil.Amount, error) {
	bal, err := fetchMinedBalance(ns)
	if err != nil {
		return 0, err
	}

//减去未贷出的每个贷方的余额
//交易。
	var op wire.OutPoint
	var block Block
	err = ns.NestedReadBucket(bucketUnspent).ForEach(func(k, v []byte) error {
		err := readCanonicalOutPoint(k, &op)
		if err != nil {
			return err
		}
		err = readUnspentBlock(v, &block)
		if err != nil {
			return err
		}
		if existsRawUnminedInput(ns, k) != nil {
			_, v := existsCredit(ns, &op.Hash, op.Index, &block)
			amt, err := fetchRawCreditAmount(v)
			if err != nil {
				return err
			}
			bal -= amt
		}
		return nil
	})
	if err != nil {
		if _, ok := err.(Error); ok {
			return 0, err
		}
		str := "failed iterating unspent outputs"
		return 0, storeError(ErrDatabase, str, err)
	}

//将任何未使用的信贷余额减至
//minconf确认书和任何（未使用的）不成熟的货币基础信贷。
	coinbaseMaturity := int32(s.chainParams.CoinbaseMaturity)
	stopConf := minConf
	if coinbaseMaturity > stopConf {
		stopConf = coinbaseMaturity
	}
	lastHeight := syncHeight - stopConf
	blockIt := makeReadReverseBlockIterator(ns)
	for blockIt.prev() {
		block := &blockIt.elem

		if block.Height < lastHeight {
			break
		}

		for i := range block.transactions {
			txHash := &block.transactions[i]
			rec, err := fetchTxRecord(ns, txHash, &block.Block)
			if err != nil {
				return 0, err
			}
			numOuts := uint32(len(rec.MsgTx.TxOut))
			for i := uint32(0); i < numOuts; i++ {
//避免双倍减少信贷金额
//如果它已经被移除
//未开采的TX
				opKey := canonicalOutPoint(txHash, i)
				if existsRawUnminedInput(ns, opKey) != nil {
					continue
				}

				_, v := existsCredit(ns, txHash, i, &block.Block)
				if v == nil {
					continue
				}
				amt, spent, err := fetchRawCreditAmountSpent(v)
				if err != nil {
					return 0, err
				}
				if spent {
					continue
				}
				confs := syncHeight - block.Height + 1
				if confs < minConf || (blockchain.IsCoinBaseTx(&rec.MsgTx) &&
					confs < coinbaseMaturity) {
					bal -= amt
				}
			}
		}
	}
	if blockIt.err != nil {
		return 0, blockIt.err
	}

//如果包括无限制输出，则增加每个输出的平衡
//未消耗的输出。
	if minConf == 0 {
		err = ns.NestedReadBucket(bucketUnminedCredits).ForEach(func(k, v []byte) error {
			if existsRawUnminedInput(ns, k) != nil {
//输出由未链接的事务使用。
//跳到下一个无限制信用证。
				return nil
			}

			amount, err := fetchRawUnminedCreditAmount(v)
			if err != nil {
				return err
			}
			bal += amount
			return nil
		})
		if err != nil {
			if _, ok := err.(Error); ok {
				return 0, err
			}
			str := "failed to iterate over unmined credits bucket"
			return 0, storeError(ErrDatabase, str, err)
		}
	}

	return bal, nil
}
