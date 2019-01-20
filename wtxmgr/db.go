
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015 BTCSuite开发者
//版权所有（c）2015版权所有
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wtxmgr

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/walletdb"
)

//命名
//
//此文件中通常使用以下变量并给出
//保留名称：
//
//ns:此包的命名空间存储桶
//B：正在操作的主铲斗
//K：单桶钥匙
//V：单个桶值
//C：桶形光标
//ck：当前光标键
//cv：当前光标值
//
//函数使用命名方案“op[raw]type[field]”，它执行
//对“type”类型执行“op”操作，可以选择处理原始密钥和
//如果使用了“raw”，则返回值。提取和提取操作可能只需要读取
//键或值的某些部分，在这种情况下，“field”描述组件
//正在返回。使用以下操作：
//
//key：返回一些数据的db key
//值：返回某些数据的db值
//放入：在桶中插入或替换一个值
//获取：读取并返回值
//读取：将值读取到输出参数中
//存在：返回某些数据的原始值（如果未找到则为零）。
//删除：删除K/V对
//提取：执行未选中的切片以提取键或值
//
//其他特定于正在操作的类型的操作
//应该在评论中解释。

//由于光标扫描整数，大尾数是首选的字节顺序。
//按顺序迭代键。
var byteOrder = binary.BigEndian

//此包假设chainhash.hash的宽度始终为
//32字节。如果这种情况发生了变化（比特币不太可能，Alts也可能发生变化）。
//偏移量必须重写。使用编译时断言
//假设成立。
var _ [32]byte = chainhash.Hash{}

//桶名
var (
	bucketBlocks         = []byte("b")
	bucketTxRecords      = []byte("t")
	bucketCredits        = []byte("c")
	bucketUnspent        = []byte("u")
	bucketDebits         = []byte("d")
	bucketUnmined        = []byte("m")
	bucketUnminedCredits = []byte("mc")
	bucketUnminedInputs  = []byte("mi")
)

//根（命名空间）存储桶键
var (
	rootCreateDate   = []byte("date")
	rootVersion      = []byte("vers")
	rootMinedBalance = []byte("bal")
)

//根桶的挖掘平衡k/v对记录所有的总平衡
//开采交易中未使用的信贷。这包括不成熟的产出，以及
//mempool事务所花费的输出，必须在
//返回给定数量的块确认的实际余额。这个
//值是序列化为uint64的金额。
func fetchMinedBalance(ns walletdb.ReadBucket) (btcutil.Amount, error) {
	v := ns.Get(rootMinedBalance)
	if len(v) != 8 {
		str := fmt.Sprintf("balance: short read (expected 8 bytes, "+
			"read %v)", len(v))
		return 0, storeError(ErrData, str, nil)
	}
	return btcutil.Amount(byteOrder.Uint64(v)), nil
}

func putMinedBalance(ns walletdb.ReadWriteBucket, amt btcutil.Amount) error {
	v := make([]byte, 8)
	byteOrder.PutUint64(v, uint64(amt))
	err := ns.Put(rootMinedBalance, v)
	if err != nil {
		str := "failed to put balance"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

//几种数据结构都被指定为规范化的序列化格式
//键或值。这些常见格式允许重用键和值
//穿过不同的桶。
//
//规范化输出点序列化格式为：
//
//[0:32]传输哈希（32字节）
//[32:36]输出索引（4字节）
//
//规范事务哈希序列化只是哈希。

func canonicalOutPoint(txHash *chainhash.Hash, index uint32) []byte {
	k := make([]byte, 36)
	copy(k, txHash[:])
	byteOrder.PutUint32(k[32:36], index)
	return k
}

func readCanonicalOutPoint(k []byte, op *wire.OutPoint) error {
	if len(k) < 36 {
		str := "short canonical outpoint"
		return storeError(ErrData, str, nil)
	}
	copy(op.Hash[:], k)
	op.Index = byteOrder.Uint32(k[32:36])
	return nil
}

//块的详细信息在块存储桶中保存为k/v对。
//块记录按其高度键入。值序列化如下：
//
//[0:32]哈希（32字节）
//[32:40]Unix时间（8字节）
//[40:44]事务哈希数（4字节）
//[44:]对于每个事务哈希：
//散列（32字节）

func keyBlockRecord(height int32) []byte {
	k := make([]byte, 4)
	byteOrder.PutUint32(k, uint32(height))
	return k
}

func valueBlockRecord(block *BlockMeta, txHash *chainhash.Hash) []byte {
	v := make([]byte, 76)
	copy(v, block.Hash[:])
	byteOrder.PutUint64(v[32:40], uint64(block.Time.Unix()))
	byteOrder.PutUint32(v[40:44], 1)
	copy(v[44:76], txHash[:])
	return v
}

//AppendrawBlockRecord返回带有事务的新块记录值
//附加到结尾的哈希和递增的事务数。
func appendRawBlockRecord(v []byte, txHash *chainhash.Hash) ([]byte, error) {
	if len(v) < 44 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketBlocks, 44, len(v))
		return nil, storeError(ErrData, str, nil)
	}
	newv := append(v[:len(v):len(v)], txHash[:]...)
	n := byteOrder.Uint32(newv[40:44])
	byteOrder.PutUint32(newv[40:44], n+1)
	return newv, nil
}

func putRawBlockRecord(ns walletdb.ReadWriteBucket, k, v []byte) error {
	err := ns.NestedReadWriteBucket(bucketBlocks).Put(k, v)
	if err != nil {
		str := "failed to store block"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

func putBlockRecord(ns walletdb.ReadWriteBucket, block *BlockMeta, txHash *chainhash.Hash) error {
	k := keyBlockRecord(block.Height)
	v := valueBlockRecord(block, txHash)
	return putRawBlockRecord(ns, k, v)
}

func fetchBlockTime(ns walletdb.ReadBucket, height int32) (time.Time, error) {
	k := keyBlockRecord(height)
	v := ns.NestedReadBucket(bucketBlocks).Get(k)
	if len(v) < 44 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketBlocks, 44, len(v))
		return time.Time{}, storeError(ErrData, str, nil)
	}
	return time.Unix(int64(byteOrder.Uint64(v[32:40])), 0), nil
}

