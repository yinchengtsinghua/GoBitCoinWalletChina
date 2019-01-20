
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package waddrmgr

import (
	"errors"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/btcsuite/btcwallet/walletdb/migration"
)

//版本是不同数据库版本的列表。最后一项应该
//反映最新的数据库状态。如果数据库恰好是一个版本
//数字低于最新值，将执行迁移以捕获
//它上升了。
var versions = []migration.Version{
	{
		Number:    2,
		Migration: upgradeToVersion2,
	},
	{
		Number:    5,
		Migration: upgradeToVersion5,
	},
	{
		Number:    6,
		Migration: populateBirthdayBlock,
	},
	{
		Number:    7,
		Migration: resetSyncedBlockToBirthday,
	},
}

//GetLatestVersion返回最新数据库版本的版本号。
func getLatestVersion() uint32 {
	return versions[len(versions)-1].Number
}

//MigrationManager是Migration.Manager接口的一个实现，
//将用于处理地址管理器的迁移。它暴露了
//成功执行迁移所需的必要参数。
type MigrationManager struct {
	ns walletdb.ReadWriteBucket
}

//一个编译时断言，以确保MigrationManager实现
//migration.manager接口。
var _ migration.Manager = (*MigrationManager)(nil)

//NewMigrationManager为地址管理器创建新的迁移管理器。
//给定的存储桶应反映所有
//地址管理器的数据包含在中。
func NewMigrationManager(ns walletdb.ReadWriteBucket) *MigrationManager {
	return &MigrationManager{ns: ns}
}

//name返回我们将尝试升级的服务的名称。
//
//注意：此方法是migration.manager接口的一部分。
func (m *MigrationManager) Name() string {
	return "wallet address manager"
}

//命名空间返回服务的顶级存储桶。
//
//注意：此方法是migration.manager接口的一部分。
func (m *MigrationManager) Namespace() walletdb.ReadWriteBucket {
	return m.ns
}

//current version返回服务数据库的当前版本。
//
//注意：此方法是migration.manager接口的一部分。
func (m *MigrationManager) CurrentVersion(ns walletdb.ReadBucket) (uint32, error) {
	if ns == nil {
		ns = m.ns
	}
	return fetchManagerVersion(ns)
}

//setversion设置服务数据库的版本。
//
//注意：此方法是migration.manager接口的一部分。
func (m *MigrationManager) SetVersion(ns walletdb.ReadWriteBucket,
	version uint32) error {

	if ns == nil {
		ns = m.ns
	}
	return putManagerVersion(m.ns, version)
}

//版本返回服务的所有可用数据库版本。
//
//注意：此方法是migration.manager接口的一部分。
func (m *MigrationManager) Versions() []migration.Version {
	return versions
}

//升级到版本2将数据库从版本1升级到版本2
//
//已初始化，下次重新扫描时将更新。
func upgradeToVersion2(ns walletdb.ReadWriteBucket) error {
	currentMgrVersion := uint32(2)

	_, err := ns.CreateBucketIfNotExists(usedAddrBucketName)
	if err != nil {
		str := "failed to create used addresses bucket"
		return managerError(ErrDatabase, str, err)
	}

	return putManagerVersion(ns, currentMgrVersion)
}

