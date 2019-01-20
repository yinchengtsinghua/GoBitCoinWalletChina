
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2017 BTCSuite开发者
//
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package waddrmgr

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcwallet/walletdb"
)

var (
//
	LatestMgrVersion = getLatestVersion()

//
//
	latestMgrVersion = LatestMgrVersion
)

//
//
//
type ObtainUserInputFunc func() ([]byte, error)

//
//
//对于从托管事务或其他部分返回的潜在错误很有用
//在walletdb数据库中。
func maybeConvertDbError(err error) error {
//当错误已经是管理器错误时，只需返回它。
	if _, ok := err.(ManagerError); ok {
		return err
	}

	return managerError(ErrDatabase, err.Error(), err)
}

//SyncStatus表示存储在
//数据库。
type syncStatus uint8

//这些常量定义了各种支持的同步状态类型。
//
//注：这些目前尚未使用，但正在为可能性进行定义。
//支持每个地址的同步状态。
const (
ssNone    syncStatus = 0 //不是IOTA，因为它们需要为DB稳定
	ssPartial syncStatus = 1
	ssFull    syncStatus = 2
)

//AddressType表示存储在数据库中的地址类型。
type addressType uint8

//这些常量定义了各种支持的地址类型。
const (
	adtChain  addressType = 0
adtImport addressType = 1 //不是IOTA，因为它们需要为DB稳定
	adtScript addressType = 2
)

//
type accountType uint8

//
const (
//
//
//
accountDefault accountType = 0 //
)

//
type dbAccountRow struct {
	acctType accountType
rawData  []byte //
}

//
//
type dbDefaultAccountRow struct {
	dbAccountRow
	pubKeyEncrypted   []byte
	privKeyEncrypted  []byte
	nextExternalIndex uint32
	nextInternalIndex uint32
	name              string
}

//
//数据库。
type dbAddressRow struct {
	addrType   addressType
	account    uint32
	addTime    uint64
	syncStatus syncStatus
rawData    []byte //
}

//
//
type dbChainAddressRow struct {
	dbAddressRow
	branch uint32
	index  uint32
}

//
//
type dbImportedAddressRow struct {
	dbAddressRow
	encryptedPubKey  []byte
	encryptedPrivKey []byte
}

//
//
type dbScriptAddressRow struct {
	dbAddressRow
	encryptedHash   []byte
	encryptedScript []byte
}

//各种数据库字段的键名。
var (
//
	nullVal = []byte{0}

//

//scopeSchemaBucket is the name of the bucket that maps a particular
//
//
	scopeSchemaBucketName = []byte("scope-schema")

//
//
//will house a scoped address manager. All buckets below are a child
//这个桶：
//
//scopebucket->scope->acctbucket
//scopebucket->scope->addrbucket
//
//
//
//
//
//
//
//
	scopeBucketName = []byte("scope")

//
//
//
	coinTypePrivKeyName = []byte("ctpriv")

//cointypeprivkeyname是特定范围内密钥的名称
//存储加密的cointype公钥的存储桶。每个范围
//将有自己的一套硬币型的公共钥匙。
	coinTypePubKeyName = []byte("ctpub")

//
//
//与帐户有关。
	acctBucketName = []byte("acct")

//addrbucketname是存储
//PubKey哈希到地址类型。这将用于快速确定
//如果某个地址在我们的控制之下。
	addrBucketName = []byte("addr")

//addracctidxbucketname用于索引中的帐户地址条目
//此索引可以映射：
//*地址哈希=>帐户ID
//*账户bucket->addr hash=>null
//
//要获取地址的帐户，请使用
//地址散列。
//
//要获取帐户的所有地址，请获取帐户存储桶，
//循环访问键并从addr获取地址行
//桶。
//
//每次创建地址时都需要更新索引，例如
//
	addrAcctIdxBucketName = []byte("addracctidx")

//acctnameidxbucketname用于创建映射帐户的索引
//
//
//
//
	acctNameIdxBucketName = []byte("acctnameidx")

//
//
//
//
//
	acctIDIdxBucketName = []byte("acctididx")

//
//
	usedAddrBucketName = []byte("usedaddrs")

//
//
	metaBucketName = []byte("meta")

//
//在经理中
	lastAccountName = []byte("lastaccount")

//mainBucketname是存储加密数据的存储桶的名称。
//加密所有其他生成的密钥的密钥，仅监视
//标志，主私钥（加密），主HD私钥
//（加密），以及版本信息。
	mainBucketName = []byte("main")

//masterhdprivname是存储主hd的密钥的名称
//私钥。此密钥使用主私有加密进行加密
//
	masterHDPrivName = []byte("mhdpriv")

//masterhdpubname是存储主HD的密钥的名称
//公钥。此密钥使用主公共加密进行加密
//加密密钥。它位于主桶下面。
	masterHDPubName = []byte("mhdpub")

//SyncBucketname是存储当前
//
	syncBucketName = []byte("sync")

//
	mgrVersionName    = []byte("mgrver")
	mgrCreateDateName = []byte("mgrcreated")

//
	masterPrivKeyName   = []byte("mpriv")
	masterPubKeyName    = []byte("mpub")
	cryptoPrivKeyName   = []byte("cpriv")
	cryptoPubKeyName    = []byte("cpub")
	cryptoScriptKeyName = []byte("cscript")
	watchingOnlyName    = []byte("watchonly")

//
	syncedToName              = []byte("syncedto")
	startBlockName            = []byte("startblock")
	birthdayName              = []byte("birthday")
	birthdayBlockName         = []byte("birthdayblock")
	birthdayBlockVerifiedName = []byte("birthdayblockverified")
)

//uint32tobytes将32位无符号整数转换为4字节片
//小尾数顺序：1->[1 0 0 0]。
func uint32ToBytes(number uint32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, number)
	return buf
}

//uint64目标字节将64位无符号整数转换为8字节片
//
func uint64ToBytes(number uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, number)
	return buf
}

//stringtobytes将字符串转换为可变长度的字节片
//
func stringToBytes(s string) []byte {
//序列化格式为：
//
//
//4字节字符串大小+字符串
	size := len(s)
	buf := make([]byte, 4+size)
	copy(buf[0:4], uint32ToBytes(uint32(size)))
	copy(buf[4:4+size], s)
	return buf
}

//
const scopeKeySize = 8

//
//
//在下面
func scopeToBytes(scope *KeyScope) [scopeKeySize]byte {
	var scopeBytes [scopeKeySize]byte
	binary.LittleEndian.PutUint32(scopeBytes[:], scope.Purpose)
	binary.LittleEndian.PutUint32(scopeBytes[4:], scope.Coin)

	return scopeBytes
}

//
//
func scopeFromBytes(scopeBytes []byte) KeyScope {
	return KeyScope{
		Purpose: binary.LittleEndian.Uint32(scopeBytes[:]),
		Coin:    binary.LittleEndian.Uint32(scopeBytes[4:]),
	}
}

//
//
func scopeSchemaToBytes(schema *ScopeAddrSchema) []byte {
	var schemaBytes [2]byte
	schemaBytes[0] = byte(schema.InternalAddrType)
	schemaBytes[1] = byte(schema.ExternalAddrType)

	return schemaBytes[:]
}

//ScopeSchemaFromBytes从
//
func scopeSchemaFromBytes(schemaBytes []byte) *ScopeAddrSchema {
	return &ScopeAddrSchema{
		InternalAddrType: AddressType(schemaBytes[0]),
		ExternalAddrType: AddressType(schemaBytes[1]),
	}
}

