
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package waddrmgr

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/snacl"
	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/davecgh/go-spew/spew"
)

//failingcryptokey是encryptordecryptor接口的一种实现
//当试图用它加密或解密时故意失败。
type failingCryptoKey struct {
	cryptoKey
}

//ENCRYPT故意在调用以测试错误路径时返回失败。
//
//这是EncryptorDecryptor接口实现的一部分。
func (c *failingCryptoKey) Encrypt(in []byte) ([]byte, error) {
	return nil, errors.New("failed to encrypt")
}

//decrypt故意在调用以测试错误路径时返回失败。
//
//这是EncryptorDecryptor接口实现的一部分。
func (c *failingCryptoKey) Decrypt(in []byte) ([]byte, error) {
	return nil, errors.New("failed to decrypt")
}

//newhash将传递的big endian十六进制字符串转换为chainhash.hash。
//
//错误，因为它将仅（且必须仅）使用硬编码调用，并且
//因此已知良好，哈希。
func newHash(hexStr string) *chainhash.Hash {
	hash, err := chainhash.NewHashFromStr(hexStr)
	if err != nil {
		panic(err)
	}
	return hash
}

//failingSecretkeyGen是一个始终返回的SecretkeyGenerator
//snacl.errdecrypter失败。
func failingSecretKeyGen(passphrase *[]byte,
	config *ScryptOptions) (*snacl.SecretKey, error) {
	return nil, snacl.ErrDecryptFailed
}

//test context用于存储有关正在运行的测试的上下文信息，该测试
//
//
//提供试块。这是必需的，因为第一个循环
//插入时，测试将针对最新的块运行，因此
//所有的输出都还不能用掉。但是，在随后的运行中，所有
//
//花了。
type testContext struct {
	t            *testing.T
	db           walletdb.DB
	rootManager  *Manager
	manager      *ScopedKeyManager
	account      uint32
	create       bool
	unlocked     bool
	watchingOnly bool
}

//addrType是正在测试的地址类型
type addrType byte

const (
	addrPubKeyHash addrType = iota
	addrScriptHash
)

//ExpectedAddr用于存储托管
//地址。不是所有字段都用于所有托管地址类型。
type expectedAddr struct {
	address        string
	addressHash    []byte
	internal       bool
	compressed     bool
	used           bool
	imported       bool
	pubKey         []byte
	privKey        []byte
	privKeyWIF     string
	script         []byte
	derivationInfo DerivationPath
}

//testnameprefix是返回前缀以显示基于测试错误的帮助程序
//测试上下文的状态。
func testNamePrefix(tc *testContext) string {
	prefix := "Open "
	if tc.create {
		prefix = "Create "
	}

	return prefix + fmt.Sprintf("account #%d", tc.account)
}

//testmanagedpubkeyaddress确保所有导出函数返回的数据
//由传递的托管公钥地址提供，与相应的
//提供的预期地址中的字段。
//
//当测试上下文指示管理器已解锁时，私有数据
//还将测试处理私有数据的功能，否则
//检查以确保返回正确的错误。
func testManagedPubKeyAddress(tc *testContext, prefix string,
	gotAddr ManagedPubKeyAddress, wantAddr *expectedAddr) bool {

//确保pubkey是托管地址的预期值。
	var gpubBytes []byte
	if gotAddr.Compressed() {
		gpubBytes = gotAddr.PubKey().SerializeCompressed()
	} else {
		gpubBytes = gotAddr.PubKey().SerializeUncompressed()
	}
	if !reflect.DeepEqual(gpubBytes, wantAddr.pubKey) {
		tc.t.Errorf("%s PubKey: unexpected public key - got %x, want "+
			"%x", prefix, gpubBytes, wantAddr.pubKey)
		return false
	}

//确保导出的pubkey字符串是托管的预期值
//地址。
	gpubHex := gotAddr.ExportPubKey()
	wantPubHex := hex.EncodeToString(wantAddr.pubKey)
	if gpubHex != wantPubHex {
		tc.t.Errorf("%s ExportPubKey: unexpected public key - got %s, "+
			"want %s", prefix, gpubHex, wantPubHex)
		return false
	}

//确保派生路径在
//地址是从磁盘读取的。
	_, gotAddrPath, ok := gotAddr.DerivationInfo()
	if !ok && !gotAddr.Imported() {
		tc.t.Errorf("%s PubKey: non-imported address has empty "+
			"derivation info", prefix)
		return false
	}
	expectedDerivationInfo := wantAddr.derivationInfo
	if gotAddrPath != expectedDerivationInfo {
		tc.t.Errorf("%s PubKey: wrong derivation info: expected %v, "+
			"got %v", prefix, spew.Sdump(gotAddrPath),
			spew.Sdump(expectedDerivationInfo))
		return false
	}

//确保私钥是托管地址的预期值。
//
//用于锁定管理器时的预期错误。
	gotPrivKey, err := gotAddr.PrivKey()
	switch {
	case tc.watchingOnly:
//确认预期的仅监视错误。
		testName := fmt.Sprintf("%s PrivKey", prefix)
		if !checkManagerError(tc.t, testName, err, ErrWatchingOnly) {
			return false
		}
	case tc.unlocked:
		if err != nil {
			tc.t.Errorf("%s PrivKey: unexpected error - got %v",
				prefix, err)
			return false
		}
		gpriv := gotPrivKey.Serialize()
		if !reflect.DeepEqual(gpriv, wantAddr.privKey) {
			tc.t.Errorf("%s PrivKey: unexpected private key - "+
				"got %x, want %x", prefix, gpriv, wantAddr.privKey)
			return false
		}
	default:
//确认预期的锁定错误。
		testName := fmt.Sprintf("%s PrivKey", prefix)
		if !checkManagerError(tc.t, testName, err, ErrLocked) {
			return false
		}
	}

//确保以钱包导入格式（WIF）导出的私钥是
//托管地址的预期值。因为只有这个
//当管理器解锁时，还应检查以下情况下的预期错误：
//经理被锁定。
	gotWIF, err := gotAddr.ExportPrivKey()
	switch {
	case tc.watchingOnly:
//确认预期的仅监视错误。
		testName := fmt.Sprintf("%s ExportPrivKey", prefix)
		if !checkManagerError(tc.t, testName, err, ErrWatchingOnly) {
			return false
		}
	case tc.unlocked:
		if err != nil {
			tc.t.Errorf("%s ExportPrivKey: unexpected error - "+
				"got %v", prefix, err)
			return false
		}
		if gotWIF.String() != wantAddr.privKeyWIF {
			tc.t.Errorf("%s ExportPrivKey: unexpected WIF - got "+
				"%v, want %v", prefix, gotWIF.String(),
				wantAddr.privKeyWIF)
			return false
		}
	default:
//确认预期的锁定错误。
		testName := fmt.Sprintf("%s ExportPrivKey", prefix)
		if !checkManagerError(tc.t, testName, err, ErrLocked) {
			return false
		}
	}

//导入的地址应返回零派生信息。
	if _, _, ok := gotAddr.DerivationInfo(); gotAddr.Imported() && ok {
		tc.t.Errorf("%s Imported: expected nil derivation info", prefix)
		return false
	}

	return true
}

//testmanagedscriptaddress确保所有导出函数返回的数据
//
//提供的预期地址中的字段。
//
//当测试上下文指示管理器已解锁时，私有数据
//还将测试处理私有数据的功能，否则
//检查以确保返回正确的错误。
func testManagedScriptAddress(tc *testContext, prefix string,
	gotAddr ManagedScriptAddress, wantAddr *expectedAddr) bool {

//确保脚本是托管地址的预期值。
//确保脚本是托管地址的预期值。自从
//这只在管理器解锁时可用，同时检查
//锁定管理器时出现的预期错误。
	gotScript, err := gotAddr.Script()
	switch {
	case tc.watchingOnly:
//确认预期的仅监视错误。
		testName := fmt.Sprintf("%s Script", prefix)
		if !checkManagerError(tc.t, testName, err, ErrWatchingOnly) {
			return false
		}
	case tc.unlocked:
		if err != nil {
			tc.t.Errorf("%s Script: unexpected error - got %v",
				prefix, err)
			return false
		}
		if !reflect.DeepEqual(gotScript, wantAddr.script) {
			tc.t.Errorf("%s Script: unexpected script - got %x, "+
				"want %x", prefix, gotScript, wantAddr.script)
			return false
		}
	default:
//确认预期的锁定错误。
		testName := fmt.Sprintf("%s Script", prefix)
		if !checkManagerError(tc.t, testName, err, ErrLocked) {
			return false
		}
	}

	return true
}