func existsBlockRecord(ns walletdb.ReadBucket, height int32) (k, v []byte) {
	k = keyBlockRecord(height)
	v = ns.NestedReadBucket(bucketBlocks).Get(k)
	return
}

func readRawBlockRecord(k, v []byte, block *blockRecord) error {
	if len(k) < 4 {
		str := fmt.Sprintf("%s: short key (expected %d bytes, read %d)",
			bucketBlocks, 4, len(k))
		return storeError(ErrData, str, nil)
	}
	if len(v) < 44 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketBlocks, 44, len(v))
		return storeError(ErrData, str, nil)
	}
	numTransactions := int(byteOrder.Uint32(v[40:44]))
	expectedLen := 44 + chainhash.HashSize*numTransactions
	if len(v) < expectedLen {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketBlocks, expectedLen, len(v))
		return storeError(ErrData, str, nil)
	}

	block.Height = int32(byteOrder.Uint32(k))
	copy(block.Hash[:], v)
	block.Time = time.Unix(int64(byteOrder.Uint64(v[32:40])), 0)
	block.transactions = make([]chainhash.Hash, numTransactions)
	off := 44
	for i := range block.transactions {
		copy(block.transactions[i][:], v[off:])
		off += chainhash.HashSize
	}

	return nil
}

type blockIterator struct {
	c    walletdb.ReadWriteCursor
	seek []byte
	ck   []byte
	cv   []byte
	elem blockRecord
	err  error
}

func makeBlockIterator(ns walletdb.ReadWriteBucket, height int32) blockIterator {
	seek := make([]byte, 4)
	byteOrder.PutUint32(seek, uint32(height))
	c := ns.NestedReadWriteBucket(bucketBlocks).ReadWriteCursor()
	return blockIterator{c: c, seek: seek}
}

func makeReadBlockIterator(ns walletdb.ReadBucket, height int32) blockIterator {
	seek := make([]byte, 4)
	byteOrder.PutUint32(seek, uint32(height))
	c := ns.NestedReadBucket(bucketBlocks).ReadCursor()
	return blockIterator{c: readCursor{c}, seek: seek}
}

//与makeBlockIterator类似，但最初将光标定位在
//最后一对K/V。与BlockIterator.prev一起使用。
func makeReverseBlockIterator(ns walletdb.ReadWriteBucket) blockIterator {
	seek := make([]byte, 4)
	byteOrder.PutUint32(seek, ^uint32(0))
	c := ns.NestedReadWriteBucket(bucketBlocks).ReadWriteCursor()
	return blockIterator{c: c, seek: seek}
}

func makeReadReverseBlockIterator(ns walletdb.ReadBucket) blockIterator {
	seek := make([]byte, 4)
	byteOrder.PutUint32(seek, ^uint32(0))
	c := ns.NestedReadBucket(bucketBlocks).ReadCursor()
	return blockIterator{c: readCursor{c}, seek: seek}
}

func (it *blockIterator) next() bool {
	if it.c == nil {
		return false
	}

	if it.ck == nil {
		it.ck, it.cv = it.c.Seek(it.seek)
	} else {
		it.ck, it.cv = it.c.Next()
	}
	if it.ck == nil {
		it.c = nil
		return false
	}

	err := readRawBlockRecord(it.ck, it.cv, &it.elem)
	if err != nil {
		it.c = nil
		it.err = err
		return false
	}

	return true
}

func (it *blockIterator) prev() bool {
	if it.c == nil {
		return false
	}

	if it.ck == nil {
		it.ck, it.cv = it.c.Seek(it.seek)
//SEEK将光标定位在下一个k/v对上，如果其中一个
//找不到此前缀。如果发生这种情况（前缀
//在这种情况下不匹配）向后移动光标。
//
//这在技术上不适用于带有
//通过将光标移动到最后一个匹配项来匹配前缀
//关键，但在处理
//由于键（和seek前缀）只是
//块高度。
		if !bytes.HasPrefix(it.ck, it.seek) {
			it.ck, it.cv = it.c.Prev()
		}
	} else {
		it.ck, it.cv = it.c.Prev()
	}
	if it.ck == nil {
		it.c = nil
		return false
	}

	err := readRawBlockRecord(it.ck, it.cv, &it.elem)
	if err != nil {
		it.c = nil
		it.err = err
		return false
	}

	return true
}

//在https://github.com/boltdb/bolt/issues/620修复之前不可用。
//func（it*blockIterator）delete（）错误
//错误：=it.c.delete（）。
//如果犯错！= nIL{
//str：=“删除块记录失败”
//StoreError（错误数据库、str、err）
//}
//返回零
//}

func (it *blockIterator) reposition(height int32) {
	it.c.Seek(keyBlockRecord(height))
}

func deleteBlockRecord(ns walletdb.ReadWriteBucket, height int32) error {
	k := keyBlockRecord(height)
	return ns.NestedReadWriteBucket(bucketBlocks).Delete(k)
}

//交易记录按如下方式键入：
//
//[0:32]事务哈希（32字节）
//[32:36]块高度（4字节）
//[36:68]块哈希（32字节）
//
//前导事务哈希允许为具有
//匹配的哈希。块高度和散列记录特定的发生率
//区块链中的交易。
//
//记录值序列化如下：
//
//[0:8]接收时间（8字节）
//[8:]序列化事务（可变）

func keyTxRecord(txHash *chainhash.Hash, block *Block) []byte {
	k := make([]byte, 68)
	copy(k, txHash[:])
	byteOrder.PutUint32(k[32:36], uint32(block.Height))
	copy(k[36:68], block.Hash[:])
	return k
}

