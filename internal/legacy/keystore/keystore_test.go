
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package keystore

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/davecgh/go-spew/spew"
)

const dummyDir = ""

var tstNetParams = &chaincfg.MainNetParams

func makeBS(height int32) *BlockStamp {
	return &BlockStamp{
		Hash:   new(chainhash.Hash),
		Height: height,
	}
}

func TestBtcAddressSerializer(t *testing.T) {
	fakeWallet := &Store{net: (*netParams)(tstNetParams)}
	kdfp := &kdfParameters{
		mem:   1024,
		nIter: 5,
	}
	if _, err := rand.Read(kdfp.salt[:]); err != nil {
		t.Error(err.Error())
		return
	}
	key := kdf([]byte("banana"), kdfp)
	privKey := make([]byte, 32)
	if _, err := rand.Read(privKey); err != nil {
		t.Error(err.Error())
		return
	}
	addr, err := newBtcAddress(fakeWallet, privKey, nil,
		makeBS(0), true)
	if err != nil {
		t.Error(err.Error())
		return
	}
	err = addr.encrypt(key)
	if err != nil {
		t.Error(err.Error())
		return
	}

	buf := new(bytes.Buffer)

	if _, err := addr.WriteTo(buf); err != nil {
		t.Error(err.Error())
		return
	}

	var readAddr btcAddress
	readAddr.store = fakeWallet
	_, err = readAddr.ReadFrom(buf)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if _, err = readAddr.unlock(key); err != nil {
		t.Error(err.Error())
		return
	}

	if !reflect.DeepEqual(addr, &readAddr) {
		t.Error("Original and read btcAddress differ.")
	}
}

func TestScriptAddressSerializer(t *testing.T) {
	fakeWallet := &Store{net: (*netParams)(tstNetParams)}
	script := []byte{txscript.OP_TRUE, txscript.OP_DUP,
		txscript.OP_DROP}
	addr, err := newScriptAddress(fakeWallet, script, makeBS(0))
	if err != nil {
		t.Error(err.Error())
		return
	}

	buf := new(bytes.Buffer)

	if _, err := addr.WriteTo(buf); err != nil {
		t.Error(err.Error())
		return
	}

	var readAddr scriptAddress
	readAddr.store = fakeWallet
	_, err = readAddr.ReadFrom(buf)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if !reflect.DeepEqual(addr, &readAddr) {
		t.Error("Original and read btcAddress differ.")
	}
}

func TestWalletCreationSerialization(t *testing.T) {
	createdAt := makeBS(0)
	w1, err := New(dummyDir, "A wallet for testing.",
		[]byte("banana"), tstNetParams, createdAt)
	if err != nil {
		t.Error("Error creating new wallet: " + err.Error())
		return
	}

	buf := new(bytes.Buffer)

	if _, err := w1.WriteTo(buf); err != nil {
		t.Error("Error writing new wallet: " + err.Error())
		return
	}

	w2 := new(Store)
	_, err = w2.ReadFrom(buf)
	if err != nil {
		t.Error("Error reading newly written wallet: " + err.Error())
		return
	}

	w1.Lock()
	w2.Lock()

	if err = w1.Unlock([]byte("banana")); err != nil {
		t.Error("Decrypting original wallet failed: " + err.Error())
		return
	}

	if err = w2.Unlock([]byte("banana")); err != nil {
		t.Error("Decrypting newly read wallet failed: " + err.Error())
		return
	}

//如果！反射深度（w1，w2）
//t.error（“在钱包中创建和读取不匹配。”）
//垃圾场（w1，w2）
//返回
//}
}