//
//
//
func fetchScopeAddrSchema(ns walletdb.ReadBucket,
	scope *KeyScope) (*ScopeAddrSchema, error) {

	schemaBucket := ns.NestedReadBucket(scopeSchemaBucketName)
	if schemaBucket == nil {
		str := fmt.Sprintf("unable to find scope schema bucket")
		return nil, managerError(ErrScopeNotFound, str, nil)
	}

	scopeKey := scopeToBytes(scope)
	schemaBytes := schemaBucket.Get(scopeKey[:])
	if schemaBytes == nil {
		str := fmt.Sprintf("unable to find scope %v", scope)
		return nil, managerError(ErrScopeNotFound, str, nil)
	}

	return scopeSchemaFromBytes(schemaBytes), nil
}

//
//
func putScopeAddrTypes(ns walletdb.ReadWriteBucket, scope *KeyScope,
	schema *ScopeAddrSchema) error {

	scopeSchemaBucket := ns.NestedReadWriteBucket(scopeSchemaBucketName)
	if scopeSchemaBucket == nil {
		str := fmt.Sprintf("unable to find scope schema bucket")
		return managerError(ErrScopeNotFound, str, nil)
	}

	scopeKey := scopeToBytes(scope)
	schemaBytes := scopeSchemaToBytes(schema)
	return scopeSchemaBucket.Put(scopeKey[:], schemaBytes)
}

func fetchReadScopeBucket(ns walletdb.ReadBucket, scope *KeyScope) (walletdb.ReadBucket, error) {
	rootScopeBucket := ns.NestedReadBucket(scopeBucketName)

	scopeKey := scopeToBytes(scope)
	scopedBucket := rootScopeBucket.NestedReadBucket(scopeKey[:])
	if scopedBucket == nil {
		str := fmt.Sprintf("unable to find scope %v", scope)
		return nil, managerError(ErrScopeNotFound, str, nil)
	}

	return scopedBucket, nil
}

func fetchWriteScopeBucket(ns walletdb.ReadWriteBucket,
	scope *KeyScope) (walletdb.ReadWriteBucket, error) {

	rootScopeBucket := ns.NestedReadWriteBucket(scopeBucketName)

	scopeKey := scopeToBytes(scope)
	scopedBucket := rootScopeBucket.NestedReadWriteBucket(scopeKey[:])
	if scopedBucket == nil {
		str := fmt.Sprintf("unable to find scope %v", scope)
		return nil, managerError(ErrScopeNotFound, str, nil)
	}

	return scopedBucket, nil
}

//
func fetchManagerVersion(ns walletdb.ReadBucket) (uint32, error) {
	mainBucket := ns.NestedReadBucket(mainBucketName)
	verBytes := mainBucket.Get(mgrVersionName)
	if verBytes == nil {
		str := "required version number not stored in database"
		return 0, managerError(ErrDatabase, str, nil)
	}
	version := binary.LittleEndian.Uint32(verBytes)
	return version, nil
}

//
func putManagerVersion(ns walletdb.ReadWriteBucket, version uint32) error {
	bucket := ns.NestedReadWriteBucket(mainBucketName)

	verBytes := uint32ToBytes(version)
	err := bucket.Put(mgrVersionName, verBytes)
	if err != nil {
		str := "failed to store version"
		return managerError(ErrDatabase, str, err)
	}
	return nil
}

//
//
//
//
func fetchMasterKeyParams(ns walletdb.ReadBucket) ([]byte, []byte, error) {
	bucket := ns.NestedReadBucket(mainBucketName)

//
	val := bucket.Get(masterPubKeyName)
	if val == nil {
		str := "required master public key parameters not stored in " +
			"database"
		return nil, nil, managerError(ErrDatabase, str, nil)
	}
	pubParams := make([]byte, len(val))
	copy(pubParams, val)

//
	var privParams []byte
	val = bucket.Get(masterPrivKeyName)
	if val != nil {
		privParams = make([]byte, len(val))
		copy(privParams, val)
	}

	return pubParams, privParams, nil
}

//
//数据库。任何一个参数都不能为零，在这种情况下，没有值为
//
func putMasterKeyParams(ns walletdb.ReadWriteBucket, pubParams, privParams []byte) error {
	bucket := ns.NestedReadWriteBucket(mainBucketName)

	if privParams != nil {
		err := bucket.Put(masterPrivKeyName, privParams)
		if err != nil {
			str := "failed to store master private key parameters"
			return managerError(ErrDatabase, str, err)
		}
	}

	if pubParams != nil {
		err := bucket.Put(masterPubKeyName, pubParams)
		if err != nil {
			str := "failed to store master public key parameters"
			return managerError(ErrDatabase, str, err)
		}
	}

	return nil
}

//
//为所有帐户派生扩展密钥。每个CoinType键都是
//
func fetchCoinTypeKeys(ns walletdb.ReadBucket, scope *KeyScope) ([]byte, []byte, error) {
	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return nil, nil, err
	}

	coinTypePubKeyEnc := scopedBucket.Get(coinTypePubKeyName)
	if coinTypePubKeyEnc == nil {
		str := "required encrypted cointype public key not stored in database"
		return nil, nil, managerError(ErrDatabase, str, nil)
	}

	coinTypePrivKeyEnc := scopedBucket.Get(coinTypePrivKeyName)
	if coinTypePrivKeyEnc == nil {
		str := "required encrypted cointype private key not stored in database"
		return nil, nil, managerError(ErrDatabase, str, nil)
	}

	return coinTypePubKeyEnc, coinTypePrivKeyEnc, nil
}

//
//
//
//
func putCoinTypeKeys(ns walletdb.ReadWriteBucket, scope *KeyScope,
	coinTypePubKeyEnc []byte, coinTypePrivKeyEnc []byte) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	if coinTypePubKeyEnc != nil {
		err := scopedBucket.Put(coinTypePubKeyName, coinTypePubKeyEnc)
		if err != nil {
			str := "failed to store encrypted cointype public key"
			return managerError(ErrDatabase, str, err)
		}
	}

	if coinTypePrivKeyEnc != nil {
		err := scopedBucket.Put(coinTypePrivKeyName, coinTypePrivKeyEnc)
		if err != nil {
			str := "failed to store encrypted cointype private key"
			return managerError(ErrDatabase, str, err)
		}
	}

	return nil
}

//
//
//
func putMasterHDKeys(ns walletdb.ReadWriteBucket, masterHDPrivEnc, masterHDPubEnc []byte) error {
//因为这是根管理器的密钥，所以我们不需要获取任何
//特殊范围，可直接插入主桶内。
	bucket := ns.NestedReadWriteBucket(mainBucketName)

//既然我们有了主桶，就可以直接存储
//相关密钥。If we're in watch only mode, then some or all of
//
	if masterHDPrivEnc != nil {
		err := bucket.Put(masterHDPrivName, masterHDPrivEnc)
		if err != nil {
			str := "failed to store encrypted master HD private key"
			return managerError(ErrDatabase, str, err)
		}
	}

	if masterHDPubEnc != nil {
		err := bucket.Put(masterHDPubName, masterHDPubEnc)
		if err != nil {
			str := "failed to store encrypted master HD public key"
			return managerError(ErrDatabase, str, err)
		}
	}

	return nil
}

