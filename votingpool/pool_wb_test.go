
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

package votingpool

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/btcsuite/btcwallet/waddrmgr"
)

func TestPoolEnsureUsedAddr(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, addrmgrNs := TstRWNamespaces(dbtx)

	var script []byte
	var addr waddrmgr.ManagedScriptAddress
	TstCreateSeries(t, dbtx, pool, []TstSeriesDef{{ReqSigs: 2, PubKeys: TstPubKeys[0:3], SeriesID: 1}})

	idx := Index(0)
	TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		err = pool.EnsureUsedAddr(ns, addrmgrNs, 1, 0, idx)
	})
	if err != nil {
		t.Fatalf("Failed to ensure used addresses: %v", err)
	}
	addr, err = pool.getUsedAddr(ns, addrmgrNs, 1, 0, 0)
	if err != nil {
		t.Fatalf("Failed to get addr from used addresses set: %v", err)
	}
	TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		script, err = addr.Script()
	})
	if err != nil {
		t.Fatalf("Failed to get script: %v", err)
	}
	wantScript, _ := pool.DepositScript(1, 0, 0)
	if !bytes.Equal(script, wantScript) {
		t.Fatalf("Script from looked up addr is not what we expect")
	}

	idx = Index(3)
	TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		err = pool.EnsureUsedAddr(ns, addrmgrNs, 1, 0, idx)
	})
	if err != nil {
		t.Fatalf("Failed to ensure used addresses: %v", err)
	}
	for _, i := range []int{0, 1, 2, 3} {
		addr, err = pool.getUsedAddr(ns, addrmgrNs, 1, 0, Index(i))
		if err != nil {
			t.Fatalf("Failed to get addr from used addresses set: %v", err)
		}
		TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
			script, err = addr.Script()
		})
		if err != nil {
			t.Fatalf("Failed to get script: %v", err)
		}
		wantScript, _ := pool.DepositScript(1, 0, Index(i))
		if !bytes.Equal(script, wantScript) {
			t.Fatalf("Script from looked up addr is not what we expect")
		}
	}
}

func TestPoolGetUsedAddr(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, addrmgrNs := TstRWNamespaces(dbtx)

	TstCreateSeries(t, dbtx, pool, []TstSeriesDef{{ReqSigs: 2, PubKeys: TstPubKeys[0:3], SeriesID: 1}})

//序列号为1、分支号为0、索引号为10的addr从未使用过，因此它应该
//返回零。
	addr, err := pool.getUsedAddr(ns, addrmgrNs, 1, 0, 10)
	if err != nil {
		t.Fatalf("Error when looking up used addr: %v", err)
	}
	if addr != nil {
		t.Fatalf("Unused address found in used addresses DB: %v", addr)
	}

//现在，我们将该地址添加到已用地址数据库，并检查该值
//getusedaddr（）返回的是我们所期望的。
	TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		err = pool.addUsedAddr(ns, addrmgrNs, 1, 0, 10)
	})
	if err != nil {
		t.Fatalf("Error when storing addr in used addresses DB: %v", err)
	}
	var script []byte
	addr, err = pool.getUsedAddr(ns, addrmgrNs, 1, 0, 10)
	if err != nil {
		t.Fatalf("Error when looking up used addr: %v", err)
	}
	TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		script, err = addr.Script()
	})
	if err != nil {
		t.Fatalf("Failed to get script: %v", err)
	}
	wantScript, _ := pool.DepositScript(1, 0, 10)
	if !bytes.Equal(script, wantScript) {
		t.Fatalf("Script from looked up addr is not what we expect")
	}
}

func TestSerializationErrors(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	_, addrmgrNs := TstRWNamespaces(dbtx)

	tests := []struct {
		version  uint32
		pubKeys  []string
		privKeys []string
		reqSigs  uint32
		err      ErrorCode
	}{
		{
			version: 2,
			pubKeys: TstPubKeys[0:3],
			err:     ErrSeriesVersion,
		},
		{
			pubKeys: []string{"NONSENSE"},
//不是有效长度的公钥。
			err: ErrSeriesSerialization,
		},
		{
			pubKeys:  TstPubKeys[0:3],
			privKeys: TstPrivKeys[0:1],
//公钥和私钥的数目应该相同。
			err: ErrSeriesSerialization,
		},
		{
			pubKeys:  TstPubKeys[0:1],
			privKeys: []string{"NONSENSE"},
//不是有效长度的私钥。
			err: ErrSeriesSerialization,
		},
	}

	active := true
	for testNum, test := range tests {
		encryptedPubs, err := encryptKeys(test.pubKeys, pool.Manager(), waddrmgr.CKTPublic)
		if err != nil {
			t.Fatalf("Test #%d - Error encrypting pubkeys: %v", testNum, err)
		}
		var encryptedPrivs [][]byte
		TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
			encryptedPrivs, err = encryptKeys(test.privKeys, pool.Manager(), waddrmgr.CKTPrivate)
		})
		if err != nil {
			t.Fatalf("Test #%d - Error encrypting privkeys: %v", testNum, err)
		}

		row := &dbSeriesRow{
			version:           test.version,
			active:            active,
			reqSigs:           test.reqSigs,
			pubKeysEncrypted:  encryptedPubs,
			privKeysEncrypted: encryptedPrivs}
		_, err = serializeSeriesRow(row)

		TstCheckError(t, fmt.Sprintf("Test #%d", testNum), err, test.err)
	}
}

