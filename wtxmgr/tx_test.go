
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wtxmgr

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/walletdb"
	_ "github.com/btcsuite/btcwallet/walletdb/bdb"
)

//接收到MainNet输出点的事务输出
//61D3696DE4C888730CBE06B0AD8ECB6D72D6108E893895AA9BC067BD7EBA3FAD:0
var (
	TstRecvSerializedTx, _          = hex.DecodeString("010000000114d9ff358894c486b4ae11c2a8cf7851b1df64c53d2e511278eff17c22fb7373000000008c493046022100995447baec31ee9f6d4ec0e05cb2a44f6b817a99d5f6de167d1c75354a946410022100c9ffc23b64d770b0e01e7ff4d25fbc2f1ca8091053078a247905c39fce3760b601410458b8e267add3c1e374cf40f1de02b59213a82e1d84c2b94096e22e2f09387009c96debe1d0bcb2356ffdcf65d2a83d4b34e72c62eccd8490dbf2110167783b2bffffffff0280969800000000001976a914479ed307831d0ac19ebc5f63de7d5f1a430ddb9d88ac38bfaa00000000001976a914dadf9e3484f28b385ddeaa6c575c0c0d18e9788a88ac00000000")
	TstRecvTx, _                    = btcutil.NewTxFromBytes(TstRecvSerializedTx)
	TstRecvTxSpendingTxBlockHash, _ = chainhash.NewHashFromStr("00000000000000017188b968a371bab95aa43522665353b646e41865abae02a4")
	TstRecvAmt                      = int64(10000000)
	TstRecvTxBlockDetails           = &BlockMeta{
		Block: Block{Hash: *TstRecvTxSpendingTxBlockHash, Height: 276425},
		Time:  time.Unix(1387737310, 0),
	}

TstRecvCurrentHeight = int32(284498) //写入时的主网区块链高度
TstRecvTxOutConfirms = 8074          //根据上述区块高度，硬编码确认数量

	TstSpendingSerializedTx, _ = hex.DecodeString("0100000003ad3fba7ebd67c09baa9538898e10d6726dcb8eadb006be0c7388c8e46d69d361000000006b4830450220702c4fbde5532575fed44f8d6e8c3432a2a9bd8cff2f966c3a79b2245a7c88db02210095d6505a57e350720cb52b89a9b56243c15ddfcea0596aedc1ba55d9fb7d5aa0012103cccb5c48a699d3efcca6dae277fee6b82e0229ed754b742659c3acdfed2651f9ffffffffdbd36173f5610e34de5c00ed092174603761595d90190f790e79cda3e5b45bc2010000006b483045022000fa20735e5875e64d05bed43d81b867f3bd8745008d3ff4331ef1617eac7c44022100ad82261fc57faac67fc482a37b6bf18158da0971e300abf5fe2f9fd39e107f58012102d4e1caf3e022757512c204bf09ff56a9981df483aba3c74bb60d3612077c9206ffffffff65536c9d964b6f89b8ef17e83c6666641bc495cb27bab60052f76cd4556ccd0d040000006a473044022068e3886e0299ffa69a1c3ee40f8b6700f5f6d463a9cf9dbf22c055a131fc4abc02202b58957fe19ff1be7a84c458d08016c53fbddec7184ac5e633f2b282ae3420ae012103b4e411b81d32a69fb81178a8ea1abaa12f613336923ee920ffbb1b313af1f4d2ffffffff02ab233200000000001976a91418808b2fbd8d2c6d022aed5cd61f0ce6c0a4cbb688ac4741f011000000001976a914f081088a300c80ce36b717a9914ab5ec8a7d283988ac00000000")
	TstSpendingTx, _           = btcutil.NewTxFromBytes(TstSpendingSerializedTx)
	TstSpendingTxBlockHeight   = int32(279143)
	TstSignedTxBlockHash, _    = chainhash.NewHashFromStr("00000000000000017188b968a371bab95aa43522665353b646e41865abae02a4")
	TstSignedTxBlockDetails    = &BlockMeta{
		Block: Block{Hash: *TstSignedTxBlockHash, Height: TstSpendingTxBlockHeight},
		Time:  time.Unix(1389114091, 0),
	}
)

func testDB() (walletdb.DB, func(), error) {
	tmpDir, err := ioutil.TempDir("", "wtxmgr_test")
	if err != nil {
		return nil, func() {}, err
	}
	db, err := walletdb.Create("bdb", filepath.Join(tmpDir, "db"))
	return db, func() { os.RemoveAll(tmpDir) }, err
}

var namespaceKey = []byte("txstore")

func testStore() (*Store, walletdb.DB, func(), error) {
	tmpDir, err := ioutil.TempDir("", "wtxmgr_test")
	if err != nil {
		return nil, nil, func() {}, err
	}

	db, err := walletdb.Create("bdb", filepath.Join(tmpDir, "db"))
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, nil, nil, err
	}

	teardown := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	var s *Store
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns, err := tx.CreateTopLevelBucket(namespaceKey)
		if err != nil {
			return err
		}
		err = Create(ns)
		if err != nil {
			return err
		}
		s, err = Open(ns, &chaincfg.TestNet3Params)
		return err
	})

	return s, db, teardown, err
}