//
//
//
func fetchMasterHDKeys(ns walletdb.ReadBucket) ([]byte, []byte, error) {
	bucket := ns.NestedReadBucket(mainBucketName)

	var masterHDPrivEnc, masterHDPubEnc []byte

//
//
//
	key := bucket.Get(masterHDPrivName)
	if key != nil {
		masterHDPrivEnc = make([]byte, len(key))
		copy(masterHDPrivEnc[:], key)
	}

	key = bucket.Get(masterHDPubName)
	if key != nil {
		masterHDPubEnc = make([]byte, len(key))
		copy(masterHDPubEnc[:], key)
	}

	return masterHDPrivEnc, masterHDPubEnc, nil
}

//
//
//
//
func fetchCryptoKeys(ns walletdb.ReadBucket) ([]byte, []byte, []byte, error) {
	bucket := ns.NestedReadBucket(mainBucketName)

//
	val := bucket.Get(cryptoPubKeyName)
	if val == nil {
		str := "required encrypted crypto public not stored in database"
		return nil, nil, nil, managerError(ErrDatabase, str, nil)
	}
	pubKey := make([]byte, len(val))
	copy(pubKey, val)

//
	var privKey []byte
	val = bucket.Get(cryptoPrivKeyName)
	if val != nil {
		privKey = make([]byte, len(val))
		copy(privKey, val)
	}

//
	var scriptKey []byte
	val = bucket.Get(cryptoScriptKeyName)
	if val != nil {
		scriptKey = make([]byte, len(val))
		copy(scriptKey, val)
	}

	return pubKey, privKey, scriptKey, nil
}

//
//
//
func putCryptoKeys(ns walletdb.ReadWriteBucket, pubKeyEncrypted, privKeyEncrypted,
	scriptKeyEncrypted []byte) error {

	bucket := ns.NestedReadWriteBucket(mainBucketName)

	if pubKeyEncrypted != nil {
		err := bucket.Put(cryptoPubKeyName, pubKeyEncrypted)
		if err != nil {
			str := "failed to store encrypted crypto public key"
			return managerError(ErrDatabase, str, err)
		}
	}

	if privKeyEncrypted != nil {
		err := bucket.Put(cryptoPrivKeyName, privKeyEncrypted)
		if err != nil {
			str := "failed to store encrypted crypto private key"
			return managerError(ErrDatabase, str, err)
		}
	}

	if scriptKeyEncrypted != nil {
		err := bucket.Put(cryptoScriptKeyName, scriptKeyEncrypted)
		if err != nil {
			str := "failed to store encrypted crypto script key"
			return managerError(ErrDatabase, str, err)
		}
	}

	return nil
}

//
func fetchWatchingOnly(ns walletdb.ReadBucket) (bool, error) {
	bucket := ns.NestedReadBucket(mainBucketName)

	buf := bucket.Get(watchingOnlyName)
	if len(buf) != 1 {
		str := "malformed watching-only flag stored in database"
		return false, managerError(ErrDatabase, str, nil)
	}

	return buf[0] != 0, nil
}

//
func putWatchingOnly(ns walletdb.ReadWriteBucket, watchingOnly bool) error {
	bucket := ns.NestedReadWriteBucket(mainBucketName)

	var encoded byte
	if watchingOnly {
		encoded = 1
	}

	if err := bucket.Put(watchingOnlyName, []byte{encoded}); err != nil {
		str := "failed to store watching only flag"
		return managerError(ErrDatabase, str, err)
	}
	return nil
}

//
//
//
func deserializeAccountRow(accountID []byte, serializedAccount []byte) (*dbAccountRow, error) {
//
//
//
//

//
//
	if len(serializedAccount) < 5 {
		str := fmt.Sprintf("malformed serialized account for key %x",
			accountID)
		return nil, managerError(ErrDatabase, str, nil)
	}

	row := dbAccountRow{}
	row.acctType = accountType(serializedAccount[0])
	rdlen := binary.LittleEndian.Uint32(serializedAccount[1:5])
	row.rawData = make([]byte, rdlen)
	copy(row.rawData, serializedAccount[5:5+rdlen])

	return &row, nil
}

//
func serializeAccountRow(row *dbAccountRow) []byte {
//
//
//
//
	rdlen := len(row.rawData)
	buf := make([]byte, 5+rdlen)
	buf[0] = byte(row.acctType)
	binary.LittleEndian.PutUint32(buf[1:5], uint32(rdlen))
	copy(buf[5:5+rdlen], row.rawData)
	return buf
}

//
//
func deserializeDefaultAccountRow(accountID []byte, row *dbAccountRow) (*dbDefaultAccountRow, error) {
//
//
//
//
//
//
//

//
//
	if len(row.rawData) < 20 {
		str := fmt.Sprintf("malformed serialized bip0044 account for "+
			"key %x", accountID)
		return nil, managerError(ErrDatabase, str, nil)
	}

	retRow := dbDefaultAccountRow{
		dbAccountRow: *row,
	}

	pubLen := binary.LittleEndian.Uint32(row.rawData[0:4])
	retRow.pubKeyEncrypted = make([]byte, pubLen)
	copy(retRow.pubKeyEncrypted, row.rawData[4:4+pubLen])
	offset := 4 + pubLen
	privLen := binary.LittleEndian.Uint32(row.rawData[offset : offset+4])
	offset += 4
	retRow.privKeyEncrypted = make([]byte, privLen)
	copy(retRow.privKeyEncrypted, row.rawData[offset:offset+privLen])
	offset += privLen
	retRow.nextExternalIndex = binary.LittleEndian.Uint32(row.rawData[offset : offset+4])
	offset += 4
	retRow.nextInternalIndex = binary.LittleEndian.Uint32(row.rawData[offset : offset+4])
	offset += 4
	nameLen := binary.LittleEndian.Uint32(row.rawData[offset : offset+4])
	offset += 4
	retRow.name = string(row.rawData[offset : offset+nameLen])

	return &retRow, nil
}

//
//
func serializeDefaultAccountRow(encryptedPubKey, encryptedPrivKey []byte,
	nextExternalIndex, nextInternalIndex uint32, name string) []byte {

//
//
//
//
//
//
//4字节下一个内部索引+4字节名称len+名称
	pubLen := uint32(len(encryptedPubKey))
	privLen := uint32(len(encryptedPrivKey))
	nameLen := uint32(len(name))
	rawData := make([]byte, 20+pubLen+privLen+nameLen)
	binary.LittleEndian.PutUint32(rawData[0:4], pubLen)
	copy(rawData[4:4+pubLen], encryptedPubKey)
	offset := 4 + pubLen
	binary.LittleEndian.PutUint32(rawData[offset:offset+4], privLen)
	offset += 4
	copy(rawData[offset:offset+privLen], encryptedPrivKey)
	offset += privLen
	binary.LittleEndian.PutUint32(rawData[offset:offset+4], nextExternalIndex)
	offset += 4
	binary.LittleEndian.PutUint32(rawData[offset:offset+4], nextInternalIndex)
	offset += 4
	binary.LittleEndian.PutUint32(rawData[offset:offset+4], nameLen)
	offset += 4
	copy(rawData[offset:offset+nameLen], name)
	return rawData
}

//ForEachKeyScope为每个已知的管理器作用域调用给定的函数
//在根管理器已知的范围集内。
func forEachKeyScope(ns walletdb.ReadBucket, fn func(KeyScope) error) error {
	bucket := ns.NestedReadBucket(scopeBucketName)

	return bucket.ForEach(func(k, v []byte) error {
//跳过非桶
		if len(k) != 8 {
			return nil
		}

		scope := KeyScope{
			Purpose: binary.LittleEndian.Uint32(k[:]),
			Coin:    binary.LittleEndian.Uint32(k[4:]),
		}

		return fn(scope)
	})
}

