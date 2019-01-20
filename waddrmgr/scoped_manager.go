
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package waddrmgr

import (
	"fmt"
	"sync"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/btcsuite/btcwallet/internal/zero"
	"github.com/btcsuite/btcwallet/walletdb"
)

//DerivationPath表示特定密钥管理器的派生路径
//范围。每个ScopedKeyManager从其
//CoinType硬键：m/用途'/CoinType'。此结构中的字段允许
//在硬币类型键之后，进一步衍生到下三个儿童级别。
//这个限制在bip0044类型派生的spriti中。我们维持一个
//
//在coinType键之外。使用此路径派生的密钥将是：
//m/purpose'/cointype'/account/branch/index，其中purpose'和cointype'是
//受特定管理者范围的约束。
type DerivationPath struct {
//帐户是帐户，或作用域中的第一个子级
//经理的硬硬币型钥匙。
	Account uint32

//分支是从上面的帐户索引派生的分支。为了
//bip0044类似于派生，这是0（外部）或1
//（内部）。然而，我们允许这个值在
//其大小范围。
	Branch uint32

//索引是派生路径中的最后一个子项。这表示
//在中作为帐户和分支机构的子级的键索引。
	Index uint32
}

//key scope表示来自内部主根键的受限键作用域
//高清链。从根管理器（m/）我们可以创建一个几乎任意的
//密钥派生路径的ScopedKeyManager数：m/purpose'/coinType'。
//这些范围内的管理者可以让我无礼地管理，因为他们拥有
//加密的cointype密钥，可以从中派生任何子密钥。
type KeyScope struct {
//目的就是这个关键范围的目的。这是第一个孩子
//主HD键。
	Purpose uint32

//硬币是代表特定硬币的价值，即
//目的键的子级。使用此密钥、任何帐户或其他
//子元素完全可以派生出来。
	Coin uint32
}

//ScopedIndex是键作用域和子索引的元组。这是用来紧凑的
//当可以推断出帐户和分支时，标识特定的子密钥
//从上下文。
type ScopedIndex struct {
//作用域是“bip44帐户”，用于派生子密钥。
	Scope KeyScope

//index是用于派生子键的bip44地址索引。
	Index uint32
}

//字符串返回描述封装的密钥路径的可读版本
//按目标键范围。
func (k *KeyScope) String() string {
	return fmt.Sprintf("m/%v'/%v'", k.Purpose, k.Coin)
}

//ScopeAddrSchema是特定键作用域的地址架构。这将是
//保留在数据库中，并且在派生任何键时将被查询
//
type ScopeAddrSchema struct {
//ExternalAddrType是分支0中所有键的地址类型。
	ExternalAddrType AddressType

//InternalAddrType是分支1内所有键的地址类型
//（更改地址）。
	InternalAddrType AddressType
}

var (
//keyscopebip0049plus是我们改进的bip0049的关键范围
//推导。我们说这是bip0049“plus”，因为我们将实际使用
//p2wkh更改所有更改地址。
	KeyScopeBIP0049Plus = KeyScope{
		Purpose: 49,
		Coin:    0,
	}

//keyscopebip084是bip0084派生的关键范围。BIP084B
//将用于派生所有p2wkh地址。
	KeyScopeBIP0084 = KeyScope{
		Purpose: 84,
		Coin:    0,
	}

//key scope bip0044是bip0044派生的关键范围。遗产
//钱包只能使用这个密钥范围，而且不能使用超出范围的密钥。
//它。
	KeyScopeBIP0044 = KeyScope{
		Purpose: 44,
		Coin:    0,
	}

//DefaultKeyScopes是将
//由根管理器在初始创建时创建。
	DefaultKeyScopes = []KeyScope{
		KeyScopeBIP0049Plus,
		KeyScopeBIP0084,
		KeyScopeBIP0044,
	}

//scopeAddrMap是从默认键作用域到作用域的映射
//每个作用域类型的地址架构。这将在
//根密钥管理器的初始创建。
	ScopeAddrMap = map[KeyScope]ScopeAddrSchema{
		KeyScopeBIP0049Plus: {
			ExternalAddrType: NestedWitnessPubKey,
			InternalAddrType: WitnessPubKey,
		},
		KeyScopeBIP0084: {
			ExternalAddrType: WitnessPubKey,
			InternalAddrType: WitnessPubKey,
		},
		KeyScopeBIP0044: {
			InternalAddrType: PubKeyHash,
			ExternalAddrType: PubKeyHash,
		},
	}
)

//ScopedKeyManager是主根密钥管理器下的子密钥管理器。这个
//根键管理器将处理根hd键（m/），而每个子范围键
//管理器将处理特定键范围的CoinType键
//
//在根密钥管理器的基础上构建，以执行自己的任意密钥
//派生，但仍受根密钥加密保护
//经理。
type ScopedKeyManager struct {
//范围是此密钥管理器的范围。我们只能生成密钥
//这是这个范围的直接子级。
	scope KeyScope

//addrschema是此子管理器的地址架构。这将是
//对派生密钥中的地址进行编码时进行咨询。
	addrSchema ScopeAddrSchema

//root manager是指向根密钥管理器的指针。我们将保持
//这是因为我们需要访问加密密钥才能
//派生子帐户密钥的任何新帐户。
	rootManager *Manager

//
//经理。
	addrs map[addrKey]ManagedAddress

//acctinfo包含有关帐户的信息，包括需要的内容
//为每个创建的帐户生成确定性链接键。
	acctInfo map[uint32]*accountInfo

//DeriveOnUnlock是需要派生的私钥列表
//下一次解锁时。当导出公共广播时会发生这种情况
//地址管理器被锁定，因为它无权访问
//中的专用扩展密钥（因此也不是基础私钥）
//命令加密它。
	deriveOnUnlock []*unlockDeriveInfo

	mtx sync.RWMutex
}

//作用域返回此作用域密钥管理器的确切密钥作用域。
func (s *ScopedKeyManager) Scope() KeyScope {
	return s.scope
}

//addrschema返回目标ScopedKeyManager的设置地址架构。
func (s *ScopedKeyManager) AddrSchema() ScopeAddrSchema {
	return s.addrSchema
}

//ZeroSensitivePublicData会尽最大努力删除所有内容并将其归零。
//与地址管理器关联的敏感公共数据，如
//层次确定性扩展公钥和加密公钥。
func (s *ScopedKeyManager) zeroSensitivePublicData() {
//清除所有帐户私钥。
	for _, acctInfo := range s.acctInfo {
		acctInfo.acctKeyPub.Zero()
		acctInfo.acctKeyPub = nil
	}
}

//干净利落地关闭经理。尽最大努力移除
//将所有与
//内存中的地址管理器。
func (s *ScopedKeyManager) Close() {
	s.mtx.Lock()
	defer s.mtx.Unlock()

//尝试从内存中清除敏感的公钥材料。
	s.zeroSensitivePublicData()
	return
}

