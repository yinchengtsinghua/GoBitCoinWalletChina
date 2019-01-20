
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

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/btcsuite/btcwallet/wtxmgr"
)

//TestOutputsSplitingNoteNoughinputs检查如果我们
//没有足够的投入来完成它。
func TestOutputSplittingNotEnoughInputs(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	net := pool.Manager().ChainParams()
	output1Amount := btcutil.Amount(2)
	output2Amount := btcutil.Amount(3)
	requests := []OutputRequest{
//这些输出请求将具有相同的服务器ID，因此我们知道
//它们将按照这里定义的顺序完成，即
//这项测试很重要。
		TstNewOutputRequest(t, 1, "34eVkREKgvvGASZW7hkgE2uNc1yycntMK6", output1Amount, net),
		TstNewOutputRequest(t, 2, "34eVkREKgvvGASZW7hkgE2uNc1yycntMK6", output2Amount, net),
	}
	seriesID, eligible := TstCreateCreditsOnNewSeries(t, dbtx, pool, []int64{7})
	w := newWithdrawal(0, requests, eligible, *TstNewChangeAddress(t, pool, seriesID, 0))
	w.txOptions = func(tx *withdrawalTx) {
//由于缺少输入，通过强制高收费触发输出分割。
//如果我们刚开始时没有足够的输入用于请求的输出，
//fulfillRequests（）将删除输出，直到我们有足够的输出。
		tx.calculateFee = TstConstantFee(3)
	}

	if err := w.fulfillRequests(); err != nil {
		t.Fatal(err)
	}

	if len(w.transactions) != 1 {
		t.Fatalf("Wrong number of finalized transactions; got %d, want 1", len(w.transactions))
	}

	tx := w.transactions[0]
	if len(tx.outputs) != 2 {
		t.Fatalf("Wrong number of outputs; got %d, want 2", len(tx.outputs))
	}

//第一个输出应该保持原样。
	if tx.outputs[0].amount != output1Amount {
		t.Fatalf("Wrong amount for first tx output; got %v, want %v",
			tx.outputs[0].amount, output1Amount)
	}

//最后一个输出应该将其数量更新为我们拥有的
//在满足所有以前的输出之后离开。
	newAmount := tx.inputTotal() - output1Amount - tx.calculateFee()
	checkLastOutputWasSplit(t, w, tx, output2Amount, newAmount)
}

func TestOutputSplittingOversizeTx(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	requestAmount := btcutil.Amount(5)
	bigInput := int64(3)
	smallInput := int64(2)
	request := TstNewOutputRequest(
		t, 1, "34eVkREKgvvGASZW7hkgE2uNc1yycntMK6", requestAmount, pool.Manager().ChainParams())
	seriesID, eligible := TstCreateCreditsOnNewSeries(t, dbtx, pool, []int64{smallInput, bigInput})
	changeStart := TstNewChangeAddress(t, pool, seriesID, 0)
	w := newWithdrawal(0, []OutputRequest{request}, eligible, *changeStart)
	w.txOptions = func(tx *withdrawalTx) {
		tx.calculateFee = TstConstantFee(0)
		tx.calculateSize = func() int {
//在添加第二个输入后立即触发输出拆分。
			if len(tx.inputs) == 2 {
				return txMaxSize + 1
			}
			return txMaxSize - 1
		}
	}

	if err := w.fulfillRequests(); err != nil {
		t.Fatal(err)
	}

	if len(w.transactions) != 2 {
		t.Fatalf("Wrong number of finalized transactions; got %d, want 2", len(w.transactions))
	}

	tx1 := w.transactions[0]
	if len(tx1.outputs) != 1 {
		t.Fatalf("Wrong number of outputs on tx1; got %d, want 1", len(tx1.outputs))
	}
	if tx1.outputs[0].amount != btcutil.Amount(bigInput) {
		t.Fatalf("Wrong amount for output in tx1; got %d, want %d", tx1.outputs[0].amount,
			bigInput)
	}

	tx2 := w.transactions[1]
	if len(tx2.outputs) != 1 {
		t.Fatalf("Wrong number of outputs on tx2; got %d, want 1", len(tx2.outputs))
	}
	if tx2.outputs[0].amount != btcutil.Amount(smallInput) {
		t.Fatalf("Wrong amount for output in tx2; got %d, want %d", tx2.outputs[0].amount,
			smallInput)
	}

	if len(w.status.outputs) != 1 {
		t.Fatalf("Wrong number of output statuses; got %d, want 1", len(w.status.outputs))
	}
	status := w.status.outputs[request.outBailmentID()].status
	if status != statusSplit {
		t.Fatalf("Wrong output status; got '%s', want '%s'", status, statusSplit)
	}
}

func TestSplitLastOutputNoOutputs(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	w := newWithdrawal(0, []OutputRequest{}, []credit{}, ChangeAddress{})
	w.current = createWithdrawalTx(t, dbtx, pool, []int64{}, []int64{})

	err = w.splitLastOutput()

	TstCheckError(t, "", err, ErrPreconditionNotMet)
}

//检查撤销请求的所有输出是否与生成的输出匹配。
//交易（s）。
func TestWithdrawalTxOutputs(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	net := pool.Manager().ChainParams()

//创建合格的输入和我们需要实现的输出列表。
	seriesID, eligible := TstCreateCreditsOnNewSeries(t, dbtx, pool, []int64{2e6, 4e6})
	outputs := []OutputRequest{
		TstNewOutputRequest(t, 1, "34eVkREKgvvGASZW7hkgE2uNc1yycntMK6", 3e6, net),
		TstNewOutputRequest(t, 2, "3PbExiaztsSYgh6zeMswC49hLUwhTQ86XG", 2e6, net),
	}
	changeStart := TstNewChangeAddress(t, pool, seriesID, 0)

	w := newWithdrawal(0, outputs, eligible, *changeStart)
	if err := w.fulfillRequests(); err != nil {
		t.Fatal(err)
	}

	if len(w.transactions) != 1 {
		t.Fatalf("Unexpected number of transactions; got %d, want 1", len(w.transactions))
	}

	tx := w.transactions[0]
//创建的Tx应包括两个符合条件的信用，因此我们希望它
//输入量2e6+4e6 Satoshis。
	inputAmount := eligible[0].Amount + eligible[1].Amount
	change := inputAmount - (outputs[0].Amount + outputs[1].Amount + tx.calculateFee())
	expectedOutputs := append(
		outputs, TstNewOutputRequest(t, 3, changeStart.addr.String(), change, net))
	msgtx := tx.toMsgTx()
	checkMsgTxOutputs(t, msgtx, expectedOutputs)
}

