
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

//包txauthor为钱包提供事务创建代码。
package txauthor

import (
	"errors"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/wallet/txrules"

	h "github.com/btcsuite/btcwallet/internal/helpers"
	"github.com/btcsuite/btcwallet/wallet/internal/txsizes"
)

//inputsource提供引用可消费输出的事务输入
//构造一个输出某个目标金额的事务。如果目标金额
//
//或者返回更详细的错误实现
//输入源错误。
type InputSource func(target btcutil.Amount) (total btcutil.Amount, inputs []*wire.TxIn,
	inputValues []btcutil.Amount, scripts [][]byte, err error)

//inputsourceError描述了未能从中提供足够的输入值
//未占用的事务输出以满足目标量。使用了类型错误
//因此，输入源可以提供自己的描述原因的实现
//例如，由于消费政策或锁定的硬币而导致的错误
//比钱包没有足够的可用输入值。
type InputSourceError interface {
	error
	InputSourceError()
}

//inputsourceerror的默认实现。
type insufficientFundsError struct{}

func (insufficientFundsError) InputSourceError() {}
func (insufficientFundsError) Error() string {
	return "insufficient funds available to construct transaction"
}

//
//输出（如果添加了一个）。
type AuthoredTx struct {
	Tx              *wire.MsgTx
	PrevScripts     [][]byte
	PrevInputValues []btcutil.Amount
	TotalInput      btcutil.Amount
ChangeIndex     int //无变化为负
}

//ChangeSource为事务创建提供p2pkh更改输出脚本。
type ChangeSource func() ([]byte, error)

//NewUnsignedTransaction创建向一个或多个付款的未签名事务
//非变更输出。根据
//交易规模。
//
//事务输入从对fetchinputs的重复调用中选择
//增加目标数量。
//
//如果任何剩余的输出值可以通过更改返回钱包
//输出不违反MEMPOOL灰尘规则，p2wpkh更改输出为
//附加到事务输出。因为更改输出可能不是
//必要时，fetchChange被调用零次或一次以生成此脚本。
//此函数必须返回p2wpkh脚本或更小的脚本，否则将进行费用估算。
//将不正确。
//
//如果成功，则为事务、花费的总输入值和所有以前的值
//返回输出脚本。如果输入源无法提供
//足够的投入值来支付每一项产出的任何必要费用，以及
//返回inputsourceError。
//
//错误：当补偿非压缩p2pkh输出时，费用估计可能会关闭。
func NewUnsignedTransaction(outputs []*wire.TxOut, relayFeePerKb btcutil.Amount,
	fetchInputs InputSource, fetchChange ChangeSource) (*AuthoredTx, error) {

	targetAmount := h.SumOutputValues(outputs)
	estimatedSize := txsizes.EstimateVirtualSize(0, 1, 0, outputs, true)
	targetFee := txrules.FeeForSerializeSize(relayFeePerKb, estimatedSize)

	for {
		inputAmount, inputs, inputValues, scripts, err := fetchInputs(targetAmount + targetFee)
		if err != nil {
			return nil, err
		}
		if inputAmount < targetAmount+targetFee {
			return nil, insufficientFundsError{}
		}

//我们计算输入的类型，我们将使用这些类型来估计
//事务的vsize。
		var nested, p2wpkh, p2pkh int
		for _, pkScript := range scripts {
			switch {
//如果这是一个p2sh输出，我们假设这是一个
//嵌套的2WKH。
			case txscript.IsPayToScriptHash(pkScript):
				nested++
			case txscript.IsPayToWitnessPubKeyHash(pkScript):
				p2wpkh++
			default:
				p2pkh++
			}
		}

		maxSignedSize := txsizes.EstimateVirtualSize(p2pkh, p2wpkh,
			nested, outputs, true)
		maxRequiredFee := txrules.FeeForSerializeSize(relayFeePerKb, maxSignedSize)
		remainingAmount := inputAmount - targetAmount
		if remainingAmount < maxRequiredFee {
			targetFee = maxRequiredFee
			continue
		}

		unsignedTransaction := &wire.MsgTx{
			Version:  wire.TxVersion,
			TxIn:     inputs,
			TxOut:    outputs,
			LockTime: 0,
		}
		changeIndex := -1
		changeAmount := inputAmount - targetAmount - maxRequiredFee
		if changeAmount != 0 && !txrules.IsDustAmount(changeAmount,
			txsizes.P2WPKHPkScriptSize, relayFeePerKb) {
			changeScript, err := fetchChange()
			if err != nil {
				return nil, err
			}
			if len(changeScript) > txsizes.P2WPKHPkScriptSize {
				return nil, errors.New("fee estimation requires change " +
					"scripts no larger than P2WPKH output scripts")
			}
			change := wire.NewTxOut(int64(changeAmount), changeScript)
			l := len(outputs)
			unsignedTransaction.TxOut = append(outputs[:l:l], change)
			changeIndex = l
		}

		return &AuthoredTx{
			Tx:              unsignedTransaction,
			PrevScripts:     scripts,
			PrevInputValues: inputValues,
			TotalInput:      inputAmount,
			ChangeIndex:     changeIndex,
		}, nil
	}
}

