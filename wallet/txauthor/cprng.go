
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package txauthor

import (
	"crypto/rand"
	"encoding/binary"
	mrand "math/rand"
	"sync"
)

//CPRNG是一种密码随机种子数学/RAND prng。它是种子的
//
//用于并发访问。
var cprng = cprngType{}

type cprngType struct {
	r  *mrand.Rand
	mu sync.Mutex
}

func init() {
	buf := make([]byte, 8)
	_, err := rand.Read(buf)
	if err != nil {
		panic("Failed to seed prng: " + err.Error())
	}

	seed := int64(binary.LittleEndian.Uint64(buf))
	cprng.r = mrand.New(mrand.NewSource(seed))
}

func (c *cprngType) Int31n(n int32) int32 {
defer c.mu.Unlock() //INT31N可能会恐慌
	c.mu.Lock()
	return c.r.Int31n(n)
}
