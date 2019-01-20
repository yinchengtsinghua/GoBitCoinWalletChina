
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
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/btcsuite/btcwallet/internal/zero"
	"github.com/btcsuite/btcwallet/walletdb"
)

//AddressType表示waddrmgr当前可以使用的各种地址类型
//生成和维护。
//
//注意：这些必须是稳定的，因为它们用于范围地址模式。
//在数据库中识别。
type AddressType uint8

const (
//pubkeyhash是一个普通的p2pkh地址。
	PubKeyHash AddressType = iota

//脚本重新打印原始脚本地址。
	Script

//rawpubkey只是要在脚本中使用的原始公钥，这是
//类型指示具有此地址类型的作用域管理器
//历史重新扫描时不应咨询。
	RawPubKey

//NestedWitnessPubKey表示嵌套在p2sh中的p2wkh输出
//输出。使用此地址类型，钱包可以从
//其他尚未识别新Segwit标准的钱包
//输出类型。接收到该地址的资金保持
//可伸缩性和可延展性修复由于Segwit在
//兼容的方式。
	NestedWitnessPubKey

//witnesspubkey表示p2wkh（付款到见证密钥哈希）地址
//类型。
	WitnessPubKey
)

//ManagedAddress是一个接口，提供有关
//由地址管理器管理的地址。这个的具体实现
//类型可以提供其他字段以提供特定于该类型的信息
//地址。
type ManagedAddress interface {
//account返回与地址关联的帐户。
	Account() uint32

//地址返回备份地址的btncutil.address。
	Address() btcutil.Address

//addrhash返回与地址相关的键或脚本哈希
	AddrHash() []byte

//如果后台地址被导入，则imported返回true
//作为地址链的一部分。
	Imported() bool

//如果为内部创建了备用地址，则内部返回true
//使用，例如事务的更改输出。
	Internal() bool

//如果备份地址被压缩，则compressed返回true。
	Compressed() bool

//如果事务中使用了备用地址，则used返回true。
	Used(ns walletdb.ReadBucket) bool

//addrtype返回托管地址的地址类型。这个罐头
//用于快速识别地址类型，无需进一步
//处理
	AddrType() AddressType
}

//managedSubkeyAddress扩展了managedAddress，并另外提供
//基于公钥的地址的公钥和私钥。
type ManagedPubKeyAddress interface {
	ManagedAddress

//pubkey返回与地址关联的公钥。
	PubKey() *btcec.PublicKey

//exportpubkey返回与地址关联的公钥
//序列化为十六进制编码字符串。
	ExportPubKey() string

//privkey返回地址的私钥。如果
//地址管理器只监视或锁定，或者地址不监视
//有钥匙。
	PrivKey() (*btcec.PrivateKey, error)

//exportprivkey返回与地址关联的私钥
//序列化为钱包导入格式（WIF）。
	ExportPrivKey() (*btcutil.WIF, error)

//DerivationInfo包含派生密钥所需的信息
//它通过从hd根目录的传统方法来备份地址。为了
//导入的键，第一个值将设置为false以指示
//我们不知道密钥是如何派生的。
	DerivationInfo() (KeyScope, DerivationPath, bool)
}

//ManagedScriptAddress扩展了ManagedAddress并表示付费脚本哈希
//比特币地址的样式。它还提供有关
//脚本。
type ManagedScriptAddress interface {
	ManagedAddress

//脚本返回与地址关联的脚本。
	Script() ([]byte, error)
}

//ManagedAddress表示公钥地址。它也可能有也可能没有
//与公钥关联的私钥。
type managedAddress struct {
	manager          *ScopedKeyManager
	derivationPath   DerivationPath
	address          btcutil.Address
	imported         bool
	internal         bool
	compressed       bool
	used             bool
	addrType         AddressType
	pubKey           *btcec.PublicKey
	privKeyEncrypted []byte
privKeyCT        []byte //解锁时不为零
	privKeyMutex     sync.Mutex
}

//强制ManagedAddress满足ManagedSubKeyAddress接口。
var _ ManagedPubKeyAddress = (*managedAddress)(nil)

