
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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/ripemd160"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/internal/legacy/rename"
)

const (
	Filename = "wallet.bin"

//kdf输出的长度（字节）。
	kdfOutputBytes = 32

//可以表示大小的注释的最大长度（字节）
//作为UIT16。
	maxCommentLen = (1 << 16) - 1
)

const (
	defaultKdfComputeTime = 0.25
	defaultKdfMaxMem      = 32 * 1024 * 1024
)

//处理密钥存储时可能出错。
var (
	ErrAddressNotFound  = errors.New("address not found")
	ErrAlreadyEncrypted = errors.New("private key is already encrypted")
	ErrChecksumMismatch = errors.New("checksum mismatch")
	ErrDuplicate        = errors.New("duplicate key or address")
	ErrMalformedEntry   = errors.New("malformed entry")
	ErrWatchingOnly     = errors.New("keystore is watching-only")
	ErrLocked           = errors.New("keystore is locked")
	ErrWrongPassphrase  = errors.New("wrong passphrase")
)

var fileID = [8]byte{0xba, 'W', 'A', 'L', 'L', 'E', 'T', 0x00}

type entryHeader byte

const (
	addrCommentHeader entryHeader = 1 << iota
	txCommentHeader
	deletedHeader
	scriptHeader
	addrHeader entryHeader = 0
)

//我们要使用binary read和binarywrite而不是binary.read
//和binary.write，因为binary包中的那些不返回
//实际写入或读取的字节数。我们得回去
//此值正确支持io.readerFrom和io.writerto
//接口。
func binaryRead(r io.Reader, order binary.ByteOrder, data interface{}) (n int64, err error) {
	var read int
	buf := make([]byte, binary.Size(data))
	if read, err = io.ReadFull(r, buf); err != nil {
		return int64(read), err
	}
	return int64(read), binary.Read(bytes.NewBuffer(buf), order, data)
}

//请参见BinaryRead（）的注释。
func binaryWrite(w io.Writer, order binary.ByteOrder, data interface{}) (n int64, err error) {
	buf := bytes.Buffer{}
	if err = binary.Write(&buf, order, data); err != nil {
		return 0, err
	}

	written, err := w.Write(buf.Bytes())
	return int64(written), err
}

//pubKeyFromPrivkey基于
//32字节的私钥。如果压缩，返回的pubkey为33字节，
//或65字节（如果未压缩）。
func pubkeyFromPrivkey(privkey []byte, compress bool) (pubkey []byte) {
	_, pk := btcec.PrivKeyFromBytes(btcec.S256(), privkey)

	if compress {
		return pk.SerializeCompressed()
	}
	return pk.SerializeUncompressed()
}

func keyOneIter(passphrase, salt []byte, memReqts uint64) []byte {
	saltedpass := append(passphrase, salt...)
	lutbl := make([]byte, memReqts)

//查找表的种子
	seed := sha512.Sum512(saltedpass)
	copy(lutbl[:sha512.Size], seed[:])

	for nByte := 0; nByte < (int(memReqts) - sha512.Size); nByte += sha512.Size {
		hash := sha512.Sum512(lutbl[nByte : nByte+sha512.Size])
		copy(lutbl[nByte+sha512.Size:nByte+2*sha512.Size], hash[:])
	}

	x := lutbl[cap(lutbl)-sha512.Size:]

	seqCt := uint32(memReqts / sha512.Size)
	nLookups := seqCt / 2
	for i := uint32(0); i < nLookups; i++ {
//军械库忽略了这里的结尾。我们假设是LE。
		newIdx := binary.LittleEndian.Uint32(x[cap(x)-4:]) % seqCt

//newidx的哈希结果索引
		vIdx := newIdx * sha512.Size
		v := lutbl[vIdx : vIdx+sha512.Size]

//XOR哈希X和哈希V
		for j := 0; j < sha512.Size; j++ {
			x[j] ^= v[j]
		}

//将新哈希保存到X
		hash := sha512.Sum512(x)
		copy(x, hash[:])
	}

	return x[:kdfOutputBytes]
}

//kdf实现了armory使用的密钥派生函数
//基于Colin Percival论文中描述的romix算法
//“通过顺序内存硬函数进行更强大的密钥派生”
//（http://www.tarsnap.com/scrypt/scrypt.pdf）。
func kdf(passphrase []byte, params *kdfParameters) []byte {
	masterKey := passphrase
	for i := uint32(0); i < params.nIter; i++ {
		masterKey = keyOneIter(masterKey, params.salt[:], params.mem)
	}
	return masterKey
}

func pad(size int, b []byte) []byte {
//如果输入超过预期大小，请防止可能出现的恐慌。
	if len(b) > size {
		size = len(b)
	}

	p := make([]byte, size)
	copy(p[size-len(b):], b)
	return p
}

//chainedprivkey使用
//以前的地址和链码。私钥和链码必须为32
//字节长，pubkey可以是33或65字节。
func chainedPrivKey(privkey, pubkey, chaincode []byte) ([]byte, error) {
	if len(privkey) != 32 {
		return nil, fmt.Errorf("invalid privkey length %d (must be 32)",
			len(privkey))
	}
	if len(chaincode) != 32 {
		return nil, fmt.Errorf("invalid chaincode length %d (must be 32)",
			len(chaincode))
	}
	switch n := len(pubkey); n {
	case btcec.PubKeyBytesLenUncompressed, btcec.PubKeyBytesLenCompressed:
//正确长度
	default:
		return nil, fmt.Errorf("invalid pubkey length %d", n)
	}

	xorbytes := make([]byte, 32)
	chainMod := chainhash.DoubleHashB(pubkey)
	for i := range xorbytes {
		xorbytes[i] = chainMod[i] ^ chaincode[i]
	}
	chainXor := new(big.Int).SetBytes(xorbytes)
	privint := new(big.Int).SetBytes(privkey)

	t := new(big.Int).Mul(chainXor, privint)
	b := t.Mod(t, btcec.S256().N).Bytes()
	return pad(32, b), nil
}

//chainedPubKey使用
//以前的公钥和链码。pubkey必须是33或65字节，并且
//链码必须是32字节长。
func chainedPubKey(pubkey, chaincode []byte) ([]byte, error) {
	var compressed bool
	switch n := len(pubkey); n {
	case btcec.PubKeyBytesLenUncompressed:
		compressed = false
	case btcec.PubKeyBytesLenCompressed:
		compressed = true
	default:
//序列化的pubkey长度不正确
		return nil, fmt.Errorf("invalid pubkey length %d", n)
	}
	if len(chaincode) != 32 {
		return nil, fmt.Errorf("invalid chaincode length %d (must be 32)",
			len(chaincode))
	}

	xorbytes := make([]byte, 32)
	chainMod := chainhash.DoubleHashB(pubkey)
	for i := range xorbytes {
		xorbytes[i] = chainMod[i] ^ chaincode[i]
	}

	oldPk, err := btcec.ParsePubKey(pubkey, btcec.S256())
	if err != nil {
		return nil, err
	}
	newX, newY := btcec.S256().ScalarMult(oldPk.X, oldPk.Y, xorbytes)
	if err != nil {
		return nil, err
	}
	newPk := &btcec.PublicKey{
		Curve: btcec.S256(),
		X:     newX,
		Y:     newY,
	}

	if compressed {
		return newPk.SerializeCompressed(), nil
	}
	return newPk.SerializeUncompressed(), nil
}

type version struct {
	major         byte
	minor         byte
	bugfix        byte
	autoincrement byte
}

//强制该版本满足io.readerFrom和
//IO.WRITERTO接口。
var _ io.ReaderFrom = &version{}
var _ io.WriterTo = &version{}

//readerFromVersion是一个io.readerFrom和io.writerTo，
//可以指定要读取的任何特定密钥存储文件格式
//取决于密钥存储文件版本。
type readerFromVersion interface {
	readFromVersion(version, io.Reader) (int64, error)
	io.WriterTo
}

func (v version) String() string {
	str := fmt.Sprintf("%d.%d", v.major, v.minor)
	if v.bugfix != 0x00 || v.autoincrement != 0x00 {
		str += fmt.Sprintf(".%d", v.bugfix)
	}
	if v.autoincrement != 0x00 {
		str += fmt.Sprintf(".%d", v.autoincrement)
	}
	return str
}

func (v version) Uint32() uint32 {
	return uint32(v.major)<<6 | uint32(v.minor)<<4 | uint32(v.bugfix)<<2 | uint32(v.autoincrement)
}

func (v *version) ReadFrom(r io.Reader) (int64, error) {
//读取版本的4个字节。
	var versBytes [4]byte
	n, err := io.ReadFull(r, versBytes[:])
	if err != nil {
		return int64(n), err
	}
	v.major = versBytes[0]
	v.minor = versBytes[1]
	v.bugfix = versBytes[2]
	v.autoincrement = versBytes[3]
	return int64(n), nil
}

func (v *version) WriteTo(w io.Writer) (int64, error) {
//为版本写入4个字节。
	versBytes := []byte{
		v.major,
		v.minor,
		v.bugfix,
		v.autoincrement,
	}
	n, err := w.Write(versBytes)
	return int64(n), err
}

//它返回v是否是早于v2的版本。
func (v version) LT(v2 version) bool {
	switch {
	case v.major < v2.major:
		return true

	case v.minor < v2.minor:
		return true

	case v.bugfix < v2.bugfix:
		return true

	case v.autoincrement < v2.autoincrement:
		return true

	default:
		return false
	}
}

//eq返回v2是否等于v。
func (v version) EQ(v2 version) bool {
	switch {
	case v.major != v2.major:
		return false

	case v.minor != v2.minor:
		return false

	case v.bugfix != v2.bugfix:
		return false

	case v.autoincrement != v2.autoincrement:
		return false

	default:
		return true
	}
}

//gt返回v是否比v2更高版本。
func (v version) GT(v2 version) bool {
	switch {
	case v.major > v2.major:
		return true

	case v.minor > v2.minor:
		return true

	case v.bugfix > v2.bugfix:
		return true

	case v.autoincrement > v2.autoincrement:
		return true

	default:
		return false
	}
}

//各种版本。
var (
//军械库是军械库使用的最新版本。
	VersArmory = version{1, 35, 0, 0}

//Vers20LastBlocks是密钥存储文件现在保存的版本
//最近看到的20个块散列。
	Vers20LastBlocks = version{1, 36, 0, 0}

//versusetNeedsPrivKeyFlag是错误修复版本，其中
//CreatePrivKeyNextUnlock地址标志已正确取消设置
//在解锁后创建并加密其私钥。
//否则，重新创建私钥将太早
//在地址链中，由于已加密
//加密地址。在此之前或在此之前的密钥存储版本
//版本包括一个特殊情况，允许复制
//加密。
	VersUnsetNeedsPrivkeyFlag = version{1, 36, 1, 0}

//VersCurrent是当前密钥存储文件版本。
	VersCurrent = VersUnsetNeedsPrivkeyFlag
)

type varEntries struct {
	store   *Store
	entries []io.WriterTo
}

func (v *varEntries) WriteTo(w io.Writer) (n int64, err error) {
	ss := v.entries

	var written int64
	for _, s := range ss {
		var err error
		if written, err = s.WriteTo(w); err != nil {
			return n + written, err
		}
		n += written
	}
	return n, nil
}