func serializeTx(tx *btcutil.Tx) []byte {
	var buf bytes.Buffer
	err := tx.MsgTx().Serialize(&buf)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestInsertsCreditsDebitsRollbacks(t *testing.T) {
	t.Parallel()

//创建收到的区块链交易的双倍支出。
	dupRecvTx, _ := btcutil.NewTxFromBytes(TstRecvSerializedTx)
//将txout amount切换到1 btc。事务存储没有
//验证txs，这样就可以测试双倍开销了
//去除。
	TstDupRecvAmount := int64(1e8)
	newDupMsgTx := dupRecvTx.MsgTx()
	newDupMsgTx.TxOut[0].Value = TstDupRecvAmount
	TstDoubleSpendTx := btcutil.NewTx(newDupMsgTx)
	TstDoubleSpendSerializedTx := serializeTx(TstDoubleSpendTx)

//创建一个“已签名”（带有无效的sigs）tx，其输出为0
//双倍消费。
	spendingTx := wire.NewMsgTx(wire.TxVersion)
	spendingTxIn := wire.NewTxIn(wire.NewOutPoint(TstDoubleSpendTx.Hash(), 0), []byte{0, 1, 2, 3, 4}, nil)
	spendingTx.AddTxIn(spendingTxIn)
	spendingTxOut1 := wire.NewTxOut(1e7, []byte{5, 6, 7, 8, 9})
	spendingTxOut2 := wire.NewTxOut(9e7, []byte{10, 11, 12, 13, 14})
	spendingTx.AddTxOut(spendingTxOut1)
	spendingTx.AddTxOut(spendingTxOut2)
	TstSpendingTx := btcutil.NewTx(spendingTx)
	TstSpendingSerializedTx := serializeTx(TstSpendingTx)
	var _ = TstSpendingTx

	tests := []struct {
		name     string
		f        func(*Store, walletdb.ReadWriteBucket) (*Store, error)
		bal, unc btcutil.Amount
		unspents map[wire.OutPoint]struct{}
		unmined  map[chainhash.Hash]struct{}
	}{
		{
			name: "new store",
			f: func(s *Store, ns walletdb.ReadWriteBucket) (*Store, error) {
				return s, nil
			},
			bal:      0,
			unc:      0,
			unspents: map[wire.OutPoint]struct{}{},
			unmined:  map[chainhash.Hash]struct{}{},
		},
		{
			name: "txout insert",
			f: func(s *Store, ns walletdb.ReadWriteBucket) (*Store, error) {
				rec, err := NewTxRecord(TstRecvSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(ns, rec, nil)
				if err != nil {
					return nil, err
				}

				err = s.AddCredit(ns, rec, nil, 0, false)
				return s, err
			},
			bal: 0,
			unc: btcutil.Amount(TstRecvTx.MsgTx().TxOut[0].Value),
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{
					Hash:  *TstRecvTx.Hash(),
					Index: 0,
				}: {},
			},
			unmined: map[chainhash.Hash]struct{}{
				*TstRecvTx.Hash(): {},
			},
		},
		{
			name: "insert duplicate unconfirmed",
			f: func(s *Store, ns walletdb.ReadWriteBucket) (*Store, error) {
				rec, err := NewTxRecord(TstRecvSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(ns, rec, nil)
				if err != nil {
					return nil, err
				}

				err = s.AddCredit(ns, rec, nil, 0, false)
				return s, err
			},
			bal: 0,
			unc: btcutil.Amount(TstRecvTx.MsgTx().TxOut[0].Value),
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{
					Hash:  *TstRecvTx.Hash(),
					Index: 0,
				}: {},
			},
			unmined: map[chainhash.Hash]struct{}{
				*TstRecvTx.Hash(): {},
			},
		},
		{
			name: "confirmed txout insert",
			f: func(s *Store, ns walletdb.ReadWriteBucket) (*Store, error) {
				rec, err := NewTxRecord(TstRecvSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(ns, rec, TstRecvTxBlockDetails)
				if err != nil {
					return nil, err
				}

				err = s.AddCredit(ns, rec, TstRecvTxBlockDetails, 0, false)
				return s, err
			},
			bal: btcutil.Amount(TstRecvTx.MsgTx().TxOut[0].Value),
			unc: 0,
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{
					Hash:  *TstRecvTx.Hash(),
					Index: 0,
				}: {},
			},
			unmined: map[chainhash.Hash]struct{}{},
		},
		{
			name: "insert duplicate confirmed",
			f: func(s *Store, ns walletdb.ReadWriteBucket) (*Store, error) {
				rec, err := NewTxRecord(TstRecvSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(ns, rec, TstRecvTxBlockDetails)
				if err != nil {
					return nil, err
				}

				err = s.AddCredit(ns, rec, TstRecvTxBlockDetails, 0, false)
				return s, err
			},
			bal: btcutil.Amount(TstRecvTx.MsgTx().TxOut[0].Value),
			unc: 0,
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{
					Hash:  *TstRecvTx.Hash(),
					Index: 0,
				}: {},
			},
			unmined: map[chainhash.Hash]struct{}{},
		},
		{
			name: "rollback confirmed credit",
			f: func(s *Store, ns walletdb.ReadWriteBucket) (*Store, error) {
				err := s.Rollback(ns, TstRecvTxBlockDetails.Height)
				return s, err
			},
			bal: 0,
			unc: btcutil.Amount(TstRecvTx.MsgTx().TxOut[0].Value),
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{
					Hash:  *TstRecvTx.Hash(),
					Index: 0,
				}: {},
			},
			unmined: map[chainhash.Hash]struct{}{
				*TstRecvTx.Hash(): {},
			},
		},
		{
			name: "insert confirmed double spend",
			f: func(s *Store, ns walletdb.ReadWriteBucket) (*Store, error) {
				rec, err := NewTxRecord(TstDoubleSpendSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(ns, rec, TstRecvTxBlockDetails)
				if err != nil {
					return nil, err
				}

				err = s.AddCredit(ns, rec, TstRecvTxBlockDetails, 0, false)
				return s, err
			},
			bal: btcutil.Amount(TstDoubleSpendTx.MsgTx().TxOut[0].Value),
			unc: 0,
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{
					Hash:  *TstDoubleSpendTx.Hash(),
					Index: 0,
				}: {},
			},
			unmined: map[chainhash.Hash]struct{}{},
		},
		{
			name: "insert unconfirmed debit",
			f: func(s *Store, ns walletdb.ReadWriteBucket) (*Store, error) {
				rec, err := NewTxRecord(TstSpendingSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(ns, rec, nil)
				return s, err
			},
			bal:      0,
			unc:      0,
			unspents: map[wire.OutPoint]struct{}{},
			unmined: map[chainhash.Hash]struct{}{
				*TstSpendingTx.Hash(): {},
			},
		},
		{
			name: "insert unconfirmed debit again",
			f: func(s *Store, ns walletdb.ReadWriteBucket) (*Store, error) {
				rec, err := NewTxRecord(TstDoubleSpendSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(ns, rec, TstRecvTxBlockDetails)
				return s, err
			},
			bal:      0,
			unc:      0,
			unspents: map[wire.OutPoint]struct{}{},
			unmined: map[chainhash.Hash]struct{}{
				*TstSpendingTx.Hash(): {},
			},
		},
		{
			name: "insert change (index 0)",
			f: func(s *Store, ns walletdb.ReadWriteBucket) (*Store, error) {
				rec, err := NewTxRecord(TstSpendingSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(ns, rec, nil)
				if err != nil {
					return nil, err
				}

				err = s.AddCredit(ns, rec, nil, 0, true)
				return s, err
			},
			bal: 0,
			unc: btcutil.Amount(TstSpendingTx.MsgTx().TxOut[0].Value),
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{
					Hash:  *TstSpendingTx.Hash(),
					Index: 0,
				}: {},
			},
			unmined: map[chainhash.Hash]struct{}{
				*TstSpendingTx.Hash(): {},
			},
		},
		{
			name: "insert output back to this own wallet (index 1)",
			f: func(s *Store, ns walletdb.ReadWriteBucket) (*Store, error) {
				rec, err := NewTxRecord(TstSpendingSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(ns, rec, nil)
				if err != nil {
					return nil, err
				}
				err = s.AddCredit(ns, rec, nil, 1, true)
				return s, err
			},
			bal: 0,
			unc: btcutil.Amount(TstSpendingTx.MsgTx().TxOut[0].Value + TstSpendingTx.MsgTx().TxOut[1].Value),
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{
					Hash:  *TstSpendingTx.Hash(),
					Index: 0,
				}: {},
				wire.OutPoint{
					Hash:  *TstSpendingTx.Hash(),
					Index: 1,
				}: {},
			},
			unmined: map[chainhash.Hash]struct{}{
				*TstSpendingTx.Hash(): {},
			},
		},
		{
			name: "confirm signed tx",
			f: func(s *Store, ns walletdb.ReadWriteBucket) (*Store, error) {
				rec, err := NewTxRecord(TstSpendingSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(ns, rec, TstSignedTxBlockDetails)
				return s, err
			},
			bal: btcutil.Amount(TstSpendingTx.MsgTx().TxOut[0].Value + TstSpendingTx.MsgTx().TxOut[1].Value),
			unc: 0,
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{
					Hash:  *TstSpendingTx.Hash(),
					Index: 0,
				}: {},
				wire.OutPoint{
					Hash:  *TstSpendingTx.Hash(),
					Index: 1,
				}: {},
			},
			unmined: map[chainhash.Hash]struct{}{},
		},
		{
			name: "rollback after spending tx",
			f: func(s *Store, ns walletdb.ReadWriteBucket) (*Store, error) {
				err := s.Rollback(ns, TstSignedTxBlockDetails.Height+1)
				return s, err
			},
			bal: btcutil.Amount(TstSpendingTx.MsgTx().TxOut[0].Value + TstSpendingTx.MsgTx().TxOut[1].Value),
			unc: 0,
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{
					Hash:  *TstSpendingTx.Hash(),
					Index: 0,
				}: {},
				wire.OutPoint{
					Hash:  *TstSpendingTx.Hash(),
					Index: 1,
				}: {},
			},
			unmined: map[chainhash.Hash]struct{}{},
		},
		{
			name: "rollback spending tx block",
			f: func(s *Store, ns walletdb.ReadWriteBucket) (*Store, error) {
				err := s.Rollback(ns, TstSignedTxBlockDetails.Height)
				return s, err
			},
			bal: 0,
			unc: btcutil.Amount(TstSpendingTx.MsgTx().TxOut[0].Value + TstSpendingTx.MsgTx().TxOut[1].Value),
			unspents: map[wire.OutPoint]struct{}{
				wire.OutPoint{
					Hash:  *TstSpendingTx.Hash(),
					Index: 0,
				}: {},
				wire.OutPoint{
					Hash:  *TstSpendingTx.Hash(),
					Index: 1,
				}: {},
			},
			unmined: map[chainhash.Hash]struct{}{
				*TstSpendingTx.Hash(): {},
			},
		},
		{
			name: "rollback double spend tx block",
			f: func(s *Store, ns walletdb.ReadWriteBucket) (*Store, error) {
				err := s.Rollback(ns, TstRecvTxBlockDetails.Height)
				return s, err
			},
			bal: 0,
			unc: btcutil.Amount(TstSpendingTx.MsgTx().TxOut[0].Value + TstSpendingTx.MsgTx().TxOut[1].Value),
			unspents: map[wire.OutPoint]struct{}{
				*wire.NewOutPoint(TstSpendingTx.Hash(), 0): {},
				*wire.NewOutPoint(TstSpendingTx.Hash(), 1): {},
			},
			unmined: map[chainhash.Hash]struct{}{
				*TstDoubleSpendTx.Hash(): {},
				*TstSpendingTx.Hash():    {},
			},
		},
		{
			name: "insert original recv txout",
			f: func(s *Store, ns walletdb.ReadWriteBucket) (*Store, error) {
				rec, err := NewTxRecord(TstRecvSerializedTx, time.Now())
				if err != nil {
					return nil, err
				}
				err = s.InsertTx(ns, rec, TstRecvTxBlockDetails)
				if err != nil {
					return nil, err
				}
				err = s.AddCredit(ns, rec, TstRecvTxBlockDetails, 0, false)
				return s, err
			},
			bal: btcutil.Amount(TstRecvTx.MsgTx().TxOut[0].Value),
			unc: 0,
			unspents: map[wire.OutPoint]struct{}{
				*wire.NewOutPoint(TstRecvTx.Hash(), 0): {},
			},
			unmined: map[chainhash.Hash]struct{}{},
		},
	}

	s, db, teardown, err := testStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	for _, test := range tests {
		err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
			ns := tx.ReadWriteBucket(namespaceKey)
			tmpStore, err := test.f(s, ns)
			if err != nil {
				t.Fatalf("%s: got error: %v", test.name, err)
			}
			s = tmpStore
			bal, err := s.Balance(ns, 1, TstRecvCurrentHeight)
			if err != nil {
				t.Fatalf("%s: Confirmed Balance failed: %v", test.name, err)
			}
			if bal != test.bal {
				t.Fatalf("%s: balance mismatch: expected: %d, got: %d", test.name, test.bal, bal)
			}
			unc, err := s.Balance(ns, 0, TstRecvCurrentHeight)
			if err != nil {
				t.Fatalf("%s: Unconfirmed Balance failed: %v", test.name, err)
			}
			unc -= bal
			if unc != test.unc {
				t.Fatalf("%s: unconfirmed balance mismatch: expected %d, got %d", test.name, test.unc, unc)
			}

//检查未使用的输出是否符合预期。
			unspent, err := s.UnspentOutputs(ns)
			if err != nil {
				t.Fatalf("%s: failed to fetch unspent outputs: %v", test.name, err)
			}
			for _, cred := range unspent {
				if _, ok := test.unspents[cred.OutPoint]; !ok {
					t.Errorf("%s: unexpected unspent output: %v", test.name, cred.OutPoint)
				}
				delete(test.unspents, cred.OutPoint)
			}
			if len(test.unspents) != 0 {
				t.Fatalf("%s: missing expected unspent output(s)", test.name)
			}

//检查未链接的TXS是否符合预期。
			unmined, err := s.UnminedTxs(ns)
			if err != nil {
				t.Fatalf("%s: cannot load unmined transactions: %v", test.name, err)
			}
			for _, tx := range unmined {
				txHash := tx.TxHash()
				if _, ok := test.unmined[txHash]; !ok {
					t.Fatalf("%s: unexpected unmined tx: %v", test.name, txHash)
				}
				delete(test.unmined, txHash)
			}
			if len(test.unmined) != 0 {
				t.Fatalf("%s: missing expected unmined tx(s)", test.name)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestFindingSpentCredits(t *testing.T) {
	t.Parallel()

	s, db, teardown, err := testStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns := dbtx.ReadWriteBucket(namespaceKey)

//插入将花费的交易和信贷。
	recvRec, err := NewTxRecord(TstRecvSerializedTx, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	err = s.InsertTx(ns, recvRec, TstRecvTxBlockDetails)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(ns, recvRec, TstRecvTxBlockDetails, 0, false)
	if err != nil {
		t.Fatal(err)
	}

//插入花费上述信贷的已确认交易。
	spendingRec, err := NewTxRecord(TstSpendingSerializedTx, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	err = s.InsertTx(ns, spendingRec, TstSignedTxBlockDetails)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(ns, spendingRec, TstSignedTxBlockDetails, 0, false)
	if err != nil {
		t.Fatal(err)
	}

	bal, err := s.Balance(ns, 1, TstSignedTxBlockDetails.Height)
	if err != nil {
		t.Fatal(err)
	}
	expectedBal := btcutil.Amount(TstSpendingTx.MsgTx().TxOut[0].Value)
	if bal != expectedBal {
		t.Fatalf("bad balance: %v != %v", bal, expectedBal)
	}
	unspents, err := s.UnspentOutputs(ns)
	if err != nil {
		t.Fatal(err)
	}
	op := wire.NewOutPoint(TstSpendingTx.Hash(), 0)
	if unspents[0].OutPoint != *op {
		t.Fatal("unspent outpoint doesn't match expected")
	}
	if len(unspents) > 1 {
		t.Fatal("has more than one unspent credit")
	}
}

func newCoinBase(outputValues ...int64) *wire.MsgTx {
	tx := wire.MsgTx{
		TxIn: []*wire.TxIn{
			{
				PreviousOutPoint: wire.OutPoint{Index: ^uint32(0)},
			},
		},
	}
	for _, val := range outputValues {
		tx.TxOut = append(tx.TxOut, &wire.TxOut{Value: val})
	}
	return &tx
}

func spendOutput(txHash *chainhash.Hash, index uint32, outputValues ...int64) *wire.MsgTx {
	tx := wire.MsgTx{
		TxIn: []*wire.TxIn{
			{
				PreviousOutPoint: wire.OutPoint{Hash: *txHash, Index: index},
			},
		},
	}
	for _, val := range outputValues {
		tx.TxOut = append(tx.TxOut, &wire.TxOut{Value: val})
	}
	return &tx
}

func spendOutputs(outputs []wire.OutPoint, outputValues ...int64) *wire.MsgTx {
	tx := &wire.MsgTx{}
	for _, output := range outputs {
		tx.TxIn = append(tx.TxIn, &wire.TxIn{PreviousOutPoint: output})
	}
	for _, value := range outputValues {
		tx.TxOut = append(tx.TxOut, &wire.TxOut{Value: value})
	}

	return tx
}

func TestCoinbases(t *testing.T) {
	t.Parallel()

	s, db, teardown, err := testStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns := dbtx.ReadWriteBucket(namespaceKey)

	b100 := BlockMeta{
		Block: Block{Height: 100},
		Time:  time.Now(),
	}

	cb := newCoinBase(20e8, 10e8, 30e8)
	cbRec, err := NewTxRecordFromMsgTx(cb, b100.Time)
	if err != nil {
		t.Fatal(err)
	}

//插入coinbase并将输出0和2标记为贷方。
	err = s.InsertTx(ns, cbRec, &b100)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(ns, cbRec, &b100, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(ns, cbRec, &b100, 2, false)
	if err != nil {
		t.Fatal(err)
	}

	coinbaseMaturity := int32(chaincfg.TestNet3Params.CoinbaseMaturity)

//如果货币基础不成熟，余额应为0，50 BTC及以上
//成熟度。
//
//深度低于成熟度时的输出永远不包括在内，无论
//所需的确认数。成熟的产出
//深度大于minconf仍被排除在外。
	type balTest struct {
		height  int32
		minConf int32
		bal     btcutil.Amount
	}
	balTests := []balTest{
//下一个街区还不成熟
		{
			height:  b100.Height + coinbaseMaturity - 2,
			minConf: 0,
			bal:     0,
		},
		{
			height:  b100.Height + coinbaseMaturity - 2,
			minConf: coinbaseMaturity,
			bal:     0,
		},

//下一个街区就成熟了
		{
			height:  b100.Height + coinbaseMaturity - 1,
			minConf: 0,
			bal:     50e8,
		},
		{
			height:  b100.Height + coinbaseMaturity - 1,
			minConf: 1,
			bal:     50e8,
		},
		{
			height:  b100.Height + coinbaseMaturity - 1,
			minConf: coinbaseMaturity - 1,
			bal:     50e8,
		},
		{
			height:  b100.Height + coinbaseMaturity - 1,
			minConf: coinbaseMaturity,
			bal:     50e8,
		},
		{
			height:  b100.Height + coinbaseMaturity - 1,
			minConf: coinbaseMaturity + 1,
			bal:     0,
		},

//在这个街区成熟
		{
			height:  b100.Height + coinbaseMaturity,
			minConf: 0,
			bal:     50e8,
		},
		{
			height:  b100.Height + coinbaseMaturity,
			minConf: 1,
			bal:     50e8,
		},
		{
			height:  b100.Height + coinbaseMaturity,
			minConf: coinbaseMaturity,
			bal:     50e8,
		},
		{
			height:  b100.Height + coinbaseMaturity,
			minConf: coinbaseMaturity + 1,
			bal:     50e8,
		},
		{
			height:  b100.Height + coinbaseMaturity,
			minConf: coinbaseMaturity + 2,
			bal:     0,
		},
	}
	for i, tst := range balTests {
		bal, err := s.Balance(ns, tst.minConf, tst.height)
		if err != nil {
			t.Fatalf("Balance test %d: Store.Balance failed: %v", i, err)
		}
		if bal != tst.bal {
			t.Errorf("Balance test %d: Got %v Expected %v", i, bal, tst.bal)
		}
	}
	if t.Failed() {
		t.Fatal("Failed balance checks after inserting coinbase")
	}

//在未链接的事务中使用coinbase tx的输出，当
//下一块将使硬币库成熟。
	spenderATime := time.Now()
	spenderA := spendOutput(&cbRec.Hash, 0, 5e8, 15e8)
	spenderARec, err := NewTxRecordFromMsgTx(spenderA, spenderATime)
	if err != nil {
		t.Fatal(err)
	}
	err = s.InsertTx(ns, spenderARec, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(ns, spenderARec, nil, 0, false)
	if err != nil {
		t.Fatal(err)
	}

	balTests = []balTest{
//下一个街区就成熟了
		{
			height:  b100.Height + coinbaseMaturity - 1,
			minConf: 0,
			bal:     35e8,
		},
		{
			height:  b100.Height + coinbaseMaturity - 1,
			minConf: 1,
			bal:     30e8,
		},
		{
			height:  b100.Height + coinbaseMaturity - 1,
			minConf: coinbaseMaturity,
			bal:     30e8,
		},
		{
			height:  b100.Height + coinbaseMaturity - 1,
			minConf: coinbaseMaturity + 1,
			bal:     0,
		},

//在这个街区成熟
		{
			height:  b100.Height + coinbaseMaturity,
			minConf: 0,
			bal:     35e8,
		},
		{
			height:  b100.Height + coinbaseMaturity,
			minConf: 1,
			bal:     30e8,
		},
		{
			height:  b100.Height + coinbaseMaturity,
			minConf: coinbaseMaturity,
			bal:     30e8,
		},
		{
			height:  b100.Height + coinbaseMaturity,
			minConf: coinbaseMaturity + 1,
			bal:     30e8,
		},
		{
			height:  b100.Height + coinbaseMaturity,
			minConf: coinbaseMaturity + 2,
			bal:     0,
		},
	}
	balTestsBeforeMaturity := balTests
	for i, tst := range balTests {
		bal, err := s.Balance(ns, tst.minConf, tst.height)
		if err != nil {
			t.Fatalf("Balance test %d: Store.Balance failed: %v", i, err)
		}
		if bal != tst.bal {
			t.Errorf("Balance test %d: Got %v Expected %v", i, bal, tst.bal)
		}
	}
	if t.Failed() {
		t.Fatal("Failed balance checks after spending coinbase with unmined transaction")
	}

//挖掘块中的支出交易，使coinbase成熟。
	bMaturity := BlockMeta{
		Block: Block{Height: b100.Height + coinbaseMaturity},
		Time:  time.Now(),
	}
	err = s.InsertTx(ns, spenderARec, &bMaturity)
	if err != nil {
		t.Fatal(err)
	}

	balTests = []balTest{
//成熟高度
		{
			height:  bMaturity.Height,
			minConf: 0,
			bal:     35e8,
		},
		{
			height:  bMaturity.Height,
			minConf: 1,
			bal:     35e8,
		},
		{
			height:  bMaturity.Height,
			minConf: 2,
			bal:     30e8,
		},
		{
			height:  bMaturity.Height,
			minConf: coinbaseMaturity,
			bal:     30e8,
		},
		{
			height:  bMaturity.Height,
			minConf: coinbaseMaturity + 1,
			bal:     30e8,
		},
		{
			height:  bMaturity.Height,
			minConf: coinbaseMaturity + 2,
			bal:     0,
		},

//成熟后的下一块高度
		{
			height:  bMaturity.Height + 1,
			minConf: 0,
			bal:     35e8,
		},
		{
			height:  bMaturity.Height + 1,
			minConf: 2,
			bal:     35e8,
		},
		{
			height:  bMaturity.Height + 1,
			minConf: 3,
			bal:     30e8,
		},
		{
			height:  bMaturity.Height + 1,
			minConf: coinbaseMaturity + 2,
			bal:     30e8,
		},
		{
			height:  bMaturity.Height + 1,
			minConf: coinbaseMaturity + 3,
			bal:     0,
		},
	}
	for i, tst := range balTests {
		bal, err := s.Balance(ns, tst.minConf, tst.height)
		if err != nil {
			t.Fatalf("Balance test %d: Store.Balance failed: %v", i, err)
		}
		if bal != tst.bal {
			t.Errorf("Balance test %d: Got %v Expected %v", i, bal, tst.bal)
		}
	}
	if t.Failed() {
		t.Fatal("Failed balance checks mining coinbase spending transaction")
	}

//创建另一个支出交易，用于从
//第一个挥金如土的人。这将用于测试删除整个
//冲突链，当CoinBase稍后被重新安排。
//
//使用与Spender A相同的输出量，并将其标记为贷方。
//这意味着平衡测试应报告相同的结果。
	spenderBTime := time.Now()
	spenderB := spendOutput(&spenderARec.Hash, 0, 5e8)
	spenderBRec, err := NewTxRecordFromMsgTx(spenderB, spenderBTime)
	if err != nil {
		t.Fatal(err)
	}
	err = s.InsertTx(ns, spenderBRec, &bMaturity)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(ns, spenderBRec, &bMaturity, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	for i, tst := range balTests {
		bal, err := s.Balance(ns, tst.minConf, tst.height)
		if err != nil {
			t.Fatalf("Balance test %d: Store.Balance failed: %v", i, err)
		}
		if bal != tst.bal {
			t.Errorf("Balance test %d: Got %v Expected %v", i, bal, tst.bal)
		}
	}
	if t.Failed() {
		t.Fatal("Failed balance checks mining second spending transaction")
	}

//重新组合使coinbase到期的块并检查余额
//再一次。
	err = s.Rollback(ns, bMaturity.Height)
	if err != nil {
		t.Fatal(err)
	}
	balTests = balTestsBeforeMaturity
	for i, tst := range balTests {
		bal, err := s.Balance(ns, tst.minConf, tst.height)
		if err != nil {
			t.Fatalf("Balance test %d: Store.Balance failed: %v", i, err)
		}
		if bal != tst.bal {
			t.Errorf("Balance test %d: Got %v Expected %v", i, bal, tst.bal)
		}
	}
	if t.Failed() {
		t.Fatal("Failed balance checks after reorging maturity block")
	}

//重新排列出包含coinbase的块。应该没有
//存储中的更多事务（因为引用了以前的输出
//到支出Tx不再存在），余额将始终是
//零。
	err = s.Rollback(ns, b100.Height)
	if err != nil {
		t.Fatal(err)
	}
	balTests = []balTest{
//电流高度
		{
			height:  b100.Height - 1,
			minConf: 0,
			bal:     0,
		},
		{
			height:  b100.Height - 1,
			minConf: 1,
			bal:     0,
		},

//下一高度
		{
			height:  b100.Height,
			minConf: 0,
			bal:     0,
		},
		{
			height:  b100.Height,
			minConf: 1,
			bal:     0,
		},
	}
	for i, tst := range balTests {
		bal, err := s.Balance(ns, tst.minConf, tst.height)
		if err != nil {
			t.Fatalf("Balance test %d: Store.Balance failed: %v", i, err)
		}
		if bal != tst.bal {
			t.Errorf("Balance test %d: Got %v Expected %v", i, bal, tst.bal)
		}
	}
	if t.Failed() {
		t.Fatal("Failed balance checks after reorging coinbase block")
	}
	unminedTxs, err := s.UnminedTxs(ns)
	if err != nil {
		t.Fatal(err)
	}
	if len(unminedTxs) != 0 {
		t.Fatalf("Should have no unmined transactions after coinbase reorg, found %d", len(unminedTxs))
	}
}

//测试将多个事务从未链接的存储桶移动到同一个块。
func TestMoveMultipleToSameBlock(t *testing.T) {
	t.Parallel()

	s, db, teardown, err := testStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns := dbtx.ReadWriteBucket(namespaceKey)

	b100 := BlockMeta{
		Block: Block{Height: 100},
		Time:  time.Now(),
	}

	cb := newCoinBase(20e8, 30e8)
	cbRec, err := NewTxRecordFromMsgTx(cb, b100.Time)
	if err != nil {
		t.Fatal(err)
	}

//插入coinbase并将两个输出标记为信用。
	err = s.InsertTx(ns, cbRec, &b100)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(ns, cbRec, &b100, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(ns, cbRec, &b100, 1, false)
	if err != nil {
		t.Fatal(err)
	}

//创建并插入两个使用两个coinbase的未关联交易
//输出。
	spenderATime := time.Now()
	spenderA := spendOutput(&cbRec.Hash, 0, 1e8, 2e8, 18e8)
	spenderARec, err := NewTxRecordFromMsgTx(spenderA, spenderATime)
	if err != nil {
		t.Fatal(err)
	}
	err = s.InsertTx(ns, spenderARec, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(ns, spenderARec, nil, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(ns, spenderARec, nil, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	spenderBTime := time.Now()
	spenderB := spendOutput(&cbRec.Hash, 1, 4e8, 8e8, 18e8)
	spenderBRec, err := NewTxRecordFromMsgTx(spenderB, spenderBTime)
	if err != nil {
		t.Fatal(err)
	}
	err = s.InsertTx(ns, spenderBRec, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(ns, spenderBRec, nil, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AddCredit(ns, spenderBRec, nil, 1, false)
	if err != nil {
		t.Fatal(err)
	}

	coinbaseMaturity := int32(chaincfg.TestNet3Params.CoinbaseMaturity)

//在这个区块中挖掘两个交易，使coinbase成熟。
	bMaturity := BlockMeta{
		Block: Block{Height: b100.Height + coinbaseMaturity},
		Time:  time.Now(),
	}
	err = s.InsertTx(ns, spenderARec, &bMaturity)
	if err != nil {
		t.Fatal(err)
	}
	err = s.InsertTx(ns, spenderBRec, &bMaturity)
	if err != nil {
		t.Fatal(err)
	}

//检查是否可以在到期日块中查询这两个事务。
	detailsA, err := s.UniqueTxDetails(ns, &spenderARec.Hash, &bMaturity.Block)
	if err != nil {
		t.Fatal(err)
	}
	if detailsA == nil {
		t.Fatal("No details found for first spender")
	}
	detailsB, err := s.UniqueTxDetails(ns, &spenderBRec.Hash, &bMaturity.Block)
	if err != nil {
		t.Fatal(err)
	}
	if detailsB == nil {
		t.Fatal("No details found for second spender")
	}

//验证余额是否在块记录上正确更新
//追加，并且不保留未链接的事务。
	balTests := []struct {
		height  int32
		minConf int32
		bal     btcutil.Amount
	}{
//成熟高度
		{
			height:  bMaturity.Height,
			minConf: 0,
			bal:     15e8,
		},
		{
			height:  bMaturity.Height,
			minConf: 1,
			bal:     15e8,
		},
		{
			height:  bMaturity.Height,
			minConf: 2,
			bal:     0,
		},

//成熟后的下一块高度
		{
			height:  bMaturity.Height + 1,
			minConf: 0,
			bal:     15e8,
		},
		{
			height:  bMaturity.Height + 1,
			minConf: 2,
			bal:     15e8,
		},
		{
			height:  bMaturity.Height + 1,
			minConf: 3,
			bal:     0,
		},
	}
	for i, tst := range balTests {
		bal, err := s.Balance(ns, tst.minConf, tst.height)
		if err != nil {
			t.Fatalf("Balance test %d: Store.Balance failed: %v", i, err)
		}
		if bal != tst.bal {
			t.Errorf("Balance test %d: Got %v Expected %v", i, bal, tst.bal)
		}
	}
	if t.Failed() {
		t.Fatal("Failed balance checks after moving both coinbase spenders")
	}
	unminedTxs, err := s.UnminedTxs(ns)
	if err != nil {
		t.Fatal(err)
	}
	if len(unminedTxs) != 0 {
		t.Fatalf("Should have no unmined transactions mining both, found %d", len(unminedTxs))
	}
}

//测试txrecord中序列化事务的可选性。
//newtxrecord和newtxrecordfrommsgtx都保存序列化事务，因此
//手动将其剥离以测试此代码路径。
func TestInsertUnserializedTx(t *testing.T) {
	t.Parallel()

	s, db, teardown, err := testStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns := dbtx.ReadWriteBucket(namespaceKey)

	tx := newCoinBase(50e8)
	rec, err := NewTxRecordFromMsgTx(tx, timeNow())
	if err != nil {
		t.Fatal(err)
	}
	b100 := makeBlockMeta(100)
	err = s.InsertTx(ns, stripSerializedTx(rec), &b100)
	if err != nil {
		t.Fatalf("Insert for stripped TxRecord failed: %v", err)
	}

//确保检索成功。
	details, err := s.UniqueTxDetails(ns, &rec.Hash, &b100.Block)
	if err != nil {
		t.Fatal(err)
	}
	rec2, err := NewTxRecordFromMsgTx(&details.MsgTx, rec.Received)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rec.SerializedTx, rec2.SerializedTx) {
		t.Fatal("Serialized txs for coinbase do not match")
	}

//现在用一个未链接的事务测试该路径。
	tx = spendOutput(&rec.Hash, 0, 50e8)
	rec, err = NewTxRecordFromMsgTx(tx, timeNow())
	if err != nil {
		t.Fatal(err)
	}
	err = s.InsertTx(ns, rec, nil)
	if err != nil {
		t.Fatal(err)
	}
	details, err = s.UniqueTxDetails(ns, &rec.Hash, nil)
	if err != nil {
		t.Fatal(err)
	}
	rec2, err = NewTxRecordFromMsgTx(&details.MsgTx, rec.Received)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rec.SerializedTx, rec2.SerializedTx) {
		t.Fatal("Serialized txs for coinbase spender do not match")
	}
}

//testremoveunlinedtx测试如果我们添加一个umined事务，那么
//能够在以后删除该未链接的事务以及
//后裔。由于未关联交易而进行的任何余额修改应
//受到尊敬。
func TestRemoveUnminedTx(t *testing.T) {
	t.Parallel()

	store, db, teardown, err := testStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

//为了重现真实的场景，我们将使用一个新的数据库
//每次与钱包互动的交易。
//
//我们将从测试开始，在高度创建一个新的coinbase输出
//100并将其插入商店。
	b100 := &BlockMeta{
		Block: Block{Height: 100},
		Time:  time.Now(),
	}
	initialBalance := int64(1e8)
	cb := newCoinBase(initialBalance)
	cbRec, err := NewTxRecordFromMsgTx(cb, b100.Time)
	if err != nil {
		t.Fatal(err)
	}
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		if err := store.InsertTx(ns, cbRec, b100); err != nil {
			t.Fatal(err)
		}
		err := store.AddCredit(ns, cbRec, b100, 0, false)
		if err != nil {
			t.Fatal(err)
		}
	})

//确定创建的coinbase输出的成熟度高度。
	coinbaseMaturity := int32(chaincfg.TestNet3Params.CoinbaseMaturity)
	maturityHeight := b100.Block.Height + coinbaseMaturity

//checkBalance is a helper function that compares the balance of the
//以预期值存储。includeUnconfirmed布尔值可以是
//用于将未确认余额作为总额的一部分
//平衡。
	checkBalance := func(expectedBalance btcutil.Amount,
		includeUnconfirmed bool) {

		t.Helper()

		minConfs := int32(1)
		if includeUnconfirmed {
			minConfs = 0
		}

		commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
			t.Helper()

			b, err := store.Balance(ns, minConfs, maturityHeight)
			if err != nil {
				t.Fatalf("unable to retrieve balance: %v", err)
			}
			if b != expectedBalance {
				t.Fatalf("expected balance of %d, got %d",
					expectedBalance, b)
			}
		})
	}

//因为我们店内没有未确认的交易，
//反映已确认和未确认产出的总余额应
//匹配初始余额。
	checkBalance(btcutil.Amount(initialBalance), false)
	checkBalance(btcutil.Amount(initialBalance), true)

//然后，我们将为coinbase输出创建一个未确认的支出，并
//把它插入商店。
	b101 := &BlockMeta{
		Block: Block{Height: 201},
		Time:  time.Now(),
	}
	changeAmount := int64(4e7)
	spendTx := spendOutput(&cbRec.Hash, 0, 5e7, changeAmount)
	spendTxRec, err := NewTxRecordFromMsgTx(spendTx, b101.Time)
	if err != nil {
		t.Fatal(err)
	}
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		if err := store.InsertTx(ns, spendTxRec, nil); err != nil {
			t.Fatal(err)
		}
		err := store.AddCredit(ns, spendTxRec, nil, 1, true)
		if err != nil {
			t.Fatal(err)
		}
	})

//将未确认的支出插入存储后，我们将查询它
//以确保正确添加。
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		unminedTxs, err := store.UnminedTxs(ns)
		if err != nil {
			t.Fatalf("unable to query for unmined txs: %v", err)
		}
		if len(unminedTxs) != 1 {
			t.Fatalf("expected 1 mined tx, instead got %v",
				len(unminedTxs))
		}
		unminedTxHash := unminedTxs[0].TxHash()
		spendTxHash := spendTx.TxHash()
		if !unminedTxHash.IsEqual(&spendTxHash) {
			t.Fatalf("mismatch tx hashes: expected %v, got %v",
				spendTxHash, unminedTxHash)
		}
	})

//既然存在未确认的支出，就不应该再存在
//确认余额。总余额现在应全部未确认。
//它应该与未确认支出的变动金额相匹配。
//转运。
	checkBalance(0, false)
	checkBalance(btcutil.Amount(changeAmount), true)

//现在，我们将从商店中删除未确认的支出事务。
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		if err := store.RemoveUnminedTx(ns, spendTxRec); err != nil {
			t.Fatal(err)
		}
	})

//我们将最后一次查询该商店的未确认交易
//以确保上述未确认支出被适当移除。
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		unminedTxs, err := store.UnminedTxs(ns)
		if err != nil {
			t.Fatalf("unable to query for unmined txs: %v", err)
		}
		if len(unminedTxs) != 0 {
			t.Fatalf("expected 0 mined txs, instead got %v",
				len(unminedTxs))
		}
	})

//最后，总余额（包括已确认和未确认）
//应再次与初始余额相匹配，作为未确定的支出
//已被删除。
	checkBalance(btcutil.Amount(initialBalance), false)
	checkBalance(btcutil.Amount(initialBalance), true)
}

//commitdbtx是一个助手函数，允许我们执行多个操作
//在特定数据库的bucket上执行单个原子操作。
func commitDBTx(t *testing.T, store *Store, db walletdb.DB,
	f func(walletdb.ReadWriteBucket)) {

	t.Helper()

	dbTx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbTx.Commit()

	ns := dbTx.ReadWriteBucket(namespaceKey)

	f(ns)
}

//TestInsertDoublesPendtx是一个辅助测试，它将输出开销加倍。这个
//布尔参数指示第一个支出事务是否应
//那个证实了。此测试可确保在发生双重花费时
//支出交易存在于mempool中，如果其中一个确认，
//那么mempool中剩余的冲突事务应该是
//从钱包商店里拿出来的。
func testInsertMempoolDoubleSpendTx(t *testing.T, first bool) {
	store, db, teardown, err := testStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

//为了重现真实的场景，我们将使用一个新的数据库
//每次与钱包互动的交易。
//
//我们将从测试开始，在高度创建一个新的coinbase输出
//100并将其插入商店。
	b100 := BlockMeta{
		Block: Block{Height: 100},
		Time:  time.Now(),
	}
	cb := newCoinBase(1e8)
	cbRec, err := NewTxRecordFromMsgTx(cb, b100.Time)
	if err != nil {
		t.Fatal(err)
	}
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		if err := store.InsertTx(ns, cbRec, &b100); err != nil {
			t.Fatal(err)
		}
		err := store.AddCredit(ns, cbRec, &b100, 0, false)
		if err != nil {
			t.Fatal(err)
		}
	})

