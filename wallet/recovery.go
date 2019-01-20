
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package wallet

import (
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/btcsuite/btcwallet/wtxmgr"
)

//
//
type RecoveryManager struct {
//recoveryWindow定义在
//attempting to recover the set of used addresses.
	recoveryWindow uint32

//将第一个块添加到批处理后，Started为true。
	started bool

//blockBatch contains a list of blocks that have not yet been searched
//for recovered addresses.
	blockBatch []wtxmgr.BlockMeta

//state encapsulates and allocates the necessary recovery state for all
//key scopes and subsidiary derivation paths.
	state *RecoveryState

//chainParams are the parameters that describe the chain we're trying
//以收回资金。
	chainParams *chaincfg.Params
}

//NewRecoveryManager initializes a new RecoveryManager with a derivation
//先看“恢复窗口”子索引，然后预先分配一个后备文件。
//一次扫描“batchsize”块的数组。
func NewRecoveryManager(recoveryWindow, batchSize uint32,
	chainParams *chaincfg.Params) *RecoveryManager {

	return &RecoveryManager{
		recoveryWindow: recoveryWindow,
		blockBatch:     make([]wtxmgr.BlockMeta, 0, batchSize),
		chainParams:    chainParams,
		state:          NewRecoveryState(recoveryWindow),
	}
}

//恢复为提供的作用域还原所有已知地址
//在walletdb命名空间中找到，除了还原
//以前发现过。此方法确保恢复状态
//水平线从上次找到的恢复地址正确开始
//尝试。
func (rm *RecoveryManager) Resurrect(ns walletdb.ReadBucket,
	scopedMgrs map[waddrmgr.KeyScope]*waddrmgr.ScopedKeyManager,
	credits []wtxmgr.Credit) error {

//首先，对于我们正在恢复的每个作用域，重新驱动
//每个分支最后找到的地址。
	for keyScope, scopedMgr := range scopedMgrs {
//加载此作用域的当前帐户属性，使用
//默认帐号。
//TODO（Conner）：如果允许，请重新扫描所有已创建的帐户
//users to use non-default address
		scopeState := rm.state.StateForScope(keyScope)
		acctProperties, err := scopedMgr.AccountProperties(
			ns, waddrmgr.DefaultAccountNum,
		)
		if err != nil {
			return err
		}

//获取外部键计数，它限制了
//需要重新驾驶。
		externalCount := acctProperties.ExternalKeyCount

//通过最后一个外部键遍历所有索引，
//派生每个地址并将其添加到外部分支
//要查找的恢复状态地址集。
		for i := uint32(0); i < externalCount; i++ {
			keyPath := externalKeyPath(i)
			addr, err := scopedMgr.DeriveFromKeyPath(ns, keyPath)
			if err != nil && err != hdkeychain.ErrInvalidChild {
				return err
			} else if err == hdkeychain.ErrInvalidChild {
				scopeState.ExternalBranch.MarkInvalidChild(i)
				continue
			}

			scopeState.ExternalBranch.AddAddr(i, addr.Address())
		}

//获取内部键计数，它限制了
//需要重新驾驶。
		internalCount := acctProperties.InternalKeyCount

//
//派生每个地址并将其添加到内部分支
//要查找的恢复状态地址集。
		for i := uint32(0); i < internalCount; i++ {
			keyPath := internalKeyPath(i)
			addr, err := scopedMgr.DeriveFromKeyPath(ns, keyPath)
			if err != nil && err != hdkeychain.ErrInvalidChild {
				return err
			} else if err == hdkeychain.ErrInvalidChild {
				scopeState.InternalBranch.MarkInvalidChild(i)
				continue
			}

			scopeState.InternalBranch.AddAddr(i, addr.Address())
		}

//关键点计数将指向下一个可以
//派生，所以我们减去一个以指向最后一个已知键。如果
//键计数为零，则找不到地址。
		if externalCount > 0 {
			scopeState.ExternalBranch.ReportFound(externalCount - 1)
		}
		if internalCount > 0 {
			scopeState.InternalBranch.ReportFound(internalCount - 1)
		}
	}

//此外，我们将重新添加任何已知钱包的输出点。
//我们的全球监视输出点集合，以便我们可以监视它们
//花费。
	for _, credit := range credits {
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(
			credit.PkScript, rm.chainParams,
		)
		if err != nil {
			return err
		}

		rm.state.AddWatchedOutPoint(&credit.OutPoint, addrs[0])
	}

	return nil
}

