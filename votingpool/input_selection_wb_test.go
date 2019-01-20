
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package votingpool

import (
	"bytes"
	"reflect"
	"sort"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/btcsuite/btcwallet/wtxmgr"
)

var (
//随机使用少量的Satoshis作为灰尘阈值
	dustThreshold btcutil.Amount = 1e4
)

func TestGetEligibleInputs(t *testing.T) {
	tearDown, db, pool, store := TstCreatePoolAndTxStore(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, addrmgrNs := TstRWNamespaces(dbtx)

	series := []TstSeriesDef{
		{ReqSigs: 2, PubKeys: TstPubKeys[1:4], SeriesID: 1},
		{ReqSigs: 2, PubKeys: TstPubKeys[3:6], SeriesID: 2},
	}
	TstCreateSeries(t, dbtx, pool, series)
	scripts := append(
		getPKScriptsForAddressRange(t, dbtx, pool, 1, 0, 2, 0, 4),
		getPKScriptsForAddressRange(t, dbtx, pool, 2, 0, 2, 0, 6)...)

//创建两个锁定到上面每个pkscripts的合格输入。
	expNoEligibleInputs := 2 * len(scripts)
	eligibleAmounts := []int64{int64(dustThreshold + 1), int64(dustThreshold + 1)}
	var inputs []wtxmgr.Credit
	for i := 0; i < len(scripts); i++ {
		created := TstCreateCreditsOnStore(t, dbtx, store, scripts[i], eligibleAmounts)
		inputs = append(inputs, created...)
	}

	startAddr := TstNewWithdrawalAddress(t, dbtx, pool, 1, 0, 0)
	lastSeriesID := uint32(2)
	currentBlock := int32(TstInputsBlock + eligibleInputMinConfirmations + 1)
	var eligibles []credit
	txmgrNs := dbtx.ReadBucket(txmgrNamespaceKey)
	TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		eligibles, err = pool.getEligibleInputs(ns, addrmgrNs,
			store, txmgrNs, *startAddr, lastSeriesID, dustThreshold, int32(currentBlock),
			eligibleInputMinConfirmations)
	})
	if err != nil {
		t.Fatal("InputSelection failed:", err)
	}

//检查我们得到了预期数量的合格输入。
	if len(eligibles) != expNoEligibleInputs {
		t.Fatalf("Wrong number of eligible inputs returned. Got: %d, want: %d.",
			len(eligibles), expNoEligibleInputs)
	}

//检查返回的eligibles是否按地址反向排序。
	if !sort.IsSorted(sort.Reverse(byAddress(eligibles))) {
		t.Fatal("Eligible inputs are not sorted.")
	}

//检查所有学分是否唯一
	checkUniqueness(t, eligibles)
}

func TestNextAddrWithVaryingHighestIndices(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, addrmgrNs := TstRWNamespaces(dbtx)

	series := []TstSeriesDef{
		{ReqSigs: 2, PubKeys: TstPubKeys[1:4], SeriesID: 1},
	}
	TstCreateSeries(t, dbtx, pool, series)
	stopSeriesID := uint32(2)

//填充分支0和0到2之间的索引的已用addr db。
	TstEnsureUsedAddr(t, dbtx, pool, 1, Branch(0), 2)

//填充分支1的已用addr db和从0到1的索引。
	TstEnsureUsedAddr(t, dbtx, pool, 1, Branch(1), 1)

//从branch==0，index==1的地址开始。
	addr := TstNewWithdrawalAddress(t, dbtx, pool, 1, 0, 1)

//对nextaddr（）的第一个调用应该给出branch==1的地址
//索引＝1。
	TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		addr, err = nextAddr(pool, ns, addrmgrNs, addr.seriesID, addr.branch, addr.index, stopSeriesID)
	})
	if err != nil {
		t.Fatalf("Failed to get next address: %v", err)
	}
	checkWithdrawalAddressMatches(t, addr, 1, Branch(1), 1)

//下一个调用应该给出branch==0，index==2的地址，因为
//分支==2没有使用过的地址。
	TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		addr, err = nextAddr(pool, ns, addrmgrNs, addr.seriesID, addr.branch, addr.index, stopSeriesID)
	})
	if err != nil {
		t.Fatalf("Failed to get next address: %v", err)
	}
	checkWithdrawalAddressMatches(t, addr, 1, Branch(0), 2)

//由于branch==1的最后一个地址是index==1的地址，因此
//呼叫将返回零。
	TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		addr, err = nextAddr(pool, ns, addrmgrNs, addr.seriesID, addr.branch, addr.index, stopSeriesID)
	})
	if err != nil {
		t.Fatalf("Failed to get next address: %v", err)
	}
	if addr != nil {
		t.Fatalf("Wrong next addr; got '%s', want 'nil'", addr.addrIdentifier())
	}
}

