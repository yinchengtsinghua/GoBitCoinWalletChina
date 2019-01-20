
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
 *版权所有（c）2014-2017 BTCSuite开发者
 *
 *使用、复制、修改和分发本软件的权限
 *特此授予免费或不收费的目的，前提是上述
 *版权声明和本许可声明出现在所有副本中。
 *
 *本软件按“原样”提供，作者不作任何保证。
 *关于本软件，包括
 *适销性和适用性。在任何情况下，作者都不对
 *任何特殊、直接、间接或后果性损害或任何损害
 *因使用、数据或利润损失而导致的任何情况，无论是在
 *因以下原因引起的合同诉讼、疏忽或其他侵权行为：
 *或与本软件的使用或性能有关。
 **/


package votingpool_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/votingpool"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/walletdb"
	_ "github.com/btcsuite/btcwallet/walletdb/bdb"
	"github.com/btcsuite/btcwallet/wtxmgr"
)

var (
	pubPassphrase  = []byte("pubPassphrase")
	privPassphrase = []byte("privPassphrase")
	seed           = bytes.Repeat([]byte{0x2a, 0x64, 0xdf, 0x08}, 8)
	fastScrypt     = &waddrmgr.ScryptOptions{N: 16, R: 8, P: 1}
)

func createWaddrmgr(ns walletdb.ReadWriteBucket, params *chaincfg.Params) (*waddrmgr.Manager, error) {
	err := waddrmgr.Create(ns, seed, pubPassphrase, privPassphrase, params,
		fastScrypt, time.Now())
	if err != nil {
		return nil, err
	}
	return waddrmgr.Open(ns, pubPassphrase, params)
}

func ExampleCreate() {
//新建walletdb.db。有关如何操作的说明，请参见WalletDB文档。
//这样做。
	db, dbTearDown, err := createWalletDB()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer dbTearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer dbtx.Commit()

//为地址管理器创建新的walletdb命名空间。
	mgrNamespace, err := dbtx.CreateTopLevelBucket([]byte("waddrmgr"))
	if err != nil {
		fmt.Println(err)
		return
	}

//创建地址管理器。
	mgr, err := createWaddrmgr(mgrNamespace, &chaincfg.MainNetParams)
	if err != nil {
		fmt.Println(err)
		return
	}

//为投票池创建walletdb命名空间。
	vpNamespace, err := dbtx.CreateTopLevelBucket([]byte("votingpool"))
	if err != nil {
		fmt.Println(err)
		return
	}

//创建投票池。
	_, err = votingpool.Create(vpNamespace, mgr, []byte{0x00})
	if err != nil {
		fmt.Println(err)
		return
	}

//输出：
//
}

//这个例子演示了如何用一个
//并获得该系列的存款地址。
func Example_depositAddress() {
//创建地址管理器和VotingPool数据库命名空间。参见示例
//有关如何执行此操作的详细信息，请参阅create（）函数。
	teardown, db, mgr := exampleCreateDBAndMgr()
	defer teardown()

	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := votingpoolNamespace(tx)

//创建投票池。
		pool, err := votingpool.Create(ns, mgr, []byte{0x00})
		if err != nil {
			return err
		}

//创建3个系列中的2个。
		seriesID := uint32(1)
		requiredSignatures := uint32(2)
		pubKeys := []string{
			"xpub661MyMwAqRbcFDDrR5jY7LqsRioFDwg3cLjc7tML3RRcfYyhXqqgCH5SqMSQdpQ1Xh8EtVwcfm8psD8zXKPcRaCVSY4GCqbb3aMEs27GitE",
			"xpub661MyMwAqRbcGsxyD8hTmJFtpmwoZhy4NBBVxzvFU8tDXD2ME49A6JjQCYgbpSUpHGP1q4S2S1Pxv2EqTjwfERS5pc9Q2yeLkPFzSgRpjs9",
			"xpub661MyMwAqRbcEbc4uYVXvQQpH9L3YuZLZ1gxCmj59yAhNy33vXxbXadmRpx5YZEupNSqWRrR7PqU6duS2FiVCGEiugBEa5zuEAjsyLJjKCh",
		}
		err = pool.CreateSeries(ns, votingpool.CurrentVersion, seriesID, requiredSignatures, pubKeys)
		if err != nil {
			return err
		}

//创建存款地址。
		addr, err := pool.DepositScriptAddress(seriesID, votingpool.Branch(0), votingpool.Index(1))
		if err != nil {
			return err
		}
		fmt.Println("Generated deposit address:", addr.EncodeAddress())
		return nil
	})
	if err != nil {
		fmt.Println(err)
		return
	}

