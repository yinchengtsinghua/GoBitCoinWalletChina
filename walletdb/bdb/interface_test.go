
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

//此文件将被复制到每个后端驱动程序目录中。各
//驱动程序应该有自己的驱动程序\test.go文件，该文件创建一个数据库和
//调用此文件中的testinterface函数以确保驱动程序正确
//实现接口。有关工作示例，请参阅BDB后端驱动程序。
//
//注意：将此文件复制到后端驱动程序文件夹时，包名称
//需要相应更改。

package bdb_test

import (
	"os"
	"testing"

	"github.com/btcsuite/btcwallet/walletdb/walletdbtest"
)

//testinterface执行此数据库驱动程序的所有接口测试。
func TestInterface(t *testing.T) {
	dbPath := "interfacetest.db"
	defer os.RemoveAll(dbPath)
	walletdbtest.TestInterface(t, dbType, dbPath)
}