func (v *varEntries) ReadFrom(r io.Reader) (n int64, err error) {
	var read int64

//删除以前的所有条目。
	v.entries = nil
	wts := v.entries

//继续读取条目，直到达到EOF。
	for {
		var header entryHeader
		if read, err = binaryRead(r, binary.LittleEndian, &header); err != nil {
//这里的EOF不是一个错误。
			if err == io.EOF {
				return n + read, nil
			}
			return n + read, err
		}
		n += read

		var wt io.WriterTo
		switch header {
		case addrHeader:
			var entry addrEntry
			entry.addr.store = v.store
			if read, err = entry.ReadFrom(r); err != nil {
				return n + read, err
			}
			n += read
			wt = &entry
		case scriptHeader:
			var entry scriptEntry
			entry.script.store = v.store
			if read, err = entry.ReadFrom(r); err != nil {
				return n + read, err
			}
			n += read
			wt = &entry
		default:
			return n, fmt.Errorf("unknown entry header: %d", uint8(header))
		}
		if wt != nil {
			wts = append(wts, wt)
			v.entries = wts
		}
	}
}

//密钥存储使用自定义网络参数类型，因此它可以是IO.readerFrom。
//由于密钥存储的序列化方式和顺序以及
//地址读取需要密钥存储的网络参数、设置和
//未知密钥存储网络上的错误必须发生在读取本身上，而不是
//在事实之后。这确实是一个黑客，但是在
//地平线，我没有太大的动机来清理这个。
type netParams chaincfg.Params

func (net *netParams) ReadFrom(r io.Reader) (int64, error) {
	var buf [4]byte
	uint32Bytes := buf[:4]

	n, err := io.ReadFull(r, uint32Bytes)
	n64 := int64(n)
	if err != nil {
		return n64, err
	}

	switch wire.BitcoinNet(binary.LittleEndian.Uint32(uint32Bytes)) {
	case wire.MainNet:
		*net = (netParams)(chaincfg.MainNetParams)
	case wire.TestNet3:
		*net = (netParams)(chaincfg.TestNet3Params)
	case wire.SimNet:
		*net = (netParams)(chaincfg.SimNetParams)
	default:
		return n64, errors.New("unknown network")
	}
	return n64, nil
}

func (net *netParams) WriteTo(w io.Writer) (int64, error) {
	var buf [4]byte
	uint32Bytes := buf[:4]

	binary.LittleEndian.PutUint32(uint32Bytes, uint32(net.Net))
	n, err := w.Write(uint32Bytes)
	n64 := int64(n)
	return n64, err
}

//字符串化字节片用作映射查找键。
type addressKey string
type transactionHashKey string

type comment []byte

func getAddressKey(addr btcutil.Address) addressKey {
	return addressKey(addr.ScriptAddress())
}

//store表示内存中的密钥存储。它实现了
//IO.readerFrom和IO.writerto要读取的接口和
//写入任何类型的字节流，包括文件。
type Store struct {
//TODO:使用原子操作进行脏操作，以便读卡器锁定
//
	dirty bool
	path  string
	dir   string
	file  string

	mtx          sync.RWMutex
	vers         version
	net          *netParams
	flags        walletFlags
	createDate   int64
	name         [32]byte
	desc         [256]byte
	highestUsed  int64
	kdfParams    kdfParameters
	keyGenerator btcAddress

//这些是非标准的，可以容纳在
//根地址和附加条目。
	recent recentBlocks

	addrMap map[addressKey]walletAddress

//此结构中的其余字段未序列化。
	passphrase       []byte
	secret           []byte
	chainIdxMap      map[int64]btcutil.Address
	importedAddrs    []walletAddress
	lastChainIdx     int64
	missingKeysStart int64
}

//新建创建并初始化新存储。名称和描述的字节长度
//不能分别超过32和256字节。所有地址私钥
//使用密码短语加密。返回的密钥存储已锁定。
func New(dir string, desc string, passphrase []byte, net *chaincfg.Params,
	createdAt *BlockStamp) (*Store, error) {

//检查输入的大小。
	if len(desc) > 256 {
		return nil, errors.New("desc exceeds 256 byte maximum size")
	}

//随机生成rootkey和chaincode。
	rootkey := make([]byte, 32)
	if _, err := rand.Read(rootkey); err != nil {
		return nil, err
	}
	chaincode := make([]byte, 32)
	if _, err := rand.Read(chaincode); err != nil {
		return nil, err
	}

//计算aes密钥并加密根地址。
	kdfp, err := computeKdfParameters(defaultKdfComputeTime, defaultKdfMaxMem)
	if err != nil {
		return nil, err
	}
	aeskey := kdf(passphrase, kdfp)

//创建并填充密钥存储。
	s := &Store{
		path: filepath.Join(dir, Filename),
		dir:  dir,
		file: Filename,
		vers: VersCurrent,
		net:  (*netParams)(net),
		flags: walletFlags{
			useEncryption: true,
			watchingOnly:  false,
		},
		createDate:  time.Now().Unix(),
		highestUsed: rootKeyChainIdx,
		kdfParams:   *kdfp,
		recent: recentBlocks{
			lastHeight: createdAt.Height,
			hashes: []*chainhash.Hash{
				createdAt.Hash,
			},
		},
		addrMap:          make(map[addressKey]walletAddress),
		chainIdxMap:      make(map[int64]btcutil.Address),
		lastChainIdx:     rootKeyChainIdx,
		missingKeysStart: rootKeyChainIdx,
		secret:           aeskey,
	}
	copy(s.desc[:], []byte(desc))

//从键和链码创建新的根地址。
	root, err := newRootBtcAddress(s, rootkey, nil, chaincode,
		createdAt)
	if err != nil {
		return nil, err
	}

//验证根地址密钥对。
	if err := root.verifyKeypairs(); err != nil {
		return nil, err
	}

	if err := root.encrypt(aeskey); err != nil {
		return nil, err
	}

	s.keyGenerator = *root

//将根地址添加到映射。
	rootAddr := s.keyGenerator.Address()
	s.addrMap[getAddressKey(rootAddr)] = &s.keyGenerator
	s.chainIdxMap[rootKeyChainIdx] = rootAddr

//密钥存储区必须被锁定返回。
	if err := s.Lock(); err != nil {
		return nil, err
	}

	return s, nil
}

//readfrom从IO.reader读取数据并将其保存到密钥存储，
//返回读取的字节数和遇到的任何错误。
func (s *Store) ReadFrom(r io.Reader) (n int64, err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	var read int64

	s.net = &netParams{}
	s.addrMap = make(map[addressKey]walletAddress)
	s.chainIdxMap = make(map[int64]btcutil.Address)

	var id [8]byte
	appendedEntries := varEntries{store: s}
	s.keyGenerator.store = s

//遍历需要读取的每个条目。中频数据
//实现IO.readerFrom，使用其readFrom函数。否则，
//数据是指向固定大小值的指针。
	datas := []interface{}{
		&id,
		&s.vers,
		s.net,
		&s.flags,
make([]byte, 6), //军械库唯一ID的字节数
		&s.createDate,
		&s.name,
		&s.desc,
		&s.highestUsed,
		&s.kdfParams,
		make([]byte, 256),
		&s.keyGenerator,
		newUnusedSpace(1024, &s.recent),
		&appendedEntries,
	}
	for _, data := range datas {
		var err error
		switch d := data.(type) {
		case readerFromVersion:
			read, err = d.readFromVersion(s.vers, r)

		case io.ReaderFrom:
			read, err = d.ReadFrom(r)

		default:
			read, err = binaryRead(r, binary.LittleEndian, d)
		}
		n += read
		if err != nil {
			return n, err
		}
	}

	if id != fileID {
		return n, errors.New("unknown file ID")
	}

//将根地址添加到地址映射。
	rootAddr := s.keyGenerator.Address()
	s.addrMap[getAddressKey(rootAddr)] = &s.keyGenerator
	s.chainIdxMap[rootKeyChainIdx] = rootAddr
	s.lastChainIdx = rootKeyChainIdx

//填充未分隔的字段。
	wts := appendedEntries.entries
	for _, wt := range wts {
		switch e := wt.(type) {
		case *addrEntry:
			addr := e.addr.Address()
			s.addrMap[getAddressKey(addr)] = &e.addr
			if e.addr.Imported() {
				s.importedAddrs = append(s.importedAddrs, &e.addr)
			} else {
				s.chainIdxMap[e.addr.chainIndex] = addr
				if s.lastChainIdx < e.addr.chainIndex {
					s.lastChainIdx = e.addr.chainIndex
				}
			}

//如果尚未创建私钥，请标记
//最早，以便在下一个密钥存储解锁时创建所有密钥。
			if e.addr.flags.createPrivKeyNextUnlock {
				switch {
				case s.missingKeysStart == rootKeyChainIdx:
					fallthrough
				case e.addr.chainIndex < s.missingKeysStart:
					s.missingKeysStart = e.addr.chainIndex
				}
			}

		case *scriptEntry:
			addr := e.script.Address()
			s.addrMap[getAddressKey(addr)] = &e.script
//始终导入脚本。
			s.importedAddrs = append(s.importedAddrs, &e.script)

		default:
			return n, errors.New("unknown appended entry")
		}
	}

	return n, nil
}

//WriteTo序列化密钥存储并将其写入IO.Writer，
//返回写入的字节数和遇到的任何错误。
func (s *Store) WriteTo(w io.Writer) (n int64, err error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return s.writeTo(w)
}

func (s *Store) writeTo(w io.Writer) (n int64, err error) {
	var wts []io.WriterTo
	var chainedAddrs = make([]io.WriterTo, len(s.chainIdxMap)-1)
	var importedAddrs []io.WriterTo
	for _, wAddr := range s.addrMap {
		switch btcAddr := wAddr.(type) {
		case *btcAddress:
			e := &addrEntry{
				addr: *btcAddr,
			}
			copy(e.pubKeyHash160[:], btcAddr.AddrHash())
			if btcAddr.Imported() {
//没有导入地址的订单。
				importedAddrs = append(importedAddrs, e)
			} else if btcAddr.chainIndex >= 0 {
//链地址被排序。这是
//有点不错，但可能没必要。
				chainedAddrs[btcAddr.chainIndex] = e
			}

		case *scriptAddress:
			e := &scriptEntry{
				script: *btcAddr,
			}
			copy(e.scriptHash160[:], btcAddr.AddrHash())
//始终导入脚本
			importedAddrs = append(importedAddrs, e)
		}
	}
	wts = append(chainedAddrs, importedAddrs...)
	appendedEntries := varEntries{store: s, entries: wts}

//遍历需要写入的每个条目。中频数据
//实现io.writerto，使用其writeto函数c。否则，
//数据是指向固定大小值的指针。
	datas := []interface{}{
		&fileID,
		&VersCurrent,
		s.net,
		&s.flags,
make([]byte, 6), //军械库唯一ID的字节数
		&s.createDate,
		&s.name,
		&s.desc,
		&s.highestUsed,
		&s.kdfParams,
		make([]byte, 256),
		&s.keyGenerator,
		newUnusedSpace(1024, &s.recent),
		&appendedEntries,
	}
	var written int64
	for _, data := range datas {
		if s, ok := data.(io.WriterTo); ok {
			written, err = s.WriteTo(w)
		} else {
			written, err = binaryWrite(w, binary.LittleEndian, data)
		}
		n += written
		if err != nil {
			return n, err
		}
	}

	return n, nil
}