//检查drawing.status是否正确地说明当我们
//
func TestFulfillRequestsNoSatisfiableOutputs(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	seriesID, eligible := TstCreateCreditsOnNewSeries(t, dbtx, pool, []int64{1e6})
	request := TstNewOutputRequest(
		t, 1, "3Qt1EaKRD9g9FeL2DGkLLswhK1AKmmXFSe", btcutil.Amount(3e6), pool.Manager().ChainParams())
	changeStart := TstNewChangeAddress(t, pool, seriesID, 0)

	w := newWithdrawal(0, []OutputRequest{request}, eligible, *changeStart)
	if err := w.fulfillRequests(); err != nil {
		t.Fatal(err)
	}

	if len(w.transactions) != 0 {
		t.Fatalf("Unexpected number of transactions; got %d, want 0", len(w.transactions))
	}

	if len(w.status.outputs) != 1 {
		t.Fatalf("Unexpected number of outputs in WithdrawalStatus; got %d, want 1",
			len(w.status.outputs))
	}

	status := w.status.outputs[request.outBailmentID()].status
	if status != statusPartial {
		t.Fatalf("Unexpected status for requested outputs; got '%s', want '%s'",
			status, statusPartial)
	}
}

//当我们没有所有的学分时，检查一些要求的输出是否没有完成。
//他们当中。
func TestFulfillRequestsNotEnoughCreditsForAllRequests(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	net := pool.Manager().ChainParams()

//创建合格的输入和我们需要实现的输出列表。
	seriesID, eligible := TstCreateCreditsOnNewSeries(t, dbtx, pool, []int64{2e6, 4e6})
	out1 := TstNewOutputRequest(
		t, 1, "34eVkREKgvvGASZW7hkgE2uNc1yycntMK6", btcutil.Amount(3e6), net)
	out2 := TstNewOutputRequest(
		t, 2, "3PbExiaztsSYgh6zeMswC49hLUwhTQ86XG", btcutil.Amount(2e6), net)
	out3 := TstNewOutputRequest(
		t, 3, "3Qt1EaKRD9g9FeL2DGkLLswhK1AKmmXFSe", btcutil.Amount(5e6), net)
	outputs := []OutputRequest{out1, out2, out3}
	changeStart := TstNewChangeAddress(t, pool, seriesID, 0)

	w := newWithdrawal(0, outputs, eligible, *changeStart)
	if err := w.fulfillRequests(); err != nil {
		t.Fatal(err)
	}

	tx := w.transactions[0]
//创建的Tx应该同时使用两个符合条件的信贷，因此我们希望它
//输入量2e6+4e6 Satoshis。
	inputAmount := eligible[0].Amount + eligible[1].Amount
//我们希望它包括请求1和2的输出，加上一个变更输出，但是
//输出请求3不应该存在，因为我们没有足够的学分。
	change := inputAmount - (out1.Amount + out2.Amount + tx.calculateFee())
	expectedOutputs := []OutputRequest{out1, out2}
	sort.Sort(byOutBailmentID(expectedOutputs))
	expectedOutputs = append(
		expectedOutputs, TstNewOutputRequest(t, 4, changeStart.addr.String(), change, net))
	msgtx := tx.toMsgTx()
	checkMsgTxOutputs(t, msgtx, expectedOutputs)

//撤回状态应说明成功完成了输出1和输出2，
//输出3不是。
	expectedStatuses := map[OutBailmentID]outputStatus{
		out1.outBailmentID(): statusSuccess,
		out2.outBailmentID(): statusSuccess,
		out3.outBailmentID(): statusPartial}
	for _, wOutput := range w.status.outputs {
		if wOutput.status != expectedStatuses[wOutput.request.outBailmentID()] {
			t.Fatalf("Unexpected status for %v; got '%s', want '%s'", wOutput.request,
				wOutput.status, expectedStatuses[wOutput.request.outBailmentID()])
		}
	}
}

//testrollbacklastoutput测试回滚一个输出的情况
//和一个输入，这样总和（in）>=总和（out）+费用。
func TestRollbackLastOutput(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	tx := createWithdrawalTx(t, dbtx, pool, []int64{3, 3, 2, 1, 3}, []int64{3, 3, 2, 2})
	initialInputs := tx.inputs
	initialOutputs := tx.outputs

	tx.calculateFee = TstConstantFee(1)
	removedInputs, removedOutput, err := tx.rollBackLastOutput()
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}

//上面的rollbacklastoutput（）调用应已删除最后一个输出
//最后一个输入。
	lastOutput := initialOutputs[len(initialOutputs)-1]
	if removedOutput != lastOutput {
		t.Fatalf("Wrong rolled back output; got %s want %s", removedOutput, lastOutput)
	}
	if len(removedInputs) != 1 {
		t.Fatalf("Unexpected number of inputs removed; got %d, want 1", len(removedInputs))
	}
	lastInput := initialInputs[len(initialInputs)-1]
	if !reflect.DeepEqual(removedInputs[0], lastInput) {
		t.Fatalf("Wrong rolled back input; got %v want %v", removedInputs[0], lastInput)
	}

//现在检查Tx中的输入和输出是否与我们
//期待。
	checkTxOutputs(t, tx, initialOutputs[:len(initialOutputs)-1])
	checkTxInputs(t, tx, initialInputs[:len(initialInputs)-1])
}

func TestRollbackLastOutputMultipleInputsRolledBack(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

//这个Tx需要最后3个输入来完成第二个输出，所以它们
//如果全部回滚并按与添加相反的顺序返回。
	tx := createWithdrawalTx(t, dbtx, pool, []int64{1, 2, 3, 4}, []int64{1, 8})
	initialInputs := tx.inputs
	initialOutputs := tx.outputs

	tx.calculateFee = TstConstantFee(0)
	removedInputs, _, err := tx.rollBackLastOutput()
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}

	if len(removedInputs) != 3 {
		t.Fatalf("Unexpected number of inputs removed; got %d, want 3", len(removedInputs))
	}
	for i, amount := range []btcutil.Amount{4, 3, 2} {
		if removedInputs[i].Amount != amount {
			t.Fatalf("Unexpected input amount; got %v, want %v", removedInputs[i].Amount, amount)
		}
	}

//现在检查Tx中的输入和输出是否与我们
//期待。
	checkTxOutputs(t, tx, initialOutputs[:len(initialOutputs)-1])
	checkTxInputs(t, tx, initialInputs[:len(initialInputs)-len(removedInputs)])
}

//
//一个输出，但不需要回滚任何输入。
func TestRollbackLastOutputNoInputsRolledBack(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	tx := createWithdrawalTx(t, dbtx, pool, []int64{4}, []int64{2, 3})
	initialInputs := tx.inputs
	initialOutputs := tx.outputs

	tx.calculateFee = TstConstantFee(1)
	removedInputs, removedOutput, err := tx.rollBackLastOutput()
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}

//上面的rollbacklastoutput（）调用应已移除
//最后一个输出，但没有输入。
	lastOutput := initialOutputs[len(initialOutputs)-1]
	if removedOutput != lastOutput {
		t.Fatalf("Wrong output; got %s want %s", removedOutput, lastOutput)
	}
	if len(removedInputs) != 0 {
		t.Fatalf("Expected no removed inputs, but got %d inputs", len(removedInputs))
	}

