
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

/*
包投票池为btcwallet提供投票池功能。

概述

投票池包的目的是使存储
使用m-of-n多交易的比特币。一个池可以有多个
序列，每个成员都有一组公钥（每个成员一个
在该池的序列中）和所需签名的最小数目（m）
需要花掉泳池里的硬币。每个成员都将持有一个私钥
匹配系列的一个公钥，并且至少有m个成员将
在使用游泳池的硬币时需要保持一致。

有关投票池及其一些用例的更多详细信息可以


此包取决于waddrmgr和walletdb包。

创建投票池

通过创建函数创建投票池。这个函数
接受将用于存储所有
与密钥为
池的ID。

加载现有池

通过加载函数加载现有的投票池，该函数接受
创建池和池盖时使用的数据库名称。

创建序列

可以通过CreateSeries方法创建序列，该方法接受
版本号、序列标识符、所需的签名数


存款地址

可以通过DepositscriptAddress创建存款地址。
方法，它从多个sig返回序列特定的p2sh地址
用序列的公钥和
根据给定的分支进行排序。构建多个SIG的过程
存款地址详细描述见
http://opentransactions.org/wiki/index.php/deposit_address_u（投票池）

替换序列


与CreateSeries方法相同的参数。

授权系列

出于安全原因，大多数私钥将离线维护，并且

必须使用授权系列方法，它只使用系列ID
以及与系列的一个公钥匹配的原始私钥。

开始提款

当从池中取出硬币时，我们采用了一个确定的过程。
以尽量降低协调交易签署的成本。为了
要做到这一点，游泳池的成员必须达成带外共识。
进程（<http://opentransactions.org/wiki/index.php/consonsistent\u process（投票池）>）
要定义以下参数，应将其传递给
开始提取方法：

 roundid：给定共识轮的唯一标识符。
 请求：投票池用户请求输出的列表
 startaddress:应该从中开始查找输入的seriesid、branch和indes
 LastSeriesID:最后一个系列的ID，我们应该从中获取输入
 changestart:要使用的第一个更改地址
 灰尘阈值：输入的最小Satoshis量需要视为合格。

然后，StartRetraction将选择给定地址中所有符合条件的输入
范围（遵循算法，网址为<http://opentransactions.org/wiki/index.php/input-selection-algorithm-pools>）
并使用它们来构造事务（<http://opentransactions.org/wiki/index.php/category:transaction-construction-algorithm-pools>
满足输出请求的。它返回一个包含

交易、包含在这些交易中的网络费用和输入
下次提取时使用的范围。

**/

package votingpool
