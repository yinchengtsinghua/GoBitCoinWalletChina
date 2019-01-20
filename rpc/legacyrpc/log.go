
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package legacyrpc

import "github.com/btcsuite/btclog"

var log = btclog.Disabled

//uselogger设置包范围的记录器。对此函数的任何调用都必须
//在创建和使用服务器之前创建（它不是并发安全的）。
func UseLogger(logger btclog.Logger) {
	log = logger
}
