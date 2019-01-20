
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

package walletdb

import (
	"errors"
)

//驱动程序注册过程中可能发生的错误。
var (
//当两个不同的数据库驱动程序时返回errdbtyperegistered
//尝试用名称数据库类型注册。
	ErrDbTypeRegistered = errors.New("database type already registered")
)

//各种数据库函数可能返回的错误。
var (
//没有为注册的驱动程序时返回errdUnknownType
//指定的数据库类型。
	ErrDbUnknownType = errors.New("unknown database type")

//当为一个
//不存在。
	ErrDbDoesNotExist = errors.New("database does not exist")

//当为数据库调用create时，返回errdbexists
//已经存在。
	ErrDbExists = errors.New("database already exists")

//当以前访问过数据库实例时返回errdnotopen
//打开或关闭后。
	ErrDbNotOpen = errors.New("database not open")

//当对数据库调用open时，返回errdbalreadyopen
//已经打开。
	ErrDbAlreadyOpen = errors.New("database already open")

//如果指定的数据库无效，则返回errInvalid。
	ErrInvalid = errors.New("invalid database")
)

//开始或提交事务时可能发生的错误。
var (
//尝试提交或回滚时返回errtxclosed
//已执行其中一个操作的事务。
	ErrTxClosed = errors.New("tx closed")

//当需要写操作时返回errtxnotwritable
//试图对只读事务访问数据库。
	ErrTxNotWritable = errors.New("tx not writable")
)

//在放入或删除值或存储桶时可能发生的错误。
var (
//尝试访问具有
//尚未创建。
	ErrBucketNotFound = errors.New("bucket not found")

//创建已存在的bucket时返回errbacketexists。
	ErrBucketExists = errors.New("bucket already exists")

//创建名称为空的bucket时返回errbacketnameRequired。
	ErrBucketNameRequired = errors.New("bucket name required")

//插入零长度密钥时返回errkeyRequired。
	ErrKeyRequired = errors.New("key required")

//插入大于maxkeysize的键时返回errkeytoolarge。
	ErrKeyTooLarge = errors.New("key too large")

//插入大于maxValueSize的值时返回errValueToArge。
	ErrValueTooLarge = errors.New("value too large")

//当尝试创建或删除
//在现有的非bucket键上或尝试创建或
//删除现有bucket键上的非bucket键。
	ErrIncompatibleValue = errors.New("incompatible value")
)