//TODO:自动设置。
func (s *Store) MarkDirty() {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.dirty = true
}

func (s *Store) WriteIfDirty() error {
	s.mtx.RLock()
	if !s.dirty {
		s.mtx.RUnlock()
		return nil
	}

//tempfile创建文件0600，因此不需要更改它。
	fi, err := ioutil.TempFile(s.dir, s.file)
	if err != nil {
		s.mtx.RUnlock()
		return err
	}
	fiPath := fi.Name()

	_, err = s.writeTo(fi)
	if err != nil {
		s.mtx.RUnlock()
		fi.Close()
		return err
	}
	err = fi.Sync()
	if err != nil {
		s.mtx.RUnlock()
		fi.Close()
		return err
	}
	fi.Close()

	err = rename.Atomic(fiPath, s.path)
	s.mtx.RUnlock()

	if err == nil {
		s.mtx.Lock()
		s.dirty = false
		s.mtx.Unlock()
	}

	return err
}

//opendir从指定目录打开一个新的密钥存储。如果文件
//不存在，将返回OS包中的错误，并且可以
//使用os.isnotexist检查以区分丢失的文件错误
//其他（包括反序列化）。
func OpenDir(dir string) (*Store, error) {
	path := filepath.Join(dir, Filename)
	fi, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer fi.Close()
	store := new(Store)
	_, err = store.ReadFrom(fi)
	if err != nil {
		return nil, err
	}
	store.path = path
	store.dir = dir
	store.file = Filename
	return store, nil
}

//unlock从密码短语和密钥存储的kdf派生一个aes密钥
//参数并解锁密钥存储区的根密钥。如果
//解锁成功，密钥存储的密钥被保存，
//允许解密任何加密的私钥。任何
//在密钥存储被锁定时创建的地址没有使用private
//此时将创建键。
func (s *Store) Unlock(passphrase []byte) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.flags.watchingOnly {
		return ErrWatchingOnly
	}

//从kdf参数和密码短语派生密钥。
	key := kdf(passphrase, &s.kdfParams)

//用派生密钥解锁根地址。
	if _, err := s.keyGenerator.unlock(key); err != nil {
		return err
	}

//如果解锁成功，请保存密码短语和AES密钥。
	s.passphrase = passphrase
	s.secret = key

	return s.createMissingPrivateKeys()
}

//Lock会尽最大努力删除所有密钥并将其归零。
//与密钥存储区关联。
func (s *Store) Lock() (err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.flags.watchingOnly {
		return ErrWatchingOnly
	}

//从密钥存储中删除明文密码。
	if s.isLocked() {
		err = ErrLocked
	} else {
		zero(s.passphrase)
		s.passphrase = nil
		zero(s.secret)
		s.secret = nil
	}

//从所有地址条目中删除明文私钥。
	for _, addr := range s.addrMap {
		if baddr, ok := addr.(*btcAddress); ok {
			_ = baddr.lock()
		}
	}

	return err
}

//changepassphrase从新的passphrase创建一个新的aes密钥，并
//用新密钥重新加密所有加密的私钥。
func (s *Store) ChangePassphrase(new []byte) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.flags.watchingOnly {
		return ErrWatchingOnly
	}

	if s.isLocked() {
		return ErrLocked
	}

	oldkey := s.secret
	newkey := kdf(new, &s.kdfParams)

	for _, wa := range s.addrMap {
//只有btcadress才有私人钥匙。
		a, ok := wa.(*btcAddress)
		if !ok {
			continue
		}

		if err := a.changeEncryptionKey(oldkey, newkey); err != nil {
			return err
		}
	}

//零旧秘密。
	zero(s.passphrase)
	zero(s.secret)

//保存新秘密。
	s.passphrase = new
	s.secret = newkey

	return nil
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

//IsLocked返回密钥存储是否已解锁（在这种情况下，
//密钥保存在内存中）或锁定。
func (s *Store) IsLocked() bool {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return s.isLocked()
}

func (s *Store) isLocked() bool {
	return len(s.secret) != 32
}

//NextChainedAddress尝试获取下一个链接地址。如果钥匙
//商店已解锁，地址链的下一个公钥和私钥是
//衍生的。如果密钥存储为locke，则只派生下一个pubkey，并且
//私钥将在下次解锁时生成。
func (s *Store) NextChainedAddress(bs *BlockStamp) (btcutil.Address, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.nextChainedAddress(bs)
}

func (s *Store) nextChainedAddress(bs *BlockStamp) (btcutil.Address, error) {
	addr, err := s.nextChainedBtcAddress(bs)
	if err != nil {
		return nil, err
	}
	return addr.Address(), nil
}

//changeAddress返回密钥存储中的下一个链接地址，标记
//变更事务输出的地址。
func (s *Store) ChangeAddress(bs *BlockStamp) (btcutil.Address, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	addr, err := s.nextChainedBtcAddress(bs)
	if err != nil {
		return nil, err
	}

	addr.flags.change = true

//创建并返回地址哈希的付款地址。
	return addr.Address(), nil
}

func (s *Store) nextChainedBtcAddress(bs *BlockStamp) (*btcAddress, error) {
//尝试获取下一个链接地址的地址哈希。
	nextAPKH, ok := s.chainIdxMap[s.highestUsed+1]
	if !ok {
		if s.isLocked() {
//链钥匙
			if err := s.extendLocked(bs); err != nil {
				return nil, err
			}
		} else {
//链接私钥和公钥。
			if err := s.extendUnlocked(bs); err != nil {
				return nil, err
			}
		}

//应将添加到内部映射，请重试查找。
		nextAPKH, ok = s.chainIdxMap[s.highestUsed+1]
		if !ok {
			return nil, errors.New("chain index map inproperly updated")
		}
	}

//查找地址。
	addr, ok := s.addrMap[getAddressKey(nextAPKH)]
	if !ok {
		return nil, errors.New("cannot find generated address")
	}

	btcAddr, ok := addr.(*btcAddress)
	if !ok {
		return nil, errors.New("found non-pubkey chained address")
	}

	s.highestUsed++

	return btcAddr, nil
}

//LastChainedAddress返回最近请求的链接
//调用NextChainedAddress的地址，或根地址，如果
//未请求链接地址。
func (s *Store) LastChainedAddress() btcutil.Address {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return s.chainIdxMap[s.highestUsed]
}

//extendUnlocked为未锁定的密钥库增加地址链。
func (s *Store) extendUnlocked(bs *BlockStamp) error {
//获取最后一个链接地址。新的链接地址将
//链接了这个地址的链码和私钥。
	a := s.chainIdxMap[s.lastChainIdx]
	waddr, ok := s.addrMap[getAddressKey(a)]
	if !ok {
		return errors.New("expected last chained address not found")
	}

	if s.isLocked() {
		return ErrLocked
	}

	lastAddr, ok := waddr.(*btcAddress)
	if !ok {
		return errors.New("found non-pubkey chained address")
	}

	privkey, err := lastAddr.unlock(s.secret)
	if err != nil {
		return err
	}
	cc := lastAddr.chaincode[:]

	privkey, err = chainedPrivKey(privkey, lastAddr.pubKeyBytes(), cc)
	if err != nil {
		return err
	}
	newAddr, err := newBtcAddress(s, privkey, nil, bs, true)
	if err != nil {
		return err
	}
	if err := newAddr.verifyKeypairs(); err != nil {
		return err
	}
	if err = newAddr.encrypt(s.secret); err != nil {
		return err
	}
	a = newAddr.Address()
	s.addrMap[getAddressKey(a)] = newAddr
	newAddr.chainIndex = lastAddr.chainIndex + 1
	s.chainIdxMap[newAddr.chainIndex] = a
	s.lastChainIdx++
	copy(newAddr.chaincode[:], cc)

	return nil
}

//extendLocked创建一个没有私钥的新地址（允许
//从锁定的密钥存储扩展地址链）从
//上次使用的链接地址，并将地址添加到密钥存储的内部
//簿记结构。
func (s *Store) extendLocked(bs *BlockStamp) error {
	a := s.chainIdxMap[s.lastChainIdx]
	waddr, ok := s.addrMap[getAddressKey(a)]
	if !ok {
		return errors.New("expected last chained address not found")
	}

	addr, ok := waddr.(*btcAddress)
	if !ok {
		return errors.New("found non-pubkey chained address")
	}

	cc := addr.chaincode[:]

	nextPubkey, err := chainedPubKey(addr.pubKeyBytes(), cc)
	if err != nil {
		return err
	}
	newaddr, err := newBtcAddressWithoutPrivkey(s, nextPubkey, nil, bs)
	if err != nil {
		return err
	}
	a = newaddr.Address()
	s.addrMap[getAddressKey(a)] = newaddr
	newaddr.chainIndex = addr.chainIndex + 1
	s.chainIdxMap[newaddr.chainIndex] = a
	s.lastChainIdx++
	copy(newaddr.chaincode[:], cc)

	if s.missingKeysStart == rootKeyChainIdx {
		s.missingKeysStart = newaddr.chainIndex
	}

	return nil
}

func (s *Store) createMissingPrivateKeys() error {
	idx := s.missingKeysStart
	if idx == rootKeyChainIdx {
		return nil
	}

//查找上一个地址。
	apkh, ok := s.chainIdxMap[idx-1]
	if !ok {
		return errors.New("missing previous chained address")
	}
	prevWAddr := s.addrMap[getAddressKey(apkh)]
	if s.isLocked() {
		return ErrLocked
	}

	prevAddr, ok := prevWAddr.(*btcAddress)
	if !ok {
		return errors.New("found non-pubkey chained address")
	}

	prevPrivKey, err := prevAddr.unlock(s.secret)
	if err != nil {
		return err
	}

	for i := idx; ; i++ {
//获取地址链中第i个地址的下一个私钥。
		ithPrivKey, err := chainedPrivKey(prevPrivKey,
			prevAddr.pubKeyBytes(), prevAddr.chaincode[:])
		if err != nil {
			return err
		}

//获取缺少私钥的地址，设置，和
//加密。
		apkh, ok := s.chainIdxMap[i]
		if !ok {
//完成了。
			break
		}
		waddr := s.addrMap[getAddressKey(apkh)]
		addr, ok := waddr.(*btcAddress)
		if !ok {
			return errors.New("found non-pubkey chained address")
		}
		addr.privKeyCT = ithPrivKey
		if err := addr.encrypt(s.secret); err != nil {
//避免错误：请参阅versusetneedsprivkeylag的注释。
			if err != ErrAlreadyEncrypted || s.vers.LT(VersUnsetNeedsPrivkeyFlag) {
				return err
			}
		}
		addr.flags.createPrivKeyNextUnlock = false

//为下一次迭代设置上一个地址和私钥。
		prevAddr = addr
		prevPrivKey = ithPrivKey
	}

	s.missingKeysStart = rootKeyChainIdx
	return nil
}

//地址返回密钥存储中地址的walletAddress结构。
//这个地址可以被类型转换成其他接口（如pubkeyaddress
//和scriptAddress）如果需要特定信息，例如密钥。
func (s *Store) Address(a btcutil.Address) (WalletAddress, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

//按地址哈希查找地址。
	btcaddr, ok := s.addrMap[getAddressKey(a)]
	if !ok {
		return nil, ErrAddressNotFound
	}

	return btcaddr, nil
}

//NET返回此密钥存储区的比特币网络参数。
func (s *Store) Net() *chaincfg.Params {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return s.netParams()
}