//testaddress确保由提供的所有导出函数返回的数据
//传递的托管地址与提供的
//需要地址。它还键入断言托管地址以确定其
//并相应调用相应的测试函数。
//
//当测试上下文指示管理器已解锁时，私有数据
//还将测试处理私有数据的功能，否则
//检查以确保返回正确的错误。
func testAddress(tc *testContext, prefix string, gotAddr ManagedAddress,
	wantAddr *expectedAddr) bool {

	if gotAddr.Account() != tc.account {
		tc.t.Errorf("ManagedAddress.Account: unexpected account - got "+
			"%d, want %d", gotAddr.Account(), tc.account)
		return false
	}

	if gotAddr.Address().EncodeAddress() != wantAddr.address {
		tc.t.Errorf("%s EncodeAddress: unexpected address - got %s, "+
			"want %s", prefix, gotAddr.Address().EncodeAddress(),
			wantAddr.address)
		return false
	}

	if !reflect.DeepEqual(gotAddr.AddrHash(), wantAddr.addressHash) {
		tc.t.Errorf("%s AddrHash: unexpected address hash - got %x, "+
			"want %x", prefix, gotAddr.AddrHash(),
			wantAddr.addressHash)
		return false
	}

	if gotAddr.Internal() != wantAddr.internal {
		tc.t.Errorf("%s Internal: unexpected internal flag - got %v, "+
			"want %v", prefix, gotAddr.Internal(), wantAddr.internal)
		return false
	}

	if gotAddr.Compressed() != wantAddr.compressed {
		tc.t.Errorf("%s Compressed: unexpected compressed flag - got "+
			"%v, want %v", prefix, gotAddr.Compressed(),
			wantAddr.compressed)
		return false
	}

	if gotAddr.Imported() != wantAddr.imported {
		tc.t.Errorf("%s Imported: unexpected imported flag - got %v, "+
			"want %v", prefix, gotAddr.Imported(), wantAddr.imported)
		return false
	}

	switch addr := gotAddr.(type) {
	case ManagedPubKeyAddress:
		if !testManagedPubKeyAddress(tc, prefix, addr, wantAddr) {
			return false
		}

	case ManagedScriptAddress:
		if !testManagedScriptAddress(tc, prefix, addr, wantAddr) {
			return false
		}
	}

	return true
}

//testExternalAddresses测试外部地址的几个方面，例如
//通过nextexternaladdress生成多个地址，确保它们可以
//按地址检索，并且当管理器被锁定时它们工作正常
//解锁。
func testExternalAddresses(tc *testContext) bool {
	prefix := testNamePrefix(tc) + " testExternalAddresses"
	var addrs []ManagedAddress
	if tc.create {
		prefix := prefix + " NextExternalAddresses"
		var addrs []ManagedAddress
		err := walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
			ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
			var err error
			addrs, err = tc.manager.NextExternalAddresses(ns, tc.account, 5)
			return err
		})
		if err != nil {
			tc.t.Errorf("%s: unexpected error: %v", prefix, err)
			return false
		}
		if len(addrs) != len(expectedExternalAddrs) {
			tc.t.Errorf("%s: unexpected number of addresses - got "+
				"%d, want %d", prefix, len(addrs),
				len(expectedExternalAddrs))
			return false
		}
	}

//设置一个闭包来测试结果，因为相同的测试需要
//在经理锁定和解锁的情况下重复。
	testResults := func() bool {
//确保返回的地址是预期的地址。什么时候？
//不在创建阶段，在
//加法器切片，所以这只在第一阶段运行
//在测试中。
		for i := 0; i < len(addrs); i++ {
			prefix := fmt.Sprintf("%s ExternalAddress #%d", prefix, i)
			if !testAddress(tc, prefix, addrs[i], &expectedExternalAddrs[i]) {
				return false
			}
		}

//确保最后一个外部地址是预期地址。
		leaPrefix := prefix + " LastExternalAddress"
		var lastAddr ManagedAddress
		err := walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
			ns := tx.ReadBucket(waddrmgrNamespaceKey)
			var err error
			lastAddr, err = tc.manager.LastExternalAddress(ns, tc.account)
			return err
		})
		if err != nil {
			tc.t.Errorf("%s: unexpected error: %v", leaPrefix, err)
			return false
		}
		if !testAddress(tc, leaPrefix, lastAddr, &expectedExternalAddrs[len(expectedExternalAddrs)-1]) {
			return false
		}

//现在，使用地址API检索每个预期的
//地址并确保它们是准确的。
		chainParams := tc.manager.ChainParams()
		for i := 0; i < len(expectedExternalAddrs); i++ {
			pkHash := expectedExternalAddrs[i].addressHash
			utilAddr, err := btcutil.NewAddressPubKeyHash(
				pkHash, chainParams,
			)
			if err != nil {
				tc.t.Errorf("%s NewAddressPubKeyHash #%d: "+
					"unexpected error: %v", prefix, i, err)
				return false
			}

			prefix := fmt.Sprintf("%s Address #%d", prefix, i)
			var addr ManagedAddress
			err = walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
				ns := tx.ReadBucket(waddrmgrNamespaceKey)
				var err error
				addr, err = tc.manager.Address(ns, utilAddr)
				return err
			})
			if err != nil {
				tc.t.Errorf("%s: unexpected error: %v", prefix,
					err)
				return false
			}

			if !testAddress(tc, prefix, addr, &expectedExternalAddrs[i]) {
				return false
			}
		}

		return true
	}

//由于管理器此时处于锁定状态，公共地址
//测试信息并检查私有功能以确保
//它们返回预期的错误。
	if !testResults() {
		return false
	}

//这一点之后的所有事情都需要重新测试
//地址管理器，不能只监视模式，所以
//在这种情况下，现在就退出。
	if tc.watchingOnly {
		return true
	}

//解锁管理器并重新测试所有地址，以确保
//私人信息也是有效的。
	err := walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		return tc.rootManager.Unlock(ns, privPassphrase)
	})
	if err != nil {
		tc.t.Errorf("Unlock: unexpected error: %v", err)
		return false
	}
	tc.unlocked = true
	if !testResults() {
		return false
	}

//重新锁定管理器以备将来测试。
	if err := tc.rootManager.Lock(); err != nil {
		tc.t.Errorf("Lock: unexpected error: %v", err)
		return false
	}
	tc.unlocked = false

	return true
}

//TestInternalAddresses测试内部地址的几个方面，例如
//
//按地址检索，并且当管理器被锁定时它们工作正常
//解锁。
func testInternalAddresses(tc *testContext) bool {
//当地址管理器不处于仅监视模式时，将其解锁。
//首先确保地址生成在
//地址管理器已解锁，稍后将被锁定。这些测试
//颠倒外部测试中以
//锁定管理器，然后将其解锁。
	if !tc.watchingOnly {
//解锁管理器并重新测试所有地址，以确保
//私人信息也是有效的。
		err := walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
			ns := tx.ReadBucket(waddrmgrNamespaceKey)
			return tc.rootManager.Unlock(ns, privPassphrase)
		})
		if err != nil {
			tc.t.Errorf("Unlock: unexpected error: %v", err)
			return false
		}
		tc.unlocked = true
	}

	prefix := testNamePrefix(tc) + " testInternalAddresses"
	var addrs []ManagedAddress
	if tc.create {
		prefix := prefix + " NextInternalAddress"
		err := walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
			ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
			var err error
			addrs, err = tc.manager.NextInternalAddresses(ns, tc.account, 5)
			return err
		})
		if err != nil {
			tc.t.Errorf("%s: unexpected error: %v", prefix, err)
			return false
		}
		if len(addrs) != len(expectedInternalAddrs) {
			tc.t.Errorf("%s: unexpected number of addresses - got "+
				"%d, want %d", prefix, len(addrs),
				len(expectedInternalAddrs))
			return false
		}
	}

//设置一个闭包来测试结果，因为相同的测试需要
//在经理锁定和解锁的情况下重复。
	testResults := func() bool {
//确保返回的地址是预期的地址。什么时候？
//不在创建阶段，在
//加法器切片，所以这只在第一阶段运行
//在测试中。
		for i := 0; i < len(addrs); i++ {
			prefix := fmt.Sprintf("%s InternalAddress #%d", prefix, i)
			if !testAddress(tc, prefix, addrs[i], &expectedInternalAddrs[i]) {
				return false
			}
		}

//确保最后一个内部地址是预期地址。
		liaPrefix := prefix + " LastInternalAddress"
		var lastAddr ManagedAddress
		err := walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
			ns := tx.ReadBucket(waddrmgrNamespaceKey)
			var err error
			lastAddr, err = tc.manager.LastInternalAddress(ns, tc.account)
			return err
		})
		if err != nil {
			tc.t.Errorf("%s: unexpected error: %v", liaPrefix, err)
			return false
		}
		if !testAddress(tc, liaPrefix, lastAddr, &expectedInternalAddrs[len(expectedInternalAddrs)-1]) {
			return false
		}

//现在，使用地址API检索每个预期的
//地址并确保它们是准确的。
		chainParams := tc.manager.ChainParams()
		for i := 0; i < len(expectedInternalAddrs); i++ {
			pkHash := expectedInternalAddrs[i].addressHash
			utilAddr, err := btcutil.NewAddressPubKeyHash(
				pkHash, chainParams,
			)
			if err != nil {
				tc.t.Errorf("%s NewAddressPubKeyHash #%d: "+
					"unexpected error: %v", prefix, i, err)
				return false
			}

			prefix := fmt.Sprintf("%s Address #%d", prefix, i)
			var addr ManagedAddress
			err = walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
				ns := tx.ReadBucket(waddrmgrNamespaceKey)
				var err error
				addr, err = tc.manager.Address(ns, utilAddr)
				return err
			})
			if err != nil {
				tc.t.Errorf("%s: unexpected error: %v", prefix,
					err)
				return false
			}

			if !testAddress(tc, prefix, addr, &expectedInternalAddrs[i]) {
				return false
			}
		}

		return true
	}

