
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

package bdb

import (
	"io"
	"os"

	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/coreos/bbolt"
)

//converterr将某些螺栓错误转换为等效的walletdb错误。
func convertErr(err error) error {
	switch err {
//数据库打开/创建错误。
	case bbolt.ErrDatabaseNotOpen:
		return walletdb.ErrDbNotOpen
	case bbolt.ErrInvalid:
		return walletdb.ErrInvalid

//事务错误。
	case bbolt.ErrTxNotWritable:
		return walletdb.ErrTxNotWritable
	case bbolt.ErrTxClosed:
		return walletdb.ErrTxClosed

//值/存储桶错误。
	case bbolt.ErrBucketNotFound:
		return walletdb.ErrBucketNotFound
	case bbolt.ErrBucketExists:
		return walletdb.ErrBucketExists
	case bbolt.ErrBucketNameRequired:
		return walletdb.ErrBucketNameRequired
	case bbolt.ErrKeyRequired:
		return walletdb.ErrKeyRequired
	case bbolt.ErrKeyTooLarge:
		return walletdb.ErrKeyTooLarge
	case bbolt.ErrValueTooLarge:
		return walletdb.ErrValueTooLarge
	case bbolt.ErrIncompatibleValue:
		return walletdb.ErrIncompatibleValue
	}

//如果以上都不适用，则返回原始错误。
	return err
}

//事务表示数据库事务。它可以是只读的，也可以是
//读写并实现walletdb-tx接口。交易
//提供一个根存储桶，所有读取和写入都针对它进行。
type transaction struct {
	boltTx *bbolt.Tx
}

func (tx *transaction) ReadBucket(key []byte) walletdb.ReadBucket {
	return tx.ReadWriteBucket(key)
}

func (tx *transaction) ReadWriteBucket(key []byte) walletdb.ReadWriteBucket {
	boltBucket := tx.boltTx.Bucket(key)
	if boltBucket == nil {
		return nil
	}
	return (*bucket)(boltBucket)
}

func (tx *transaction) CreateTopLevelBucket(key []byte) (walletdb.ReadWriteBucket, error) {
	boltBucket, err := tx.boltTx.CreateBucket(key)
	if err != nil {
		return nil, convertErr(err)
	}
	return (*bucket)(boltBucket), nil
}

func (tx *transaction) DeleteTopLevelBucket(key []byte) error {
	err := tx.boltTx.DeleteBucket(key)
	if err != nil {
		return convertErr(err)
	}
	return nil
}

//提交提交通过根bucket进行的所有更改，以及
//它的所有子存储桶都要持久存储。
//
//此函数是walletdb.tx接口实现的一部分。
func (tx *transaction) Commit() error {
	return convertErr(tx.boltTx.Commit())
}

//回滚将撤消对根bucket和所有
//它的子桶。
//
//此函数是walletdb.tx接口实现的一部分。
func (tx *transaction) Rollback() error {
	return convertErr(tx.boltTx.Rollback())
}

//bucket是一种内部类型，用于表示键/值对的集合
//实现了walletdb bucket接口。
type bucket bbolt.Bucket

//强制bucket实现walletdb bucket接口。
var _ walletdb.ReadWriteBucket = (*bucket)(nil)

//NestedReadWriteBucket检索具有给定键的嵌套Bucket。退换商品
//如果桶不存在，则为零。
//
//此函数是walletdb.readWriteBucket接口实现的一部分。
func (b *bucket) NestedReadWriteBucket(key []byte) walletdb.ReadWriteBucket {
	boltBucket := (*bbolt.Bucket)(b).Bucket(key)
//不要向nil指针返回非nil接口。
	if boltBucket == nil {
		return nil
	}
	return (*bucket)(boltBucket)
}

func (b *bucket) NestedReadBucket(key []byte) walletdb.ReadBucket {
	return b.NestedReadWriteBucket(key)
}

//createBucket创建并返回具有给定键的新嵌套bucket。
//如果bucket已经存在，则返回errbacketexists，errbacketnameRequired
//如果键为空，或者如果键值为其他值，则返回errcompatibleValue
//无效。
//
//此函数是walletdb.bucket接口实现的一部分。
func (b *bucket) CreateBucket(key []byte) (walletdb.ReadWriteBucket, error) {
	boltBucket, err := (*bbolt.Bucket)(b).CreateBucket(key)
	if err != nil {
		return nil, convertErr(err)
	}
	return (*bucket)(boltBucket), nil
}

//CreateBacketifnotexists创建并返回一个新的嵌套bucket，其中
//给定的键（如果它不存在）。返回errbacketnameRequired，如果
//如果键值无效，则键为空或errcompatibleValue。
//
//此函数是walletdb.bucket接口实现的一部分。
func (b *bucket) CreateBucketIfNotExists(key []byte) (walletdb.ReadWriteBucket, error) {
	boltBucket, err := (*bbolt.Bucket)(b).CreateBucketIfNotExists(key)
	if err != nil {
		return nil, convertErr(err)
	}
	return (*bucket)(boltBucket), nil
}

//deletenestedback删除具有给定键的嵌套bucket。退换商品
//如果尝试对只读事务执行errtxnotwritable，并且
//如果指定的存储桶不存在，则返回errbacketnotfound。
//
//此函数是walletdb.bucket接口实现的一部分。
func (b *bucket) DeleteNestedBucket(key []byte) error {
	return convertErr((*bbolt.Bucket)(b).DeleteBucket(key))
}