//现在检查Tx中的输入和输出是否与我们
//期待。
	checkTxOutputs(t, tx, initialOutputs[:len(initialOutputs)-1])
	checkTxInputs(t, tx, initialInputs)
}

//TestRollbackLastOutputinesountientOutputs检查
//RollbackLastOutput如果少于两个，则返回错误
//事务中的输出。
func TestRollBackLastOutputInsufficientOutputs(t *testing.T) {
	tx := newWithdrawalTx(defaultTxOptions)
	_, _, err := tx.rollBackLastOutput()
	TstCheckError(t, "", err, ErrPreconditionNotMet)

	output := &WithdrawalOutput{request: TstNewOutputRequest(
		t, 1, "34eVkREKgvvGASZW7hkgE2uNc1yycntMK6", btcutil.Amount(3), &chaincfg.MainNetParams)}
	tx.addOutput(output.request)
	_, _, err = tx.rollBackLastOutput()
	TstCheckError(t, "", err, ErrPreconditionNotMet)
}

//TestRollbackLastOutputWhennewOutput添加了检查，以便回滚最后一个
//如果一个Tx在我们添加一个新的输出之后变得太大，则输出。
func TestRollbackLastOutputWhenNewOutputAdded(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	net := pool.Manager().ChainParams()
	series, eligible := TstCreateCreditsOnNewSeries(t, dbtx, pool, []int64{5, 5})
	requests := []OutputRequest{
//这是按寄托ID订购的
		TstNewOutputRequest(t, 1, "34eVkREKgvvGASZW7hkgE2uNc1yycntMK6", 1, net),
		TstNewOutputRequest(t, 2, "3PbExiaztsSYgh6zeMswC49hLUwhTQ86XG", 2, net),
	}
	changeStart := TstNewChangeAddress(t, pool, series, 0)

	w := newWithdrawal(0, requests, eligible, *changeStart)
	w.txOptions = func(tx *withdrawalTx) {
		tx.calculateFee = TstConstantFee(0)
		tx.calculateSize = func() int {
//在添加第二个输出后立即触发输出拆分。
			if len(tx.outputs) > 1 {
				return txMaxSize + 1
			}
			return txMaxSize - 1
		}
	}

	if err := w.fulfillRequests(); err != nil {
		t.Fatal("Unexpected error:", err)
	}

//此时，我们应该有两个最终的交易。
	if len(w.transactions) != 2 {
		t.Fatalf("Wrong number of finalized transactions; got %d, want 2", len(w.transactions))
	}

//第一个Tx应具有一个输出（1）和一个更改输出（4）
//萨索斯
	firstTx := w.transactions[0]
	req1 := requests[0]
	checkTxOutputs(t, firstTx,
		[]*withdrawalTxOut{{request: req1, amount: req1.Amount}})
	checkTxChangeAmount(t, firstTx, btcutil.Amount(4))

//第二个Tx应具有一个2输出和一个3 Satoshis变化输出。
	secondTx := w.transactions[1]
	req2 := requests[1]
	checkTxOutputs(t, secondTx,
		[]*withdrawalTxOut{{request: req2, amount: req2.Amount}})
	checkTxChangeAmount(t, secondTx, btcutil.Amount(3))
}

//TestRollbackLastOutputWhennewinput添加了检查以回滚最后一个
//如果一个Tx在我们添加一个新的输入之后变得太大，则输出。
func TestRollbackLastOutputWhenNewInputAdded(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	net := pool.Manager().ChainParams()
	series, eligible := TstCreateCreditsOnNewSeries(t, dbtx, pool, []int64{6, 5, 4, 3, 2, 1})
	requests := []OutputRequest{
//这是由OutbailmentIdHash手动订购的，它是
//它们将由w.fulfillRequests（）完成。
		TstNewOutputRequest(t, 1, "34eVkREKgvvGASZW7hkgE2uNc1yycntMK6", 1, net),
		TstNewOutputRequest(t, 3, "3Qt1EaKRD9g9FeL2DGkLLswhK1AKmmXFSe", 6, net),
		TstNewOutputRequest(t, 2, "3PbExiaztsSYgh6zeMswC49hLUwhTQ86XG", 3, net),
	}
	changeStart := TstNewChangeAddress(t, pool, series, 0)

	w := newWithdrawal(0, requests, eligible, *changeStart)
	w.txOptions = func(tx *withdrawalTx) {
		tx.calculateFee = TstConstantFee(0)
		tx.calculateSize = func() int {
//一旦向事务中添加第四个输入，就使事务过大。
			if len(tx.inputs) > 3 {
				return txMaxSize + 1
			}
			return txMaxSize - 1
		}
	}

//在中添加第四个输入后，应立即触发回滚。
//以满足第二个请求。
	if err := w.fulfillRequests(); err != nil {
		t.Fatal("Unexpected error:", err)
	}

//此时，我们应该有两个最终的交易。
	if len(w.transactions) != 2 {
		t.Fatalf("Wrong number of finalized transactions; got %d, want 2", len(w.transactions))
	}

//第一个Tx应该有一个输出，输出量为1，第一个输入来自
//合格输入（最后一个切片项）的堆栈，没有更改输出。
	firstTx := w.transactions[0]
	req1 := requests[0]
	checkTxOutputs(t, firstTx,
		[]*withdrawalTxOut{{request: req1, amount: req1.Amount}})
	checkTxInputs(t, firstTx, eligible[5:6])

//第二个Tx应具有最后两个请求的输出（相同
//命令将它们传递给newretraction），并且需要3个输入
//实现这一点（按相反的顺序，当它们被传递给newdrawing时，
//这就是fulfillRequests（）如何使用它们），并且没有更改输出。
	secondTx := w.transactions[1]
	wantOutputs := []*withdrawalTxOut{
		{request: requests[1], amount: requests[1].Amount},
		{request: requests[2], amount: requests[2].Amount}}
	checkTxOutputs(t, secondTx, wantOutputs)
	wantInputs := []credit{eligible[4], eligible[3], eligible[2]}
	checkTxInputs(t, secondTx, wantInputs)
}

func TestWithdrawalTxRemoveOutput(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	tx := createWithdrawalTx(t, dbtx, pool, []int64{}, []int64{1, 2})
	outputs := tx.outputs
//确保我们已使用预期的
//输出。
	checkTxOutputs(t, tx, outputs)

	remainingOutput := tx.outputs[0]
	wantRemovedOutput := tx.outputs[1]

	gotRemovedOutput := tx.removeOutput()

//检查弹出的输出是否正确。
	if gotRemovedOutput != wantRemovedOutput {
		t.Fatalf("Removed output wrong; got %v, want %v", gotRemovedOutput, wantRemovedOutput)
	}
//剩余的输出是正确的。
	checkTxOutputs(t, tx, []*withdrawalTxOut{remainingOutput})

//确保剩余的输出确实是正确的。
	if tx.outputs[0] != remainingOutput {
		t.Fatalf("Wrong output: got %v, want %v", tx.outputs[0], remainingOutput)
	}
}

