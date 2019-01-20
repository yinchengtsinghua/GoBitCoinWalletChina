
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package chain

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/waddrmgr"
)

//BlockFilter用于迭代扫描块以查找
//兴趣。这是通过构造反向索引映射
//ScopedIndex的地址，它允许
//报告匹配项的子派生路径。
//
//一旦初始化，块过滤器就可以用来扫描任意数量的块。
//直到对“filterblock”的调用返回true。这允许反向
//在地址集不需要恢复的情况下要恢复的索引
//被改变。在报告匹配之后，新的BlockFilter应该
//用包含任何新键的更新地址集初始化
//现在在我们的展望之内。
//
//我们分别跟踪内部和外部地址，以节省
//内存中占用的空间量。具体来说，帐户和分支机构
//使用默认作用域时，组合只贡献1位信息
//钱包用的。因此，我们可以避免在
//通过不存储完整的派生路径来实现感兴趣的地址，而
//选择允许调用者上下文推断帐户（默认帐户）
//和分支（内部或外部）。
type BlockFilterer struct {
//params指定当前网络的链参数。
	Params *chaincfg.Params

//exReverseFilter保存将外部地址映射到的反向索引
//派生它的作用域索引。
	ExReverseFilter map[string]waddrmgr.ScopedIndex

//inReverseFilter保存一个反向索引，将内部地址映射到
//派生它的作用域索引。
	InReverseFilter map[string]waddrmgr.ScopedIndex

//WathcedOutpoints是一组由
//钱包。这允许块过滤器检查
//超越自我。
	WatchedOutPoints map[wire.OutPoint]btcutil.Address

//FoundExternal是一个两层地图，记录了
//
	FoundExternal map[waddrmgr.KeyScope]map[uint32]struct{}

//
//在单个块中找到内部地址。
	FoundInternal map[waddrmgr.KeyScope]map[uint32]struct{}

//found outpoints是在单个块中找到的一组输出点，其
//地址属于钱包。
	FoundOutPoints map[wire.OutPoint]btcutil.Address

//relevanttxns记录在特定块中找到的事务
//包含来自exReverseFilter或
//在每个过滤器中。
	RelevantTxns []*wire.MsgTx
}

//newblockfilter为当前的
//我们正在搜索的外部和内部地址，用于
//扫描连续块以查找感兴趣的地址。特定的块过滤器
//可以重复使用，直到“fitlerblock”的第一个调用返回true。
func NewBlockFilterer(params *chaincfg.Params,
	req *FilterBlocksRequest) *BlockFilterer {

//按请求的地址字符串构造反向索引
//外部地址。
	nExAddrs := len(req.ExternalAddrs)
	exReverseFilter := make(map[string]waddrmgr.ScopedIndex, nExAddrs)
	for scopedIndex, addr := range req.ExternalAddrs {
		exReverseFilter[addr.EncodeAddress()] = scopedIndex
	}

//按请求的地址字符串构造反向索引
//内部地址。
	nInAddrs := len(req.InternalAddrs)
	inReverseFilter := make(map[string]waddrmgr.ScopedIndex, nInAddrs)
	for scopedIndex, addr := range req.InternalAddrs {
		inReverseFilter[addr.EncodeAddress()] = scopedIndex
	}

	foundExternal := make(map[waddrmgr.KeyScope]map[uint32]struct{})
	foundInternal := make(map[waddrmgr.KeyScope]map[uint32]struct{})
	foundOutPoints := make(map[wire.OutPoint]btcutil.Address)

	return &BlockFilterer{
		Params:           params,
		ExReverseFilter:  exReverseFilter,
		InReverseFilter:  inReverseFilter,
		WatchedOutPoints: req.WatchedOutPoints,
		FoundExternal:    foundExternal,
		FoundInternal:    foundInternal,
		FoundOutPoints:   foundOutPoints,
	}
}

