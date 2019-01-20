
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

package snacl

import (
	"bytes"
	"testing"
)

var (
	password = []byte("sikrit")
	message  = []byte("this is a secret message of sorts")
	key      *SecretKey
	params   []byte
	blob     []byte
)

func TestNewSecretKey(t *testing.T) {
	var err error
	key, err = NewSecretKey(&password, DefaultN, DefaultR, DefaultP)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestMarshalSecretKey(t *testing.T) {
	params = key.Marshal()
}

func TestUnmarshalSecretKey(t *testing.T) {
	var sk SecretKey
	if err := sk.Unmarshal(params); err != nil {
		t.Errorf("unexpected unmarshal error: %v", err)
		return
	}

	if err := sk.DeriveKey(&password); err != nil {
		t.Errorf("unexpected DeriveKey error: %v", err)
		return
	}

	if !bytes.Equal(sk.Key[:], key.Key[:]) {
		t.Errorf("keys not equal")
	}
}

func TestUnmarshalSecretKeyInvalid(t *testing.T) {
	var sk SecretKey
	if err := sk.Unmarshal(params); err != nil {
		t.Errorf("unexpected unmarshal error: %v", err)
		return
	}

	p := []byte("wrong password")
	if err := sk.DeriveKey(&p); err != ErrInvalidPassword {
		t.Errorf("wrong password didn't fail")
		return
	}
}

func TestEncrypt(t *testing.T) {
	var err error

	blob, err = key.Encrypt(message)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestDecrypt(t *testing.T) {
	decryptedMessage, err := key.Decrypt(blob)
	if err != nil {
		t.Error(err)
		return
	}

	if !bytes.Equal(decryptedMessage, message) {
		t.Errorf("decryption failed")
		return
	}
}

func TestDecryptCorrupt(t *testing.T) {
	blob[len(blob)-15] = blob[len(blob)-15] + 1
	_, err := key.Decrypt(blob)
	if err == nil {
		t.Errorf("corrupt message decrypted")
		return
	}
}

func TestZero(t *testing.T) {
	var zeroKey [32]byte

	key.Zero()
	if !bytes.Equal(key.Key[:], zeroKey[:]) {
		t.Errorf("zero key failed")
	}
}

func TestDeriveKey(t *testing.T) {
	if err := key.DeriveKey(&password); err != nil {
		t.Errorf("unexpected DeriveKey key failure: %v", err)
	}
}

func TestDeriveKeyInvalid(t *testing.T) {
	bogusPass := []byte("bogus")
	if err := key.DeriveKey(&bogusPass); err != ErrInvalidPassword {
		t.Errorf("unexpected DeriveKey key failure: %v", err)
	}
}