//解锁解密并存储指向关联私钥的指针。它将
//如果密钥无效或加密的私钥不可用，则失败。
//返回的明文私钥将始终是安全的副本。
//由调用方使用，而不必担心在地址期间将其归零
//锁。
func (a *managedAddress) unlock(key EncryptorDecryptor) ([]byte, error) {
//
	a.privKeyMutex.Lock()
	defer a.privKeyMutex.Unlock()

	if len(a.privKeyCT) == 0 {
		privKey, err := key.Decrypt(a.privKeyEncrypted)
		if err != nil {
			str := fmt.Sprintf("failed to decrypt private key for "+
				"%s", a.address)
			return nil, managerError(ErrCrypto, str, err)
		}

		a.privKeyCT = privKey
	}

	privKeyCopy := make([]byte, len(a.privKeyCT))
	copy(privKeyCopy, a.privKeyCT)
	return privKeyCopy, nil
}

//lock使关联的明文私钥归零。
func (a *managedAddress) lock() {
//
//地址。
	a.privKeyMutex.Lock()
	zero.Bytes(a.privKeyCT)
	a.privKeyCT = nil
	a.privKeyMutex.Unlock()
}

//account返回与地址关联的帐号。
//
//
func (a *managedAddress) Account() uint32 {
	return a.derivationPath.Account
}

//
//
//
//
func (a *managedAddress) AddrType() AddressType {
	return a.addrType
}

//
//
//
//
func (a *managedAddress) Address() btcutil.Address {
	return a.address
}

//
//
//
func (a *managedAddress) AddrHash() []byte {
	var hash []byte

	switch n := a.address.(type) {
	case *btcutil.AddressPubKeyHash:
		hash = n.Hash160()[:]
	case *btcutil.AddressScriptHash:
		hash = n.Hash160()[:]
	case *btcutil.AddressWitnessPubKeyHash:
		hash = n.Hash160()[:]
	}

	return hash
}

//
//
//
//
func (a *managedAddress) Imported() bool {
	return a.imported
}

//
//更改事务的输出。
//
//
func (a *managedAddress) Internal() bool {
	return a.internal
}

//
//
//这是ManagedAddress接口实现的一部分。
func (a *managedAddress) Compressed() bool {
	return a.compressed
}

//
//
//这是ManagedAddress接口实现的一部分。
func (a *managedAddress) Used(ns walletdb.ReadBucket) bool {
	return a.manager.fetchUsed(ns, a.AddrHash())
}

//pubkey返回与地址关联的公钥。
//
//这是ManagedSubKeyAddress接口实现的一部分。
func (a *managedAddress) PubKey() *btcec.PublicKey {
	return a.pubKey
}

//pubKeyBytes返回托管地址的序列化公钥字节
//基于托管地址是否标记为压缩。
func (a *managedAddress) pubKeyBytes() []byte {
	if a.compressed {
		return a.pubKey.SerializeCompressed()
	}
	return a.pubKey.SerializeUncompressed()
}

//exportpubkey返回与地址关联的公钥
//序列化为十六进制编码字符串。
//
//这是ManagedSubKeyAddress接口实现的一部分。
func (a *managedAddress) ExportPubKey() string {
	return hex.EncodeToString(a.pubKeyBytes())
}

//privkey返回地址的私钥。如果地址
//
//
//这是ManagedSubKeyAddress接口实现的一部分。
func (a *managedAddress) PrivKey() (*btcec.PrivateKey, error) {
//
	if a.manager.rootManager.WatchOnly() {
		return nil, managerError(ErrWatchingOnly, errWatchingOnly, nil)
	}

	a.manager.mtx.Lock()
	defer a.manager.mtx.Unlock()

//必须解锁帐户管理器才能解密私钥。
	if a.manager.rootManager.IsLocked() {
		return nil, managerError(ErrLocked, errLocked, nil)
	}

//
//
//返回的私钥可能从调用方下无效。
	privKeyCopy, err := a.unlock(a.manager.rootManager.cryptoKeyPriv)
	if err != nil {
		return nil, err
	}

	privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKeyCopy)
	zero.Bytes(privKeyCopy)
	return privKey, nil
}