//keytomanaged为提供的派生密钥返回新的托管地址，并且
//它的派生路径由帐户、分支和索引组成。
//
//传递的派生密钥在创建新地址后归零。
//
//调用此函数时必须保留管理器锁以进行写入。
func (s *ScopedKeyManager) keyToManaged(derivedKey *hdkeychain.ExtendedKey,
	account, branch, index uint32) (ManagedAddress, error) {

	var addrType AddressType
	if branch == InternalBranch {
		addrType = s.addrSchema.InternalAddrType
	} else {
		addrType = s.addrSchema.ExternalAddrType
	}

	derivationPath := DerivationPath{
		Account: account,
		Branch:  branch,
		Index:   index,
	}

//基于公钥或私钥创建新的托管地址
//取决于传递的密钥是否是私有的。另外，把钥匙调零
//从中创建托管地址之后。
	ma, err := newManagedAddressFromExtKey(
		s, derivationPath, derivedKey, addrType,
	)
	defer derivedKey.Zero()
	if err != nil {
		return nil, err
	}

	if !derivedKey.IsPrivate() {
//将托管地址添加到需要的地址列表中
//
//解锁。
		info := unlockDeriveInfo{
			managedAddr: ma,
			branch:      branch,
			index:       index,
		}
		s.deriveOnUnlock = append(s.deriveOnUnlock, &info)
	}

	if branch == InternalBranch {
		ma.internal = true
	}

	return ma, nil
}

//派生键返回基于
//给定帐户信息、分支和索引的私有标志。
func (s *ScopedKeyManager) deriveKey(acctInfo *accountInfo, branch,
	index uint32, private bool) (*hdkeychain.ExtendedKey, error) {

//根据是否选择公钥或私钥扩展密钥
//已指定私有标志。这反过来又允许公众或
//私生子派生。
	acctKey := acctInfo.acctKeyPub
	if private {
		acctKey = acctInfo.acctKeyPriv
	}

//派生并返回键。
	branchKey, err := acctKey.Child(branch)
	if err != nil {
		str := fmt.Sprintf("failed to derive extended key branch %d",
			branch)
		return nil, managerError(ErrKeyChain, str, err)
	}

	addressKey, err := branchKey.Child(index)
branchKey.Zero() //使用后将分支键归零。
	if err != nil {
		str := fmt.Sprintf("failed to derive child extended key -- "+
			"branch %d, child %d",
			branch, index)
		return nil, managerError(ErrKeyChain, str, err)
	}

	return addressKey, nil
}

//LoadAccountInfo尝试加载和缓存有关给定
//数据库中的帐户。这包括获得新的
//并跟踪内部和外部分支的状态。
//
//调用此函数时必须保留管理器锁以进行写入。
func (s *ScopedKeyManager) loadAccountInfo(ns walletdb.ReadBucket,
	account uint32) (*accountInfo, error) {

//从缓存返回帐户信息（如果可用）。
	if acctInfo, ok := s.acctInfo[account]; ok {
		return acctInfo, nil
	}

//
//从数据库加载信息。
	rowInterface, err := fetchAccountInfo(ns, &s.scope, account)
	if err != nil {
		return nil, maybeConvertDbError(err)
	}

//确保帐户类型为默认帐户。
	row, ok := rowInterface.(*dbDefaultAccountRow)
	if !ok {
		str := fmt.Sprintf("unsupported account type %T", row)
		return nil, managerError(ErrDatabase, str, nil)
	}

//使用加密公钥解密帐户public extended
//关键。
	serializedKeyPub, err := s.rootManager.cryptoKeyPub.Decrypt(row.pubKeyEncrypted)
	if err != nil {
		str := fmt.Sprintf("failed to decrypt public key for account %d",
			account)
		return nil, managerError(ErrCrypto, str, err)
	}
	acctKeyPub, err := hdkeychain.NewKeyFromString(string(serializedKeyPub))
	if err != nil {
		str := fmt.Sprintf("failed to create extended public key for "+
			"account %d", account)
		return nil, managerError(ErrKeyChain, str, err)
	}

//使用已知信息创建新帐户信息。其余的
//字段填写在下面。
	acctInfo := &accountInfo{
		acctName:          row.name,
		acctKeyEncrypted:  row.privKeyEncrypted,
		acctKeyPub:        acctKeyPub,
		nextExternalIndex: row.nextExternalIndex,
		nextInternalIndex: row.nextInternalIndex,
	}

	if !s.rootManager.isLocked() {
//使用加密私钥解密帐户private
//扩展键。
		decrypted, err := s.rootManager.cryptoKeyPriv.Decrypt(acctInfo.acctKeyEncrypted)
		if err != nil {
			str := fmt.Sprintf("failed to decrypt private key for "+
				"account %d", account)
			return nil, managerError(ErrCrypto, str, err)
		}

		acctKeyPriv, err := hdkeychain.NewKeyFromString(string(decrypted))
		if err != nil {
			str := fmt.Sprintf("failed to create extended private "+
				"key for account %d", account)
			return nil, managerError(ErrKeyChain, str, err)
		}
		acctInfo.acctKeyPriv = acctKeyPriv
	}

//派生并缓存最后一个外部地址的托管地址。
	branch, index := ExternalBranch, row.nextExternalIndex
	if index > 0 {
		index--
	}
	lastExtKey, err := s.deriveKey(
		acctInfo, branch, index, !s.rootManager.isLocked(),
	)
	if err != nil {
		return nil, err
	}
	lastExtAddr, err := s.keyToManaged(lastExtKey, account, branch, index)
	if err != nil {
		return nil, err
	}
	acctInfo.lastExternalAddr = lastExtAddr

//派生并缓存最后一个内部地址的托管地址。
	branch, index = InternalBranch, row.nextInternalIndex
	if index > 0 {
		index--
	}
	lastIntKey, err := s.deriveKey(
		acctInfo, branch, index, !s.rootManager.isLocked(),
	)
	if err != nil {
		return nil, err
	}
	lastIntAddr, err := s.keyToManaged(lastIntKey, account, branch, index)
	if err != nil {
		return nil, err
	}
	acctInfo.lastInternalAddr = lastIntAddr

//将其添加到缓存中，并在一切成功时返回。
	s.acctInfo[account] = acctInfo
	return acctInfo, nil
}

//AccountProperties返回与帐户关联的属性，例如
//帐号、名称以及派生和导入的密钥的数目。
func (s *ScopedKeyManager) AccountProperties(ns walletdb.ReadBucket,
	account uint32) (*AccountProperties, error) {

	defer s.mtx.RUnlock()
	s.mtx.RLock()

	props := &AccountProperties{AccountNumber: account}

//在密钥可以导入任何帐户之前，特殊处理是
//对于导入的帐户是必需的。
//
//在导入的帐户上使用LoadAccountInfo时出错，因为
//
//密钥，导入的帐户没有。
//
//由于当前只有导入的帐户允许导入，因此
//任何其他帐户的导入密钥为零，并且由于
//导入的帐户不能包含未导入的密钥，外部和
//它的内部键计数为零。
	if account != ImportedAddrAccount {
		acctInfo, err := s.loadAccountInfo(ns, account)
		if err != nil {
			return nil, err
		}
		props.AccountName = acctInfo.acctName
		props.ExternalKeyCount = acctInfo.nextExternalIndex
		props.InternalKeyCount = acctInfo.nextInternalIndex
	} else {
props.AccountName = ImportedAddrAccountName //保留，不可挂起

//
		var importedKeyCount uint32
		count := func(interface{}) error {
			importedKeyCount++
			return nil
		}
		err := forEachAccountAddress(ns, &s.scope, ImportedAddrAccount, count)
		if err != nil {
			return nil, err
		}
		props.ImportedKeyCount = importedKeyCount
	}

	return props, nil
}