//addToBlockBatch附加块信息，包括哈希和高度，
//到要搜索的一批块。
func (rm *RecoveryManager) AddToBlockBatch(hash *chainhash.Hash, height int32,
	timestamp time.Time) {

	if !rm.started {
		log.Infof("Seed birthday surpassed, starting recovery "+
			"of wallet from height=%d hash=%v with "+
			"recovery-window=%d", height, *hash, rm.recoveryWindow)
		rm.started = true
	}

	block := wtxmgr.BlockMeta{
		Block: wtxmgr.Block{
			Hash:   *hash,
			Height: height,
		},
		Time: timestamp,
	}
	rm.blockBatch = append(rm.blockBatch, block)
}

//blockbatch返回尚未搜索的块的缓冲区。
func (rm *RecoveryManager) BlockBatch() []wtxmgr.BlockMeta {
	return rm.blockBatch
}

//resetblockbatch重置内部块缓冲区以节省内存。
func (rm *RecoveryManager) ResetBlockBatch() {
	rm.blockBatch = rm.blockBatch[:0]
}

//状态返回当前的恢复状态。
func (rm *RecoveryManager) State() *RecoveryState {
	return rm.state
}

//recoveryState管理ScopereCoveryStates的初始化和查找
//对于任何活动使用的键范围。
//
//为了确保正确恢复所有地址，窗口
//大小应为最大可能块间和块内的总和
//特定分支的已用地址之间的间隔。
//
//这些定义如下：
//-块间间隙：派生子索引之间的最大差异
//任何块中使用的最后一个地址和使用的下一个地址
//在后面的街区。
//-块内间隙：派生子索引之间的最大差异
//任何块中使用的第一个地址和
//同一块。
type RecoveryState struct {
//recoveryWindow定义在
//正在尝试恢复已使用的地址集。这个值将是
//用于为每个请求的作用域实例化新的recoverystate。
	recoveryWindow uint32

//作用域维护每个请求的键作用域到其活动的映射
//恢复状态。
	scopes map[waddrmgr.KeyScope]*ScopeRecoveryState

//watchedOutpoints包含
//钱包。当在
//重新扫描
	watchedOutPoints map[wire.OutPoint]btcutil.Address
}

//new recoverystate使用提供的
//
//特定的键作用域将接收相同的recoveryWindow。
func NewRecoveryState(recoveryWindow uint32) *RecoveryState {
	scopes := make(map[waddrmgr.KeyScope]*ScopeRecoveryState)

	return &RecoveryState{
		recoveryWindow:   recoveryWindow,
		scopes:           scopes,
		watchedOutPoints: make(map[wire.OutPoint]btcutil.Address),
	}
}

//StateForScope为提供的键作用域返回ScopereCoveryState。如果一
//不存在，将使用recoverystate的
//恢复窗口。
func (rs *RecoveryState) StateForScope(
	keyScope waddrmgr.KeyScope) *ScopeRecoveryState {

//如果帐户恢复状态已存在，请将其返回。
	if scopeState, ok := rs.scopes[keyScope]; ok {
		return scopeState
	}

//否则，使用
//选择的恢复窗口。
	rs.scopes[keyScope] = NewScopeRecoveryState(rs.recoveryWindow)

	return rs.scopes[keyScope]
}

//watchedOutpoints返回已知属于的全局输出点集
//在恢复过程中放在钱包里。
func (rs *RecoveryState) WatchedOutPoints() map[wire.OutPoint]btcutil.Address {
	return rs.watchedOutPoints
}

//addWatchedOutpoint更新恢复状态的已知输出点集
//我们将监控恢复期间的支出。
func (rs *RecoveryState) AddWatchedOutPoint(outPoint *wire.OutPoint,
	addr btcutil.Address) {

	rs.watchedOutPoints[*outPoint] = addr
}

//ScopereCoveryState用于管理所生成地址的恢复
//在一个特定的BIP32帐户下。每个帐户同时跟踪外部和
//内部分支恢复状态，两者都使用相同的恢复窗口。
type ScopeRecoveryState struct {
//ExternalBranch是为生成的地址的恢复状态
//外部使用，即接收地址。
	ExternalBranch *BranchRecoveryState

//InternalBranch是为生成的地址的恢复状态
//内部使用，即更改地址。
	InternalBranch *BranchRecoveryState
}