//
//
//
//这是ManagedSubKeyAddress接口实现的一部分。
func (a *managedAddress) ExportPrivKey() (*btcutil.WIF, error) {
	pk, err := a.PrivKey()
	if err != nil {
		return nil, err
	}

	return btcutil.NewWIF(pk, a.manager.rootManager.chainParams, a.compressed)
}

//
//
//
//
//
//这是ManagedSubKeyAddress接口实现的一部分。
func (a *managedAddress) DerivationInfo() (KeyScope, DerivationPath, bool) {
	var (
		scope KeyScope
		path  DerivationPath
	)

//
//不知道密钥是如何派生的。
	if a.imported {
		return scope, path, false
	}

	return a.manager.Scope(), a.derivationPath, true
}

//
//
//
func newManagedAddressWithoutPrivKey(m *ScopedKeyManager,
	derivationPath DerivationPath, pubKey *btcec.PublicKey, compressed bool,
	addrType AddressType) (*managedAddress, error) {

//
	var pubKeyHash []byte
	if compressed {
		pubKeyHash = btcutil.Hash160(pubKey.SerializeCompressed())
	} else {
		pubKeyHash = btcutil.Hash160(pubKey.SerializeUncompressed())
	}

	var address btcutil.Address
	var err error

	switch addrType {

	case NestedWitnessPubKey:
//
//
//
//

//
		witAddr, err := btcutil.NewAddressWitnessPubKeyHash(
			pubKeyHash, m.rootManager.chainParams,
		)
		if err != nil {
			return nil, err
		}

//
//
		witnessProgram, err := txscript.PayToAddrScript(witAddr)
		if err != nil {
			return nil, err
		}

//
//
//
//<sig，pubkey>pair作为证人。
		address, err = btcutil.NewAddressScriptHash(
			witnessProgram, m.rootManager.chainParams,
		)
		if err != nil {
			return nil, err
		}

	case PubKeyHash:
		address, err = btcutil.NewAddressPubKeyHash(
			pubKeyHash, m.rootManager.chainParams,
		)
		if err != nil {
			return nil, err
		}

	case WitnessPubKey:
		address, err = btcutil.NewAddressWitnessPubKeyHash(
			pubKeyHash, m.rootManager.chainParams,
		)
		if err != nil {
			return nil, err
		}
	}

	return &managedAddress{
		manager:          m,
		address:          address,
		derivationPath:   derivationPath,
		imported:         false,
		internal:         false,
		addrType:         addrType,
		compressed:       compressed,
		pubKey:           pubKey,
		privKeyEncrypted: nil,
		privKeyCT:        nil,
	}, nil
}

//
//
//
func newManagedAddress(s *ScopedKeyManager, derivationPath DerivationPath,
	privKey *btcec.PrivateKey, compressed bool,
	addrType AddressType) (*managedAddress, error) {

//
//
//
//
	privKeyBytes := privKey.Serialize()
	privKeyEncrypted, err := s.rootManager.cryptoKeyPriv.Encrypt(privKeyBytes)
	if err != nil {
		str := "failed to encrypt private key"
		return nil, managerError(ErrCrypto, str, err)
	}

//
//
	ecPubKey := (*btcec.PublicKey)(&privKey.PublicKey)
	managedAddr, err := newManagedAddressWithoutPrivKey(
		s, derivationPath, ecPubKey, compressed, addrType,
	)
	if err != nil {
		return nil, err
	}
	managedAddr.privKeyEncrypted = privKeyEncrypted
	managedAddr.privKeyCT = privKeyBytes

	return managedAddr, nil
}