func TestWithdrawalTxRemoveInput(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	tx := createWithdrawalTx(t, dbtx, pool, []int64{1, 2}, []int64{})
	inputs := tx.inputs
//确保我们已经创建了具有预期输入的事务
	checkTxInputs(t, tx, inputs)

	remainingInput := tx.inputs[0]
	wantRemovedInput := tx.inputs[1]

	gotRemovedInput := tx.removeInput()

//检查弹出的输入是否正确。
	if !reflect.DeepEqual(gotRemovedInput, wantRemovedInput) {
		t.Fatalf("Popped input wrong; got %v, want %v", gotRemovedInput, wantRemovedInput)
	}
	checkTxInputs(t, tx, inputs[0:1])

//确保剩余的输入确实是正确的。
	if !reflect.DeepEqual(tx.inputs[0], remainingInput) {
		t.Fatalf("Wrong input: got %v, want %v", tx.inputs[0], remainingInput)
	}
}

func TestWithdrawalTxAddChange(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	input, output, fee := int64(4e6), int64(3e6), int64(10)
	tx := createWithdrawalTx(t, dbtx, pool, []int64{input}, []int64{output})
	tx.calculateFee = TstConstantFee(btcutil.Amount(fee))

	if !tx.addChange([]byte{}) {
		t.Fatal("tx.addChange() returned false, meaning it did not add a change output")
	}

	msgtx := tx.toMsgTx()
	if len(msgtx.TxOut) != 2 {
		t.Fatalf("Unexpected number of txouts; got %d, want 2", len(msgtx.TxOut))
	}
	gotChange := msgtx.TxOut[1].Value
	wantChange := input - output - fee
	if gotChange != wantChange {
		t.Fatalf("Unexpected change amount; got %v, want %v", gotChange, wantChange)
	}
}

//TestDrawAltxaddChangeNochange检查DrawAltx.AddChange（）是否不
//支付完所有费用后，如果没有剩余的Satoshis，则添加一个更改输出
//输出+费用。
func TestWithdrawalTxAddChangeNoChange(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	input, output, fee := int64(4e6), int64(4e6), int64(0)
	tx := createWithdrawalTx(t, dbtx, pool, []int64{input}, []int64{output})
	tx.calculateFee = TstConstantFee(btcutil.Amount(fee))

	if tx.addChange([]byte{}) {
		t.Fatal("tx.addChange() returned true, meaning it added a change output")
	}
	msgtx := tx.toMsgTx()
	if len(msgtx.TxOut) != 1 {
		t.Fatalf("Unexpected number of txouts; got %d, want 1", len(msgtx.TxOut))
	}
}

func TestWithdrawalTxToMsgTxNoInputsOrOutputsOrChange(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	tx := createWithdrawalTx(t, dbtx, pool, []int64{}, []int64{})
	msgtx := tx.toMsgTx()
	compareMsgTxAndWithdrawalTxOutputs(t, msgtx, tx)
	compareMsgTxAndWithdrawalTxInputs(t, msgtx, tx)
}

func TestWithdrawalTxToMsgTxNoInputsOrOutputsWithChange(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	tx := createWithdrawalTx(t, dbtx, pool, []int64{}, []int64{})
	tx.changeOutput = wire.NewTxOut(int64(1), []byte{})

	msgtx := tx.toMsgTx()

	compareMsgTxAndWithdrawalTxOutputs(t, msgtx, tx)
	compareMsgTxAndWithdrawalTxInputs(t, msgtx, tx)
}

func TestWithdrawalTxToMsgTxWithInputButNoOutputsWithChange(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	tx := createWithdrawalTx(t, dbtx, pool, []int64{1}, []int64{})
	tx.changeOutput = wire.NewTxOut(int64(1), []byte{})

	msgtx := tx.toMsgTx()

	compareMsgTxAndWithdrawalTxOutputs(t, msgtx, tx)
	compareMsgTxAndWithdrawalTxInputs(t, msgtx, tx)
}

func TestWithdrawalTxToMsgTxWithInputOutputsAndChange(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	tx := createWithdrawalTx(t, dbtx, pool, []int64{1, 2, 3}, []int64{4, 5, 6})
	tx.changeOutput = wire.NewTxOut(int64(7), []byte{})

	msgtx := tx.toMsgTx()

	compareMsgTxAndWithdrawalTxOutputs(t, msgtx, tx)
	compareMsgTxAndWithdrawalTxInputs(t, msgtx, tx)
}

func TestWithdrawalTxInputTotal(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	tx := createWithdrawalTx(t, dbtx, pool, []int64{5}, []int64{})

	if tx.inputTotal() != btcutil.Amount(5) {
		t.Fatalf("Wrong total output; got %v, want %v", tx.outputTotal(), btcutil.Amount(5))
	}
}

func TestWithdrawalTxOutputTotal(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	tx := createWithdrawalTx(t, dbtx, pool, []int64{}, []int64{4})
	tx.changeOutput = wire.NewTxOut(int64(1), []byte{})

	if tx.outputTotal() != btcutil.Amount(4) {
		t.Fatalf("Wrong total output; got %v, want %v", tx.outputTotal(), btcutil.Amount(4))
	}
}

func TestWithdrawalInfoMatch(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	roundID := uint32(0)
	wi := createAndFulfillWithdrawalRequests(t, dbtx, pool, roundID)

//为请求、StartAddress和ChangeStart使用新创建的值
//以模拟如果我们从
//数据库中的序列化数据。
	requestsCopy := make([]OutputRequest, len(wi.requests))
	copy(requestsCopy, wi.requests)
	startAddr := TstNewWithdrawalAddress(t, dbtx, pool, wi.startAddress.seriesID, wi.startAddress.branch,
		wi.startAddress.index)
	changeStart := TstNewChangeAddress(t, pool, wi.changeStart.seriesID, wi.changeStart.index)

//首先检查所有字段相同时是否匹配。
	matches := wi.match(requestsCopy, *startAddr, wi.lastSeriesID, *changeStart, wi.dustThreshold)
	if !matches {
		t.Fatal("Should match when everything is identical.")
	}

//如果输出请求的顺序不同，它也会匹配。
	diffOrderRequests := make([]OutputRequest, len(requestsCopy))
	copy(diffOrderRequests, requestsCopy)
	diffOrderRequests[0], diffOrderRequests[1] = requestsCopy[1], requestsCopy[0]
	matches = wi.match(diffOrderRequests, *startAddr, wi.lastSeriesID, *changeStart,
		wi.dustThreshold)
	if !matches {
		t.Fatal("Should match when requests are in different order.")
	}

//当输出请求不同时，它不应该匹配。
	diffRequests := diffOrderRequests
	diffRequests[0] = OutputRequest{}
	matches = wi.match(diffRequests, *startAddr, wi.lastSeriesID, *changeStart, wi.dustThreshold)
	if matches {
		t.Fatal("Should not match as requests is not equal.")
	}

//当LastSeriesID不相等时，它不应匹配。
	matches = wi.match(requestsCopy, *startAddr, wi.lastSeriesID+1, *changeStart, wi.dustThreshold)
	if matches {
		t.Fatal("Should not match as lastSeriesID is not equal.")
	}

//当dustThreshold不相等时，不应匹配。
	matches = wi.match(requestsCopy, *startAddr, wi.lastSeriesID, *changeStart, wi.dustThreshold+1)
	if matches {
		t.Fatal("Should not match as dustThreshold is not equal.")
	}

//当startaddress不相等时，它不应匹配。
	diffStartAddr := TstNewWithdrawalAddress(t, dbtx, pool, startAddr.seriesID, startAddr.branch+1,
		startAddr.index)
	matches = wi.match(requestsCopy, *diffStartAddr, wi.lastSeriesID, *changeStart,
		wi.dustThreshold)
	if matches {
		t.Fatal("Should not match as startAddress is not equal.")
	}

//当changestart不相等时，它不应匹配。
	diffChangeStart := TstNewChangeAddress(t, pool, changeStart.seriesID, changeStart.index+1)
	matches = wi.match(requestsCopy, *startAddr, wi.lastSeriesID, *diffChangeStart,
		wi.dustThreshold)
	if matches {
		t.Fatal("Should not match as changeStart is not equal.")
	}
}