//输出：
//
}

//此示例演示如何通过加载私有
//系列的一个公钥的键。
func Example_empowerSeries() {
//创建地址管理器和VotingPool数据库命名空间。参见示例
//有关如何执行此操作的详细信息，请参阅create（）函数。
	teardown, db, mgr := exampleCreateDBAndMgr()
	defer teardown()

//创建池和序列。有关详细信息，请参阅DepositAddress示例
//
	pool, seriesID := exampleCreatePoolAndSeries(db, mgr)

	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := votingpoolNamespace(tx)
		addrmgrNs := addrmgrNamespace(tx)

//现在用它的一个私钥来增强这个系列。注意按顺序
//为此，我们需要解锁地址管理器。
		err := mgr.Unlock(addrmgrNs, privPassphrase)
		if err != nil {
			return err
		}
		defer mgr.Lock()
		privKey := "xprv9s21ZrQH143K2j9PK4CXkCu8sgxkpUxCF7p1KVwiV5tdnkeYzJXReUkxz5iB2FUzTXC1L15abCDG4RMxSYT5zhm67uvsnLYxuDhZfoFcB6a"
		return pool.EmpowerSeries(ns, seriesID, privKey)
	})
	if err != nil {
		fmt.Println(err)
		return
	}

//输出：
//
}

//此示例演示如何使用pool.startRetraction方法。
func Example_startWithdrawal() {
//创建地址管理器和VotingPool数据库命名空间。参见示例
//有关如何执行此操作的详细信息，请参阅create（）函数。
	teardown, db, mgr := exampleCreateDBAndMgr()
	defer teardown()

//创建池和序列。有关详细信息，请参阅DepositAddress示例
//
	pool, seriesID := exampleCreatePoolAndSeries(db, mgr)

	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := votingpoolNamespace(tx)
		addrmgrNs := addrmgrNamespace(tx)
		txmgrNs := txmgrNamespace(tx)

//创建事务存储以供以后使用。
		txstore := exampleCreateTxStore(txmgrNs)

//
		err := mgr.Unlock(addrmgrNs, privPassphrase)
		if err != nil {
			return err
		}
		defer mgr.Lock()

		addr, _ := btcutil.DecodeAddress("1MirQ9bwyQcGVJPwKUgapu5ouK2E2Ey4gX", mgr.ChainParams())
		pkScript, _ := txscript.PayToAddrScript(addr)
		requests := []votingpool.OutputRequest{
			{
				PkScript:    pkScript,
				Address:     addr,
				Amount:      1e6,
				Server:      "server-id",
				Transaction: 123,
			},
		}
		changeStart, err := pool.ChangeAddress(seriesID, votingpool.Index(0))
		if err != nil {
			return err
		}
//
//序列，我们无法为未使用的
//分支/IDX对。
		err = pool.EnsureUsedAddr(ns, addrmgrNs, seriesID, votingpool.Branch(1), votingpool.Index(0))
		if err != nil {
			return err
		}
		startAddr, err := pool.WithdrawalAddress(ns, addrmgrNs, seriesID, votingpool.Branch(1), votingpool.Index(0))
		if err != nil {
			return err
		}
		lastSeriesID := seriesID
		dustThreshold := btcutil.Amount(1e4)
		currentBlock := int32(19432)
		roundID := uint32(0)
		_, err = pool.StartWithdrawal(ns, addrmgrNs,
			roundID, requests, *startAddr, lastSeriesID, *changeStart, txstore, txmgrNs, currentBlock,
			dustThreshold)
		return err
	})
	if err != nil {
		fmt.Println(err)
		return
	}

//输出：
//
}

