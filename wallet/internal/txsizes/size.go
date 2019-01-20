
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

package txsizes

import (
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/wire"

	h "github.com/btcsuite/btcwallet/internal/helpers"
)

//
const (
//redemp2pkhsigscriptsize是最坏情况（最大）的序列化大小
//一个事务输入脚本，它重新调用一个压缩的p2pkh输出。
//计算如下：
//
//- OpDATA
//-72字节的der签名+1字节的叹息
//- OpthDATA33
//-33字节序列化压缩pubkey
	RedeemP2PKHSigScriptSize = 1 + 73 + 1 + 33

//p2pkhpkscriptsize是事务输出脚本的大小，该脚本
//支付到压缩的pubkey散列。计算如下：
//
//-奥普杜普
//- OpHHAS160
//- OPYDATAY20
//-20字节pubkey哈希
//
//-奥普克西格
	P2PKHPkScriptSize = 1 + 1 + 1 + 20 + 1 + 1

//redemp2pkhinputsize是
//交易输入，赎回压缩的p2pkh输出。它是
//计算如下：
//
//-32字节以前的Tx
//-4字节输出索引
//-1字节压缩int编码值107
//-107字节签名脚本
//-4字节序列
	RedeemP2PKHInputSize = 32 + 4 + 1 + RedeemP2PKHSigScriptSize + 4

//p2pkhoutputsize是具有
//p2pkh输出脚本。计算如下：
//
//-8字节输出值
//-1字节压缩int编码值25
//-25字节p2pkh输出脚本
	P2PKHOutputSize = 8 + 1 + P2PKHPkScriptSize

//p2wpkhpkscriptsize是事务输出脚本的大小，该脚本
//支付给证人Pubkey哈希。计算如下：
//
//- op00
//- OPYDATAY20
//-20字节pubkey哈希
	P2WPKHPkScriptSize = 1 + 1 + 20

//p2wpkhoutputsize是具有
//p2wpkh输出脚本。计算如下：
//
//-8字节输出值
//-1字节压缩int编码值22
//-22字节p2pkh输出脚本
	P2WPKHOutputSize = 8 + 1 + P2WPKHPkScriptSize

//redemp2wpkhscriptsize是事务输入脚本的大小
//这需要支付见证公钥哈希（p2wpkh）的费用。赎回
//p2wpkh花销的脚本必须为空。
	RedeemP2WPKHScriptSize = 0

//redemp2wpkhinputsize是事务的最坏情况大小
//输入补偿p2wpkh输出。计算如下：
//
//-32字节以前的Tx
//-4字节输出索引
//-1字节编码空的兑换脚本
//-0字节兑换脚本
//-4字节序列
	RedeemP2WPKHInputSize = 32 + 4 + 1 + RedeemP2WPKHScriptSize + 4

//redeemnestedp2wpkhscriptsize是事务的最坏情况大小
//输入脚本，用于重新获取嵌套在p2sh中的付款见证密钥哈希
//（2SH -2WPKPH）。计算如下：
//
//-1字节压缩int编码值22
//- op00
//-1字节压缩int编码值20
//-20字节密钥哈希
	RedeemNestedP2WPKHScriptSize = 1 + 1 + 1 + 20

//
//交易输入赎回p2sh-p2wpkh输出。它是
//计算如下：
//
//-32字节以前的Tx
//-4字节输出索引
//-1字节压缩int编码值23
//-23字节兑换脚本（scriptsig）
//-4字节序列
	RedeemNestedP2WPKHInputSize = 32 + 4 + 1 +
		RedeemNestedP2WPKHScriptSize + 4

//
//使用p2wpkh和嵌套p2wpkh输出的见证。它
//计算如下：
//
//-1 wu compact int编码值2（项数）
//-1 wu compact int编码值73
//-72吴德签名+1吴叹息
//-1 wu compact int编码值33
//-33 wu序列化压缩pubkey
	RedeemP2WPKHInputWitnessWeight = 1 + 1 + 73 + 1 + 33
)

//EstimaterializeSize返回
//花费inputcount的签名事务压缩p2pkh输出的数目
//并包含来自txout的每个事务输出。估计的大小是
//如果addChangeOutput为true，则对额外的p2pkh更改输出递增。
func EstimateSerializeSize(inputCount int, txOuts []*wire.TxOut, addChangeOutput bool) int {
	changeSize := 0
	outputCount := len(txOuts)
	if addChangeOutput {
		changeSize = P2PKHOutputSize
		outputCount++
	}

//8个附加字节用于版本和锁定时间
	return 8 + wire.VarIntSerializeSize(uint64(inputCount)) +
		wire.VarIntSerializeSize(uint64(outputCount)) +
		inputCount*RedeemP2PKHInputSize +
		h.SumOutputSerializeSizes(txOuts) +
		changeSize
}

//EstimateVirtualSize返回
//花费给定数量的p2pkh、p2wpkh和
//
//来自TXOUT。对于额外的p2pkh，估计值递增。
//如果addChangeOutput为true，则更改输出。
func EstimateVirtualSize(numP2PKHIns, numP2WPKHIns, numNestedP2WPKHIns int,
	txOuts []*wire.TxOut, addChangeOutput bool) int {
	changeSize := 0
	outputCount := len(txOuts)
	if addChangeOutput {
//我们总是使用p2wpkh作为变更输出。
		changeSize = P2WPKHOutputSize
		outputCount++
	}

//
//事务输入和输出数量+兑换脚本大小+
//
	baseSize := 8 +
		wire.VarIntSerializeSize(
			uint64(numP2PKHIns+numP2WPKHIns+numNestedP2WPKHIns)) +
		wire.VarIntSerializeSize(uint64(len(txOuts))) +
		numP2PKHIns*RedeemP2PKHInputSize +
		numP2WPKHIns*RedeemP2WPKHInputSize +
		numNestedP2WPKHIns*RedeemNestedP2WPKHInputSize +
		h.SumOutputSerializeSizes(txOuts) +
		changeSize

//如果此事务有任何见证输入，我们必须计算
//见证数据。
	witnessWeight := 0
	if numP2WPKHIns+numNestedP2WPKHIns > 0 {
//Segwit标记+标记的额外2个重量单位。
		witnessWeight = 2 +
			wire.VarIntSerializeSize(
				uint64(numP2WPKHIns+numNestedP2WPKHIns)) +
			numP2WPKHIns*RedeemP2WPKHInputWitnessWeight +
			numNestedP2WPKHIns*RedeemP2WPKHInputWitnessWeight
	}

//我们在证人体重上加3，以确保结果是
//总是向上取整。
	return baseSize + (witnessWeight+3)/blockchain.WitnessScaleFactor
}