//filterblock解析所提供块中的所有txn，并搜索
//包含外部或内部背面感兴趣的地址
//过滤器。如果块包含非零的
//感兴趣的地址，或块中的事务从输出点支出
//由钱包控制。
func (bf *BlockFilterer) FilterBlock(block *wire.MsgBlock) bool {
	var hasRelevantTxns bool
	for _, tx := range block.Transactions {
		if bf.FilterTx(tx) {
			bf.RelevantTxns = append(bf.RelevantTxns, tx)
			hasRelevantTxns = true
		}
	}

	return hasRelevantTxns
}

//filtertx扫描提供的txn中的所有txout，测试是否找到
//地址与外部或内部反向中包含的地址匹配
//索引。如果txn包含非零的
//感兴趣的地址，或交易支出来自
//属于钱包。
func (bf *BlockFilterer) FilterTx(tx *wire.MsgTx) bool {
	var isRelevant bool

//首先，检查这个事务的输入，看看它们是否花费了
//属于钱包的输入。除了检查
//观察输出点，我们还检查foundoutpoints，以防txn花费
//来自在同一块中创建的输出点。
	for _, in := range tx.TxIn {
		if _, ok := bf.WatchedOutPoints[in.PreviousOutPoint]; ok {
			isRelevant = true
		}
		if _, ok := bf.FoundOutPoints[in.PreviousOutPoint]; ok {
			isRelevant = true
		}
	}

//现在，分析此事务创建的所有输出，并查看
//如果他们有任何地址知道钱包使用我们的背面
//外部和内部地址的索引。如果新输出是
//找到后，我们将把输出点添加到我们的foundoutpoints集合中。
	for i, out := range tx.TxOut {
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(
			out.PkScript, bf.Params,
		)
		if err != nil {
			log.Warnf("Could not parse output script in %s:%d: %v",
				tx.TxHash(), i, err)
			continue
		}

		if !bf.FilterOutputAddrs(addrs) {
			continue
		}

//如果我们已经达到这一点，那么输出包含
//感兴趣的地址。
		isRelevant = true

//将包含地址的输出点记录到
//找到个输出点，以便调用方可以更新其全局
//一组被监视的输出点。
		outPoint := wire.OutPoint{
			Hash:  *btcutil.NewTx(tx).Hash(),
			Index: uint32(i),
		}

		bf.FoundOutPoints[outPoint] = addrs[0]
	}

	return isRelevant
}

//filteroutputaddrs根据块筛选器测试地址集
//外部和内部反向地址索引。如果找到了，他们是
//分别添加到外部和内部找到的地址集。这个
//方法返回true如果提供的地址的非零个数为
//兴趣。
func (bf *BlockFilterer) FilterOutputAddrs(addrs []btcutil.Address) bool {
	var isRelevant bool
	for _, addr := range addrs {
		addrStr := addr.EncodeAddress()
		if scopedIndex, ok := bf.ExReverseFilter[addrStr]; ok {
			bf.foundExternal(scopedIndex)
			isRelevant = true
		}
		if scopedIndex, ok := bf.InReverseFilter[addrStr]; ok {
			bf.foundInternal(scopedIndex)
			isRelevant = true
		}
	}

	return isRelevant
}

//FoundExternal将作用域索引标记为在块筛选器中找到的索引
//找到外部映射。如果这是为特定范围找到的第一个索引，
//在标记索引之前，将初始化作用域的第二层映射。
func (bf *BlockFilterer) foundExternal(scopedIndex waddrmgr.ScopedIndex) {
	if _, ok := bf.FoundExternal[scopedIndex.Scope]; !ok {
		bf.FoundExternal[scopedIndex.Scope] = make(map[uint32]struct{})
	}
	bf.FoundExternal[scopedIndex.Scope][scopedIndex.Index] = struct{}{}
}

//FoundInternal将作用域索引标记为在块筛选器中找到的
//找到内部映射。如果这是为特定范围找到的第一个索引，
//在标记索引之前，将初始化作用域的第二层映射。
func (bf *BlockFilterer) foundInternal(scopedIndex waddrmgr.ScopedIndex) {
	if _, ok := bf.FoundInternal[scopedIndex.Scope]; !ok {
		bf.FoundInternal[scopedIndex.Scope] = make(map[uint32]struct{})
	}
	bf.FoundInternal[scopedIndex.Scope][scopedIndex.Index] = struct{}{}
}