//然后，我们将根据相同的coinbase输出创建两个支出，依次
//复制双重消费场景。
	firstSpend := spendOutput(&cbRec.Hash, 0, 5e7, 5e7)
	firstSpendRec, err := NewTxRecordFromMsgTx(firstSpend, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	secondSpend := spendOutput(&cbRec.Hash, 0, 4e7, 6e7)
	secondSpendRec, err := NewTxRecordFromMsgTx(secondSpend, time.Now())
	if err != nil {
		t.Fatal(err)
	}

//我们会在不确认的情况下把它们都放进商店。
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		if err := store.InsertTx(ns, firstSpendRec, nil); err != nil {
			t.Fatal(err)
		}
		err := store.AddCredit(ns, firstSpendRec, nil, 0, false)
		if err != nil {
			t.Fatal(err)
		}
	})
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		if err := store.InsertTx(ns, secondSpendRec, nil); err != nil {
			t.Fatal(err)
		}
		err := store.AddCredit(ns, secondSpendRec, nil, 0, false)
		if err != nil {
			t.Fatal(err)
		}
	})

//确保在未确认的交易中发现这两项支出
//在钱包的商店里。
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		unminedTxs, err := store.UnminedTxs(ns)
		if err != nil {
			t.Fatal(err)
		}
		if len(unminedTxs) != 2 {
			t.Fatalf("expected 2 unmined txs, got %v",
				len(unminedTxs))
		}
	})