func valueTxRecord(rec *TxRecord) ([]byte, error) {
	var v []byte
	if rec.SerializedTx == nil {
		txSize := rec.MsgTx.SerializeSize()
		v = make([]byte, 8, 8+txSize)
		err := rec.MsgTx.Serialize(bytes.NewBuffer(v[8:]))
		if err != nil {
			str := fmt.Sprintf("unable to serialize transaction %v", rec.Hash)
			return nil, storeError(ErrInput, str, err)
		}
		v = v[:cap(v)]
	} else {
		v = make([]byte, 8+len(rec.SerializedTx))
		copy(v[8:], rec.SerializedTx)
	}
	byteOrder.PutUint64(v, uint64(rec.Received.Unix()))
	return v, nil
}

func putTxRecord(ns walletdb.ReadWriteBucket, rec *TxRecord, block *Block) error {
	k := keyTxRecord(&rec.Hash, block)
	v, err := valueTxRecord(rec)
	if err != nil {
		return err
	}
	err = ns.NestedReadWriteBucket(bucketTxRecords).Put(k, v)
	if err != nil {
		str := fmt.Sprintf("%s: put failed for %v", bucketTxRecords, rec.Hash)
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

func putRawTxRecord(ns walletdb.ReadWriteBucket, k, v []byte) error {
	err := ns.NestedReadWriteBucket(bucketTxRecords).Put(k, v)
	if err != nil {
		str := fmt.Sprintf("%s: put failed", bucketTxRecords)
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

func readRawTxRecord(txHash *chainhash.Hash, v []byte, rec *TxRecord) error {
	if len(v) < 8 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketTxRecords, 8, len(v))
		return storeError(ErrData, str, nil)
	}
	rec.Hash = *txHash
	rec.Received = time.Unix(int64(byteOrder.Uint64(v)), 0)
	err := rec.MsgTx.Deserialize(bytes.NewReader(v[8:]))
	if err != nil {
		str := fmt.Sprintf("%s: failed to deserialize transaction %v",
			bucketTxRecords, txHash)
		return storeError(ErrData, str, err)
	}
	return nil
}

func readRawTxRecordBlock(k []byte, block *Block) error {
	if len(k) < 68 {
		str := fmt.Sprintf("%s: short key (expected %d bytes, read %d)",
			bucketTxRecords, 68, len(k))
		return storeError(ErrData, str, nil)
	}
	block.Height = int32(byteOrder.Uint32(k[32:36]))
	copy(block.Hash[:], k[36:68])
	return nil
}

func fetchTxRecord(ns walletdb.ReadBucket, txHash *chainhash.Hash, block *Block) (*TxRecord, error) {
	k := keyTxRecord(txHash, block)
	v := ns.NestedReadBucket(bucketTxRecords).Get(k)

	rec := new(TxRecord)
	err := readRawTxRecord(txHash, v, rec)
	return rec, err
}

//托多：这读起来太多了。将pkscript位置传递给
//避免Wire.msgtx反序列化。
func fetchRawTxRecordPkScript(k, v []byte, index uint32) ([]byte, error) {
	var rec TxRecord
copy(rec.Hash[:], k) //愚蠢但需要一个阵列
	err := readRawTxRecord(&rec.Hash, v, &rec)
	if err != nil {
		return nil, err
	}
	if int(index) >= len(rec.MsgTx.TxOut) {
		str := "missing transaction output for credit index"
		return nil, storeError(ErrData, str, nil)
	}
	return rec.MsgTx.TxOut[index].PkScript, nil
}

func existsTxRecord(ns walletdb.ReadBucket, txHash *chainhash.Hash, block *Block) (k, v []byte) {
	k = keyTxRecord(txHash, block)
	v = ns.NestedReadBucket(bucketTxRecords).Get(k)
	return
}

func existsRawTxRecord(ns walletdb.ReadBucket, k []byte) (v []byte) {
	return ns.NestedReadBucket(bucketTxRecords).Get(k)
}

func deleteTxRecord(ns walletdb.ReadWriteBucket, txHash *chainhash.Hash, block *Block) error {
	k := keyTxRecord(txHash, block)
	return ns.NestedReadWriteBucket(bucketTxRecords).Delete(k)
}

//latesttxrecord搜索最新记录的挖掘事务记录
//匹配的哈希。在哈希冲突的情况下，最新的
//返回块。如果找不到匹配的交易记录，则返回（nil，nil）。
func latestTxRecord(ns walletdb.ReadBucket, txHash *chainhash.Hash) (k, v []byte) {
	prefix := txHash[:]
	c := ns.NestedReadBucket(bucketTxRecords).ReadCursor()
	ck, cv := c.Seek(prefix)
	var lastKey, lastVal []byte
	for bytes.HasPrefix(ck, prefix) {
		lastKey, lastVal = ck, cv
		ck, cv = c.Next()
	}
	return lastKey, lastVal
}

//所有交易信用（输出）都按如下方式键入：
//
//[0:32]事务哈希（32字节）
//[32:36]块高度（4字节）
//[36:68]块哈希（32字节）
//[68:72]输出索引（4字节）
//
//前68个字节与事务记录的键匹配，可以使用
//作为一个前缀过滤器，按顺序迭代所有的信用证。
//
//信用值序列化如下：
//
//[0:8]数量（8字节）
//[8]标志（1字节）
//0x01
//0x02：改变
//[9:81]可选借记卡密钥（72字节）
//[9:41]Spender事务哈希（32字节）
//[41:45]Spender块高度（4字节）
//[45:77]Spender块哈希（32字节）
//[77:81]Spender事务输入索引（4字节）
//
//只有当信用证由另一方使用时，才包括可选的借记密钥。
//开采借方

func keyCredit(txHash *chainhash.Hash, index uint32, block *Block) []byte {
	k := make([]byte, 72)
	copy(k, txHash[:])
	byteOrder.PutUint32(k[32:36], uint32(block.Height))
	copy(k[36:68], block.Hash[:])
	byteOrder.PutUint32(k[68:72], index)
	return k
}

//价值未使用的信贷为未使用的信贷创建新的信贷价值。所有
//信用证是未使用的，只在以后才标记为已使用，因此没有
//用于创建已用或未用信用的值函数。
func valueUnspentCredit(cred *credit) []byte {
	v := make([]byte, 9)
	byteOrder.PutUint64(v, uint64(cred.amount))
	if cred.change {
		v[8] |= 1 << 1
	}
	return v
}

func putRawCredit(ns walletdb.ReadWriteBucket, k, v []byte) error {
	err := ns.NestedReadWriteBucket(bucketCredits).Put(k, v)
	if err != nil {
		str := "failed to put credit"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

//PutunPentCredit为未使用的信用记录。它可能只是
//当信用证已经被知道是未使用，或被
//未确认的交易。
func putUnspentCredit(ns walletdb.ReadWriteBucket, cred *credit) error {
	k := keyCredit(&cred.outPoint.Hash, cred.outPoint.Index, &cred.block)
	v := valueUnspentCredit(cred)
	return putRawCredit(ns, k, v)
}

func extractRawCreditTxRecordKey(k []byte) []byte {
	return k[0:68]
}

func extractRawCreditIndex(k []byte) uint32 {
	return byteOrder.Uint32(k[68:72])
}

//fetchrawcreditAmount返回贷方的金额。
func fetchRawCreditAmount(v []byte) (btcutil.Amount, error) {
	if len(v) < 9 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketCredits, 9, len(v))
		return 0, storeError(ErrData, str, nil)
	}
	return btcutil.Amount(byteOrder.Uint64(v)), nil
}

//fetchrawcreditamountspeed返回信用证的金额以及
//信用被消耗了。
func fetchRawCreditAmountSpent(v []byte) (btcutil.Amount, bool, error) {
	if len(v) < 9 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketCredits, 9, len(v))
		return 0, false, storeError(ErrData, str, nil)
	}
	return btcutil.Amount(byteOrder.Uint64(v)), v[8]&(1<<0) != 0, nil
}