//
//
func forEachAccount(ns walletdb.ReadBucket, scope *KeyScope,
	fn func(account uint32) error) error {

	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	acctBucket := scopedBucket.NestedReadBucket(acctBucketName)
	return acctBucket.ForEach(func(k, v []byte) error {
//跳过桶。
		if v == nil {
			return nil
		}
		return fn(binary.LittleEndian.Uint32(k))
	})
}

//
func fetchLastAccount(ns walletdb.ReadBucket, scope *KeyScope) (uint32, error) {
	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return 0, err
	}

	metaBucket := scopedBucket.NestedReadBucket(metaBucketName)

	val := metaBucket.Get(lastAccountName)
	if len(val) != 4 {
		str := fmt.Sprintf("malformed metadata '%s' stored in database",
			lastAccountName)
		return 0, managerError(ErrDatabase, str, nil)
	}

	account := binary.LittleEndian.Uint32(val[0:4])
	return account, nil
}

//fetchaccountname从
//数据库。
func fetchAccountName(ns walletdb.ReadBucket, scope *KeyScope,
	account uint32) (string, error) {

	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return "", err
	}

	acctIDxBucket := scopedBucket.NestedReadBucket(acctIDIdxBucketName)

	val := acctIDxBucket.Get(uint32ToBytes(account))
	if val == nil {
		str := fmt.Sprintf("account %d not found", account)
		return "", managerError(ErrAccountNotFound, str, nil)
	}

	offset := uint32(0)
	nameLen := binary.LittleEndian.Uint32(val[offset : offset+4])
	offset += 4
	acctName := string(val[offset : offset+nameLen])

	return acctName, nil
}

//
//数据库。
func fetchAccountByName(ns walletdb.ReadBucket, scope *KeyScope,
	name string) (uint32, error) {

	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return 0, err
	}

	idxBucket := scopedBucket.NestedReadBucket(acctNameIdxBucketName)

	val := idxBucket.Get(stringToBytes(name))
	if val == nil {
		str := fmt.Sprintf("account name '%s' not found", name)
		return 0, managerError(ErrAccountNotFound, str, nil)
	}

	return binary.LittleEndian.Uint32(val), nil
}

//
//数据库。
func fetchAccountInfo(ns walletdb.ReadBucket, scope *KeyScope,
	account uint32) (interface{}, error) {

	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return nil, err
	}

	acctBucket := scopedBucket.NestedReadBucket(acctBucketName)

	accountID := uint32ToBytes(account)
	serializedRow := acctBucket.Get(accountID)
	if serializedRow == nil {
		str := fmt.Sprintf("account %d not found", account)
		return nil, managerError(ErrAccountNotFound, str, nil)
	}

	row, err := deserializeAccountRow(accountID, serializedRow)
	if err != nil {
		return nil, err
	}

	switch row.acctType {
	case accountDefault:
		return deserializeDefaultAccountRow(accountID, row)
	}

	str := fmt.Sprintf("unsupported account type '%d'", row.acctType)
	return nil, managerError(ErrDatabase, str, nil)
}

//
func deleteAccountNameIndex(ns walletdb.ReadWriteBucket, scope *KeyScope,
	name string) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadWriteBucket(acctNameIdxBucketName)

//
	err = bucket.Delete(stringToBytes(name))
	if err != nil {
		str := fmt.Sprintf("failed to delete account name index key %s", name)
		return managerError(ErrDatabase, str, err)
	}
	return nil
}

//
func deleteAccountIDIndex(ns walletdb.ReadWriteBucket, scope *KeyScope,
	account uint32) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadWriteBucket(acctIDIdxBucketName)

//
	err = bucket.Delete(uint32ToBytes(account))
	if err != nil {
		str := fmt.Sprintf("failed to delete account id index key %d", account)
		return managerError(ErrDatabase, str, err)
	}
	return nil
}

//PutAccountNameIndex将给定的密钥存储到
//数据库。
func putAccountNameIndex(ns walletdb.ReadWriteBucket, scope *KeyScope,
	account uint32, name string) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadWriteBucket(acctNameIdxBucketName)

//写下由帐户名键入的帐号。
	err = bucket.Put(stringToBytes(name), uint32ToBytes(account))
	if err != nil {
		str := fmt.Sprintf("failed to store account name index key %s", name)
		return managerError(ErrDatabase, str, err)
	}
	return nil
}

//
func putAccountIDIndex(ns walletdb.ReadWriteBucket, scope *KeyScope,
	account uint32, name string) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadWriteBucket(acctIDIdxBucketName)

//
	err = bucket.Put(uint32ToBytes(account), stringToBytes(name))
	if err != nil {
		str := fmt.Sprintf("failed to store account id index key %s", name)
		return managerError(ErrDatabase, str, err)
	}
	return nil
}

//
//数据库。
func putAddrAccountIndex(ns walletdb.ReadWriteBucket, scope *KeyScope,
	account uint32, addrHash []byte) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadWriteBucket(addrAcctIdxBucketName)

//
	err = bucket.Put(addrHash, uint32ToBytes(account))
	if err != nil {
		return nil
	}

	bucket, err = bucket.CreateBucketIfNotExists(uint32ToBytes(account))
	if err != nil {
		return err
	}

//
	err = bucket.Put(addrHash, nullVal)
	if err != nil {
		str := fmt.Sprintf("failed to store address account index key %s", addrHash)
		return managerError(ErrDatabase, str, err)
	}
	return nil
}

//
//
func putAccountRow(ns walletdb.ReadWriteBucket, scope *KeyScope,
	account uint32, row *dbAccountRow) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadWriteBucket(acctBucketName)

//
	err = bucket.Put(uint32ToBytes(account), serializeAccountRow(row))
	if err != nil {
		str := fmt.Sprintf("failed to store account %d", account)
		return managerError(ErrDatabase, str, err)
	}
	return nil
}

//
func putAccountInfo(ns walletdb.ReadWriteBucket, scope *KeyScope,
	account uint32, encryptedPubKey, encryptedPrivKey []byte,
	nextExternalIndex, nextInternalIndex uint32, name string) error {

	rawData := serializeDefaultAccountRow(
		encryptedPubKey, encryptedPrivKey, nextExternalIndex,
		nextInternalIndex, name,
	)

//

	acctRow := dbAccountRow{
		acctType: accountDefault,
		rawData:  rawData,
	}
	if err := putAccountRow(ns, scope, account, &acctRow); err != nil {
		return err
	}

//
	if err := putAccountIDIndex(ns, scope, account, name); err != nil {
		return err
	}

//
	if err := putAccountNameIndex(ns, scope, account, name); err != nil {
		return err
	}

	return nil
}

//
//数据库。
func putLastAccount(ns walletdb.ReadWriteBucket, scope *KeyScope,
	account uint32) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadWriteBucket(metaBucketName)

	err = bucket.Put(lastAccountName, uint32ToBytes(account))
	if err != nil {
		str := fmt.Sprintf("failed to update metadata '%s'", lastAccountName)
		return managerError(ErrDatabase, str, err)
	}
	return nil
}

