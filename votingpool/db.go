
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package votingpool

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/snacl"
	"github.com/btcsuite/btcwallet/walletdb"
)

//这些常量定义给定加密扩展的序列化长度
//公钥或私钥。
const (
//我们可以这样计算加密扩展密钥长度：
//snacl.overhead==加密开销（16）
//实际基准58扩展密钥长度=（111）
//snacl.noncosize==用于加密的nonce大小（24）
	seriesKeyLength = snacl.Overhead + 111 + snacl.NonceSize
//4个字节版本+1个字节活动+4个字节nkeys+4个字节reqsigs
	seriesMinSerial = 4 + 1 + 4 + 4
//15是投票池中的最大密钥数，每个密钥对应1个
//公钥和私钥
	seriesMaxSerial = seriesMinSerial + 15*seriesKeyLength*2
//我们支持的序列化系列的版本
	seriesMaxVersion = 1
)

var (
	usedAddrsBucketName   = []byte("usedaddrs")
	seriesBucketName      = []byte("series")
	withdrawalsBucketName = []byte("withdrawals")
//表示不存在的私钥的字符串
	seriesNullPrivKey = [seriesKeyLength]byte{}
)

type dbSeriesRow struct {
	version           uint32
	active            bool
	reqSigs           uint32
	pubKeysEncrypted  [][]byte
	privKeysEncrypted [][]byte
}

type dbWithdrawalRow struct {
	Requests      []dbOutputRequest
	StartAddress  dbWithdrawalAddress
	ChangeStart   dbChangeAddress
	LastSeriesID  uint32
	DustThreshold btcutil.Amount
	Status        dbWithdrawalStatus
}

type dbWithdrawalAddress struct {
	SeriesID uint32
	Branch   Branch
	Index    Index
}

type dbChangeAddress struct {
	SeriesID uint32
	Index    Index
}

type dbOutputRequest struct {
	Addr        string
	Amount      btcutil.Amount
	Server      string
	Transaction uint32
}

type dbWithdrawalOutput struct {
//我们在这里存储发件ID，因为我们需要查找
//反序列化时，dbdrawtalrow中对应的dboutputRequest。
	OutBailmentID OutBailmentID
	Status        outputStatus
	Outpoints     []dbOutBailmentOutpoint
}

type dbOutBailmentOutpoint struct {
	Ntxid  Ntxid
	Index  uint32
	Amount btcutil.Amount
}

type dbChangeAwareTx struct {
	SerializedMsgTx []byte
	ChangeIdx       int32
}

type dbWithdrawalStatus struct {
	NextInputAddr  dbWithdrawalAddress
	NextChangeAddr dbChangeAddress
	Fees           btcutil.Amount
	Outputs        map[OutBailmentID]dbWithdrawalOutput
	Sigs           map[Ntxid]TxSigs
	Transactions   map[Ntxid]dbChangeAwareTx
}

//getusedaddrbacketid返回给定序列的已用地址存储桶ID
//分支。它的形式是seriesid:branch。
func getUsedAddrBucketID(seriesID uint32, branch Branch) []byte {
	var bucketID [9]byte
	binary.LittleEndian.PutUint32(bucketID[0:4], seriesID)
	bucketID[4] = ':'
	binary.LittleEndian.PutUint32(bucketID[5:9], uint32(branch))
	return bucketID[:]
}

//putuseDaddrHash将一个条目（key==index，value==encryptedHash）添加到
//地址给定池、系列和分支的存储桶。
func putUsedAddrHash(ns walletdb.ReadWriteBucket, poolID []byte, seriesID uint32, branch Branch,
	index Index, encryptedHash []byte) error {

	usedAddrs := ns.NestedReadWriteBucket(poolID).NestedReadWriteBucket(usedAddrsBucketName)
	bucket, err := usedAddrs.CreateBucketIfNotExists(getUsedAddrBucketID(seriesID, branch))
	if err != nil {
		return newError(ErrDatabase, "failed to store used address hash", err)
	}
	return bucket.Put(uint32ToBytes(uint32(index)), encryptedHash)
}

//getusedaddrhash返回使用的
//地址给定池、系列和分支的存储桶。
func getUsedAddrHash(ns walletdb.ReadBucket, poolID []byte, seriesID uint32, branch Branch,
	index Index) []byte {

	usedAddrs := ns.NestedReadBucket(poolID).NestedReadBucket(usedAddrsBucketName)
	bucket := usedAddrs.NestedReadBucket(getUsedAddrBucketID(seriesID, branch))
	if bucket == nil {
		return nil
	}
	return bucket.Get(uint32ToBytes(uint32(index)))
}

