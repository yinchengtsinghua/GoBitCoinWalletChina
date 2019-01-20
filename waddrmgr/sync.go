
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

package waddrmgr

import (
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcwallet/walletdb"
)

//blockstamp定义一个块（按高度和唯一的哈希），用于标记
//区块链中地址管理器元素是
//同步到。
type BlockStamp struct {
	Height    int32
	Hash      chainhash.Hash
	Timestamp time.Time
}

//SyncState存储管理器的同步状态。它包括最近
//将块视为高度，以及开始和当前同步块戳。
type syncState struct {
//StartBlock是第一个可安全用于启动
//再扫描。它是使用创建管理器的块，或者
//提供导入地址或脚本的最早块。
	startBlock BlockStamp

//syncedto是当前块管理器中的地址已知
//要与同步。
	syncedTo BlockStamp
}

//NewSyncState返回具有所提供参数的新同步状态。
func newSyncState(startBlock, syncedTo *BlockStamp) *syncState {

	return &syncState{
		startBlock: *startBlock,
		syncedTo:   *syncedTo,
	}
}

//setsyncedto将地址管理器标记为与最近看到的
//块戳描述的块。当提供的块戳为零时，
//
//将使用导入的地址。这有效地允许经理
//标记为未同步回已知最早的地址点
//出现在区块链中。
func (m *Manager) SetSyncedTo(ns walletdb.ReadWriteBucket, bs *BlockStamp) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

//使用存储的开始块戳并重置最近的哈希值和高度
//当提供的块戳为零时。
	if bs == nil {
		bs = &m.syncState.startBlock
	}

//更新数据库。
	err := PutSyncedTo(ns, bs)
	if err != nil {
		return err
	}

//更新数据库后立即更新内存。
	m.syncState.syncedTo = *bs
	return nil
}

//syncedto返回有关块高度和地址哈希的详细信息
//经理至少是通过同步的。目的是打电话的人
//可以使用此信息智能启动重新扫描以同步回
//最后一个已知良好区块的最佳链条。
func (m *Manager) SyncedTo() BlockStamp {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	return m.syncState.syncedTo
}

//block hash返回特定块高度的块哈希。这个
//将信息与链后端进行比较，以查看
//REORG正在发生，它向后走了多远。
func (m *Manager) BlockHash(ns walletdb.ReadBucket, height int32) (
	*chainhash.Hash, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	return fetchBlockHash(ns, height)
}

//Birthday返回生日，或者最早可以使用密钥的时间，
//为了经理。
func (m *Manager) Birthday() time.Time {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	return m.birthday
}

//setbirthday设置生日，或者最早可以使用密钥的时间，
//为了经理。
func (m *Manager) SetBirthday(ns walletdb.ReadWriteBucket,
	birthday time.Time) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	m.birthday = birthday
	return putBirthday(ns, birthday)
}

//birthday block返回生日块，或密钥可能具有的最早块
//为经理使用。还返回一个布尔值来指示
//生日块已验证为正确。
func (m *Manager) BirthdayBlock(ns walletdb.ReadBucket) (BlockStamp, bool, error) {
	birthdayBlock, err := FetchBirthdayBlock(ns)
	if err != nil {
		return BlockStamp{}, false, err
	}

	return birthdayBlock, fetchBirthdayBlockVerification(ns), nil
}

//setbirthdayblock设置生日块，或密钥可能具有的最早时间
//为经理使用。验证的布尔值可用于指定
//是否应检查此生日块是否正常，以确定是否存在
//存在一个更好的候选，以防止更少的块获取。
func (m *Manager) SetBirthdayBlock(ns walletdb.ReadWriteBucket,
	block BlockStamp, verified bool) error {

	if err := putBirthdayBlock(ns, block); err != nil {
		return err
	}
	return putBirthdayBlockVerification(ns, verified)
}