//fetchrawCreditAmountChange返回信用证的金额以及
//信用证被标记为变更。
func fetchRawCreditAmountChange(v []byte) (btcutil.Amount, bool, error) {
	if len(v) < 9 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketCredits, 9, len(v))
		return 0, false, storeError(ErrData, str, nil)
	}
	return btcutil.Amount(byteOrder.Uint64(v)), v[8]&(1<<1) != 0, nil
}

//fetchrawcreditunspentvalue返回原始信贷密钥的未使用值。
//这可用于将信用证标记为未使用。
func fetchRawCreditUnspentValue(k []byte) ([]byte, error) {
	if len(k) < 72 {
		str := fmt.Sprintf("%s: short key (expected %d bytes, read %d)",
			bucketCredits, 72, len(k))
		return nil, storeError(ErrData, str, nil)
	}
	return k[32:68], nil
}

//SpendrawCredit用特定的密钥标记信用
//在某些事务发生时由输入花费的块。借记
//返回金额。
func spendCredit(ns walletdb.ReadWriteBucket, k []byte, spender *indexedIncidence) (btcutil.Amount, error) {
	v := ns.NestedReadBucket(bucketCredits).Get(k)
	newv := make([]byte, 81)
	copy(newv, v)
	v = newv
	v[8] |= 1 << 0
	copy(v[9:41], spender.txHash[:])
	byteOrder.PutUint32(v[41:45], uint32(spender.block.Height))
	copy(v[45:77], spender.block.Hash[:])
	byteOrder.PutUint32(v[77:81], spender.index)

	return btcutil.Amount(byteOrder.Uint64(v[0:8])), putRawCredit(ns, k, v)
}

//unspendrawcredit将给定密钥的信用重写为unspent。这个
//返回贷方的输出金额。如果没有，它会毫无错误地返回
//存在该密钥的信用。
func unspendRawCredit(ns walletdb.ReadWriteBucket, k []byte) (btcutil.Amount, error) {
	b := ns.NestedReadWriteBucket(bucketCredits)
	v := b.Get(k)
	if v == nil {
		return 0, nil
	}
	newv := make([]byte, 9)
	copy(newv, v)
	newv[8] &^= 1 << 0

	err := b.Put(k, newv)
	if err != nil {
		str := "failed to put credit"
		return 0, storeError(ErrDatabase, str, err)
	}
	return btcutil.Amount(byteOrder.Uint64(v[0:8])), nil
}

func existsCredit(ns walletdb.ReadBucket, txHash *chainhash.Hash, index uint32, block *Block) (k, v []byte) {
	k = keyCredit(txHash, index, block)
	v = ns.NestedReadBucket(bucketCredits).Get(k)
	return
}

func existsRawCredit(ns walletdb.ReadBucket, k []byte) []byte {
	return ns.NestedReadBucket(bucketCredits).Get(k)
}

func deleteRawCredit(ns walletdb.ReadWriteBucket, k []byte) error {
	err := ns.NestedReadWriteBucket(bucketCredits).Delete(k)
	if err != nil {
		str := "failed to delete credit"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

//CreditIterator允许按顺序迭代
//挖掘的事务。
//
//示例用法：
//
//前缀：=keytxrecord（txshash，block）
//它：=makeCreditIterator（ns，prefix）
//对于它.NEXT（）{
///使用它。
////如有必要，请从it.ck、it.cv中读取其他详细信息
//}
//如果是的话！= nIL{
////句柄错误
//}
//
//如果信用卡由
//非关联交易。要检查此情况：
//
//k：=canonicaloutpoint（&txshash，it.elem.index）
//it.elem.speed=existsrawunminedinput（ns，k）！=零
type creditIterator struct {
c      walletdb.ReadWriteCursor //最终迭代后设置为零
	prefix []byte
	ck     []byte
	cv     []byte
	elem   CreditRecord
	err    error
}

func makeCreditIterator(ns walletdb.ReadWriteBucket, prefix []byte) creditIterator {
	c := ns.NestedReadWriteBucket(bucketCredits).ReadWriteCursor()
	return creditIterator{c: c, prefix: prefix}
}

func makeReadCreditIterator(ns walletdb.ReadBucket, prefix []byte) creditIterator {
	c := ns.NestedReadBucket(bucketCredits).ReadCursor()
	return creditIterator{c: readCursor{c}, prefix: prefix}
}

func (it *creditIterator) readElem() error {
	if len(it.ck) < 72 {
		str := fmt.Sprintf("%s: short key (expected %d bytes, read %d)",
			bucketCredits, 72, len(it.ck))
		return storeError(ErrData, str, nil)
	}
	if len(it.cv) < 9 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketCredits, 9, len(it.cv))
		return storeError(ErrData, str, nil)
	}
	it.elem.Index = byteOrder.Uint32(it.ck[68:72])
	it.elem.Amount = btcutil.Amount(byteOrder.Uint64(it.cv))
	it.elem.Spent = it.cv[8]&(1<<0) != 0
	it.elem.Change = it.cv[8]&(1<<1) != 0
	return nil
}