//地址管理器可以在此处锁定或解锁，具体取决于
//关于它是否是一个只负责观察的经理。解锁后，
//这将测试公共和私有地址数据是否准确。
//当它被锁上时，它一定只是在看，所以只有公众
//测试地址信息并检查专用功能
//以确保返回预期的errWatchingOnly错误。
	if !testResults() {
		return false
	}

//这之后的所有操作都涉及到锁定地址管理器和
//用锁定的管理器重新测试地址。然而，对于
//只看模式，这已经发生了，现在退出
//那个案子。
	if tc.watchingOnly {
		return true
	}

//锁定管理器并重新测试所有地址，以确保
//公共信息仍然有效，私有功能返回
//预期的错误。
	if err := tc.rootManager.Lock(); err != nil {
		tc.t.Errorf("Lock: unexpected error: %v", err)
		return false
	}
	tc.unlocked = false
	if !testResults() {
		return false
	}

	return true
}

//testlocking测试地址管理器工作的基本锁定语义
//
//以及未锁定的条件。
func testLocking(tc *testContext) bool {
	if tc.unlocked {
		tc.t.Error("testLocking called with an unlocked manager")
		return false
	}
	if !tc.rootManager.IsLocked() {
		tc.t.Error("IsLocked: returned false on locked manager")
		return false
	}

//锁定已锁定的锁定管理器应返回错误。误差
//应为errlocked或errwatchingOnly，具体取决于
//地址管理器。
	err := tc.rootManager.Lock()
	wantErrCode := ErrLocked
	if tc.watchingOnly {
		wantErrCode = ErrWatchingOnly
	}
	if !checkManagerError(tc.t, "Lock", err, wantErrCode) {
		return false
	}

//确保使用正确的密码短语解锁不会返回任何
//意外错误，管理器正确报告它已解锁。
//既然只看地址管理器不能解锁，也要确保
//
	err = walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		return tc.rootManager.Unlock(ns, privPassphrase)
	})
	if tc.watchingOnly {
		if !checkManagerError(tc.t, "Unlock", err, ErrWatchingOnly) {
			return false
		}
	} else if err != nil {
		tc.t.Errorf("Unlock: unexpected error: %v", err)
		return false
	}
	if !tc.watchingOnly && tc.rootManager.IsLocked() {
		tc.t.Error("IsLocked: returned true on unlocked manager")
		return false
	}

//允许再次解锁管理器。因为只看地址
//管理者不能解锁，也要确保正确的错误。
//案例。
	err = walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		return tc.rootManager.Unlock(ns, privPassphrase)
	})
	if tc.watchingOnly {
		if !checkManagerError(tc.t, "Unlock2", err, ErrWatchingOnly) {
			return false
		}
	} else if err != nil {
		tc.t.Errorf("Unlock: unexpected error: %v", err)
		return false
	}
	if !tc.watchingOnly && tc.rootManager.IsLocked() {
		tc.t.Error("IsLocked: returned true on unlocked manager")
		return false
	}

//使用无效的密码短语解锁管理器必须导致
//错误和锁定的管理器。
	err = walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		return tc.rootManager.Unlock(ns, []byte("invalidpassphrase"))
	})
	wantErrCode = ErrWrongPassphrase
	if tc.watchingOnly {
		wantErrCode = ErrWatchingOnly
	}
	if !checkManagerError(tc.t, "Unlock", err, wantErrCode) {
		return false
	}
	if !tc.rootManager.IsLocked() {
		tc.t.Error("IsLocked: manager is unlocked after failed unlock " +
			"attempt")
		return false
	}

	return true
}

//tesimportprivatekey测试导入私钥是否正常工作。它
//
//当管理器被锁定时，地址给出了预期值，并且
//解锁。
//
//此函数要求在调用管理器时该管理器已被锁定并返回
//经理被锁住了。
func testImportPrivateKey(tc *testContext) bool {
	tests := []struct {
		name       string
		in         string
		blockstamp BlockStamp
		expected   expectedAddr
	}{
		{
			name: "wif for uncompressed pubkey address",
			in:   "5HueCGU8rMjxEXxiPuD5BDku4MkFqeZyd4dZ1jvhTVqvbTLvyTJ",
			expected: expectedAddr{
				address:     "1GAehh7TsJAHuUAeKZcXf5CnwuGuGgyX2S",
				addressHash: hexToBytes("a65d1a239d4ec666643d350c7bb8fc44d2881128"),
				internal:    false,
				imported:    true,
				compressed:  false,
				pubKey: hexToBytes("04d0de0aaeaefad02b8bdc8a01a1b8b11c696bd3" +
					"d66a2c5f10780d95b7df42645cd85228a6fb29940e858e7e558" +
					"42ae2bd115d1ed7cc0e82d934e929c97648cb0a"),
				privKey: hexToBytes("0c28fca386c7a227600b2fe50b7cae11ec86d3bf1fbe471be89827e19d72aa1d"),
//在测试期间，privkeywif设置为in字段
			},
		},
		{
			name: "wif for compressed pubkey address",
			in:   "KwdMAjGmerYanjeui5SHS7JkmpZvVipYvB2LJGU1ZxJwYvP98617",
			expected: expectedAddr{
				address:     "1LoVGDgRs9hTfTNJNuXKSpywcbdvwRXpmK",
				addressHash: hexToBytes("d9351dcbad5b8f3b8bfa2f2cdc85c28118ca9326"),
				internal:    false,
				imported:    true,
				compressed:  true,
				pubKey:      hexToBytes("02d0de0aaeaefad02b8bdc8a01a1b8b11c696bd3d66a2c5f10780d95b7df42645c"),
				privKey:     hexToBytes("0c28fca386c7a227600b2fe50b7cae11ec86d3bf1fbe471be89827e19d72aa1d"),
//在测试期间，privkeywif设置为in字段
			},
		},
	}

//必须解锁管理器才能导入私钥，但是
//只看经理是不能解锁的。
	if !tc.watchingOnly {
		err := walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
			ns := tx.ReadBucket(waddrmgrNamespaceKey)
			return tc.rootManager.Unlock(ns, privPassphrase)
		})
		if err != nil {
			tc.t.Errorf("Unlock: unexpected error: %v", err)
			return false
		}
		tc.unlocked = true
	}

//只有在测试的创建阶段才导入私钥。
	tc.account = ImportedAddrAccount
	prefix := testNamePrefix(tc) + " testImportPrivateKey"
	if tc.create {
		for i, test := range tests {
			test.expected.privKeyWIF = test.in
			wif, err := btcutil.DecodeWIF(test.in)
			if err != nil {
				tc.t.Errorf("%s DecodeWIF #%d (%s): unexpected "+
					"error: %v", prefix, i, test.name, err)
				continue
			}
			var addr ManagedPubKeyAddress
			err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
				ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
				var err error
				addr, err = tc.manager.ImportPrivateKey(ns, wif, &test.blockstamp)
				return err
			})
			if err != nil {
				tc.t.Errorf("%s ImportPrivateKey #%d (%s): "+
					"unexpected error: %v", prefix, i,
					test.name, err)
				continue
			}
			if !testAddress(tc, prefix+" ImportPrivateKey", addr,
				&test.expected) {
				continue
			}
		}
	}

//设置一个闭包来测试结果，因为相同的测试需要
//在经理解锁并锁定的情况下重复。
	chainParams := tc.manager.ChainParams()
	testResults := func() bool {
		failed := false
		for i, test := range tests {
			test.expected.privKeyWIF = test.in

//使用地址API检索每个预期的
//新地址并确保它们是准确的。
			utilAddr, err := btcutil.NewAddressPubKeyHash(
				test.expected.addressHash, chainParams)
			if err != nil {
				tc.t.Errorf("%s NewAddressPubKeyHash #%d (%s): "+
					"unexpected error: %v", prefix, i,
					test.name, err)
				failed = true
				continue
			}
			taPrefix := fmt.Sprintf("%s Address #%d (%s)", prefix,
				i, test.name)
			var ma ManagedAddress
			err = walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
				ns := tx.ReadBucket(waddrmgrNamespaceKey)
				var err error
				ma, err = tc.manager.Address(ns, utilAddr)
				return err
			})
			if err != nil {
				tc.t.Errorf("%s: unexpected error: %v", taPrefix,
					err)
				failed = true
				continue
			}
			if !testAddress(tc, taPrefix, ma, &test.expected) {
				failed = true
				continue
			}
		}

		return !failed
	}

//地址管理器可以在此处锁定或解锁，具体取决于
//关于它是否是一个只负责观察的经理。解锁后，
//这将测试公共和私有地址数据是否准确。
//当它被锁上时，它一定只是在看，所以只有公众
//测试地址信息并检查专用功能
//以确保返回预期的errWatchingOnly错误。
	if !testResults() {
		return false
	}

//这之后的所有操作都涉及到锁定地址管理器和
//用锁定的管理器重新测试地址。然而，对于
//只看模式，这已经发生了，现在退出
//那个案子。
	if tc.watchingOnly {
		return true
	}