func (s *Store) netParams() *chaincfg.Params {
	return (*chaincfg.Params)(s.net)
}

//setsyncstatus设置单个密钥存储地址的同步状态。这个
//如果在密钥存储中找不到地址，则可能出错。
//
//将地址标记为未同步时，只有未同步的类型才重要。
//该值被忽略。
func (s *Store) SetSyncStatus(a btcutil.Address, ss SyncStatus) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	wa, ok := s.addrMap[getAddressKey(a)]
	if !ok {
		return ErrAddressNotFound
	}
	wa.setSyncStatus(ss)
	return nil
}

//setSyncedWith标记已在要进入的密钥存储中同步地址
//与块戳描述的最近看到的块同步。
//未同步的地址不受此方法影响，必须标记
//与markaddresssynced同步或markallsynced同步
//与BS同步。
//
//如果bs为nil，则整个密钥存储将标记为未同步。
func (s *Store) SetSyncedWith(bs *BlockStamp) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if bs == nil {
		s.recent.hashes = s.recent.hashes[:0]
		s.recent.lastHeight = s.keyGenerator.firstBlock
		s.keyGenerator.setSyncStatus(Unsynced(s.keyGenerator.firstBlock))
		return
	}

//检查是否要回滚上次看到的历史记录。
//如果是，并且此bs已保存，则删除任何内容
//之后返回。OtherWire，删除以前的哈希。
	if bs.Height < s.recent.lastHeight {
		maybeIdx := len(s.recent.hashes) - 1 - int(s.recent.lastHeight-bs.Height)
		if maybeIdx >= 0 && maybeIdx < len(s.recent.hashes) &&
			*s.recent.hashes[maybeIdx] == *bs.Hash {

			s.recent.lastHeight = bs.Height
//将删除的哈希进行子切片。
			s.recent.hashes = s.recent.hashes[:maybeIdx]
			return
		}
		s.recent.hashes = nil
	}

	if bs.Height != s.recent.lastHeight+1 {
		s.recent.hashes = nil
	}

	s.recent.lastHeight = bs.Height

	if len(s.recent.hashes) == 20 {
//为最近的哈希留出空间。
		copy(s.recent.hashes, s.recent.hashes[1:])

//在最后一个位置设置新块。
		s.recent.hashes[19] = bs.Hash
	} else {
		s.recent.hashes = append(s.recent.hashes, bs.Hash)
	}
}

//同步返回至少标记钱包的块的详细信息
//同步通过。高度是重新扫描应该从何时开始的高度
//将钱包同步回最佳链。
//
//注意：如果同步块的哈希未知，则哈希将为零，并且
//必须从其他地方获得。之前必须明确检查
//取消对指针的引用。
func (s *Store) SyncedTo() (hash *chainhash.Hash, height int32) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	switch h, ok := s.keyGenerator.SyncStatus().(PartialSync); {
	case ok && int32(h) > s.recent.lastHeight:
		height = int32(h)
	default:
		height = s.recent.lastHeight
		if n := len(s.recent.hashes); n != 0 {
			hash = s.recent.hashes[n-1]
		}
	}
	for _, a := range s.addrMap {
		var syncHeight int32
		switch e := a.SyncStatus().(type) {
		case Unsynced:
			syncHeight = int32(e)
		case PartialSync:
			syncHeight = int32(e)
		case FullSync:
			continue
		}
		if syncHeight < height {
			height = syncHeight
			hash = nil

//不能低于0。
			if height == 0 {
				return
			}
		}
	}
	return
}

//NewIteraterecentBlocks返回最近看到的块的迭代器。
//迭代器从最近添加的块开始，prev应该
//用于访问早期块。
func (s *Store) NewIterateRecentBlocks() *BlockIterator {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return s.recent.iter(s)
}

//importprivatekey将WIF私钥导入密钥库。进口的
//使用压缩或未压缩的序列化创建地址
//公钥，取决于wif的compresspubkey bool。
func (s *Store) ImportPrivateKey(wif *btcutil.WIF, bs *BlockStamp) (btcutil.Address, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.flags.watchingOnly {
		return nil, ErrWatchingOnly
	}

//首先，必须检查正在导入的密钥是否不会导致
//地址重复。
	pkh := btcutil.Hash160(wif.SerializePubKey())
	if _, ok := s.addrMap[addressKey(pkh)]; ok {
		return nil, ErrDuplicate
	}

//必须解锁密钥存储才能加密导入的私钥。
	if s.isLocked() {
		return nil, ErrLocked
	}

//使用此私钥创建新地址。
	privKey := wif.PrivKey.Serialize()
	btcaddr, err := newBtcAddress(s, privKey, nil, bs, wif.CompressPubKey)
	if err != nil {
		return nil, err
	}
	btcaddr.chainIndex = importedKeyChainIdx

//如果导入高度低于当前同步高度，则标记为未同步
//高度。
	if len(s.recent.hashes) != 0 && bs.Height < s.recent.lastHeight {
		btcaddr.flags.unsynced = true
	}

//用派生的aes密钥加密导入的地址。
	if err = btcaddr.encrypt(s.secret); err != nil {
		return nil, err
	}

	addr := btcaddr.Address()
//将地址添加到密钥存储的簿记结构中。添加到
//映射将导致导入的地址被序列化
//在下次WRITETO调用时。
	s.addrMap[getAddressKey(addr)] = btcaddr
	s.importedAddrs = append(s.importedAddrs, btcaddr)

//创建并返回地址。
	return addr, nil
}

//importscript使用用户提供的脚本创建新的脚本地址
//并将其添加到密钥存储区。
func (s *Store) ImportScript(script []byte, bs *BlockStamp) (btcutil.Address, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.flags.watchingOnly {
		return nil, ErrWatchingOnly
	}

	if _, ok := s.addrMap[addressKey(btcutil.Hash160(script))]; ok {
		return nil, ErrDuplicate
	}

//使用此私钥创建新地址。
	scriptaddr, err := newScriptAddress(s, script, bs)
	if err != nil {
		return nil, err
	}

//如果导入高度低于当前同步高度，则标记为未同步
//高度。
	if len(s.recent.hashes) != 0 && bs.Height < s.recent.lastHeight {
		scriptaddr.flags.unsynced = true
	}

//将地址添加到密钥存储的簿记结构中。添加到
//映射将导致导入的地址被序列化
//在下次WRITETO调用时。
	addr := scriptaddr.Address()
	s.addrMap[getAddressKey(addr)] = scriptaddr
	s.importedAddrs = append(s.importedAddrs, scriptaddr)

//创建并返回地址。
	return addr, nil
}

//createDate返回密钥存储创建时间的Unix时间。这个
//用于将密钥存储创建时间与块头和
//设置更好的重新扫描位置的最小块高度。
func (s *Store) CreateDate() int64 {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	return s.createDate
}

//exportwatchingwallet创建并返回一个新的密钥存储库
//地址在w中，但作为一个只监视没有任何私钥的密钥存储。
//监视密钥存储创建的新地址将与新地址匹配
//创建了原始密钥存储（由于公钥地址链接），但是
//将丢失关联的私钥。
func (s *Store) ExportWatchingWallet() (*Store, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

//如果密钥存储区已在监视，则不要继续。
	if s.flags.watchingOnly {
		return nil, ErrWatchingOnly
	}

//将w的成员复制到新的密钥存储中，但标记为仅监视和
//不包括任何私钥。
	ws := &Store{
		vers: s.vers,
		net:  s.net,
		flags: walletFlags{
			useEncryption: false,
			watchingOnly:  true,
		},
		name:        s.name,
		desc:        s.desc,
		createDate:  s.createDate,
		highestUsed: s.highestUsed,
		recent: recentBlocks{
			lastHeight: s.recent.lastHeight,
		},

		addrMap: make(map[addressKey]walletAddress),

//托多·奥加给我列个单子
		chainIdxMap:  make(map[int64]btcutil.Address),
		lastChainIdx: s.lastChainIdx,
	}

	kgwc := s.keyGenerator.watchingCopy(ws)
	ws.keyGenerator = *(kgwc.(*btcAddress))
	if len(s.recent.hashes) != 0 {
		ws.recent.hashes = make([]*chainhash.Hash, 0, len(s.recent.hashes))
		for _, hash := range s.recent.hashes {
			hashCpy := *hash
			ws.recent.hashes = append(ws.recent.hashes, &hashCpy)
		}
	}
	for apkh, addr := range s.addrMap {
		if !addr.Imported() {
//如果！进口。
			btcAddr := addr.(*btcAddress)

			ws.chainIdxMap[btcAddr.chainIndex] =
				addr.Address()
		}
		apkhCopy := apkh
		ws.addrMap[apkhCopy] = addr.watchingCopy(ws)
	}
	if len(s.importedAddrs) != 0 {
		ws.importedAddrs = make([]walletAddress, 0,
			len(s.importedAddrs))
		for _, addr := range s.importedAddrs {
			ws.importedAddrs = append(ws.importedAddrs, addr.watchingCopy(ws))
		}
	}

	return ws, nil
}

//同步状态是所有同步变量的接口类型。
type SyncStatus interface {
	ImplementsSyncStatus()
}

type (
//未同步是表示未同步地址的类型。当这是
//由密钥存储方法返回，该值是第一次看到的记录值
//块高度。
	Unsynced int32

//partialSync是一种表示部分同步地址的类型（用于
//例如，由于部分完成的重新扫描的结果）。
	PartialSync int32

//fullsync是一种表示与同步的地址的类型。
//最近看到的街区。
	FullSync struct{}
)

//实现同步状态是为了使未同步状态成为同步状态。
func (u Unsynced) ImplementsSyncStatus() {}

//implementssyncStatus的实现目的是使partialsync成为syncstatus。
func (p PartialSync) ImplementsSyncStatus() {}

//实现同步状态是为了使完全同步成为同步状态。
func (f FullSync) ImplementsSyncStatus() {}

//WalletAddress是一个接口，它提供有关
//由密钥存储管理的地址。这种类型的具体实现可以
//提供更多字段以提供特定于该类型的信息
//地址。
type WalletAddress interface {
//地址返回备份地址的btncutil.address。
	Address() btcutil.Address
//addrhash返回与地址相关的键或脚本哈希
	AddrHash() string
//FirstBlock返回地址可能位于的第一个块。
	FirstBlock() int32
//如果备份地址被导入，则compressed返回true
//作为地址链的一部分。
	Imported() bool
//
//更改事务的输出。
	Change() bool
//如果备份地址被压缩，则compressed返回true。
	Compressed() bool
//syncstatus返回地址的当前同步状态。
	SyncStatus() SyncStatus
}

//SortedActiveAddress返回所有已
//请求生成。其中不包括未使用的地址
//关键池。如果需要订购地址，请使用此选项。否则，
//活动连衣裙是首选。
func (s *Store) SortedActiveAddresses() []WalletAddress {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	addrs := make([]WalletAddress, 0,
		s.highestUsed+int64(len(s.importedAddrs))+1)
	for i := int64(rootKeyChainIdx); i <= s.highestUsed; i++ {
		a := s.chainIdxMap[i]
		info, ok := s.addrMap[getAddressKey(a)]
		if ok {
			addrs = append(addrs, info)
		}
	}
	for _, addr := range s.importedAddrs {
		addrs = append(addrs, addr)
	}
	return addrs
}

