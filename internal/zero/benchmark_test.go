
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

package zero_test

import (
	"testing"

	. "github.com/btcsuite/btcwallet/internal/zero"
)

var (
bytes32 = make([]byte, 32) //典型钥匙尺寸
bytes64 = make([]byte, 64) //密码哈希大小
	bytea32 = new([32]byte)
	bytea64 = new([64]byte)
)

//xor是这个包中的“慢”字节归零实现。
//最初被替换。如果此函数比
//此包在将来的go版本中导出的函数（可能
//通过调用runtime.memclr），将“优化”版本替换为
//这个。
func xor(b []byte) {
	for i := range b {
		b[i] ^= b[i]
	}
}

//zrange是一个可选的零实现，当前
//比这个包提供的函数慢，可能更快
//在未来的发布中。切换到此或XOR实现
//如果他们变得更快。
func zrange(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func BenchmarkXor32(b *testing.B) {
	for i := 0; i < b.N; i++ {
		xor(bytes32)
	}
}

func BenchmarkXor64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		xor(bytes64)
	}
}

func BenchmarkRange32(b *testing.B) {
	for i := 0; i < b.N; i++ {
		zrange(bytes32)
	}
}

func BenchmarkRange64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		zrange(bytes64)
	}
}

func BenchmarkBytes32(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Bytes(bytes32)
	}
}

func BenchmarkBytes64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Bytes(bytes64)
	}
}

func BenchmarkBytea32(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Bytea32(bytea32)
	}
}

func BenchmarkBytea64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Bytea64(bytea64)
	}
}
