
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

package waddrmgr

import (
	"fmt"
	"strconv"

	"github.com/btcsuite/btcutil/hdkeychain"
)

var (
//erralreadyexists是用于
//erralreadyexists错误代码。
	errAlreadyExists = "the specified address manager already exists"

//errcointyptoohigh是用于
//错误类型为高错误代码。
	errCoinTypeTooHigh = "coin type may not exceed " +
		strconv.FormatUint(hdkeychain.HardenedKeyStart-1, 10)

//
//错误代码。
	errAcctTooHigh = "account number may not exceed " +
		strconv.FormatUint(hdkeychain.HardenedKeyStart-1, 10)

//
//
	errLocked = "address manager is locked"

//errWatchingOnly是用于
//
	errWatchingOnly = "address manager is watching-only"
)

//错误代码标识一种错误。
type ErrorCode int

//
const (
//
//
//设置为从数据库返回的基础错误。
	ErrDatabase ErrorCode = iota

//errUpgrade表示需要升级管理器。这应该
//
//
	ErrUpgrade

//errKeyChain表示钥匙链出现错误，通常是
//由于无法创建扩展密钥或派生子项
//
//
	ErrKeyChain

//errCrypto表示与加密相关的操作出错。
//例如解密或加密数据，解析EC公钥，
//或者从密码中派生密钥。当此错误代码为
//
//错误。
	ErrCrypto

//
//
	ErrInvalidKeyType

//
	ErrNoExist

//
	ErrAlreadyExists

//
//
//
	ErrCoinTypeTooHigh

//
//大于MaxAccountNum常量定义的最大允许值。
	ErrAccountNumTooHigh

//
//
	ErrLocked

//
//
//
	ErrWatchingOnly

//
	ErrInvalidAccount

//errAddressNotFound表示请求的地址未知
//
	ErrAddressNotFound

//
//
	ErrAccountNotFound

//
	ErrDuplicateAddress

//
	ErrDuplicateAccount

//
//
	ErrTooManyAddresses

//errWrongPassphrase表示指定的密码不正确。
//这可以用于公钥或私钥。
	ErrWrongPassphrase

//
//
	ErrWrongNet

//
//
	ErrCallBackBreak

//
//因为是空的。
	ErrEmptyPassphrase

//找不到目标作用域时返回errscopenotfound
//在数据库中。
	ErrScopeNotFound

//当我们尝试检索
//
	ErrBirthdayBlockNotSet

//尝试检索的哈希时返回errblocknotfound
//我们不知道的街区。
	ErrBlockNotFound
)

//将错误代码值映射回其常量名，以便进行漂亮的打印。
var errorCodeStrings = map[ErrorCode]string{
	ErrDatabase:          "ErrDatabase",
	ErrUpgrade:           "ErrUpgrade",
	ErrKeyChain:          "ErrKeyChain",
	ErrCrypto:            "ErrCrypto",
	ErrInvalidKeyType:    "ErrInvalidKeyType",
	ErrNoExist:           "ErrNoExist",
	ErrAlreadyExists:     "ErrAlreadyExists",
	ErrCoinTypeTooHigh:   "ErrCoinTypeTooHigh",
	ErrAccountNumTooHigh: "ErrAccountNumTooHigh",
	ErrLocked:            "ErrLocked",
	ErrWatchingOnly:      "ErrWatchingOnly",
	ErrInvalidAccount:    "ErrInvalidAccount",
	ErrAddressNotFound:   "ErrAddressNotFound",
	ErrAccountNotFound:   "ErrAccountNotFound",
	ErrDuplicateAddress:  "ErrDuplicateAddress",
	ErrDuplicateAccount:  "ErrDuplicateAccount",
	ErrTooManyAddresses:  "ErrTooManyAddresses",
	ErrWrongPassphrase:   "ErrWrongPassphrase",
	ErrWrongNet:          "ErrWrongNet",
	ErrCallBackBreak:     "ErrCallBackBreak",
	ErrEmptyPassphrase:   "ErrEmptyPassphrase",
	ErrScopeNotFound:     "ErrScopeNotFound",
}

//字符串将错误代码返回为人类可读的名称。
func (e ErrorCode) String() string {
	if s := errorCodeStrings[e]; s != "" {
		return s
	}
	return fmt.Sprintf("Unknown ErrorCode (%d)", int(e))
}

//
//
//
//锁定地址管理器的私钥，数据库错误
//（errDatabase）、密钥链派生错误（errKeyChain）和错误
//与加密（errCrypto）相关。
//
//
//
//失败。
//
//errDatabase、errKeyChain和errCrypto错误代码也将具有
//带基础错误的错误字段集。
type ManagerError struct {
ErrorCode   ErrorCode //描述错误的类型
Description string    //问题的人类可读描述
Err         error     //潜在错误
}

//错误满足错误接口并打印人类可读的错误。
func (e ManagerError) Error() string {
	if e.Err != nil {
		return e.Description + ": " + e.Err.Error()
	}
	return e.Description
}

//
func managerError(c ErrorCode, desc string, err error) ManagerError {
	return ManagerError{ErrorCode: c, Description: desc, Err: err}
}

//
//通过返回代码为errcallbackbreak的错误来执行函数
var Break = managerError(ErrCallBackBreak, "callback break", nil)

//ISERR返回该错误是否为带有匹配错误的ManagerError
//代码。
func IsError(err error, code ErrorCode) bool {
	e, ok := err.(ManagerError)
	return ok && e.ErrorCode == code
}
