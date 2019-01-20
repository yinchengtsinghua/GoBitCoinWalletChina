
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
包bdb实现了一个使用boltdb作为备份的walletdb实例
数据存储。

用法

此包只是walletdb包的驱动程序，并提供数据库
“BDB”类型。open和create函数所采用的唯一参数是
作为字符串的数据库路径：

 db，err：=walletdb.open（“bdb”，“path/to/database.db”）。
 如果犯错！= nIL{
  //句柄错误
 }

 db，err：=walletdb.create（“bdb”，“path/to/database.db”）。
 如果犯错！= nIL{
  //句柄错误
 }
**/

package bdb
