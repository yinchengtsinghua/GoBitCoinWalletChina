
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
 *版权所有（c）2014-2016 BTCSuite开发者
 *
 *使用、复制、修改和分发本软件的权限
 *特此授予免费或不收费的目的，前提是上述
 *版权声明和本许可声明出现在所有副本中。
 
 *本软件按“原样”提供，作者不作任何保证。
 *关于本软件，包括
 *适销性和适用性。在任何情况下，作者都不对
 *任何特殊、直接、间接或后果性损害或任何损害
 *因使用、数据或利润损失而导致的任何情况，无论是在
 *因以下原因引起的合同诉讼、疏忽或其他侵权行为：
 *或与本软件的使用或性能有关。
 **/


//帮助器创建要在测试中使用的参数化对象。

package votingpool

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/btcsuite/btcwallet/wtxmgr"
)

var (
//种子是用于创建扩展键的主种子。
	seed           = bytes.Repeat([]byte{0x2a, 0x64, 0xdf, 0x08}, 8)
	pubPassphrase  = []byte("_DJr{fL4H0O}*-0\n:V1izc)(6BomK")
	privPassphrase = []byte("81lUHXnOMZ@?XXd7O9xyDIWIbXX-lj")
	uniqueCounter  = uint32(0)
//创建所有测试输入的块高度。
	TstInputsBlock = int32(10)
)

func getUniqueID() uint32 {
	return atomic.AddUint32(&uniqueCounter, 1)
}

//创建一个具有给定输入和输出量的取款机。
func createWithdrawalTx(t *testing.T, dbtx walletdb.ReadWriteTx, pool *Pool, inputAmounts []int64, outputAmounts []int64) *withdrawalTx {
	net := pool.Manager().ChainParams()
	tx := newWithdrawalTx(defaultTxOptions)
	_, credits := TstCreateCreditsOnNewSeries(t, dbtx, pool, inputAmounts)
	for _, c := range credits {
		tx.addInput(c)
	}
	for i, amount := range outputAmounts {
		request := TstNewOutputRequest(
			t, uint32(i), "34eVkREKgvvGASZW7hkgE2uNc1yycntMK6", btcutil.Amount(amount), net)
		tx.addOutput(request)
	}
	return tx
}

func createMsgTx(pkScript []byte, amts []int64) *wire.MsgTx {
	msgtx := &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  chainhash.Hash{},
					Index: 0xffffffff,
				},
				SignatureScript: []byte{txscript.OP_NOP},
				Sequence:        0xffffffff,
			},
		},
		LockTime: 0,
	}

	for _, amt := range amts {
		msgtx.AddTxOut(wire.NewTxOut(amt, pkScript))
	}
	return msgtx
}

func TstNewDepositScript(t *testing.T, p *Pool, seriesID uint32, branch Branch, idx Index) []byte {
	script, err := p.DepositScript(seriesID, branch, idx)
	if err != nil {
		t.Fatalf("Failed to create deposit script for series %d, branch %d, index %d: %v",
			seriesID, branch, idx, err)
	}
	return script
}

func TstRNamespaces(tx walletdb.ReadTx) (votingpoolNs, addrmgrNs walletdb.ReadBucket) {
	return tx.ReadBucket(votingpoolNamespaceKey), tx.ReadBucket(addrmgrNamespaceKey)
}

func TstRWNamespaces(tx walletdb.ReadWriteTx) (votingpoolNs, addrmgrNs walletdb.ReadWriteBucket) {
	return tx.ReadWriteBucket(votingpoolNamespaceKey), tx.ReadWriteBucket(addrmgrNamespaceKey)
}

func TstTxStoreRWNamespace(tx walletdb.ReadWriteTx) walletdb.ReadWriteBucket {
	return tx.ReadWriteBucket(txmgrNamespaceKey)
}

//tstensuresedaddr确保由给定的序列/分支定义的地址，以及
//index==0..idx存在于给定池的已用地址集中。
func TstEnsureUsedAddr(t *testing.T, dbtx walletdb.ReadWriteTx, p *Pool, seriesID uint32, branch Branch, idx Index) []byte {
	ns, addrmgrNs := TstRWNamespaces(dbtx)
	addr, err := p.getUsedAddr(ns, addrmgrNs, seriesID, branch, idx)
	if err != nil {
		t.Fatal(err)
	} else if addr != nil {
		var script []byte
		TstRunWithManagerUnlocked(t, p.Manager(), addrmgrNs, func() {
			script, err = addr.Script()
		})
		if err != nil {
			t.Fatal(err)
		}
		return script
	}
	TstRunWithManagerUnlocked(t, p.Manager(), addrmgrNs, func() {
		err = p.EnsureUsedAddr(ns, addrmgrNs, seriesID, branch, idx)
	})
	if err != nil {
		t.Fatal(err)
	}
	return TstNewDepositScript(t, p, seriesID, branch, idx)
}