//DeriveFromKeyPath尝试派生最大的子键（在bip0044下
//方案）。如果无法进行密钥派生，则
//
func (s *ScopedKeyManager) DeriveFromKeyPath(ns walletdb.ReadBucket,
	kp DerivationPath) (ManagedAddress, error) {

	s.mtx.Lock()
	defer s.mtx.Unlock()

	extKey, err := s.deriveKeyFromPath(
		ns, kp.Account, kp.Branch, kp.Index, !s.rootManager.IsLocked(),
	)
	if err != nil {
		return nil, err
	}

	return s.keyToManaged(extKey, kp.Account, kp.Branch, kp.Index)
}

//DeriveKeyFromPath返回公共或私有派生扩展密钥
//基于给定帐户、分支和索引的私有标志。
//
//调用此函数时必须保留管理器锁以进行写入。
func (s *ScopedKeyManager) deriveKeyFromPath(ns walletdb.ReadBucket, account, branch,
	index uint32, private bool) (*hdkeychain.ExtendedKey, error) {

//查找帐户密钥信息。
	acctInfo, err := s.loadAccountInfo(ns, account)
	if err != nil {
		return nil, err
	}

	return s.deriveKey(acctInfo, branch, index, private)
}

//ChainAddressRowToManaged返回基于链接的新托管地址
//从数据库加载的地址数据。
//
//调用此函数时必须保留管理器锁以进行写入。
func (s *ScopedKeyManager) chainAddressRowToManaged(ns walletdb.ReadBucket,
	row *dbChainAddressRow) (ManagedAddress, error) {

//因为当调用此命令时，管理器的互斥被假定为保持不变
//函数中，我们使用内部的islocked来避免死锁。
	isLocked := s.rootManager.isLocked()

	addressKey, err := s.deriveKeyFromPath(
		ns, row.account, row.branch, row.index, !isLocked,
	)
	if err != nil {
		return nil, err
	}

	return s.keyToManaged(addressKey, row.account, row.branch, row.index)
}

//importedAddressRowToManaged返回基于导入的新托管地址
//从数据库加载的地址数据。
func (s *ScopedKeyManager) importedAddressRowToManaged(row *dbImportedAddressRow) (ManagedAddress, error) {

//使用加密公钥解密导入的公钥。
	pubBytes, err := s.rootManager.cryptoKeyPub.Decrypt(row.encryptedPubKey)
	if err != nil {
		str := "failed to decrypt public key for imported address"
		return nil, managerError(ErrCrypto, str, err)
	}

	pubKey, err := btcec.ParsePubKey(pubBytes, btcec.S256())
	if err != nil {
		str := "invalid public key for imported address"
		return nil, managerError(ErrCrypto, str, err)
	}

//因为这是一个导入的地址，所以我们不会填充完整的
//派生路径，因为我们没有足够的信息来这样做。
	derivationPath := DerivationPath{
		Account: row.account,
	}

	compressed := len(pubBytes) == btcec.PubKeyBytesLenCompressed
	ma, err := newManagedAddressWithoutPrivKey(
		s, derivationPath, pubKey, compressed,
		s.addrSchema.ExternalAddrType,
	)
	if err != nil {
		return nil, err
	}
	ma.privKeyEncrypted = row.encryptedPrivKey
	ma.imported = true

	return ma, nil
}

//
//从数据库加载的地址数据。
func (s *ScopedKeyManager) scriptAddressRowToManaged(row *dbScriptAddressRow) (ManagedAddress, error) {
//使用加密公钥解密导入的脚本哈希。
	scriptHash, err := s.rootManager.cryptoKeyPub.Decrypt(row.encryptedHash)
	if err != nil {
		str := "failed to decrypt imported script hash"
		return nil, managerError(ErrCrypto, str, err)
	}

	return newScriptAddress(s, row.account, scriptHash, row.encryptedScript)
}

//RowInterfaceToManaged基于给定的
//从数据库加载的地址数据。它将自动选择
//合适的类型。
//
//调用此函数时必须保留管理器锁以进行写入。
func (s *ScopedKeyManager) rowInterfaceToManaged(ns walletdb.ReadBucket,
	rowInterface interface{}) (ManagedAddress, error) {

	switch row := rowInterface.(type) {
	case *dbChainAddressRow:
		return s.chainAddressRowToManaged(ns, row)

	case *dbImportedAddressRow:
		return s.importedAddressRowToManaged(row)

	case *dbScriptAddressRow:
		return s.scriptAddressRowToManaged(row)
	}

	str := fmt.Sprintf("unsupported address type %T", rowInterface)
	return nil, managerError(ErrDatabase, str, nil)
}

//LoadAndCacheAddress尝试从数据库加载传递的地址
//并缓存关联的托管地址。
//
//调用此函数时必须保留管理器锁以进行写入。
func (s *ScopedKeyManager) loadAndCacheAddress(ns walletdb.ReadBucket,
	address btcutil.Address) (ManagedAddress, error) {

//尝试从数据库加载原始地址信息。
	rowInterface, err := fetchAddress(ns, &s.scope, address.ScriptAddress())
	if err != nil {
		if merr, ok := err.(*ManagerError); ok {
			desc := fmt.Sprintf("failed to fetch address '%s': %v",
				address.ScriptAddress(), merr.Description)
			merr.Description = desc
			return nil, merr
		}
		return nil, maybeConvertDbError(err)
	}

//为基于特定地址类型的新托管地址
//类型上。
	managedAddr, err := s.rowInterfaceToManaged(ns, rowInterface)
	if err != nil {
		return nil, err
	}

//缓存并返回新的托管地址。
	s.addrs[addrKey(managedAddr.Address().ScriptAddress())] = managedAddr

	return managedAddr, nil
}

//existsaddress返回传递的地址是否为
//地址管理器。
//
//必须在保持管理器锁可读取的情况下调用此函数。
func (s *ScopedKeyManager) existsAddress(ns walletdb.ReadBucket, addressID []byte) bool {
//首先检查内存映射，因为它比数据库访问快。
	if _, ok := s.addrs[addrKey(addressID)]; ok {
		return true
	}

//如果上面没有找到，请检查数据库。
	return existsAddress(ns, &s.scope, addressID)
}

