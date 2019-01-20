
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



概述






BIP044这个设置意味着只要用户写下种子（甚至更好
是为种子使用助记键），它们的所有地址和私钥都可以



一种是面向公众的数据，另一种是面向私人的数据
数据。可以为不需要密码的呼叫者硬编码公共密码。
如果只有一个
需要密码。这些选择提供可用性与安全性

需要比普通的EC公钥更小心地处理，因为它们可以


因为一个帐户将允许他们知道你将使用的所有派生地址和






键）用于公共、私有和脚本数据。一些例子包括付款
与按脚本付费哈希关联的地址、扩展的HD密钥和脚本
地址。这个方案使得更改密码更加有效，因为只有

（这是重新键入*所需要的）。这将导致完全加密







通常通过将地址管理器锁定来完成，这意味着没有私有的

私有密钥和脚本密钥将被解密并加载到内存中，然后




锁定和解锁











层次确定的钥匙链，允许所有地址和私钥


这个功能。地址管理器在创建后立即被锁定。



通过open函数打开现有的地址管理器。这个函数
















关于地址，如它们的私钥和脚本。










地址。

请求现有地址

除了生成新地址外，对旧地址的访问通常
必修的。最值得注意的是，签署交易以赎回它们。这个
地址函数提供此功能并返回ManagedAddress。

导入地址

而建议的方法是使用上面讨论的链接地址
因为它们可以被确定地再生，以避免长期的资金损失
由于用户拥有主种子，因此已经存在许多地址，
因此，该包提供了导入现有私有
钱包导入格式（WIF）的密钥，以及相关的公共密钥和
地址。

导入脚本

为了支持按脚本付费散列事务，脚本必须安全


交易。





它本身不知道哪些地址是同步的。只有它


网络


适当的地址并拒绝导入的地址和脚本


错误





将包含在错误字段中。

比特币改进建议

该包包括以下BIP概述的概念：

  bip0032（https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki）
  bip0043（https://github.com/bitcoin/bips/blob/master/bip-0043.mediawiki）
  bip0044（https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki）
**/

package waddrmgr