func TestNextAddr(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, addrmgrNs := TstRWNamespaces(dbtx)

	series := []TstSeriesDef{
		{ReqSigs: 2, PubKeys: TstPubKeys[1:4], SeriesID: 1},
		{ReqSigs: 2, PubKeys: TstPubKeys[3:6], SeriesID: 2},
	}
	TstCreateSeries(t, dbtx, pool, series)
	stopSeriesID := uint32(3)

	lastIdx := Index(10)
//用seriesid==1，branch==0..3的条目填充已用地址数据库，
//Idx＝0…10。
	for _, i := range []int{0, 1, 2, 3} {
		TstEnsureUsedAddr(t, dbtx, pool, 1, Branch(i), lastIdx)
	}
	addr := TstNewWithdrawalAddress(t, dbtx, pool, 1, 0, lastIdx-1)
//nextAddr（）首先只增加分支，范围从0到3
//这里（因为我们的系列有3个公钥）。
	for _, i := range []int{1, 2, 3} {
		TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
			addr, err = nextAddr(pool, ns, addrmgrNs, addr.seriesID, addr.branch, addr.index, stopSeriesID)
		})
		if err != nil {
			t.Fatalf("Failed to get next address: %v", err)
		}
		checkWithdrawalAddressMatches(t, addr, 1, Branch(i), lastIdx-1)
	}

//
//idx=lastidx-1，所以接下来的4个呼叫应该给我们地址
//branch=[0-3]和idx=lastidx。
	for _, i := range []int{0, 1, 2, 3} {
		TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
			addr, err = nextAddr(pool, ns, addrmgrNs, addr.seriesID, addr.branch, addr.index, stopSeriesID)
		})
		if err != nil {
			t.Fatalf("Failed to get next address: %v", err)
		}
		checkWithdrawalAddressMatches(t, addr, 1, Branch(i), lastIdx)
	}

//用seriesid==2，branch==0..3的条目填充已用地址数据库，
//Idx＝0…10。
	for _, i := range []int{0, 1, 2, 3} {
		TstEnsureUsedAddr(t, dbtx, pool, 2, Branch(i), lastIdx)
	}
//现在我们已经完成了所有可用的分支/IDX组合，所以
//我们应该转到下一个系列，从branch=0，idx=0重新开始。
	for _, i := range []int{0, 1, 2, 3} {
		TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
			addr, err = nextAddr(pool, ns, addrmgrNs, addr.seriesID, addr.branch, addr.index, stopSeriesID)
		})
		if err != nil {
			t.Fatalf("Failed to get next address: %v", err)
		}
		checkWithdrawalAddressMatches(t, addr, 2, Branch(i), 0)
	}

//最后检查nextaddr（）在我们到达最后一个时是否返回nil
//stopseriesid之前的可用地址。
	addr = TstNewWithdrawalAddress(t, dbtx, pool, 2, 3, lastIdx)
	TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		addr, err = nextAddr(pool, ns, addrmgrNs, addr.seriesID, addr.branch, addr.index, stopSeriesID)
	})
	if err != nil {
		t.Fatalf("Failed to get next address: %v", err)
	}
	if addr != nil {
		t.Fatalf("Wrong WithdrawalAddress; got %s, want nil", addr.addrIdentifier())
	}
}

func TestEligibleInputsAreEligible(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	var chainHeight int32 = 1000
	_, credits := TstCreateCreditsOnNewSeries(t, dbtx, pool, []int64{int64(dustThreshold)})
	c := credits[0]
//确保信用卡足够旧，可以通过minconf支票。
	c.BlockMeta.Height = int32(eligibleInputMinConfirmations)

	if !pool.isCreditEligible(c, eligibleInputMinConfirmations, chainHeight, dustThreshold) {
		t.Errorf("Input is not eligible and it should be.")
	}
}

func TestNonEligibleInputsAreNotEligible(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	var chainHeight int32 = 1000
	_, credits := TstCreateCreditsOnNewSeries(t, dbtx, pool, []int64{int64(dustThreshold - 1)})
	c := credits[0]
//确保信用卡足够旧，可以通过minconf支票。
	c.BlockMeta.Height = int32(eligibleInputMinConfirmations)

//检查是否拒绝低于灰尘阈值的信用。
	if pool.isCreditEligible(c, eligibleInputMinConfirmations, chainHeight, dustThreshold) {
		t.Errorf("Input is eligible and it should not be.")
	}

//检查没有足够保兑的信用证是否被拒绝。
	_, credits = TstCreateCreditsOnNewSeries(t, dbtx, pool, []int64{int64(dustThreshold)})
	c = credits[0]
//如果已确认，则计算如下：链高度-bh+
//目标，这很奇怪，但我为什么要放902
//
	c.BlockMeta.Height = int32(902)
	if pool.isCreditEligible(c, eligibleInputMinConfirmations, chainHeight, dustThreshold) {
		t.Errorf("Input is eligible and it should not be.")
	}
}