//随机化输出位置随机化事务输出的位置
//将其与随机输出交换。将返回新索引。这应该是
//签名前完成。
func RandomizeOutputPosition(outputs []*wire.TxOut, index int) int {
	r := cprng.Int31n(int32(len(outputs)))
	outputs[r], outputs[index] = outputs[index], outputs[r]
	return int(r)
}

//随机化更改位置随机化已编写事务的位置
//更改输出。这应该在签署之前完成。
func (tx *AuthoredTx) RandomizeChangePosition() {
	tx.ChangeIndex = RandomizeOutputPosition(tx.Tx.TxOut, tx.ChangeIndex)
}

//
//正在构造事务输入签名。秘密是由
//上一个输出脚本的对应地址。查找地址
//使用源代码的区块链参数创建，表示
//SecretsSource只能管理单个链的机密。
//
//TODO:重写此接口以查找私钥并兑现脚本
//pubkeys、pubkey散列、脚本散列等作为单独的接口方法。
//这将删除接口的chainParams要求，并且可以
//避免从以前的输出脚本到地址的不必要转换。
//如果不修改txscript包，则无法执行此操作。
type SecretsSource interface {
	txscript.KeyDB
	txscript.ScriptDB
	ChainParams() *chaincfg.Params
}

//addallinputscripts通过添加输入修改事务
//每个输入的脚本。以前的输出脚本被每个输入兑现
//在prevpkscripts中传递，并且切片长度必须与
//输入。使用SecretsSource查找私钥和兑现脚本
//基于上一个输出脚本。
func AddAllInputScripts(tx *wire.MsgTx, prevPkScripts [][]byte, inputValues []btcutil.Amount,
	secrets SecretsSource) error {

	inputs := tx.TxIn
	hashCache := txscript.NewTxSigHashes(tx)
	chainParams := secrets.ChainParams()

	if len(inputs) != len(prevPkScripts) {
		return errors.New("tx.TxIn and prevPkScripts slices must " +
			"have equal length")
	}

	for i := range inputs {
		pkScript := prevPkScripts[i]

		switch {
//如果这是一个p2sh输出，那么谁的脚本哈希预映像是
//见证程序，然后我们需要使用修改后的签名
//同时生成sigscript和见证的函数
//脚本。
		case txscript.IsPayToScriptHash(pkScript):
			err := spendNestedWitnessPubKeyHash(inputs[i], pkScript,
				int64(inputValues[i]), chainParams, secrets,
				tx, hashCache, i)
			if err != nil {
				return err
			}
		case txscript.IsPayToWitnessPubKeyHash(pkScript):
			err := spendWitnessKeyHash(inputs[i], pkScript,
				int64(inputValues[i]), chainParams, secrets,
				tx, hashCache, i)
			if err != nil {
				return err
			}
		default:
			sigScript := inputs[i].SignatureScript
			script, err := txscript.SignTxOutput(chainParams, tx, i,
				pkScript, txscript.SigHashAll, secrets, secrets,
				sigScript)
			if err != nil {
				return err
			}
			inputs[i].SignatureScript = script
		}
	}

	return nil
}

