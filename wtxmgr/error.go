
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2017 BTCSuite开发者
//版权所有（c）2015-2016法令开发商
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wtxmgr

import "fmt"

//错误代码标识错误的类别。
type ErrorCode uint8

//这些常量用于标识特定的错误。
const (
//errdatabase表示基础数据库出错。什么时候？
//设置了此错误代码，错误的err字段将
//设置为从数据库返回的基础错误。
	ErrDatabase ErrorCode = iota

//errdata描述了存储在事务中的数据的错误
//数据库不正确。这可能是由于缺少值、值
//大小错误，或来自不同存储桶的数据与
//本身。从errdata恢复需要重新生成所有
//事务历史或手动数据库手术。如果失败是
//不是由于数据损坏，此错误类别表示
//此包中的编程错误。
	ErrData

//errint描述了一个错误，其中变量传递到
//调用方的函数显然不正确。示例包括
//传递不序列化或试图插入的事务
//索引处不存在交易输出的信贷。
	ErrInput

//erralreadyexists描述了一个错误，创建存储时无法
//继续，因为命名空间中已存在存储区。
	ErrAlreadyExists

//errnoexists描述由于以下原因无法打开存储的错误
//它不在命名空间中。这个错误应该是
//通过创建新商店来处理。
	ErrNoExists

//errNeedsUpgrade描述了在打开存储时的错误，其中
//数据库包含存储的旧版本。
	ErrNeedsUpgrade

//errUnknownVersion描述存储区已存在的错误
//但数据库版本比此已知的最新版本更新
//软件。这可能表示二进制文件过时。
	ErrUnknownVersion
)

var errStrs = [...]string{
	ErrDatabase:       "ErrDatabase",
	ErrData:           "ErrData",
	ErrInput:          "ErrInput",
	ErrAlreadyExists:  "ErrAlreadyExists",
	ErrNoExists:       "ErrNoExists",
	ErrNeedsUpgrade:   "ErrNeedsUpgrade",
	ErrUnknownVersion: "ErrUnknownVersion",
}

//字符串将错误代码返回为人类可读的名称。
func (e ErrorCode) String() string {
	if e < ErrorCode(len(errStrs)) {
		return errStrs[e]
	}
	return fmt.Sprintf("ErrorCode(%d)", e)
}

//错误为存储期间可能发生的错误提供单一类型
//操作。
type Error struct {
Code ErrorCode //描述错误的类型
Desc string    //问题的人类可读描述
Err  error     //基本错误，可选
}

//错误满足错误接口并打印人类可读的错误。
func (e Error) Error() string {
	if e.Err != nil {
		return e.Desc + ": " + e.Err.Error()
	}
	return e.Desc
}

func storeError(c ErrorCode, desc string, err error) Error {
	return Error{Code: c, Desc: desc, Err: err}
}

//IsNoexists返回错误是否为带有errnoexists错误的错误
//代码。
func IsNoExists(err error) bool {
	serr, ok := err.(Error)
	return ok && serr.Code == ErrNoExists
}