func TestSerialization(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	_, addrmgrNs := TstRWNamespaces(dbtx)

	tests := []struct {
		version  uint32
		active   bool
		pubKeys  []string
		privKeys []string
		reqSigs  uint32
	}{
		{
			version: 1,
			active:  true,
			pubKeys: TstPubKeys[0:1],
			reqSigs: 1,
		},
		{
			version:  0,
			active:   false,
			pubKeys:  TstPubKeys[0:1],
			privKeys: TstPrivKeys[0:1],
			reqSigs:  1,
		},
		{
			pubKeys:  TstPubKeys[0:3],
			privKeys: []string{TstPrivKeys[0], "", ""},
			reqSigs:  2,
		},
		{
			pubKeys: TstPubKeys[0:5],
			reqSigs: 3,
		},
		{
			pubKeys:  TstPubKeys[0:7],
			privKeys: []string{"", TstPrivKeys[1], "", TstPrivKeys[3], "", "", ""},
			reqSigs:  4,
		},
	}

	var encryptedPrivs [][]byte
	for testNum, test := range tests {
		encryptedPubs, err := encryptKeys(test.pubKeys, pool.Manager(), waddrmgr.CKTPublic)
		if err != nil {
			t.Fatalf("Test #%d - Error encrypting pubkeys: %v", testNum, err)
		}
		TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
			encryptedPrivs, err = encryptKeys(test.privKeys, pool.Manager(), waddrmgr.CKTPrivate)
		})
		if err != nil {
			t.Fatalf("Test #%d - Error encrypting privkeys: %v", testNum, err)
		}

		row := &dbSeriesRow{
			version:           test.version,
			active:            test.active,
			reqSigs:           test.reqSigs,
			pubKeysEncrypted:  encryptedPubs,
			privKeysEncrypted: encryptedPrivs,
		}
		serialized, err := serializeSeriesRow(row)
		if err != nil {
			t.Fatalf("Test #%d - Error in serialization %v", testNum, err)
		}

		row, err = deserializeSeriesRow(serialized)
		if err != nil {
			t.Fatalf("Test #%d - Failed to deserialize %v %v", testNum, serialized, err)
		}

		if row.version != test.version {
			t.Errorf("Serialization #%d - version mismatch: got %d want %d",
				testNum, row.version, test.version)
		}

		if row.active != test.active {
			t.Errorf("Serialization #%d - active mismatch: got %v want %v",
				testNum, row.active, test.active)
		}

		if row.reqSigs != test.reqSigs {
			t.Errorf("Serialization #%d - row reqSigs off. Got %d, want %d",
				testNum, row.reqSigs, test.reqSigs)
		}

		if len(row.pubKeysEncrypted) != len(test.pubKeys) {
			t.Errorf("Serialization #%d - Wrong no. of pubkeys. Got %d, want %d",
				testNum, len(row.pubKeysEncrypted), len(test.pubKeys))
		}

		for i, encryptedPub := range encryptedPubs {
			got := string(row.pubKeysEncrypted[i])

			if got != string(encryptedPub) {
				t.Errorf("Serialization #%d - Pubkey deserialization. Got %v, want %v",
					testNum, got, string(encryptedPub))
			}
		}

		if len(row.privKeysEncrypted) != len(row.pubKeysEncrypted) {
			t.Errorf("Serialization #%d - no. privkeys (%d) != no. pubkeys (%d)",
				testNum, len(row.privKeysEncrypted), len(row.pubKeysEncrypted))
		}

		for i, encryptedPriv := range encryptedPrivs {
			got := string(row.privKeysEncrypted[i])

			if got != string(encryptedPriv) {
				t.Errorf("Serialization #%d - Privkey deserialization. Got %v, want %v",
					testNum, got, string(encryptedPriv))
			}
		}
	}
}

func TestDeserializationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		serialized []byte
		err        ErrorCode
	}{
		{
			serialized: make([]byte, seriesMaxSerial+1),
//字节太多（在seriesMaxSerial上）。
			err: ErrSeriesSerialization,
		},
		{
			serialized: make([]byte, seriesMinSerial-1),
//字节不足（在seriemsinserial下）。
			err: ErrSeriesSerialization,
		},
		{
			serialized: []byte{
1, 0, 0, 0, //4字节（版本）
0,          //1字节（活动）
2, 0, 0, 0, //4字节（reqsigs）
3, 0, 0, 0, //4字节（NKEY）
			},
//这里我们有不变的数据，但是缺少任何公钥/私钥。
			err: ErrSeriesSerialization,
		},
		{
			serialized: []byte{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
//不支持的版本。
			err: ErrSeriesVersion,
		},
	}

	for testNum, test := range tests {
		_, err := deserializeSeriesRow(test.serialized)

		TstCheckError(t, fmt.Sprintf("Test #%d", testNum), err, test.err)
	}
}

