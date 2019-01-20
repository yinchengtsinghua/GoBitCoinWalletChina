
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

//Package txrules provides transaction rules that should be followed by
//事务作者，用于广泛的mempool接受和快速挖掘。
package txrules

import (
	"errors"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//DefaultRelayFeeperkB是mempool的默认最低中继费用策略。
const DefaultRelayFeePerKb btcutil.Amount = 1e3

//GetDustThreshold用于定义输出低于的数量
//确定为灰尘。阈值确定为中继费的3倍。
func GetDustThreshold(scriptSize int, relayFeePerKb btcutil.Amount) btcutil.Amount {
//计算网络的总（估计）成本。这是
//使用输出的序列化大小加上串行
//赎回它的事务输入的大小。假设输出
//要压缩p2pkh，因为这是最常见的脚本类型。使用
//
//
	totalSize := 8 + wire.VarIntSerializeSize(uint64(scriptSize)) +
		scriptSize + 148

	byteFee := relayFeePerKb / 1000
	relayFee := btcutil.Amount(totalSize) * byteFee
	return 3 * relayFee
}

//IsDustAmount确定事务输出值和脚本长度是否
//使输出被视为灰尘。有粉尘输出的交易是
//不标准，并被具有默认策略的内存池拒绝。
func IsDustAmount(amount btcutil.Amount, scriptSize int, relayFeePerKb btcutil.Amount) bool {
	return amount < GetDustThreshold(scriptSize, relayFeePerKb)
}

//IsDustOutput determines whether a transaction output is considered dust.
//Transactions with dust outputs are not standard and are rejected by mempools
//使用默认策略。
func IsDustOutput(output *wire.TxOut, relayFeePerKb btcutil.Amount) bool {
//单独携带数据的非独立输出不检查灰尘。
	if txscript.GetScriptClass(output.PkScript) == txscript.NullDataTy {
		return false
	}

//所有其他不可依赖的输出都被视为灰尘。
	if txscript.IsUnspendable(output.PkScript) {
		return true
	}

	return IsDustAmount(btcutil.Amount(output.Value), len(output.PkScript),
		relayFeePerKb)
}

//违反交易规则
var (
	ErrAmountNegative   = errors.New("transaction output amount is negative")
	ErrAmountExceedsMax = errors.New("transaction output amount exceeds maximum value")
	ErrOutputIsDust     = errors.New("transaction output is dust")
)

//CheckOutput performs simple consensus and policy tests on a transaction
//输出。
func CheckOutput(output *wire.TxOut, relayFeePerKb btcutil.Amount) error {
	if output.Value < 0 {
		return ErrAmountNegative
	}
	if output.Value > btcutil.MaxSatoshi {
		return ErrAmountExceedsMax
	}
	if IsDustOutput(output, relayFeePerKb) {
		return ErrOutputIsDust
	}
	return nil
}

//
//
func FeeForSerializeSize(relayFeePerKb btcutil.Amount, txSerializeSize int) btcutil.Amount {
	fee := relayFeePerKb * btcutil.Amount(txSerializeSize) / 1000

	if fee == 0 && relayFeePerKb > 0 {
		fee = relayFeePerKb
	}

	if fee < 0 || fee > btcutil.MaxSatoshi {
		fee = btcutil.MaxSatoshi
	}

	return fee
}