//activeaddresss返回活动付款地址之间的映射
//以及他们的全部信息。这些地址不包括
//密钥池。如果必须对地址进行排序，请使用SortedActiveAddress。
func (s *Store) ActiveAddresses() map[btcutil.Address]WalletAddress {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	addrs := make(map[btcutil.Address]WalletAddress)
	for i := int64(rootKeyChainIdx); i <= s.highestUsed; i++ {
		a := s.chainIdxMap[i]
		addr := s.addrMap[getAddressKey(a)]
		addrs[addr.Address()] = addr
	}
	for _, addr := range s.importedAddrs {
		addrs[addr.Address()] = addr
	}
	return addrs
}

//extendActiveAddress从
//地址链并将每个地址链标记为活动。这是用来恢复的
//来自密钥存储备份的确定（未导入）地址，或
//在加密密钥存储区与
//私钥和一个导出的监视密钥存储没有。
//
//每个新地址的bcutil.address返回一个切片。
//必须为这些地址重新扫描区块链。
func (s *Store) ExtendActiveAddresses(n int) ([]btcutil.Address, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	last := s.addrMap[getAddressKey(s.chainIdxMap[s.highestUsed])]
	bs := &BlockStamp{Height: last.FirstBlock()}

	addrs := make([]btcutil.Address, n)
	for i := 0; i < n; i++ {
		addr, err := s.nextChainedAddress(bs)
		if err != nil {
			return nil, err
		}
		addrs[i] = addr
	}
	return addrs, nil
}

type walletFlags struct {
	useEncryption bool
	watchingOnly  bool
}

func (wf *walletFlags) ReadFrom(r io.Reader) (int64, error) {
	var b [8]byte
	n, err := io.ReadFull(r, b[:])
	if err != nil {
		return int64(n), err
	}

	wf.useEncryption = b[0]&(1<<0) != 0
	wf.watchingOnly = b[0]&(1<<1) != 0

	return int64(n), nil
}

func (wf *walletFlags) WriteTo(w io.Writer) (int64, error) {
	var b [8]byte
	if wf.useEncryption {
		b[0] |= 1 << 0
	}
	if wf.watchingOnly {
		b[0] |= 1 << 1
	}
	n, err := w.Write(b[:])
	return int64(n), err
}

type addrFlags struct {
	hasPrivKey              bool
	hasPubKey               bool
	encrypted               bool
	createPrivKeyNextUnlock bool
	compressed              bool
	change                  bool
	unsynced                bool
	partialSync             bool
}

func (af *addrFlags) ReadFrom(r io.Reader) (int64, error) {
	var b [8]byte
	n, err := io.ReadFull(r, b[:])
	if err != nil {
		return int64(n), err
	}

	af.hasPrivKey = b[0]&(1<<0) != 0
	af.hasPubKey = b[0]&(1<<1) != 0
	af.encrypted = b[0]&(1<<2) != 0
	af.createPrivKeyNextUnlock = b[0]&(1<<3) != 0
	af.compressed = b[0]&(1<<4) != 0
	af.change = b[0]&(1<<5) != 0
	af.unsynced = b[0]&(1<<6) != 0
	af.partialSync = b[0]&(1<<7) != 0

//目前（至少在只监视密钥存储实现之前）
//btcwallet应拒绝打开任何未加密的地址。这个
//仅当存在要加密的私钥时，检查才有意义，而
//如果密钥池仅从最后一个扩展而来，则可能不存在
//没有写入公钥和私钥。
	if af.hasPrivKey && !af.encrypted {
		return int64(n), errors.New("private key is unencrypted")
	}

	return int64(n), nil
}

func (af *addrFlags) WriteTo(w io.Writer) (int64, error) {
	var b [8]byte
	if af.hasPrivKey {
		b[0] |= 1 << 0
	}
	if af.hasPubKey {
		b[0] |= 1 << 1
	}
	if af.hasPrivKey && !af.encrypted {
//我们只支持加密的私钥。
		return 0, errors.New("address must be encrypted")
	}
	if af.encrypted {
		b[0] |= 1 << 2
	}
	if af.createPrivKeyNextUnlock {
		b[0] |= 1 << 3
	}
	if af.compressed {
		b[0] |= 1 << 4
	}
	if af.change {
		b[0] |= 1 << 5
	}
	if af.unsynced {
		b[0] |= 1 << 6
	}
	if af.partialSync {
		b[0] |= 1 << 7
	}

	n, err := w.Write(b[:])
	return int64(n), err
}

//RecentBlocks最多可容纳最后20个看到的块哈希以及
//最近看到的块的块高度。
type recentBlocks struct {
	hashes     []*chainhash.Hash
	lastHeight int32
}

func (rb *recentBlocks) readFromVersion(v version, r io.Reader) (int64, error) {
	if !v.LT(Vers20LastBlocks) {
//使用当前版本。
		return rb.ReadFrom(r)
	}

//旧文件版本只保存最近看到的
//块高度和散列，不是最后20个。

	var read int64

//读取高度。
var heightBytes [4]byte //Int32为4字节
	n, err := io.ReadFull(r, heightBytes[:])
	read += int64(n)
	if err != nil {
		return read, err
	}
	rb.lastHeight = int32(binary.LittleEndian.Uint32(heightBytes[:]))

//如果高度为-1，则最后一个同步块未知，因此不要尝试
//读取块哈希。
	if rb.lastHeight == -1 {
		rb.hashes = nil
		return read, nil
	}

//读取块哈希。
	var syncedBlockHash chainhash.Hash
	n, err = io.ReadFull(r, syncedBlockHash[:])
	read += int64(n)
	if err != nil {
		return read, err
	}

	rb.hashes = []*chainhash.Hash{
		&syncedBlockHash,
	}

	return read, nil
}

func (rb *recentBlocks) ReadFrom(r io.Reader) (int64, error) {
	var read int64

//读取已保存块的数目。这不应超过20。
var nBlockBytes [4]byte //uint32为4字节
	n, err := io.ReadFull(r, nBlockBytes[:])
	read += int64(n)
	if err != nil {
		return read, err
	}
	nBlocks := binary.LittleEndian.Uint32(nBlockBytes[:])
	if nBlocks > 20 {
		return read, errors.New("number of last seen blocks exceeds maximum of 20")
	}

//阅读最近看到的块高度。
var heightBytes [4]byte //Int32为4字节
	n, err = io.ReadFull(r, heightBytes[:])
	read += int64(n)
	if err != nil {
		return read, err
	}
	height := int32(binary.LittleEndian.Uint32(heightBytes[:]))

//高度不应为-1（或任何其他负数）
//既然现在我们至少应该读一本
//已知块体。
	if height < 0 {
		return read, errors.New("expected a block but specified height is negative")
	}

//设置上次看到的高度。
	rb.lastHeight = height

//读取nblocks块哈希。哈希值应在
//从最旧到最新的顺序，但无法检查
//在这里。
	rb.hashes = make([]*chainhash.Hash, 0, nBlocks)
	for i := uint32(0); i < nBlocks; i++ {
		var blockHash chainhash.Hash
		n, err := io.ReadFull(r, blockHash[:])
		read += int64(n)
		if err != nil {
			return read, err
		}
		rb.hashes = append(rb.hashes, &blockHash)
	}

	return read, nil
}

func (rb *recentBlocks) WriteTo(w io.Writer) (int64, error) {
	var written int64

//写入已保存块的数目。这不应超过20。
	nBlocks := uint32(len(rb.hashes))
	if nBlocks > 20 {
		return written, errors.New("number of last seen blocks exceeds maximum of 20")
	}
	if nBlocks != 0 && rb.lastHeight < 0 {
		return written, errors.New("number of block hashes is positive, but height is negative")
	}
var nBlockBytes [4]byte //uint32为4字节
	binary.LittleEndian.PutUint32(nBlockBytes[:], nBlocks)
	n, err := w.Write(nBlockBytes[:])
	written += int64(n)
	if err != nil {
		return written, err
	}

//写最近看到的块高度。
var heightBytes [4]byte //Int32为4字节
	binary.LittleEndian.PutUint32(heightBytes[:], uint32(rb.lastHeight))
	n, err = w.Write(heightBytes[:])
	written += int64(n)
	if err != nil {
		return written, err
	}

//写入块哈希。
	for _, hash := range rb.hashes {
		n, err := w.Write(hash[:])
		written += int64(n)
		if err != nil {
			return written, err
		}
	}

	return written, nil
}

//BlockIterator允许最近的
//看到的街区。
type BlockIterator struct {
	storeMtx *sync.RWMutex
	height   int32
	index    int
	rb       *recentBlocks
}

func (rb *recentBlocks) iter(s *Store) *BlockIterator {
	if rb.lastHeight == -1 || len(rb.hashes) == 0 {
		return nil
	}
	return &BlockIterator{
		storeMtx: &s.mtx,
		height:   rb.lastHeight,
		index:    len(rb.hashes) - 1,
		rb:       rb,
	}
}

func (it *BlockIterator) Next() bool {
	it.storeMtx.RLock()
	defer it.storeMtx.RUnlock()

	if it.index+1 >= len(it.rb.hashes) {
		return false
	}
	it.index++
	return true
}

func (it *BlockIterator) Prev() bool {
	it.storeMtx.RLock()
	defer it.storeMtx.RUnlock()

	if it.index-1 < 0 {
		return false
	}
	it.index--
	return true
}

func (it *BlockIterator) BlockStamp() BlockStamp {
	it.storeMtx.RLock()
	defer it.storeMtx.RUnlock()

	return BlockStamp{
		Height: it.rb.lastHeight - int32(len(it.rb.hashes)-1-it.index),
		Hash:   it.rb.hashes[it.index],
	}
}

//UnusedSpace是一种包装类型，用于读取或写入一个或多个类型
//那个btcwallet放在武器库的密钥存储文件留下的一个未使用的空间中。
//格式。
type unusedSpace struct {
nBytes int //武器库剩余的未使用字节数。
	rfvs   []readerFromVersion
}

func newUnusedSpace(nBytes int, rfvs ...readerFromVersion) *unusedSpace {
	return &unusedSpace{
		nBytes: nBytes,
		rfvs:   rfvs,
	}
}

func (u *unusedSpace) readFromVersion(v version, r io.Reader) (int64, error) {
	var read int64

	for _, rfv := range u.rfvs {
		n, err := rfv.readFromVersion(v, r)
		if err != nil {
			return read + n, err
		}
		read += n
		if read > int64(u.nBytes) {
			return read, errors.New("read too much from armory's unused space")
		}
	}

//读取剩余的实际未使用的字节。
	unused := make([]byte, u.nBytes-int(read))
	n, err := io.ReadFull(r, unused)
	return read + int64(n), err
}

func (u *unusedSpace) WriteTo(w io.Writer) (int64, error) {
	var written int64

	for _, wt := range u.rfvs {
		n, err := wt.WriteTo(w)
		if err != nil {
			return written + n, err
		}
		written += n
		if written > int64(u.nBytes) {
			return written, errors.New("wrote too much to armory's unused space")
		}
	}

//写入剩余的实际未使用的字节。
	unused := make([]byte, u.nBytes-int(written))
	n, err := w.Write(unused)
	return written + int64(n), err
}

//WalletAddress是一个内部接口，用于围绕
//不同的地址类型。
type walletAddress interface {
	io.ReaderFrom
	io.WriterTo
	WalletAddress
	watchingCopy(*Store) walletAddress
	setSyncStatus(SyncStatus)
}