//foreach使用bucket中的每个键/值对调用传递的函数。
//这包括嵌套的bucket，在这种情况下，值为nil，但它不
//在这些嵌套存储桶中包含键/值对。
//
//注意：此函数返回的值仅在
//交易。在事务结束后尝试访问它们将
//可能导致访问冲突。
//
//此函数是walletdb.bucket接口实现的一部分。
func (b *bucket) ForEach(fn func(k, v []byte) error) error {
	return convertErr((*bbolt.Bucket)(b).ForEach(fn))
}

//PUT将指定的键/值对保存到存储桶中。不需要的钥匙
//添加已存在的键，覆盖已存在的键。退换商品
//
//
//
func (b *bucket) Put(key, value []byte) error {
	return convertErr((*bbolt.Bucket)(b).Put(key, value))
}

//
//
//
//注意：此函数返回的值仅在
//交易。在事务结束后尝试访问它
//可能导致访问冲突。
//
//此函数是walletdb.bucket接口实现的一部分。
func (b *bucket) Get(key []byte) []byte {
	return (*bbolt.Bucket)(b).Get(key)
}

//删除从存储桶中删除指定的键。删除一个键
//不存在不返回错误。如果尝试，则返回errtxnotwritable
//针对只读事务。
//
//此函数是walletdb.bucket接口实现的一部分。
func (b *bucket) Delete(key []byte) error {
	return convertErr((*bbolt.Bucket)(b).Delete(key))
}

func (b *bucket) ReadCursor() walletdb.ReadCursor {
	return b.ReadWriteCursor()
}

//readwritecursor返回一个新的光标，允许在bucket的
//键/值对和嵌套存储桶的前向或后向顺序。
//
//此函数是walletdb.bucket接口实现的一部分。
func (b *bucket) ReadWriteCursor() walletdb.ReadWriteCursor {
	return (*cursor)((*bbolt.Bucket)(b).Cursor())
}

//光标表示位于键/值对和嵌套存储桶上的光标
//桶。
//
//注意，在bucket更改和任何
//对存储桶的修改，但光标除外。删除，无效
//光标。无效后，必须重新定位光标或键
//并且返回的值可能是不可预测的。
type cursor bbolt.Cursor

//删除删除光标所在的当前键/值对
//使光标无效。如果尝试只读，则返回errtxnotwritable
//事务，或当光标指向
//嵌套桶。
//
//此函数是walletdb.cursor接口实现的一部分。
func (c *cursor) Delete() error {
	return convertErr((*bbolt.Cursor)(c).Delete())
}

//首先将光标定位在第一个键/值对上，然后返回该对。
//
//此函数是walletdb.cursor接口实现的一部分。
func (c *cursor) First() (key, value []byte) {
	return (*bbolt.Cursor)(c).First()
}

//Last将光标定位在最后一个键/值对上并返回该对。
//
//此函数是walletdb.cursor接口实现的一部分。
func (c *cursor) Last() (key, value []byte) {
	return (*bbolt.Cursor)(c).Last()
}

//下一步将光标向前移动一个键/值对，并返回新对。
//
//此函数是walletdb.cursor接口实现的一部分。
func (c *cursor) Next() (key, value []byte) {
	return (*bbolt.Cursor)(c).Next()
}

//prev将光标向后移动一个键/值对，并返回新对。
//
//此函数是walletdb.cursor接口实现的一部分。
func (c *cursor) Prev() (key, value []byte) {
	return (*bbolt.Cursor)(c).Prev()
}

//SEEK将光标定位在传递的SEEK键上。如果密钥不存在，
//搜索后光标移动到下一个键。返回新对。
//
//此函数是walletdb.cursor接口实现的一部分。
func (c *cursor) Seek(seek []byte) (key, value []byte) {
	return (*bbolt.Cursor)(c).Seek(seek)
}

//db表示持久化和实现的命名空间集合
//walletdb.db接口。所有数据库访问都是通过
//通过特定命名空间获取的事务。
type db bbolt.DB

//enforce db实现walletdb.db接口。
var _ walletdb.DB = (*db)(nil)

func (db *db) beginTx(writable bool) (*transaction, error) {
	boltTx, err := (*bbolt.DB)(db).Begin(writable)
	if err != nil {
		return nil, convertErr(err)
	}
	return &transaction{boltTx: boltTx}, nil
}

func (db *db) BeginReadTx() (walletdb.ReadTx, error) {
	return db.beginTx(false)
}

func (db *db) BeginReadWriteTx() (walletdb.ReadWriteTx, error) {
	return db.beginTx(true)
}

//copy将数据库的副本写入提供的写入程序。此呼叫将
//启动只读事务以执行所有操作。
//
//此函数是walletdb.db接口实现的一部分。
func (db *db) Copy(w io.Writer) error {
	return convertErr((*bbolt.DB)(db).View(func(tx *bbolt.Tx) error {
		return tx.Copy(w)
	}))
}

//CLOSE干净地关闭数据库并同步所有数据。
//
//此函数是walletdb.db接口实现的一部分。
func (db *db) Close() error {
	return convertErr((*bbolt.DB)(db).Close())
}

//filesexists报告命名文件或目录是否存在。
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

//opendb以提供的路径打开数据库。walletdb.errdbdoesnotex列表
//如果数据库不存在且未设置创建标志，则返回。
func openDB(dbPath string, create bool) (walletdb.DB, error) {
	if !create && !fileExists(dbPath) {
		return nil, walletdb.ErrDbDoesNotExist
	}

	boltDB, err := bbolt.Open(dbPath, 0600, nil)
	return (*db)(boltDB), convertErr(err)
}