//spendwitnesskeyhash生成并为花费
//传递了具有指定输入量的pkscript。输入金额*必须*
//对应于上一个pkscript的输出值，否则验证
//由于bip0143中定义的新sighash摘要算法包括
//the input value in the sighash.
func spendWitnessKeyHash(txIn *wire.TxIn, pkScript []byte,
	inputValue int64, chainParams *chaincfg.Params, secrets SecretsSource,
	tx *wire.MsgTx, hashCache *txscript.TxSigHashes, idx int) error {

//首先获取与此p2wkh地址相关联的密钥对。
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(pkScript,
		chainParams)
	if err != nil {
		return err
	}
	privKey, compressed, err := secrets.GetKey(addrs[0])
	if err != nil {
		return err
	}
	pubKey := privKey.PubKey()

//一旦我们有了密钥对，就生成p2wkh地址类型，尊重
//生成的密钥的压缩类型。
	var pubKeyHash []byte
	if compressed {
		pubKeyHash = btcutil.Hash160(pubKey.SerializeCompressed())
	} else {
		pubKeyHash = btcutil.Hash160(pubKey.SerializeUncompressed())
	}
	p2wkhAddr, err := btcutil.NewAddressWitnessPubKeyHash(pubKeyHash, chainParams)
	if err != nil {
		return err
	}

//通过具体的地址类型，我们现在可以生成
//用于生成有效见证的相应见证程序
//这将允许我们花费这个产出。
	witnessProgram, err := txscript.PayToAddrScript(p2wkhAddr)
	if err != nil {
		return err
	}
	witnessScript, err := txscript.WitnessSignature(tx, hashCache, idx,
		inputValue, witnessProgram, txscript.SigHashAll, privKey, true)
	if err != nil {
		return err
	}

	txIn.Witness = witnessScript

	return nil
}

//spendnestedWitnessPubkey为生成sigscript和有效见证
//使用指定的输入量花费传递的pkscript。生成的
//sigscript是与查询的
//关键。见证堆栈与使用常规
//输出为2WKH。输入量*必须*对应于
//previous pkScript, or else verification will fail since the new sighash
//bip0143中定义的摘要算法包括sighash中的输入值。
func spendNestedWitnessPubKeyHash(txIn *wire.TxIn, pkScript []byte,
	inputValue int64, chainParams *chaincfg.Params, secrets SecretsSource,
	tx *wire.MsgTx, hashCache *txscript.TxSigHashes, idx int) error {

//首先，我们需要获得与这个p2sh输出相关的密钥对。
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(pkScript,
		chainParams)
	if err != nil {
		return err
	}
	privKey, compressed, err := secrets.GetKey(addrs[0])
	if err != nil {
		return err
	}
	pubKey := privKey.PubKey()

	var pubKeyHash []byte
	if compressed {
		pubKeyHash = btcutil.Hash160(pubKey.SerializeCompressed())
	} else {
		pubKeyHash = btcutil.Hash160(pubKey.SerializeUncompressed())
	}

//接下来，我们将生成一个有效的sigscript，它将允许我们
//p2sh输出。sigscript将只包含
//与匹配的公钥对应的p2wkh见证程序
//地址。
	p2wkhAddr, err := btcutil.NewAddressWitnessPubKeyHash(pubKeyHash, chainParams)
	if err != nil {
		return err
	}
	witnessProgram, err := txscript.PayToAddrScript(p2wkhAddr)
	if err != nil {
		return err
	}
	bldr := txscript.NewScriptBuilder()
	bldr.AddData(witnessProgram)
	sigScript, err := bldr.Script()
	if err != nil {
		return err
	}
	txIn.SignatureScript = sigScript

//在sigscript就位后，我们接下来将生成适当的见证人
//that'll allow us to spend the p2wkh output.
	witnessScript, err := txscript.WitnessSignature(tx, hashCache, idx,
		inputValue, witnessProgram, txscript.SigHashAll, privKey, compressed)
	if err != nil {
		return err
	}

	txIn.Witness = witnessScript

	return nil
}

//addallinputscripts通过添加输入脚本来修改编写的事务
//对于已编写事务的每个输入。私钥和兑换脚本
//are looked up using a SecretsSource based on the previous output script.
func (tx *AuthoredTx) AddAllInputScripts(secrets SecretsSource) error {
	return AddAllInputScripts(tx.Tx, tx.PrevScripts, tx.PrevInputValues, secrets)
}