func TestChaining(t *testing.T) {
	tests := []struct {
		name                       string
		cc                         []byte
		origPrivateKey             []byte
		nextPrivateKeyUncompressed []byte
		nextPrivateKeyCompressed   []byte
		nextPublicKeyUncompressed  []byte
		nextPublicKeyCompressed    []byte
	}{
		{
			name:           "chaintest 1",
			cc:             []byte("3318959fff419ab8b556facb3c429a86"),
			origPrivateKey: []byte("5ffc975976eaaa1f7b179f384ebbc053"),
			nextPrivateKeyUncompressed: []byte{
				0xd3, 0xfe, 0x2e, 0x96, 0x44, 0x12, 0x2d, 0xaa,
				0x80, 0x8e, 0x36, 0x17, 0xb5, 0x9f, 0x8c, 0xd2,
				0x72, 0x8c, 0xaf, 0xf1, 0xdb, 0xd6, 0x4a, 0x92,
				0xd7, 0xc7, 0xee, 0x2b, 0x56, 0x34, 0xe2, 0x87,
			},
			nextPrivateKeyCompressed: []byte{
				0x08, 0x56, 0x7a, 0x1b, 0x89, 0x56, 0x2e, 0xfa,
				0xb4, 0x02, 0x59, 0x69, 0x10, 0xc3, 0x60, 0x1f,
				0x34, 0xf0, 0x55, 0x02, 0x8a, 0xbf, 0x37, 0xf5,
				0x22, 0x80, 0x9f, 0xd2, 0xe5, 0x42, 0x5b, 0x2d,
			},
			nextPublicKeyUncompressed: []byte{
				0x04, 0xdd, 0x70, 0x31, 0xa5, 0xf9, 0x06, 0x70,
				0xd3, 0x9a, 0x24, 0x5b, 0xd5, 0x73, 0xdd, 0xb6,
				0x15, 0x81, 0x0b, 0x78, 0x19, 0xbc, 0xc8, 0x26,
				0xc9, 0x16, 0x86, 0x73, 0xae, 0xe4, 0xc0, 0xed,
				0x39, 0x81, 0xb4, 0x86, 0x2d, 0x19, 0x8c, 0x67,
				0x9c, 0x93, 0x99, 0xf6, 0xd2, 0x3f, 0xd1, 0x53,
				0x9e, 0xed, 0xbd, 0x07, 0xd6, 0x4f, 0xa9, 0x81,
				0x61, 0x85, 0x46, 0x84, 0xb1, 0xa0, 0xed, 0xbc,
				0xa7,
			},
			nextPublicKeyCompressed: []byte{
				0x02, 0x2c, 0x48, 0x73, 0x37, 0x35, 0x74, 0x7f,
				0x05, 0x58, 0xc1, 0x4e, 0x0d, 0x18, 0xc2, 0xbf,
				0xcc, 0x83, 0xa2, 0x4d, 0x64, 0xab, 0xba, 0xea,
				0xeb, 0x4c, 0xcd, 0x4c, 0x0c, 0x21, 0xc4, 0x30,
				0x0f,
			},
		},
	}

	for _, test := range tests {
//为原始文件创建未压缩和压缩的公钥
//私钥。
		origPubUncompressed := pubkeyFromPrivkey(test.origPrivateKey, false)
		origPubCompressed := pubkeyFromPrivkey(test.origPrivateKey, true)

//创建下一个链接的私钥，从两个未压缩的
//和压缩的公钥。
		nextPrivUncompressed, err := chainedPrivKey(test.origPrivateKey,
			origPubUncompressed, test.cc)
		if err != nil {
			t.Errorf("%s: Uncompressed chainedPrivKey failed: %v", test.name, err)
			return
		}
		nextPrivCompressed, err := chainedPrivKey(test.origPrivateKey,
			origPubCompressed, test.cc)
		if err != nil {
			t.Errorf("%s: Compressed chainedPrivKey failed: %v", test.name, err)
			return
		}

//验证新的私钥是否与预期值匹配
//在测试用例中。
		if !bytes.Equal(nextPrivUncompressed, test.nextPrivateKeyUncompressed) {
			t.Errorf("%s: Next private key (from uncompressed pubkey) does not match expected.\nGot: %s\nExpected: %s",
				test.name, spew.Sdump(nextPrivUncompressed), spew.Sdump(test.nextPrivateKeyUncompressed))
			return
		}
		if !bytes.Equal(nextPrivCompressed, test.nextPrivateKeyCompressed) {
			t.Errorf("%s: Next private key (from compressed pubkey) does not match expected.\nGot: %s\nExpected: %s",
				test.name, spew.Sdump(nextPrivCompressed), spew.Sdump(test.nextPrivateKeyCompressed))
			return
		}

//创建从下一个私钥生成的下一个公钥。
		nextPubUncompressedFromPriv := pubkeyFromPrivkey(nextPrivUncompressed, false)
		nextPubCompressedFromPriv := pubkeyFromPrivkey(nextPrivCompressed, true)

//通过直接链接到原始pubkeys创建下一个pubkeys
//公钥（不使用原始的私钥）。
		nextPubUncompressedFromPub, err := chainedPubKey(origPubUncompressed, test.cc)
		if err != nil {
			t.Errorf("%s: Uncompressed chainedPubKey failed: %v", test.name, err)
			return
		}
		nextPubCompressedFromPub, err := chainedPubKey(origPubCompressed, test.cc)
		if err != nil {
			t.Errorf("%s: Compressed chainedPubKey failed: %v", test.name, err)
			return
		}

//公钥（用于生成比特币地址）必须匹配。
		if !bytes.Equal(nextPubUncompressedFromPriv, nextPubUncompressedFromPub) {
			t.Errorf("%s: Uncompressed public keys do not match.", test.name)
		}
		if !bytes.Equal(nextPubCompressedFromPriv, nextPubCompressedFromPub) {
			t.Errorf("%s: Compressed public keys do not match.", test.name)
		}

//验证所有生成的公钥是否与预期的匹配
//测试用例中的值。
		if !bytes.Equal(nextPubUncompressedFromPub, test.nextPublicKeyUncompressed) {
			t.Errorf("%s: Next uncompressed public keys do not match expected value.\nGot: %s\nExpected: %s",
				test.name, spew.Sdump(nextPubUncompressedFromPub), spew.Sdump(test.nextPublicKeyUncompressed))
			return
		}
		if !bytes.Equal(nextPubCompressedFromPub, test.nextPublicKeyCompressed) {
			t.Errorf("%s: Next compressed public keys do not match expected value.\nGot: %s\nExpected: %s",
				test.name, spew.Sdump(nextPubCompressedFromPub), spew.Sdump(test.nextPublicKeyCompressed))
			return
		}

//用下一个私钥对数据签名，并用验证签名
//下一个发布键。
		pubkeyUncompressed, err := btcec.ParsePubKey(nextPubUncompressedFromPub, btcec.S256())
		if err != nil {
			t.Errorf("%s: Unable to parse next uncompressed pubkey: %v", test.name, err)
			return
		}
		pubkeyCompressed, err := btcec.ParsePubKey(nextPubCompressedFromPub, btcec.S256())
		if err != nil {
			t.Errorf("%s: Unable to parse next compressed pubkey: %v", test.name, err)
			return
		}
		privkeyUncompressed := &btcec.PrivateKey{
			PublicKey: *pubkeyUncompressed.ToECDSA(),
			D:         new(big.Int).SetBytes(nextPrivUncompressed),
		}
		privkeyCompressed := &btcec.PrivateKey{
			PublicKey: *pubkeyCompressed.ToECDSA(),
			D:         new(big.Int).SetBytes(nextPrivCompressed),
		}
		data := "String to sign."
		sig, err := privkeyUncompressed.Sign([]byte(data))
		if err != nil {
			t.Errorf("%s: Unable to sign data with next private key (chained from uncompressed pubkey): %v",
				test.name, err)
			return
		}
		ok := sig.Verify([]byte(data), privkeyUncompressed.PubKey())
		if !ok {
			t.Errorf("%s: btcec signature verification failed for next keypair (chained from uncompressed pubkey).",
				test.name)
			return
		}
		sig, err = privkeyCompressed.Sign([]byte(data))
		if err != nil {
			t.Errorf("%s: Unable to sign data with next private key (chained from compressed pubkey): %v",
				test.name, err)
			return
		}
		ok = sig.Verify([]byte(data), privkeyCompressed.PubKey())
		if !ok {
			t.Errorf("%s: btcec signature verification failed for next keypair (chained from compressed pubkey).",
				test.name)
			return
		}
	}
}

