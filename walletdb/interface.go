
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

//这个界面受到了
//https://github.com/boltdb/bolt作者：Ben B.Johnson。

package walletdb

import "io"

//readtx表示只能用于读取的数据库事务。如果
//必须进行数据库更新，请使用readwritetx。
type ReadTx interface {
//read bucket打开根bucket进行只读访问。如果桶
//键描述的不存在，返回nil。
	ReadBucket(key []byte) ReadBucket

//回滚将关闭事务，如果
//数据库被写入事务修改。
	Rollback() error
}

//readwritetx表示可用于两次读取的数据库事务
//然后写。当只需要读取时，考虑使用readtx。
type ReadWriteTx interface {
	ReadTx

//read write bucket打开根bucket进行读/写访问。如果
//键描述的bucket不存在，返回nil。
	ReadWriteBucket(key []byte) ReadWriteBucket

//CreateTopLevelBucket为键创建顶级存储桶，如果它
//不存在。它返回的新创建的存储桶。
	CreateTopLevelBucket(key []byte) (ReadWriteBucket, error)

//DeleteTopLevelBucket删除键的顶级存储桶。这个
//如果找不到存储桶或键只键入一个值，则会出错。
//而不是水桶。
	DeleteTopLevelBucket(key []byte) error

//提交提交已在事务根目录上的所有更改
//存储桶及其所有子存储桶进行持久存储。
	Commit() error
}

//readbucket表示一个bucket（数据库中的层次结构）
//只允许执行读取操作。
type ReadBucket interface {
//NestedReadBucket检索具有给定键的嵌套Bucket。
//如果bucket不存在，则返回nil。
	NestedReadBucket(key []byte) ReadBucket

//foreach调用传递的函数，其中每个键/值对位于
//桶。这包括嵌套的bucket，在这种情况下，值
//为零，但不包括键/值对
//嵌套桶。
//
//注意：此函数返回的值仅在
//交易。在事务结束后尝试访问它们
//导致未定义的行为。此约束可防止
//数据复制并支持内存映射数据库
//实施。
	ForEach(func(k, v []byte) error) error

//get返回给定键的值。如果键为，则返回nil
//此存储桶（或嵌套存储桶）中不存在。
//
//注意：此函数返回的值仅在
//交易。在事务结束后尝试访问它
//导致未定义的行为。此约束可防止
//数据复制并支持内存映射数据库
//实施。
	Get(key []byte) []byte

	ReadCursor() ReadCursor
}

//readwritebucket表示一个bucket（在
//数据库），允许执行读写操作。
type ReadWriteBucket interface {
	ReadBucket

//NestedReadWriteBucket检索具有给定键的嵌套Bucket。
//如果bucket不存在，则返回nil。
	NestedReadWriteBucket(key []byte) ReadWriteBucket

//createBucket创建并返回具有给定
//关键。如果bucket已经存在，则返回errbacketexists，
//如果密钥为空或errUncompatibleValue，则需要errBucketnameRequired
//如果键值对特定数据库无效
//实施。其他错误可能取决于
//实施。
	CreateBucket(key []byte) (ReadWriteBucket, error)

//CreateBacketifnotexists创建并返回一个新的嵌套bucket，其中
//给定的键（如果它不存在）。退换商品
//如果密钥为空或errUncompatibleValue，则需要errBucketnameRequired
//如果键值对特定数据库无效
//后端。其他错误可能取决于实现。
	CreateBucketIfNotExists(key []byte) (ReadWriteBucket, error)

//deletenestedback删除具有给定键的嵌套bucket。
//如果对只读事务尝试，则返回errtxnotwritable
//如果指定的存储桶不存在，则返回errbacketNotFound。
	DeleteNestedBucket(key []byte) error

//PUT将指定的键/值对保存到存储桶中。做的钥匙
//已添加不存在的键，已存在的键是
//改写。如果尝试对
//只读事务。
	Put(key, value []byte) error

//删除从存储桶中删除指定的键。删除密钥
//不存在不会返回错误。退换商品
//如果尝试对只读事务执行errtxnotwritable操作。
	Delete(key []byte) error

//cursor返回一个新的光标，允许在bucket的
//键/值对和嵌套存储桶的前向或后向顺序。
	ReadWriteCursor() ReadWriteCursor
}

//readCursor表示可以定位在开始位置或
//桶的键/值对结束，并在桶中迭代对。
//此类型只允许执行数据库读取操作。
type ReadCursor interface {
//首先将光标定位在第一个键/值对上并返回
//这对。
	First() (key, value []byte)

//Last将光标定位在最后一个键/值对上，并返回
//一对。
	Last() (key, value []byte)

//下一步将光标向前移动一个键/值对，并返回新的
//一对。
	Next() (key, value []byte)

//prev将光标向后移动一个键/值对，并返回新的
//一对。
	Prev() (key, value []byte)

//SEEK将光标定位在传递的SEEK键上。如果钥匙确实
//不存在，搜索后光标移到下一个键。退换商品
//新的一对。
	Seek(seek []byte) (key, value []byte)
}