//
//地址管理器。托管地址与中传递的地址不同
//它还可能包含需要签名的额外信息
//交易记录，如支付至Pubkey的关联私钥和
//支付到PubKey哈希地址和与之关联的脚本
//付费脚本哈希地址。
func (s *ScopedKeyManager) Address(ns walletdb.ReadBucket,
	address btcutil.Address) (ManagedAddress, error) {

//如果我们正在访问
//地址是pkh或sh。如果我们通过pk
//地址，将pk转换为pkh地址，以便我们可以从
//加法器地图和数据库。
	if pka, ok := address.(*btcutil.AddressPubKey); ok {
		address = pka.AddressPubKeyHash()
	}

//从缓存返回地址（如果可用）。
//
//注意：这里不使用延迟锁，因为写锁是
//如果查找失败，则需要。
	s.mtx.RLock()
	if ma, ok := s.addrs[addrKey(address.ScriptAddress())]; ok {
		s.mtx.RUnlock()
		return ma, nil
	}
	s.mtx.RUnlock()

	s.mtx.Lock()
	defer s.mtx.Unlock()

//尝试从数据库加载地址。
	return s.loadAndCacheAddress(ns, address)
}

//addraccount返回给定地址所属的帐户。
func (s *ScopedKeyManager) AddrAccount(ns walletdb.ReadBucket,
	address btcutil.Address) (uint32, error) {

	account, err := fetchAddrAccount(ns, &s.scope, address.ScriptAddress())
	if err != nil {
		return 0, maybeConvertDbError(err)
	}

	return account, nil
}

//NextAddresses返回从
//由内部标志指示的分支。
//
//调用此函数时必须保留管理器锁以进行写入。
func (s *ScopedKeyManager) nextAddresses(ns walletdb.ReadWriteBucket,
	account uint32, numAddresses uint32, internal bool) ([]ManagedAddress, error) {

//下一个地址只能为
//已创建。
	acctInfo, err := s.loadAccountInfo(ns, account)
	if err != nil {
		return nil, err
	}

//根据地址管理器是否
//被锁定。
	acctKey := acctInfo.acctKeyPub
	if !s.rootManager.IsLocked() {
		acctKey = acctInfo.acctKeyPriv
	}

//根据是否为
//内部地址。
	branchNum, nextIndex := ExternalBranch, acctInfo.nextExternalIndex
	if internal {
		branchNum = InternalBranch
		nextIndex = acctInfo.nextInternalIndex
	}

	addrType := s.addrSchema.ExternalAddrType
	if internal {
		addrType = s.addrSchema.InternalAddrType
	}

//确保请求的地址数不超过最大值
//允许用于此帐户。
	if numAddresses > MaxAddressesPerAccount || nextIndex+numAddresses >
		MaxAddressesPerAccount {
		str := fmt.Sprintf("%d new addresses would exceed the maximum "+
			"allowed number of addresses per account of %d",
			numAddresses, MaxAddressesPerAccount)
		return nil, managerError(ErrTooManyAddresses, str, nil)
	}

//派生适当的分支键，并确保完成后将其归零。
	branchKey, err := acctKey.Child(branchNum)
	if err != nil {
		str := fmt.Sprintf("failed to derive extended key branch %d",
			branchNum)
		return nil, managerError(ErrKeyChain, str, err)
	}
defer branchKey.Zero() //完成后确保分支键归零。

//创建请求的地址数并跟踪索引
//每一个。
	addressInfo := make([]*unlockDeriveInfo, 0, numAddresses)
	for i := uint32(0); i < numAddresses; i++ {
//一个特定的孩子
//无效，请使用循环派生下一个有效子级。
		var nextKey *hdkeychain.ExtendedKey
		for {
//派生外部链分支中的下一个子级。
			key, err := branchKey.Child(nextIndex)
			if err != nil {
//当此特定子级无效时，跳到
//下一个索引。
				if err == hdkeychain.ErrInvalidChild {
					nextIndex++
					continue
				}

				str := fmt.Sprintf("failed to generate child %d",
					nextIndex)
				return nil, managerError(ErrKeyChain, str, err)
			}
			key.SetNet(s.rootManager.chainParams)

			nextIndex++
			nextKey = key
			break
		}

//既然我们知道这个密钥可以使用，我们将创建
//正确的派生路径，以便可以获得这些信息
//给来访者。
		derivationPath := DerivationPath{
			Account: account,
			Branch:  branchNum,
			Index:   nextIndex - 1,
		}

//
//密钥取决于生成的密钥是否是私有的。
//另外，在创建托管地址后将下一个键归零
//从它。
		addr, err := newManagedAddressFromExtKey(
			s, derivationPath, nextKey, addrType,
		)
		if err != nil {
			return nil, err
		}
		if internal {
			addr.internal = true
		}
		managedAddr := addr
		nextKey.Zero()

		info := unlockDeriveInfo{
			managedAddr: managedAddr,
			branch:      branchNum,
			index:       nextIndex - 1,
		}
		addressInfo = append(addressInfo, &info)
	}

//现在所有地址都已成功生成，请更新
//单个事务中的数据库。
	for _, info := range addressInfo {
		ma := info.managedAddr
		addressID := ma.Address().ScriptAddress()

		switch a := ma.(type) {
		case *managedAddress:
			err := putChainedAddress(
				ns, &s.scope, addressID, account, ssFull,
				info.branch, info.index, adtChain,
			)
			if err != nil {
				return nil, maybeConvertDbError(err)
			}
		case *scriptAddress:
			encryptedHash, err := s.rootManager.cryptoKeyPub.Encrypt(a.AddrHash())
			if err != nil {
				str := fmt.Sprintf("failed to encrypt script hash %x",
					a.AddrHash())
				return nil, managerError(ErrCrypto, str, err)
			}

			err = putScriptAddress(
				ns, &s.scope, a.AddrHash(), ImportedAddrAccount,
				ssNone, encryptedHash, a.scriptEncrypted,
			)
			if err != nil {
				return nil, maybeConvertDbError(err)
			}
		}
	}

//最后更新下一个地址跟踪并将地址添加到
//新生成的地址成功后的缓存
//添加到数据库。
	managedAddresses := make([]ManagedAddress, 0, len(addressInfo))
	for _, info := range addressInfo {
		ma := info.managedAddr
		s.addrs[addrKey(ma.Address().ScriptAddress())] = ma

//将新的托管地址添加到
//当地址管理器为
//下一步解锁。
		if s.rootManager.IsLocked() && !s.rootManager.WatchOnly() {
			s.deriveOnUnlock = append(s.deriveOnUnlock, info)
		}

		managedAddresses = append(managedAddresses, ma)
	}

//设置跟踪的最后一个地址和下一个地址。
	ma := addressInfo[len(addressInfo)-1].managedAddr
	if internal {
		acctInfo.nextInternalIndex = nextIndex
		acctInfo.lastInternalAddr = ma
	} else {
		acctInfo.nextExternalIndex = nextIndex
		acctInfo.lastExternalAddr = ma
	}

	return managedAddresses, nil
}