func TestWalletPubkeyChaining(t *testing.T) {
	w, err := New(dummyDir, "A wallet for testing.",
		[]byte("banana"), tstNetParams, makeBS(0))
	if err != nil {
		t.Error("Error creating new wallet: " + err.Error())
		return
	}
	if !w.IsLocked() {
		t.Error("New wallet is not locked.")
	}

//获取下一个链接地址。钱包是锁着的，所以这个会锁住
//关闭最后一个pubkey，而不是privkey。
	addrWithoutPrivkey, err := w.NextChainedAddress(makeBS(0))
	if err != nil {
		t.Errorf("Failed to extend address chain from pubkey: %v", err)
		return
	}

//查找地址信息。即使没有私人的帮助也能成功
//密钥可用。
	info, err := w.Address(addrWithoutPrivkey)
	if err != nil {
		t.Errorf("Failed to get info about address without private key: %v", err)
		return
	}

	pkinfo := info.(PubKeyAddress)
//清醒检查
	if !info.Compressed() {
		t.Errorf("Pubkey should be compressed.")
		return
	}
	if info.Imported() {
		t.Errorf("Should not be marked as imported.")
		return
	}

	pka := info.(PubKeyAddress)

//尝试查找它的私钥。这应该会失败。
	_, err = pka.PrivKey()
	if err == nil {
		t.Errorf("Incorrectly returned nil error for looking up private key for address without one saved.")
		return
	}

//反序列化w并序列化到新钱包中。剩下的支票
//在这个测试中，测试对象既有一个新的，也有一个“打开和关闭”
//钱包里有丢失的私人钥匙。
	serializedWallet := new(bytes.Buffer)
	_, err = w.WriteTo(serializedWallet)
	if err != nil {
		t.Errorf("Error writing wallet with missing private key: %v", err)
		return
	}
	w2 := new(Store)
	_, err = w2.ReadFrom(serializedWallet)
	if err != nil {
		t.Errorf("Error reading wallet with missing private key: %v", err)
		return
	}

//打开钱包。这将触发为创建私钥
//地址。
	if err = w.Unlock([]byte("banana")); err != nil {
		t.Errorf("Can't unlock original wallet: %v", err)
		return
	}
	if err = w2.Unlock([]byte("banana")); err != nil {
		t.Errorf("Can't unlock re-read wallet: %v", err)
		return
	}

//地址相同，变量名更好。
	addrWithPrivKey := addrWithoutPrivkey

//再次尝试私钥查找。现在应该可以使用私钥了。
	key1, err := pka.PrivKey()
	if err != nil {
		t.Errorf("Private key for original wallet was not created! %v", err)
		return
	}

	info2, err := w.Address(addrWithPrivKey)
	if err != nil {
		t.Errorf("no address in re-read wallet")
	}
	pka2 := info2.(PubKeyAddress)
	key2, err := pka2.PrivKey()
	if err != nil {
		t.Errorf("Private key for re-read wallet was not created! %v", err)
		return
	}

//两个钱包返回的钥匙必须匹配。
	if !reflect.DeepEqual(key1, key2) {
		t.Errorf("Private keys for address originally created without one mismtach between original and re-read wallet.")
		return
	}

//用私钥签名一些数据，然后用pubkey验证签名。
	hash := []byte("hash to sign")
	sig, err := key1.Sign(hash)
	if err != nil {
		t.Errorf("Unable to sign hash with the created private key: %v", err)
		return
	}
	pubKey := pkinfo.PubKey()
	ok := sig.Verify(hash, pubKey)
	if !ok {
		t.Errorf("btcec signature verification failed; address's pubkey mismatches the privkey.")
		return
	}

	nextAddr, err := w.NextChainedAddress(makeBS(0))
	if err != nil {
		t.Errorf("Unable to create next address after finding the privkey: %v", err)
		return
	}

	nextInfo, err := w.Address(nextAddr)
	if err != nil {
		t.Errorf("Couldn't get info about the next address in the chain: %v", err)
		return
	}
	nextPkInfo := nextInfo.(PubKeyAddress)
	nextKey, err := nextPkInfo.PrivKey()
	if err != nil {
		t.Errorf("Couldn't get private key for the next address in the chain: %v", err)
		return
	}

//在这里也做签名检查，这次是下一次
//不带私钥的地址。
	sig, err = nextKey.Sign(hash)
	if err != nil {
		t.Errorf("Unable to sign hash with the created private key: %v", err)
		return
	}
	pubKey = nextPkInfo.PubKey()
	ok = sig.Verify(hash, pubKey)
	if !ok {
		t.Errorf("btcec signature verification failed; next address's keypair does not match.")
		return
	}

//检查串行钱包是否正确地标记了“需要私人”
//“以后按键”标志。
	buf := new(bytes.Buffer)
	w2.WriteTo(buf)
	w2.ReadFrom(buf)
	err = w2.Unlock([]byte("banana"))
	if err != nil {
		t.Errorf("Unlock after serialize/deserialize failed: %v", err)
		return
	}
}