//
//信息。这被用作各种地址类型的公共基
//反序列化公共部分。
func deserializeAddressRow(serializedAddress []byte) (*dbAddressRow, error) {
//序列化地址格式为：
//<addrtype><account><addedtime><syncstatus><rawdata>
//
//1字节addrType+4字节account+8字节addTime+1字节
//同步状态+4字节原始数据长度+原始数据

//鉴于上述情况，入口的长度必须至少为
//常量值大小。
	if len(serializedAddress) < 18 {
		str := "malformed serialized address"
		return nil, managerError(ErrDatabase, str, nil)
	}

	row := dbAddressRow{}
	row.addrType = addressType(serializedAddress[0])
	row.account = binary.LittleEndian.Uint32(serializedAddress[1:5])
	row.addTime = binary.LittleEndian.Uint64(serializedAddress[5:13])
	row.syncStatus = syncStatus(serializedAddress[13])
	rdlen := binary.LittleEndian.Uint32(serializedAddress[14:18])
	row.rawData = make([]byte, rdlen)
	copy(row.rawData, serializedAddress[18:18+rdlen])

	return &row, nil
}

//SerializeAddressRow返回传递的地址行的序列化。
func serializeAddressRow(row *dbAddressRow) []byte {
//序列化地址格式为：
//<addrtype><account><addedtime><syncstatus><commentlen><comment>
//
//
//1字节addrType+4字节account+8字节addTime+1字节
//同步状态+4字节原始数据长度+原始数据
	rdlen := len(row.rawData)
	buf := make([]byte, 18+rdlen)
	buf[0] = byte(row.addrType)
	binary.LittleEndian.PutUint32(buf[1:5], row.account)
	binary.LittleEndian.PutUint64(buf[5:13], row.addTime)
	buf[13] = byte(row.syncStatus)
	binary.LittleEndian.PutUint32(buf[14:18], uint32(rdlen))
	copy(buf[18:18+rdlen], row.rawData)
	return buf
}

//
//
func deserializeChainedAddress(row *dbAddressRow) (*dbChainAddressRow, error) {
//
//
//
//
	if len(row.rawData) != 8 {
		str := "malformed serialized chained address"
		return nil, managerError(ErrDatabase, str, nil)
	}

	retRow := dbChainAddressRow{
		dbAddressRow: *row,
	}

	retRow.branch = binary.LittleEndian.Uint32(row.rawData[0:4])
	retRow.index = binary.LittleEndian.Uint32(row.rawData[4:8])

	return &retRow, nil
}

//
//
func serializeChainedAddress(branch, index uint32) []byte {
//
//<分支>索引>
//
//
	rawData := make([]byte, 8)
	binary.LittleEndian.PutUint32(rawData[0:4], branch)
	binary.LittleEndian.PutUint32(rawData[4:8], index)
	return rawData
}

//
//行作为导入地址。
func deserializeImportedAddress(row *dbAddressRow) (*dbImportedAddressRow, error) {
//
//
//
//
//

//鉴于上述情况，入口的长度必须至少为
//常量值大小。
	if len(row.rawData) < 8 {
		str := "malformed serialized imported address"
		return nil, managerError(ErrDatabase, str, nil)
	}

	retRow := dbImportedAddressRow{
		dbAddressRow: *row,
	}

	pubLen := binary.LittleEndian.Uint32(row.rawData[0:4])
	retRow.encryptedPubKey = make([]byte, pubLen)
	copy(retRow.encryptedPubKey, row.rawData[4:4+pubLen])
	offset := 4 + pubLen
	privLen := binary.LittleEndian.Uint32(row.rawData[offset : offset+4])
	offset += 4
	retRow.encryptedPrivKey = make([]byte, privLen)
	copy(retRow.encryptedPrivKey, row.rawData[offset:offset+privLen])

	return &retRow, nil
}

//
//导入的地址。
func serializeImportedAddress(encryptedPubKey, encryptedPrivKey []byte) []byte {
//
//<encpubkeylen><encpubkey><encprivkeylen><encprivkey>
//
//4字节加密pubkey len+加密pubkey+4字节加密
//
	pubLen := uint32(len(encryptedPubKey))
	privLen := uint32(len(encryptedPrivKey))
	rawData := make([]byte, 8+pubLen+privLen)
	binary.LittleEndian.PutUint32(rawData[0:4], pubLen)
	copy(rawData[4:4+pubLen], encryptedPubKey)
	offset := 4 + pubLen
	binary.LittleEndian.PutUint32(rawData[offset:offset+4], privLen)
	offset += 4
	copy(rawData[offset:offset+privLen], encryptedPrivKey)
	return rawData
}

//
//
func deserializeScriptAddress(row *dbAddressRow) (*dbScriptAddressRow, error) {
//
//
//
//
//

//鉴于上述情况，入口的长度必须至少为
//常量值大小。
	if len(row.rawData) < 8 {
		str := "malformed serialized script address"
		return nil, managerError(ErrDatabase, str, nil)
	}

	retRow := dbScriptAddressRow{
		dbAddressRow: *row,
	}

	hashLen := binary.LittleEndian.Uint32(row.rawData[0:4])
	retRow.encryptedHash = make([]byte, hashLen)
	copy(retRow.encryptedHash, row.rawData[4:4+hashLen])
	offset := 4 + hashLen
	scriptLen := binary.LittleEndian.Uint32(row.rawData[offset : offset+4])
	offset += 4
	retRow.encryptedScript = make([]byte, scriptLen)
	copy(retRow.encryptedScript, row.rawData[offset:offset+scriptLen])

	return &retRow, nil
}

//SerializeScriptAddress返回的原始数据字段的序列化
//脚本地址。
func serializeScriptAddress(encryptedHash, encryptedScript []byte) []byte {
//序列化脚本地址原始数据格式为：
//
//
//4字节加密脚本哈希长度+加密脚本哈希+4字节
//

	hashLen := uint32(len(encryptedHash))
	scriptLen := uint32(len(encryptedScript))
	rawData := make([]byte, 8+hashLen+scriptLen)
	binary.LittleEndian.PutUint32(rawData[0:4], hashLen)
	copy(rawData[4:4+hashLen], encryptedHash)
	offset := 4 + hashLen
	binary.LittleEndian.PutUint32(rawData[offset:offset+4], scriptLen)
	offset += 4
	copy(rawData[offset:offset+scriptLen], encryptedScript)
	return rawData
}

//
//
//
//
//
func fetchAddressByHash(ns walletdb.ReadBucket, scope *KeyScope,
	addrHash []byte) (interface{}, error) {

	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return nil, err
	}

	bucket := scopedBucket.NestedReadBucket(addrBucketName)

	serializedRow := bucket.Get(addrHash[:])
	if serializedRow == nil {
		str := "address not found"
		return nil, managerError(ErrAddressNotFound, str, nil)
	}

	row, err := deserializeAddressRow(serializedRow)
	if err != nil {
		return nil, err
	}

	switch row.addrType {
	case adtChain:
		return deserializeChainedAddress(row)
	case adtImport:
		return deserializeImportedAddress(row)
	case adtScript:
		return deserializeScriptAddress(row)
	}

	str := fmt.Sprintf("unsupported address type '%d'", row.addrType)
	return nil, managerError(ErrDatabase, str, nil)
}

//
func fetchAddressUsed(ns walletdb.ReadBucket, scope *KeyScope,
	addressID []byte) bool {

	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return false
	}

	bucket := scopedBucket.NestedReadBucket(usedAddrBucketName)

	addrHash := sha256.Sum256(addressID)
	return bucket.Get(addrHash[:]) != nil
}

