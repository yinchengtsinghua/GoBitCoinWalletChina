
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

package bdb_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/btcsuite/btcwallet/walletdb"
	_ "github.com/btcsuite/btcwallet/walletdb/bdb"
)

//dbtype是此驱动程序的数据库类型名称。
const dbType = "bdb"

//testcreateopenfail确保与创建和打开
//数据库处理正确。
func TestCreateOpenFail(t *testing.T) {
//确保尝试打开不存在的数据库时返回
//预期的错误。
	wantErr := walletdb.ErrDbDoesNotExist
	if _, err := walletdb.Open(dbType, "noexist.db"); err != wantErr {
		t.Errorf("Open: did not receive expected error - got %v, "+
			"want %v", err, wantErr)
		return
	}

//确保尝试打开的数据库的编号不正确
//参数返回预期的错误。
	wantErr = fmt.Errorf("invalid arguments to %s.Open -- expected "+
		"database path", dbType)
	if _, err := walletdb.Open(dbType, 1, 2, 3); err.Error() != wantErr.Error() {
		t.Errorf("Open: did not receive expected error - got %v, "+
			"want %v", err, wantErr)
		return
	}

//确保尝试打开的数据库类型无效
//第一个参数返回预期的错误。
	wantErr = fmt.Errorf("first argument to %s.Open is invalid -- "+
		"expected database path string", dbType)
	if _, err := walletdb.Open(dbType, 1); err.Error() != wantErr.Error() {
		t.Errorf("Open: did not receive expected error - got %v, "+
			"want %v", err, wantErr)
		return
	}

//确保尝试创建的数据库的编号不正确
//参数返回预期的错误。
	wantErr = fmt.Errorf("invalid arguments to %s.Create -- expected "+
		"database path", dbType)
	if _, err := walletdb.Create(dbType, 1, 2, 3); err.Error() != wantErr.Error() {
		t.Errorf("Create: did not receive expected error - got %v, "+
			"want %v", err, wantErr)
		return
	}

//确保尝试打开的数据库类型无效
//第一个参数返回预期的错误。
	wantErr = fmt.Errorf("first argument to %s.Create is invalid -- "+
		"expected database path string", dbType)
	if _, err := walletdb.Create(dbType, 1); err.Error() != wantErr.Error() {
		t.Errorf("Create: did not receive expected error - got %v, "+
			"want %v", err, wantErr)
		return
	}

//确保对已关闭数据库的操作返回预期的
//错误。
	dbPath := "createfail.db"
	db, err := walletdb.Create(dbType, dbPath)
	if err != nil {
		t.Errorf("Create: unexpected error: %v", err)
		return
	}
	defer os.Remove(dbPath)
	db.Close()

	wantErr = walletdb.ErrDbNotOpen
	if _, err := db.BeginReadTx(); err != wantErr {
		t.Errorf("Namespace: did not receive expected error - got %v, "+
			"want %v", err, wantErr)
		return
	}
}

//TestPersistence确保存储的值在关闭后仍然有效，并且
//重新打开数据库。
func TestPersistence(t *testing.T) {
//创建新数据库以运行测试。
	dbPath := "persistencetest.db"
	db, err := walletdb.Create(dbType, dbPath)
	if err != nil {
		t.Errorf("Failed to create test database (%s) %v", dbType, err)
		return
	}
	defer os.Remove(dbPath)
	defer db.Close()

//创建一个名称空间并将一些值放入其中，以便测试它们
//重新开放的存在。
	storeValues := map[string]string{
		"ns1key1": "foo1",
		"ns1key2": "foo2",
		"ns1key3": "foo3",
	}
	ns1Key := []byte("ns1")
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns1, err := tx.CreateTopLevelBucket(ns1Key)
		if err != nil {
			return err
		}

		for k, v := range storeValues {
			if err := ns1.Put([]byte(k), []byte(v)); err != nil {
				return fmt.Errorf("Put: unexpected error: %v", err)
			}
		}

		return nil
	})
	if err != nil {
		t.Errorf("ns1 Update: unexpected error: %v", err)
		return
	}

//关闭并重新打开数据库以确保值保持不变。
	db.Close()
	db, err = walletdb.Open(dbType, dbPath)
	if err != nil {
		t.Errorf("Failed to open test database (%s) %v", dbType, err)
		return
	}
	defer db.Close()

//确保之前存储在第三个命名空间中的值仍然存在
//并且是正确的。
	err = walletdb.View(db, func(tx walletdb.ReadTx) error {
		ns1 := tx.ReadBucket(ns1Key)
		if ns1 == nil {
			return fmt.Errorf("ReadTx.ReadBucket: unexpected nil root bucket")
		}

		for k, v := range storeValues {
			gotVal := ns1.Get([]byte(k))
			if !reflect.DeepEqual(gotVal, []byte(v)) {
				return fmt.Errorf("Get: key '%s' does not "+
					"match expected value - got %s, want %s",
					k, gotVal, v)
			}
		}

		return nil
	})
	if err != nil {
		t.Errorf("ns1 View: unexpected error: %v", err)
		return
	}
}