//然后，我们会确认第一次或第二次消费，具体取决于
//布尔值通过了，其高度足够深，允许我们
//成功地使用coinbase输出。
	coinbaseMaturity := int32(chaincfg.TestNet3Params.CoinbaseMaturity)
	bMaturity := BlockMeta{
		Block: Block{Height: b100.Height + coinbaseMaturity},
		Time:  time.Now(),
	}

	var confirmedSpendRec *TxRecord
	if first {
		confirmedSpendRec = firstSpendRec
	} else {
		confirmedSpendRec = secondSpendRec
	}
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		err := store.InsertTx(ns, confirmedSpendRec, &bMaturity)
		if err != nil {
			t.Fatal(err)
		}
		err = store.AddCredit(
			ns, confirmedSpendRec, &bMaturity, 0, false,
		)
		if err != nil {
			t.Fatal(err)
		}
	})

//现在应该触发存储以删除任何其他挂起的double
//为这个coinbase输出花费，正如我们已经看到的
//一个。因此，我们不应该看到任何其他未确认的交易
//在它里面。我们还确保确认的交易
//现在钱包里的utxo是正确的。
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		unminedTxs, err := store.UnminedTxs(ns)
		if err != nil {
			t.Fatal(err)
		}
		if len(unminedTxs) != 0 {
			t.Fatalf("expected 0 unmined txs, got %v",
				len(unminedTxs))
		}

		minedTxs, err := store.UnspentOutputs(ns)
		if err != nil {
			t.Fatal(err)
		}
		if len(minedTxs) != 1 {
			t.Fatalf("expected 1 mined tx, got %v", len(minedTxs))
		}
		if !minedTxs[0].Hash.IsEqual(&confirmedSpendRec.Hash) {
			t.Fatalf("expected confirmed tx hash %v, got %v",
				confirmedSpendRec.Hash, minedTxs[0].Hash)
		}
	})
}

