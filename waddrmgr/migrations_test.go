
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package waddrmgr

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcwallet/walletdb"
)

//applyMigration是一个助手函数，它允许我们断言
//迁移前后的顶级存储桶。这可用于确保
//迁移的正确性。
func applyMigration(t *testing.T, beforeMigration, afterMigration,
	migration func(walletdb.ReadWriteBucket) error, shouldFail bool) {

	t.Helper()

//
	teardown, db, _ := setupManager(t)
	defer teardown()

//首先，我们将运行beforemigration闭包，其中包含
//在继续执行之前需要数据库修改/断言
//迁移。
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		if ns == nil {
			return errors.New("top-level namespace does not exist")
		}
		return beforeMigration(ns)
	})
	if err != nil {
		t.Fatalf("unable to run beforeMigration func: %v", err)
	}

//然后，我们将运行迁移本身，如果不匹配，则会失败。
//它的预期结果。
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		if ns == nil {
			return errors.New("top-level namespace does not exist")
		}
		return migration(ns)
	})
	if err != nil && !shouldFail {
		t.Fatalf("unable to perform migration: %v", err)
	} else if err == nil && shouldFail {
		t.Fatal("expected migration to fail, but did not")
	}

//最后，我们将运行AfterMigration闭包，其中包含
//为了保证比迁移更重要的断言
//成功。
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		if ns == nil {
			return errors.New("top-level namespace does not exist")
		}
		return afterMigration(ns)
	})
	if err != nil {
		t.Fatalf("unable to run afterMigration func: %v", err)
	}
}

//testmigrationpupupulatebirthdayblock确保迁移到
//
func TestMigrationPopulateBirthdayBlock(t *testing.T) {
	t.Parallel()

	var expectedHeight int32
	beforeMigration := func(ns walletdb.ReadWriteBucket) error {
//为了测试这个迁移，我们将从写入磁盘10开始
//随机块。
		block := &BlockStamp{}
		for i := int32(1); i <= 10; i++ {
			block.Height = i
			blockHash := bytes.Repeat([]byte(string(i)), 32)
			copy(block.Hash[:], blockHash)
			if err := PutSyncedTo(ns, block); err != nil {
				return err
			}
		}

//插入块后，我们假设生日
//块对应于链中的第7个块（共11个）。
//为此，我们需要将生日时间戳设置为
//在创世纪之后6个街区的估计时间戳。
		genesisTimestamp := chaincfg.MainNetParams.GenesisBlock.Header.Timestamp
		delta := time.Hour
		expectedHeight = int32(delta.Seconds() / 600)
		birthday := genesisTimestamp.Add(delta)
		if err := putBirthday(ns, birthday); err != nil {
			return err
		}

//最后，由于迁移还没有开始，我们应该
//在数据库中找不到生日块。
		_, err := FetchBirthdayBlock(ns)
		if !IsError(err, ErrBirthdayBlockNotSet) {
			return fmt.Errorf("expected ErrBirthdayBlockNotSet, "+
				"got %v", err)
		}

		return nil
	}

//迁移完成后，我们应该看到生日
//块现在存在，并设置为正确的预期高度。
	afterMigration := func(ns walletdb.ReadWriteBucket) error {
		birthdayBlock, err := FetchBirthdayBlock(ns)
		if err != nil {
			return err
		}

		if birthdayBlock.Height != expectedHeight {
			return fmt.Errorf("expected birthday block with "+
				"height %d, got %d", expectedHeight,
				birthdayBlock.Height)
		}

		return nil
	}

//我们现在可以应用迁移，并期望它不会失败。
	applyMigration(
		t, beforeMigration, afterMigration, populateBirthdayBlock,
		false,
	)
}