//
func markAddressUsed(ns walletdb.ReadWriteBucket, scope *KeyScope,
	addressID []byte) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadWriteBucket(usedAddrBucketName)

	addrHash := sha256.Sum256(addressID)
	val := bucket.Get(addrHash[:])
	if val != nil {
		return nil
	}

	err = bucket.Put(addrHash[:], []byte{0})
	if err != nil {
		str := fmt.Sprintf("failed to mark address used %x", addressID)
		return managerError(ErrDatabase, str, err)
	}

	return nil
}

//
//数据库。返回的值是特定
//
//
//失败。
func fetchAddress(ns walletdb.ReadBucket, scope *KeyScope,
	addressID []byte) (interface{}, error) {

	addrHash := sha256.Sum256(addressID)
	return fetchAddressByHash(ns, scope, addrHash[:])
}

//
//
func putAddress(ns walletdb.ReadWriteBucket, scope *KeyScope,
	addressID []byte, row *dbAddressRow) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadWriteBucket(addrBucketName)

//
//
//
	addrHash := sha256.Sum256(addressID)
	err = bucket.Put(addrHash[:], serializeAddressRow(row))
	if err != nil {
		str := fmt.Sprintf("failed to store address %x", addressID)
		return managerError(ErrDatabase, str, err)
	}

//
	return putAddrAccountIndex(ns, scope, row.account, addrHash[:])
}

//
//数据库。
func putChainedAddress(ns walletdb.ReadWriteBucket, scope *KeyScope,
	addressID []byte, account uint32, status syncStatus, branch,
	index uint32, addrType addressType) error {

	scopedBucket, err := fetchWriteScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	addrRow := dbAddressRow{
		addrType:   addrType,
		account:    account,
		addTime:    uint64(time.Now().Unix()),
		syncStatus: status,
		rawData:    serializeChainedAddress(branch, index),
	}
	if err := putAddress(ns, scope, addressID, &addrRow); err != nil {
		return err
	}

//
//分支机构。
	accountID := uint32ToBytes(account)
	bucket := scopedBucket.NestedReadWriteBucket(acctBucketName)
	serializedAccount := bucket.Get(accountID)

//
	row, err := deserializeAccountRow(accountID, serializedAccount)
	if err != nil {
		return err
	}
	arow, err := deserializeDefaultAccountRow(accountID, row)
	if err != nil {
		return err
	}

//
//
	nextExternalIndex := arow.nextExternalIndex
	nextInternalIndex := arow.nextInternalIndex
	if branch == InternalBranch {
		nextInternalIndex = index + 1
	} else {
		nextExternalIndex = index + 1
	}

//使用更新的索引重新序列化帐户并存储它。
	row.rawData = serializeDefaultAccountRow(
		arow.pubKeyEncrypted, arow.privKeyEncrypted, nextExternalIndex,
		nextInternalIndex, arow.name,
	)
	err = bucket.Put(accountID, serializeAccountRow(row))
	if err != nil {
		str := fmt.Sprintf("failed to update next index for "+
			"address %x, account %d", addressID, account)
		return managerError(ErrDatabase, str, err)
	}
	return nil
}

//
//数据库。
func putImportedAddress(ns walletdb.ReadWriteBucket, scope *KeyScope,
	addressID []byte, account uint32, status syncStatus,
	encryptedPubKey, encryptedPrivKey []byte) error {

	rawData := serializeImportedAddress(encryptedPubKey, encryptedPrivKey)
	addrRow := dbAddressRow{
		addrType:   adtImport,
		account:    account,
		addTime:    uint64(time.Now().Unix()),
		syncStatus: status,
		rawData:    rawData,
	}
	return putAddress(ns, scope, addressID, &addrRow)
}

//
//数据库。
func putScriptAddress(ns walletdb.ReadWriteBucket, scope *KeyScope,
	addressID []byte, account uint32, status syncStatus,
	encryptedHash, encryptedScript []byte) error {

	rawData := serializeScriptAddress(encryptedHash, encryptedScript)
	addrRow := dbAddressRow{
		addrType:   adtScript,
		account:    account,
		addTime:    uint64(time.Now().Unix()),
		syncStatus: status,
		rawData:    rawData,
	}
	if err := putAddress(ns, scope, addressID, &addrRow); err != nil {
		return err
	}

	return nil
}

//
func existsAddress(ns walletdb.ReadBucket, scope *KeyScope, addressID []byte) bool {
	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return false
	}

	bucket := scopedBucket.NestedReadBucket(addrBucketName)

	addrHash := sha256.Sum256(addressID)
	return bucket.Get(addrHash[:]) != nil
}

//
//
//
func fetchAddrAccount(ns walletdb.ReadBucket, scope *KeyScope,
	addressID []byte) (uint32, error) {

	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return 0, err
	}

	bucket := scopedBucket.NestedReadBucket(addrAcctIdxBucketName)

	addrHash := sha256.Sum256(addressID)
	val := bucket.Get(addrHash[:])
	if val == nil {
		str := "address not found"
		return 0, managerError(ErrAddressNotFound, str, nil)
	}
	return binary.LittleEndian.Uint32(val), nil
}

//
//
func forEachAccountAddress(ns walletdb.ReadBucket, scope *KeyScope,
	account uint32, fn func(rowInterface interface{}) error) error {

	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadBucket(addrAcctIdxBucketName).
		NestedReadBucket(uint32ToBytes(account))

//如果索引存储桶缺少帐户，则没有
//
	if bucket == nil {
		return nil
	}

	err = bucket.ForEach(func(k, v []byte) error {
//跳过桶。
		if v == nil {
			return nil
		}

		addrRow, err := fetchAddressByHash(ns, scope, k)
		if err != nil {
			if merr, ok := err.(*ManagerError); ok {
				desc := fmt.Sprintf("failed to fetch address hash '%s': %v",
					k, merr.Description)
				merr.Description = desc
				return merr
			}
			return err
		}

		return fn(addrRow)
	})
	if err != nil {
		return maybeConvertDbError(err)
	}
	return nil
}

//
//
func forEachActiveAddress(ns walletdb.ReadBucket, scope *KeyScope,
	fn func(rowInterface interface{}) error) error {

	scopedBucket, err := fetchReadScopeBucket(ns, scope)
	if err != nil {
		return err
	}

	bucket := scopedBucket.NestedReadBucket(addrBucketName)

	err = bucket.ForEach(func(k, v []byte) error {
//跳过桶。
		if v == nil {
			return nil
		}

//
//价值观。
		addrRow, err := fetchAddressByHash(ns, scope, k)
		if merr, ok := err.(*ManagerError); ok {
			desc := fmt.Sprintf("failed to fetch address hash '%s': %v",
				k, merr.Description)
			merr.Description = desc
			return merr
		}
		if err != nil {
			return err
		}

		return fn(addrRow)
	})
	if err != nil {
		return maybeConvertDbError(err)
	}
	return nil
}