//testinsertmempooldoublespendedconfirmedfirsttx确保当一个双消费
//如果
//首先确认看到的支出，然后确认
//应将Mempool从钱包商店中取出。
func TestInsertMempoolDoubleSpendConfirmedFirstTx(t *testing.T) {
	t.Parallel()
	testInsertMempoolDoubleSpendTx(t, true)
}

//testinsertmempooldoublespendedconfirmedfirsttx确保当一个双消费
//如果
//确认看到的第二个支出，然后在
//应将Mempool从钱包商店中取出。
func TestInsertMempoolDoubleSpendConfirmSecondTx(t *testing.T) {
	t.Parallel()
	testInsertMempoolDoubleSpendTx(t, false)
}

//Testinsertconfirmeddoublependtx测试当一个或多个double花费
//发生并且消费交易确认钱包不知道，
//then the unconfirmed double spends within the mempool should be removed from
//钱包的商店。
func TestInsertConfirmedDoubleSpendTx(t *testing.T) {
	t.Parallel()

	store, db, teardown, err := testStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

//为了重现真实的场景，我们将使用一个新的数据库
//每次与钱包互动的交易。
//
//我们将从测试开始，在高度创建一个新的coinbase输出
//100并将其插入商店。
	b100 := BlockMeta{
		Block: Block{Height: 100},
		Time:  time.Now(),
	}
	cb1 := newCoinBase(1e8)
	cbRec1, err := NewTxRecordFromMsgTx(cb1, b100.Time)
	if err != nil {
		t.Fatal(err)
	}
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		if err := store.InsertTx(ns, cbRec1, &b100); err != nil {
			t.Fatal(err)
		}
		err := store.AddCredit(ns, cbRec1, &b100, 0, false)
		if err != nil {
			t.Fatal(err)
		}
	})

