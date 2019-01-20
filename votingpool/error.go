
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

package votingpool

import "fmt"

//错误代码标识一种错误
type ErrorCode int

const (
//
//
	ErrInputSelection ErrorCode = iota

//errDrawAlProcessing表示在处理
//提款请求。
	ErrWithdrawalProcessing

//errUnknownSubkey表示不属于给定的
//系列。
	ErrUnknownPubKey

//errserieserialization表示在
//序列化或反序列化一个或多个序列以存储到
//数据库。
	ErrSeriesSerialization

//errseriesversion表示我们被要求处理一个系列
//不支持其版本
	ErrSeriesVersion

//errseriesnotexists表示已尝试访问
//不存在的序列。
	ErrSeriesNotExists

//errseriesalreadyexists表示已尝试
//创建已存在的序列。
	ErrSeriesAlreadyExists

//errseriesalreadyempowered表示已经授权的系列
//在预期未授权的情况下使用。
	ErrSeriesAlreadyEmpowered

//errseriesnotactive表示需要活动序列，但
//
	ErrSeriesNotActive

//errkeyisprivate表示在公用密钥
//应该有一个。
	ErrKeyIsPrivate

//errkeyispublic表示使用公钥的位置
//应该有一个。
	ErrKeyIsPublic

//errkineuter表示在尝试断开私钥时出现问题。
	ErrKeyNeuter

//errKeyMismatch表示该键不是预期的键。
	ErrKeyMismatch

//errKeyPrivatePublicMismatch表示private和
//公钥不同。
	ErrKeysPrivatePublicMismatch

//errKeyDuplicate表示密钥重复。
	ErrKeyDuplicate

//errtoOfewPublicKeys指示所需的最小公共
//未满足密钥。
	ErrTooFewPublicKeys

//errpoolReadyExists表示已尝试
//创建已存在的投票池。
	ErrPoolAlreadyExists

//errpoolnotexists表示已尝试访问
//不存在的投票池。
	ErrPoolNotExists

//errscriptCreation指示创建存款脚本
//失败。
	ErrScriptCreation

//ErrToomanyReqSignatures表示需要太多
//请求签名。
	ErrTooManyReqSignatures

//errInvalidBranch指示给定的分支编号无效
//对于给定的一组公钥。
	ErrInvalidBranch

//errInvalidValue指示给定函数参数的值
//无效。
	ErrInvalidValue

//errdatabase表示基础数据库出错。
	ErrDatabase

//errKeyChain表示钥匙链出现错误，通常是
//由于无法创建扩展密钥或派生子项
//扩展密钥。
	ErrKeyChain

//errCrypto表示与加密相关的操作出错。
//例如解密或加密数据，解析EC公钥，
//或者从密码中派生密钥。
	ErrCrypto

//errrawsigning表示在生成原始数据的过程中出错。
//事务输入的签名。
	ErrRawSigning

//errPreconditionNotMet表示自
//未满足预条件。
	ErrPreconditionNotMet

//errtxsigning表示在对事务进行签名时出错。
	ErrTxSigning

//errseriesidnotsequential表示试图创建具有
//不是亮片的身份证。
	ErrSeriesIDNotSequential

//errInvalidScriptHash指示无效的p2sh。
	ErrInvalidScriptHash

//errDrawFromUnusedADdr表示试图从
//以前没有使用过的地址。
	ErrWithdrawFromUnusedAddr

//errseriesidinvalid表示试图用
//无效ID。
	ErrSeriesIDInvalid

//errRetractortXstorage表示存储提取时出错
//交易。
	ErrWithdrawalTxStorage

//errDrawAltorage表示序列化或
//正在反序列化撤消信息。
	ErrWithdrawalStorage

//
//错误代码以检查它们是否都正确
//错误代码字符串中的翻译。
	lastErr
)

//将错误代码值映射回其常量名，以便进行漂亮的打印。
var errorCodeStrings = map[ErrorCode]string{
	ErrInputSelection:            "ErrInputSelection",
	ErrWithdrawalProcessing:      "ErrWithdrawalProcessing",
	ErrUnknownPubKey:             "ErrUnknownPubKey",
	ErrSeriesSerialization:       "ErrSeriesSerialization",
	ErrSeriesVersion:             "ErrSeriesVersion",
	ErrSeriesNotExists:           "ErrSeriesNotExists",
	ErrSeriesAlreadyExists:       "ErrSeriesAlreadyExists",
	ErrSeriesAlreadyEmpowered:    "ErrSeriesAlreadyEmpowered",
	ErrSeriesIDNotSequential:     "ErrSeriesIDNotSequential",
	ErrSeriesIDInvalid:           "ErrSeriesIDInvalid",
	ErrSeriesNotActive:           "ErrSeriesNotActive",
	ErrKeyIsPrivate:              "ErrKeyIsPrivate",
	ErrKeyIsPublic:               "ErrKeyIsPublic",
	ErrKeyNeuter:                 "ErrKeyNeuter",
	ErrKeyMismatch:               "ErrKeyMismatch",
	ErrKeysPrivatePublicMismatch: "ErrKeysPrivatePublicMismatch",
	ErrKeyDuplicate:              "ErrKeyDuplicate",
	ErrTooFewPublicKeys:          "ErrTooFewPublicKeys",
	ErrPoolAlreadyExists:         "ErrPoolAlreadyExists",
	ErrPoolNotExists:             "ErrPoolNotExists",
	ErrScriptCreation:            "ErrScriptCreation",
	ErrTooManyReqSignatures:      "ErrTooManyReqSignatures",
	ErrInvalidBranch:             "ErrInvalidBranch",
	ErrInvalidValue:              "ErrInvalidValue",
	ErrDatabase:                  "ErrDatabase",
	ErrKeyChain:                  "ErrKeyChain",
	ErrCrypto:                    "ErrCrypto",
	ErrRawSigning:                "ErrRawSigning",
	ErrPreconditionNotMet:        "ErrPreconditionNotMet",
	ErrTxSigning:                 "ErrTxSigning",
	ErrInvalidScriptHash:         "ErrInvalidScriptHash",
	ErrWithdrawFromUnusedAddr:    "ErrWithdrawFromUnusedAddr",
	ErrWithdrawalTxStorage:       "ErrWithdrawalTxStorage",
	ErrWithdrawalStorage:         "ErrWithdrawalStorage",
}

//字符串将错误代码返回为人类可读的名称。
func (e ErrorCode) String() string {
	if s := errorCodeStrings[e]; s != "" {
		return s
	}
	return fmt.Sprintf("Unknown ErrorCode (%d)", int(e))
}

//错误是在
//投票池的操作。
type Error struct {
ErrorCode   ErrorCode //描述错误的类型
Description string    //问题的人类可读描述
Err         error     //潜在错误
}

//错误满足错误接口并打印人类可读的错误。
func (e Error) Error() string {
	if e.Err != nil {
		return e.Description + ": " + e.Err.Error()
	}
	return e.Description
}

//newError创建新错误。
func newError(c ErrorCode, desc string, err error) Error {
	return Error{ErrorCode: c, Description: desc, Err: err}
}