func TestGetWithdrawalStatus(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, addrmgrNs := TstRWNamespaces(dbtx)

	roundID := uint32(0)
	wi := createAndFulfillWithdrawalRequests(t, dbtx, pool, roundID)

	serialized, err := serializeWithdrawal(wi.requests, wi.startAddress, wi.lastSeriesID,
		wi.changeStart, wi.dustThreshold, wi.status)
	if err != nil {
		t.Fatal(err)
	}
	err = putWithdrawal(ns, pool.ID, roundID, serialized)
	if err != nil {
		t.Fatal(err)
	}

//在这里我们应该得到一个与wi.status匹配的取款状态。
	var status *WithdrawalStatus
	TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		status, err = getWithdrawalStatus(pool, ns, addrmgrNs, roundID, wi.requests, wi.startAddress,
			wi.lastSeriesID, wi.changeStart, wi.dustThreshold)
	})
	if err != nil {
		t.Fatal(err)
	}
	TstCheckWithdrawalStatusMatches(t, wi.status, *status)

//这里我们应该得到一个零的取款状态，因为参数不是
//与存储的具有此roundID的drawintalstatus相同。
	dustThreshold := wi.dustThreshold + 1
	TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		status, err = getWithdrawalStatus(pool, ns, addrmgrNs, roundID, wi.requests, wi.startAddress,
			wi.lastSeriesID, wi.changeStart, dustThreshold)
	})
	if err != nil {
		t.Fatal(err)
	}
	if status != nil {
		t.Fatalf("Expected a nil status, got %v", status)
	}
}

