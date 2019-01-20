
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wtxmgr

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/walletdb"
)

var (
//花：假
//输出：10 BTC
	exampleTxRecordA *TxRecord

//花费：A：0
//输出：5 BTC、5 BTC
	exampleTxRecordB *TxRecord
)

func init() {
	tx := spendOutput(&chainhash.Hash{}, 0, 10e8)
	rec, err := NewTxRecordFromMsgTx(tx, timeNow())
	if err != nil {
		panic(err)
	}
	exampleTxRecordA = rec

	tx = spendOutput(&exampleTxRecordA.Hash, 0, 5e8, 5e8)
	rec, err = NewTxRecordFromMsgTx(tx, timeNow())
	if err != nil {
		panic(err)
	}
	exampleTxRecordB = rec
}

var exampleBlock100 = makeBlockMeta(100)

//此示例演示如何报告给定的未链接和
//挖掘的事务给出0、1和6个块确认。
func ExampleStore_Balance() {
	s, db, teardown, err := testStore()
	defer teardown()
	if err != nil {
		fmt.Println(err)
		return
	}

//打印0个块确认、1个确认和6的余额
//确认。
	printBalances := func(syncHeight int32) {
		dbtx, err := db.BeginReadTx()
		if err != nil {
			fmt.Println(err)
			return
		}
		defer dbtx.Rollback()
		ns := dbtx.ReadBucket(namespaceKey)
		zeroConfBal, err := s.Balance(ns, 0, syncHeight)
		if err != nil {
			fmt.Println(err)
			return
		}
		oneConfBal, err := s.Balance(ns, 1, syncHeight)
		if err != nil {
			fmt.Println(err)
			return
		}
		sixConfBal, err := s.Balance(ns, 6, syncHeight)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("%v, %v, %v\n", zeroConfBal, oneConfBal, sixConfBal)
	}

//插入一个输出10 BTC的事务，取消链接并标记输出
//作为信用。
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(namespaceKey)
		err := s.InsertTx(ns, exampleTxRecordA, nil)
		if err != nil {
			return err
		}
		return s.AddCredit(ns, exampleTxRecordA, nil, 0, false)
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	printBalances(100)

//在块100中挖掘交易记录，然后使用
//同步高度为100和105块。
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(namespaceKey)
		return s.InsertTx(ns, exampleTxRecordA, &exampleBlock100)
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	printBalances(100)
	printBalances(105)

//输出：
//10个BTC，0个BTC，0个BTC
//10个BTC，10个BTC，0个BTC
//10个BTC，10个BTC，10个BTC
}

func ExampleStore_Rollback() {
	s, db, teardown, err := testStore()
	defer teardown()
	if err != nil {
		fmt.Println(err)
		return
	}

	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(namespaceKey)

//在高度为100的块中插入输出10 BTC的事务。
		err := s.InsertTx(ns, exampleTxRecordA, &exampleBlock100)
		if err != nil {
			return err
		}

//从块100开始回滚所有内容。
		err = s.Rollback(ns, 100)
		if err != nil {
			return err
		}

//断言该事务现在已取消链接。
		details, err := s.TxDetails(ns, &exampleTxRecordA.Hash)
		if err != nil {
			return err
		}
		if details == nil {
			return fmt.Errorf("no details found")
		}
		fmt.Println(details.Block.Height)
		return nil
	})
	if err != nil {
		fmt.Println(err)
		return
	}

//输出：
//- 1
}

func Example_basicUsage() {
//打开数据库。
	db, dbTeardown, err := testDB()
	defer dbTeardown()
	if err != nil {
		fmt.Println(err)
		return
	}

//打开读写事务以对数据库进行操作。
	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer dbtx.Commit()

//为事务存储创建存储桶。
	b, err := dbtx.CreateTopLevelBucket([]byte("txstore"))
	if err != nil {
		fmt.Println(err)
		return
	}

//在提供的命名空间中创建并打开事务存储。
	err = Create(b)
	if err != nil {
		fmt.Println(err)
		return
	}
	s, err := Open(b, &chaincfg.TestNet3Params)
	if err != nil {
		fmt.Println(err)
		return
	}

//插入一个输出10个BTC到钱包地址的无链接事务
//在输出端0。
	err = s.InsertTx(b, exampleTxRecordA, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = s.AddCredit(b, exampleTxRecordA, nil, 0, false)
	if err != nil {
		fmt.Println(err)
		return
	}

//插入花费输出的第二个事务，并创建两个
//输出。将第二个（5 BTC）标记为钱包兑换。
	err = s.InsertTx(b, exampleTxRecordB, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = s.AddCredit(b, exampleTxRecordB, nil, 1, true)
	if err != nil {
		fmt.Println(err)
		return
	}

//在高度为100的块中挖掘每个事务。
	err = s.InsertTx(b, exampleTxRecordA, &exampleBlock100)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = s.InsertTx(b, exampleTxRecordB, &exampleBlock100)
	if err != nil {
		fmt.Println(err)
		return
	}

//打印一个确认余额。
	bal, err := s.Balance(b, 1, 100)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(bal)

//获取未暂停的输出。
	utxos, err := s.UnspentOutputs(b)
	if err != nil {
		fmt.Println(err)
	}
	expectedOutPoint := wire.OutPoint{Hash: exampleTxRecordB.Hash, Index: 1}
	for _, utxo := range utxos {
		fmt.Println(utxo.OutPoint == expectedOutPoint)
	}

//输出：
//5 BTC
//真
}