//然后，我们将从相同的coinbase输出中创建三个支出。这个
//前两个将保持未确认，而最后一个应该确认和
//从钱包商店中取出剩余的未确认信息。
	firstSpend1 := spendOutput(&cbRec1.Hash, 0, 5e7)
	firstSpendRec1, err := NewTxRecordFromMsgTx(firstSpend1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		if err := store.InsertTx(ns, firstSpendRec1, nil); err != nil {
			t.Fatal(err)
		}
		err := store.AddCredit(ns, firstSpendRec1, nil, 0, false)
		if err != nil {
			t.Fatal(err)
		}
	})

	secondSpend1 := spendOutput(&cbRec1.Hash, 0, 4e7)
	secondSpendRec1, err := NewTxRecordFromMsgTx(secondSpend1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		if err := store.InsertTx(ns, secondSpendRec1, nil); err != nil {
			t.Fatal(err)
		}
		err := store.AddCredit(ns, secondSpendRec1, nil, 0, false)
		if err != nil {
			t.Fatal(err)
		}
	})

//我们还将创建另一个输出，其中一个未确认，另一个
//已确认的支出交易也会进行支出。
	cb2 := newCoinBase(2e8)
	cbRec2, err := NewTxRecordFromMsgTx(cb2, b100.Time)
	if err != nil {
		t.Fatal(err)
	}
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		if err := store.InsertTx(ns, cbRec2, &b100); err != nil {
			t.Fatal(err)
		}
		err := store.AddCredit(ns, cbRec2, &b100, 0, false)
		if err != nil {
			t.Fatal(err)
		}
	})

	firstSpend2 := spendOutput(&cbRec2.Hash, 0, 5e7)
	firstSpendRec2, err := NewTxRecordFromMsgTx(firstSpend2, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		if err := store.InsertTx(ns, firstSpendRec2, nil); err != nil {
			t.Fatal(err)
		}
		err := store.AddCredit(ns, firstSpendRec2, nil, 0, false)
		if err != nil {
			t.Fatal(err)
		}
	})