//newscoperecoveryState用所选的初始化scoperecoveryState
//恢复窗口。
func NewScopeRecoveryState(recoveryWindow uint32) *ScopeRecoveryState {
	return &ScopeRecoveryState{
		ExternalBranch: NewBranchRecoveryState(recoveryWindow),
		InternalBranch: NewBranchRecoveryState(recoveryWindow),
	}
}

//BranchRecoveryState保持所需状态以便正确
//恢复从特定帐户的内部或外部派生的地址
//派生分支。
//
//分支恢复状态支持以下操作：
//-根据找到的索引扩展展望范围。
//-在范围内用索引注册派生地址。
//-报告属于范围的无效子索引。
//-报告已找到地址。
//-检索分支的所有当前派生地址。
//-通过子索引查找特定地址。
type BranchRecoveryState struct {
//recoveryWindow定义在
//正在尝试恢复此分支上的地址集。
	recoveryWindow uint32

//Horizion记录了这个分支所监视的最高子索引。
	horizon uint32

//nextunfound将继承者的子索引维护为
//在恢复此分支期间找到的索引。
	nextUnfound uint32

//地址是子索引到所有活动监视的地址的映射
//
	addresses map[uint32]btcutil.Address

//invalidChildren记录派生到的子索引集
//无效的密钥。
	invalidChildren map[uint32]struct{}
}

//
//跟踪帐户派生路径的外部或内部分支。
func NewBranchRecoveryState(recoveryWindow uint32) *BranchRecoveryState {
	return &BranchRecoveryState{
		recoveryWindow:  recoveryWindow,
		addresses:       make(map[uint32]btcutil.Address),
		invalidChildren: make(map[uint32]struct{}),
	}
}

//extendhorizon返回当前范围和
//必须派生才能保持所需的恢复窗口。
func (brs *BranchRecoveryState) ExtendHorizon() (uint32, uint32) {

//计算新的地平线，它应该超过我们最后找到的地址
//在恢复窗口旁边。
	curHorizon := brs.horizon

	nInvalid := brs.NumInvalidInHorizon()
	minValidHorizon := brs.nextUnfound + brs.recoveryWindow + nInvalid

//如果当前的视界足够，我们就不必推导出
//新的钥匙。
	if curHorizon >= minValidHorizon {
		return curHorizon, 0
	}

//否则，我们应该派生的地址数对应于
//两个地平线的三角洲，我们更新了我们的新地平线。
	delta := minValidHorizon - curHorizon
	brs.horizon = minValidHorizon

	return curHorizon, delta
}

//addaddr将lookahead中新派生的地址添加到
//此分支的已知地址。
func (brs *BranchRecoveryState) AddAddr(index uint32, addr btcutil.Address) {
	brs.addresses[index] = addr
}

//getaddr返回从给定子索引派生的地址。
func (brs *BranchRecoveryState) GetAddr(index uint32) btcutil.Address {
	return brs.addresses[index]
}

//如果报告的索引超过
//当前值。
func (brs *BranchRecoveryState) ReportFound(index uint32) {
	if index >= brs.nextUnfound {
		brs.nextUnfound = index + 1

//删除所有低于上一个索引的无效子索引
//发现指数。我们不需要再保留这些条目了，
//因为它们不会影响我们所要求的展望。
		for childIndex := range brs.invalidChildren {
			if childIndex < index {
				delete(brs.invalidChildren, childIndex)
			}
		}
	}
}

//markinvalidchild记录特定子索引导致
//地址无效。此外，正如我们所期望的，分支的范围是递增的。
//调用方执行其他派生以替换无效的子级。
//这是用来确保我们总是有适当的前瞻性，当
//遇到无效的子级。
func (brs *BranchRecoveryState) MarkInvalidChild(index uint32) {
	brs.invalidChildren[index] = struct{}{}
	brs.horizon++
}

//nextunfound返回找到的最高继承者的子索引
//子索引。
func (brs *BranchRecoveryState) NextUnfound() uint32 {
	return brs.nextUnfound
}

//addrs返回当前所有派生子索引到其
//相应的地址。
func (brs *BranchRecoveryState) Addrs() map[uint32]btcutil.Address {
	return brs.addresses
}

//
//在最后发现的和当前地平线之间。这将通知您还有多少
//为保持正确数量的有效地址而派生的索引
//在我们的视野之内。
func (brs *BranchRecoveryState) NumInvalidInHorizon() uint32 {
	var nInvalid uint32
	for childIndex := range brs.invalidChildren {
		if brs.nextUnfound <= childIndex && childIndex < brs.horizon {
			nInvalid++
		}
	}

	return nInvalid
}