//NewManagedAddResfRomextKey基于传递的
//
//
//
func newManagedAddressFromExtKey(s *ScopedKeyManager,
	derivationPath DerivationPath, key *hdkeychain.ExtendedKey,
	addrType AddressType) (*managedAddress, error) {

//
//
	var managedAddr *managedAddress
	if key.IsPrivate() {
		privKey, err := key.ECPrivKey()
		if err != nil {
			return nil, err
		}

//确保临时私钥大整数在
//使用。
		managedAddr, err = newManagedAddress(
			s, derivationPath, privKey, true, addrType,
		)
		if err != nil {
			return nil, err
		}
	} else {
		pubKey, err := key.ECPubKey()
		if err != nil {
			return nil, err
		}

		managedAddr, err = newManagedAddressWithoutPrivKey(
			s, derivationPath, pubKey, true,
			addrType,
		)
		if err != nil {
			return nil, err
		}
	}

	return managedAddr, nil
}

//
type scriptAddress struct {
	manager         *ScopedKeyManager
	account         uint32
	address         *btcutil.AddressScriptHash
	scriptEncrypted []byte
	scriptCT        []byte
	scriptMutex     sync.Mutex
	used            bool
}

//
var _ ManagedScriptAddress = (*scriptAddress)(nil)

//
//
//
//
func (a *scriptAddress) unlock(key EncryptorDecryptor) ([]byte, error) {
//
	a.scriptMutex.Lock()
	defer a.scriptMutex.Unlock()

	if len(a.scriptCT) == 0 {
		script, err := key.Decrypt(a.scriptEncrypted)
		if err != nil {
			str := fmt.Sprintf("failed to decrypt script for %s",
				a.address)
			return nil, managerError(ErrCrypto, str, err)
		}

		a.scriptCT = script
	}

	scriptCopy := make([]byte, len(a.scriptCT))
	copy(scriptCopy, a.scriptCT)
	return scriptCopy, nil
}

//lock使关联的明文私钥归零。
func (a *scriptAddress) lock() {
//
	a.scriptMutex.Lock()
	zero.Bytes(a.scriptCT)
	a.scriptCT = nil
	a.scriptMutex.Unlock()
}

//
//
//
//这是ManagedAddress接口实现的一部分。
func (a *scriptAddress) Account() uint32 {
	return a.account
}

//
//
//
//这是ManagedAddress接口实现的一部分。
func (a *scriptAddress) AddrType() AddressType {
	return Script
}

//
//
//
//这是ManagedAddress接口实现的一部分。
func (a *scriptAddress) Address() btcutil.Address {
	return a.address
}

//
//
//这是ManagedAddress接口实现的一部分。
func (a *scriptAddress) AddrHash() []byte {
	return a.address.Hash160()[:]
}

//
//
//
//这是ManagedAddress接口实现的一部分。
func (a *scriptAddress) Imported() bool {
	return true
}

//
//
//
//这是ManagedAddress接口实现的一部分。
func (a *scriptAddress) Internal() bool {
	return false
}

//compressed返回false，因为脚本地址从未被压缩。
//
//这是ManagedAddress接口实现的一部分。
func (a *scriptAddress) Compressed() bool {
	return false
}

//
//
//这是ManagedAddress接口实现的一部分。
func (a *scriptAddress) Used(ns walletdb.ReadBucket) bool {
	return a.manager.fetchUsed(ns, a.AddrHash())
}

//脚本返回与地址关联的脚本。
//
//
func (a *scriptAddress) Script() ([]byte, error) {
//
	if a.manager.rootManager.WatchOnly() {
		return nil, managerError(ErrWatchingOnly, errWatchingOnly, nil)
	}

	a.manager.mtx.Lock()
	defer a.manager.mtx.Unlock()

//
	if a.manager.rootManager.IsLocked() {
		return nil, managerError(ErrLocked, errLocked, nil)
	}

//
//
//
	return a.unlock(a.manager.rootManager.cryptoKeyScript)
}

//
func newScriptAddress(m *ScopedKeyManager, account uint32, scriptHash,
	scriptEncrypted []byte) (*scriptAddress, error) {

	address, err := btcutil.NewAddressScriptHashFromHash(
		scriptHash, m.rootManager.chainParams,
	)
	if err != nil {
		return nil, err
	}

	return &scriptAddress{
		manager:         m,
		account:         account,
		address:         address,
		scriptEncrypted: scriptEncrypted,
	}, nil
}