func TestValidateAndDecryptKeys(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	_, addrmgrNs := TstRWNamespaces(dbtx)

	rawPubKeys, err := encryptKeys(TstPubKeys[0:2], pool.Manager(), waddrmgr.CKTPublic)
	if err != nil {
		t.Fatalf("Failed to encrypt public keys: %v", err)
	}

	var rawPrivKeys [][]byte
	TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		rawPrivKeys, err = encryptKeys([]string{TstPrivKeys[0], ""}, pool.Manager(), waddrmgr.CKTPrivate)
	})
	if err != nil {
		t.Fatalf("Failed to encrypt private keys: %v", err)
	}

	var pubKeys, privKeys []*hdkeychain.ExtendedKey
	TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		pubKeys, privKeys, err = validateAndDecryptKeys(rawPubKeys, rawPrivKeys, pool)
	})
	if err != nil {
		t.Fatalf("Error when validating/decrypting keys: %v", err)
	}

	if len(pubKeys) != 2 {
		t.Fatalf("Unexpected number of decrypted public keys: got %d, want 2", len(pubKeys))
	}
	if len(privKeys) != 2 {
		t.Fatalf("Unexpected number of decrypted private keys: got %d, want 2", len(privKeys))
	}

	if pubKeys[0].String() != TstPubKeys[0] || pubKeys[1].String() != TstPubKeys[1] {
		t.Fatalf("Public keys don't match: %v!=%v ", TstPubKeys[0:2], pubKeys)
	}

	if privKeys[0].String() != TstPrivKeys[0] || privKeys[1] != nil {
		t.Fatalf("Private keys don't match: %v, %v", []string{TstPrivKeys[0], ""}, privKeys)
	}

	neuteredKey, err := privKeys[0].Neuter()
	if err != nil {
		t.Fatalf("Unable to neuter private key: %v", err)
	}
	if pubKeys[0].String() != neuteredKey.String() {
		t.Errorf("Public key (%v) does not match neutered private key (%v)",
			pubKeys[0].String(), neuteredKey.String())
	}
}

func TestValidateAndDecryptKeysErrors(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	_, addrmgrNs := TstRWNamespaces(dbtx)

	encryptedPubKeys, err := encryptKeys(TstPubKeys[0:1], pool.Manager(), waddrmgr.CKTPublic)
	if err != nil {
		t.Fatalf("Failed to encrypt public key: %v", err)
	}

	var encryptedPrivKeys [][]byte
	TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		encryptedPrivKeys, err = encryptKeys(TstPrivKeys[1:2], pool.Manager(), waddrmgr.CKTPrivate)
	})
	if err != nil {
		t.Fatalf("Failed to encrypt private key: %v", err)
	}

	tests := []struct {
		rawPubKeys  [][]byte
		rawPrivKeys [][]byte
		err         ErrorCode
	}{
		{
//公钥的数目与私钥的数目不匹配。
			rawPubKeys:  [][]byte{[]byte(TstPubKeys[0])},
			rawPrivKeys: [][]byte{},
			err:         ErrKeysPrivatePublicMismatch,
		},
		{
//解密公钥失败。
			rawPubKeys:  [][]byte{[]byte(TstPubKeys[0])},
			rawPrivKeys: [][]byte{[]byte(TstPrivKeys[0])},
			err:         ErrCrypto,
		},
		{
//解密私钥失败。
			rawPubKeys:  encryptedPubKeys,
			rawPrivKeys: [][]byte{[]byte(TstPrivKeys[0])},
			err:         ErrCrypto,
		},
		{
//一个公钥和一个私钥，但它们不匹配。
			rawPubKeys:  encryptedPubKeys,
			rawPrivKeys: encryptedPrivKeys,
			err:         ErrKeyMismatch,
		},
	}

	for i, test := range tests {
		TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
			_, _, err = validateAndDecryptKeys(test.rawPubKeys, test.rawPrivKeys, pool)
		})
		TstCheckError(t, fmt.Sprintf("Test #%d", i), err, test.err)
	}
}

func encryptKeys(keys []string, mgr *waddrmgr.Manager, keyType waddrmgr.CryptoKeyType) ([][]byte, error) {
	encryptedKeys := make([][]byte, len(keys))
	var err error
	for i, key := range keys {
		if key == "" {
			encryptedKeys[i] = nil
		} else {
			encryptedKeys[i], err = mgr.Encrypt(keyType, []byte(key))
		}
		if err != nil {
			return nil, err
		}
	}
	return encryptedKeys, nil
}
