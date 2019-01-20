
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

package netparams

import "github.com/btcsuite/btcd/chaincfg"

//参数用于对各种网络的参数进行分组，例如
//网络和测试网络。
type Params struct {
	*chaincfg.Params
	RPCClientPort string
	RPCServerPort string
}

//mainnetparams包含特定于运行btcwallet和
//主网络上的BTCD（Wire.Mainnet）。
var MainNetParams = Params{
	Params:        &chaincfg.MainNetParams,
	RPCClientPort: "8334",
	RPCServerPort: "8332",
}

//testnet3参数包含特定于运行btcwallet和
//测试网络上的BTCD（版本3）（Wire.TestNet3）。
var TestNet3Params = Params{
	Params:        &chaincfg.TestNet3Params,
	RPCClientPort: "18334",
	RPCServerPort: "18332",
}

//simnetparams包含特定于模拟测试网络的参数
//（电线，SimNet）
var SimNetParams = Params{
	Params:        &chaincfg.SimNetParams,
	RPCClientPort: "18556",
	RPCServerPort: "18554",
}