//此时，我们将看到
//商店。
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		unminedTxs, err := store.UnminedTxs(ns)
		if err != nil {
			t.Fatal(err)
		}
		if len(unminedTxs) != 3 {
			t.Fatalf("expected 3 unmined txs, got %d",
				len(unminedTxs))
		}
	})

//然后，我们将在足够深的高度插入已确认的支出
//允许我们成功地花费coinbase输出。
	coinbaseMaturity := int32(chaincfg.TestNet3Params.CoinbaseMaturity)
	bMaturity := BlockMeta{
		Block: Block{Height: b100.Height + coinbaseMaturity},
		Time:  time.Now(),
	}
	outputsToSpend := []wire.OutPoint{
		{Hash: cbRec1.Hash, Index: 0},
		{Hash: cbRec2.Hash, Index: 0},
	}
	confirmedSpend := spendOutputs(outputsToSpend, 3e7)
	confirmedSpendRec, err := NewTxRecordFromMsgTx(
		confirmedSpend, bMaturity.Time,
	)
	if err != nil {
		t.Fatal(err)
	}
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		err := store.InsertTx(ns, confirmedSpendRec, &bMaturity)
		if err != nil {
			t.Fatal(err)
		}
		err = store.AddCredit(
			ns, confirmedSpendRec, &bMaturity, 0, false,
		)
		if err != nil {
			t.Fatal(err)
		}
	})