//getmaxusedidx从used addresses存储桶返回使用的最高索引
//指定池、系列和分支的。
func getMaxUsedIdx(ns walletdb.ReadBucket, poolID []byte, seriesID uint32, branch Branch) (Index, error) {
	maxIdx := Index(0)
	usedAddrs := ns.NestedReadBucket(poolID).NestedReadBucket(usedAddrsBucketName)
	bucket := usedAddrs.NestedReadBucket(getUsedAddrBucketID(seriesID, branch))
	if bucket == nil {
		return maxIdx, nil
	}
//Fixme：这远不是最佳的，应该通过存储来优化
//数据库中的一个单独的键，每个键使用的IDX最高。
//系列/分支，或者通过执行大间隙线性正向搜索+
//二进制向后搜索（例如，检查1000000、2000000…。直到它
//不存在，然后使用二进制搜索查找最大值
//发现边界）。
	err := bucket.ForEach(
		func(k, v []byte) error {
			idx := Index(bytesToUint32(k))
			if idx > maxIdx {
				maxIdx = idx
			}
			return nil
		})
	if err != nil {
		return Index(0), newError(ErrDatabase, "failed to get highest idx of used addresses", err)
	}
	return maxIdx, nil
}

//
//在投票池ID和其中的其他两个存储桶之后，存储序列和
//已使用该池的地址。
func putPool(ns walletdb.ReadWriteBucket, poolID []byte) error {
	poolBucket, err := ns.CreateBucket(poolID)
	if err != nil {
		return newError(ErrDatabase, fmt.Sprintf("cannot create pool %v", poolID), err)
	}
	_, err = poolBucket.CreateBucket(seriesBucketName)
	if err != nil {
		return newError(ErrDatabase, fmt.Sprintf("cannot create series bucket for pool %v",
			poolID), err)
	}
	_, err = poolBucket.CreateBucket(usedAddrsBucketName)
	if err != nil {
		return newError(ErrDatabase, fmt.Sprintf("cannot create used addrs bucket for pool %v",
			poolID), err)
	}
	_, err = poolBucket.CreateBucket(withdrawalsBucketName)
	if err != nil {
		return newError(
			ErrDatabase, fmt.Sprintf("cannot create withdrawals bucket for pool %v", poolID), err)
	}
	return nil
}

//
//桶，由ID键控。
func loadAllSeries(ns walletdb.ReadBucket, poolID []byte) (map[uint32]*dbSeriesRow, error) {
	bucket := ns.NestedReadBucket(poolID).NestedReadBucket(seriesBucketName)
	allSeries := make(map[uint32]*dbSeriesRow)
	err := bucket.ForEach(
		func(k, v []byte) error {
			seriesID := bytesToUint32(k)
			series, err := deserializeSeriesRow(v)
			if err != nil {
				return err
			}
			allSeries[seriesID] = series
			return nil
		})
	if err != nil {
		return nil, err
	}
	return allSeries, nil
}

//existspool检查是否存在以给定的
//投票池ID。
func existsPool(ns walletdb.ReadBucket, poolID []byte) bool {
	bucket := ns.NestedReadBucket(poolID)
	return bucket != nil
}

//PutSeries将给定的序列存储在以下面的名称命名的投票池存储桶中
//池。投票池存储桶不需要预先创建。
func putSeries(ns walletdb.ReadWriteBucket, poolID []byte, version, ID uint32, active bool, reqSigs uint32, pubKeysEncrypted, privKeysEncrypted [][]byte) error {
	row := &dbSeriesRow{
		version:           version,
		active:            active,
		reqSigs:           reqSigs,
		pubKeysEncrypted:  pubKeysEncrypted,
		privKeysEncrypted: privKeysEncrypted,
	}
	return putSeriesRow(ns, poolID, ID, row)
}

//putseriesrow将给定的序列行存储在名为
//后池。不需要创建投票池存储桶
//事先。
func putSeriesRow(ns walletdb.ReadWriteBucket, poolID []byte, ID uint32, row *dbSeriesRow) error {
	bucket, err := ns.CreateBucketIfNotExists(poolID)
	if err != nil {
		str := fmt.Sprintf("cannot create bucket %v", poolID)
		return newError(ErrDatabase, str, err)
	}
	bucket, err = bucket.CreateBucketIfNotExists(seriesBucketName)
	if err != nil {
		return err
	}
	serialized, err := serializeSeriesRow(row)
	if err != nil {
		return err
	}
	err = bucket.Put(uint32ToBytes(ID), serialized)
	if err != nil {
		str := fmt.Sprintf("cannot put series %v into bucket %v", serialized, poolID)
		return newError(ErrDatabase, str, err)
	}
	return nil
}

