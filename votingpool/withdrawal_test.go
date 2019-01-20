
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

package votingpool_test

import (
	"bytes"
	"testing"

	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	vp "github.com/btcsuite/btcwallet/votingpool"
)

func TestStartWithdrawal(t *testing.T) {
	tearDown, db, pool, store := vp.TstCreatePoolAndTxStore(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, addrmgrNs := vp.TstRWNamespaces(dbtx)
	txmgrNs := vp.TstTxStoreRWNamespace(dbtx)

	mgr := pool.Manager()

	masters := []*hdkeychain.ExtendedKey{
		vp.TstCreateMasterKey(t, bytes.Repeat([]byte{0x00, 0x01}, 16)),
		vp.TstCreateMasterKey(t, bytes.Repeat([]byte{0x02, 0x01}, 16)),
		vp.TstCreateMasterKey(t, bytes.Repeat([]byte{0x03, 0x01}, 16))}
	def := vp.TstCreateSeriesDef(t, pool, 2, masters)
	vp.TstCreateSeries(t, dbtx, pool, []vp.TstSeriesDef{def})
//创建合格的输入和我们需要实现的输出列表。
	vp.TstCreateSeriesCreditsOnStore(t, dbtx, pool, def.SeriesID, []int64{5e6, 4e6}, store)
	address1 := "34eVkREKgvvGASZW7hkgE2uNc1yycntMK6"
	address2 := "3PbExiaztsSYgh6zeMswC49hLUwhTQ86XG"
	requests := []vp.OutputRequest{
		vp.TstNewOutputRequest(t, 1, address1, 4e6, mgr.ChainParams()),
		vp.TstNewOutputRequest(t, 2, address2, 1e6, mgr.ChainParams()),
	}
	changeStart := vp.TstNewChangeAddress(t, pool, def.SeriesID, 0)

	startAddr := vp.TstNewWithdrawalAddress(t, dbtx, pool, def.SeriesID, 0, 0)
	lastSeriesID := def.SeriesID
	dustThreshold := btcutil.Amount(1e4)
	currentBlock := int32(vp.TstInputsBlock + vp.TstEligibleInputMinConfirmations + 1)
	var status *vp.WithdrawalStatus
	vp.TstRunWithManagerUnlocked(t, mgr, addrmgrNs, func() {
		status, err = pool.StartWithdrawal(ns, addrmgrNs, 0, requests, *startAddr, lastSeriesID, *changeStart,
			store, txmgrNs, currentBlock, dustThreshold)
	})
	if err != nil {
		t.Fatal(err)
	}

//检查所有输出是否成功完成。
	checkWithdrawalOutputs(t, status, map[string]btcutil.Amount{address1: 4e6, address2: 1e6})

	if status.Fees() != btcutil.Amount(1e3) {
		t.Fatalf("Wrong amount for fees; got %v, want %v", status.Fees(), btcutil.Amount(1e3))
	}

//这种提款只产生了一个变化的单一交易
//输出，因此下一个更改地址将与
//索引增加了1。
	nextChangeAddr := status.NextChangeAddr()
	if nextChangeAddr.SeriesID() != changeStart.SeriesID() {
		t.Fatalf("Wrong nextChangeStart series; got %d, want %d", nextChangeAddr.SeriesID(),
			changeStart.SeriesID())
	}
	if nextChangeAddr.Index() != changeStart.Index()+1 {
		t.Fatalf("Wrong nextChangeStart index; got %d, want %d", nextChangeAddr.Index(),
			changeStart.Index()+1)
	}

//注意：ntxid是确定性的，所以我们在这里硬编码它，但是如果测试
//或者代码的更改方式导致生成的事务
//变化（例如不同的输入/输出），NTXID也会变化，并且
//这将需要更新。
	ntxid := vp.Ntxid("eb753083db55bd0ad2eb184bfd196a7ea8b90eaa000d9293e892999695af2519")
	txSigs := status.Sigs()[ntxid]

//最后，我们使用signtx（）构造signaturescripts（使用raw
//签名）必须解锁经理，因为签名涉及查找
//赎回加密存储的脚本。
	msgtx := status.TstGetMsgTx(ntxid)
	vp.TstRunWithManagerUnlocked(t, mgr, addrmgrNs, func() {
		if err = vp.SignTx(msgtx, txSigs, mgr, addrmgrNs, store, txmgrNs); err != nil {
			t.Fatal(err)
		}
	})

//使用相同参数的任何后续startRetraction（）调用都将
//返回先前存储的取款状态。
	var status2 *vp.WithdrawalStatus
	vp.TstRunWithManagerUnlocked(t, mgr, addrmgrNs, func() {
		status2, err = pool.StartWithdrawal(ns, addrmgrNs, 0, requests, *startAddr, lastSeriesID, *changeStart,
			store, txmgrNs, currentBlock, dustThreshold)
	})
	if err != nil {
		t.Fatal(err)
	}
	vp.TstCheckWithdrawalStatusMatches(t, *status, *status2)
}

func checkWithdrawalOutputs(
	t *testing.T, wStatus *vp.WithdrawalStatus, amounts map[string]btcutil.Amount) {
	fulfilled := wStatus.Outputs()
	if len(fulfilled) != 2 {
		t.Fatalf("Unexpected number of outputs in WithdrawalStatus; got %d, want %d",
			len(fulfilled), 2)
	}
	for _, output := range fulfilled {
		addr := output.Address()
		amount, ok := amounts[addr]
		if !ok {
			t.Fatalf("Unexpected output addr: %s", addr)
		}

		status := output.Status()
		if status != "success" {
			t.Fatalf(
				"Unexpected status for output %v; got '%s', want 'success'", output, status)
		}

		outpoints := output.Outpoints()
		if len(outpoints) != 1 {
			t.Fatalf(
				"Unexpected number of outpoints for output %v; got %d, want 1", output,
				len(outpoints))
		}

		gotAmount := outpoints[0].Amount()
		if gotAmount != amount {
			t.Fatalf("Unexpected amount for output %v; got %v, want %v", output, gotAmount, amount)
		}
	}
}