func TestWatchingWalletExport(t *testing.T) {
	createdAt := makeBS(0)
	w, err := New(dummyDir, "A wallet for testing.",
		[]byte("banana"), tstNetParams, createdAt)
	if err != nil {
		t.Error("Error creating new wallet: " + err.Error())
		return
	}

//在钱包中保留一组活动地址。
	activeAddrs := make(map[addressKey]struct{})

//添加根地址。
	activeAddrs[getAddressKey(w.LastChainedAddress())] = struct{}{}

//从W.
	ww, err := w.ExportWatchingWallet()
	if err != nil {
		t.Errorf("Could not create watching wallet: %v", err)
		return
	}

//验证钱包标志的正确性。
	if ww.flags.useEncryption {
		t.Errorf("Watching wallet marked as using encryption (but nothing to encrypt).")
		return
	}
	if !ww.flags.watchingOnly {
		t.Errorf("Wallet should be watching-only but is not marked so.")
		return
	}

//验证所有标志是否按预期设置。
	if ww.keyGenerator.flags.encrypted {
		t.Errorf("Watching root address should not be encrypted (nothing to encrypt)")
		return
	}
	if ww.keyGenerator.flags.hasPrivKey {
		t.Errorf("Watching root address marked as having a private key.")
		return
	}
	if !ww.keyGenerator.flags.hasPubKey {
		t.Errorf("Watching root address marked as missing a public key.")
		return
	}
	if ww.keyGenerator.flags.createPrivKeyNextUnlock {
		t.Errorf("Watching root address marked as needing a private key to be generated later.")
		return
	}
	for apkh, waddr := range ww.addrMap {
		switch addr := waddr.(type) {
		case *btcAddress:
			if addr.flags.encrypted {
				t.Errorf("Chained address should not be encrypted (nothing to encrypt)")
				return
			}
			if addr.flags.hasPrivKey {
				t.Errorf("Chained address marked as having a private key.")
				return
			}
			if !addr.flags.hasPubKey {
				t.Errorf("Chained address marked as missing a public key.")
				return
			}
			if addr.flags.createPrivKeyNextUnlock {
				t.Errorf("Chained address marked as needing a private key to be generated later.")
				return
			}
		case *scriptAddress:
			t.Errorf("Chained address was a script!")
			return
		default:
			t.Errorf("Chained address unknown type!")
			return
		}

		if _, ok := activeAddrs[apkh]; !ok {
			t.Errorf("Address from watching wallet not found in original wallet.")
			return
		}
		delete(activeAddrs, apkh)
	}
	if len(activeAddrs) != 0 {
		t.Errorf("%v address(es) were not exported to watching wallet.", len(activeAddrs))
		return
	}

//检查每个钱包创建的新地址是否匹配。这个
//原来的钱包是解锁的，所以地址链与私钥。
	if err := w.Unlock([]byte("banana")); err != nil {
		t.Errorf("Unlocking original wallet failed: %v", err)
	}

//测试看钱包比赛时的延长活动服
//手动请求原始钱包的地址。
	var newAddrs []btcutil.Address
	for i := 0; i < 10; i++ {
		addr, err := w.NextChainedAddress(createdAt)
		if err != nil {
			t.Errorf("Cannot get next chained address for original wallet: %v", err)
			return
		}
		newAddrs = append(newAddrs, addr)
	}
	newWWAddrs, err := ww.ExtendActiveAddresses(10)
	if err != nil {
		t.Errorf("Cannot extend active addresses for watching wallet: %v", err)
		return
	}
	for i := range newAddrs {
		if newAddrs[i].EncodeAddress() != newWWAddrs[i].EncodeAddress() {
			t.Errorf("Extended active addresses do not match manually requested addresses.")
			return
		}
	}

//手动测试原始钱包的扩展激活标题
//请求收看钱包的地址。
	newWWAddrs = nil
	for i := 0; i < 10; i++ {
		addr, err := ww.NextChainedAddress(createdAt)
		if err != nil {
			t.Errorf("Cannot get next chained address for watching wallet: %v", err)
			return
		}
		newWWAddrs = append(newWWAddrs, addr)
	}
	newAddrs, err = w.ExtendActiveAddresses(10)
	if err != nil {
		t.Errorf("Cannot extend active addresses for original wallet: %v", err)
		return
	}
	for i := range newAddrs {
		if newAddrs[i].EncodeAddress() != newWWAddrs[i].EncodeAddress() {
			t.Errorf("Extended active addresses do not match manually requested addresses.")
			return
		}
	}

//测试（反）看钱包系列。
	buf := new(bytes.Buffer)
	_, err = ww.WriteTo(buf)
	if err != nil {
		t.Errorf("Cannot write watching wallet: %v", err)
		return
	}
	ww2 := new(Store)
	_, err = ww2.ReadFrom(buf)
	if err != nil {
		t.Errorf("Cannot read watching wallet: %v", err)
		return
	}

//检查（反）序列监视钱包是否与导出钱包匹配。
	if !reflect.DeepEqual(ww, ww2) {
		t.Error("Exported and read-in watching wallets do not match.")
		return
	}

//确认无意义功能故障，错误正确。
	if err := ww.Lock(); err != ErrWatchingOnly {
		t.Errorf("Nonsensical func Lock returned no or incorrect error: %v", err)
		return
	}
	if err := ww.Unlock([]byte("banana")); err != ErrWatchingOnly {
		t.Errorf("Nonsensical func Unlock returned no or incorrect error: %v", err)
		return
	}
	generator, err := ww.Address(w.keyGenerator.Address())
	if err != nil {
		t.Errorf("generator isnt' present in wallet")
	}
	gpk := generator.(PubKeyAddress)
	if _, err := gpk.PrivKey(); err != ErrWatchingOnly {
		t.Errorf("Nonsensical func AddressKey returned no or incorrect error: %v", err)
		return
	}
	if _, err := ww.ExportWatchingWallet(); err != ErrWatchingOnly {
		t.Errorf("Nonsensical func ExportWatchingWallet returned no or incorrect error: %v", err)
		return
	}
	pk, _ := btcec.PrivKeyFromBytes(btcec.S256(), make([]byte, 32))
	wif, err := btcutil.NewWIF(pk, tstNetParams, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ww.ImportPrivateKey(wif, createdAt); err != ErrWatchingOnly {
		t.Errorf("Nonsensical func ImportPrivateKey returned no or incorrect error: %v", err)
		return
	}
}

func TestImportPrivateKey(t *testing.T) {
	createHeight := int32(100)
	createdAt := makeBS(createHeight)
	w, err := New(dummyDir, "A wallet for testing.",
		[]byte("banana"), tstNetParams, createdAt)
	if err != nil {
		t.Error("Error creating new wallet: " + err.Error())
		return
	}

	if err = w.Unlock([]byte("banana")); err != nil {
		t.Errorf("Can't unlock original wallet: %v", err)
		return
	}

	pk, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		t.Error("Error generating private key: " + err.Error())
		return
	}

//验证整个钱包的同步高度是否与
//应为CreateHeight。
	if _, h := w.SyncedTo(); h != createHeight {
		t.Errorf("Initial sync height %v does not match expected %v.", h, createHeight)
		return
	}

//输入PRIV密钥
	wif, err := btcutil.NewWIF((*btcec.PrivateKey)(pk), tstNetParams, false)
	if err != nil {
		t.Fatal(err)
	}
	importHeight := int32(50)
	importedAt := makeBS(importHeight)
	address, err := w.ImportPrivateKey(wif, importedAt)
	if err != nil {
		t.Error("importing private key: " + err.Error())
		return
	}

	addr, err := w.Address(address)
	if err != nil {
		t.Error("privkey just imported missing: " + err.Error())
		return
	}
	pka := addr.(PubKeyAddress)

//查找地址
	pk2, err := pka.PrivKey()
	if err != nil {
		t.Error("error looking up key: " + err.Error())
	}

	if !reflect.DeepEqual(pk, pk2) {
		t.Error("original and looked-up private keys do not match.")
		return
	}

//验证同步高度现在是否与（较小的）导入高度匹配。
	if _, h := w.SyncedTo(); h != importHeight {
		t.Errorf("After import sync height %v does not match expected %v.", h, importHeight)
		return
	}

//继续，去士气，检查那里。

//钱包测试（反）系列化。
	buf := new(bytes.Buffer)
	_, err = w.WriteTo(buf)
	if err != nil {
		t.Errorf("Cannot write wallet: %v", err)
		return
	}
	w2 := new(Store)
	_, err = w2.ReadFrom(buf)
	if err != nil {
		t.Errorf("Cannot read wallet: %v", err)
		return
	}

//验证重新初始化后的同步高度是否符合预期。
	if _, h := w2.SyncedTo(); h != importHeight {
		t.Errorf("After reserialization sync height %v does not match expected %v.", h, importHeight)
		return
	}

//将导入的地址标记为与中间某个块部分同步
//导入高度和链高度。
	partialHeight := (createHeight-importHeight)/2 + importHeight
	if err := w2.SetSyncStatus(address, PartialSync(partialHeight)); err != nil {
		t.Errorf("Cannot mark address partially synced: %v", err)
		return
	}
	if _, h := w2.SyncedTo(); h != partialHeight {
		t.Errorf("After address partial sync, sync height %v does not match expected %v.", h, partialHeight)
		return
	}

//使用部分同步测试序列化。
	buf.Reset()
	_, err = w2.WriteTo(buf)
	if err != nil {
		t.Errorf("Cannot write wallet: %v", err)
		return
	}
	w3 := new(Store)
	_, err = w3.ReadFrom(buf)
	if err != nil {
		t.Errorf("Cannot read wallet: %v", err)
		return
	}

//序列化后测试正确的部分高度。
	if _, h := w3.SyncedTo(); h != partialHeight {
		t.Errorf("After address partial sync and reserialization, sync height %v does not match expected %v.",
			h, partialHeight)
		return
	}

//将导入的地址标记为完全不同步，并验证同步高度现在是否为
//导入高度。
	if err := w3.SetSyncStatus(address, Unsynced(0)); err != nil {
		t.Errorf("Cannot mark address synced: %v", err)
		return
	}
	if _, h := w3.SyncedTo(); h != importHeight {
		t.Errorf("After address unsync, sync height %v does not match expected %v.", h, importHeight)
		return
	}

//将导入的地址标记为与最近看到的块同步，并验证
//同步高度现在等于最近的块（钱包上的块
//创造）。
	if err := w3.SetSyncStatus(address, FullSync{}); err != nil {
		t.Errorf("Cannot mark address synced: %v", err)
		return
	}
	if _, h := w3.SyncedTo(); h != createHeight {
		t.Errorf("After address sync, sync height %v does not match expected %v.", h, createHeight)
		return
	}

	if err = w3.Unlock([]byte("banana")); err != nil {
		t.Errorf("Can't unlock deserialised wallet: %v", err)
		return
	}

	addr3, err := w3.Address(address)
	if err != nil {
		t.Error("privkey in deserialised wallet missing : " +
			err.Error())
		return
	}
	pka3 := addr3.(PubKeyAddress)

//查找地址
	pk2, err = pka3.PrivKey()
	if err != nil {
		t.Error("error looking up key in deserialized wallet: " + err.Error())
	}

	if !reflect.DeepEqual(pk, pk2) {
		t.Error("original and deserialized private keys do not match.")
		return
	}

}

