
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
 *版权所有（c）2015-2016 BTCSuite开发者
 *
 *使用、复制、修改和分发本软件的权限
 *特此授予免费或不收费的目的，前提是上述
 *版权声明和本许可声明出现在所有副本中。
 *
 *本软件按“原样”提供，作者不作任何保证。
 *关于本软件，包括
 *适销性和适用性。在任何情况下，作者都不对
 *任何特殊、直接、间接或后果性损害或任何损害
 *因使用、数据或利润损失而导致的任何情况，无论是在
 *因以下原因引起的合同诉讼、疏忽或其他侵权行为：
 *或与本软件的使用或性能有关。
 **/


package votingpool

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/btcsuite/btcwallet/wtxmgr"
)

const eligibleInputMinConfirmations = 100

//信贷是对wtxmgr的抽象。信贷用于
//投票池提款交易。
type credit struct {
	wtxmgr.Credit
	addr WithdrawalAddress
}

func newCredit(c wtxmgr.Credit, addr WithdrawalAddress) credit {
	return credit{Credit: c, addr: addr}
}

func (c *credit) String() string {
	return fmt.Sprintf("credit of %v locked to %v", c.Amount, c.addr)
}

//ByAddress定义满足排序所需的方法。用于排序的接口
//按他们的地址划分学分。
type byAddress []credit

func (c byAddress) Len() int      { return len(c) }
func (c byAddress) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

//
//位置j处的元素。“小于”关系定义为
//在元组（seriesid、index、index、index和
//分支，txsha，outputindex）。
func (c byAddress) Less(i, j int) bool {
	iAddr := c[i].addr
	jAddr := c[j].addr
	if iAddr.seriesID < jAddr.seriesID {
		return true
	}
	if iAddr.seriesID > jAddr.seriesID {
		return false
	}

//序列ID相等，因此请比较索引。
	if iAddr.index < jAddr.index {
		return true
	}
	if iAddr.index > jAddr.index {
		return false
	}

//seriesid和index相等，所以比较branch。
	if iAddr.branch < jAddr.branch {
		return true
	}
	if iAddr.branch > jAddr.branch {
		return false
	}

//seriesid、index和branch相等，所以比较hash。
	txidComparison := bytes.Compare(c[i].OutPoint.Hash[:], c[j].OutPoint.Hash[:])
	if txidComparison < 0 {
		return true
	}
	if txidComparison > 0 {
		return false
	}

//seriesid、index、branch和hash相等，因此比较输出
//索引。
	return c[i].OutPoint.Index < c[j].OutPoint.Index
}

//GetEligibleInputs返回地址介于StartAddress之间的合格输入
//最后使用的地址是lastseriesid。它们是根据
//他们的地址。
func (p *Pool) getEligibleInputs(ns, addrmgrNs walletdb.ReadBucket, store *wtxmgr.Store, txmgrNs walletdb.ReadBucket, startAddress WithdrawalAddress,
	lastSeriesID uint32, dustThreshold btcutil.Amount, chainHeight int32,
	minConf int) ([]credit, error) {

	if p.Series(lastSeriesID) == nil {
		str := fmt.Sprintf("lastSeriesID (%d) does not exist", lastSeriesID)
		return nil, newError(ErrSeriesNotExists, str, nil)
	}
	unspents, err := store.UnspentOutputs(txmgrNs)
	if err != nil {
		return nil, newError(ErrInputSelection, "failed to get unspent outputs", err)
	}
	addrMap, err := groupCreditsByAddr(unspents, p.manager.ChainParams())
	if err != nil {
		return nil, err
	}
	var inputs []credit
	address := startAddress
	for {
		log.Debugf("Looking for eligible inputs at address %v", address.addrIdentifier())
		if candidates, ok := addrMap[address.addr.EncodeAddress()]; ok {
			var eligibles []credit
			for _, c := range candidates {
				candidate := newCredit(c, address)
				if p.isCreditEligible(candidate, minConf, chainHeight, dustThreshold) {
					eligibles = append(eligibles, candidate)
				}
			}
			inputs = append(inputs, eligibles...)
		}
		nAddr, err := nextAddr(p, ns, addrmgrNs, address.seriesID, address.branch, address.index, lastSeriesID+1)
		if err != nil {
			return nil, newError(ErrInputSelection, "failed to get next withdrawal address", err)
		} else if nAddr == nil {
			log.Debugf("getEligibleInputs: reached last addr, stopping")
			break
		}
		address = *nAddr
	}
	sort.Sort(sort.Reverse(byAddress(inputs)))
	return inputs, nil
}

