
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

//该文件实现了基于范围的归零，从Go 1.5开始，
//使用Duff的设备优化。

package zero

import "math/big"

//字节将传递切片中的所有字节设置为零。这习惯于
//从内存中显式清除私钥材料。
//
//一般来说，更喜欢使用固定大小的归零函数（bytea*）
//当将字节归零时，因为它们比变量更有效
//大小为零的func字节。
func Bytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

//big int将传递的big int中的所有字节设置为零，然后将
//数值为0。这与简单地设置值不同，因为它
//专门清除底层字节，而只需设置值
//没有。这对于强制清除私钥非常有用。
func BigInt(x *big.Int) {
	b := x.Bits()
	for i := range b {
		b[i] = 0
	}
	x.SetInt64(0)
}
