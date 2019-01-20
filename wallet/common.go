
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016版权所有
//版权所有（c）2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wallet

import (
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

//注意：以下常见类型不应引用钱包类型。
//长期目标是将它们移动到自己的包中，以便数据库
//访问API可以直接创建它们，以便钱包返回。

//blockIdentity标识一个块或缺少一个块（用于描述
//
type BlockIdentity struct {
	Hash   chainhash.Hash
	Height int32
}

//none返回实例是否没有描述的块。什么时候？
//与一个事务关联，这表示该事务是未链接的。
func (b *BlockIdentity) None() bool {
//错误：因为dcrwallet在不同的地方使用0和-1来引用
//对于非关联交易，这必须与两者都核对，并且不能
//可用于代表Genesis区块。
	return *b == BlockIdentity{Height: -1} || *b == BlockIdentity{}
}

//OutputKind描述一种事务输出。这习惯于
//区分coinbase、stakebase和正常输出。
type OutputKind byte

//定义的OutputKind常量
const (
	OutputKindNormal OutputKind = iota
	OutputKindCoinbase
)

//TransactionOutput描述的输出至少是部分或部分
//由钱包控制。根据上下文，这可能指
//未消耗的输出或已消耗的输出。
type TransactionOutput struct {
	OutPoint   wire.OutPoint
	Output     wire.TxOut
	OutputKind OutputKind
//当数据库可以返回更多信息时，应该稍后添加这些信息。
//有效地：
//tx锁定时间uint32
//tx到期日期32
	ContainingBlock BlockIdentity
	ReceiveTime     time.Time
}

//OutputRedeemer标识用于重新定义输出的事务输入。
type OutputRedeemer struct {
	TxHash     chainhash.Hash
	InputIndex uint32
}

//p2shmultisigoutput描述了一个事务输出，其中包含一个付费脚本哈希
//输出脚本和导入的赎回脚本。以及常见的细节
//在输出中，这个结构还包括脚本的p2sh地址
//从中创建，以及兑换它所需的签名数。
//
//TODO:返回所需签名的数量可能很有用
//由这个钱包创建。
type P2SHMultiSigOutput struct {
//TODO:向此结构添加TransactionOutput成员并删除这些成员
//由它复制的字段。这提高了一致性。只有
//现在还没有完成，因为wtxmgr API不支持
//正在获取其他事务输出数据以及
//多重信息。
	OutPoint        wire.OutPoint
	OutputAmount    btcutil.Amount
	ContainingBlock BlockIdentity

	P2SHAddress  *btcutil.AddressScriptHash
	RedeemScript []byte
M, N         uint8           //需要m/n签名才能兑现
Redeemer     *OutputRedeemer //无，除非用完
}
