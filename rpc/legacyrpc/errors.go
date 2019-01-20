
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package legacyrpc

import (
	"errors"

	"github.com/btcsuite/btcd/btcjson"
)

//TODO（JRICK）：有几个错误路径可以“替换”各种错误
//在btcjson包中出现更合适的错误。创建一张地图
//这些替换，以便在RPC处理程序
//返回并在封送错误之前。

//简化特定类别的报告的错误类型
//错误及其*btcjson.rpcerror创建。
type (
//反序列化错误描述了由于错误而导致的反序列化失败
//用户输入。它对应于btcjson.errrpcd标准化。
	DeserializationError struct {
		error
	}

//InvalidParameterError描述传递的无效参数
//用户。它对应于btcjson.errrpcinvalidParameter。
	InvalidParameterError struct {
		error
	}

//ParseError描述了由于错误的用户输入而导致的分析失败。它
//对应于btcjson.errrpcparse。
	ParseError struct {
		error
	}
)

//在这里定义一次以避免下面的重复的错误变量。
var (
	ErrNeedPositiveAmount = InvalidParameterError{
		errors.New("amount must be positive"),
	}

	ErrNeedPositiveMinconf = InvalidParameterError{
		errors.New("minconf must be positive"),
	}

	ErrAddressNotInWallet = btcjson.RPCError{
		Code:    btcjson.ErrRPCWallet,
		Message: "address not found in wallet",
	}

	ErrAccountNameNotFound = btcjson.RPCError{
		Code:    btcjson.ErrRPCWalletInvalidAccountName,
		Message: "account name not found",
	}

	ErrUnloadedWallet = btcjson.RPCError{
		Code:    btcjson.ErrRPCWallet,
		Message: "Request requires a wallet but wallet has not loaded yet",
	}

	ErrWalletUnlockNeeded = btcjson.RPCError{
		Code:    btcjson.ErrRPCWalletUnlockNeeded,
		Message: "Enter the wallet passphrase with walletpassphrase first",
	}

	ErrNotImportedAccount = btcjson.RPCError{
		Code:    btcjson.ErrRPCWallet,
		Message: "imported addresses must belong to the imported account",
	}

	ErrNoTransactionInfo = btcjson.RPCError{
		Code:    btcjson.ErrRPCNoTxInfo,
		Message: "No information for transaction",
	}

	ErrReservedAccountName = btcjson.RPCError{
		Code:    btcjson.ErrRPCInvalidParameter,
		Message: "Account name is reserved by RPC server",
	}
)
