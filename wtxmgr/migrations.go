
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package wtxmgr

import (
	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/btcsuite/btcwallet/walletdb/migration"
)

//版本是不同数据库版本的列表。最后一项应该
//反映最新的数据库状态。如果数据库恰好是一个版本
//数字低于最新值，将执行迁移以捕获
//它上升了。
var versions = []migration.Version{
	{
		Number:    1,
		Migration: nil,
	},
	{
		Number:    2,
		Migration: dropTransactionHistory,
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

//NewMigrationManager为事务创建新的迁移管理器
//经理。给定的存储桶应反映顶级存储桶，其中
//其中包含事务管理器的数据。
func NewMigrationManager(ns walletdb.ReadWriteBucket) *MigrationManager {
	return &MigrationManager{ns: ns}
}

//name返回我们将尝试升级的服务的名称。
//
//注意：此方法是migration.manager接口的一部分。
func (m *MigrationManager) Name() string {
	return "wallet transaction manager"
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
	return fetchVersion(m.ns)
}

//setversion设置服务数据库的版本。
//
//注意：此方法是migration.manager接口的一部分。
func (m *MigrationManager) SetVersion(ns walletdb.ReadWriteBucket,
	version uint32) error {

	if ns == nil {
		ns = m.ns
	}
	return putVersion(m.ns, version)
}

//版本返回服务的所有可用数据库版本。
//
//注意：此方法是migration.manager接口的一部分。
func (m *MigrationManager) Versions() []migration.Version {
	return versions
}

//DropTransactionHistory是尝试重新创建
//具有干净状态的事务存储。
func dropTransactionHistory(ns walletdb.ReadWriteBucket) error {
	log.Info("Dropping wallet transaction history")

//要删除商店的事务历史记录，我们需要删除所有
//相关的子代存储桶和键/值对。
	if err := deleteBuckets(ns); err != nil {
		return err
	}
	if err := ns.Delete(rootMinedBalance); err != nil {
		return err
	}

//把所有的东西都拿走了，我们现在就重新做我们的桶。
	if err := createBuckets(ns); err != nil {
		return err
	}

//最后，我们将为我们的已开采余额插入一个0值。
	return putMinedBalance(ns, 0)
}
