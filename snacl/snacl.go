
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package snacl

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"io"
	"runtime/debug"

	"github.com/btcsuite/btcwallet/internal/zero"
	"github.com/btcsuite/golangcrypto/nacl/secretbox"
	"github.com/btcsuite/golangcrypto/scrypt"
)

var (
	prng = rand.Reader
)

//错误类型和消息。
var (
	ErrInvalidPassword = errors.New("invalid password")
	ErrMalformed       = errors.New("malformed data")
	ErrDecryptFailed   = errors.New("unable to decrypt")
)

//加密方案所需的各种常量。
const (
//为了方便起见，请在这里暴露Secretbox的头顶警察。
	Overhead  = secretbox.Overhead
	KeySize   = 32
	NonceSize = 24
DefaultN  = 16384 //2 ^ 14
	DefaultR  = 8
	DefaultP  = 1
)

//CryptoKey表示可用于加密和解密的密钥
//数据。
type CryptoKey [KeySize]byte

//Encrypt对传递的数据进行加密。
func (ck *CryptoKey) Encrypt(in []byte) ([]byte, error) {
	var nonce [NonceSize]byte
	_, err := io.ReadFull(prng, nonce[:])
	if err != nil {
		return nil, err
	}
	blob := secretbox.Seal(nil, in, &nonce, (*[KeySize]byte)(ck))
	return append(nonce[:], blob...), nil
}

//解密解密传递的数据。必须是加密的输出
//功能。
func (ck *CryptoKey) Decrypt(in []byte) ([]byte, error) {
	if len(in) < NonceSize {
		return nil, ErrMalformed
	}

	var nonce [NonceSize]byte
	copy(nonce[:], in[:NonceSize])
	blob := in[NonceSize:]

	opened, ok := secretbox.Open(nil, blob, &nonce, (*[KeySize]byte)(ck))
	if !ok {
		return nil, ErrDecryptFailed
	}

	return opened, nil
}

//零通过手动调零所有内存来清除键。这是为了安全起见
//良心应用程序，希望在使用后将记忆归零
//而不是等到垃圾回收站回收。这个
//此呼叫后密钥不再可用。
func (ck *CryptoKey) Zero() {
	zero.Bytea32((*[KeySize]byte)(ck))
}

//generatecryptokey生成一个新的crypotgraphically随机键。
func GenerateCryptoKey() (*CryptoKey, error) {
	var key CryptoKey
	_, err := io.ReadFull(prng, key[:])
	if err != nil {
		return nil, err
	}

	return &key, nil
}

//参数不是秘密的，可以以纯文本形式存储。
type Parameters struct {
	Salt   [KeySize]byte
	Digest [sha256.Size]byte
	N      int
	R      int
	P      int
}

//SecretKey包含一个加密密钥以及从
//口令。它应该只在内存中使用。
type SecretKey struct {
	Key        *CryptoKey
	Parameters Parameters
}

//派生键填充关键字段。
func (sk *SecretKey) deriveKey(password *[]byte) error {
	key, err := scrypt.Key(*password, sk.Parameters.Salt[:],
		sk.Parameters.N,
		sk.Parameters.R,
		sk.Parameters.P,
		len(sk.Key))
	if err != nil {
		return err
	}
	copy(sk.Key[:], key)
	zero.Bytes(key)

//我不喜欢强制垃圾收集，但scrypt会分配一个
//大量的内存，在没有GC循环的情况下将其重新调用
//中间意味着你最终需要两倍的内存。为了
//例如，如果您的加密参数需要1GB和
//你连续两次调用它，如果没有这个，你最终会分配2GB
//自从第一个GB可能还没有发布。
	debug.FreeOSMemory()

	return nil
}

//marshal返回参数字段，该字段以适合于
//存储。这样的结果可以以明文形式存储。
func (sk *SecretKey) Marshal() []byte {
	params := &sk.Parameters

//参数的编组格式如下：
//<salt><digest><n><r><p>
//
//keysize+sha256.size+n（8字节）+r（8字节）+p（8字节）
	marshalled := make([]byte, KeySize+sha256.Size+24)

	b := marshalled
	copy(b[:KeySize], params.Salt[:])
	b = b[KeySize:]
	copy(b[:sha256.Size], params.Digest[:])
	b = b[sha256.Size:]
	binary.LittleEndian.PutUint64(b[:8], uint64(params.N))
	b = b[8:]
	binary.LittleEndian.PutUint64(b[:8], uint64(params.R))
	b = b[8:]
	binary.LittleEndian.PutUint64(b[:8], uint64(params.P))

	return marshalled
}

//取消标记取消标记从
//将密码改为sk。
func (sk *SecretKey) Unmarshal(marshalled []byte) error {
	if sk.Key == nil {
		sk.Key = (*CryptoKey)(&[KeySize]byte{})
	}

//参数的编组格式如下：
//<salt><digest><n><r><p>
//
//keysize+sha256.size+n（8字节）+r（8字节）+p（8字节）
	if len(marshalled) != KeySize+sha256.Size+24 {
		return ErrMalformed
	}

	params := &sk.Parameters
	copy(params.Salt[:], marshalled[:KeySize])
	marshalled = marshalled[KeySize:]
	copy(params.Digest[:], marshalled[:sha256.Size])
	marshalled = marshalled[sha256.Size:]
	params.N = int(binary.LittleEndian.Uint64(marshalled[:8]))
	marshalled = marshalled[8:]
	params.R = int(binary.LittleEndian.Uint64(marshalled[:8]))
	marshalled = marshalled[8:]
	params.P = int(binary.LittleEndian.Uint64(marshalled[:8]))

	return nil
}

//零使基础密钥归零，同时保持参数不变。
//这有效地使密钥在通过
//派生键函数。
func (sk *SecretKey) Zero() {
	sk.Key.Zero()
}

//DeriveKey派生基础密钥并确保它与
//需要摘要。只应在以前调用
//零函数或在初始解组时。
func (sk *SecretKey) DeriveKey(password *[]byte) error {
	if err := sk.deriveKey(password); err != nil {
		return err
	}

//验证口令
	digest := sha256.Sum256(sk.Key[:])
	if subtle.ConstantTimeCompare(digest[:], sk.Parameters.Digest[:]) != 1 {
		return ErrInvalidPassword
	}

	return nil
}

//加密以字节加密并返回JSON blob。
func (sk *SecretKey) Encrypt(in []byte) ([]byte, error) {
	return sk.Key.Encrypt(in)
}

//
func (sk *SecretKey) Decrypt(in []byte) ([]byte, error) {
	return sk.Key.Decrypt(in)
}

//NewSecretKey基于传递的参数返回SecretKey结构。
func NewSecretKey(password *[]byte, N, r, p int) (*SecretKey, error) {
	sk := SecretKey{
		Key: (*CryptoKey)(&[KeySize]byte{}),
	}
//设置参数
	sk.Parameters.N = N
	sk.Parameters.R = r
	sk.Parameters.P = p
	_, err := io.ReadFull(prng, sk.Parameters.Salt[:])
	if err != nil {
		return nil, err
	}

//派生密钥
	err = sk.deriveKey(password)
	if err != nil {
		return nil, err
	}

//商店文摘
	sk.Parameters.Digest = sha256.Sum256(sk.Key[:])

	return &sk, nil
}