func TestImportScript(t *testing.T) {
	createHeight := int32(100)
	createdAt := makeBS(createHeight)
	w, err := New(dummyDir, "A wallet for testing.",
		[]byte("banana"), tstNetParams, createdAt)
	if err != nil {
		t.Error("Error creating new wallet: " + err.Error())
		return
	}

	if err = w.Unlock([]byte("banana")); err != nil {
		t.Errorf("Can't unlock original wallet: %v", err)
		return
	}

//验证整个钱包的同步高度是否与
//应为CreateHeight。
	if _, h := w.SyncedTo(); h != createHeight {
		t.Errorf("Initial sync height %v does not match expected %v.", h, createHeight)
		return
	}

	script := []byte{txscript.OP_TRUE, txscript.OP_DUP,
		txscript.OP_DROP}
	importHeight := int32(50)
	stamp := makeBS(importHeight)
	address, err := w.ImportScript(script, stamp)
	if err != nil {
		t.Error("error importing script: " + err.Error())
		return
	}

//查找地址
	ainfo, err := w.Address(address)
	if err != nil {
		t.Error("error looking up script: " + err.Error())
	}

	sinfo := ainfo.(ScriptAddress)

	if !bytes.Equal(script, sinfo.Script()) {
		t.Error("original and looked-up script do not match.")
		return
	}

	if sinfo.ScriptClass() != txscript.NonStandardTy {
		t.Error("script type incorrect.")
		return
	}

	if sinfo.RequiredSigs() != 0 {
		t.Error("required sigs funny number")
		return
	}

	if len(sinfo.Addresses()) != 0 {
		t.Error("addresses in bogus script.")
		return
	}

	if sinfo.Address().EncodeAddress() != address.EncodeAddress() {
		t.Error("script address doesn't match entry.")
		return
	}

	if string(sinfo.Address().ScriptAddress()) != sinfo.AddrHash() {
		t.Error("script hash doesn't match address.")
		return
	}

	if sinfo.FirstBlock() != importHeight {
		t.Error("funny first block")
		return
	}

	if !sinfo.Imported() {
		t.Error("imported script info not imported.")
		return
	}

	if sinfo.Change() {
		t.Error("imported script is change.")
		return
	}

	if sinfo.Compressed() {
		t.Error("imported script is compressed.")
		return
	}

//验证同步高度现在是否与（较小的）导入高度匹配。
	if _, h := w.SyncedTo(); h != importHeight {
		t.Errorf("After import sync height %v does not match expected %v.", h, importHeight)
		return
	}

//检查它是否包括在活动付款地址中。
	found := false
	for _, wa := range w.SortedActiveAddresses() {
		if wa.Address() == address {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Imported script address was not returned with sorted active payment addresses.")
		return
	}
	if _, ok := w.ActiveAddresses()[address]; !ok {
		t.Errorf("Imported script address was not returned with unsorted active payment addresses.")
		return
	}

//继续，去士气，检查那里。

//钱包测试（反）系列化。
	buf := new(bytes.Buffer)
	_, err = w.WriteTo(buf)
	if err != nil {
		t.Errorf("Cannot write wallet: %v", err)
		return
	}
	w2 := new(Store)
	_, err = w2.ReadFrom(buf)
	if err != nil {
		t.Errorf("Cannot read wallet: %v", err)
		return
	}

//验证同步高度是否与重新初始化后预期的高度匹配。
	if _, h := w2.SyncedTo(); h != importHeight {
		t.Errorf("After reserialization sync height %v does not match expected %v.", h, importHeight)
		return
	}

//查找地址
	ainfo2, err := w2.Address(address)
	if err != nil {
		t.Error("error looking up info in deserialized wallet: " + err.Error())
	}

	sinfo2 := ainfo2.(ScriptAddress)
//再检查一遍。我们不能使用reflect.deepequals，因为
//内部有指向钱包结构的指针。
	if sinfo2.Address().EncodeAddress() != address.EncodeAddress() {
		t.Error("script address doesn't match entry.")
		return
	}

	if string(sinfo2.Address().ScriptAddress()) != sinfo2.AddrHash() {
		t.Error("script hash doesn't match address.")
		return
	}

	if sinfo2.FirstBlock() != importHeight {
		t.Error("funny first block")
		return
	}

	if !sinfo2.Imported() {
		t.Error("imported script info not imported.")
		return
	}

	if sinfo2.Change() {
		t.Error("imported script is change.")
		return
	}

	if sinfo2.Compressed() {
		t.Error("imported script is compressed.")
		return
	}

	if !bytes.Equal(sinfo.Script(), sinfo2.Script()) {
		t.Error("original and serailised scriptinfo scripts "+
			"don't match %s != %s", spew.Sdump(sinfo.Script()),
			spew.Sdump(sinfo2.Script()))
	}

	if sinfo.ScriptClass() != sinfo2.ScriptClass() {
		t.Error("original and serailised scriptinfo class "+
			"don't match: %s != %s", sinfo.ScriptClass(),
			sinfo2.ScriptClass())
		return
	}

	if !reflect.DeepEqual(sinfo.Addresses(), sinfo2.Addresses()) {
		t.Error("original and serailised scriptinfo addresses "+
			"don't match (%s) != (%s)", spew.Sdump(sinfo.Addresses),
			spew.Sdump(sinfo2.Addresses()))
		return
	}

	if sinfo.RequiredSigs() != sinfo.RequiredSigs() {
		t.Errorf("original and serailised scriptinfo requiredsigs "+
			"don't match %d != %d", sinfo.RequiredSigs(),
			sinfo2.RequiredSigs())
		return
	}

//检查它是否包括在活动付款地址中。
	found = false
	for _, wa := range w.SortedActiveAddresses() {
		if wa.Address() == address {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("After reserialiation, imported script address was not returned with sorted " +
			"active payment addresses.")
		return
	}
	if _, ok := w.ActiveAddresses()[address]; !ok {
		t.Errorf("After reserialiation, imported script address was not returned with unsorted " +
			"active payment addresses.")
		return
	}

//将导入的地址标记为与中间某个块部分同步
//导入高度和链高度。
	partialHeight := (createHeight-importHeight)/2 + importHeight
	if err := w2.SetSyncStatus(address, PartialSync(partialHeight)); err != nil {
		t.Errorf("Cannot mark address partially synced: %v", err)
		return
	}
	if _, h := w2.SyncedTo(); h != partialHeight {
		t.Errorf("After address partial sync, sync height %v does not match expected %v.", h, partialHeight)
		return
	}

//使用部分同步测试序列化。
	buf.Reset()
	_, err = w2.WriteTo(buf)
	if err != nil {
		t.Errorf("Cannot write wallet: %v", err)
		return
	}
	w3 := new(Store)
	_, err = w3.ReadFrom(buf)
	if err != nil {
		t.Errorf("Cannot read wallet: %v", err)
		return
	}

//序列化后测试正确的部分高度。
	if _, h := w3.SyncedTo(); h != partialHeight {
		t.Errorf("After address partial sync and reserialization, sync height %v does not match expected %v.",
			h, partialHeight)
		return
	}

//将导入的地址标记为完全不同步，并验证同步高度现在是否为
//导入高度。
	if err := w3.SetSyncStatus(address, Unsynced(0)); err != nil {
		t.Errorf("Cannot mark address synced: %v", err)
		return
	}
	if _, h := w3.SyncedTo(); h != importHeight {
		t.Errorf("After address unsync, sync height %v does not match expected %v.", h, importHeight)
		return
	}

//将导入的地址标记为与最近看到的块同步，并验证
//同步高度现在等于最近的块（钱包上的块
//创造）。
	if err := w3.SetSyncStatus(address, FullSync{}); err != nil {
		t.Errorf("Cannot mark address synced: %v", err)
		return
	}
	if _, h := w3.SyncedTo(); h != createHeight {
		t.Errorf("After address sync, sync height %v does not match expected %v.", h, createHeight)
		return
	}

	if err = w3.Unlock([]byte("banana")); err != nil {
		t.Errorf("Can't unlock deserialised wallet: %v", err)
		return
	}
}

func TestChangePassphrase(t *testing.T) {
	createdAt := makeBS(0)
	w, err := New(dummyDir, "A wallet for testing.",
		[]byte("banana"), tstNetParams, createdAt)
	if err != nil {
		t.Error("Error creating new wallet: " + err.Error())
		return
	}

//用已锁定的钱包更改密码时，必须在errwalletlocked失败。
	if err := w.ChangePassphrase([]byte("potato")); err != ErrLocked {
		t.Errorf("Changing passphrase on a locked wallet did not fail correctly: %v", err)
		return
	}

//打开钱包，以便更改密码。
	if err := w.Unlock([]byte("banana")); err != nil {
		t.Errorf("Cannot unlock: %v", err)
		return
	}

//获取根地址及其私钥。这和私人的比较
//密钥发布密码短语更改。
	rootAddr := w.LastChainedAddress()

	rootAddrInfo, err := w.Address(rootAddr)
	if err != nil {
		t.Error("can't find root address: " + err.Error())
		return
	}
	rapka := rootAddrInfo.(PubKeyAddress)

	rootPrivKey, err := rapka.PrivKey()
	if err != nil {
		t.Errorf("Cannot get root address' private key: %v", err)
		return
	}

//更改密码。
	if err := w.ChangePassphrase([]byte("potato")); err != nil {
		t.Errorf("Changing passphrase failed: %v", err)
		return
	}

//钱包仍应解锁。
	if w.IsLocked() {
		t.Errorf("Wallet should be unlocked after passphrase change.")
		return
	}

//把它锁起来。
	if err := w.Lock(); err != nil {
		t.Errorf("Cannot lock wallet after passphrase change: %v", err)
		return
	}

//用旧密码解锁。这必须以errwronpassphrase失败。
	if err := w.Unlock([]byte("banana")); err != ErrWrongPassphrase {
		t.Errorf("Unlocking with old passphrases did not fail correctly: %v", err)
		return
	}

//用新密码解锁。这必须成功。
	if err := w.Unlock([]byte("potato")); err != nil {
		t.Errorf("Unlocking with new passphrase failed: %v", err)
		return
	}

//再次获取根地址的私钥。
	rootAddrInfo2, err := w.Address(rootAddr)
	if err != nil {
		t.Error("can't find root address: " + err.Error())
		return
	}
	rapka2 := rootAddrInfo2.(PubKeyAddress)

	rootPrivKey2, err := rapka2.PrivKey()
	if err != nil {
		t.Errorf("Cannot get root address' private key after passphrase change: %v", err)
		return
	}

//私钥必须匹配。
	if !reflect.DeepEqual(rootPrivKey, rootPrivKey2) {
		t.Errorf("Private keys before and after unlock differ.")
		return
	}
}
