
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

package votingpool

import (
	"fmt"
	"sort"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/btcsuite/btcwallet/internal/zero"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/walletdb"
)

const (
	minSeriesPubKeys = 3
//当前版本是用于新创建的系列的版本。
	CurrentVersion = 1
)

//分支是用于表示序列中分支编号的类型。
type Branch uint32

//索引是用于表示序列中索引号的类型。
type Index uint32

//seriesdata表示给定池的序列。
type SeriesData struct {
	version uint32
//序列是否处于活动状态。这是序列化/反序列化的，但
//现在没有办法停用一个系列。
	active bool
//A.K.A.“M”中的“M/N需要签名”。
	reqSigs     uint32
	publicKeys  []*hdkeychain.ExtendedKey
	privateKeys []*hdkeychain.ExtendedKey
}

//池表示公证服务器安全地
//存储和结算客户的加密货币存款和兑换
//有效提款。有关排列工作方式的详细信息，请参见
//http://opentransactions.org/wiki/index.php？标题=类别：投票池
type Pool struct {
	ID           []byte
	seriesLookup map[uint32]*SeriesData
	manager      *waddrmgr.Manager
}

//pool address表示投票池p2sh地址，由
//使用给定的
//分支/索引并构造一个m-of-n多SIG脚本。
type PoolAddress interface {
	SeriesID() uint32
	Branch() Branch
	Index() Index
}

type poolAddress struct {
	pool     *Pool
	addr     btcutil.Address
	script   []byte
	seriesID uint32
	branch   Branch
	index    Index
}

//ChangeAddress是用于事务更改的投票池地址
//输出。所有更改地址都具有分支==0。
type ChangeAddress struct {
	*poolAddress
}

//
//可用于取款。
type WithdrawalAddress struct {
	*poolAddress
}

//创建在数据库中创建具有给定ID的新条目
//并返回表示它的池。
func Create(ns walletdb.ReadWriteBucket, m *waddrmgr.Manager, poolID []byte) (*Pool, error) {
	err := putPool(ns, poolID)
	if err != nil {
		str := fmt.Sprintf("unable to add voting pool %v to db", poolID)
		return nil, newError(ErrPoolAlreadyExists, str, err)
	}
	return newPool(m, poolID), nil
}

//LOAD获取具有给定ID的数据库中的条目并返回池
//代表它。
func Load(ns walletdb.ReadBucket, m *waddrmgr.Manager, poolID []byte) (*Pool, error) {
	if !existsPool(ns, poolID) {
		str := fmt.Sprintf("unable to find voting pool %v in db", poolID)
		return nil, newError(ErrPoolNotExists, str, nil)
	}
	p := newPool(m, poolID)
	if err := p.LoadAllSeries(ns); err != nil {
		return nil, err
	}
	return p, nil
}

//NewPool创建新的池实例。
func newPool(m *waddrmgr.Manager, poolID []byte) *Pool {
	return &Pool{
		ID:           poolID,
		seriesLookup: make(map[uint32]*SeriesData),
		manager:      m,
	}
}

//loadAndGetDepositscript生成并返回给定seriesID的存款脚本，
//poolid标识的池的分支和索引。
func LoadAndGetDepositScript(ns walletdb.ReadBucket, m *waddrmgr.Manager, poolID string, seriesID uint32, branch Branch, index Index) ([]byte, error) {
	pid := []byte(poolID)
	p, err := Load(ns, m, pid)
	if err != nil {
		return nil, err
	}
	script, err := p.DepositScript(seriesID, branch, index)
	if err != nil {
		return nil, err
	}
	return script, nil
}

