
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package wtxmgr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/btcsuite/btcwallet/walletdb"
)

//applyMigration是一个助手函数，它允许我们断言
//迁移前后的顶级存储桶。这可用于确保
//迁移的正确性。
func applyMigration(t *testing.T,
	beforeMigration, afterMigration func(walletdb.ReadWriteBucket, *Store) error,
	migration func(walletdb.ReadWriteBucket) error, shouldFail bool) {

	t.Helper()

//我们将首先设置由数据库支持的事务存储。
	store, db, teardown, err := testStore()
	if err != nil {
		t.Fatalf("unable to create test store: %v", err)
	}
	defer teardown()

//首先，我们将运行beforemigration闭包，其中包含
//在继续执行之前需要数据库修改/断言
//迁移。
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(namespaceKey)
		if ns == nil {
			return errors.New("top-level namespace does not exist")
		}
		return beforeMigration(ns, store)
	})
	if err != nil {
		t.Fatalf("unable to run beforeMigration func: %v", err)
	}

//然后，我们将运行迁移本身，如果不匹配，则会失败。
//它的预期结果。
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(namespaceKey)
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
		ns := tx.ReadWriteBucket(namespaceKey)
		if ns == nil {
			return errors.New("top-level namespace does not exist")
		}
		return afterMigration(ns, store)
	})
	if err != nil {
		t.Fatalf("unable to run afterMigration func: %v", err)
	}
}

//TestMigrationDropTransactionHistory确保重置事务存储
//在删除其事务历史记录后进入干净状态。
func TestMigrationDropTransactionHistory(t *testing.T) {
	t.Parallel()

//checkTransactions是一个助手函数，它将断言
//基于迁移是否
//是否完成。
	checkTransactions := func(ns walletdb.ReadWriteBucket, s *Store,
		afterMigration bool) error {

//我们应该在
//迁移，之后没有。
		utxos, err := s.UnspentOutputs(ns)
		if err != nil {
			return err
		}
		if len(utxos) == 0 && !afterMigration {
			return errors.New("expected to find 1 utxo, found none")
		}
		if len(utxos) > 0 && afterMigration {
			return fmt.Errorf("expected to find 0 utxos, found %d",
				len(utxos))
		}

//我们应该在
//迁移，之后没有。
		unconfirmedTxs, err := s.UnminedTxs(ns)
		if err != nil {
			return err
		}
		if len(unconfirmedTxs) == 0 && !afterMigration {
			return errors.New("expected to find 1 unconfirmed " +
				"transaction, found none")
		}
		if len(unconfirmedTxs) > 0 && afterMigration {
			return fmt.Errorf("expected to find 0 unconfirmed "+
				"transactions, found %d", len(unconfirmedTxs))
		}

//在迁移之前我们应该有一个非零的平衡，并且
//零之后。
		minedBalance, err := fetchMinedBalance(ns)
		if err != nil {
			return err
		}
		if minedBalance == 0 && !afterMigration {
			return errors.New("expected non-zero balance before " +
				"migration")
		}
		if minedBalance > 0 && afterMigration {
			return fmt.Errorf("expected zero balance after "+
				"migration, got %d", minedBalance)
		}

		return nil
	}

	beforeMigration := func(ns walletdb.ReadWriteBucket, s *Store) error {
//我们将从向商店添加两个事务开始：a
//确认交易和未确认交易。
//确认的交易将从CoinBase输出中支出，
//而未确认的将花费来自已确认的
//交易。
		cb := newCoinBase(1e8)
		cbRec, err := NewTxRecordFromMsgTx(cb, timeNow())
		if err != nil {
			return err
		}

		b := &BlockMeta{Block: Block{Height: 100}}
		confirmedSpend := spendOutput(&cbRec.Hash, 0, 5e7, 4e7)
		confirmedSpendRec, err := NewTxRecordFromMsgTx(
			confirmedSpend, timeNow(),
		)
		if err := s.InsertTx(ns, confirmedSpendRec, b); err != nil {
			return err
		}
		err = s.AddCredit(ns, confirmedSpendRec, b, 1, true)
		if err != nil {
			return err
		}

		unconfimedSpend := spendOutput(
			&confirmedSpendRec.Hash, 0, 5e6, 5e6,
		)
		unconfirmedSpendRec, err := NewTxRecordFromMsgTx(
			unconfimedSpend, timeNow(),
		)
		if err != nil {
			return err
		}
		if err := s.InsertTx(ns, unconfirmedSpendRec, nil); err != nil {
			return err
		}
		err = s.AddCredit(ns, unconfirmedSpendRec, nil, 1, true)
		if err != nil {
			return err
		}

//确保这些事务存在于存储区中。
		return checkTransactions(ns, s, false)
	}

	afterMigration := func(ns walletdb.ReadWriteBucket, s *Store) error {
//假设迁移成功，我们应该看到
//商店在
//迁移。
		return checkTransactions(ns, s, true)
	}

//我们现在可以应用迁移，并期望它不会失败。
	applyMigration(
		t, beforeMigration, afterMigration, dropTransactionHistory,
		false,
	)
}