//
//为内部或外部分支派生。如果孩子在
//LastIndex无效，此方法将继续执行，直到下一个有效子级
//找到了。如果方法未能正确扩展地址，则返回错误。
//达到请求的索引。
//
//调用此函数时必须保留管理器锁以进行写入。
func (s *ScopedKeyManager) extendAddresses(ns walletdb.ReadWriteBucket,
	account uint32, lastIndex uint32, internal bool) error {

//下一个地址只能为
//已创建。
	acctInfo, err := s.loadAccountInfo(ns, account)
	if err != nil {
		return err
	}

//根据地址管理器是否
//被锁定。
	acctKey := acctInfo.acctKeyPub
	if !s.rootManager.IsLocked() {
		acctKey = acctInfo.acctKeyPriv
	}

//根据是否为
//内部地址。
	branchNum, nextIndex := ExternalBranch, acctInfo.nextExternalIndex
	if internal {
		branchNum = InternalBranch
		nextIndex = acctInfo.nextInternalIndex
	}

	addrType := s.addrSchema.ExternalAddrType
	if internal {
		addrType = s.addrSchema.InternalAddrType
	}

//如果请求的最后一个索引已经低于下一个索引，我们
//可以早点回来。
	if lastIndex < nextIndex {
		return nil
	}

//确保请求的地址数不超过最大值
//允许用于此帐户。
	if lastIndex > MaxAddressesPerAccount {
		str := fmt.Sprintf("last index %d would exceed the maximum "+
			"allowed number of addresses per account of %d",
			lastIndex, MaxAddressesPerAccount)
		return managerError(ErrTooManyAddresses, str, nil)
	}

//派生适当的分支键，并确保完成后将其归零。
	branchKey, err := acctKey.Child(branchNum)
	if err != nil {
		str := fmt.Sprintf("failed to derive extended key branch %d",
			branchNum)
		return managerError(ErrKeyChain, str, err)
	}
defer branchKey.Zero() //完成后确保分支键归零。

//从该分支的nextindex开始，派生到
//包括请求的最后一个索引。如果一个无效的孩子是
//检测到，此循环将继续派生，直到找到下一个
//后续索引。
	addressInfo := make([]*unlockDeriveInfo, 0, lastIndex-nextIndex)
	for nextIndex <= lastIndex {
//一个特定的孩子
//无效，请使用循环派生下一个有效子级。
		var nextKey *hdkeychain.ExtendedKey
		for {
//派生外部链分支中的下一个子级。
			key, err := branchKey.Child(nextIndex)
			if err != nil {
//当此特定子级无效时，跳到
//下一个索引。
				if err == hdkeychain.ErrInvalidChild {
					nextIndex++
					continue
				}

				str := fmt.Sprintf("failed to generate child %d",
					nextIndex)
				return managerError(ErrKeyChain, str, err)
			}
			key.SetNet(s.rootManager.chainParams)

			nextIndex++
			nextKey = key
			break
		}

//既然我们知道这个密钥可以使用，我们将创建
//正确的派生路径，以便可以获得这些信息
//给来访者。
		derivationPath := DerivationPath{
			Account: account,
			Branch:  branchNum,
			Index:   nextIndex - 1,
		}

//基于公用或专用创建新的托管地址
//密钥取决于生成的密钥是否是私有的。
//另外，在创建托管地址后将下一个键归零
//从它。
		addr, err := newManagedAddressFromExtKey(
			s, derivationPath, nextKey, addrType,
		)
		if err != nil {
			return err
		}
		if internal {
			addr.internal = true
		}
		managedAddr := addr
		nextKey.Zero()

		info := unlockDeriveInfo{
			managedAddr: managedAddr,
			branch:      branchNum,
			index:       nextIndex - 1,
		}
		addressInfo = append(addressInfo, &info)
	}

//现在所有地址都已成功生成，请更新
//单个事务中的数据库。
	for _, info := range addressInfo {
		ma := info.managedAddr
		addressID := ma.Address().ScriptAddress()

		switch a := ma.(type) {
		case *managedAddress:
			err := putChainedAddress(
				ns, &s.scope, addressID, account, ssFull,
				info.branch, info.index, adtChain,
			)
			if err != nil {
				return maybeConvertDbError(err)
			}
		case *scriptAddress:
			encryptedHash, err := s.rootManager.cryptoKeyPub.Encrypt(a.AddrHash())
			if err != nil {
				str := fmt.Sprintf("failed to encrypt script hash %x",
					a.AddrHash())
				return managerError(ErrCrypto, str, err)
			}

			err = putScriptAddress(
				ns, &s.scope, a.AddrHash(), ImportedAddrAccount,
				ssNone, encryptedHash, a.scriptEncrypted,
			)
			if err != nil {
				return maybeConvertDbError(err)
			}
		}
	}

//最后更新下一个地址跟踪并将地址添加到
//新生成的地址成功后的缓存
//添加到数据库。
	for _, info := range addressInfo {
		ma := info.managedAddr
		s.addrs[addrKey(ma.Address().ScriptAddress())] = ma

//将新的托管地址添加到
//当地址管理器为
//下一步解锁。
		if s.rootManager.IsLocked() && !s.rootManager.WatchOnly() {
			s.deriveOnUnlock = append(s.deriveOnUnlock, info)
		}
	}

//设置跟踪的最后一个地址和下一个地址。
	ma := addressInfo[len(addressInfo)-1].managedAddr
	if internal {
		acctInfo.nextInternalIndex = nextIndex
		acctInfo.lastInternalAddr = ma
	} else {
		acctInfo.nextExternalIndex = nextIndex
		acctInfo.lastExternalAddr = ma
	}

	return nil
}

