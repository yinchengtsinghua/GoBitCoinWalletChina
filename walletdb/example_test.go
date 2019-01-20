
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
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/btcsuite/btcwallet/walletdb"
	_ "github.com/btcsuite/btcwallet/walletdb/bdb"
)

//此示例演示如何创建新数据库。
func ExampleCreate() {
//此示例假定导入了bdb（bolt db）驱动程序。
//
//进口（
//“github.com/btcsuite/btcwallet/walletdb（Github.com/btcsuite/btcwallet/walletdb）”。
//“github.com/btcsuite/btcwallet/walletdb/bdb”网站
//）

//创建一个数据库，并安排它在退出时关闭和删除。
//通常情况下，您不希望像这样立即删除数据库
//这一点，但在本例中已经完成了，以确保示例的清理
//自己站起来。
	dbPath := filepath.Join(os.TempDir(), "examplecreate.db")
	db, err := walletdb.Create("bdb", dbPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.Remove(dbPath)
	defer db.Close()

//输出：
}

//exampleNum用作exampleLoadDB函数中要提供的计数器
//每个示例的唯一数据库名称。
var exampleNum = 0

//示例中使用了exampleLoadDB来删除设置代码。
func exampleLoadDB() (walletdb.DB, func(), error) {
	dbName := fmt.Sprintf("exampleload%d.db", exampleNum)
	dbPath := filepath.Join(os.TempDir(), dbName)
	db, err := walletdb.Create("bdb", dbPath)
	if err != nil {
		return nil, nil, err
	}
	teardownFunc := func() {
		db.Close()
		os.Remove(dbPath)
	}
	exampleNum++

	return db, teardownFunc, err
}

//此示例演示如何创建新的顶级存储桶。
func ExampleDB_createTopLevelBucket() {
//为此示例加载数据库并将其调度为
//在出口处关闭和移除。有关更多信息，请参见创建示例
//有关此步骤的详细信息。
	db, teardownFunc, err := exampleLoadDB()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer teardownFunc()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer dbtx.Commit()

//根据需要在数据库中获取或创建存储桶。这个桶
//通常传递给特定子包的内容
//他们自己的工作区，不用担心钥匙冲突。
	bucketKey := []byte("walletsubpackage")
	bucket, err := dbtx.CreateTopLevelBucket(bucketKey)
	if err != nil {
		fmt.Println(err)
		return
	}

//防止未使用的错误。
	_ = bucket

//输出：
}

//此示例演示如何创建新数据库，从
//它，并对命名空间使用托管读写事务来存储
//并检索数据。
func Example_basicUsage() {
//此示例假定导入了bdb（bolt db）驱动程序。
//
//进口（
//“github.com/btcsuite/btcwallet/walletdb（Github.com/btcsuite/btcwallet/walletdb）”。
//“github.com/btcsuite/btcwallet/walletdb/bdb”网站
//）

//创建一个数据库，并安排它在退出时关闭和删除。
//通常情况下，您不希望像这样立即删除数据库
//这一点，但在本例中已经完成了，以确保示例的清理
//自己站起来。
	dbPath := filepath.Join(os.TempDir(), "exampleusage.db")
	db, err := walletdb.Create("bdb", dbPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.Remove(dbPath)
	defer db.Close()

//根据需要在数据库中获取或创建存储桶。这个桶
//通常传递给特定子包的内容
//他们自己的工作区，不用担心钥匙冲突。
	bucketKey := []byte("walletsubpackage")
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		bucket := tx.ReadWriteBucket(bucketKey)
		if bucket == nil {
			_, err = tx.CreateTopLevelBucket(bucketKey)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
		return
	}

//使用命名空间的update函数执行托管
//读写事务。事务将自动滚动
//返回所提供的内部函数是否返回非零错误。
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
//所有数据都存储在命名空间的根bucket中，
//或根桶的嵌套桶。不是真的
//必须将其存储在这样一个单独的变量中，但是
//为了示例的目的，这里已经完成了
//说明。
		rootBucket := tx.ReadWriteBucket(bucketKey)

//直接将键/值对存储在根bucket中。
		key := []byte("mykey")
		value := []byte("myvalue")
		if err := rootBucket.Put(key, value); err != nil {
			return err
		}

//把钥匙读回来，确保它匹配。
		if !bytes.Equal(rootBucket.Get(key), value) {
			return fmt.Errorf("unexpected value for key '%s'", key)
		}

//在根bucket下创建一个新的嵌套bucket。
		nestedBucketKey := []byte("mybucket")
		nestedBucket, err := rootBucket.CreateBucket(nestedBucketKey)
		if err != nil {
			return err
		}

//从上面设置在根存储桶中的键不
//存在于新的嵌套存储桶中。
		if nestedBucket.Get(key) != nil {
			return fmt.Errorf("key '%s' is not expected nil", key)
		}

		return nil
	})
	if err != nil {
		fmt.Println(err)
		return
	}

//输出：
}