//readWriteCursor表示可以定位在
//桶的键/值对的开始或结束，并在
//桶。允许此抽象执行数据库读写操作
//操作。
type ReadWriteCursor interface {
	ReadCursor

//删除删除光标所在的当前键/值对
//使光标无效。如果尝试返回errcompatibleValue
//当光标指向嵌套的bucket时。
	Delete() error
}

//bucket is empty返回桶是否为空，即是否存在
//没有键/值对或嵌套存储桶。
func BucketIsEmpty(bucket ReadBucket) bool {
	k, v := bucket.ReadCursor().First()
	return k == nil && v == nil
}

//DB表示ACID数据库。所有数据库访问都是通过
//读或读+写事务。
type DB interface {
//beginreadtx打开数据库读取事务。
	BeginReadTx() (ReadTx, error)

//BeginReadWriteTx打开数据库读写事务。
	BeginReadWriteTx() (ReadWriteTx, error)

//copy将数据库的副本写入提供的写入程序。这个
//调用将启动只读事务以执行所有操作。
	Copy(w io.Writer) error

//CLOSE干净地关闭数据库并同步所有数据。
	Close() error
}

//视图打开数据库读取事务，并用
//作为参数传递的事务。F退出后，事务被回滚
//回来。如果F错误，则返回其错误，而不是回滚错误（如果有）
//发生）。
func View(db DB, f func(tx ReadTx) error) error {
	tx, err := db.BeginReadTx()
	if err != nil {
		return err
	}
	err = f(tx)
	rollbackErr := tx.Rollback()
	if err != nil {
		return err
	}
	if rollbackErr != nil {
		return rollbackErr
	}
	return nil
}

//更新打开数据库读/写事务并执行函数f
//将事务作为参数传递。F退出后，如果F没有
//错误，事务已提交。否则，如果F出错，则
//事务回滚。如果回滚失败，则返回原始错误
//F返回的仍然返回。如果提交失败，则提交错误为
//返回。
func Update(db DB, f func(tx ReadWriteTx) error) error {
	tx, err := db.BeginReadWriteTx()
	if err != nil {
		return err
	}
	err = f(tx)
	if err != nil {
//希望返回原始错误，而不是回滚错误，如果
//任何事情发生。
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

//驱动程序定义后端驱动程序在注册时使用的结构
//它们本身是实现数据库接口的后端。
type Driver struct {
//dbtype是用于唯一标识特定
//数据库驱动程序。只能有一个同名的驱动程序。
	DbType string

//create是将在所有用户指定的情况下调用的函数
//用于创建数据库的参数。此函数必须返回
//如果数据库已存在，则errdbexists。
	Create func(args ...interface{}) (DB, error)

//open是将在所有用户指定的情况下调用的函数
//用于打开数据库的参数。此函数必须返回
//如果尚未创建数据库，则返回errdbdoesnotex。
	Open func(args ...interface{}) (DB, error)
}

//DriverList保存所有注册的数据库后端。
var drivers = make(map[string]*Driver)

//RegisterDriver向可用接口添加后端数据库驱动程序。
//如果驱动程序的数据库类型具有
//已经注册。
func RegisterDriver(driver Driver) error {
	if _, exists := drivers[driver.DbType]; exists {
		return ErrDbTypeRegistered
	}

	drivers[driver.DbType] = &driver
	return nil
}

//SupportedDrivers返回表示数据库的字符串切片
//已注册并因此受支持的驱动程序。
func SupportedDrivers() []string {
	supportedDBs := make([]string, 0, len(drivers))
	for _, drv := range drivers {
		supportedDBs = append(supportedDBs, drv.DbType)
	}
	return supportedDBs
}

//创建初始化并打开指定类型的数据库。论点
//特定于数据库类型驱动程序。有关
//有关详细信息，请参阅数据库驱动程序。
//
//如果数据库类型未注册，则返回errdUnknownType。
func Create(dbType string, args ...interface{}) (DB, error) {
	drv, exists := drivers[dbType]
	if !exists {
		return nil, ErrDbUnknownType
	}

	return drv.Create(args...)
}

//打开打开指定类型的现有数据库。这些论点是
//特定于数据库类型驱动程序。请参阅数据库的文档
//有关详细信息，请参阅驱动程序。
//
//如果数据库类型未注册，则返回errdUnknownType。
func Open(dbType string, args ...interface{}) (DB, error) {
	drv, exists := drivers[dbType]
	if !exists {
		return nil, ErrDbUnknownType
	}

	return drv.Open(args...)
}