//锁定管理器并重新测试所有地址，以确保
//私有信息返回预期的错误。
	if err := tc.rootManager.Lock(); err != nil {
		tc.t.Errorf("Lock: unexpected error: %v", err)
		return false
	}
	tc.unlocked = false
	if !testResults() {
		return false
	}

	return true
}

//
//它们可以在导入后按地址检索，并且
//当管理器被锁定和解锁时，地址会给出预期的值。
//
//此函数要求在调用管理器时该管理器已被锁定并返回
//经理被锁住了。
func testImportScript(tc *testContext) bool {
	tests := []struct {
		name       string
		in         []byte
		blockstamp BlockStamp
		expected   expectedAddr
	}{
		{
			name: "p2sh uncompressed pubkey",
			in: hexToBytes("41048b65a0e6bb200e6dac05e74281b1ab9a41e8" +
				"0006d6b12d8521e09981da97dd96ac72d24d1a7d" +
				"ed9493a9fc20fdb4a714808f0b680f1f1d935277" +
				"48b5e3f629ffac"),
			expected: expectedAddr{
				address:     "3MbyWAu9UaoBewR3cArF1nwf4aQgVwzrA5",
				addressHash: hexToBytes("da6e6a632d96dc5530d7b3c9f3017725d023093e"),
				internal:    false,
				imported:    true,
				compressed:  false,
//测试期间，脚本设置为In字段。
			},
		},
		{
			name: "p2sh multisig",
			in: hexToBytes("524104cb9c3c222c5f7a7d3b9bd152f363a0b6d5" +
				"4c9eb312c4d4f9af1e8551b6c421a6a4ab0e2910" +
				"5f24de20ff463c1c91fcf3bf662cdde4783d4799" +
				"f787cb7c08869b4104ccc588420deeebea22a7e9" +
				"00cc8b68620d2212c374604e3487ca08f1ff3ae1" +
				"2bdc639514d0ec8612a2d3c519f084d9a00cbbe3" +
				"b53d071e9b09e71e610b036aa24104ab47ad1939" +
				"edcb3db65f7fedea62bbf781c5410d3f22a7a3a5" +
				"6ffefb2238af8627363bdf2ed97c1f89784a1aec" +
				"db43384f11d2acc64443c7fc299cef0400421a53ae"),
			expected: expectedAddr{
				address:     "34CRZpt8j81rgh9QhzuBepqPi4cBQSjhjr",
				addressHash: hexToBytes("1b800cec1fe92222f36a502c139bed47c5959715"),
				internal:    false,
				imported:    true,
				compressed:  false,
//测试期间，脚本设置为In字段。
			},
		},
	}

//必须解锁管理器才能导入私钥，并且
//测试私有数据。但是，一个只看的经理不能
//解锁。
	if !tc.watchingOnly {
		err := walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
			ns := tx.ReadBucket(waddrmgrNamespaceKey)
			return tc.rootManager.Unlock(ns, privPassphrase)
		})
		if err != nil {
			tc.t.Errorf("Unlock: unexpected error: %v", err)
			return false
		}
		tc.unlocked = true
	}

//仅在测试的创建阶段导入脚本。
	tc.account = ImportedAddrAccount
	prefix := testNamePrefix(tc)
	if tc.create {
		for i, test := range tests {
			test.expected.script = test.in
			prefix := fmt.Sprintf("%s ImportScript #%d (%s)", prefix,
				i, test.name)

			var addr ManagedScriptAddress
			err := walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
				ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
				var err error
				addr, err = tc.manager.ImportScript(ns, test.in, &test.blockstamp)
				return err
			})
			if err != nil {
				tc.t.Errorf("%s: unexpected error: %v", prefix,
					err)
				continue
			}
			if !testAddress(tc, prefix, addr, &test.expected) {
				continue
			}
		}
	}

//设置一个闭包来测试结果，因为相同的测试需要
//在经理解锁并锁定的情况下重复。
	chainParams := tc.manager.ChainParams()
	testResults := func() bool {
		failed := false
		for i, test := range tests {
			test.expected.script = test.in

//使用地址API检索每个预期的
//新地址并确保它们是准确的。
			utilAddr, err := btcutil.NewAddressScriptHash(test.in,
				chainParams)
			if err != nil {
				tc.t.Errorf("%s NewAddressScriptHash #%d (%s): "+
					"unexpected error: %v", prefix, i,
					test.name, err)
				failed = true
				continue
			}
			taPrefix := fmt.Sprintf("%s Address #%d (%s)", prefix,
				i, test.name)
			var ma ManagedAddress
			err = walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
				ns := tx.ReadBucket(waddrmgrNamespaceKey)
				var err error
				ma, err = tc.manager.Address(ns, utilAddr)
				return err
			})
			if err != nil {
				tc.t.Errorf("%s: unexpected error: %v", taPrefix,
					err)
				failed = true
				continue
			}
			if !testAddress(tc, taPrefix, ma, &test.expected) {
				failed = true
				continue
			}
		}

		return !failed
	}

//地址管理器可以在此处锁定或解锁，具体取决于
//关于它是否是一个只负责观察的经理。解锁后，
//这将测试公共和私有地址数据是否准确。
//当它被锁上时，它一定只是在看，所以只有公众
//测试地址信息并检查专用功能
//以确保返回预期的errWatchingOnly错误。
	if !testResults() {
		return false
	}

//这之后的所有操作都涉及到锁定地址管理器和
//用锁定的管理器重新测试地址。然而，对于
//只看模式，这已经发生了，现在退出
//那个案子。
	if tc.watchingOnly {
		return true
	}

//锁定管理器并重新测试所有地址，以确保
//私有信息返回预期的错误。
	if err := tc.rootManager.Lock(); err != nil {
		tc.t.Errorf("Lock: unexpected error: %v", err)
		return false
	}
	tc.unlocked = false
	if !testResults() {
		return false
	}

	return true
}

//
func testMarkUsed(tc *testContext) bool {
	tests := []struct {
		name string
		typ  addrType
		in   []byte
	}{
		{
			name: "managed address",
			typ:  addrPubKeyHash,
			in:   hexToBytes("2ef94abb9ee8f785d087c3ec8d6ee467e92d0d0a"),
		},
		{
			name: "script address",
			typ:  addrScriptHash,
			in:   hexToBytes("da6e6a632d96dc5530d7b3c9f3017725d023093e"),
		},
	}

	prefix := "MarkUsed"
	chainParams := tc.manager.ChainParams()
	for i, test := range tests {
		addrHash := test.in

		var addr btcutil.Address
		var err error
		switch test.typ {
		case addrPubKeyHash:
			addr, err = btcutil.NewAddressPubKeyHash(addrHash, chainParams)
		case addrScriptHash:
			addr, err = btcutil.NewAddressScriptHashFromHash(addrHash, chainParams)
		default:
			panic("unreachable")
		}
		if err != nil {
			tc.t.Errorf("%s #%d: NewAddress unexpected error: %v", prefix, i, err)
			continue
		}

		err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
			ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)

			maddr, err := tc.manager.Address(ns, addr)
			if err != nil {
				tc.t.Errorf("%s #%d: Address unexpected error: %v", prefix, i, err)
				return nil
			}
			if tc.create {
//测试最初地址是否未标记为已使用
				used := maddr.Used(ns)
				if used != false {
					tc.t.Errorf("%s #%d: unexpected used flag -- got "+
						"%v, want %v", prefix, i, used, false)
				}
			}
			err = tc.manager.MarkUsed(ns, addr)
			if err != nil {
				tc.t.Errorf("%s #%d: unexpected error: %v", prefix, i, err)
				return nil
			}
			used := maddr.Used(ns)
			if used != true {
				tc.t.Errorf("%s #%d: unexpected used flag -- got "+
					"%v, want %v", prefix, i, used, true)
			}
			return nil
		})
		if err != nil {
			tc.t.Errorf("Unexpected error %v", err)
		}
	}

	return true
}

//testchangepassphrase确保更改公共和私有密码。
//按预期工作。
func testChangePassphrase(tc *testContext) bool {
//由于未能将密码短语更改为
//通过替换生成函数1生成新的密钥
//那是故意的错误。
	testName := "ChangePassphrase (public) with invalid new secret key"

	oldKeyGen := SetSecretKeyGen(failingSecretKeyGen)
	err := walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return tc.rootManager.ChangePassphrase(
			ns, pubPassphrase, pubPassphrase2, false, fastScrypt,
		)
	})
	if !checkManagerError(tc.t, testName, err, ErrCrypto) {
		return false
	}

//
	testName = "ChangePassphrase (public) with invalid old passphrase"
	SetSecretKeyGen(oldKeyGen)
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return tc.rootManager.ChangePassphrase(
			ns, []byte("bogus"), pubPassphrase2, false, fastScrypt,
		)
	})
	if !checkManagerError(tc.t, testName, err, ErrWrongPassphrase) {
		return false
	}

//更改公共密码。
	testName = "ChangePassphrase (public)"
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return tc.rootManager.ChangePassphrase(
			ns, pubPassphrase, pubPassphrase2, false, fastScrypt,
		)
	})
	if err != nil {
		tc.t.Errorf("%s: unexpected error: %v", testName, err)
		return false
	}