type btcAddress struct {
	store             *Store
	address           btcutil.Address
	flags             addrFlags
	chaincode         [32]byte
	chainIndex        int64
chainDepth        int64 //未使用的
	initVector        [16]byte
	privKey           [32]byte
	pubKey            *btcec.PublicKey
	firstSeen         int64
	lastSeen          int64
	firstBlock        int32
partialSyncHeight int32  //这是从军械库的“最后一块”区域重新申请的。
privKeyCT         []byte //解锁时不为零。
}

const (
//根地址的链索引为-1。每个后续
//链接地址增加索引。
	rootKeyChainIdx = -1

//导入的私钥不是链的一部分，并且具有
//特殊索引为-2。
	importedKeyChainIdx = -2
)

const (
	pubkeyCompressed   byte = 0x2
	pubkeyUncompressed byte = 0x4
)

type publicKey []byte

func (k *publicKey) ReadFrom(r io.Reader) (n int64, err error) {
	var read int64
	var format byte
	read, err = binaryRead(r, binary.LittleEndian, &format)
	if err != nil {
		return n + read, err
	}
	n += read

//从格式中删除奇怪之处
	noodd := format
	noodd &= ^byte(0x1)

	var s []byte
	switch noodd {
	case pubkeyUncompressed:
//读取剩余的64字节。
		s = make([]byte, 64)

	case pubkeyCompressed:
//读取剩余的32个字节。
		s = make([]byte, 32)

	default:
		return n, errors.New("unrecognized pubkey format")
	}

	read, err = binaryRead(r, binary.LittleEndian, &s)
	if err != nil {
		return n + read, err
	}
	n += read

	*k = append([]byte{format}, s...)
	return
}

func (k *publicKey) WriteTo(w io.Writer) (n int64, err error) {
	return binaryWrite(w, binary.LittleEndian, []byte(*k))
}

//pubkeyaddress实现walletaddress，并另外提供
//基于pubkey的地址的pubkey。
type PubKeyAddress interface {
	WalletAddress
//pubkey返回与地址关联的公钥。
	PubKey() *btcec.PublicKey
//exportpubkey返回与地址关联的公钥
//序列化为十六进制编码字符串。
	ExportPubKey() string
//privkey返回地址的私钥。
//如果密钥存储仅监视，密钥存储被锁定，则可能失败。
//或者地址没有任何密钥。
	PrivKey() (*btcec.PrivateKey, error)
//exportprivkey导出WIF私钥。
	ExportPrivKey() (*btcutil.WIF, error)
}

//NewBtcAddress初始化并返回新地址。私人必需品
//为32字节。IV必须是16字节，或者为零（在这种情况下是
//随机生成）。
func newBtcAddress(wallet *Store, privkey, iv []byte, bs *BlockStamp, compressed bool) (addr *btcAddress, err error) {
	if len(privkey) != 32 {
		return nil, errors.New("private key is not 32 bytes")
	}

	addr, err = newBtcAddressWithoutPrivkey(wallet,
		pubkeyFromPrivkey(privkey, compressed), iv, bs)
	if err != nil {
		return nil, err
	}

	addr.flags.createPrivKeyNextUnlock = false
	addr.flags.hasPrivKey = true
	addr.privKeyCT = privkey

	return addr, nil
}

//NewBtcAddressWithOutprivKey初始化并返回一个新地址
//稍后必须找到的未知（当时）私钥。PUBKEY必须是
//33或65字节，IV必须为16字节或空（在这种情况下是
//随机生成）。
func newBtcAddressWithoutPrivkey(s *Store, pubkey, iv []byte, bs *BlockStamp) (addr *btcAddress, err error) {
	var compressed bool
	switch n := len(pubkey); n {
	case btcec.PubKeyBytesLenCompressed:
		compressed = true
	case btcec.PubKeyBytesLenUncompressed:
		compressed = false
	default:
		return nil, fmt.Errorf("invalid pubkey length %d", n)
	}
	if len(iv) == 0 {
		iv = make([]byte, 16)
		if _, err := rand.Read(iv); err != nil {
			return nil, err
		}
	} else if len(iv) != 16 {
		return nil, errors.New("init vector must be nil or 16 bytes large")
	}

	pk, err := btcec.ParsePubKey(pubkey, btcec.S256())
	if err != nil {
		return nil, err
	}

	address, err := btcutil.NewAddressPubKeyHash(btcutil.Hash160(pubkey), s.netParams())
	if err != nil {
		return nil, err
	}

	addr = &btcAddress{
		flags: addrFlags{
			hasPrivKey:              false,
			hasPubKey:               true,
			encrypted:               false,
			createPrivKeyNextUnlock: true,
			compressed:              compressed,
			change:                  false,
			unsynced:                false,
		},
		store:      s,
		address:    address,
		firstSeen:  time.Now().Unix(),
		firstBlock: bs.Height,
		pubKey:     pk,
	}
	copy(addr.initVector[:], iv)

	return addr, nil
}

//newrootbtcadress生成新地址，同时设置
//将此地址表示为根的链代码和链索引
//地址。
func newRootBtcAddress(s *Store, privKey, iv, chaincode []byte,
	bs *BlockStamp) (addr *btcAddress, err error) {

	if len(chaincode) != 32 {
		return nil, errors.New("chaincode is not 32 bytes")
	}

//使用提供的输入创建新的btcadress。本遗嘱
//始终使用压缩的pubkey。
	addr, err = newBtcAddress(s, privKey, iv, bs, true)
	if err != nil {
		return nil, err
	}

	copy(addr.chaincode[:], chaincode)
	addr.chainIndex = rootKeyChainIdx

	return addr, err
}

//VerifyKeyPairs使用解析的私钥和
//用解析的公钥验证签名。如果这些
//步骤失败，密钥对生成失败，任何资金发送到此
//地址将不可挂起。此步骤需要一个未加密的或
//解锁的btcadress。
func (a *btcAddress) verifyKeypairs() error {
	if len(a.privKeyCT) != 32 {
		return errors.New("private key unavailable")
	}

	privKey := &btcec.PrivateKey{
		PublicKey: *a.pubKey.ToECDSA(),
		D:         new(big.Int).SetBytes(a.privKeyCT),
	}

	data := "String to sign."
	sig, err := privKey.Sign([]byte(data))
	if err != nil {
		return err
	}

	ok := sig.Verify([]byte(data), privKey.PubKey())
	if !ok {
		return errors.New("pubkey verification failed")
	}
	return nil
}

//readFrom从IO.reader读取加密地址。
func (a *btcAddress) ReadFrom(r io.Reader) (n int64, err error) {
	var read int64

//校验和
	var chkPubKeyHash uint32
	var chkChaincode uint32
	var chkInitVector uint32
	var chkPrivKey uint32
	var chkPubKey uint32
	var pubKeyHash [ripemd160.Size]byte
	var pubKey publicKey

//将序列化密钥存储读取到addr字段和校验和中。
	datas := []interface{}{
		&pubKeyHash,
		&chkPubKeyHash,
make([]byte, 4), //版本
		&a.flags,
		&a.chaincode,
		&chkChaincode,
		&a.chainIndex,
		&a.chainDepth,
		&a.initVector,
		&chkInitVector,
		&a.privKey,
		&chkPrivKey,
		&pubKey,
		&chkPubKey,
		&a.firstSeen,
		&a.lastSeen,
		&a.firstBlock,
		&a.partialSyncHeight,
	}
	for _, data := range datas {
		if rf, ok := data.(io.ReaderFrom); ok {
			read, err = rf.ReadFrom(r)
		} else {
			read, err = binaryRead(r, binary.LittleEndian, data)
		}
		if err != nil {
			return n + read, err
		}
		n += read
	}

//验证校验和，尽可能纠正错误。
	checks := []struct {
		data []byte
		chk  uint32
	}{
		{pubKeyHash[:], chkPubKeyHash},
		{a.chaincode[:], chkChaincode},
		{a.initVector[:], chkInitVector},
		{a.privKey[:], chkPrivKey},
		{pubKey, chkPubKey},
	}
	for i := range checks {
		if err = verifyAndFix(checks[i].data, checks[i].chk); err != nil {
			return n, err
		}
	}

	if !a.flags.hasPubKey {
		return n, errors.New("read in an address without a public key")
	}
	pk, err := btcec.ParsePubKey(pubKey, btcec.S256())
	if err != nil {
		return n, err
	}
	a.pubKey = pk

	addr, err := btcutil.NewAddressPubKeyHash(pubKeyHash[:], a.store.netParams())
	if err != nil {
		return n, err
	}
	a.address = addr

	return n, nil
}

func (a *btcAddress) WriteTo(w io.Writer) (n int64, err error) {
	var written int64

	pubKey := a.pubKeyBytes()

	hash := a.address.ScriptAddress()
	datas := []interface{}{
		&hash,
		walletHash(hash),
make([]byte, 4), //版本
		&a.flags,
		&a.chaincode,
		walletHash(a.chaincode[:]),
		&a.chainIndex,
		&a.chainDepth,
		&a.initVector,
		walletHash(a.initVector[:]),
		&a.privKey,
		walletHash(a.privKey[:]),
		&pubKey,
		walletHash(pubKey),
		&a.firstSeen,
		&a.lastSeen,
		&a.firstBlock,
		&a.partialSyncHeight,
	}
	for _, data := range datas {
		if wt, ok := data.(io.WriterTo); ok {
			written, err = wt.WriteTo(w)
		} else {
			written, err = binaryWrite(w, binary.LittleEndian, data)
		}
		if err != nil {
			return n + written, err
		}
		n += written
	}
	return n, nil
}

//加密尝试加密地址的明文私钥，
//如果地址已加密或私钥为
//不是32字节。如果成功，则设置加密标志。
func (a *btcAddress) encrypt(key []byte) error {
	if a.flags.encrypted {
		return ErrAlreadyEncrypted
	}
	if len(a.privKeyCT) != 32 {
		return errors.New("invalid clear text private key")
	}

	aesBlockEncrypter, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	aesEncrypter := cipher.NewCFBEncrypter(aesBlockEncrypter, a.initVector[:])

	aesEncrypter.XORKeyStream(a.privKey[:], a.privKeyCT)

	a.flags.hasPrivKey = true
	a.flags.encrypted = true
	return nil
}

//锁定删除此地址对其明文的引用
//私钥。如果地址未加密，此函数将失败。
func (a *btcAddress) lock() error {
	if !a.flags.encrypted {
		return errors.New("unable to lock unencrypted address")
	}

	zero(a.privKeyCT)
	a.privKeyCT = nil
	return nil
}

//解锁解密并存储指向地址私钥的指针，
//如果地址未加密或提供的密钥
//不正确。返回的明文私钥将始终是副本
//可供来电者安全使用，无需担心
//在地址锁定期间归零。
func (a *btcAddress) unlock(key []byte) (privKeyCT []byte, err error) {
	if !a.flags.encrypted {
		return nil, errors.New("unable to unlock unencrypted address")
	}

//用AES密钥解密私钥。
	aesBlockDecrypter, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesDecrypter := cipher.NewCFBDecrypter(aesBlockDecrypter, a.initVector[:])
	privkey := make([]byte, 32)
	aesDecrypter.XORKeyStream(privkey, a.privKey[:])

//如果已经保存了秘密，只需比较字节。
	if len(a.privKeyCT) == 32 {
		if !bytes.Equal(a.privKeyCT, privkey) {
			return nil, ErrWrongPassphrase
		}
		privKeyCT := make([]byte, 32)
		copy(privKeyCT, a.privKeyCT)
		return privKeyCT, nil
	}

	x, y := btcec.S256().ScalarBaseMult(privkey)
	if x.Cmp(a.pubKey.X) != 0 || y.Cmp(a.pubKey.Y) != 0 {
		return nil, ErrWrongPassphrase
	}

	privkeyCopy := make([]byte, 32)
	copy(privkeyCopy, privkey)
	a.privKeyCT = privkey
	return privkeyCopy, nil
}