func (it *creditIterator) next() bool {
	if it.c == nil {
		return false
	}

	if it.ck == nil {
		it.ck, it.cv = it.c.Seek(it.prefix)
	} else {
		it.ck, it.cv = it.c.Next()
	}
	if !bytes.HasPrefix(it.ck, it.prefix) {
		it.c = nil
		return false
	}

	err := it.readElem()
	if err != nil {
		it.err = err
		return false
	}
	return true
}

//未使用的索引记录未使用的已开采信用证的所有输出点
//任何其他挖掘的事务记录（但可能由内存池使用）
//事务处理）。
//
//键使用规范的输出点序列化：
//
//[0:32]事务哈希（32字节）
//[32:36]输出索引（4字节）
//
//值序列化如下：
//
//[0:4]块高度（4字节）
//[4:36]块哈希（32字节）

func valueUnspent(block *Block) []byte {
	v := make([]byte, 36)
	byteOrder.PutUint32(v, uint32(block.Height))
	copy(v[4:36], block.Hash[:])
	return v
}

func putUnspent(ns walletdb.ReadWriteBucket, outPoint *wire.OutPoint, block *Block) error {
	k := canonicalOutPoint(&outPoint.Hash, outPoint.Index)
	v := valueUnspent(block)
	err := ns.NestedReadWriteBucket(bucketUnspent).Put(k, v)
	if err != nil {
		str := "cannot put unspent"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

func putRawUnspent(ns walletdb.ReadWriteBucket, k, v []byte) error {
	err := ns.NestedReadWriteBucket(bucketUnspent).Put(k, v)
	if err != nil {
		str := "cannot put unspent"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

func readUnspentBlock(v []byte, block *Block) error {
	if len(v) < 36 {
		str := "short unspent value"
		return storeError(ErrData, str, nil)
	}
	block.Height = int32(byteOrder.Uint32(v))
	copy(block.Hash[:], v[4:36])
	return nil
}

//existsunspent返回未暂停输出的键和相应的
//信用卡的钥匙。如果没有记录未暂停的输出，则
//信用密钥为零。
func existsUnspent(ns walletdb.ReadBucket, outPoint *wire.OutPoint) (k, credKey []byte) {
	k = canonicalOutPoint(&outPoint.Hash, outPoint.Index)
	credKey = existsRawUnspent(ns, k)
	return k, credKey
}

//existsrawunspent如果记录有输出，则返回信用密钥
//用于未使用的未使用的原始密钥。如果k/v对不存在，则返回nil。
func existsRawUnspent(ns walletdb.ReadBucket, k []byte) (credKey []byte) {
	if len(k) < 36 {
		return nil
	}
	v := ns.NestedReadBucket(bucketUnspent).Get(k)
	if len(v) < 36 {
		return nil
	}
	credKey = make([]byte, 72)
	copy(credKey, k[:32])
	copy(credKey[32:68], v)
	copy(credKey[68:72], k[32:36])
	return credKey
}

func deleteRawUnspent(ns walletdb.ReadWriteBucket, k []byte) error {
	err := ns.NestedReadWriteBucket(bucketUnspent).Delete(k)
	if err != nil {
		str := "failed to delete unspent"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

//所有交易借记（支出贷记的输入）都按如下方式键入：
//
//[0:32]事务哈希（32字节）
//[32:36]块高度（4字节）
//[36:68]块哈希（32字节）
//[68:72]输入索引（4字节）
//
//前68个字节与事务记录的键匹配，可以使用
//作为前缀过滤器，按顺序遍历所有借项。
//
//借方值序列化如下：
//
//[0:8]数量（8字节）
//[8:80]Credits Bucket键（72字节）
//[8:40]事务哈希（32字节）
//[40:44]块高度（4字节）
//[44:76]块哈希（32字节）
//[76:80]输出索引（4字节）

func keyDebit(txHash *chainhash.Hash, index uint32, block *Block) []byte {
	k := make([]byte, 72)
	copy(k, txHash[:])
	byteOrder.PutUint32(k[32:36], uint32(block.Height))
	copy(k[36:68], block.Hash[:])
	byteOrder.PutUint32(k[68:72], index)
	return k
}

func putDebit(ns walletdb.ReadWriteBucket, txHash *chainhash.Hash, index uint32, amount btcutil.Amount, block *Block, credKey []byte) error {
	k := keyDebit(txHash, index, block)

	v := make([]byte, 80)
	byteOrder.PutUint64(v, uint64(amount))
	copy(v[8:80], credKey)

	err := ns.NestedReadWriteBucket(bucketDebits).Put(k, v)
	if err != nil {
		str := fmt.Sprintf("failed to update debit %s input %d",
			txHash, index)
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

func extractRawDebitCreditKey(v []byte) []byte {
	return v[8:80]
}

//existsdebit检查是否存在借方。如果找到，借记和
//返回以前的信用密钥。如果借方不存在，则两个键
//是零。
func existsDebit(ns walletdb.ReadBucket, txHash *chainhash.Hash, index uint32, block *Block) (k, credKey []byte, err error) {
	k = keyDebit(txHash, index, block)
	v := ns.NestedReadBucket(bucketDebits).Get(k)
	if v == nil {
		return nil, nil, nil
	}
	if len(v) < 80 {
		str := fmt.Sprintf("%s: short read (expected 80 bytes, read %v)",
			bucketDebits, len(v))
		return nil, nil, storeError(ErrData, str, nil)
	}
	return k, v[8:80], nil
}

func deleteRawDebit(ns walletdb.ReadWriteBucket, k []byte) error {
	err := ns.NestedReadWriteBucket(bucketDebits).Delete(k)
	if err != nil {
		str := "failed to delete debit"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

//Debititerator允许按顺序迭代
//挖掘的事务。
//
//示例用法：
//
//前缀：=keytxrecord（txshash，block）
//它：=makeDebititerator（ns，prefix）
//对于它.NEXT（）{
///使用它。
////如有必要，请从it.ck、it.cv中读取其他详细信息
//}
//如果是的话！= nIL{
////句柄错误
//}
type debitIterator struct {
c      walletdb.ReadWriteCursor //最终迭代后设置为零
	prefix []byte
	ck     []byte
	cv     []byte
	elem   DebitRecord
	err    error
}

func makeDebitIterator(ns walletdb.ReadWriteBucket, prefix []byte) debitIterator {
	c := ns.NestedReadWriteBucket(bucketDebits).ReadWriteCursor()
	return debitIterator{c: c, prefix: prefix}
}

func makeReadDebitIterator(ns walletdb.ReadBucket, prefix []byte) debitIterator {
	c := ns.NestedReadBucket(bucketDebits).ReadCursor()
	return debitIterator{c: readCursor{c}, prefix: prefix}
}

func (it *debitIterator) readElem() error {
	if len(it.ck) < 72 {
		str := fmt.Sprintf("%s: short key (expected %d bytes, read %d)",
			bucketDebits, 72, len(it.ck))
		return storeError(ErrData, str, nil)
	}
	if len(it.cv) < 80 {
		str := fmt.Sprintf("%s: short read (expected %d bytes, read %d)",
			bucketDebits, 80, len(it.cv))
		return storeError(ErrData, str, nil)
	}
	it.elem.Index = byteOrder.Uint32(it.ck[68:72])
	it.elem.Amount = btcutil.Amount(byteOrder.Uint64(it.cv))
	return nil
}

func (it *debitIterator) next() bool {
	if it.c == nil {
		return false
	}

	if it.ck == nil {
		it.ck, it.cv = it.c.Seek(it.prefix)
	} else {
		it.ck, it.cv = it.c.Next()
	}
	if !bytes.HasPrefix(it.ck, it.prefix) {
		it.c = nil
		return false
	}

	err := it.readElem()
	if err != nil {
		it.err = err
		return false
	}
	return true
}

//所有未链接的事务都保存在由键控的未链接存储桶中。
//事务哈希。该值与挖掘的事务记录的值匹配：
//
//[0:8]接收时间（8字节）
//[8:]序列化事务（可变）

func putRawUnmined(ns walletdb.ReadWriteBucket, k, v []byte) error {
	err := ns.NestedReadWriteBucket(bucketUnmined).Put(k, v)
	if err != nil {
		str := "failed to put unmined record"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

func readRawUnminedHash(k []byte, txHash *chainhash.Hash) error {
	if len(k) < 32 {
		str := "short unmined key"
		return storeError(ErrData, str, nil)
	}
	copy(txHash[:], k)
	return nil
}

func existsRawUnmined(ns walletdb.ReadBucket, k []byte) (v []byte) {
	return ns.NestedReadBucket(bucketUnmined).Get(k)
}

func deleteRawUnmined(ns walletdb.ReadWriteBucket, k []byte) error {
	err := ns.NestedReadWriteBucket(bucketUnmined).Delete(k)
	if err != nil {
		str := "failed to delete unmined record"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

//未限定的事务信用使用规范化序列化格式：
//
//[0:32]事务哈希（32字节）
//[32:36]输出索引（4字节）
//
//该值与挖掘的信用证使用的格式相匹配，但使用的标志是
//从不设置，也不包括可选的借方记录。简化的
//格式是这样的：
//
//[0:8]数量（8字节）
//[8]标志（1字节）
//0x02：改变

func valueUnminedCredit(amount btcutil.Amount, change bool) []byte {
	v := make([]byte, 9)
	byteOrder.PutUint64(v, uint64(amount))
	if change {
		v[8] = 1 << 1
	}
	return v
}

func putRawUnminedCredit(ns walletdb.ReadWriteBucket, k, v []byte) error {
	err := ns.NestedReadWriteBucket(bucketUnminedCredits).Put(k, v)
	if err != nil {
		str := "cannot put unmined credit"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

func fetchRawUnminedCreditIndex(k []byte) (uint32, error) {
	if len(k) < 36 {
		str := "short unmined credit key"
		return 0, storeError(ErrData, str, nil)
	}
	return byteOrder.Uint32(k[32:36]), nil
}

func fetchRawUnminedCreditAmount(v []byte) (btcutil.Amount, error) {
	if len(v) < 9 {
		str := "short unmined credit value"
		return 0, storeError(ErrData, str, nil)
	}
	return btcutil.Amount(byteOrder.Uint64(v)), nil
}

func fetchRawUnminedCreditAmountChange(v []byte) (btcutil.Amount, bool, error) {
	if len(v) < 9 {
		str := "short unmined credit value"
		return 0, false, storeError(ErrData, str, nil)
	}
	amt := btcutil.Amount(byteOrder.Uint64(v))
	change := v[8]&(1<<1) != 0
	return amt, change, nil
}

func existsRawUnminedCredit(ns walletdb.ReadBucket, k []byte) []byte {
	return ns.NestedReadBucket(bucketUnminedCredits).Get(k)
}

func deleteRawUnminedCredit(ns walletdb.ReadWriteBucket, k []byte) error {
	err := ns.NestedReadWriteBucket(bucketUnminedCredits).Delete(k)
	if err != nil {
		str := "failed to delete unmined credit"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

//unlinedcrediterator允许在所有信用证上按顺序进行光标迭代，
//从一个未开采的交易。
//
//示例用法：
//
//它：=makeunlinedcrediterator（ns，txshash）
//对于它.NEXT（）{
////使用it.elem、it.ck和it.cv
////或者，使用它.delete（）删除此k/v对
//}
//如果是的话！= nIL{
////句柄错误
//}
//
//信贷的挥霍并不是出于绩效的考虑（因为
//对于未使用的信用证，需要在另一个bucket中进行另一次查找）。如果这样
//如果需要，可以这样检查：
//
//花费：=existsrawunminedinput（ns，it.ck）！=零
type unminedCreditIterator struct {
	c      walletdb.ReadWriteCursor
	prefix []byte
	ck     []byte
	cv     []byte
	elem   CreditRecord
	err    error
}

type readCursor struct {
	walletdb.ReadCursor
}

func (r readCursor) Delete() error {
	str := "failed to delete current cursor item from read-only cursor"
	return storeError(ErrDatabase, str, walletdb.ErrTxNotWritable)
}

func makeUnminedCreditIterator(ns walletdb.ReadWriteBucket, txHash *chainhash.Hash) unminedCreditIterator {
	c := ns.NestedReadWriteBucket(bucketUnminedCredits).ReadWriteCursor()
	return unminedCreditIterator{c: c, prefix: txHash[:]}
}

func makeReadUnminedCreditIterator(ns walletdb.ReadBucket, txHash *chainhash.Hash) unminedCreditIterator {
	c := ns.NestedReadBucket(bucketUnminedCredits).ReadCursor()
	return unminedCreditIterator{c: readCursor{c}, prefix: txHash[:]}
}

func (it *unminedCreditIterator) readElem() error {
	index, err := fetchRawUnminedCreditIndex(it.ck)
	if err != nil {
		return err
	}
	amount, change, err := fetchRawUnminedCreditAmountChange(it.cv)
	if err != nil {
		return err
	}

	it.elem.Index = index
	it.elem.Amount = amount
	it.elem.Change = change
//故意花不定

	return nil
}

func (it *unminedCreditIterator) next() bool {
	if it.c == nil {
		return false
	}

	if it.ck == nil {
		it.ck, it.cv = it.c.Seek(it.prefix)
	} else {
		it.ck, it.cv = it.c.Next()
	}
	if !bytes.HasPrefix(it.ck, it.prefix) {
		it.c = nil
		return false
	}

	err := it.readElem()
	if err != nil {
		it.err = err
		return false
	}
	return true
}

//在https://github.com/boltdb/bolt/issues/620修复之前不可用。
//func（it*unlinedcrediterator）delete（）错误
//错误：=it.c.delete（）。
//如果犯错！= nIL{
//str：=“删除未满信用失败”
//返回storeError（errDatabase、str、err）
//}
//返回零
//}

func (it *unminedCreditIterator) reposition(txHash *chainhash.Hash, index uint32) {
	it.c.Seek(canonicalOutPoint(txHash, index))
}

//未链接的事务花费的输出点保存在未链接的输入中。
//桶。对于两个已开采的矿藏，此bucket在每个先前花费的输出之间映射
//和未链接的事务，到未链接事务的哈希。
//
//该键按如下方式序列化：
//
//[0:32]事务哈希（32字节）
//[32:36]输出索引（4字节）
//
//值序列化如下：
//
//[0:32]事务哈希（32字节）

//putrawUnminedInput维护具有
//花了一个前哨。桶中的每个条目都由输出点键控
//花了。
func putRawUnminedInput(ns walletdb.ReadWriteBucket, k, v []byte) error {
	spendTxHashes := ns.NestedReadBucket(bucketUnminedInputs).Get(k)
	spendTxHashes = append(spendTxHashes, v...)
	err := ns.NestedReadWriteBucket(bucketUnminedInputs).Put(k, spendTxHashes)
	if err != nil {
		str := "failed to put unmined input"
		return storeError(ErrDatabase, str, err)
	}

	return nil
}

func existsRawUnminedInput(ns walletdb.ReadBucket, k []byte) (v []byte) {
	return ns.NestedReadBucket(bucketUnminedInputs).Get(k)
}

//fetchuminedinputspendtxhash获取
//使用序列化的输出点。
func fetchUnminedInputSpendTxHashes(ns walletdb.ReadBucket, k []byte) []chainhash.Hash {
	rawSpendTxHashes := ns.NestedReadBucket(bucketUnminedInputs).Get(k)
	if rawSpendTxHashes == nil {
		return nil
	}

//每个事务哈希为32个字节。
	spendTxHashes := make([]chainhash.Hash, 0, len(rawSpendTxHashes)/32)
	for len(rawSpendTxHashes) > 0 {
		var spendTxHash chainhash.Hash
		copy(spendTxHash[:], rawSpendTxHashes[:32])
		spendTxHashes = append(spendTxHashes, spendTxHash)
		rawSpendTxHashes = rawSpendTxHashes[32:]
	}

	return spendTxHashes
}

func deleteRawUnminedInput(ns walletdb.ReadWriteBucket, k []byte) error {
	err := ns.NestedReadWriteBucket(bucketUnminedInputs).Delete(k)
	if err != nil {
		str := "failed to delete unmined input"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

//OpenStore从传递的命名空间打开现有的事务存储。
func openStore(ns walletdb.ReadBucket) error {
	version, err := fetchVersion(ns)
	if err != nil {
		return err
	}

	latestVersion := getLatestVersion()
	if version < latestVersion {
		str := fmt.Sprintf("a database upgrade is required to upgrade "+
			"wtxmgr from recorded version %d to the latest version %d",
			version, latestVersion)
		return storeError(ErrNeedsUpgrade, str, nil)
	}

	if version > latestVersion {
		str := fmt.Sprintf("version recorded version %d is newer that "+
			"latest understood version %d", version, latestVersion)
		return storeError(ErrUnknownVersion, str, nil)
	}

	return nil
}

//CreateStore在传递的
//命名空间。如果存储已存在，则返回erralreadyexists。
func createStore(ns walletdb.ReadWriteBucket) error {
//确保命名空间存储桶中当前不存在任何内容。
	ck, cv := ns.ReadCursor().First()
	if ck != nil || cv != nil {
		const str = "namespace is not empty"
		return storeError(ErrAlreadyExists, str, nil)
	}

//编写最新的商店版本。
	if err := putVersion(ns, getLatestVersion()); err != nil {
		return err
	}

//保存商店的创建日期。
	var v [8]byte
	byteOrder.PutUint64(v[:], uint64(time.Now().Unix()))
	err := ns.Put(rootCreateDate, v[:])
	if err != nil {
		str := "failed to store database creation time"
		return storeError(ErrDatabase, str, err)
	}

//写一个零平衡。
	byteOrder.PutUint64(v[:], 0)
	err = ns.Put(rootMinedBalance, v[:])
	if err != nil {
		str := "failed to write zero balance"
		return storeError(ErrDatabase, str, err)
	}

//最后，创建所有必需的后代桶。
	return createBuckets(ns)
}

//创建Buckets创建
//正确履行职责的交易商店。
func createBuckets(ns walletdb.ReadWriteBucket) error {
	if _, err := ns.CreateBucket(bucketBlocks); err != nil {
		str := "failed to create blocks bucket"
		return storeError(ErrDatabase, str, err)
	}
	if _, err := ns.CreateBucket(bucketTxRecords); err != nil {
		str := "failed to create tx records bucket"
		return storeError(ErrDatabase, str, err)
	}
	if _, err := ns.CreateBucket(bucketCredits); err != nil {
		str := "failed to create credits bucket"
		return storeError(ErrDatabase, str, err)
	}
	if _, err := ns.CreateBucket(bucketDebits); err != nil {
		str := "failed to create debits bucket"
		return storeError(ErrDatabase, str, err)
	}
	if _, err := ns.CreateBucket(bucketUnspent); err != nil {
		str := "failed to create unspent bucket"
		return storeError(ErrDatabase, str, err)
	}
	if _, err := ns.CreateBucket(bucketUnmined); err != nil {
		str := "failed to create unmined bucket"
		return storeError(ErrDatabase, str, err)
	}
	if _, err := ns.CreateBucket(bucketUnminedCredits); err != nil {
		str := "failed to create unmined credits bucket"
		return storeError(ErrDatabase, str, err)
	}
	if _, err := ns.CreateBucket(bucketUnminedInputs); err != nil {
		str := "failed to create unmined inputs bucket"
		return storeError(ErrDatabase, str, err)
	}

	return nil
}

//删除存储桶删除
//正确履行职责的交易商店。
func deleteBuckets(ns walletdb.ReadWriteBucket) error {
	if err := ns.DeleteNestedBucket(bucketBlocks); err != nil {
		str := "failed to delete blocks bucket"
		return storeError(ErrDatabase, str, err)
	}
	if err := ns.DeleteNestedBucket(bucketTxRecords); err != nil {
		str := "failed to delete tx records bucket"
		return storeError(ErrDatabase, str, err)
	}
	if err := ns.DeleteNestedBucket(bucketCredits); err != nil {
		str := "failed to delete credits bucket"
		return storeError(ErrDatabase, str, err)
	}
	if err := ns.DeleteNestedBucket(bucketDebits); err != nil {
		str := "failed to delete debits bucket"
		return storeError(ErrDatabase, str, err)
	}
	if err := ns.DeleteNestedBucket(bucketUnspent); err != nil {
		str := "failed to delete unspent bucket"
		return storeError(ErrDatabase, str, err)
	}
	if err := ns.DeleteNestedBucket(bucketUnmined); err != nil {
		str := "failed to delete unmined bucket"
		return storeError(ErrDatabase, str, err)
	}
	if err := ns.DeleteNestedBucket(bucketUnminedCredits); err != nil {
		str := "failed to delete unmined credits bucket"
		return storeError(ErrDatabase, str, err)
	}
	if err := ns.DeleteNestedBucket(bucketUnminedInputs); err != nil {
		str := "failed to delete unmined inputs bucket"
		return storeError(ErrDatabase, str, err)
	}

	return nil
}

//PutVersion修改存储的版本以反映给定的版本
//号码。
func putVersion(ns walletdb.ReadWriteBucket, version uint32) error {
	var v [4]byte
	byteOrder.PutUint32(v[:], version)
	if err := ns.Put(rootVersion, v[:]); err != nil {
		str := "failed to store database version"
		return storeError(ErrDatabase, str, err)
	}

	return nil
}

//fetchversion获取存储的当前版本。
func fetchVersion(ns walletdb.ReadBucket) (uint32, error) {
	v := ns.Get(rootVersion)
	if len(v) != 4 {
		str := "no transaction store exists in namespace"
		return 0, storeError(ErrNoExists, str, nil)
	}

	return byteOrder.Uint32(v), nil
}

func scopedUpdate(db walletdb.DB, namespaceKey []byte, f func(walletdb.ReadWriteBucket) error) error {
	tx, err := db.BeginReadWriteTx()
	if err != nil {
		str := "cannot begin update"
		return storeError(ErrDatabase, str, err)
	}
	err = f(tx.ReadWriteBucket(namespaceKey))
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			const desc = "rollback failed"
			serr, ok := err.(Error)
			if !ok {
//这真的不应该发生。
				return storeError(ErrDatabase, desc, rollbackErr)
			}
			serr.Desc = desc + ": " + serr.Desc
			return serr
		}
		return err
	}
	err = tx.Commit()
	if err != nil {
		str := "commit failed"
		return storeError(ErrDatabase, str, err)
	}
	return nil
}

func scopedView(db walletdb.DB, namespaceKey []byte, f func(walletdb.ReadBucket) error) error {
	tx, err := db.BeginReadTx()
	if err != nil {
		str := "cannot begin view"
		return storeError(ErrDatabase, str, err)
	}
	err = f(tx.ReadBucket(namespaceKey))
	rollbackErr := tx.Rollback()
	if err != nil {
		return err
	}
	if rollbackErr != nil {
		str := "cannot close view"
		return storeError(ErrDatabase, str, rollbackErr)
	}
	return nil
}