//确保已成功更改公共密码短语。我们这样做
//能够用新的密码短语重新派生公钥。
	secretKey := snacl.SecretKey{Key: &snacl.CryptoKey{}}
	secretKey.Parameters = tc.rootManager.masterKeyPub.Parameters
	if err := secretKey.DeriveKey(&pubPassphrase2); err != nil {
		tc.t.Errorf("%s: passphrase does not match", testName)
		return false
	}

//把私人密码改回原样。
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return tc.rootManager.ChangePassphrase(
			ns, pubPassphrase2, pubPassphrase, false, fastScrypt,
		)
	})
	if err != nil {
		tc.t.Errorf("%s: unexpected error: %v", testName, err)
		return false
	}

//尝试用无效的旧密码更改私人密码。
//错误应为errWrongPassphrase或errWatchingOnly，具体取决于
//地址管理器的类型。
	testName = "ChangePassphrase (private) with invalid old passphrase"
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return tc.rootManager.ChangePassphrase(
			ns, []byte("bogus"), privPassphrase2, true, fastScrypt,
		)
	})
	wantErrCode := ErrWrongPassphrase
	if tc.watchingOnly {
		wantErrCode = ErrWatchingOnly
	}
	if !checkManagerError(tc.t, testName, err, wantErrCode) {
		return false
	}

//在这之后的所有事情都涉及到测试
//可以成功更改地址管理器的密码。
//这不可能只监视模式，所以现在退出
//案例。
	if tc.watchingOnly {
		return true
	}

//更改私人密码。
	testName = "ChangePassphrase (private)"
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return tc.rootManager.ChangePassphrase(
			ns, privPassphrase, privPassphrase2, true, fastScrypt,
		)
	})
	if err != nil {
		tc.t.Errorf("%s: unexpected error: %v", testName, err)
		return false
	}

//使用新密码短语解锁管理器，以确保其更改为
//预期。
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return tc.rootManager.Unlock(ns, privPassphrase2)
	})
	if err != nil {
		tc.t.Errorf("%s: failed to unlock with new private "+
			"passphrase: %v", testName, err)
		return false
	}
	tc.unlocked = true

//将私人密码改回经理时的密码。
//解锁以确保路径也正常工作。
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return tc.rootManager.ChangePassphrase(
			ns, privPassphrase2, privPassphrase, true, fastScrypt,
		)
	})
	if err != nil {
		tc.t.Errorf("%s: unexpected error: %v", testName, err)
		return false
	}
	if tc.rootManager.IsLocked() {
		tc.t.Errorf("%s: manager is locked", testName)
		return false
	}

//重新锁定管理器以备将来测试。
	if err := tc.rootManager.Lock(); err != nil {
		tc.t.Errorf("Lock: unexpected error: %v", err)
		return false
	}
	tc.unlocked = false

	return true
}

//testNewAccount测试地址管理器的新帐户创建功能
//果不其然。
func testNewAccount(tc *testContext) bool {
	if tc.watchingOnly {
//在仅监视模式下创建新帐户应返回errWatchingOnly
		err := walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
			ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
			_, err := tc.manager.NewAccount(ns, "test")
			return err
		})
		if !checkManagerError(
			tc.t, "Create account in watching-only mode", err,
			ErrWatchingOnly,
		) {
			tc.manager.Close()
			return false
		}
		return true
	}
//钱包锁定时创建新帐户应返回errlocked
	err := walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		_, err := tc.manager.NewAccount(ns, "test")
		return err
	})
	if !checkManagerError(
		tc.t, "Create account when wallet is locked", err, ErrLocked,
	) {
		tc.manager.Close()
		return false
	}
//解锁钱包以解密所需的CoinType密钥
//派生帐户密钥
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		err := tc.rootManager.Unlock(ns, privPassphrase)
		return err
	})
	if err != nil {
		tc.t.Errorf("Unlock: unexpected error: %v", err)
		return false
	}
	tc.unlocked = true

	testName := "acct-create"
	expectedAccount := tc.account + 1
	if !tc.create {
//在打开模式下创建新帐户
		testName = "acct-open"
		expectedAccount++
	}
	var account uint32
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		var err error
		account, err = tc.manager.NewAccount(ns, testName)
		return err
	})
	if err != nil {
		tc.t.Errorf("NewAccount: unexpected error: %v", err)
		return false
	}
	if account != expectedAccount {
		tc.t.Errorf("NewAccount "+
			"account mismatch -- got %d, "+
			"want %d", account, expectedAccount)
		return false
	}

//测试重复帐户名错误
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		_, err := tc.manager.NewAccount(ns, testName)
		return err
	})
	wantErrCode := ErrDuplicateAccount
	if !checkManagerError(tc.t, testName, err, wantErrCode) {
		return false
	}
//测试帐户名验证
testName = "" //不允许空帐户名
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		_, err := tc.manager.NewAccount(ns, testName)
		return err
	})
	wantErrCode = ErrInvalidAccount
	if !checkManagerError(tc.t, testName, err, wantErrCode) {
		return false
	}
testName = "imported" //
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		_, err := tc.manager.NewAccount(ns, testName)
		return err
	})
	wantErrCode = ErrInvalidAccount
	if !checkManagerError(tc.t, testName, err, wantErrCode) {
		return false
	}
	return true
}

//testLookupAccount测试地址管理器的基本帐户查找功能
//按预期工作。
func testLookupAccount(tc *testContext) bool {
//查找在testnewaccount中早期创建的帐户
	expectedAccounts := map[string]uint32{
		defaultAccountName:      DefaultAccountNum,
		ImportedAddrAccountName: ImportedAddrAccount,
	}
	for acctName, expectedAccount := range expectedAccounts {
		var account uint32
		err := walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
			ns := tx.ReadBucket(waddrmgrNamespaceKey)
			var err error
			account, err = tc.manager.LookupAccount(ns, acctName)
			return err
		})
		if err != nil {
			tc.t.Errorf("LookupAccount: unexpected error: %v", err)
			return false
		}
		if account != expectedAccount {
			tc.t.Errorf("LookupAccount "+
				"account mismatch -- got %d, "+
				"want %d", account, expectedAccount)
			return false
		}
	}
//未找到测试帐户错误
	testName := "non existent account"
	err := walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		_, err := tc.manager.LookupAccount(ns, testName)
		return err
	})
	wantErrCode := ErrAccountNotFound
	if !checkManagerError(tc.t, testName, err, wantErrCode) {
		return false
	}

//测试最后一个帐户
	var lastAccount uint32
	err = walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		var err error
		lastAccount, err = tc.manager.LastAccount(ns)
		return err
	})
	var expectedLastAccount uint32
	expectedLastAccount = 1
	if !tc.create {
//现有钱包经理将拥有3个账户
		expectedLastAccount = 2
	}
	if lastAccount != expectedLastAccount {
		tc.t.Errorf("LookupAccount "+
			"account mismatch -- got %d, "+
			"want %d", lastAccount, expectedLastAccount)
		return false
	}

//测试默认帐户地址的帐户查找
	var expectedAccount uint32
	for i, addr := range expectedAddrs {
		addr, err := btcutil.NewAddressPubKeyHash(addr.addressHash,
			tc.manager.ChainParams())
		if err != nil {
			tc.t.Errorf("AddrAccount #%d: unexpected error: %v", i, err)
			return false
		}
		var account uint32
		err = walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
			ns := tx.ReadBucket(waddrmgrNamespaceKey)
			var err error
			account, err = tc.manager.AddrAccount(ns, addr)
			return err
		})
		if err != nil {
			tc.t.Errorf("AddrAccount #%d: unexpected error: %v", i, err)
			return false
		}
		if account != expectedAccount {
			tc.t.Errorf("AddrAccount "+
				"account mismatch -- got %d, "+
				"want %d", account, expectedAccount)
			return false
		}
	}
	return true
}

//testrenameaccount测试地址管理器的重命名帐户功能
//果不其然。
func testRenameAccount(tc *testContext) bool {
	var acctName string
	err := walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		var err error
		acctName, err = tc.manager.AccountName(ns, tc.account)
		return err
	})
	if err != nil {
		tc.t.Errorf("AccountName: unexpected error: %v", err)
		return false
	}
	testName := acctName + "-renamed"
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return tc.manager.RenameAccount(ns, tc.account, testName)
	})
	if err != nil {
		tc.t.Errorf("RenameAccount: unexpected error: %v", err)
		return false
	}
	var newName string
	err = walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		var err error
		newName, err = tc.manager.AccountName(ns, tc.account)
		return err
	})
	if err != nil {
		tc.t.Errorf("AccountName: unexpected error: %v", err)
		return false
	}
	if newName != testName {
		tc.t.Errorf("RenameAccount "+
			"account name mismatch -- got %s, "+
			"want %s", newName, testName)
		return false
	}
//测试重复帐户名错误
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return tc.manager.RenameAccount(ns, tc.account, testName)
	})
	wantErrCode := ErrDuplicateAccount
	if !checkManagerError(tc.t, testName, err, wantErrCode) {
		return false
	}
//
	err = walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		_, err := tc.manager.LookupAccount(ns, acctName)
		return err
	})
	wantErrCode = ErrAccountNotFound
	if !checkManagerError(tc.t, testName, err, wantErrCode) {
		return false
	}
	return true
}