//changeEncryptionKey重新加密地址的私钥
//使用新的AES加密密钥。old key必须是旧的aes加密密钥
//用于解密私钥。
func (a *btcAddress) changeEncryptionKey(oldkey, newkey []byte) error {
//地址必须具有私钥并加密才能继续。
	if !a.flags.hasPrivKey {
		return errors.New("no private key")
	}
	if !a.flags.encrypted {
		return errors.New("address is not encrypted")
	}

	privKeyCT, err := a.unlock(oldkey)
	if err != nil {
		return err
	}

	aesBlockEncrypter, err := aes.NewCipher(newkey)
	if err != nil {
		return err
	}
	newIV := make([]byte, len(a.initVector))
	if _, err := rand.Read(newIV); err != nil {
		return err
	}
	copy(a.initVector[:], newIV)
	aesEncrypter := cipher.NewCFBEncrypter(aesBlockEncrypter, a.initVector[:])
	aesEncrypter.XORKeyStream(a.privKey[:], privKeyCT)

	return nil
}

//address返回pub键地址，实现addressinfo。
func (a *btcAddress) Address() btcutil.Address {
	return a.address
}

//addrhash返回pub键散列，实现walletaddress。
func (a *btcAddress) AddrHash() string {
	return string(a.address.ScriptAddress())
}

//FirstBlock返回地址所在的第一个块，实现
//AddressInfo。
func (a *btcAddress) FirstBlock() int32 {
	return a.firstBlock
}

//如果导入了地址或链接地址，则imported返回pub，
//实现地址信息。
func (a *btcAddress) Imported() bool {
	return a.chainIndex == importedKeyChainIdx
}

//如果地址创建为更改地址，则change返回true，
//实现地址信息。
func (a *btcAddress) Change() bool {
	return a.flags.change
}

//如果地址备份键被压缩，则compressed返回true，
//实现地址信息。
func (a *btcAddress) Compressed() bool {
	return a.flags.compressed
}

//syncstatus返回当前地址的syncstatus类型
//同步。对于未同步的类型，该值是第一次看到的记录值
//地址的块高度。
func (a *btcAddress) SyncStatus() SyncStatus {
	switch {
	case a.flags.unsynced && !a.flags.partialSync:
		return Unsynced(a.firstBlock)
	case a.flags.unsynced && a.flags.partialSync:
		return PartialSync(a.partialSyncHeight)
	default:
		return FullSync{}
	}
}

//pubkey返回地址的十六进制编码pubkey。实施
//PUBKEY地址。
func (a *btcAddress) PubKey() *btcec.PublicKey {
	return a.pubKey
}

func (a *btcAddress) pubKeyBytes() []byte {
	if a.Compressed() {
		return a.pubKey.SerializeCompressed()
	}
	return a.pubKey.SerializeUncompressed()
}

//exportpubkey返回与序列化为的地址关联的公钥
//十六进制编码的字符串。实例PubKeyAddress
func (a *btcAddress) ExportPubKey() string {
	return hex.EncodeToString(a.pubKeyBytes())
}

//privkey通过返回私钥或错误来实现pubkeyaddress
//如果密钥存储被锁定，则仅监视或缺少私钥。
func (a *btcAddress) PrivKey() (*btcec.PrivateKey, error) {
	if a.store.flags.watchingOnly {
		return nil, ErrWatchingOnly
	}

	if !a.flags.hasPrivKey {
		return nil, errors.New("no private key for address")
	}

//必须解锁密钥存储才能解密私钥。
	if a.store.isLocked() {
		return nil, ErrLocked
	}

//用密钥存储密码解锁地址。UNLOCK返回的副本
//明文私钥，甚至可以安全使用
//地址锁定期间。
	privKeyCT, err := a.unlock(a.store.secret)
	if err != nil {
		return nil, err
	}

	return &btcec.PrivateKey{
		PublicKey: *a.pubKey.ToECDSA(),
		D:         new(big.Int).SetBytes(privKeyCT),
	}, nil
}

//exportprivkey将私钥导出为WIF，以便作为字符串进行编码
//在钱包里输入表格。
func (a *btcAddress) ExportPrivKey() (*btcutil.WIF, error) {
	pk, err := a.PrivKey()
	if err != nil {
		return nil, err
	}
//newwif仅在网络为零时出错。在这种情况下，恐慌，
//因为我们的程序的假设是如此的破碎，这需要
//立即捕获，这里的堆栈跟踪比
//别处。
	wif, err := btcutil.NewWIF((*btcec.PrivateKey)(pk), a.store.netParams(),
		a.Compressed())
	if err != nil {
		panic(err)
	}
	return wif, nil
}

//watchingcopy创建不带私钥的地址的副本。
//这用于在监视密钥存储时填充来自
//普通密钥存储。
func (a *btcAddress) watchingCopy(s *Store) walletAddress {
	return &btcAddress{
		store:   s,
		address: a.address,
		flags: addrFlags{
			hasPrivKey:              false,
			hasPubKey:               true,
			encrypted:               false,
			createPrivKeyNextUnlock: false,
			compressed:              a.flags.compressed,
			change:                  a.flags.change,
			unsynced:                a.flags.unsynced,
		},
		chaincode:         a.chaincode,
		chainIndex:        a.chainIndex,
		chainDepth:        a.chainDepth,
		pubKey:            a.pubKey,
		firstSeen:         a.firstSeen,
		lastSeen:          a.lastSeen,
		firstBlock:        a.firstBlock,
		partialSyncHeight: a.partialSyncHeight,
	}
}

//setsyncstatus设置地址标志和可能的部分同步高度
//取决于S的类型。
func (a *btcAddress) setSyncStatus(s SyncStatus) {
	switch e := s.(type) {
	case Unsynced:
		a.flags.unsynced = true
		a.flags.partialSync = false
		a.partialSyncHeight = 0

	case PartialSync:
		a.flags.unsynced = true
		a.flags.partialSync = true
		a.partialSyncHeight = int32(e)

	case FullSync:
		a.flags.unsynced = false
		a.flags.partialSync = false
		a.partialSyncHeight = 0
	}
}

//请注意，这里没有加密位，因为如果我们对脚本进行了加密
//然后在区块链上使用它，这提供了一个简单的已知明文
//密钥存储文件。已确定p2sh事务中的脚本是
//不是秘密，任何理智的情况都需要签名
//确实有秘密）。
type scriptFlags struct {
	hasScript   bool
	change      bool
	unsynced    bool
	partialSync bool
}

//readFrom通过将r读取到sf来实现IO.readerFrom接口。
func (sf *scriptFlags) ReadFrom(r io.Reader) (int64, error) {
	var b [8]byte
	n, err := io.ReadFull(r, b[:])
	if err != nil {
		return int64(n), err
	}

//我们为类似的字段匹配addrflags中的位。因此，hasscript使用
//与haspubkey相同的位和change位对于两者都是相同的。
	sf.hasScript = b[0]&(1<<1) != 0
	sf.change = b[0]&(1<<5) != 0
	sf.unsynced = b[0]&(1<<6) != 0
	sf.partialSync = b[0]&(1<<7) != 0

	return int64(n), nil
}

//writeto通过将sf写入w来实现io.writeto接口。
func (sf *scriptFlags) WriteTo(w io.Writer) (int64, error) {
	var b [8]byte
	if sf.hasScript {
		b[0] |= 1 << 1
	}
	if sf.change {
		b[0] |= 1 << 5
	}
	if sf.unsynced {
		b[0] |= 1 << 6
	}
	if sf.partialSync {
		b[0] |= 1 << 7
	}

	n, err := w.Write(b[:])
	return int64(n), err
}

//p2shscript表示密钥存储中的可变长度脚本项。
type p2SHScript []byte

//readFrom通过从中读取p2sh脚本来实现readerFrom接口。
//r格式<4 bytes little endian length><script bytes>
func (a *p2SHScript) ReadFrom(r io.Reader) (n int64, err error) {
//读取长度
	var lenBytes [4]byte

	read, err := io.ReadFull(r, lenBytes[:])
	n += int64(read)
	if err != nil {
		return n, err
	}

	length := binary.LittleEndian.Uint32(lenBytes[:])

	script := make([]byte, length)

	read, err = io.ReadFull(r, script)
	n += int64(read)
	if err != nil {
		return n, err
	}

	*a = script

	return n, nil
}

//writeto通过将p2sh脚本写入w来实现writerto接口。
//格式<4 bytes little endian length><script bytes>
func (a *p2SHScript) WriteTo(w io.Writer) (n int64, err error) {
//准备并写入32位小尾数长度头段
	var lenBytes [4]byte
	binary.LittleEndian.PutUint32(lenBytes[:], uint32(len(*a)))

	written, err := w.Write(lenBytes[:])
	n += int64(written)
	if err != nil {
		return n, err
	}

//现在自己写字节。
	written, err = w.Write(*a)

	return n + int64(written), err
}

type scriptAddress struct {
	store             *Store
	address           btcutil.Address
	class             txscript.ScriptClass
	addresses         []btcutil.Address
	reqSigs           int
	flags             scriptFlags
script            p2SHScript //可变长度
	firstSeen         int64
	lastSeen          int64
	firstBlock        int32
	partialSyncHeight int32
}

//scriptAddress是一个接口，它表示的“按脚本付费”哈希样式为
//比特币地址。
type ScriptAddress interface {
	WalletAddress
//返回与地址关联的脚本。
	Script() []byte
//返回与地址关联的脚本的类。
	ScriptClass() txscript.ScriptClass
//返回签名事务所需的地址
//脚本地址。
	Addresses() []btcutil.Address
//返回脚本地址所需的签名数。
	RequiredSigs() int
}

//newscriptAddress初始化并返回新的p2sh地址。
//IV必须是16字节，或者为零（在这种情况下，它是随机生成的）。
func newScriptAddress(s *Store, script []byte, bs *BlockStamp) (addr *scriptAddress, err error) {
	class, addresses, reqSigs, err :=
		txscript.ExtractPkScriptAddrs(script, s.netParams())
	if err != nil {
		return nil, err
	}

	scriptHash := btcutil.Hash160(script)

	address, err := btcutil.NewAddressScriptHashFromHash(scriptHash, s.netParams())
	if err != nil {
		return nil, err
	}

	addr = &scriptAddress{
		store:     s,
		address:   address,
		addresses: addresses,
		class:     class,
		reqSigs:   reqSigs,
		flags: scriptFlags{
			hasScript: true,
			change:    false,
		},
		script:     script,
		firstSeen:  time.Now().Unix(),
		firstBlock: bs.Height,
	}

	return addr, nil
}