//
//
//
//
//
//
//
func deletePrivateKeys(ns walletdb.ReadWriteBucket) error {
	bucket := ns.NestedReadWriteBucket(mainBucketName)

//
//
	if err := bucket.Delete(masterPrivKeyName); err != nil {
		str := "failed to delete master private key parameters"
		return managerError(ErrDatabase, str, err)
	}
	if err := bucket.Delete(cryptoPrivKeyName); err != nil {
		str := "failed to delete crypto private key"
		return managerError(ErrDatabase, str, err)
	}
	if err := bucket.Delete(cryptoScriptKeyName); err != nil {
		str := "failed to delete crypto script key"
		return managerError(ErrDatabase, str, err)
	}
	if err := bucket.Delete(masterHDPrivName); err != nil {
		str := "failed to delete master HD priv key"
		return managerError(ErrDatabase, str, err)
	}

//
//同时删除所有已知作用域的键。
	scopeBucket := ns.NestedReadWriteBucket(scopeBucketName)
	err := scopeBucket.ForEach(func(scopeKey, _ []byte) error {
		if len(scopeKey) != 8 {
			return nil
		}

		managerScopeBucket := scopeBucket.NestedReadWriteBucket(scopeKey)

		if err := managerScopeBucket.Delete(coinTypePrivKeyName); err != nil {
			str := "failed to delete cointype private key"
			return managerError(ErrDatabase, str, err)
		}

//
		bucket = managerScopeBucket.NestedReadWriteBucket(acctBucketName)
		err := bucket.ForEach(func(k, v []byte) error {
//跳过桶。
			if v == nil {
				return nil
			}

//
			row, err := deserializeAccountRow(k, v)
			if err != nil {
				return err
			}

			switch row.acctType {
			case accountDefault:
				arow, err := deserializeDefaultAccountRow(k, row)
				if err != nil {
					return err
				}

//
//把它储存起来。
				row.rawData = serializeDefaultAccountRow(
					arow.pubKeyEncrypted, nil,
					arow.nextExternalIndex, arow.nextInternalIndex,
					arow.name,
				)
				err = bucket.Put(k, serializeAccountRow(row))
				if err != nil {
					str := "failed to delete account private key"
					return managerError(ErrDatabase, str, err)
				}
			}

			return nil
		})
		if err != nil {
			return maybeConvertDbError(err)
		}

//
		bucket = managerScopeBucket.NestedReadWriteBucket(addrBucketName)
		err = bucket.ForEach(func(k, v []byte) error {
//跳过桶。
			if v == nil {
				return nil
			}

//
//价值观。
			row, err := deserializeAddressRow(v)
			if err != nil {
				return err
			}

			switch row.addrType {
			case adtImport:
				irow, err := deserializeImportedAddress(row)
				if err != nil {
					return err
				}

//
//
				row.rawData = serializeImportedAddress(
					irow.encryptedPubKey, nil)
				err = bucket.Put(k, serializeAddressRow(row))
				if err != nil {
					str := "failed to delete imported private key"
					return managerError(ErrDatabase, str, err)
				}

			case adtScript:
				srow, err := deserializeScriptAddress(row)
				if err != nil {
					return err
				}

//
//
				row.rawData = serializeScriptAddress(srow.encryptedHash,
					nil)
				err = bucket.Put(k, serializeAddressRow(row))
				if err != nil {
					str := "failed to delete imported script"
					return managerError(ErrDatabase, str, err)
				}
			}

			return nil
		})
		if err != nil {
			return maybeConvertDbError(err)
		}

		return nil
	})
	if err != nil {
		return maybeConvertDbError(err)
	}

	return nil
}

//
//数据库。
func fetchSyncedTo(ns walletdb.ReadBucket) (*BlockStamp, error) {
	bucket := ns.NestedReadBucket(syncBucketName)

//
//
//
//
	buf := bucket.Get(syncedToName)
	if len(buf) < 36 {
		str := "malformed sync information stored in database"
		return nil, managerError(ErrDatabase, str, nil)
	}

	var bs BlockStamp
	bs.Height = int32(binary.LittleEndian.Uint32(buf[0:4]))
	copy(bs.Hash[:], buf[4:36])

	if len(buf) == 40 {
		bs.Timestamp = time.Unix(
			int64(binary.LittleEndian.Uint32(buf[36:])), 0,
		)
	}

	return &bs, nil
}

//
func PutSyncedTo(ns walletdb.ReadWriteBucket, bs *BlockStamp) error {
	bucket := ns.NestedReadWriteBucket(syncBucketName)
	errStr := fmt.Sprintf("failed to store sync information %v", bs.Hash)

//
//
//
//
	if bs.Height > 0 {
		if _, err := fetchBlockHash(ns, bs.Height-1); err != nil {
			return managerError(ErrDatabase, errStr, err)
		}
	}

//
	height := make([]byte, 4)
	binary.BigEndian.PutUint32(height, uint32(bs.Height))
	err := bucket.Put(height, bs.Hash[0:32])
	if err != nil {
		return managerError(ErrDatabase, errStr, err)
	}

//
//
//
//
	buf := make([]byte, 40)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(bs.Height))
	copy(buf[4:36], bs.Hash[0:32])
	binary.LittleEndian.PutUint32(buf[36:], uint32(bs.Timestamp.Unix()))

	err = bucket.Put(syncedToName, buf)
	if err != nil {
		return managerError(ErrDatabase, errStr, err)
	}
	return nil
}

//
//数据库。
func fetchBlockHash(ns walletdb.ReadBucket, height int32) (*chainhash.Hash, error) {
	bucket := ns.NestedReadBucket(syncBucketName)
	errStr := fmt.Sprintf("failed to fetch block hash for height %d", height)

	heightBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(heightBytes, uint32(height))
	hashBytes := bucket.Get(heightBytes)
	if hashBytes == nil {
		err := errors.New("block not found")
		return nil, managerError(ErrBlockNotFound, errStr, err)
	}
	if len(hashBytes) != 32 {
		err := fmt.Errorf("couldn't get hash from database")
		return nil, managerError(ErrDatabase, errStr, err)
	}
	var hash chainhash.Hash
	if err := hash.SetBytes(hashBytes); err != nil {
		return nil, managerError(ErrDatabase, errStr, err)
	}
	return &hash, nil
}

//
//数据库。
func FetchStartBlock(ns walletdb.ReadBucket) (*BlockStamp, error) {
	bucket := ns.NestedReadBucket(syncBucketName)

//
//
//
//
	buf := bucket.Get(startBlockName)
	if len(buf) != 36 {
		str := "malformed start block stored in database"
		return nil, managerError(ErrDatabase, str, nil)
	}

	var bs BlockStamp
	bs.Height = int32(binary.LittleEndian.Uint32(buf[0:4]))
	copy(bs.Hash[:], buf[4:36])
	return &bs, nil
}

//
func putStartBlock(ns walletdb.ReadWriteBucket, bs *BlockStamp) error {
	bucket := ns.NestedReadWriteBucket(syncBucketName)

//
//<blockheight><blockhash>
//
//
	buf := make([]byte, 36)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(bs.Height))
	copy(buf[4:36], bs.Hash[0:32])

	err := bucket.Put(startBlockName, buf)
	if err != nil {
		str := fmt.Sprintf("failed to store start block %v", bs.Hash)
		return managerError(ErrDatabase, str, err)
	}
	return nil
}

//fetchbirthday从数据库加载管理器的bithday时间戳。
func fetchBirthday(ns walletdb.ReadBucket) (time.Time, error) {
	var t time.Time

	bucket := ns.NestedReadBucket(syncBucketName)
	birthdayTimestamp := bucket.Get(birthdayName)
	if len(birthdayTimestamp) != 8 {
		str := "malformed birthday stored in database"
		return t, managerError(ErrDatabase, str, nil)
	}

	t = time.Unix(int64(binary.BigEndian.Uint64(birthdayTimestamp)), 0)

	return t, nil
}