func TestCreditSortingByAddress(t *testing.T) {
	teardown, db, pool := TstCreatePool(t)
	defer teardown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	series := []TstSeriesDef{
		{ReqSigs: 2, PubKeys: TstPubKeys[1:4], SeriesID: 1},
		{ReqSigs: 2, PubKeys: TstPubKeys[3:6], SeriesID: 2},
	}
	TstCreateSeries(t, dbtx, pool, series)

	shaHash0 := bytes.Repeat([]byte{0}, 32)
	shaHash1 := bytes.Repeat([]byte{1}, 32)
	shaHash2 := bytes.Repeat([]byte{2}, 32)
	c0 := newDummyCredit(t, dbtx, pool, 1, 0, 0, shaHash0, 0)
	c1 := newDummyCredit(t, dbtx, pool, 1, 0, 0, shaHash0, 1)
	c2 := newDummyCredit(t, dbtx, pool, 1, 0, 0, shaHash1, 0)
	c3 := newDummyCredit(t, dbtx, pool, 1, 0, 0, shaHash2, 0)
	c4 := newDummyCredit(t, dbtx, pool, 1, 0, 1, shaHash0, 0)
	c5 := newDummyCredit(t, dbtx, pool, 1, 1, 0, shaHash0, 0)
	c6 := newDummyCredit(t, dbtx, pool, 2, 0, 0, shaHash0, 0)

	randomCredits := [][]credit{
		{c6, c5, c4, c3, c2, c1, c0},
		{c2, c1, c0, c6, c5, c4, c3},
		{c6, c4, c5, c2, c3, c0, c1},
	}

	want := []credit{c0, c1, c2, c3, c4, c5, c6}

	for _, random := range randomCredits {
		sort.Sort(byAddress(random))
		got := random

		if len(got) != len(want) {
			t.Fatalf("Sorted credit slice size wrong: Got: %d, want: %d",
				len(got), len(want))
		}

		for idx := 0; idx < len(want); idx++ {
			if !reflect.DeepEqual(got[idx], want[idx]) {
				t.Errorf("Wrong output index. Got: %v, want: %v",
					got[idx], want[idx])
			}
		}
	}
}

//NewDummyCredit使用给定的哈希和OutpointIdx创建新的信用，
//锁定到由给定的
//系列/索引/分支。
func newDummyCredit(t *testing.T, dbtx walletdb.ReadWriteTx, pool *Pool, series uint32, index Index, branch Branch,
	txHash []byte, outpointIdx uint32) credit {
	var hash chainhash.Hash
	if err := hash.SetBytes(txHash); err != nil {
		t.Fatal(err)
	}
//确保给定序列/分支/索引定义的地址出现在
//作为提取地址要求的已用地址集。
	TstEnsureUsedAddr(t, dbtx, pool, series, branch, index)
	addr := TstNewWithdrawalAddress(t, dbtx, pool, series, branch, index)
	c := wtxmgr.Credit{
		OutPoint: wire.OutPoint{
			Hash:  hash,
			Index: outpointIdx,
		},
	}
	return newCredit(c, *addr)
}

func checkUniqueness(t *testing.T, credits byAddress) {
	type uniq struct {
		series      uint32
		branch      Branch
		index       Index
		hash        chainhash.Hash
		outputIndex uint32
	}

	uniqMap := make(map[uniq]bool)
	for _, c := range credits {
		u := uniq{
			series:      c.addr.SeriesID(),
			branch:      c.addr.Branch(),
			index:       c.addr.Index(),
			hash:        c.OutPoint.Hash,
			outputIndex: c.OutPoint.Index,
		}
		if _, exists := uniqMap[u]; exists {
			t.Fatalf("Duplicate found: %v", u)
		} else {
			uniqMap[u] = true
		}
	}
}

func getPKScriptsForAddressRange(t *testing.T, dbtx walletdb.ReadWriteTx, pool *Pool, seriesID uint32,
	startBranch, stopBranch Branch, startIdx, stopIdx Index) [][]byte {
	var pkScripts [][]byte
	for idx := startIdx; idx <= stopIdx; idx++ {
		for branch := startBranch; branch <= stopBranch; branch++ {
			pkScripts = append(pkScripts, TstCreatePkScript(t, dbtx, pool, seriesID, branch, idx))
		}
	}
	return pkScripts
}

func checkWithdrawalAddressMatches(t *testing.T, addr *WithdrawalAddress, seriesID uint32,
	branch Branch, index Index) {
	if addr.SeriesID() != seriesID {
		t.Fatalf("Wrong seriesID; got %d, want %d", addr.SeriesID(), seriesID)
	}
	if addr.Branch() != branch {
		t.Fatalf("Wrong branch; got %d, want %d", addr.Branch(), branch)
	}
	if addr.Index() != index {
		t.Fatalf("Wrong index; got %d, want %d", addr.Index(), index)
	}
}