//反序列化riesRow将序列存储反序列化为dbseriesRow结构。
func deserializeSeriesRow(serializedSeries []byte) (*dbSeriesRow, error) {
//序列化序列格式为：
//<version><active><reqsigs><nkeys><pubkey1><privkey1>…<pubkeyn><privkeyn>
//
//4字节版本+1字节活动+4字节请求信号+4字节键
//+serieskeylength*2*nkeys（1个用于priv，1个用于pub）

//鉴于上述情况，序列化序列的长度应为
//至少是常数的长度。
	if len(serializedSeries) < seriesMinSerial {
		str := fmt.Sprintf("serialized series is too short: %v", serializedSeries)
		return nil, newError(ErrSeriesSerialization, str, nil)
	}

//公共密钥的最大数目为15，公共密钥的最大数目相同。
//这给了我们一个上界。
	if len(serializedSeries) > seriesMaxSerial {
		str := fmt.Sprintf("serialized series is too long: %v", serializedSeries)
		return nil, newError(ErrSeriesSerialization, str, nil)
	}

//跟踪下一组要反序列化的字节的位置。
	current := 0
	row := dbSeriesRow{}

	row.version = bytesToUint32(serializedSeries[current : current+4])
	if row.version > seriesMaxVersion {
		str := fmt.Sprintf("deserialization supports up to version %v not %v",
			seriesMaxVersion, row.version)
		return nil, newError(ErrSeriesVersion, str, nil)
	}
	current += 4

	row.active = serializedSeries[current] == 0x01
	current++

	row.reqSigs = bytesToUint32(serializedSeries[current : current+4])
	current += 4

	nKeys := bytesToUint32(serializedSeries[current : current+4])
	current += 4

//检查我们是否有正确的字节数。
	if len(serializedSeries) < current+int(nKeys)*seriesKeyLength*2 {
		str := fmt.Sprintf("serialized series has not enough data: %v", serializedSeries)
		return nil, newError(ErrSeriesSerialization, str, nil)
	} else if len(serializedSeries) > current+int(nKeys)*seriesKeyLength*2 {
		str := fmt.Sprintf("serialized series has too much data: %v", serializedSeries)
		return nil, newError(ErrSeriesSerialization, str, nil)
	}

//反序列化pubkey/privkey对。
	row.pubKeysEncrypted = make([][]byte, nKeys)
	row.privKeysEncrypted = make([][]byte, nKeys)
	for i := 0; i < int(nKeys); i++ {
		pubKeyStart := current + seriesKeyLength*i*2
		pubKeyEnd := current + seriesKeyLength*i*2 + seriesKeyLength
		privKeyEnd := current + seriesKeyLength*(i+1)*2
		row.pubKeysEncrypted[i] = serializedSeries[pubKeyStart:pubKeyEnd]
		privKeyEncrypted := serializedSeries[pubKeyEnd:privKeyEnd]
		if bytes.Equal(privKeyEncrypted, seriesNullPrivKey[:]) {
			row.privKeysEncrypted[i] = nil
		} else {
			row.privKeysEncrypted[i] = privKeyEncrypted
		}
	}

	return &row, nil
}

//SerializesRow将DBSeriesRow结构序列化为存储格式。
func serializeSeriesRow(row *dbSeriesRow) ([]byte, error) {
//序列化序列格式为：
//<version><active><reqsigs><nkeys><pubkey1><privkey1>…<pubkeyn><privkeyn>
//
//4字节版本+1字节活动+4字节请求信号+4字节键
//+serieskeylength*2*nkeys（1个用于priv，1个用于pub）
	serializedLen := 4 + 1 + 4 + 4 + (seriesKeyLength * 2 * len(row.pubKeysEncrypted))

	if len(row.privKeysEncrypted) != 0 &&
		len(row.pubKeysEncrypted) != len(row.privKeysEncrypted) {
		str := fmt.Sprintf("different # of pub (%v) and priv (%v) keys",
			len(row.pubKeysEncrypted), len(row.privKeysEncrypted))
		return nil, newError(ErrSeriesSerialization, str, nil)
	}

	if row.version > seriesMaxVersion {
		str := fmt.Sprintf("serialization supports up to version %v, not %v",
			seriesMaxVersion, row.version)
		return nil, newError(ErrSeriesVersion, str, nil)
	}

	serialized := make([]byte, 0, serializedLen)
	serialized = append(serialized, uint32ToBytes(row.version)...)
	if row.active {
		serialized = append(serialized, 0x01)
	} else {
		serialized = append(serialized, 0x00)
	}
	serialized = append(serialized, uint32ToBytes(row.reqSigs)...)
	nKeys := uint32(len(row.pubKeysEncrypted))
	serialized = append(serialized, uint32ToBytes(nKeys)...)

	var privKeyEncrypted []byte
	for i, pubKeyEncrypted := range row.pubKeysEncrypted {
//检查加密长度是否正确
		if len(pubKeyEncrypted) != seriesKeyLength {
			str := fmt.Sprintf("wrong length of Encrypted Public Key: %v",
				pubKeyEncrypted)
			return nil, newError(ErrSeriesSerialization, str, nil)
		}
		serialized = append(serialized, pubKeyEncrypted...)

		if len(row.privKeysEncrypted) == 0 {
			privKeyEncrypted = seriesNullPrivKey[:]
		} else {
			privKeyEncrypted = row.privKeysEncrypted[i]
		}

		if privKeyEncrypted == nil {
			serialized = append(serialized, seriesNullPrivKey[:]...)
		} else if len(privKeyEncrypted) != seriesKeyLength {
			str := fmt.Sprintf("wrong length of Encrypted Private Key: %v",
				len(privKeyEncrypted))
			return nil, newError(ErrSeriesSerialization, str, nil)
		} else {
			serialized = append(serialized, privKeyEncrypted...)
		}
	}
	return serialized, nil
}