//PutBirthday将提供的生日时间戳存储到数据库中。
func putBirthday(ns walletdb.ReadWriteBucket, t time.Time) error {
	var birthdayTimestamp [8]byte
	binary.BigEndian.PutUint64(birthdayTimestamp[:], uint64(t.Unix()))

	bucket := ns.NestedReadWriteBucket(syncBucketName)
	if err := bucket.Put(birthdayName, birthdayTimestamp[:]); err != nil {
		str := "failed to store birthday"
		return managerError(ErrDatabase, str, err)
	}

	return nil
}

//FetchBirthDayBlock从数据库中检索生日块。
//
//
//
//
//
func FetchBirthdayBlock(ns walletdb.ReadBucket) (BlockStamp, error) {
	var block BlockStamp

	bucket := ns.NestedReadBucket(syncBucketName)
	birthdayBlock := bucket.Get(birthdayBlockName)
	if birthdayBlock == nil {
		str := "birthday block not set"
		return block, managerError(ErrBirthdayBlockNotSet, str, nil)
	}
	if len(birthdayBlock) != 44 {
		str := "malformed birthday block stored in database"
		return block, managerError(ErrDatabase, str, nil)
	}

	block.Height = int32(binary.BigEndian.Uint32(birthdayBlock[:4]))
	copy(block.Hash[:], birthdayBlock[4:36])
	t := int64(binary.BigEndian.Uint64(birthdayBlock[36:]))
	block.Timestamp = time.Unix(t, 0)

	return block, nil
}

//
//
//
//
//[4:36]块哈希
//[36:44]块时间戳
func putBirthdayBlock(ns walletdb.ReadWriteBucket, block BlockStamp) error {
	var birthdayBlock [44]byte
	binary.BigEndian.PutUint32(birthdayBlock[:4], uint32(block.Height))
	copy(birthdayBlock[4:36], block.Hash[:])
	binary.BigEndian.PutUint64(birthdayBlock[36:], uint64(block.Timestamp.Unix()))

	bucket := ns.NestedReadWriteBucket(syncBucketName)
	if err := bucket.Put(birthdayBlockName, birthdayBlock[:]); err != nil {
		str := "failed to store birthday block"
		return managerError(ErrDatabase, str, err)
	}

	return nil
}

//fetchbirthdayblockverification检索确定
//钱包已经证实它的生日卡是正确的。
func fetchBirthdayBlockVerification(ns walletdb.ReadBucket) bool {
	bucket := ns.NestedReadBucket(syncBucketName)
	verifiedValue := bucket.Get(birthdayBlockVerifiedName)

//如果没有验证状态，我们可以假设它没有
//
	if verifiedValue == nil {
		return false
	}

//
	verified := binary.BigEndian.Uint16(verifiedValue[:])
	return verified != 0
}

//PutBirthDayBlockVerification存储了一点确定
//
func putBirthdayBlockVerification(ns walletdb.ReadWriteBucket, verified bool) error {
//将布尔值转换为二进制表示形式中的整数
//无法将布尔值直接插入为
//键/值对。
	verifiedValue := uint16(0)
	if verified {
		verifiedValue = 1
	}

	var verifiedBytes [2]byte
	binary.BigEndian.PutUint16(verifiedBytes[:], verifiedValue)

	bucket := ns.NestedReadWriteBucket(syncBucketName)
	err := bucket.Put(birthdayBlockVerifiedName, verifiedBytes[:])
	if err != nil {
		str := "failed to store birthday block verification"
		return managerError(ErrDatabase, str, err)
	}

	return nil
}

//managerxists返回是否已创建管理器
//在给定的数据库命名空间中。
func managerExists(ns walletdb.ReadBucket) bool {
	if ns == nil {
		return false
	}
	mainBucket := ns.NestedReadBucket(mainBucketName)
	return mainBucket != nil
}

//
//
//ScopedManager还需要履行其职责。
func createScopedManagerNS(ns walletdb.ReadWriteBucket, scope *KeyScope) error {
//首先，我们将为这个特定的
//
	scopeKey := scopeToBytes(scope)
	scopeBucket, err := ns.CreateBucket(scopeKey[:])
	if err != nil {
		str := "failed to create sync bucket"
		return managerError(ErrDatabase, str, err)
	}

	_, err = scopeBucket.CreateBucket(acctBucketName)
	if err != nil {
		str := "failed to create account bucket"
		return managerError(ErrDatabase, str, err)
	}

	_, err = scopeBucket.CreateBucket(addrBucketName)
	if err != nil {
		str := "failed to create address bucket"
		return managerError(ErrDatabase, str, err)
	}

//usedaddrbacketname bucket是在Manager版本1发布后添加的
	_, err = scopeBucket.CreateBucket(usedAddrBucketName)
	if err != nil {
		str := "failed to create used addresses bucket"
		return managerError(ErrDatabase, str, err)
	}

	_, err = scopeBucket.CreateBucket(addrAcctIdxBucketName)
	if err != nil {
		str := "failed to create address index bucket"
		return managerError(ErrDatabase, str, err)
	}

	_, err = scopeBucket.CreateBucket(acctNameIdxBucketName)
	if err != nil {
		str := "failed to create an account name index bucket"
		return managerError(ErrDatabase, str, err)
	}

	_, err = scopeBucket.CreateBucket(acctIDIdxBucketName)
	if err != nil {
		str := "failed to create an account id index bucket"
		return managerError(ErrDatabase, str, err)
	}

	_, err = scopeBucket.CreateBucket(metaBucketName)
	if err != nil {
		str := "failed to create a meta bucket"
		return managerError(ErrDatabase, str, err)
	}

	return nil
}

//
//
//
//
//
func createManagerNS(ns walletdb.ReadWriteBucket,
	defaultScopes map[KeyScope]ScopeAddrSchema) error {

//
//
	mainBucket, err := ns.CreateBucket(mainBucketName)
	if err != nil {
		str := "failed to create main bucket"
		return managerError(ErrDatabase, str, err)
	}
	_, err = ns.CreateBucket(syncBucketName)
	if err != nil {
		str := "failed to create sync bucket"
		return managerError(ErrDatabase, str, err)
	}

//
//
	scopeBucket, err := ns.CreateBucket(scopeBucketName)
	if err != nil {
		str := "failed to create scope bucket"
		return managerError(ErrDatabase, str, err)
	}
	scopeSchemas, err := ns.CreateBucket(scopeSchemaBucketName)
	if err != nil {
		str := "failed to create scope schema bucket"
		return managerError(ErrDatabase, str, err)
	}

//
//
	for scope, scopeSchema := range defaultScopes {
//
//
//喜欢。
		scopeKey := scopeToBytes(&scope)
		schemaBytes := scopeSchemaToBytes(&scopeSchema)
		err := scopeSchemas.Put(scopeKey[:], schemaBytes)
		if err != nil {
			return err
		}

		err = createScopedManagerNS(scopeBucket, &scope)
		if err != nil {
			return err
		}

		err = putLastAccount(ns, &scope, DefaultAccountNum)
		if err != nil {
			return err
		}
	}

	if err := putManagerVersion(ns, latestMgrVersion); err != nil {
		return err
	}

	createDate := uint64(time.Now().Unix())
	var dateBytes [8]byte
	binary.LittleEndian.PutUint64(dateBytes[:], createDate)
	err = mainBucket.Put(mgrCreateDateName, dateBytes[:])
	if err != nil {
		str := "failed to store database creation time"
		return managerError(ErrDatabase, str, err)
	}

	return nil
}