//NextAddr根据输入选择返回下一个提取地址
//规则：http://opentransactions.org/wiki/index.php/input-selection-algorithm-pools
//如果新地址'seriesid大于等于stopseriesid，则返回nil。
func nextAddr(p *Pool, ns, addrmgrNs walletdb.ReadBucket, seriesID uint32, branch Branch, index Index, stopSeriesID uint32) (
	*WithdrawalAddress, error) {
	series := p.Series(seriesID)
	if series == nil {
		return nil, newError(ErrSeriesNotExists, fmt.Sprintf("unknown seriesID: %d", seriesID), nil)
	}
	branch++
	if int(branch) > len(series.publicKeys) {
		highestIdx, err := p.highestUsedSeriesIndex(ns, seriesID)
		if err != nil {
			return nil, err
		}
		if index > highestIdx {
			seriesID++
			log.Debugf("nextAddr(): reached last branch (%d) and highest used index (%d), "+
				"moving on to next series (%d)", branch, index, seriesID)
			index = 0
		} else {
			index++
		}
		branch = 0
	}

	if seriesID >= stopSeriesID {
		return nil, nil
	}

	addr, err := p.WithdrawalAddress(ns, addrmgrNs, seriesID, branch, index)
	if err != nil && err.(Error).ErrorCode == ErrWithdrawFromUnusedAddr {
//使用的索引会因分支而异，因此有时我们会尝试
//获取一个以前没有使用过的取款地址
//我们只需要继续下一个。
		log.Debugf("nextAddr(): skipping addr (series #%d, branch #%d, index #%d) as it hasn't "+
			"been used before", seriesID, branch, index)
		return nextAddr(p, ns, addrmgrNs, seriesID, branch, index, stopSeriesID)
	}
	return addr, err
}

//highestusedseriesindex返回此池中所有
//已使用给定序列ID的地址。如果没有使用则返回0
//具有给定序列ID的地址。
func (p *Pool) highestUsedSeriesIndex(ns walletdb.ReadBucket, seriesID uint32) (Index, error) {
	maxIdx := Index(0)
	series := p.Series(seriesID)
	if series == nil {
		return maxIdx,
			newError(ErrSeriesNotExists, fmt.Sprintf("unknown seriesID: %d", seriesID), nil)
	}
	for i := range series.publicKeys {
		idx, err := p.highestUsedIndexFor(ns, seriesID, Branch(i))
		if err != nil {
			return Index(0), err
		}
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	return maxIdx, nil
}

//GroupCreditsByddr将信用片从字符串转换为映射
//对与相关联的未暂停输出的编码地址的表示
//那个地址。
func groupCreditsByAddr(credits []wtxmgr.Credit, chainParams *chaincfg.Params) (
	map[string][]wtxmgr.Credit, error) {
	addrMap := make(map[string][]wtxmgr.Credit)
	for _, c := range credits {
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(c.PkScript, chainParams)
		if err != nil {
			return nil, newError(ErrInputSelection, "failed to obtain input address", err)
		}
//因为我们的学分都是P2SH，所以我们不应该超过一个
//每个信用证的地址，所以如果假设是
//违反。
		if len(addrs) != 1 {
			return nil, newError(ErrInputSelection, "input doesn't have exactly one address", nil)
		}
		encAddr := addrs[0].EncodeAddress()
		if v, ok := addrMap[encAddr]; ok {
			addrMap[encAddr] = append(v, c)
		} else {
			addrMap[encAddr] = []wtxmgr.Credit{c}
		}
	}

	return addrMap, nil
}

//iscreditlegibility测试给定的可信度
//到确认的次数，灰尘阈值和它不是
//宪章产出。
func (p *Pool) isCreditEligible(c credit, minConf int, chainHeight int32,
	dustThreshold btcutil.Amount) bool {
	if c.Amount < dustThreshold {
		return false
	}
	if confirms(c.BlockMeta.Block.Height, chainHeight) < int32(minConf) {
		return false
	}
	if p.isCharterOutput(c) {
		return false
	}

	return true
}

//ischarteroutput-todo:为了确定这一点，我们需要txid
//以及目前租船产量的产出指数，我们还没有。
func (p *Pool) isCharterOutput(c credit) bool {
	return false
}

//确认返回块中某个事务的确认数
//高度tx高度（或未确认tx的-1）给定链条高度
//光洁度。
func confirms(txHeight, curHeight int32) int32 {
	switch {
	case txHeight == -1, txHeight > curHeight:
		return 0
	default:
		return curHeight - txHeight + 1
	}
}