//loadAndCreateSeries使用给定的ID加载池，如果没有，则创建一个新的池
//存在，然后创建并返回具有给定seriesid、rawpubkey的序列
//和ReqSIGS。请参阅CreateSeries以了解对rawSubkeys和reqSigs强制执行的约束。
func LoadAndCreateSeries(ns walletdb.ReadWriteBucket, m *waddrmgr.Manager, version uint32,
	poolID string, seriesID, reqSigs uint32, rawPubKeys []string) error {
	pid := []byte(poolID)
	p, err := Load(ns, m, pid)
	if err != nil {
		vpErr := err.(Error)
		if vpErr.ErrorCode == ErrPoolNotExists {
			p, err = Create(ns, m, pid)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return p.CreateSeries(ns, version, seriesID, reqSigs, rawPubKeys)
}

//loadAndReplaceSeries使用给定的ID加载投票池并调用replaceSeries，
//将给定的序列ID、公钥和reqsigs传递给它。
func LoadAndReplaceSeries(ns walletdb.ReadWriteBucket, m *waddrmgr.Manager, version uint32,
	poolID string, seriesID, reqSigs uint32, rawPubKeys []string) error {
	pid := []byte(poolID)
	p, err := Load(ns, m, pid)
	if err != nil {
		return err
	}
	return p.ReplaceSeries(ns, version, seriesID, reqSigs, rawPubKeys)
}

//loadandempowerseries使用给定的ID加载投票池，并调用authorizeseries，
//将给定的序列ID和私钥传递给它。
func LoadAndEmpowerSeries(ns walletdb.ReadWriteBucket, m *waddrmgr.Manager,
	poolID string, seriesID uint32, rawPrivKey string) error {
	pid := []byte(poolID)
	pool, err := Load(ns, m, pid)
	if err != nil {
		return err
	}
	return pool.EmpowerSeries(ns, seriesID, rawPrivKey)
}

//序列返回具有给定ID的序列，否则返回nil
//存在。
func (p *Pool) Series(seriesID uint32) *SeriesData {
	series, exists := p.seriesLookup[seriesID]
	if !exists {
		return nil
	}
	return series
}

//manager返回此池使用的waddrmgr.manager。
func (p *Pool) Manager() *waddrmgr.Manager {
	return p.manager
}

//saveseriestodisk将给定的序列ID和数据存储在数据库中，
//首先加密公钥/私钥扩展密钥。
//
//必须在池的管理器未锁定的情况下调用此方法。
func (p *Pool) saveSeriesToDisk(ns walletdb.ReadWriteBucket, seriesID uint32, data *SeriesData) error {
	var err error
	encryptedPubKeys := make([][]byte, len(data.publicKeys))
	for i, pubKey := range data.publicKeys {
		encryptedPubKeys[i], err = p.manager.Encrypt(
			waddrmgr.CKTPublic, []byte(pubKey.String()))
		if err != nil {
			str := fmt.Sprintf("key %v failed encryption", pubKey)
			return newError(ErrCrypto, str, err)
		}
	}
	encryptedPrivKeys := make([][]byte, len(data.privateKeys))
	for i, privKey := range data.privateKeys {
		if privKey == nil {
			encryptedPrivKeys[i] = nil
		} else {
			encryptedPrivKeys[i], err = p.manager.Encrypt(
				waddrmgr.CKTPrivate, []byte(privKey.String()))
		}
		if err != nil {
			str := fmt.Sprintf("key %v failed encryption", privKey)
			return newError(ErrCrypto, str, err)
		}
	}

	err = putSeries(ns, p.ID, data.version, seriesID, data.active,
		data.reqSigs, encryptedPubKeys, encryptedPrivKeys)
	if err != nil {
		str := fmt.Sprintf("cannot put series #%d into db", seriesID)
		return newError(ErrSeriesSerialization, str, err)
	}
	return nil
}

//CanonicalKeyOrder将按规范返回输入的副本
//有序的，被定义为词典编纂的。
func CanonicalKeyOrder(keys []string) []string {
	orderedKeys := make([]string, len(keys))
	copy(orderedKeys, keys)
	sort.Sort(sort.StringSlice(orderedKeys))
	return orderedKeys
}

//将给定的字符串切片转换为扩展键切片，
//检查它们是否都是有效的公钥（而不是私钥），
//没有重复的。
func convertAndValidatePubKeys(rawPubKeys []string) ([]*hdkeychain.ExtendedKey, error) {
	seenKeys := make(map[string]bool)
	keys := make([]*hdkeychain.ExtendedKey, len(rawPubKeys))
	for i, rawPubKey := range rawPubKeys {
		if _, seen := seenKeys[rawPubKey]; seen {
			str := fmt.Sprintf("duplicated public key: %v", rawPubKey)
			return nil, newError(ErrKeyDuplicate, str, nil)
		}
		seenKeys[rawPubKey] = true

		key, err := hdkeychain.NewKeyFromString(rawPubKey)
		if err != nil {
			str := fmt.Sprintf("invalid extended public key %v", rawPubKey)
			return nil, newError(ErrKeyChain, str, err)
		}

		if key.IsPrivate() {
			str := fmt.Sprintf("private keys not accepted: %v", rawPubKey)
			return nil, newError(ErrKeyIsPrivate, str, nil)
		}
		keys[i] = key
	}
	return keys, nil
}

//putseries使用给定的参数创建新的series数据，并对
//给定的公钥（使用CanonicalKeyOrder），验证和转换它们
//到hdkeychain.extendedkeys，将其保存到磁盘并添加到此投票中
//池的序列查找映射。它还确保inrawpubkeys至少
//minseriespubkeys items和reqsigs不大于中的项数
//iNRAWPUBKEY。
//
//必须在池的管理器未锁定的情况下调用此方法。
func (p *Pool) putSeries(ns walletdb.ReadWriteBucket, version, seriesID, reqSigs uint32, inRawPubKeys []string) error {
	if len(inRawPubKeys) < minSeriesPubKeys {
		str := fmt.Sprintf("need at least %d public keys to create a series", minSeriesPubKeys)
		return newError(ErrTooFewPublicKeys, str, nil)
	}

	if reqSigs > uint32(len(inRawPubKeys)) {
		str := fmt.Sprintf(
			"the number of required signatures cannot be more than the number of keys")
		return newError(ErrTooManyReqSignatures, str, nil)
	}

	rawPubKeys := CanonicalKeyOrder(inRawPubKeys)

	keys, err := convertAndValidatePubKeys(rawPubKeys)
	if err != nil {
		return err
	}

	data := &SeriesData{
		version:     version,
		active:      false,
		reqSigs:     reqSigs,
		publicKeys:  keys,
		privateKeys: make([]*hdkeychain.ExtendedKey, len(keys)),
	}

	err = p.saveSeriesToDisk(ns, seriesID, data)
	if err != nil {
		return err
	}
	p.seriesLookup[seriesID] = data
	return nil
}

//CreateSeries将创建并返回一个新的不存在的序列。
//
//-seriesid必须大于或等于1；
//-rawSubkey必须包含三个或更多公钥；
//-reqsigs必须小于或等于rawpubkeys中的公钥数。
func (p *Pool) CreateSeries(ns walletdb.ReadWriteBucket, version, seriesID, reqSigs uint32, rawPubKeys []string) error {
	if seriesID == 0 {
		return newError(ErrSeriesIDInvalid, "series ID cannot be 0", nil)
	}

	if series := p.Series(seriesID); series != nil {
		str := fmt.Sprintf("series #%d already exists", seriesID)
		return newError(ErrSeriesAlreadyExists, str, nil)
	}

	if seriesID != 1 {
		if _, ok := p.seriesLookup[seriesID-1]; !ok {
			str := fmt.Sprintf("series #%d cannot be created because series #%d does not exist",
				seriesID, seriesID-1)
			return newError(ErrSeriesIDNotSequential, str, nil)
		}
	}

	return p.putSeries(ns, version, seriesID, reqSigs, rawPubKeys)
}

//activateSeries将具有给定ID的序列标记为active。
func (p *Pool) ActivateSeries(ns walletdb.ReadWriteBucket, seriesID uint32) error {
	series := p.Series(seriesID)
	if series == nil {
		str := fmt.Sprintf("series #%d does not exist, cannot activate it", seriesID)
		return newError(ErrSeriesNotExists, str, nil)
	}
	series.active = true
	err := p.saveSeriesToDisk(ns, seriesID, series)
	if err != nil {
		return err
	}
	p.seriesLookup[seriesID] = series
	return nil
}

//ReplaceSeries将替换现有的系列。
//
//-rawSubkey必须包含三个或更多公钥
//-reqsigs必须小于或等于rawpubkeys中的公钥数。
func (p *Pool) ReplaceSeries(ns walletdb.ReadWriteBucket, version, seriesID, reqSigs uint32, rawPubKeys []string) error {
	series := p.Series(seriesID)
	if series == nil {
		str := fmt.Sprintf("series #%d does not exist, cannot replace it", seriesID)
		return newError(ErrSeriesNotExists, str, nil)
	}

	if series.IsEmpowered() {
		str := fmt.Sprintf("series #%d has private keys and cannot be replaced", seriesID)
		return newError(ErrSeriesAlreadyEmpowered, str, nil)
	}

	return p.putSeries(ns, version, seriesID, reqSigs, rawPubKeys)
}

//decryptextendedkey使用manager.decrypt（）来解密加密的字节片并返回
//表示它的扩展（公共或私有）密钥。
//
//
func (p *Pool) decryptExtendedKey(keyType waddrmgr.CryptoKeyType, encrypted []byte) (*hdkeychain.ExtendedKey, error) {
	decrypted, err := p.manager.Decrypt(keyType, encrypted)
	if err != nil {
		str := fmt.Sprintf("cannot decrypt key %v", encrypted)
		return nil, newError(ErrCrypto, str, err)
	}
	result, err := hdkeychain.NewKeyFromString(string(decrypted))
	zero.Bytes(decrypted)
	if err != nil {
		str := fmt.Sprintf("cannot get key from string %v", decrypted)
		return nil, newError(ErrKeyChain, str, err)
	}
	return result, nil
}

//
//
//并返回它们。
//
//必须在池的管理器未锁定的情况下调用此函数。
func validateAndDecryptKeys(rawPubKeys, rawPrivKeys [][]byte, p *Pool) (pubKeys, privKeys []*hdkeychain.ExtendedKey, err error) {
	pubKeys = make([]*hdkeychain.ExtendedKey, len(rawPubKeys))
	privKeys = make([]*hdkeychain.ExtendedKey, len(rawPrivKeys))
	if len(pubKeys) != len(privKeys) {
		return nil, nil, newError(ErrKeysPrivatePublicMismatch,
			"the pub key and priv key arrays should have the same number of elements",
			nil)
	}

	for i, encryptedPub := range rawPubKeys {
		pubKey, err := p.decryptExtendedKey(waddrmgr.CKTPublic, encryptedPub)
		if err != nil {
			return nil, nil, err
		}
		pubKeys[i] = pubKey

		encryptedPriv := rawPrivKeys[i]
		var privKey *hdkeychain.ExtendedKey
		if encryptedPriv == nil {
			privKey = nil
		} else {
			privKey, err = p.decryptExtendedKey(waddrmgr.CKTPrivate, encryptedPriv)
			if err != nil {
				return nil, nil, err
			}
		}
		privKeys[i] = privKey

		if privKey != nil {
			checkPubKey, err := privKey.Neuter()
			if err != nil {
				str := fmt.Sprintf("cannot neuter key %v", privKey)
				return nil, nil, newError(ErrKeyNeuter, str, err)
			}
			if pubKey.String() != checkPubKey.String() {
				str := fmt.Sprintf("public key %v different than expected %v",
					pubKey, checkPubKey)
				return nil, nil, newError(ErrKeyMismatch, str, nil)
			}
		}
	}
	return pubKeys, privKeys, nil
}

//loadallseries获取所有序列（解密它们的公共和私有
//扩展键），并填充
//序列查找地图。如果有任何私人扩展密钥
//一个系列，它还将确保它们具有匹配的扩展公钥
//在这一系列中。
//
//必须在池的管理器未锁定的情况下调用此方法。
//Fixme：我们应该能够摆脱这个问题（以及loadallseries/series查找）
//通过使series（）直接从数据库加载序列数据。
func (p *Pool) LoadAllSeries(ns walletdb.ReadBucket) error {
	series, err := loadAllSeries(ns, p.ID)
	if err != nil {
		return err
	}
	for id, series := range series {
		pubKeys, privKeys, err := validateAndDecryptKeys(
			series.pubKeysEncrypted, series.privKeysEncrypted, p)
		if err != nil {
			return err
		}
		p.seriesLookup[id] = &SeriesData{
			publicKeys:  pubKeys,
			privateKeys: privKeys,
			reqSigs:     series.reqSigs,
		}
	}
	return nil
}

//根据分支编号更改公钥的顺序。
//考虑到三个公共键ABC，这意味着：
//
//-分支机构1:ABC（一键优先）
//
//
func branchOrder(pks []*hdkeychain.ExtendedKey, branch Branch) ([]*hdkeychain.ExtendedKey, error) {
	if pks == nil {
//
//
		return nil, newError(ErrInvalidValue, "pks cannot be nil", nil)
	}

	if branch > Branch(len(pks)) {
		return nil, newError(
			ErrInvalidBranch, "branch number is bigger than number of public keys", nil)
	}

	if branch == 0 {
		numKeys := len(pks)
		res := make([]*hdkeychain.ExtendedKey, numKeys)
		copy(res, pks)
//反向PK
		for i, j := 0, numKeys-1; i < j; i, j = i+1, j-1 {
			res[i], res[j] = res[j], res[i]
		}
		return res, nil
	}

	tmp := make([]*hdkeychain.ExtendedKey, len(pks))
	tmp[0] = pks[branch-1]
	j := 1
	for i := 0; i < len(pks); i++ {
		if i != int(branch-1) {
			tmp[j] = pks[i]
			j++
		}
	}
	return tmp, nil
}

//
//
func (p *Pool) DepositScriptAddress(seriesID uint32, branch Branch, index Index) (btcutil.Address, error) {
	script, err := p.DepositScript(seriesID, branch, index)
	if err != nil {
		return nil, err
	}
	return p.addressFor(script)
}

func (p *Pool) addressFor(script []byte) (btcutil.Address, error) {
	scriptHash := btcutil.Hash160(script)
	return btcutil.NewAddressScriptHashFromHash(scriptHash, p.manager.ChainParams())
}

//depositscript构造并返回一个多签名赎回脚本，其中
//属于该系列的公钥的某个数字（series.reqsigs）。
//要使事务成功，需要使用给定的ID签名。
func (p *Pool) DepositScript(seriesID uint32, branch Branch, index Index) ([]byte, error) {
	series := p.Series(seriesID)
	if series == nil {
		str := fmt.Sprintf("series #%d does not exist", seriesID)
		return nil, newError(ErrSeriesNotExists, str, nil)
	}

	pubKeys, err := branchOrder(series.publicKeys, branch)
	if err != nil {
		return nil, err
	}

	pks := make([]*btcutil.AddressPubKey, len(pubKeys))
	for i, key := range pubKeys {
		child, err := key.Child(uint32(index))
//TODO:实现获取下一个索引，直到找到有效的索引为止，
//如果存在hdKeyChain.errInvalidChild。
		if err != nil {
			str := fmt.Sprintf("child #%d for this pubkey %d does not exist", index, i)
			return nil, newError(ErrKeyChain, str, err)
		}
		pubkey, err := child.ECPubKey()
		if err != nil {
			str := fmt.Sprintf("child #%d for this pubkey %d does not exist", index, i)
			return nil, newError(ErrKeyChain, str, err)
		}
		pks[i], err = btcutil.NewAddressPubKey(pubkey.SerializeCompressed(),
			p.manager.ChainParams())
		if err != nil {
			str := fmt.Sprintf(
				"child #%d for this pubkey %d could not be converted to an address",
				index, i)
			return nil, newError(ErrKeyChain, str, err)
		}
	}

	script, err := txscript.MultiSigScript(pks, int(series.reqSigs))
	if err != nil {
		str := fmt.Sprintf("error while making multisig script hash, %d", len(pks))
		return nil, newError(ErrScriptCreation, str, err)
	}

	return script, nil
}

//ChangeAddress返回给定序列ID的新投票池地址，以及
//第0个分支上的索引（保留用于更改地址）。系列
//具有给定ID的必须是活动的。
func (p *Pool) ChangeAddress(seriesID uint32, index Index) (*ChangeAddress, error) {
	series := p.Series(seriesID)
	if series == nil {
		return nil, newError(ErrSeriesNotExists,
			fmt.Sprintf("series %d does not exist", seriesID), nil)
	}
	if !series.active {
		str := fmt.Sprintf("ChangeAddress must be on active series; series #%d is not", seriesID)
		return nil, newError(ErrSeriesNotActive, str, nil)
	}

	script, err := p.DepositScript(seriesID, Branch(0), index)
	if err != nil {
		return nil, err
	}
	pAddr, err := p.poolAddress(seriesID, Branch(0), index, script)
	if err != nil {
		return nil, err
	}
	return &ChangeAddress{poolAddress: pAddr}, nil
}

//提款地址查询地址管理器中的p2sh地址
//使用给定的序列/分支/索引生成的兑换脚本并使用
//以填充返回的提款地址。这样做是因为我们
//应该只从以前使用的地址中提取，而且因为
//处理取款时，我们可能会在大量地址上迭代，并且
//重新生成所有脚本的兑换脚本太贵了。
//必须在管理器未锁定的情况下调用此方法。
func (p *Pool) WithdrawalAddress(ns, addrmgrNs walletdb.ReadBucket, seriesID uint32, branch Branch, index Index) (
	*WithdrawalAddress, error) {
//TODO:确保给定的序列是热的。
	addr, err := p.getUsedAddr(ns, addrmgrNs, seriesID, branch, index)
	if err != nil {
		return nil, err
	}
	if addr == nil {
		str := fmt.Sprintf("cannot withdraw from unused addr (series: %d, branch: %d, index: %d)",
			seriesID, branch, index)
		return nil, newError(ErrWithdrawFromUnusedAddr, str, nil)
	}
	script, err := addr.Script()
	if err != nil {
		return nil, err
	}
	pAddr, err := p.poolAddress(seriesID, branch, index, script)
	if err != nil {
		return nil, err
	}
	return &WithdrawalAddress{poolAddress: pAddr}, nil
}

func (p *Pool) poolAddress(seriesID uint32, branch Branch, index Index, script []byte) (
	*poolAddress, error) {
	addr, err := p.addressFor(script)
	if err != nil {
		return nil, err
	}
	return &poolAddress{
			pool: p, seriesID: seriesID, branch: branch, index: index, addr: addr,
			script: script},
		nil
}

//授权系列将给定的扩展私钥（原始格式）添加到
//与给定的ID串联，从而允许其签署存款/取款
//脚本。具有给定ID的序列必须存在，密钥必须是有效的
//专用扩展密钥，必须与系列的扩展公钥之一匹配。
//
//必须在池的管理器未锁定的情况下调用此方法。
func (p *Pool) EmpowerSeries(ns walletdb.ReadWriteBucket, seriesID uint32, rawPrivKey string) error {
//确保此系列存在
	series := p.Series(seriesID)
	if series == nil {
		str := fmt.Sprintf("series %d does not exist for this voting pool",
			seriesID)
		return newError(ErrSeriesNotExists, str, nil)
	}

//检查私钥是否有效。
	privKey, err := hdkeychain.NewKeyFromString(rawPrivKey)
	if err != nil {
		str := fmt.Sprintf("invalid extended private key %v", rawPrivKey)
		return newError(ErrKeyChain, str, err)
	}
	if !privKey.IsPrivate() {
		str := fmt.Sprintf(
			"to empower a series you need the extended private key, not an extended public key %v",
			privKey)
		return newError(ErrKeyIsPublic, str, err)
	}

	pubKey, err := privKey.Neuter()
	if err != nil {
		str := fmt.Sprintf("invalid extended private key %v, can't convert to public key",
			rawPrivKey)
		return newError(ErrKeyNeuter, str, err)
	}

	lookingFor := pubKey.String()
	found := false

//确保私钥在序列中具有相应的公钥，
//能够赋予它权力。
	for i, publicKey := range series.publicKeys {
		if publicKey.String() == lookingFor {
			found = true
			series.privateKeys[i] = privKey
		}
	}

	if !found {
		str := fmt.Sprintf(
			"private Key does not have a corresponding public key in this series")
		return newError(ErrKeysPrivatePublicMismatch, str, nil)
	}

	if err = p.saveSeriesToDisk(ns, seriesID, series); err != nil {
		return err
	}

	return nil
}

//ensureuseddr确保在我们的已用地址db中为给定的
//序列ID、分支和指定索引之前的所有索引。必须用
//经理解锁了。
func (p *Pool) EnsureUsedAddr(ns, addrmgrNs walletdb.ReadWriteBucket, seriesID uint32, branch Branch, index Index) error {
	lastIdx, err := p.highestUsedIndexFor(ns, seriesID, branch)
	if err != nil {
		return err
	}
	if lastIdx == 0 {
//当没有使用的地址时，highestusedindexfor（）返回0
//给定seriesid/分支，因此我们这样做是为了确保
//索引＝0。
		if err := p.addUsedAddr(ns, addrmgrNs, seriesID, branch, lastIdx); err != nil {
			return err
		}
	}
	lastIdx++
	for lastIdx <= index {
		if err := p.addUsedAddr(ns, addrmgrNs, seriesID, branch, lastIdx); err != nil {
			return err
		}
		lastIdx++
	}
	return nil
}

//addusedaddr为给定的seriesid/branch/index创建一个存款脚本，
//确保将其导入地址管理器并最终添加脚本
//散列到我们使用的地址数据库。必须在经理解锁的情况下调用。
func (p *Pool) addUsedAddr(ns, addrmgrNs walletdb.ReadWriteBucket, seriesID uint32, branch Branch, index Index) error {
	script, err := p.DepositScript(seriesID, branch, index)
	if err != nil {
		return err
	}

//首先确保地址管理器有我们的脚本。那样就没有办法了
//将其保存在已用地址数据库中，但不保存在地址管理器中。
//TODO:决定我们希望addr管理器重新扫描多远，并设置
//木版印的高度是这样的。
	manager, err := p.manager.FetchScopedKeyManager(waddrmgr.KeyScopeBIP0044)
	if err != nil {
		return err
	}
	_, err = manager.ImportScript(addrmgrNs, script, &waddrmgr.BlockStamp{})
	if err != nil && err.(waddrmgr.ManagerError).ErrorCode != waddrmgr.ErrDuplicateAddress {
		return err
	}

	encryptedHash, err := p.manager.Encrypt(waddrmgr.CKTPublic, btcutil.Hash160(script))
	if err != nil {
		return newError(ErrCrypto, "failed to encrypt script hash", err)
	}
	err = putUsedAddrHash(ns, p.ID, seriesID, branch, index, encryptedHash)
	if err != nil {
		return newError(ErrDatabase, "failed to store used addr script hash", err)
	}

	return nil
}

//getusedaddr从中获取给定序列、分支和索引的脚本哈希
//已用地址db并使用它查找managedscriptAddress
//来自地址管理器。必须在经理解锁的情况下调用。
func (p *Pool) getUsedAddr(ns, addrmgrNs walletdb.ReadBucket, seriesID uint32, branch Branch, index Index) (
	waddrmgr.ManagedScriptAddress, error) {

	mgr := p.manager
	encryptedHash := getUsedAddrHash(ns, p.ID, seriesID, branch, index)
	if encryptedHash == nil {
		return nil, nil
	}
	hash, err := p.manager.Decrypt(waddrmgr.CKTPublic, encryptedHash)
	if err != nil {
		return nil, newError(ErrCrypto, "failed to decrypt stored script hash", err)
	}
	addr, err := btcutil.NewAddressScriptHashFromHash(hash, mgr.ChainParams())
	if err != nil {
		return nil, newError(ErrInvalidScriptHash, "failed to parse script hash", err)
	}
	mAddr, err := mgr.Address(addrmgrNs, addr)
	if err != nil {
		return nil, err
	}
	return mAddr.(waddrmgr.ManagedScriptAddress), nil
}

//highestusedindexfor返回此池已用地址的最高索引
//具有给定的序列ID和分支。如果没有使用则返回0
//具有给定序列ID和分支的地址。
func (p *Pool) highestUsedIndexFor(ns walletdb.ReadBucket, seriesID uint32, branch Branch) (Index, error) {
	return getMaxUsedIdx(ns, p.ID, seriesID, branch)
}

//string返回基础比特币支付地址的字符串编码。
func (a *poolAddress) String() string {
	return a.addr.EncodeAddress()
}

func (a *poolAddress) addrIdentifier() string {
	return fmt.Sprintf("PoolAddress seriesID:%d, branch:%d, index:%d", a.seriesID, a.branch,
		a.index)
}

func (a *poolAddress) redeemScript() []byte {
	return a.script
}

func (a *poolAddress) series() *SeriesData {
	return a.pool.Series(a.seriesID)
}

func (a *poolAddress) SeriesID() uint32 {
	return a.seriesID
}

func (a *poolAddress) Branch() Branch {
	return a.branch
}

func (a *poolAddress) Index() Index {
	return a.index
}

//如果此系列被授权（即，如果它具有
//至少加载了一个私钥）。
func (s *SeriesData) IsEmpowered() bool {
	for _, key := range s.privateKeys {
		if key != nil {
			return true
		}
	}
	return false
}

func (s *SeriesData) getPrivKeyFor(pubKey *hdkeychain.ExtendedKey) (*hdkeychain.ExtendedKey, error) {
	for i, key := range s.publicKeys {
		if key.String() == pubKey.String() {
			return s.privateKeys[i], nil
		}
	}
	return nil, newError(ErrUnknownPubKey, fmt.Sprintf("unknown public key '%s'",
		pubKey.String()), nil)
}