func TstCreatePkScript(t *testing.T, dbtx walletdb.ReadWriteTx, p *Pool, seriesID uint32, branch Branch, idx Index) []byte {
	script := TstEnsureUsedAddr(t, dbtx, p, seriesID, branch, idx)
	addr, err := p.addressFor(script)
	if err != nil {
		t.Fatal(err)
	}
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		t.Fatal(err)
	}
	return pkScript
}

type TstSeriesDef struct {
	ReqSigs  uint32
	PubKeys  []string
	PrivKeys []string
	SeriesID uint32
	Inactive bool
}

//TSTCreateSeries为给定切片中的每个定义创建一个新序列
//TSTSseries定义。如果定义包含任何私钥，则序列为
//赋予他们权力。
func TstCreateSeries(t *testing.T, dbtx walletdb.ReadWriteTx, pool *Pool, definitions []TstSeriesDef) {
	ns, addrmgrNs := TstRWNamespaces(dbtx)
	for _, def := range definitions {
		err := pool.CreateSeries(ns, CurrentVersion, def.SeriesID, def.ReqSigs, def.PubKeys)
		if err != nil {
			t.Fatalf("Cannot creates series %d: %v", def.SeriesID, err)
		}
		TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
			for _, key := range def.PrivKeys {
				if err := pool.EmpowerSeries(ns, def.SeriesID, key); err != nil {
					t.Fatal(err)
				}
			}
		})
		pool.Series(def.SeriesID).active = !def.Inactive
	}
}