//testforeachaccount测试地址的retrieve all accounts函数
//经理按预期工作。
func testForEachAccount(tc *testContext) bool {
	prefix := testNamePrefix(tc) + " testForEachAccount"
	expectedAccounts := []uint32{0, 1}
	if !tc.create {
//现有钱包经理将拥有3个账户
		expectedAccounts = append(expectedAccounts, 2)
	}
//导入的帐户
	expectedAccounts = append(expectedAccounts, ImportedAddrAccount)
	var accounts []uint32
	err := walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		return tc.manager.ForEachAccount(ns, func(account uint32) error {
			accounts = append(accounts, account)
			return nil
		})
	})
	if err != nil {
		tc.t.Errorf("%s: unexpected error: %v", prefix, err)
		return false
	}
	if len(accounts) != len(expectedAccounts) {
		tc.t.Errorf("%s: unexpected number of accounts - got "+
			"%d, want %d", prefix, len(accounts),
			len(expectedAccounts))
		return false
	}
	for i, account := range accounts {
		if expectedAccounts[i] != account {
			tc.t.Errorf("%s #%d: "+
				"account mismatch -- got %d, "+
				"want %d", prefix, i, account, expectedAccounts[i])
		}
	}
	return true
}

//迭代给定的
//使用Manager API的帐户地址按预期工作。
func testForEachAccountAddress(tc *testContext) bool {
	prefix := testNamePrefix(tc) + " testForEachAccountAddress"
//
	expectedAddrMap := make(map[string]*expectedAddr, len(expectedAddrs))
	for i := 0; i < len(expectedAddrs); i++ {
		expectedAddrMap[expectedAddrs[i].address] = &expectedAddrs[i]
	}

	var addrs []ManagedAddress
	err := walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		return tc.manager.ForEachAccountAddress(ns, tc.account,
			func(maddr ManagedAddress) error {
				addrs = append(addrs, maddr)
				return nil
			})
	})
	if err != nil {
		tc.t.Errorf("%s: unexpected error: %v", prefix, err)
		return false
	}

	for i := 0; i < len(addrs); i++ {
		prefix := fmt.Sprintf("%s: #%d", prefix, i)
		gotAddr := addrs[i]
		wantAddr := expectedAddrMap[gotAddr.Address().String()]
		if !testAddress(tc, prefix, gotAddr, wantAddr) {
			return false
		}
		delete(expectedAddrMap, gotAddr.Address().String())
	}

	if len(expectedAddrMap) != 0 {
		tc.t.Errorf("%s: unexpected addresses -- got %d, want %d", prefix,
			len(expectedAddrMap), 0)
		return false
	}

	return true
}

//testmanagerapi测试由manager api提供的函数以及
//ManagedAddress、ManagedSubKeyAddress和ManagedScriptAddress
//接口。
func testManagerAPI(tc *testContext) {
	testLocking(tc)
	testExternalAddresses(tc)
	testInternalAddresses(tc)
	testImportPrivateKey(tc)
	testImportScript(tc)
	testMarkUsed(tc)
	testChangePassphrase(tc)

//重置默认帐户
	tc.account = 0
	testNewAccount(tc)
	testLookupAccount(tc)
	testForEachAccount(tc)
	testForEachAccountAddress(tc)

//重命名帐户1“帐户创建”
	tc.account = 1
	testRenameAccount(tc)
}

//testwatchingOnly测试只监视地址的各个方面
//
//
func testWatchingOnly(tc *testContext) bool {
//制作当前数据库的副本，以便将副本转换为
//只看。
	woMgrName := "mgrtestwo.bin"
	_ = os.Remove(woMgrName)
	fi, err := os.OpenFile(woMgrName, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		tc.t.Errorf("%v", err)
		return false
	}
	if err := tc.db.Copy(fi); err != nil {
		fi.Close()
		tc.t.Errorf("%v", err)
		return false
	}
	fi.Close()
	defer os.Remove(woMgrName)

//打开新的数据库副本并获取地址管理器命名空间。
	db, err := walletdb.Open("bdb", woMgrName)
	if err != nil {
		tc.t.Errorf("openDbNamespace: unexpected error: %v", err)
		return false
	}
	defer db.Close()

//使用命名空间打开管理器，并将其转换为仅监视。
	var mgr *Manager
	err = walletdb.View(db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		var err error
		mgr, err = Open(ns, pubPassphrase, &chaincfg.MainNetParams)
		return err
	})
	if err != nil {
		tc.t.Errorf("%v", err)
		return false
	}
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return mgr.ConvertToWatchingOnly(ns)
	})
	if err != nil {
		tc.t.Errorf("%v", err)
		return false
	}

//对转换后的管理器和
//关闭它。我们还将从
//经理才能使用。
	scopedMgr, err := mgr.FetchScopedKeyManager(KeyScopeBIP0044)
	if err != nil {
		tc.t.Errorf("unable to fetch bip 44 scope %v", err)
		return false
	}
	testManagerAPI(&testContext{
		t:            tc.t,
		db:           db,
		rootManager:  mgr,
		manager:      scopedMgr,
		account:      0,
		create:       false,
		watchingOnly: true,
	})
	mgr.Close()

//打开仅监视管理器并再次运行所有测试。
	err = walletdb.View(db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		var err error
		mgr, err = Open(ns, pubPassphrase, &chaincfg.MainNetParams)
		return err
	})
	if err != nil {
		tc.t.Errorf("Open Watching-Only: unexpected error: %v", err)
		return false
	}
	defer mgr.Close()

	scopedMgr, err = mgr.FetchScopedKeyManager(KeyScopeBIP0044)
	if err != nil {
		tc.t.Errorf("unable to fetch bip 44 scope %v", err)
		return false
	}

	testManagerAPI(&testContext{
		t:            tc.t,
		db:           db,
		rootManager:  mgr,
		manager:      scopedMgr,
		account:      0,
		create:       false,
		watchingOnly: true,
	})

	return true
}

//testsync测试设置管理器同步状态的各个方面。
func testSync(tc *testContext) bool {
//确保将管理器同步到零将导致“同步到”状态
//是最早的块体（本例中为创世块体）。
	err := walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return tc.rootManager.SetSyncedTo(ns, nil)
	})
	if err != nil {
		tc.t.Errorf("SetSyncedTo unexpected err on nil: %v", err)
		return false
	}
	blockStamp := BlockStamp{
		Height: 0,
		Hash:   *chaincfg.MainNetParams.GenesisHash,
	}
	gotBlockStamp := tc.rootManager.SyncedTo()
	if gotBlockStamp != blockStamp {
		tc.t.Errorf("SyncedTo unexpected block stamp on nil -- "+
			"got %v, want %v", gotBlockStamp, blockStamp)
		return false
	}

//如果我们更新到新的更新的块时间戳，则
//检索它应该作为最著名的状态返回。
	latestHash, err := chainhash.NewHash(seed)
	if err != nil {
		tc.t.Errorf("%v", err)
		return false
	}
	blockStamp = BlockStamp{
		Height:    1,
		Hash:      *latestHash,
		Timestamp: time.Unix(1234, 0),
	}
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return tc.rootManager.SetSyncedTo(ns, &blockStamp)
	})
	if err != nil {
		tc.t.Errorf("SetSyncedTo unexpected err on nil: %v", err)
		return false
	}
	gotBlockStamp = tc.rootManager.SyncedTo()
	if gotBlockStamp != blockStamp {
		tc.t.Errorf("SyncedTo unexpected block stamp on nil -- "+
			"got %v, want %v", gotBlockStamp, blockStamp)
		return false
	}

	return true
}

//testmanager对地址管理器API执行一整套测试。
//它使用测试上下文，因为地址管理器是持久的，并且
//许多测试都涉及到特定的状态。
func TestManager(t *testing.T) {
	t.Parallel()

	teardown, db := emptyDB(t)
	defer teardown()

//打开不存在的管理器以确保预期错误为
//返回。
	err := walletdb.View(db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		_, err := Open(ns, pubPassphrase, &chaincfg.MainNetParams)
		return err
	})
	if !checkManagerError(t, "Open non-existant", err, ErrNoExist) {
		return
	}

//创建新经理。
	var mgr *Manager
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns, err := tx.CreateTopLevelBucket(waddrmgrNamespaceKey)
		if err != nil {
			return err
		}
		err = Create(
			ns, seed, pubPassphrase, privPassphrase,
			&chaincfg.MainNetParams, fastScrypt, time.Time{},
		)
		if err != nil {
			return err
		}
		mgr, err = Open(ns, pubPassphrase, &chaincfg.MainNetParams)
		return err
	})
	if err != nil {
		t.Errorf("Create/Open: unexpected error: %v", err)
		return
	}

//注意：这里不使用延迟关闭，因为部分测试是
//显式关闭管理器，然后打开现有的管理器。

//尝试再次创建管理器以确保预期错误为
//返回。
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return Create(
			ns, seed, pubPassphrase, privPassphrase,
			&chaincfg.MainNetParams, fastScrypt, time.Time{},
		)
	})
	if !checkManagerError(t, "Create existing", err, ErrAlreadyExists) {
		mgr.Close()
		return
	}