//nextexternaladdress返回指定数量的下一个链接地址
//供地址管理器外部使用的。
func (s *ScopedKeyManager) NextExternalAddresses(ns walletdb.ReadWriteBucket,
	account uint32, numAddresses uint32) ([]ManagedAddress, error) {

//
	if account > MaxAccountNum {
		err := managerError(ErrAccountNumTooHigh, errAcctTooHigh, nil)
		return nil, err
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.nextAddresses(ns, account, numAddresses, false)
}

//NextInternalAddresses返回指定数量的下一个链接地址
//用于内部使用，如从地址管理器更改。
func (s *ScopedKeyManager) NextInternalAddresses(ns walletdb.ReadWriteBucket,
	account uint32, numAddresses uint32) ([]ManagedAddress, error) {

//强制使用最大帐号。
	if account > MaxAccountNum {
		err := managerError(ErrAccountNumTooHigh, errAcctTooHigh, nil)
		return nil, err
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.nextAddresses(ns, account, numAddresses, true)
}

//extendexternaladdresses确保通过
//最后一个索引被导出并存储在钱包中。这是用来确保
//钱包的持续状态赶上了一个被发现的外部孩子
//在恢复过程中。
func (s *ScopedKeyManager) ExtendExternalAddresses(ns walletdb.ReadWriteBucket,
	account uint32, lastIndex uint32) error {

	if account > MaxAccountNum {
		err := managerError(ErrAccountNumTooHigh, errAcctTooHigh, nil)
		return err
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.extendAddresses(ns, account, lastIndex, false)
}

//扩展的内部地址确保所有有效的内部密钥通过
//最后一个索引被导出并存储在钱包中。这是用来确保
//钱包的持续状态赶上了一个被发现的内部孩子
//在恢复过程中。
func (s *ScopedKeyManager) ExtendInternalAddresses(ns walletdb.ReadWriteBucket,
	account uint32, lastIndex uint32) error {

	if account > MaxAccountNum {
		err := managerError(ErrAccountNumTooHigh, errAcctTooHigh, nil)
		return err
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.extendAddresses(ns, account, lastIndex, true)
}

//LastExternalAddress返回最近请求的链接外部地址
//调用给定帐户的NextInternalAddress的地址。第一
//如果没有，将返回帐户的外部地址
//以前请求过。
//
//如果提供的帐号大于
//大于maxaccountnum常量，或者没有用于
//通过账户。返回的任何其他错误通常都是意外的。
func (s *ScopedKeyManager) LastExternalAddress(ns walletdb.ReadBucket,
	account uint32) (ManagedAddress, error) {

//强制使用最大帐号。
	if account > MaxAccountNum {
		err := managerError(ErrAccountNumTooHigh, errAcctTooHigh, nil)
		return nil, err
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

//加载已传递帐户的帐户信息。它通常是
//已缓存，但如果没有，则将从数据库加载。
	acctInfo, err := s.loadAccountInfo(ns, account)
	if err != nil {
		return nil, err
	}

	if acctInfo.nextExternalIndex > 0 {
		return acctInfo.lastExternalAddr, nil
	}

	return nil, managerError(ErrAddressNotFound, "no previous external address", nil)
}

//
//为给定帐户调用NextInternalAddress的地址。第一
//如果没有内部地址，将返回帐户的内部地址。
//以前请求过。
//
//如果提供的帐号大于
//大于maxaccountnum常量，或者没有用于
//通过账户。返回的任何其他错误通常都是意外的。
func (s *ScopedKeyManager) LastInternalAddress(ns walletdb.ReadBucket,
	account uint32) (ManagedAddress, error) {

//强制使用最大帐号。
	if account > MaxAccountNum {
		err := managerError(ErrAccountNumTooHigh, errAcctTooHigh, nil)
		return nil, err
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

//加载已传递帐户的帐户信息。它通常是
//已缓存，但如果没有，则将从数据库加载。
	acctInfo, err := s.loadAccountInfo(ns, account)
	if err != nil {
		return nil, err
	}

	if acctInfo.nextInternalIndex > 0 {
		return acctInfo.lastInternalAddr, nil
	}

	return nil, managerError(ErrAddressNotFound, "no previous internal address", nil)
}

//newrawaccount为作用域管理器创建一个新帐户。这种方法
//与NewAccount方法不同的是，此方法采用
//直接使用数字*，而不是为帐户使用字符串名称，然后
//将其映射到下一个最高帐号。
func (s *ScopedKeyManager) NewRawAccount(ns walletdb.ReadWriteBucket, number uint32) error {
	if s.rootManager.WatchOnly() {
		return managerError(ErrWatchingOnly, errWatchingOnly, nil)
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.rootManager.IsLocked() {
		return managerError(ErrLocked, errLocked, nil)
	}

//因为这是一个特别的帐户，可能不遵循我们的正常线性
//派生，我们将基于
//帐号。
	name := fmt.Sprintf("act:%v", number)
	return s.newAccount(ns, number, name)
}

//new account创建并返回存储在管理器中的新帐户
//给定的帐户名。如果已经存在同名帐户，
//将返回errDuplicateCount。因为创建新帐户需要
//访问CoinType密钥（从中派生扩展帐户密钥）；
//它要求经理解锁。
func (s *ScopedKeyManager) NewAccount(ns walletdb.ReadWriteBucket, name string) (uint32, error) {
	if s.rootManager.WatchOnly() {
		return 0, managerError(ErrWatchingOnly, errWatchingOnly, nil)
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.rootManager.IsLocked() {
		return 0, managerError(ErrLocked, errLocked, nil)
	}

//
//事务获取最新帐号以生成下一个
//帐号
	account, err := fetchLastAccount(ns, &s.scope)
	if err != nil {
		return 0, err
	}
	account++

//验证名称后，我们将为新帐户创建一个新帐户。
//连续账户。
	if err := s.newAccount(ns, account, name); err != nil {
		return 0, err
	}

	return account, nil
}

//new account是一个助手函数，它派生一个新的精确帐号，
//
//数据库。
//
//注意：必须在保持管理器锁以便写入的情况下调用此函数。
func (s *ScopedKeyManager) newAccount(ns walletdb.ReadWriteBucket,
	account uint32, name string) error {

//验证帐户名。
	if err := ValidateAccountName(name); err != nil {
		return err
	}

//检查同名帐户是否不存在
	_, err := s.lookupAccount(ns, name)
	if err == nil {
		str := fmt.Sprintf("account with the same name already exists")
		return managerError(ErrDuplicateAccount, str, err)
	}

//
//扩展键
	_, coinTypePrivEnc, err := fetchCoinTypeKeys(ns, &s.scope)
	if err != nil {
		return err
	}

//解密coinType密钥。
	serializedKeyPriv, err := s.rootManager.cryptoKeyPriv.Decrypt(coinTypePrivEnc)
	if err != nil {
		str := fmt.Sprintf("failed to decrypt cointype serialized private key")
		return managerError(ErrLocked, str, err)
	}
	coinTypeKeyPriv, err := hdkeychain.NewKeyFromString(string(serializedKeyPriv))
	zero.Bytes(serializedKeyPriv)
	if err != nil {
		str := fmt.Sprintf("failed to create cointype extended private key")
		return managerError(ErrKeyChain, str, err)
	}

//
	acctKeyPriv, err := deriveAccountKey(coinTypeKeyPriv, account)
	coinTypeKeyPriv.Zero()
	if err != nil {
		str := "failed to convert private key for account"
		return managerError(ErrKeyChain, str, err)
	}
	acctKeyPub, err := acctKeyPriv.Neuter()
	if err != nil {
		str := "failed to convert public key for account"
		return managerError(ErrKeyChain, str, err)
	}

//使用关联的加密密钥加密默认帐户密钥。
	acctPubEnc, err := s.rootManager.cryptoKeyPub.Encrypt(
		[]byte(acctKeyPub.String()),
	)
	if err != nil {
		str := "failed to  encrypt public key for account"
		return managerError(ErrCrypto, str, err)
	}
	acctPrivEnc, err := s.rootManager.cryptoKeyPriv.Encrypt(
		[]byte(acctKeyPriv.String()),
	)
	if err != nil {
		str := "failed to encrypt private key for account"
		return managerError(ErrCrypto, str, err)
	}

//我们有加密的帐户扩展密钥，因此将它们保存到
//数据库
	err = putAccountInfo(
		ns, &s.scope, account, acctPubEnc, acctPrivEnc, 0, 0, name,
	)
	if err != nil {
		return err
	}

//保存上一个帐户元数据
	return putLastAccount(ns, &s.scope, account)
}

//重命名帐户根据给定的
//具有给定名称的帐号。如果帐户名相同
//已经存在，将返回errDuplicateCount。
func (s *ScopedKeyManager) RenameAccount(ns walletdb.ReadWriteBucket,
	account uint32, name string) error {

	s.mtx.Lock()
	defer s.mtx.Unlock()

//确保未重命名保留帐户。
	if isReservedAccountNum(account) {
		str := "reserved account cannot be renamed"
		return managerError(ErrInvalidAccount, str, nil)
	}

//检查具有新名称的帐户是否不存在
	_, err := s.lookupAccount(ns, name)
	if err == nil {
		str := fmt.Sprintf("account with the same name already exists")
		return managerError(ErrDuplicateAccount, str, err)
	}

//验证帐户名
	if err := ValidateAccountName(name); err != nil {
		return err
	}

	rowInterface, err := fetchAccountInfo(ns, &s.scope, account)
	if err != nil {
		return err
	}

//确保帐户类型为默认帐户。
	row, ok := rowInterface.(*dbDefaultAccountRow)
	if !ok {
		str := fmt.Sprintf("unsupported account type %T", row)
		err = managerError(ErrDatabase, str, nil)
	}

//从帐户ID索引中删除旧名称密钥。
	if err = deleteAccountIDIndex(ns, &s.scope, account); err != nil {
		return err
	}

//从帐户名索引中删除旧名称密钥。
	if err = deleteAccountNameIndex(ns, &s.scope, row.name); err != nil {
		return err
	}
	err = putAccountInfo(
		ns, &s.scope, account, row.pubKeyEncrypted,
		row.privKeyEncrypted, row.nextExternalIndex,
		row.nextInternalIndex, name,
	)
	if err != nil {
		return err
	}

//使用新名称（如果已缓存）和数据库更新内存中的帐户信息
//
	if err == nil {
		if acctInfo, ok := s.acctInfo[account]; ok {
			acctInfo.acctName = name
		}
	}

	return err
}

//importprivatekey将WIF私钥导入地址管理器。这个
//导入的地址是使用压缩或未压缩的
//序列化公钥，取决于wif的compresspubkey bool。
//
//所有导入的地址将是
//importedAddAccount常量。
//
//注意：当地址管理器仅监视时，私钥本身将
//不存储或不可用，因为它是私有数据。相反，只有
//将存储公钥。这意味着它是最重要的，私钥是
//保留在其他地方，因为只监视地址管理器将永远无法访问
//对它。
//
//如果地址管理器被锁定而没有被锁定，此函数将返回一个错误。
//只看，或不看同一个网络的密钥试图导入。
//如果地址已经存在，它也会返回一个错误。任何其他
//返回的错误通常是意外的。
func (s *ScopedKeyManager) ImportPrivateKey(ns walletdb.ReadWriteBucket,
	wif *btcutil.WIF, bs *BlockStamp) (ManagedPubKeyAddress, error) {

//确保地址用于网络地址管理器
//与关联。
	if !wif.IsForNet(s.rootManager.chainParams) {
		str := fmt.Sprintf("private key is not for the same network the "+
			"address manager is configured for (%s)",
			s.rootManager.chainParams.Name)
		return nil, managerError(ErrWrongNet, str, nil)
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

//必须解锁管理器才能加密导入的私钥。
	if s.rootManager.IsLocked() && !s.rootManager.WatchOnly() {
		return nil, managerError(ErrLocked, errLocked, nil)
	}

//防止重复。
	serializedPubKey := wif.SerializePubKey()
	pubKeyHash := btcutil.Hash160(serializedPubKey)
	alreadyExists := s.existsAddress(ns, pubKeyHash)
	if alreadyExists {
		str := fmt.Sprintf("address for public key %x already exists",
			serializedPubKey)
		return nil, managerError(ErrDuplicateAddress, str, nil)
	}

//
	encryptedPubKey, err := s.rootManager.cryptoKeyPub.Encrypt(
		serializedPubKey,
	)
	if err != nil {
		str := fmt.Sprintf("failed to encrypt public key for %x",
			serializedPubKey)
		return nil, managerError(ErrCrypto, str, err)
	}

//当不是仅监视地址管理器时加密私钥。
	var encryptedPrivKey []byte
	if !s.rootManager.WatchOnly() {
		privKeyBytes := wif.PrivKey.Serialize()
		encryptedPrivKey, err = s.rootManager.cryptoKeyPriv.Encrypt(privKeyBytes)
		zero.Bytes(privKeyBytes)
		if err != nil {
			str := fmt.Sprintf("failed to encrypt private key for %x",
				serializedPubKey)
			return nil, managerError(ErrCrypto, str, err)
		}
	}

//当新导入的地址
//在当前的之前。
	s.rootManager.mtx.Lock()
	updateStartBlock := bs.Height < s.rootManager.syncState.startBlock.Height
	s.rootManager.mtx.Unlock()

//将新导入的地址保存到数据库并更新开始块（如果
//需要）在单个事务中。
	err = putImportedAddress(
		ns, &s.scope, pubKeyHash, ImportedAddrAccount, ssNone,
		encryptedPubKey, encryptedPrivKey,
	)
	if err != nil {
		return nil, err
	}

	if updateStartBlock {
		err := putStartBlock(ns, bs)
		if err != nil {
			return nil, err
		}
	}

//现在数据库已更新，请在
//如果需要的话，也要有记忆。
	if updateStartBlock {
		s.rootManager.mtx.Lock()
		s.rootManager.syncState.startBlock = *bs
		s.rootManager.mtx.Unlock()
	}

//导入的密钥的完整派生路径不完整，因为
//不知道它是如何派生出来的。
	importedDerivationPath := DerivationPath{
		Account: ImportedAddrAccount,
	}

//基于导入的地址创建新的托管地址。
	var managedAddr *managedAddress
	if !s.rootManager.WatchOnly() {
		managedAddr, err = newManagedAddress(
			s, importedDerivationPath, wif.PrivKey,
			wif.CompressPubKey, s.addrSchema.ExternalAddrType,
		)
	} else {
		pubKey := (*btcec.PublicKey)(&wif.PrivKey.PublicKey)
		managedAddr, err = newManagedAddressWithoutPrivKey(
			s, importedDerivationPath, pubKey, wif.CompressPubKey,
			s.addrSchema.ExternalAddrType,
		)
	}
	if err != nil {
		return nil, err
	}
	managedAddr.imported = true

//将新的托管地址添加到最近地址的缓存中，然后
//把它还给我。
	s.addrs[addrKey(managedAddr.Address().ScriptAddress())] = managedAddr
	return managedAddr, nil
}

//importscript将用户提供的脚本导入地址管理器。这个
//导入的脚本将充当付费脚本哈希地址。
//
//所有导入的脚本地址都将是由
//importedAddAccount常量。
//
//当地址管理器只监视时，脚本本身不会
//存储或可用，因为它被视为私有数据。
//
//如果地址管理器被锁定而没有被锁定，此函数将返回一个错误。
//
//通常是意外的。
func (s *ScopedKeyManager) ImportScript(ns walletdb.ReadWriteBucket,
	script []byte, bs *BlockStamp) (ManagedScriptAddress, error) {

	s.mtx.Lock()
	defer s.mtx.Unlock()

//必须解锁管理器才能加密导入的脚本。
	if s.rootManager.IsLocked() && !s.rootManager.WatchOnly() {
		return nil, managerError(ErrLocked, errLocked, nil)
	}

//防止重复。
	scriptHash := btcutil.Hash160(script)
	alreadyExists := s.existsAddress(ns, scriptHash)
	if alreadyExists {
		str := fmt.Sprintf("address for script hash %x already exists",
			scriptHash)
		return nil, managerError(ErrDuplicateAddress, str, nil)
	}

//使用加密公钥加密脚本哈希，因此
//仅当地址管理器被锁定或正在监视时才可访问。
	encryptedHash, err := s.rootManager.cryptoKeyPub.Encrypt(scriptHash)
	if err != nil {
		str := fmt.Sprintf("failed to encrypt script hash %x",
			scriptHash)
		return nil, managerError(ErrCrypto, str, err)
	}

//使用加密脚本加密用于存储在数据库中的脚本
//当不是一个只监视地址管理器时。
	var encryptedScript []byte
	if !s.rootManager.WatchOnly() {
		encryptedScript, err = s.rootManager.cryptoKeyScript.Encrypt(
			script,
		)
		if err != nil {
			str := fmt.Sprintf("failed to encrypt script for %x",
				scriptHash)
			return nil, managerError(ErrCrypto, str, err)
		}
	}

//当新导入的地址
//在当前的之前。
	updateStartBlock := false
	s.rootManager.mtx.Lock()
	if bs.Height < s.rootManager.syncState.startBlock.Height {
		updateStartBlock = true
	}
	s.rootManager.mtx.Unlock()

//将新导入的地址保存到数据库并更新开始块（如果
//需要）在单个事务中。
	err = putScriptAddress(
		ns, &s.scope, scriptHash, ImportedAddrAccount, ssNone,
		encryptedHash, encryptedScript,
	)
	if err != nil {
		return nil, maybeConvertDbError(err)
	}

	if updateStartBlock {
		err := putStartBlock(ns, bs)
		if err != nil {
			return nil, maybeConvertDbError(err)
		}
	}

//现在数据库已更新，请在
//如果需要的话，也要有记忆。
	if updateStartBlock {
		s.rootManager.mtx.Lock()
		s.rootManager.syncState.startBlock = *bs
		s.rootManager.mtx.Unlock()
	}

//基于导入的脚本创建新的托管地址。也，
//如果不是只监视地址管理器，请复制脚本
//因为它将在锁和调用方传递的脚本上被清除
//不应从呼叫者下方清除。
	scriptAddr, err := newScriptAddress(
		s, ImportedAddrAccount, scriptHash, encryptedScript,
	)
	if err != nil {
		return nil, err
	}
	if !s.rootManager.WatchOnly() {
		scriptAddr.scriptCT = make([]byte, len(script))
		copy(scriptAddr.scriptCT, script)
	}

//将新的托管地址添加到最近地址的缓存中，然后
//把它还给我。
	s.addrs[addrKey(scriptHash)] = scriptAddr
	return scriptAddr, nil
}

//lookupaccount加载存储在管理器中的给定
//帐户名
//
//必须在保持管理器锁可读取的情况下调用此函数。
func (s *ScopedKeyManager) lookupAccount(ns walletdb.ReadBucket, name string) (uint32, error) {
	return fetchAccountByName(ns, &s.scope, name)
}

//lookupaccount加载存储在管理器中的给定
//帐户名
func (s *ScopedKeyManager) LookupAccount(ns walletdb.ReadBucket, name string) (uint32, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return s.lookupAccount(ns, name)
}

//如果所提供的地址ID被标记为已使用，则fetchused返回true。
func (s *ScopedKeyManager) fetchUsed(ns walletdb.ReadBucket,
	addressID []byte) bool {

	return fetchAddressUsed(ns, &s.scope, addressID)
}

//markused更新所提供地址的已用标志。
func (s *ScopedKeyManager) MarkUsed(ns walletdb.ReadWriteBucket,
	address btcutil.Address) error {

	addressID := address.ScriptAddress()
	err := markAddressUsed(ns, &s.scope, addressID)
	if err != nil {
		return maybeConvertDbError(err)
	}

//清除可能对已用地址有过时条目的缓存
	s.mtx.Lock()
	delete(s.addrs, addrKey(addressID))
	s.mtx.Unlock()
	return nil
}

//chainParams返回此地址管理器的链参数。
func (s *ScopedKeyManager) ChainParams() *chaincfg.Params {
//注意：这里不需要mutex，因为net字段没有更改
//在创建管理器实例之后。

	return s.rootManager.chainParams
}

//account name返回存储在
//经理。
func (s *ScopedKeyManager) AccountName(ns walletdb.ReadBucket, account uint32) (string, error) {
	return fetchAccountName(ns, &s.scope, account)
}

//foreachaccount使用存储在
//
func (s *ScopedKeyManager) ForEachAccount(ns walletdb.ReadBucket,
	fn func(account uint32) error) error {

	return forEachAccount(ns, &s.scope, fn)
}

//
func (s *ScopedKeyManager) LastAccount(ns walletdb.ReadBucket) (uint32, error) {
	return fetchLastAccount(ns, &s.scope)
}

//foreachaccountaddress使用
//假定帐户存储在管理器中，出错时提前中断。
func (s *ScopedKeyManager) ForEachAccountAddress(ns walletdb.ReadBucket,
	account uint32, fn func(maddr ManagedAddress) error) error {

	s.mtx.Lock()
	defer s.mtx.Unlock()

	addrFn := func(rowInterface interface{}) error {
		managedAddr, err := s.rowInterfaceToManaged(ns, rowInterface)
		if err != nil {
			return err
		}
		return fn(managedAddr)
	}
	err := forEachAccountAddress(ns, &s.scope, account, addrFn)
	if err != nil {
		return maybeConvertDbError(err)
	}

	return nil
}

//foreachActiveAccountAddress调用给定函数，每个函数都处于活动状态
//存储在管理器中的给定帐户的地址，出错时提前中断。
//
//todo（tuxcanfly）：实际上只返回活动地址
func (s *ScopedKeyManager) ForEachActiveAccountAddress(ns walletdb.ReadBucket, account uint32,
	fn func(maddr ManagedAddress) error) error {

	return s.ForEachAccountAddress(ns, account, fn)
}

//foreachActiveAddress使用每个活动地址调用给定函数
//存储在管理器中，出错时提前中断。
func (s *ScopedKeyManager) ForEachActiveAddress(ns walletdb.ReadBucket,
	fn func(addr btcutil.Address) error) error {

	s.mtx.Lock()
	defer s.mtx.Unlock()

	addrFn := func(rowInterface interface{}) error {
		managedAddr, err := s.rowInterfaceToManaged(ns, rowInterface)
		if err != nil {
			return err
		}
		return fn(managedAddr.Address())
	}

	err := forEachActiveAddress(ns, &s.scope, addrFn)
	if err != nil {
		return maybeConvertDbError(err)
	}

	return nil
}