func TstCreateMasterKey(t *testing.T, seed []byte) *hdkeychain.ExtendedKey {
	key, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

//CreateMasterKeys创建具有唯一种子的计数主扩展键。
func createMasterKeys(t *testing.T, count int) []*hdkeychain.ExtendedKey {
	keys := make([]*hdkeychain.ExtendedKey, count)
	for i := range keys {
		keys[i] = TstCreateMasterKey(t, bytes.Repeat(uint32ToBytes(getUniqueID()), 4))
	}
	return keys
}

//tstcreateseriesdef创建具有唯一seriesd的tstsseriesdef，给定
//从private列表中提取的reqsigs和原始公钥/私钥
//钥匙。新系列将获得所有私钥的授权。
func TstCreateSeriesDef(t *testing.T, pool *Pool, reqSigs uint32, keys []*hdkeychain.ExtendedKey) TstSeriesDef {
	pubKeys := make([]string, len(keys))
	privKeys := make([]string, len(keys))
	for i, key := range keys {
		privKeys[i] = key.String()
		pubkey, _ := key.Neuter()
		pubKeys[i] = pubkey.String()
	}
	seriesID := uint32(len(pool.seriesLookup)) + 1
	return TstSeriesDef{
		ReqSigs: reqSigs, SeriesID: seriesID, PubKeys: pubKeys, PrivKeys: privKeys}
}

func TstCreatePoolAndTxStore(t *testing.T) (tearDown func(), db walletdb.DB, pool *Pool, store *wtxmgr.Store) {
	teardown, db, pool := TstCreatePool(t)
	store = TstCreateTxStore(t, db)
	return teardown, db, pool, store
}

//tstcreatecreditsonnewseries创建新系列（具有唯一ID）和
//在branch==1且index==0的情况下，锁定到序列地址的信贷切片。
//新系列将使用三分之二的配置，并将获得
//它的所有私钥。
func TstCreateCreditsOnNewSeries(t *testing.T, dbtx walletdb.ReadWriteTx, pool *Pool, amounts []int64) (uint32, []credit) {
	masters := []*hdkeychain.ExtendedKey{
		TstCreateMasterKey(t, bytes.Repeat(uint32ToBytes(getUniqueID()), 4)),
		TstCreateMasterKey(t, bytes.Repeat(uint32ToBytes(getUniqueID()), 4)),
		TstCreateMasterKey(t, bytes.Repeat(uint32ToBytes(getUniqueID()), 4)),
	}
	def := TstCreateSeriesDef(t, pool, 2, masters)
	TstCreateSeries(t, dbtx, pool, []TstSeriesDef{def})
	return def.SeriesID, TstCreateSeriesCredits(t, dbtx, pool, def.SeriesID, amounts)
}

//tstcreateseriescredits为金额中的每个项目创建新的贷方
//切片，锁定到给定序列的地址，分支==1，索引==0。
func TstCreateSeriesCredits(t *testing.T, dbtx walletdb.ReadWriteTx, pool *Pool, seriesID uint32, amounts []int64) []credit {
	addr := TstNewWithdrawalAddress(t, dbtx, pool, seriesID, Branch(1), Index(0))
	pkScript, err := txscript.PayToAddrScript(addr.addr)
	if err != nil {
		t.Fatal(err)
	}
	msgTx := createMsgTx(pkScript, amounts)
	txHash := msgTx.TxHash()
	credits := make([]credit, len(amounts))
	for i := range msgTx.TxOut {
		c := wtxmgr.Credit{
			OutPoint: wire.OutPoint{
				Hash:  txHash,
				Index: uint32(i),
			},
			BlockMeta: wtxmgr.BlockMeta{
				Block: wtxmgr.Block{Height: TstInputsBlock},
			},
			Amount:   btcutil.Amount(msgTx.TxOut[i].Value),
			PkScript: msgTx.TxOut[i].PkScript,
		}
		credits[i] = newCredit(c, *addr)
	}
	return credits
}

//tstcreateseriescreditsonstore在给定存储中为插入新的信用证
//金额切片中的每个项目。这些信用卡被锁定在投票池
//由给定的序列ID组成的地址，分支==1，索引==0。
func TstCreateSeriesCreditsOnStore(t *testing.T, dbtx walletdb.ReadWriteTx, pool *Pool, seriesID uint32, amounts []int64,
	store *wtxmgr.Store) []credit {
	branch := Branch(1)
	idx := Index(0)
	pkScript := TstCreatePkScript(t, dbtx, pool, seriesID, branch, idx)
	eligible := make([]credit, len(amounts))
	for i, credit := range TstCreateCreditsOnStore(t, dbtx, store, pkScript, amounts) {
		eligible[i] = newCredit(credit, *TstNewWithdrawalAddress(t, dbtx, pool, seriesID, branch, idx))
	}
	return eligible
}

//tstcreatecreditsonstore在给定存储中为插入新的信用
//金额切片中的每个项目。
func TstCreateCreditsOnStore(t *testing.T, dbtx walletdb.ReadWriteTx, s *wtxmgr.Store, pkScript []byte, amounts []int64) []wtxmgr.Credit {
	msgTx := createMsgTx(pkScript, amounts)
	meta := &wtxmgr.BlockMeta{
		Block: wtxmgr.Block{Height: TstInputsBlock},
	}

	rec, err := wtxmgr.NewTxRecordFromMsgTx(msgTx, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	txmgrNs := dbtx.ReadWriteBucket(txmgrNamespaceKey)
	if err := s.InsertTx(txmgrNs, rec, meta); err != nil {
		t.Fatal("Failed to create inputs: ", err)
	}

	credits := make([]wtxmgr.Credit, len(msgTx.TxOut))
	for i := range msgTx.TxOut {
		if err := s.AddCredit(txmgrNs, rec, meta, uint32(i), false); err != nil {
			t.Fatal("Failed to create inputs: ", err)
		}
		credits[i] = wtxmgr.Credit{
			OutPoint: wire.OutPoint{
				Hash:  rec.Hash,
				Index: uint32(i),
			},
			BlockMeta: *meta,
			Amount:    btcutil.Amount(msgTx.TxOut[i].Value),
			PkScript:  msgTx.TxOut[i].PkScript,
		}
	}
	return credits
}

var (
	addrmgrNamespaceKey    = []byte("waddrmgr")
	votingpoolNamespaceKey = []byte("votingpool")
	txmgrNamespaceKey      = []byte("testtxstore")
)

//tstcreatepool在新的walletdb上创建一个池并返回它。它也
//返回一个TearDown函数，该函数关闭管理器并删除目录
//用于存储数据库。
func TstCreatePool(t *testing.T) (tearDownFunc func(), db walletdb.DB, pool *Pool) {
//这最终应该转移到其他地方，因为不是所有的测试
//调用此函数，但现在唯一的选项是
//t.parallel（）在每个测试中调用。
	t.Parallel()

//创建新的Wallet数据库和地址管理器。
	dir, err := ioutil.TempDir("", "pool_test")
	if err != nil {
		t.Fatalf("Failed to create db dir: %v", err)
	}
	db, err = walletdb.Create("bdb", filepath.Join(dir, "wallet.db"))
	if err != nil {
		t.Fatalf("Failed to create wallet DB: %v", err)
	}
	var addrMgr *waddrmgr.Manager
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs, err := tx.CreateTopLevelBucket(addrmgrNamespaceKey)
		if err != nil {
			return err
		}
		votingpoolNs, err := tx.CreateTopLevelBucket(votingpoolNamespaceKey)
		if err != nil {
			return err
		}
		fastScrypt := &waddrmgr.ScryptOptions{N: 16, R: 8, P: 1}
		err = waddrmgr.Create(addrmgrNs, seed, pubPassphrase, privPassphrase,
			&chaincfg.MainNetParams, fastScrypt, time.Now())
		if err != nil {
			return err
		}
		addrMgr, err = waddrmgr.Open(addrmgrNs, pubPassphrase, &chaincfg.MainNetParams)
		if err != nil {
			return err
		}
		pool, err = Create(votingpoolNs, addrMgr, []byte{0x00})
		return err
	})
	if err != nil {
		t.Fatalf("Could not set up DB: %v", err)
	}
	tearDownFunc = func() {
		addrMgr.Close()
		db.Close()
		os.RemoveAll(dir)
	}
	return tearDownFunc, db, pool
}