//
//可以从我们的角度正确地检测到链的高度估计
//尚未到达。
func TestMigrationPopulateBirthdayBlockEstimateTooFar(t *testing.T) {
	t.Parallel()

	const numBlocks = 1000
	chainParams := chaincfg.MainNetParams

	var expectedHeight int32
	beforeMigration := func(ns walletdb.ReadWriteBucket) error {
//要测试此迁移，我们将从写入磁盘999开始
//随机块来模拟高度为1000的同步链。
		block := &BlockStamp{}
		for i := int32(1); i < numBlocks; i++ {
			block.Height = i
			blockHash := bytes.Repeat([]byte(string(i)), 32)
			copy(block.Hash[:], blockHash)
			if err := PutSyncedTo(ns, block); err != nil {
				return err
			}
		}

//插入块后，我们假设生日
//区块对应于链中的第900个区块。做
//
//之后899个块的估计时间戳
//起源。但是，如果平均块
//时间不是10分钟，这会偏离估计高度
//测试时高度比链条长
//网络（testnet、regtest等）未完全同步
//钱包）相反，迁移应该能够处理这个问题
//减去一天的积木直到找到一个积木
//它知道的。
//
//我们会让移民假设我们的生日在街区
//链条中有1001个。因为这个块不存在于
//从数据库的角度来看，一天的数据块
//从估计值中减去，这应该给我们一个有效的
//块高度。
		genesisTimestamp := chainParams.GenesisBlock.Header.Timestamp
		delta := numBlocks * 10 * time.Minute
		expectedHeight = numBlocks - 144

		birthday := genesisTimestamp.Add(delta)
		if err := putBirthday(ns, birthday); err != nil {
			return err
		}

//最后，由于迁移还没有开始，我们应该
//在数据库中找不到生日块。
		_, err := FetchBirthdayBlock(ns)
		if !IsError(err, ErrBirthdayBlockNotSet) {
			return fmt.Errorf("expected ErrBirthdayBlockNotSet, "+
				"got %v", err)
		}

		return nil
	}

//迁移完成后，我们应该看到生日
//块现在存在，并设置为正确的预期高度。
	afterMigration := func(ns walletdb.ReadWriteBucket) error {
		birthdayBlock, err := FetchBirthdayBlock(ns)
		if err != nil {
			return err
		}

		if birthdayBlock.Height != expectedHeight {
			return fmt.Errorf("expected birthday block height %d, "+
				"got %d", expectedHeight, birthdayBlock.Height)
		}

		return nil
	}

//我们现在可以应用迁移，并期望它不会失败。
	applyMigration(
		t, beforeMigration, afterMigration, populateBirthdayBlock,
		false,
	)
}

//TestMigrationResetsSyncedBlockToBirthDay确保钱包能够正确看到
//重置后，它与块同步为生日块。
func TestMigrationResetSyncedBlockToBirthday(t *testing.T) {
	t.Parallel()

	var birthdayBlock BlockStamp
	beforeMigration := func(ns walletdb.ReadWriteBucket) error {
//为了测试这个迁移，我们假设我们同步到一个链
//有100个街区，我们的生日是50个街区。
		block := &BlockStamp{}
		for i := int32(1); i < 100; i++ {
			block.Height = i
			blockHash := bytes.Repeat([]byte(string(i)), 32)
			copy(block.Hash[:], blockHash)
			if err := PutSyncedTo(ns, block); err != nil {
				return err
			}
		}

		const birthdayHeight = 50
		birthdayHash, err := fetchBlockHash(ns, birthdayHeight)
		if err != nil {
			return err
		}

		birthdayBlock = BlockStamp{
			Hash: *birthdayHash, Height: birthdayHeight,
		}

		return putBirthdayBlock(ns, birthdayBlock)
	}

	afterMigration := func(ns walletdb.ReadWriteBucket) error {
//迁移成功后，我们应该看到
//数据库的同步块现在反映生日块。
		syncedBlock, err := fetchSyncedTo(ns)
		if err != nil {
			return err
		}

		if syncedBlock.Height != birthdayBlock.Height {
			return fmt.Errorf("expected synced block height %d, "+
				"got %d", birthdayBlock.Height,
				syncedBlock.Height)
		}
		if !syncedBlock.Hash.IsEqual(&birthdayBlock.Hash) {
			return fmt.Errorf("expected synced block height %v, "+
				"got %v", birthdayBlock.Hash, syncedBlock.Hash)
		}

		return nil
	}

//我们现在可以应用迁移，并期望它不会失败。
	applyMigration(
		t, beforeMigration, afterMigration, resetSyncedBlockToBirthday,
		false,
	)
}

//TestMigrationResetsSyncedBlockToBirthDayWithNoBirthDayBlock确保
//如果没有，则无法将同步到块重置为生日块
//可用。
func TestMigrationResetSyncedBlockToBirthdayWithNoBirthdayBlock(t *testing.T) {
	t.Parallel()

//复制数据库不知道
//生日街区，我们不设。这将导致迁移到
//失败了。
	beforeMigration := func(walletdb.ReadWriteBucket) error {
		return nil
	}
	afterMigration := func(walletdb.ReadWriteBucket) error {
		return nil
	}
	applyMigration(
		t, beforeMigration, afterMigration, resetSyncedBlockToBirthday,
		true,
	)
}