//升级到版本5将数据库从版本4升级到版本5。后
//此更新无法使用新的ScopedKeyManager功能。这是应该的。
//事实上，在版本5中，我们现在将加密的master private存储在
//磁盘上的键。但是，使用bip0044键作用域，用户仍然可以
//创建旧的p2pkh地址。
func upgradeToVersion5(ns walletdb.ReadWriteBucket) error {
//首先，我们将检查是否有任何现有的segwit地址，其中
//无法升级到新版本。如果是，我们中止并警告
//用户。
	err := ns.NestedReadBucket(addrBucketName).ForEach(
		func(k []byte, v []byte) error {
			row, err := deserializeAddressRow(v)
			if err != nil {
				return err
			}
			if row.addrType > adtScript {
				return fmt.Errorf("segwit address exists in " +
					"wallet, can't upgrade from v4 to " +
					"v5: well, we tried  ¯\\_(ツ)_/¯")
			}
			return nil
		})
	if err != nil {
		return err
	}

//接下来，我们将写出新的数据库版本。
	if err := putManagerVersion(ns, 5); err != nil {
		return err
	}

//首先，我们需要创建在新的
//数据库版本。
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

//通过创建bucket，我们现在可以创建默认的bip0044
//作用域，它将是在此之后数据库中唯一可用的作用域
//更新。
	scopeKey := scopeToBytes(&KeyScopeBIP0044)
	scopeSchema := ScopeAddrMap[KeyScopeBIP0044]
	schemaBytes := scopeSchemaToBytes(&scopeSchema)
	if err := scopeSchemas.Put(scopeKey[:], schemaBytes); err != nil {
		return err
	}
	if err := createScopedManagerNS(scopeBucket, &KeyScopeBIP0044); err != nil {
		return err
	}

	bip44Bucket := scopeBucket.NestedReadWriteBucket(scopeKey[:])

//创建bucket后，我们现在需要将*每个*项移植到
//前一个主bucket，进入新的默认范围。
	mainBucket := ns.NestedReadWriteBucket(mainBucketName)

//首先，我们将转移到加密硬币类型的私人和公共
//新子存储桶的键。
	encCoinPrivKeys := mainBucket.Get(coinTypePrivKeyName)
	encCoinPubKeys := mainBucket.Get(coinTypePubKeyName)

	err = bip44Bucket.Put(coinTypePrivKeyName, encCoinPrivKeys)
	if err != nil {
		return err
	}
	err = bip44Bucket.Put(coinTypePubKeyName, encCoinPubKeys)
	if err != nil {
		return err
	}

	if err := mainBucket.Delete(coinTypePrivKeyName); err != nil {
		return err
	}
	if err := mainBucket.Delete(coinTypePubKeyName); err != nil {
		return err
	}

//接下来，我们将把元桶中的所有内容转移到
//
	metaBucket := ns.NestedReadWriteBucket(metaBucketName)
	lastAccount := metaBucket.Get(lastAccountName)
	if err := metaBucket.Delete(lastAccountName); err != nil {
		return err
	}

	scopedMetaBucket := bip44Bucket.NestedReadWriteBucket(metaBucketName)
	err = scopedMetaBucket.Put(lastAccountName, lastAccount)
	if err != nil {
		return err
	}

//
//以前在主桶下面，放入新的作用域桶。我们将
//通过获取需要修改的所有键的切片来实现。
//然后在每个桶中递归，移动两个嵌套桶
//和键/值对。
	keysToMigrate := [][]byte{
		acctBucketName, addrBucketName, usedAddrBucketName,
		addrAcctIdxBucketName, acctNameIdxBucketName, acctIDIdxBucketName,
	}

//递归地迁移每个bucket。
	for _, bucketKey := range keysToMigrate {
		err := migrateRecursively(ns, bip44Bucket, bucketKey)
		if err != nil {
			return err
		}
	}

	return nil
}

//将嵌套存储桶从一个存储桶移动到另一个存储桶，
//根据需要递归到嵌套存储桶中。
func migrateRecursively(src, dst walletdb.ReadWriteBucket,
	bucketKey []byte) error {
//在这个bucket键中，我们将迁移，然后删除每个键。
	bucketToMigrate := src.NestedReadWriteBucket(bucketKey)
	newBucket, err := dst.CreateBucketIfNotExists(bucketKey)
	if err != nil {
		return err
	}
	err = bucketToMigrate.ForEach(func(k, v []byte) error {
		if nestedBucket := bucketToMigrate.
			NestedReadBucket(k); nestedBucket != nil {
//我们有一个嵌套的bucket，所以可以循环使用它。
			return migrateRecursively(bucketToMigrate, newBucket, k)
		}

		if err := newBucket.Put(k, v); err != nil {
			return err
		}

		return bucketToMigrate.Delete(k)
	})
	if err != nil {
		return err
	}
//最后，我们将删除bucket本身。
	if err := src.DeleteNestedBucket(bucketKey); err != nil {
		return err
	}
	return nil
}

