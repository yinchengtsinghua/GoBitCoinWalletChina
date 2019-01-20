
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

package walletdb_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/btcsuite/btcwallet/walletdb"
	_ "github.com/btcsuite/btcwallet/walletdb/bdb"
)

var (
//ignoredbtypes是运行测试时应忽略的类型
//迭代所有支持的数据库类型。这允许添加一些测试
//在允许其他测试的情况下，用于测试目的的伪驱动程序
//轻松迭代所有支持的驱动程序。
	ignoreDbTypes = map[string]bool{"createopenfail": true}
)

//TestAddDuplicateDriver确保添加重复驱动程序不会
//覆盖现有的。
func TestAddDuplicateDriver(t *testing.T) {
	supportedDrivers := walletdb.SupportedDrivers()
	if len(supportedDrivers) == 0 {
		t.Errorf("no backends to test")
		return
	}
	dbType := supportedDrivers[0]

//bogusCreatedB是一个函数，它充当一个伪造的创建和打开
//驱动程序功能并故意返回可能
//如果接口允许重复的驱动程序覆盖
//现有的一个。
	bogusCreateDB := func(args ...interface{}) (walletdb.DB, error) {
		return nil, fmt.Errorf("duplicate driver allowed for database "+
			"type [%v]", dbType)
	}

//创建一个试图替换现有驱动程序的驱动程序。设置其
//创建并打开导致测试失败的函数，如果
//它们被调用。
	driver := walletdb.Driver{
		DbType: dbType,
		Create: bogusCreateDB,
		Open:   bogusCreateDB,
	}
	err := walletdb.RegisterDriver(driver)
	if err != walletdb.ErrDbTypeRegistered {
		t.Errorf("unexpected duplicate driver registration error - "+
			"got %v, want %v", err, walletdb.ErrDbTypeRegistered)
	}

	dbPath := "dupdrivertest.db"
	db, err := walletdb.Create(dbType, dbPath)
	if err != nil {
		t.Errorf("failed to create database: %v", err)
		return
	}
	db.Close()
	_ = os.Remove(dbPath)

}

//testcreateopenfail确保打开或关闭时发生的错误
//正确处理数据库。
func TestCreateOpenFail(t *testing.T) {
//bogusCreatedB是一个函数，它充当一个伪造的创建和打开
//故意返回故障的驱动程序功能
//检测。
	dbType := "createopenfail"
	openError := fmt.Errorf("failed to create or open database for "+
		"database type [%v]", dbType)
	bogusCreateDB := func(args ...interface{}) (walletdb.DB, error) {
		return nil, openError
	}

//创建和添加在创建或打开时故意失败的驱动程序
//确保正确处理数据库打开和创建时的错误。
	driver := walletdb.Driver{
		DbType: dbType,
		Create: bogusCreateDB,
		Open:   bogusCreateDB,
	}
	walletdb.RegisterDriver(driver)

//确保使用新类型创建数据库失败，预期为
//错误。
	_, err := walletdb.Create(dbType)
	if err != openError {
		t.Errorf("expected error not received - got: %v, want %v", err,
			openError)
		return
	}

//确保打开具有新类型的数据库失败，并且
//错误。
	_, err = walletdb.Open(dbType)
	if err != openError {
		t.Errorf("expected error not received - got: %v, want %v", err,
			openError)
		return
	}
}

//testcreateopenunsupported确保尝试创建或打开
//正确处理了不支持的数据库类型。
func TestCreateOpenUnsupported(t *testing.T) {
//确保使用不支持的类型创建数据库失败
//期望误差。
	dbType := "unsupported"
	_, err := walletdb.Create(dbType)
	if err != walletdb.ErrDbUnknownType {
		t.Errorf("expected error not received - got: %v, want %v", err,
			walletdb.ErrDbUnknownType)
		return
	}

//确保打开不支持类型的数据库失败
//期望误差。
	_, err = walletdb.Open(dbType)
	if err != walletdb.ErrDbUnknownType {
		t.Errorf("expected error not received - got: %v, want %v", err,
			walletdb.ErrDbUnknownType)
		return
	}
}