func TstCreateTxStore(t *testing.T, db walletdb.DB) *wtxmgr.Store {
	var store *wtxmgr.Store
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		txmgrNs, err := tx.CreateTopLevelBucket(txmgrNamespaceKey)
		if err != nil {
			return err
		}
		err = wtxmgr.Create(txmgrNs)
		if err != nil {
			return err
		}
		store, err = wtxmgr.Open(txmgrNs, &chaincfg.MainNetParams)
		return err
	})
	if err != nil {
		t.Fatalf("Failed to create txmgr: %v", err)
	}
	return store
}

func TstNewOutputRequest(t *testing.T, transaction uint32, address string, amount btcutil.Amount,
	net *chaincfg.Params) OutputRequest {
	addr, err := btcutil.DecodeAddress(address, net)
	if err != nil {
		t.Fatalf("Unable to decode address %s", address)
	}
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		t.Fatalf("Unable to generate pkScript for %v", addr)
	}
	return OutputRequest{
		PkScript:    pkScript,
		Address:     addr,
		Amount:      amount,
		Server:      "server",
		Transaction: transaction,
	}
}

func TstNewWithdrawalOutput(r OutputRequest, status outputStatus,
	outpoints []OutBailmentOutpoint) *WithdrawalOutput {
	output := &WithdrawalOutput{
		request:   r,
		status:    status,
		outpoints: outpoints,
	}
	return output
}

func TstNewWithdrawalAddress(t *testing.T, dbtx walletdb.ReadWriteTx, p *Pool, seriesID uint32, branch Branch,
	index Index) (addr *WithdrawalAddress) {
	TstEnsureUsedAddr(t, dbtx, p, seriesID, branch, index)
	ns, addrmgrNs := TstRNamespaces(dbtx)
	var err error
	TstRunWithManagerUnlocked(t, p.Manager(), addrmgrNs, func() {
		addr, err = p.WithdrawalAddress(ns, addrmgrNs, seriesID, branch, index)
	})
	if err != nil {
		t.Fatalf("Failed to get WithdrawalAddress: %v", err)
	}
	return addr
}

func TstNewChangeAddress(t *testing.T, p *Pool, seriesID uint32, idx Index) (addr *ChangeAddress) {
	addr, err := p.ChangeAddress(seriesID, idx)
	if err != nil {
		t.Fatalf("Failed to get ChangeAddress: %v", err)
	}
	return addr
}

func TstConstantFee(fee btcutil.Amount) func() btcutil.Amount {
	return func() btcutil.Amount { return fee }
}

func createAndFulfillWithdrawalRequests(t *testing.T, dbtx walletdb.ReadWriteTx, pool *Pool, roundID uint32) withdrawalInfo {

	params := pool.Manager().ChainParams()
	seriesID, eligible := TstCreateCreditsOnNewSeries(t, dbtx, pool, []int64{2e6, 4e6})
	requests := []OutputRequest{
		TstNewOutputRequest(t, 1, "34eVkREKgvvGASZW7hkgE2uNc1yycntMK6", 3e6, params),
		TstNewOutputRequest(t, 2, "3PbExiaztsSYgh6zeMswC49hLUwhTQ86XG", 2e6, params),
	}
	changeStart := TstNewChangeAddress(t, pool, seriesID, 0)
	dustThreshold := btcutil.Amount(1e4)
	startAddr := TstNewWithdrawalAddress(t, dbtx, pool, seriesID, 1, 0)
	lastSeriesID := seriesID
	w := newWithdrawal(roundID, requests, eligible, *changeStart)
	if err := w.fulfillRequests(); err != nil {
		t.Fatal(err)
	}
	return withdrawalInfo{
		requests:      requests,
		startAddress:  *startAddr,
		changeStart:   *changeStart,
		lastSeriesID:  lastSeriesID,
		dustThreshold: dustThreshold,
		status:        *w.status,
	}
}