//在创建模式下运行所有Manager API测试并关闭
//经理完成后
	scopedMgr, err := mgr.FetchScopedKeyManager(KeyScopeBIP0044)
	if err != nil {
		t.Fatalf("unable to fetch default scope: %v", err)
	}
	testManagerAPI(&testContext{
		t:            t,
		db:           db,
		manager:      scopedMgr,
		rootManager:  mgr,
		account:      0,
		create:       true,
		watchingOnly: false,
	})
	mgr.Close()

//打开管理器并以打开模式再次运行所有测试，其中
//避免像创建模式测试那样重新插入新地址。
	err = walletdb.View(db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		var err error
		mgr, err = Open(ns, pubPassphrase, &chaincfg.MainNetParams)
		return err
	})
	if err != nil {
		t.Errorf("Open: unexpected error: %v", err)
		return
	}
	defer mgr.Close()

	scopedMgr, err = mgr.FetchScopedKeyManager(KeyScopeBIP0044)
	if err != nil {
		t.Fatalf("unable to fetch default scope: %v", err)
	}
	tc := &testContext{
		t:            t,
		db:           db,
		manager:      scopedMgr,
		rootManager:  mgr,
		account:      0,
		create:       false,
		watchingOnly: false,
	}
	testManagerAPI(tc)

//现在地址管理器已经在
//创建并打开模式，测试仅监视版本。
	testWatchingOnly(tc)

//确保管理器同步状态功能按预期工作。
	testSync(tc)

//解锁管理器，使其可以在解锁时关闭，以确保
//它毫无问题地工作。
	err = walletdb.View(db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		return mgr.Unlock(ns, privPassphrase)
	})
	if err != nil {
		t.Errorf("Unlock: unexpected error: %v", err)
	}
}

//testmanagerincorrectversion确保无法访问管理器
//如果其版本与最新版本不匹配。
func TestManagerHigherVersion(t *testing.T) {
	t.Parallel()

	teardown, db, _ := setupManager(t)
	defer teardown()

//我们将把经理的版本更新为比最新版本高一个。
	latestVersion := getLatestVersion()
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		if ns == nil {
			return errors.New("top-level namespace does not exist")
		}
		return putManagerVersion(ns, latestVersion+1)
	})
	if err != nil {
		t.Fatalf("unable to update manager version %v", err)
	}

//然后，在尝试在不执行升级的情况下打开它时，我们
//应该会看到错误errUpgrade。
	err = walletdb.View(db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		_, err := Open(ns, pubPassphrase, &chaincfg.MainNetParams)
		return err
	})
	if !checkManagerError(t, "Upgrade needed", err, ErrUpgrade) {
		t.Fatalf("expected error ErrUpgrade, got %v", err)
	}

//我们还将更新它，使其比最新的低一个。
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		if ns == nil {
			return errors.New("top-level namespace does not exist")
		}
		return putManagerVersion(ns, latestVersion-1)
	})
	if err != nil {
		t.Fatalf("unable to update manager version %v", err)
	}

//最后，尝试在不执行升级的情况下打开它
//最新版本，我们也应该看到错误
//错误升级。
	err = walletdb.View(db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		_, err := Open(ns, pubPassphrase, &chaincfg.MainNetParams)
		return err
	})
	if !checkManagerError(t, "Upgrade needed", err, ErrUpgrade) {
		t.Fatalf("expected error ErrUpgrade, got %v", err)
	}
}

//TestEncryptDecryptErrors确保在加密和
//解密数据返回预期的错误。
func TestEncryptDecryptErrors(t *testing.T) {
	t.Parallel()

	teardown, db, mgr := setupManager(t)
	defer teardown()

	invalidKeyType := CryptoKeyType(0xff)
	if _, err := mgr.Encrypt(invalidKeyType, []byte{}); err == nil {
		t.Fatalf("Encrypt accepted an invalid key type!")
	}

	if _, err := mgr.Decrypt(invalidKeyType, []byte{}); err == nil {
		t.Fatalf("Encrypt accepted an invalid key type!")
	}

	if !mgr.IsLocked() {
		t.Fatal("Manager should be locked at this point.")
	}

	var err error
//现在，MGR被锁定，并用private加密/解密
//钥匙应该失效。
	_, err = mgr.Encrypt(CKTPrivate, []byte{})
	checkManagerError(t, "encryption with private key fails when manager is locked",
		err, ErrLocked)

	_, err = mgr.Decrypt(CKTPrivate, []byte{})
	checkManagerError(t, "decryption with private key fails when manager is locked",
		err, ErrLocked)

//为这些测试解锁管理器
	err = walletdb.View(db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		return mgr.Unlock(ns, privPassphrase)
	})
	if err != nil {
		t.Fatal("Attempted to unlock the manager, but failed:", err)
	}

//确保在加密和解密中覆盖errCrypto错误路径。
//我们将使用一个模拟的私钥，在运行这些密钥时会失败
//方法。
	mgr.cryptoKeyPriv = &failingCryptoKey{}

	_, err = mgr.Encrypt(CKTPrivate, []byte{})
	checkManagerError(t, "failed encryption", err, ErrCrypto)

	_, err = mgr.Decrypt(CKTPrivate, []byte{})
	checkManagerError(t, "failed decryption", err, ErrCrypto)
}

//testencryptdecrypt确保使用
//各种加密密钥类型按预期工作。
func TestEncryptDecrypt(t *testing.T) {
	t.Parallel()

	teardown, db, mgr := setupManager(t)
	defer teardown()

	plainText := []byte("this is a plaintext")

//确保地址管理器已解锁
	err := walletdb.View(db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		return mgr.Unlock(ns, privPassphrase)
	})
	if err != nil {
		t.Fatal("Attempted to unlock the manager, but failed:", err)
	}

	keyTypes := []CryptoKeyType{
		CKTPublic,
		CKTPrivate,
		CKTScript,
	}

	for _, keyType := range keyTypes {
		cipherText, err := mgr.Encrypt(keyType, plainText)
		if err != nil {
			t.Fatalf("Failed to encrypt plaintext: %v", err)
		}

		decryptedCipherText, err := mgr.Decrypt(keyType, cipherText)
		if err != nil {
			t.Fatalf("Failed to decrypt plaintext: %v", err)
		}

		if !reflect.DeepEqual(decryptedCipherText, plainText) {
			t.Fatal("Got:", decryptedCipherText, ", want:", plainText)
		}
	}
}

//TestScopedKeyManagerManagement调用方能够正确进行的测试
//在默认设置之外创建、检索和使用新的作用域管理器
//创建的范围。
func TestScopedKeyManagerManagement(t *testing.T) {
	t.Parallel()

	teardown, db := emptyDB(t)
	defer teardown()

//我们将通过创建一个新的根管理器来开始测试
//用于测试期间。
	var mgr *Manager
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns, err := tx.CreateTopLevelBucket(waddrmgrNamespaceKey)
		if err != nil {
			return err
		}
		err = Create(
			ns, seed, pubPassphrase, privPassphrase,
			&chaincfg.MainNetParams, fastScrypt, time.Time{},
		)
		if err != nil {
			return err
		}

		mgr, err = Open(ns, pubPassphrase, &chaincfg.MainNetParams)
		if err != nil {
			return err
		}

		return mgr.Unlock(ns, privPassphrase)
	})
	if err != nil {
		t.Fatalf("create/open: unexpected error: %v", err)
	}

//所有默认作用域都应该已创建并加载到
//初始打开时的内存。
	for _, scope := range DefaultKeyScopes {
		_, err := mgr.FetchScopedKeyManager(scope)
		if err != nil {
			t.Fatalf("unable to fetch scope %v: %v", scope, err)
		}
	}

//接下来，确保如果我们为
//每个默认范围类型，然后根据
//他们的图式。
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)

		for _, scope := range DefaultKeyScopes {
			sMgr, err := mgr.FetchScopedKeyManager(scope)
			if err != nil {
				t.Fatalf("unable to fetch scope %v: %v", scope, err)
			}

			externalAddr, err := sMgr.NextExternalAddresses(
				ns, DefaultAccountNum, 1,
			)
			if err != nil {
				t.Fatalf("unable to derive external addr: %v", err)
			}

//外部地址应符合规定
//此作用域密钥管理器的addr架构。
			if externalAddr[0].AddrType() != ScopeAddrMap[scope].ExternalAddrType {
				t.Fatalf("addr type mismatch: expected %v, got %v",
					externalAddr[0].AddrType(),
					ScopeAddrMap[scope].ExternalAddrType)
			}

			internalAddr, err := sMgr.NextInternalAddresses(
				ns, DefaultAccountNum, 1,
			)
			if err != nil {
				t.Fatalf("unable to derive internal addr: %v", err)
			}

//同样，内部地址应该与
//为此作用域密钥管理器指定的addr架构。
			if internalAddr[0].AddrType() != ScopeAddrMap[scope].InternalAddrType {
				t.Fatalf("addr type mismatch: expected %v, got %v",
					internalAddr[0].AddrType(),
					ScopeAddrMap[scope].InternalAddrType)
			}
		}

		return err
	})
	if err != nil {
		t.Fatalf("unable to read db: %v", err)
	}