//serializedrawing构造一个dbdrawintalrow并将其序列化（使用
//编码/gob），以便将其存储在数据库中。
func serializeWithdrawal(requests []OutputRequest, startAddress WithdrawalAddress,
	lastSeriesID uint32, changeStart ChangeAddress, dustThreshold btcutil.Amount,
	status WithdrawalStatus) ([]byte, error) {

	dbStartAddr := dbWithdrawalAddress{
		SeriesID: startAddress.SeriesID(),
		Branch:   startAddress.Branch(),
		Index:    startAddress.Index(),
	}
	dbChangeStart := dbChangeAddress{
		SeriesID: startAddress.SeriesID(),
		Index:    startAddress.Index(),
	}
	dbRequests := make([]dbOutputRequest, len(requests))
	for i, request := range requests {
		dbRequests[i] = dbOutputRequest{
			Addr:        request.Address.EncodeAddress(),
			Amount:      request.Amount,
			Server:      request.Server,
			Transaction: request.Transaction,
		}
	}
	dbOutputs := make(map[OutBailmentID]dbWithdrawalOutput, len(status.outputs))
	for oid, output := range status.outputs {
		dbOutpoints := make([]dbOutBailmentOutpoint, len(output.outpoints))
		for i, outpoint := range output.outpoints {
			dbOutpoints[i] = dbOutBailmentOutpoint{
				Ntxid:  outpoint.ntxid,
				Index:  outpoint.index,
				Amount: outpoint.amount,
			}
		}
		dbOutputs[oid] = dbWithdrawalOutput{
			OutBailmentID: output.request.outBailmentID(),
			Status:        output.status,
			Outpoints:     dbOutpoints,
		}
	}
	dbTransactions := make(map[Ntxid]dbChangeAwareTx, len(status.transactions))
	for ntxid, tx := range status.transactions {
		var buf bytes.Buffer
		buf.Grow(tx.SerializeSize())
		if err := tx.Serialize(&buf); err != nil {
			return nil, err
		}
		dbTransactions[ntxid] = dbChangeAwareTx{
			SerializedMsgTx: buf.Bytes(),
			ChangeIdx:       tx.changeIdx,
		}
	}
	nextChange := status.nextChangeAddr
	dbStatus := dbWithdrawalStatus{
		NextChangeAddr: dbChangeAddress{
			SeriesID: nextChange.seriesID,
			Index:    nextChange.index,
		},
		Fees:         status.fees,
		Outputs:      dbOutputs,
		Sigs:         status.sigs,
		Transactions: dbTransactions,
	}
	row := dbWithdrawalRow{
		Requests:      dbRequests,
		StartAddress:  dbStartAddr,
		LastSeriesID:  lastSeriesID,
		ChangeStart:   dbChangeStart,
		DustThreshold: dustThreshold,
		Status:        dbStatus,
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(row); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

//deserializewithdrawal将给定的字节片反序列化为dbdrawtalrow，
//将其转换为取款信息并返回。此函数必须运行
//地址管理器解锁。
func deserializeWithdrawal(p *Pool, ns, addrmgrNs walletdb.ReadBucket, serialized []byte) (*withdrawalInfo, error) {
	var row dbWithdrawalRow
	if err := gob.NewDecoder(bytes.NewReader(serialized)).Decode(&row); err != nil {
		return nil, newError(ErrWithdrawalStorage, "cannot deserialize withdrawal information",
			err)
	}
	wInfo := &withdrawalInfo{
		lastSeriesID:  row.LastSeriesID,
		dustThreshold: row.DustThreshold,
	}
	chainParams := p.Manager().ChainParams()
	wInfo.requests = make([]OutputRequest, len(row.Requests))
//按发件箱ID索引的请求的映射；需要填充
//退出状态。稍后输出。
	requestsByOID := make(map[OutBailmentID]OutputRequest)
	for i, req := range row.Requests {
		addr, err := btcutil.DecodeAddress(req.Addr, chainParams)
		if err != nil {
			return nil, newError(ErrWithdrawalStorage,
				"cannot deserialize addr for requested output", err)
		}
		pkScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return nil, newError(ErrWithdrawalStorage, "invalid addr for requested output", err)
		}
		request := OutputRequest{
			Address:     addr,
			Amount:      req.Amount,
			PkScript:    pkScript,
			Server:      req.Server,
			Transaction: req.Transaction,
		}
		wInfo.requests[i] = request
		requestsByOID[request.outBailmentID()] = request
	}
	startAddr := row.StartAddress
	wAddr, err := p.WithdrawalAddress(ns, addrmgrNs, startAddr.SeriesID, startAddr.Branch, startAddr.Index)
	if err != nil {
		return nil, newError(ErrWithdrawalStorage, "cannot deserialize startAddress", err)
	}
	wInfo.startAddress = *wAddr

	cAddr, err := p.ChangeAddress(row.ChangeStart.SeriesID, row.ChangeStart.Index)
	if err != nil {
		return nil, newError(ErrWithdrawalStorage, "cannot deserialize changeStart", err)
	}
	wInfo.changeStart = *cAddr

//TODO:复制到row.status.nextinputaddr。未完成，因为开始撤消
//还没有更新。
	nextChangeAddr := row.Status.NextChangeAddr
	cAddr, err = p.ChangeAddress(nextChangeAddr.SeriesID, nextChangeAddr.Index)
	if err != nil {
		return nil, newError(ErrWithdrawalStorage,
			"cannot deserialize nextChangeAddress for withdrawal", err)
	}
	wInfo.status = WithdrawalStatus{
		nextChangeAddr: *cAddr,
		fees:           row.Status.Fees,
		outputs:        make(map[OutBailmentID]*WithdrawalOutput, len(row.Status.Outputs)),
		sigs:           row.Status.Sigs,
		transactions:   make(map[Ntxid]changeAwareTx, len(row.Status.Transactions)),
	}
	for oid, output := range row.Status.Outputs {
		outpoints := make([]OutBailmentOutpoint, len(output.Outpoints))
		for i, outpoint := range output.Outpoints {
			outpoints[i] = OutBailmentOutpoint{
				ntxid:  outpoint.Ntxid,
				index:  outpoint.Index,
				amount: outpoint.Amount,
			}
		}
		wInfo.status.outputs[oid] = &WithdrawalOutput{
			request:   requestsByOID[output.OutBailmentID],
			status:    output.Status,
			outpoints: outpoints,
		}
	}
	for ntxid, tx := range row.Status.Transactions {
		var msgtx wire.MsgTx
		if err := msgtx.Deserialize(bytes.NewBuffer(tx.SerializedMsgTx)); err != nil {
			return nil, newError(ErrWithdrawalStorage, "cannot deserialize transaction", err)
		}
		wInfo.status.transactions[ntxid] = changeAwareTx{
			MsgTx:     &msgtx,
			changeIdx: tx.ChangeIdx,
		}
	}
	return wInfo, nil
}

func putWithdrawal(ns walletdb.ReadWriteBucket, poolID []byte, roundID uint32, serialized []byte) error {
	bucket := ns.NestedReadWriteBucket(poolID)
	return bucket.Put(uint32ToBytes(roundID), serialized)
}

func getWithdrawal(ns walletdb.ReadBucket, poolID []byte, roundID uint32) []byte {
	bucket := ns.NestedReadBucket(poolID)
	return bucket.Get(uint32ToBytes(roundID))
}

//uint32tobytes将32位无符号整数转换为4字节片
//小尾数顺序：1->[1 0 0 0]。
func uint32ToBytes(number uint32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, number)
	return buf
}

//bytestouint32将4字节的片按小尾数顺序转换为32位
//无符号整数：[1 0 0 0]->1。
func bytesToUint32(encoded []byte) uint32 {
	return binary.LittleEndian.Uint32(encoded)
}