func TestSignMultiSigUTXO(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	_, addrmgrNs := TstRWNamespaces(dbtx)

//用我们要签名的单个输入创建一个新的Tx。
	mgr := pool.Manager()
	tx := createWithdrawalTx(t, dbtx, pool, []int64{4e6}, []int64{4e6})
	sigs, err := getRawSigs([]*withdrawalTx{tx})
	if err != nil {
		t.Fatal(err)
	}

	msgtx := tx.toMsgTx()
	txSigs := sigs[tx.ntxid()]

idx := 0 //我们要签名的Tx输入的索引。
	pkScript := tx.inputs[idx].PkScript
	TstRunWithManagerUnlocked(t, mgr, addrmgrNs, func() {
		if err = signMultiSigUTXO(mgr, addrmgrNs, msgtx, idx, pkScript, txSigs[idx]); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSignMultiSigUTXOUnparseablePkScript(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	_, addrmgrNs := TstRWNamespaces(dbtx)

	mgr := pool.Manager()
	tx := createWithdrawalTx(t, dbtx, pool, []int64{4e6}, []int64{})
	msgtx := tx.toMsgTx()

	unparseablePkScript := []byte{0x01}
	err = signMultiSigUTXO(mgr, addrmgrNs, msgtx, 0, unparseablePkScript, []RawSig{{}})

	TstCheckError(t, "", err, ErrTxSigning)
}

func TestSignMultiSigUTXOPkScriptNotP2SH(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	_, addrmgrNs := TstRWNamespaces(dbtx)

	mgr := pool.Manager()
	tx := createWithdrawalTx(t, dbtx, pool, []int64{4e6}, []int64{})
	addr, _ := btcutil.DecodeAddress("1MirQ9bwyQcGVJPwKUgapu5ouK2E2Ey4gX", mgr.ChainParams())
	pubKeyHashPkScript, _ := txscript.PayToAddrScript(addr.(*btcutil.AddressPubKeyHash))
	msgtx := tx.toMsgTx()

	err = signMultiSigUTXO(mgr, addrmgrNs, msgtx, 0, pubKeyHashPkScript, []RawSig{{}})

	TstCheckError(t, "", err, ErrTxSigning)
}

func TestSignMultiSigUTXORedeemScriptNotFound(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	_, addrmgrNs := TstRWNamespaces(dbtx)

	mgr := pool.Manager()
	tx := createWithdrawalTx(t, dbtx, pool, []int64{4e6}, []int64{})
//这是地址管理器没有兑现的p2sh地址
//脚本。
	addr, _ := btcutil.DecodeAddress("3Hb4xcebcKg4DiETJfwjh8sF4uDw9rqtVC", mgr.ChainParams())
	if _, err := mgr.Address(addrmgrNs, addr); err == nil {
		t.Fatalf("Address %s found in manager when it shouldn't", addr)
	}
	msgtx := tx.toMsgTx()

	pkScript, _ := txscript.PayToAddrScript(addr.(*btcutil.AddressScriptHash))
	err = signMultiSigUTXO(mgr, addrmgrNs, msgtx, 0, pkScript, []RawSig{{}})

	TstCheckError(t, "", err, ErrTxSigning)
}

func TestSignMultiSigUTXONotEnoughSigs(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	_, addrmgrNs := TstRWNamespaces(dbtx)

	mgr := pool.Manager()
	tx := createWithdrawalTx(t, dbtx, pool, []int64{4e6}, []int64{})
	sigs, err := getRawSigs([]*withdrawalTx{tx})
	if err != nil {
		t.Fatal(err)
	}
	msgtx := tx.toMsgTx()
	txSigs := sigs[tx.ntxid()]

idx := 0 //我们要签名的Tx输入的索引。
//这里我们提供reqsigs-1签名给signmultisigtxo（）。
	reqSigs := tx.inputs[idx].addr.series().TstGetReqSigs()
	txInSigs := txSigs[idx][:reqSigs-1]
	pkScript := tx.inputs[idx].PkScript
	TstRunWithManagerUnlocked(t, mgr, addrmgrNs, func() {
		err = signMultiSigUTXO(mgr, addrmgrNs, msgtx, idx, pkScript, txInSigs)
	})

	TstCheckError(t, "", err, ErrTxSigning)
}

func TestSignMultiSigUTXOWrongRawSigs(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	_, addrmgrNs := TstRWNamespaces(dbtx)

	mgr := pool.Manager()
	tx := createWithdrawalTx(t, dbtx, pool, []int64{4e6}, []int64{})
	sigs := []RawSig{{0x00}, {0x01}}

idx := 0 //我们要签名的Tx输入的索引。
	pkScript := tx.inputs[idx].PkScript
	TstRunWithManagerUnlocked(t, mgr, addrmgrNs, func() {
		err = signMultiSigUTXO(mgr, addrmgrNs, tx.toMsgTx(), idx, pkScript, sigs)
	})

	TstCheckError(t, "", err, ErrTxSigning)
}

func TestGetRawSigs(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	_, addrmgrNs := TstRWNamespaces(dbtx)

	tx := createWithdrawalTx(t, dbtx, pool, []int64{5e6, 4e6}, []int64{})

	sigs, err := getRawSigs([]*withdrawalTx{tx})
	if err != nil {
		t.Fatal(err)
	}
	msgtx := tx.toMsgTx()
	txSigs := sigs[tx.ntxid()]
	if len(txSigs) != len(tx.inputs) {
		t.Fatalf("Unexpected number of sig lists; got %d, want %d", len(txSigs), len(tx.inputs))
	}

	checkNonEmptySigsForPrivKeys(t, txSigs, tx.inputs[0].addr.series().privateKeys)

//因为我们有所有必要的签名（m-of-n），所以我们构造
//sigsnature脚本并执行它们以确保原始签名
//有效。
	signTxAndValidate(t, pool.Manager(), addrmgrNs, msgtx, txSigs, tx.inputs)
}

func TestGetRawSigsOnlyOnePrivKeyAvailable(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	tx := createWithdrawalTx(t, dbtx, pool, []int64{5e6, 4e6}, []int64{})
//删除信用证系列中除第一个以外的所有私钥。
	series := tx.inputs[0].addr.series()
	for i := range series.privateKeys[1:] {
		series.privateKeys[i] = nil
	}

	sigs, err := getRawSigs([]*withdrawalTx{tx})
	if err != nil {
		t.Fatal(err)
	}

	txSigs := sigs[tx.ntxid()]
	if len(txSigs) != len(tx.inputs) {
		t.Fatalf("Unexpected number of sig lists; got %d, want %d", len(txSigs), len(tx.inputs))
	}

	checkNonEmptySigsForPrivKeys(t, txSigs, series.privateKeys)
}

func TestGetRawSigsUnparseableRedeemScript(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	tx := createWithdrawalTx(t, dbtx, pool, []int64{5e6, 4e6}, []int64{})
//更改其中一个Tx输入的兑换脚本，以强制
//GeTracWig（）。
	tx.inputs[0].addr.script = []byte{0x01}

	_, err = getRawSigs([]*withdrawalTx{tx})

	TstCheckError(t, "", err, ErrRawSigning)
}

func TestGetRawSigsInvalidAddrBranch(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	tx := createWithdrawalTx(t, dbtx, pool, []int64{5e6, 4e6}, []int64{})
//将输入地址的分支更改为无效值，以强制
//getrawsigs（）中的错误。
	tx.inputs[0].addr.branch = Branch(999)

	_, err = getRawSigs([]*withdrawalTx{tx})

	TstCheckError(t, "", err, ErrInvalidBranch)
}

//testOutbailmentID排序可以正确排序切片的测试
//的输出请求。
func TestOutBailmentIDSort(t *testing.T) {
	or00 := OutputRequest{cachedHash: []byte{0, 0}}
	or01 := OutputRequest{cachedHash: []byte{0, 1}}
	or10 := OutputRequest{cachedHash: []byte{1, 0}}
	or11 := OutputRequest{cachedHash: []byte{1, 1}}

	want := []OutputRequest{or00, or01, or10, or11}
	random := []OutputRequest{or11, or00, or10, or01}

	sort.Sort(byOutBailmentID(random))

	if !reflect.DeepEqual(random, want) {
		t.Fatalf("Sort failed; got %v, want %v", random, want)
	}
}

func TestTxTooBig(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	tx := createWithdrawalTx(t, dbtx, pool, []int64{5}, []int64{1})

	tx.calculateSize = func() int { return txMaxSize - 1 }
	if tx.isTooBig() {
		t.Fatalf("Tx is smaller than max size (%d < %d) but was considered too big",
			tx.calculateSize(), txMaxSize)
	}

//大小等于TxMaxSize的Tx应被视为太大。
	tx.calculateSize = func() int { return txMaxSize }
	if !tx.isTooBig() {
		t.Fatalf("Tx size is equal to the max size (%d == %d) but was not considered too big",
			tx.calculateSize(), txMaxSize)
	}

	tx.calculateSize = func() int { return txMaxSize + 1 }
	if !tx.isTooBig() {
		t.Fatalf("Tx size is bigger than max size (%d > %d) but was not considered too big",
			tx.calculateSize(), txMaxSize)
	}
}

func TestTxSizeCalculation(t *testing.T) {
	tearDown, db, pool := TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	_, addrmgrNs := TstRWNamespaces(dbtx)

	tx := createWithdrawalTx(t, dbtx, pool, []int64{1, 5}, []int64{2})

	size := tx.calculateSize()

//现在添加一个更改输出，获取一个msgtx，对其进行签名并使其序列化
//与上面的值进行比较。我们需要更换计算费用
//方法，以便下面的tx.addChange（）调用始终添加更改
//输出。
	tx.calculateFee = TstConstantFee(1)
	seriesID := tx.inputs[0].addr.SeriesID()
	tx.addChange(TstNewChangeAddress(t, pool, seriesID, 0).addr.ScriptAddress())
	msgtx := tx.toMsgTx()
	sigs, err := getRawSigs([]*withdrawalTx{tx})
	if err != nil {
		t.Fatal(err)
	}
	signTxAndValidate(t, pool.Manager(), addrmgrNs, msgtx, sigs[tx.ntxid()], tx.inputs)

//ECDSA签名的长度可变（71-73字节），但
//calculateSize（）我们在最坏情况下使用一个虚拟签名（73
//字节），因此这里的估计值最多可以大2个字节
//在每个输入的sigscript中签名。
	maxDiff := 2 * len(msgtx.TxIn) * int(tx.inputs[0].addr.series().reqSigs)
//更糟的是，有可能
//实际签名说明位于其中一个uint*的上边界。
//类型，当这种情况发生时，我们的虚拟签名脚本可能
//不能用同一uint*类型表示的长度
//实际的，所以我们也需要在这里说明。按照
//wire.varintSeriesize（），最大的区别是4
//
//需要UIT64。
	maxDiff += 4 * len(msgtx.TxIn)
	if size-msgtx.SerializeSize() > maxDiff {
		t.Fatalf("Size difference bigger than maximum expected: %d - %d > %d",
			size, msgtx.SerializeSize(), maxDiff)
	} else if size-msgtx.SerializeSize() < 0 {
		t.Fatalf("Tx size (%d) bigger than estimated size (%d)", msgtx.SerializeSize(), size)
	}
}

func TestTxFeeEstimationForSmallTx(t *testing.T) {
	tx := newWithdrawalTx(defaultTxOptions)

//小于1000字节的Tx应收取10000的费用。
//萨索斯
	tx.calculateSize = func() int { return 999 }
	fee := tx.calculateFee()

	wantFee := btcutil.Amount(1e3)
	if fee != wantFee {
		t.Fatalf("Unexpected tx fee; got %v, want %v", fee, wantFee)
	}
}

func TestTxFeeEstimationForLargeTx(t *testing.T) {
	tx := newWithdrawalTx(defaultTxOptions)

//大于1000字节的Tx应收取1E3的费用。
//每1000字节，Satoshis加上1E3。
	tx.calculateSize = func() int { return 3000 }
	fee := tx.calculateFee()

	wantFee := btcutil.Amount(4e3)
	if fee != wantFee {
		t.Fatalf("Unexpected tx fee; got %v, want %v", fee, wantFee)
	}
}

func TestStoreTransactionsWithoutChangeOutput(t *testing.T) {
	tearDown, db, pool, store := TstCreatePoolAndTxStore(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	txmgrNs := dbtx.ReadWriteBucket(txmgrNamespaceKey)

	wtx := createWithdrawalTxWithStoreCredits(t, dbtx, store, pool, []int64{4e6}, []int64{3e6})
	tx := &changeAwareTx{MsgTx: wtx.toMsgTx(), changeIdx: int32(-1)}
	if err := storeTransactions(store, txmgrNs, []*changeAwareTx{tx}); err != nil {
		t.Fatal(err)
	}

	credits, err := store.UnspentOutputs(txmgrNs)
	if err != nil {
		t.Fatal(err)
	}
	if len(credits) != 0 {
		t.Fatalf("Unexpected number of credits in txstore; got %d, want 0", len(credits))
	}
}

func TestStoreTransactionsWithChangeOutput(t *testing.T) {
	tearDown, db, pool, store := TstCreatePoolAndTxStore(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	txmgrNs := dbtx.ReadWriteBucket(txmgrNamespaceKey)

	wtx := createWithdrawalTxWithStoreCredits(t, dbtx, store, pool, []int64{5e6}, []int64{1e6, 1e6})
	wtx.changeOutput = wire.NewTxOut(int64(3e6), []byte{})
	msgtx := wtx.toMsgTx()
	tx := &changeAwareTx{MsgTx: msgtx, changeIdx: int32(len(msgtx.TxOut) - 1)}

	if err := storeTransactions(store, txmgrNs, []*changeAwareTx{tx}); err != nil {
		t.Fatal(err)
	}

	hash := msgtx.TxHash()
	txDetails, err := store.TxDetails(txmgrNs, &hash)
	if err != nil {
		t.Fatal(err)
	}
	if txDetails == nil {
		t.Fatal("The new tx doesn't seem to have been stored")
	}

	storedTx := txDetails.TxRecord.MsgTx
	outputTotal := int64(0)
	for i, txOut := range storedTx.TxOut {
		if int32(i) != tx.changeIdx {
			outputTotal += txOut.Value
		}
	}
	if outputTotal != int64(2e6) {
		t.Fatalf("Unexpected output amount; got %v, want %v", outputTotal, int64(2e6))
	}

	inputTotal := btcutil.Amount(0)
	for _, debit := range txDetails.Debits {
		inputTotal += debit.Amount
	}
	if inputTotal != btcutil.Amount(5e6) {
		t.Fatalf("Unexpected input amount; got %v, want %v", inputTotal, btcutil.Amount(5e6))
	}

	credits, err := store.UnspentOutputs(txmgrNs)
	if err != nil {
		t.Fatal(err)
	}
	if len(credits) != 1 {
		t.Fatalf("Unexpected number of credits in txstore; got %d, want 1", len(credits))
	}
	changeOutpoint := wire.OutPoint{Hash: hash, Index: uint32(tx.changeIdx)}
	if credits[0].OutPoint != changeOutpoint {
		t.Fatalf("Credit's outpoint (%v) doesn't match the one from change output (%v)",
			credits[0].OutPoint, changeOutpoint)
	}
}

//CreateDrawAltxWithStoreCredits在给定存储中创建新的信贷
//对于inputamounts中的每个条目，并使用它们构造一个取款机
//输出中的每个条目都有一个输出。
func createWithdrawalTxWithStoreCredits(t *testing.T, dbtx walletdb.ReadWriteTx, store *wtxmgr.Store, pool *Pool,
	inputAmounts []int64, outputAmounts []int64) *withdrawalTx {
	masters := []*hdkeychain.ExtendedKey{
		TstCreateMasterKey(t, bytes.Repeat(uint32ToBytes(getUniqueID()), 4)),
		TstCreateMasterKey(t, bytes.Repeat(uint32ToBytes(getUniqueID()), 4)),
		TstCreateMasterKey(t, bytes.Repeat(uint32ToBytes(getUniqueID()), 4)),
	}
	def := TstCreateSeriesDef(t, pool, 2, masters)
	TstCreateSeries(t, dbtx, pool, []TstSeriesDef{def})
	net := pool.Manager().ChainParams()
	tx := newWithdrawalTx(defaultTxOptions)
	for _, c := range TstCreateSeriesCreditsOnStore(t, dbtx, pool, def.SeriesID, inputAmounts, store) {
		tx.addInput(c)
	}
	for i, amount := range outputAmounts {
		request := TstNewOutputRequest(
			t, uint32(i), "34eVkREKgvvGASZW7hkgE2uNc1yycntMK6", btcutil.Amount(amount), net)
		tx.addOutput(request)
	}
	return tx
}

//checknonemptysigsforprivkeys检查txsigs中的每个签名列表是否具有
//给定列表中的每个非空私钥都有一个非空签名。这个
//确保每个签名列表符合
//http://opentransactions.org/wiki/index.php/siglist。
func checkNonEmptySigsForPrivKeys(t *testing.T, txSigs TxSigs, privKeys []*hdkeychain.ExtendedKey) {
	for _, txInSigs := range txSigs {
		if len(txInSigs) != len(privKeys) {
			t.Fatalf("Number of items in sig list (%d) does not match number of privkeys (%d)",
				len(txInSigs), len(privKeys))
		}
		for sigIdx, sig := range txInSigs {
			key := privKeys[sigIdx]
			if bytes.Equal(sig, []byte{}) && key != nil {
				t.Fatalf("Empty signature (idx=%d) but key (%s) is available",
					sigIdx, key.String())
			} else if !bytes.Equal(sig, []byte{}) && key == nil {
				t.Fatalf("Signature not empty (idx=%d) but key is not available", sigIdx)
			}
		}
	}
}

//checkTxOutputs使用reflect.deepequal（）确保Tx输出匹配
//给定的提取参数片。
func checkTxOutputs(t *testing.T, tx *withdrawalTx, outputs []*withdrawalTxOut) {
	nOutputs := len(outputs)
	if len(tx.outputs) != nOutputs {
		t.Fatalf("Wrong number of outputs in tx; got %d, want %d", len(tx.outputs), nOutputs)
	}
	for i, output := range tx.outputs {
		if !reflect.DeepEqual(output, outputs[i]) {
			t.Fatalf("Unexpected output; got %s, want %s", output, outputs[i])
		}
	}
}

//checkmsgtxoutputs检查pkscript和
//给定的msgtx匹配pkscript和切片中每个项的数量
//输出请求。
func checkMsgTxOutputs(t *testing.T, msgtx *wire.MsgTx, requests []OutputRequest) {
	nRequests := len(requests)
	if len(msgtx.TxOut) != nRequests {
		t.Fatalf("Unexpected number of TxOuts; got %d, want %d", len(msgtx.TxOut), nRequests)
	}
	for i, request := range requests {
		txOut := msgtx.TxOut[i]
		if !bytes.Equal(txOut.PkScript, request.PkScript) {
			t.Fatalf(
				"Unexpected pkScript for request %d; got %v, want %v", i, txOut.PkScript,
				request.PkScript)
		}
		gotAmount := btcutil.Amount(txOut.Value)
		if gotAmount != request.Amount {
			t.Fatalf(
				"Unexpected amount for request %d; got %v, want %v", i, gotAmount, request.Amount)
		}
	}
}

//检查tx输入确保tx.输入与给定输入匹配。
func checkTxInputs(t *testing.T, tx *withdrawalTx, inputs []credit) {
	if len(tx.inputs) != len(inputs) {
		t.Fatalf("Wrong number of inputs in tx; got %d, want %d", len(tx.inputs), len(inputs))
	}
	for i, input := range tx.inputs {
		if !reflect.DeepEqual(input, inputs[i]) {
			t.Fatalf("Unexpected input; got %v, want %v", input, inputs[i])
		}
	}
}

//signtxandvalidate将为给定的每个输入构造签名脚本
//事务（使用给定的原始签名和来自信贷的pkscripts）并执行
//这些脚本用于验证它们。
func signTxAndValidate(t *testing.T, mgr *waddrmgr.Manager, addrmgrNs walletdb.ReadBucket, tx *wire.MsgTx, txSigs TxSigs,
	credits []credit) {
	for i := range tx.TxIn {
		pkScript := credits[i].PkScript
		TstRunWithManagerUnlocked(t, mgr, addrmgrNs, func() {
			if err := signMultiSigUTXO(mgr, addrmgrNs, tx, i, pkScript, txSigs[i]); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func compareMsgTxAndWithdrawalTxInputs(t *testing.T, msgtx *wire.MsgTx, tx *withdrawalTx) {
	if len(msgtx.TxIn) != len(tx.inputs) {
		t.Fatalf("Wrong number of inputs; got %d, want %d", len(msgtx.TxIn), len(tx.inputs))
	}

	for i, txin := range msgtx.TxIn {
		outpoint := tx.inputs[i].OutPoint
		if txin.PreviousOutPoint != outpoint {
			t.Fatalf("Wrong outpoint; got %v expected %v", txin.PreviousOutPoint, outpoint)
		}
	}
}

func compareMsgTxAndWithdrawalTxOutputs(t *testing.T, msgtx *wire.MsgTx, tx *withdrawalTx) {
	nOutputs := len(tx.outputs)

	if tx.changeOutput != nil {
		nOutputs++
	}

	if len(msgtx.TxOut) != nOutputs {
		t.Fatalf("Unexpected number of TxOuts; got %d, want %d", len(msgtx.TxOut), nOutputs)
	}

	for i, output := range tx.outputs {
		outputRequest := output.request
		txOut := msgtx.TxOut[i]
		if !bytes.Equal(txOut.PkScript, outputRequest.PkScript) {
			t.Fatalf(
				"Unexpected pkScript for outputRequest %d; got %x, want %x",
				i, txOut.PkScript, outputRequest.PkScript)
		}
		gotAmount := btcutil.Amount(txOut.Value)
		if gotAmount != outputRequest.Amount {
			t.Fatalf(
				"Unexpected amount for outputRequest %d; got %v, want %v",
				i, gotAmount, outputRequest.Amount)
		}
	}

//最后检查更改输出是否存在
	if tx.changeOutput != nil {
		msgTxChange := msgtx.TxOut[len(msgtx.TxOut)-1]
		if msgTxChange != tx.changeOutput {
			t.Fatalf("wrong TxOut in msgtx; got %v, want %v", msgTxChange, tx.changeOutput)
		}
	}
}

func checkTxChangeAmount(t *testing.T, tx *withdrawalTx, amount btcutil.Amount) {
	if !tx.hasChange() {
		t.Fatalf("Transaction has no change.")
	}
	if tx.changeOutput.Value != int64(amount) {
		t.Fatalf("Wrong change output amount; got %d, want %d",
			tx.changeOutput.Value, int64(amount))
	}
}

//checkLastOutputwasSplit确保
//假设Tx与NewAmount匹配，并且SplitRequest金额等于
//折纸金额-新金额。它还检查splitrequest是否相同（除了
//对于其数量）发送至Tx中最后一个输出的请求。
func checkLastOutputWasSplit(t *testing.T, w *withdrawal, tx *withdrawalTx,
	origAmount, newAmount btcutil.Amount) {
	splitRequest := w.pendingRequests[0]
	lastOutput := tx.outputs[len(tx.outputs)-1]
	if lastOutput.amount != newAmount {
		t.Fatalf("Wrong amount in last output; got %s, want %s", lastOutput.amount, newAmount)
	}

	wantSplitAmount := origAmount - newAmount
	if splitRequest.Amount != wantSplitAmount {
		t.Fatalf("Wrong amount in split output; got %v, want %v", splitRequest.Amount,
			wantSplitAmount)
	}

//检查拆分请求是否与
//原来的一个。
	origRequest := lastOutput.request
	if !bytes.Equal(origRequest.PkScript, splitRequest.PkScript) {
		t.Fatalf("Wrong pkScript in split request; got %x, want %x", splitRequest.PkScript,
			origRequest.PkScript)
	}
	if origRequest.Server != splitRequest.Server {
		t.Fatalf("Wrong server in split request; got %s, want %s", splitRequest.Server,
			origRequest.Server)
	}
	if origRequest.Transaction != splitRequest.Transaction {
		t.Fatalf("Wrong transaction # in split request; got %d, want %d", splitRequest.Transaction,
			origRequest.Transaction)
	}

	status := w.status.outputs[origRequest.outBailmentID()].status
	if status != statusPartial {
		t.Fatalf("Wrong output status; got '%s', want '%s'", status, statusPartial)
	}
}