//readFrom从IO.reader读取脚本地址。
func (sa *scriptAddress) ReadFrom(r io.Reader) (n int64, err error) {
	var read int64

//校验和
	var chkScriptHash uint32
	var chkScript uint32
	var scriptHash [ripemd160.Size]byte

//将序列化密钥存储读取到addr字段和校验和中。
	datas := []interface{}{
		&scriptHash,
		&chkScriptHash,
make([]byte, 4), //版本
		&sa.flags,
		&sa.script,
		&chkScript,
		&sa.firstSeen,
		&sa.lastSeen,
		&sa.firstBlock,
		&sa.partialSyncHeight,
	}
	for _, data := range datas {
		if rf, ok := data.(io.ReaderFrom); ok {
			read, err = rf.ReadFrom(r)
		} else {
			read, err = binaryRead(r, binary.LittleEndian, data)
		}
		if err != nil {
			return n + read, err
		}
		n += read
	}

//验证校验和，尽可能纠正错误。
	checks := []struct {
		data []byte
		chk  uint32
	}{
		{scriptHash[:], chkScriptHash},
		{sa.script, chkScript},
	}
	for i := range checks {
		if err = verifyAndFix(checks[i].data, checks[i].chk); err != nil {
			return n, err
		}
	}

	address, err := btcutil.NewAddressScriptHashFromHash(scriptHash[:],
		sa.store.netParams())
	if err != nil {
		return n, err
	}

	sa.address = address

	if !sa.flags.hasScript {
		return n, errors.New("read in an addresss with no script")
	}

	class, addresses, reqSigs, err :=
		txscript.ExtractPkScriptAddrs(sa.script, sa.store.netParams())
	if err != nil {
		return n, err
	}

	sa.class = class
	sa.addresses = addresses
	sa.reqSigs = reqSigs

	return n, nil
}

//WriteTo通过将脚本地址写入w来实现io.writerto。
func (sa *scriptAddress) WriteTo(w io.Writer) (n int64, err error) {
	var written int64

	hash := sa.address.ScriptAddress()
	datas := []interface{}{
		&hash,
		walletHash(hash),
make([]byte, 4), //版本
		&sa.flags,
		&sa.script,
		walletHash(sa.script),
		&sa.firstSeen,
		&sa.lastSeen,
		&sa.firstBlock,
		&sa.partialSyncHeight,
	}
	for _, data := range datas {
		if wt, ok := data.(io.WriterTo); ok {
			written, err = wt.WriteTo(w)
		} else {
			written, err = binaryWrite(w, binary.LittleEndian, data)
		}
		if err != nil {
			return n + written, err
		}
		n += written
	}
	return n, nil
}

//address返回btcadress的btcutil.AddressScriptHash。
func (sa *scriptAddress) Address() btcutil.Address {
	return sa.address
}

//addrhash返回脚本散列，实现addressinfo。
func (sa *scriptAddress) AddrHash() string {
	return string(sa.address.ScriptAddress())
}

//firstblock返回地址已知的第一个blockheight。
func (sa *scriptAddress) FirstBlock() int32 {
	return sa.firstBlock
}

//由于脚本地址总是
//导入的地址，而不是任何链的一部分。
func (sa *scriptAddress) Imported() bool {
	return true
}

//如果地址创建为更改地址，则change返回true。
func (sa *scriptAddress) Change() bool {
	return sa.flags.change
}

//compressed返回false，因为脚本地址从未被压缩。
//实现WalletAddress。
func (sa *scriptAddress) Compressed() bool {
	return false
}

//脚本返回由地址表示的脚本。它不应该
//被修改。
func (sa *scriptAddress) Script() []byte {
	return sa.script
}

//地址返回必须对脚本签名的地址列表。
func (sa *scriptAddress) Addresses() []btcutil.Address {
	return sa.addresses
}

//scriptclass返回地址的脚本类型。
func (sa *scriptAddress) ScriptClass() txscript.ScriptClass {
	return sa.class
}

//RequiredSigs返回脚本所需的签名数。
func (sa *scriptAddress) RequiredSigs() int {
	return sa.reqSigs
}

//syncstatus返回当前地址的syncstatus类型
//同步。对于未同步的类型，该值是第一次看到的记录值
//地址的块高度。
//
func (sa *scriptAddress) SyncStatus() SyncStatus {
	switch {
	case sa.flags.unsynced && !sa.flags.partialSync:
		return Unsynced(sa.firstBlock)
	case sa.flags.unsynced && sa.flags.partialSync:
		return PartialSync(sa.partialSyncHeight)
	default:
		return FullSync{}
	}
}

//setsyncstatus设置地址标志和可能的部分同步高度
//取决于S的类型。
func (sa *scriptAddress) setSyncStatus(s SyncStatus) {
	switch e := s.(type) {
	case Unsynced:
		sa.flags.unsynced = true
		sa.flags.partialSync = false
		sa.partialSyncHeight = 0

	case PartialSync:
		sa.flags.unsynced = true
		sa.flags.partialSync = true
		sa.partialSyncHeight = int32(e)

	case FullSync:
		sa.flags.unsynced = false
		sa.flags.partialSync = false
		sa.partialSyncHeight = 0
	}
}

//watchingcopy创建不带私钥的地址的副本。
//这用于在监视密钥存储中填充来自
//普通密钥存储。
func (sa *scriptAddress) watchingCopy(s *Store) walletAddress {
	return &scriptAddress{
		store:     s,
		address:   sa.address,
		addresses: sa.addresses,
		class:     sa.class,
		reqSigs:   sa.reqSigs,
		flags: scriptFlags{
			change:   sa.flags.change,
			unsynced: sa.flags.unsynced,
		},
		script:            sa.script,
		firstSeen:         sa.firstSeen,
		lastSeen:          sa.lastSeen,
		firstBlock:        sa.firstBlock,
		partialSyncHeight: sa.partialSyncHeight,
	}
}

func walletHash(b []byte) uint32 {
	sum := chainhash.DoubleHashB(b)
	return binary.LittleEndian.Uint32(sum)
}

//TODO（JRICK）添加错误更正。
func verifyAndFix(b []byte, chk uint32) error {
	if walletHash(b) != chk {
		return ErrChecksumMismatch
	}
	return nil
}

type kdfParameters struct {
	mem   uint64
	nIter uint32
	salt  [32]byte
}

//computekdfParameters将最佳猜测参数返回到
//内存硬键派生函数使计算持续
//使用不超过maxmem字节的内存时的targetsec秒。
func computeKdfParameters(targetSec float64, maxMem uint64) (*kdfParameters, error) {
	params := &kdfParameters{}
	if _, err := rand.Read(params.salt[:]); err != nil {
		return nil, err
	}

	testKey := []byte("This is an example key to test KDF iteration speed")

	memoryReqtBytes := uint64(1024)
	approxSec := float64(0)

	for approxSec <= targetSec/4 && memoryReqtBytes < maxMem {
		memoryReqtBytes *= 2
		before := time.Now()
		_ = keyOneIter(testKey, params.salt[:], memoryReqtBytes)
		approxSec = time.Since(before).Seconds()
	}

	allItersSec := float64(0)
	nIter := uint32(1)
for allItersSec < 0.02 { //这是一个直接从军械库来源得到的魔法数字。
		nIter *= 2
		before := time.Now()
		for i := uint32(0); i < nIter; i++ {
			_ = keyOneIter(testKey, params.salt[:], memoryReqtBytes)
		}
		allItersSec = time.Since(before).Seconds()
	}

	params.mem = memoryReqtBytes
	params.nIter = nIter

	return params, nil
}

func (params *kdfParameters) WriteTo(w io.Writer) (n int64, err error) {
	var written int64

	memBytes := make([]byte, 8)
	nIterBytes := make([]byte, 4)
	binary.LittleEndian.PutUint64(memBytes, params.mem)
	binary.LittleEndian.PutUint32(nIterBytes, params.nIter)
	chkedBytes := append(memBytes, nIterBytes...)
	chkedBytes = append(chkedBytes, params.salt[:]...)

	datas := []interface{}{
		&params.mem,
		&params.nIter,
		&params.salt,
		walletHash(chkedBytes),
make([]byte, 256-(binary.Size(params)+4)), //衬垫
	}
	for _, data := range datas {
		if written, err = binaryWrite(w, binary.LittleEndian, data); err != nil {
			return n + written, err
		}
		n += written
	}

	return n, nil
}

func (params *kdfParameters) ReadFrom(r io.Reader) (n int64, err error) {
	var read int64

//这些必须被读取，但不能直接保存到参数中。
	chkedBytes := make([]byte, 44)
	var chk uint32
	padding := make([]byte, 256-(binary.Size(params)+4))

	datas := []interface{}{
		chkedBytes,
		&chk,
		padding,
	}
	for _, data := range datas {
		if read, err = binaryRead(r, binary.LittleEndian, data); err != nil {
			return n + read, err
		}
		n += read
	}

//校验校验和
	if err = verifyAndFix(chkedBytes, chk); err != nil {
		return n, err
	}

//读参数
	buf := bytes.NewBuffer(chkedBytes)
	datas = []interface{}{
		&params.mem,
		&params.nIter,
		&params.salt,
	}
	for _, data := range datas {
		if err = binary.Read(buf, binary.LittleEndian, data); err != nil {
			return n, err
		}
	}

	return n, nil
}

type addrEntry struct {
	pubKeyHash160 [ripemd160.Size]byte
	addr          btcAddress
}

func (e *addrEntry) WriteTo(w io.Writer) (n int64, err error) {
	var written int64

//写入头
	if written, err = binaryWrite(w, binary.LittleEndian, addrHeader); err != nil {
		return n + written, err
	}
	n += written

//写入哈希
	if written, err = binaryWrite(w, binary.LittleEndian, &e.pubKeyHash160); err != nil {
		return n + written, err
	}
	n += written

//写btcadress
	written, err = e.addr.WriteTo(w)
	n += written
	return n, err
}

func (e *addrEntry) ReadFrom(r io.Reader) (n int64, err error) {
	var read int64

	if read, err = binaryRead(r, binary.LittleEndian, &e.pubKeyHash160); err != nil {
		return n + read, err
	}
	n += read

	read, err = e.addr.ReadFrom(r)
	return n + read, err
}

//scriptEntry是p2sh脚本的条目类型。
type scriptEntry struct {
	scriptHash160 [ripemd160.Size]byte
	script        scriptAddress
}

//writeto通过将条目写入w来实现io.writerto。
func (e *scriptEntry) WriteTo(w io.Writer) (n int64, err error) {
	var written int64

//写入头
	if written, err = binaryWrite(w, binary.LittleEndian, scriptHeader); err != nil {
		return n + written, err
	}
	n += written

//写入哈希
	if written, err = binaryWrite(w, binary.LittleEndian, &e.scriptHash160); err != nil {
		return n + written, err
	}
	n += written

//写btcadress
	written, err = e.script.WriteTo(w)
	n += written
	return n, err
}

//readFrom通过从e中读取条目来实现io.readerFrom。
func (e *scriptEntry) ReadFrom(r io.Reader) (n int64, err error) {
	var read int64

	if read, err = binaryRead(r, binary.LittleEndian, &e.scriptHash160); err != nil {
		return n + read, err
	}
	n += read

	read, err = e.script.ReadFrom(r)
	return n + read, err
}

//块戳定义一个块（按高度和唯一哈希），并且
//用于在区块链中标记密钥存储元素是
//同步到。
type BlockStamp struct {
	Hash   *chainhash.Hash
	Height int32
}