func createWalletDB() (walletdb.DB, func(), error) {
	dir, err := ioutil.TempDir("", "votingpool_example")
	if err != nil {
		return nil, nil, err
	}
	db, err := walletdb.Create("bdb", filepath.Join(dir, "wallet.db"))
	if err != nil {
		return nil, nil, err
	}
	dbTearDown := func() {
		db.Close()
		os.RemoveAll(dir)
	}
	return db, dbTearDown, nil
}

var (
	addrmgrNamespaceKey    = []byte("addrmgr")
	txmgrNamespaceKey      = []byte("txmgr")
	votingpoolNamespaceKey = []byte("votingpool")
)

func addrmgrNamespace(dbtx walletdb.ReadWriteTx) walletdb.ReadWriteBucket {
	return dbtx.ReadWriteBucket(addrmgrNamespaceKey)
}

func txmgrNamespace(dbtx walletdb.ReadWriteTx) walletdb.ReadWriteBucket {
	return dbtx.ReadWriteBucket(txmgrNamespaceKey)
}

func votingpoolNamespace(dbtx walletdb.ReadWriteTx) walletdb.ReadWriteBucket {
	return dbtx.ReadWriteBucket(votingpoolNamespaceKey)
}

func exampleCreateDBAndMgr() (teardown func(), db walletdb.DB, mgr *waddrmgr.Manager) {
	db, dbTearDown, err := createWalletDB()
	if err != nil {
		dbTearDown()
		panic(err)
	}

//为地址管理器创建新的walletdb命名空间。
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs, err := tx.CreateTopLevelBucket(addrmgrNamespaceKey)
		if err != nil {
			return err
		}
		_, err = tx.CreateTopLevelBucket(votingpoolNamespaceKey)
		if err != nil {
			return err
		}
		_, err = tx.CreateTopLevelBucket(txmgrNamespaceKey)
		if err != nil {
			return err
		}
//创建地址管理器
		mgr, err = createWaddrmgr(addrmgrNs, &chaincfg.MainNetParams)
		return err
	})
	if err != nil {
		dbTearDown()
		panic(err)
	}

	teardown = func() {
		mgr.Close()
		dbTearDown()
	}

	return teardown, db, mgr
}

func exampleCreatePoolAndSeries(db walletdb.DB, mgr *waddrmgr.Manager) (pool *votingpool.Pool, seriesID uint32) {
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := votingpoolNamespace(tx)
		var err error
		pool, err = votingpool.Create(ns, mgr, []byte{0x00})
		if err != nil {
			return err
		}

//创建3个系列中的2个。
		seriesID = uint32(1)
		requiredSignatures := uint32(2)
		pubKeys := []string{
			"xpub661MyMwAqRbcFDDrR5jY7LqsRioFDwg3cLjc7tML3RRcfYyhXqqgCH5SqMSQdpQ1Xh8EtVwcfm8psD8zXKPcRaCVSY4GCqbb3aMEs27GitE",
			"xpub661MyMwAqRbcGsxyD8hTmJFtpmwoZhy4NBBVxzvFU8tDXD2ME49A6JjQCYgbpSUpHGP1q4S2S1Pxv2EqTjwfERS5pc9Q2yeLkPFzSgRpjs9",
			"xpub661MyMwAqRbcEbc4uYVXvQQpH9L3YuZLZ1gxCmj59yAhNy33vXxbXadmRpx5YZEupNSqWRrR7PqU6duS2FiVCGEiugBEa5zuEAjsyLJjKCh",
		}
		err = pool.CreateSeries(ns, votingpool.CurrentVersion, seriesID, requiredSignatures, pubKeys)
		if err != nil {
			return err
		}
		return pool.ActivateSeries(ns, seriesID)
	})
	if err != nil {
		panic(err)
	}

	return pool, seriesID
}

func exampleCreateTxStore(ns walletdb.ReadWriteBucket) *wtxmgr.Store {
	err := wtxmgr.Create(ns)
	if err != nil {
		panic(err)
	}
	s, err := wtxmgr.Open(ns, &chaincfg.MainNetParams)
	if err != nil {
		panic(err)
	}
	return s
}