//Now that the confirmed spend exists within the store, we should no
//再看一看里面未经证实的花销。我们还确保
//确认的交易，现在在
//钱包是正确的。
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		unminedTxs, err := store.UnminedTxs(ns)
		if err != nil {
			t.Fatal(err)
		}
		if len(unminedTxs) != 0 {
			t.Fatalf("expected 0 unmined txs, got %v",
				len(unminedTxs))
		}

		minedTxs, err := store.UnspentOutputs(ns)
		if err != nil {
			t.Fatal(err)
		}
		if len(minedTxs) != 1 {
			t.Fatalf("expected 1 mined tx, got %v", len(minedTxs))
		}
		if !minedTxs[0].Hash.IsEqual(&confirmedSpendRec.Hash) {
			t.Fatalf("expected confirmed tx hash %v, got %v",
				confirmedSpend, minedTxs[0].Hash)
		}
	})
}

//TestAddDuplicateCreditAfterConfirm旨在测试重复的情况
//未确认的信用证在初始信用证已经存在后添加到商店
//证实。这会导致输出在存储区中被复制，从而
//导致在查询钱包的utxo集时产生双倍开销。
func TestAddDuplicateCreditAfterConfirm(t *testing.T) {
	t.Parallel()

	store, db, teardown, err := testStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

//为了重现真实的场景，我们将使用一个新的数据库
//每次与钱包互动的交易。
//
//我们将从测试开始，在高度创建一个新的coinbase输出
//100并将其插入商店。
	b100 := &BlockMeta{
		Block: Block{Height: 100},
		Time:  time.Now(),
	}
	cb := newCoinBase(1e8)
	cbRec, err := NewTxRecordFromMsgTx(cb, b100.Time)
	if err != nil {
		t.Fatal(err)
	}
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		if err := store.InsertTx(ns, cbRec, b100); err != nil {
			t.Fatal(err)
		}
		err := store.AddCredit(ns, cbRec, b100, 0, false)
		if err != nil {
			t.Fatal(err)
		}
	})

//我们将确认商店中有一个未使用的输出。
//应该是上面创建的coinbase输出。
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		minedTxs, err := store.UnspentOutputs(ns)
		if err != nil {
			t.Fatal(err)
		}
		if len(minedTxs) != 1 {
			t.Fatalf("expected 1 mined tx, got %v", len(minedTxs))
		}
		if !minedTxs[0].Hash.IsEqual(&cbRec.Hash) {
			t.Fatalf("expected tx hash %v, got %v", cbRec.Hash,
				minedTxs[0].Hash)
		}
	})

//然后，我们将为coinbase输出创建一个未确认的支出。
	b101 := &BlockMeta{
		Block: Block{Height: 101},
		Time:  time.Now(),
	}
	spendTx := spendOutput(&cbRec.Hash, 0, 5e7, 4e7)
	spendTxRec, err := NewTxRecordFromMsgTx(spendTx, b101.Time)
	if err != nil {
		t.Fatal(err)
	}
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		if err := store.InsertTx(ns, spendTxRec, nil); err != nil {
			t.Fatal(err)
		}
		err := store.AddCredit(ns, spendTxRec, nil, 1, true)
		if err != nil {
			t.Fatal(err)
		}
	})

//在下一个高度确认支出交易。
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		if err := store.InsertTx(ns, spendTxRec, b101); err != nil {
			t.Fatal(err)
		}
		err := store.AddCredit(ns, spendTxRec, b101, 1, true)
		if err != nil {
			t.Fatal(err)
		}
	})

//我们应该再次看到一个未消耗的输出，这个
//时间是支出事务的更改输出。
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		minedTxs, err := store.UnspentOutputs(ns)
		if err != nil {
			t.Fatal(err)
		}
		if len(minedTxs) != 1 {
			t.Fatalf("expected 1 mined txs, got %v", len(minedTxs))
		}
		if !minedTxs[0].Hash.IsEqual(&spendTxRec.Hash) {
			t.Fatalf("expected tx hash %v, got %v", spendTxRec.Hash,
				minedTxs[0].Hash)
		}
	})

//现在，我们将再次插入支出交易，这次是
//未经证实的如果后端恰好转发
//unconfirmed chain.RelevantTx notification to the client even after it
//已经确认，这导致我们再次将其添加到商店。
//
//托多（威尔默）：理想情况下，这不应该发生，所以我们应该确定
//这是真正的原因。
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		if err := store.InsertTx(ns, spendTxRec, nil); err != nil {
			t.Fatal(err)
		}
		err := store.AddCredit(ns, spendTxRec, nil, 1, true)
		if err != nil {
			t.Fatal(err)
		}
	})

//最后，我们将确保更改输出仍然是唯一未使用的
//存储中的输出。
	commitDBTx(t, store, db, func(ns walletdb.ReadWriteBucket) {
		minedTxs, err := store.UnspentOutputs(ns)
		if err != nil {
			t.Fatal(err)
		}
		if len(minedTxs) != 1 {
			t.Fatalf("expected 1 mined txs, got %v", len(minedTxs))
		}
		if !minedTxs[0].Hash.IsEqual(&spendTxRec.Hash) {
			t.Fatalf("expected tx hash %v, got %v", spendTxRec.Hash,
				minedTxs[0].Hash)
		}
	})
}