//PopulateBirthDayblock是一种尝试填充生日的迁移。
//钱包的一块。这是必要的，以便在我们需要
//重新扫描钱包，我们可以从这个街区开始，而不是
//而不是来自创世纪板块。
//
//注意：此迁移不能保证生日块的正确性。
//因为我们不存储块时间戳，所以必须进行健全性检查
//启动钱包以确保我们不会错过任何相关
//重新扫描时发生的事件。
func populateBirthdayBlock(ns walletdb.ReadWriteBucket) error {
//为了确定
//生日时间戳的相应块高度。既然我们这样做了
//不是存储块时间戳，我们需要估计我们的高度
//查看Genesis时间戳并假设每10个发生一个块
//分钟。这可能是不安全的，并导致我们实际上错过了链
//事件，因此在钱包尝试同步之前进行健全性检查
//本身。
//
//我们将从获取生日时间戳开始。
	birthdayTimestamp, err := fetchBirthday(ns)
	if err != nil {
		return fmt.Errorf("unable to fetch birthday timestamp: %v", err)
	}

	log.Infof("Setting the wallet's birthday block from timestamp=%v",
		birthdayTimestamp)

//现在，我们需要确定Genesis块的时间戳
//相应的链。
	genesisHash, err := fetchBlockHash(ns, 0)
	if err != nil {
		return fmt.Errorf("unable to fetch genesis block hash: %v", err)
	}

	var genesisTimestamp time.Time
	switch *genesisHash {
	case *chaincfg.MainNetParams.GenesisHash:
		genesisTimestamp =
			chaincfg.MainNetParams.GenesisBlock.Header.Timestamp

	case *chaincfg.TestNet3Params.GenesisHash:
		genesisTimestamp =
			chaincfg.TestNet3Params.GenesisBlock.Header.Timestamp

	case *chaincfg.RegressionNetParams.GenesisHash:
		genesisTimestamp =
			chaincfg.RegressionNetParams.GenesisBlock.Header.Timestamp

	case *chaincfg.SimNetParams.GenesisHash:
		genesisTimestamp =
			chaincfg.SimNetParams.GenesisBlock.Header.Timestamp

	default:
		return fmt.Errorf("unknown genesis hash %v", genesisHash)
	}

//通过检索时间戳，我们可以通过
//取两者之差除以平均块
//时间（10分钟）。
	birthdayHeight := int32((birthdayTimestamp.Sub(genesisTimestamp).Seconds() / 600))

//既然我们有了高度估计，我们就可以得到相应的
//阻止并将其设置为我们的生日阻止。
	birthdayHash, err := fetchBlockHash(ns, birthdayHeight)

//为了确保我们记录下我们从链条上知道的高度，
//我们会确保能找到这个高度估计。否则，我们会
//继续减去一天的积木，直到我们找到一个。
	for IsError(err, ErrBlockNotFound) {
		birthdayHeight -= 144
		if birthdayHeight < 0 {
			birthdayHeight = 0
		}
		birthdayHash, err = fetchBlockHash(ns, birthdayHeight)
	}
	if err != nil {
		return err
	}

	log.Infof("Estimated birthday block from timestamp=%v: height=%d, "+
		"hash=%v", birthdayTimestamp, birthdayHeight, birthdayHash)

//注意：生日块的时间戳没有设置，因为我们没有
//存储每个块的时间戳。
	return putBirthdayBlock(ns, BlockStamp{
		Height: birthdayHeight,
		Hash:   *birthdayHash,
	})
}

//ResetSynchedBlockToBirthDay是一种迁移，用于重置钱包当前的
//同步块到其生日块。这基本上是向
//强制重新扫描钱包。
func resetSyncedBlockToBirthday(ns walletdb.ReadWriteBucket) error {
	syncBucket := ns.NestedReadWriteBucket(syncBucketName)
	if syncBucket == nil {
		return errors.New("sync bucket does not exist")
	}

	birthdayBlock, err := FetchBirthdayBlock(ns)
	if err != nil {
		return err
	}

	return PutSyncedTo(ns, &birthdayBlock)
}