//既然管理器已打开，我们将创建一个“测试”范围，我们将
//在剩下的测试中使用。
	testScope := KeyScope{
		Purpose: 99,
		Coin:    0,
	}
	addrSchema := ScopeAddrSchema{
		ExternalAddrType: NestedWitnessPubKey,
		InternalAddrType: WitnessPubKey,
	}
	var scopedMgr *ScopedKeyManager
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)

		scopedMgr, err = mgr.NewScopedKeyManager(ns, testScope, addrSchema)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		t.Fatalf("unable to read db: %v", err)
	}

//经理刚成立，我们应该可以在
//根管理器。
	if _, err := mgr.FetchScopedKeyManager(testScope); err != nil {
		t.Fatalf("attempt to read created mgr failed: %v", err)
	}

	var externalAddr, internalAddr []ManagedAddress
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)

//我们现在将创建一个新的外部地址，以确保
//检索正确的类型。
		externalAddr, err = scopedMgr.NextExternalAddresses(
			ns, DefaultAccountNum, 1,
		)
		if err != nil {
			t.Fatalf("unable to derive external addr: %v", err)
		}

		internalAddr, err = scopedMgr.NextInternalAddresses(
			ns, DefaultAccountNum, 1,
		)
		if err != nil {
			t.Fatalf("unable to derive internal addr: %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("open: unexpected error: %v", err)
	}

//确保地址类型与预期匹配。
	if externalAddr[0].AddrType() != NestedWitnessPubKey {
		t.Fatalf("addr type mismatch: expected %v, got %v",
			NestedWitnessPubKey, externalAddr[0].AddrType())
	}
	_, ok := externalAddr[0].Address().(*btcutil.AddressScriptHash)
	if !ok {
		t.Fatalf("wrong type: %T", externalAddr[0].Address())
	}

//我们还将创建一个内部地址并确保
//正确搭配。
	if internalAddr[0].AddrType() != WitnessPubKey {
		t.Fatalf("addr type mismatch: expected %v, got %v",
			WitnessPubKey, internalAddr[0].AddrType())
	}
	_, ok = internalAddr[0].Address().(*btcutil.AddressWitnessPubKeyHash)
	if !ok {
		t.Fatalf("wrong type: %T", externalAddr[0].Address())
	}

//现在，我们将通过关闭然后重新启动来模拟重新启动
//经理。
	mgr.Close()
	err = walletdb.View(db, func(tx walletdb.ReadTx) error {
		ns := tx.ReadBucket(waddrmgrNamespaceKey)
		var err error
		mgr, err = Open(ns, pubPassphrase, &chaincfg.MainNetParams)
		if err != nil {
			return err
		}

		return mgr.Unlock(ns, privPassphrase)
	})
	if err != nil {
		t.Fatalf("open: unexpected error: %v", err)
	}
	defer mgr.Close()

//我们应该能够找到新的范围管理器
//创建。
	scopedMgr, err = mgr.FetchScopedKeyManager(testScope)
	if err != nil {
		t.Fatalf("attempt to read created mgr failed: %v", err)
	}

//如果我们获取最后生成的外部地址，它应该映射
//精确到我们刚生成的地址。
	var lastAddr ManagedAddress
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)

		lastAddr, err = scopedMgr.LastExternalAddress(
			ns, DefaultAccountNum,
		)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		t.Fatalf("open: unexpected error: %v", err)
	}
	if !bytes.Equal(lastAddr.AddrHash(), externalAddr[0].AddrHash()) {
		t.Fatalf("mismatch addr hashes: expected %x, got %x",
			externalAddr[0].AddrHash(), lastAddr.AddrHash())
	}

//重新启动后，应重新加载所有默认作用域。
	for _, scope := range DefaultKeyScopes {
		_, err := mgr.FetchScopedKeyManager(scope)
		if err != nil {
			t.Fatalf("unable to fetch scope %v: %v", scope, err)
		}
	}

//最后，如果我们试图查询最后一个根管理器
//地址，应能找到私钥等。
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)

		_, err := mgr.Address(ns, lastAddr.Address())
		if err != nil {
			return fmt.Errorf("unable to find addr: %v", err)
		}

		err = mgr.MarkUsed(ns, lastAddr.Address())
		if err != nil {
			return fmt.Errorf("unable to mark addr as "+
				"used: %v", err)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("unable to find addr: %v", err)
	}
}

//testroothdkeyneutring调用方无法创建新范围的测试
//一旦从数据库中删除了根hd密钥，管理器就可以执行此操作。
func TestRootHDKeyNeutering(t *testing.T) {
	t.Parallel()

	teardown, db := emptyDB(t)
	defer teardown()

//我们将通过创建一个新的根管理器来开始测试
//用于测试期间。
	var mgr *Manager
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns, err := tx.CreateTopLevelBucket(waddrmgrNamespaceKey)
		if err != nil {
			return err
		}
		err = Create(
			ns, seed, pubPassphrase, privPassphrase,
			&chaincfg.MainNetParams, fastScrypt, time.Time{},
		)
		if err != nil {
			return err
		}

		mgr, err = Open(ns, pubPassphrase, &chaincfg.MainNetParams)
		if err != nil {
			return err
		}

		return mgr.Unlock(ns, privPassphrase)
	})
	if err != nil {
		t.Fatalf("create/open: unexpected error: %v", err)
	}
	defer mgr.Close()

//打开根管理器后，我们现在将创建一个新的作用域管理器
//用于此测试。
	testScope := KeyScope{
		Purpose: 99,
		Coin:    0,
	}
	addrSchema := ScopeAddrSchema{
		ExternalAddrType: NestedWitnessPubKey,
		InternalAddrType: WitnessPubKey,
	}
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)

		_, err := mgr.NewScopedKeyManager(ns, testScope, addrSchema)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		t.Fatalf("unable to read db: %v", err)
	}

//创建了管理器后，我们现在将对根hd私钥进行中性化。
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)

		return mgr.NeuterRootKey(ns)
	})
	if err != nil {
		t.Fatalf("unable to read db: %v", err)
	}

//如果我们试图创建*另一个*作用域，这应该会失败，因为根
//密钥不再在数据库中。
	testScope = KeyScope{
		Purpose: 100,
		Coin:    0,
	}
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)

		_, err := mgr.NewScopedKeyManager(ns, testScope, addrSchema)
		if err != nil {
			return err
		}

		return nil
	})
	if err == nil {
		t.Fatalf("new scoped manager creation should have failed")
	}
}

//调用方能够正确创建和使用的测试newrawaccount
//只使用帐号而不是字符串创建的原始帐户
//最终映射到一个帐号。
func TestNewRawAccount(t *testing.T) {
	t.Parallel()

	teardown, db := emptyDB(t)
	defer teardown()

//我们将通过创建一个新的根管理器来开始测试
//用于测试期间。
	var mgr *Manager
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns, err := tx.CreateTopLevelBucket(waddrmgrNamespaceKey)
		if err != nil {
			return err
		}
		err = Create(
			ns, seed, pubPassphrase, privPassphrase,
			&chaincfg.MainNetParams, fastScrypt, time.Time{},
		)
		if err != nil {
			return err
		}

		mgr, err = Open(ns, pubPassphrase, &chaincfg.MainNetParams)
		if err != nil {
			return err
		}

		return mgr.Unlock(ns, privPassphrase)
	})
	if err != nil {
		t.Fatalf("create/open: unexpected error: %v", err)
	}
	defer mgr.Close()

//现在我们已经创建了管理器，我们将获取其中一个默认值
//此测试中使用的范围。
	scopedMgr, err := mgr.FetchScopedKeyManager(KeyScopeBIP0084)
	if err != nil {
		t.Fatalf("unable to fetch scope %v: %v", KeyScopeBIP0084, err)
	}

//检索到作用域管理器后，我们将尝试创建新的原始
//按编号记帐。
	const accountNum = 1000
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		return scopedMgr.NewRawAccount(ns, accountNum)
	})
	if err != nil {
		t.Fatalf("unable to create new account: %v", err)
	}

//创建帐户后，我们应该能够派生新地址
//从帐户。
	var accountAddrNext ManagedAddress
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)

		addrs, err := scopedMgr.NextExternalAddresses(
			ns, accountNum, 1,
		)
		if err != nil {
			return err
		}

		accountAddrNext = addrs[0]
		return nil
	})
	if err != nil {
		t.Fatalf("unable to create addr: %v", err)
	}

//此外，我们应该能够手动派生特定的目标
//钥匙。
	var accountTargetAddr ManagedAddress
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(waddrmgrNamespaceKey)

		keyPath := DerivationPath{
			Account: accountNum,
			Branch:  0,
			Index:   0,
		}
		accountTargetAddr, err = scopedMgr.DeriveFromKeyPath(
			ns, keyPath,
		)
		return err
	})
	if err != nil {
		t.Fatalf("unable to derive addr: %v", err)
	}

//我们刚刚得到的两个键应该完全匹配。
	if accountAddrNext.AddrType() != accountTargetAddr.AddrType() {
		t.Fatalf("wrong addr type: %v vs %v",
			accountAddrNext.AddrType(), accountTargetAddr.AddrType())
	}
	if !bytes.Equal(accountAddrNext.AddrHash(), accountTargetAddr.AddrHash()) {
		t.Fatalf("wrong pubkey hash: %x vs %x", accountAddrNext.AddrHash(),
			accountTargetAddr.AddrHash())
	}
}
