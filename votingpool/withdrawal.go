
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
	"crypto/sha256"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/btcsuite/btcwallet/wtxmgr"
)

//最大Tx大小（字节）。这应该和比特币一样
//最大_标准_tx_尺寸。
const txMaxSize = 100000

//feeincretment是最低交易费（0.00001 BTC，以Satoshis为单位）
//添加到需要收费的交易中。
const feeIncrement = 1e3

type outputStatus byte

const (
	statusSuccess outputStatus = iota
	statusPartial
	statusSplit
)

//OutbailmentID是用户发件箱的唯一ID，包括
//用户连接到的服务器的名称和事务号，
//在该服务器内部。
type OutBailmentID string

//ntxid是给定比特币交易的规范化ID，生成
//通过在所有输入上使用空白的SIG脚本散列序列化的Tx。
type Ntxid string

//OutputRequest表示
//退出，并包含有关用户发件箱请求的信息。
type OutputRequest struct {
	Address  btcutil.Address
	Amount   btcutil.Amount
	PkScript []byte

//接收到发件箱请求的公证服务器。
	Server string

//发件箱请求的服务器特定事务号。
	Transaction uint32

//cachedHash用于缓存OutbailmentID的哈希，因此
//只需计算一次。
	cachedHash []byte
}

//提款输出表示可能完成的输出请求。
type WithdrawalOutput struct {
	request OutputRequest
	status  outputStatus
//完成输出请求的输出点。如果我们
//需要将请求拆分为多个事务。
	outpoints []OutBailmentOutpoint
}

//OutpailmentOutpoint表示为满足OutputRequest而创建的输出点之一。
type OutBailmentOutpoint struct {
	ntxid  Ntxid
	index  uint32
	amount btcutil.Amount
}

//changeawaretx只是一个包装线.msgtx，它知道它的变化。
//输出，如果有的话。
type changeAwareTx struct {
	*wire.MsgTx
changeIdx int32 //-1如果没有变化输出。
}

//
//每个请求输出的状态、网络费用总额和
//下一个输入和更改地址，以便在随后的取款请求中使用。
type WithdrawalStatus struct {
	nextInputAddr  WithdrawalAddress
	nextChangeAddr ChangeAddress
	fees           btcutil.Amount
	outputs        map[OutBailmentID]*WithdrawalOutput
	sigs           map[Ntxid]TxSigs
	transactions   map[Ntxid]changeAwareTx
}

//提款信息包含现有提款的所有详细信息，包括
//原始请求参数和
//开始退出。
type withdrawalInfo struct {
	requests      []OutputRequest
	startAddress  WithdrawalAddress
	changeStart   ChangeAddress
	lastSeriesID  uint32
	dustThreshold btcutil.Amount
	status        WithdrawalStatus
}

//txsigs是原始签名列表（多个sig中的每个pubkey一个）
//脚本）对于给定的事务输入。它们应该与公钥的顺序相匹配
//在脚本中，当
//PubKey未知。
type TxSigs [][]RawSig

//rawsig表示的解锁脚本中包含的签名之一
//从p2sh utxos输入支出。
type RawSig []byte

//ByAmount定义满足排序所需的方法。接口到
//按输出请求的数量对其切片进行排序。
type byAmount []OutputRequest

func (u byAmount) Len() int           { return len(u) }
func (u byAmount) Less(i, j int) bool { return u[i].Amount < u[j].Amount }
func (u byAmount) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }

//ByOutbailmentID定义满足排序所需的方法。要排序的接口
//OutpailmentIdHash所生成的输出请求切片。
type byOutBailmentID []OutputRequest

func (s byOutBailmentID) Len() int      { return len(s) }
func (s byOutBailmentID) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s byOutBailmentID) Less(i, j int) bool {
	return bytes.Compare(s[i].outBailmentIDHash(), s[j].outBailmentIDHash()) < 0
}

func (s outputStatus) String() string {
	strings := map[outputStatus]string{
		statusSuccess: "success",
		statusPartial: "partial-",
		statusSplit:   "split",
	}
	return strings[s]
}

func (tx *changeAwareTx) addSelfToStore(store *wtxmgr.Store, txmgrNs walletdb.ReadWriteBucket) error {
	rec, err := wtxmgr.NewTxRecordFromMsgTx(tx.MsgTx, time.Now())
	if err != nil {
		return newError(ErrWithdrawalTxStorage, "error constructing TxRecord for storing", err)
	}

	if err := store.InsertTx(txmgrNs, rec, nil); err != nil {
		return newError(ErrWithdrawalTxStorage, "error adding tx to store", err)
	}
	if tx.changeIdx != -1 {
		if err = store.AddCredit(txmgrNs, rec, nil, uint32(tx.changeIdx), true); err != nil {
			return newError(ErrWithdrawalTxStorage, "error adding tx credits to store", err)
		}
	}
	return nil
}

//输出返回所有输出的Outbailment ID到抽出输出的映射
//本次提款请求。
func (s *WithdrawalStatus) Outputs() map[OutBailmentID]*WithdrawalOutput {
	return s.outputs
}

//sigs为tx中的每个输入返回ntxids到签名列表的映射
//和那个NTXID。
func (s *WithdrawalStatus) Sigs() map[Ntxid]TxSigs {
	return s.sigs
}

//费用返回包括在所有交易中的网络费用总额
//作为提取的一部分生成。
func (s *WithdrawalStatus) Fees() btcutil.Amount {
	return s.fees
}

//nextinputaddr返回应用作
//后续提款的起始地址。
func (s *WithdrawalStatus) NextInputAddr() WithdrawalAddress {
	return s.nextInputAddr
}

//NextChangeAddr返回应用作
//变更后续提款的开始。
func (s *WithdrawalStatus) NextChangeAddr() ChangeAddress {
	return s.nextChangeAddr
}

//字符串使OutputRequest满足Stringer接口。
func (r OutputRequest) String() string {
	return fmt.Sprintf("OutputRequest %s to send %v to %s", r.outBailmentID(), r.Amount, r.Address)
}

func (r OutputRequest) outBailmentID() OutBailmentID {
	return OutBailmentID(fmt.Sprintf("%s:%d", r.Server, r.Transaction))
}

//OutbailmentIdHash返回排序时使用的字节片
//输出请求。
func (r OutputRequest) outBailmentIDHash() []byte {
	if r.cachedHash != nil {
		return r.cachedHash
	}
	str := r.Server + strconv.Itoa(int(r.Transaction))
	hasher := sha256.New()
//hasher.write（）总是返回nil作为错误，因此在这里忽略它是安全的。
	_, _ = hasher.Write([]byte(str))
	id := hasher.Sum(nil)
	r.cachedHash = id
	return id
}

func (o *WithdrawalOutput) String() string {
	return fmt.Sprintf("WithdrawalOutput for %s", o.request)
}

func (o *WithdrawalOutput) addOutpoint(outpoint OutBailmentOutpoint) {
	o.outpoints = append(o.outpoints, outpoint)
}

//status返回此提款的状态。
func (o *WithdrawalOutput) Status() string {
	return o.status.String()
}

//address返回此提款输出地址的字符串表示形式。
func (o *WithdrawalOutput) Address() string {
	return o.request.Address.String()
}

//Outpoints返回一个切片，其中包含创建给
//完成这个输出。
func (o *WithdrawalOutput) Outpoints() []OutBailmentOutpoint {
	return o.outpoints
}

//Amount返回此OutpailmentOutpoint中的金额（以Satoshis为单位）。
func (o OutBailmentOutpoint) Amount() btcutil.Amount {
	return o.amount
}

//撤销保存pool.retraction（）完成其工作所需的所有状态。
type withdrawal struct {
	roundID         uint32
	status          *WithdrawalStatus
	transactions    []*withdrawalTx
	pendingRequests []OutputRequest
	eligibleInputs  []credit
	current         *withdrawalTx
//txOptions是一个函数，用于为创建为
//部分退出。它被定义为一个函数字段，因为它
//存在的主要目的是为了测试可以模拟提取字段。
	txOptions func(tx *withdrawalTx)
}

//提款Xout包装OutputRequest并提供单独的金额字段。
//这是必要的，因为有些请求可能部分完成或拆分
//跨交易。
type withdrawalTxOut struct {
//注意，在分割输出的情况下，这里的输出请求将
//为原件的副本，金额为
//最初请求的金额减去其他人完成的金额
//退出。如果需要，可以获得原始输出请求
//退出状态输出。
	request OutputRequest
	amount  btcutil.Amount
}

//string使draftAltXout满足stringer接口。
func (o *withdrawalTxOut) String() string {
	return fmt.Sprintf("withdrawalTxOut fulfilling %v of %s", o.amount, o.request)
}

func (o *withdrawalTxOut) pkScript() []byte {
	return o.request.PkScript
}

//drawAltx表示由提取过程构造的事务。
type withdrawalTx struct {
	inputs  []credit
	outputs []*withdrawalTxOut
	fee     btcutil.Amount

//changeoutput保存有关此事务更改的信息。
	changeOutput *wire.TxOut

//CalculateSize返回此的估计序列化大小（以字节为单位）
//有关如何完成的详细信息，请参阅calculatetxsize（）。我们使用
//结构字段而不是方法，以便可以在测试中替换它。
	calculateSize func() int
//计算费用计算此TX的预期网络费用。我们使用
//结构字段而不是方法，以便可以在测试中替换它。
	calculateFee func() btcutil.Amount
}

//newdrawAltx创建一个新的drawAltx并调用setOptions（）。
//传递新创建的Tx。
func newWithdrawalTx(setOptions func(tx *withdrawalTx)) *withdrawalTx {
	tx := &withdrawalTx{}
	tx.calculateSize = func() int { return calculateTxSize(tx) }
	tx.calculateFee = func() btcutil.Amount {
		return btcutil.Amount(1+tx.calculateSize()/1000) * feeIncrement
	}
	setOptions(tx)
	return tx
}

//ntxid返回此事务的唯一ID。
func (tx *withdrawalTx) ntxid() Ntxid {
	msgtx := tx.toMsgTx()
	var empty []byte
	for _, txin := range msgtx.TxIn {
		txin.SignatureScript = empty
	}
	return Ntxid(msgtx.TxHash().String())
}

//如果给定Tx的大小（以字节为单位）大于
//大于或等于TxMaxSize。
func (tx *withdrawalTx) isTooBig() bool {
//在比特币中，只有小于
//max_standard_tx_size；这就是为什么我们考虑任何大于等于tx max size的
//太大了。
	return tx.calculateSize() >= txMaxSize
}

//inputTotal返回此tx中所有输入的总和。
func (tx *withdrawalTx) inputTotal() (total btcutil.Amount) {
	for _, input := range tx.inputs {
		total += input.Amount
	}
	return total
}

//OutputTotal返回此Tx中所有输出的总和。它不返回
//如果Tx有变更输出，则包括变更输出的金额。
func (tx *withdrawalTx) outputTotal() (total btcutil.Amount) {
	for _, output := range tx.outputs {
		total += output.amount
	}
	return total
}

//如果此事务具有更改输出，则HasChange返回true。
func (tx *withdrawalTx) hasChange() bool {
	return tx.changeOutput != nil
}

//tomsgtx使用该tx的输入和输出生成btcwire.msgtx。
func (tx *withdrawalTx) toMsgTx() *wire.MsgTx {
	msgtx := wire.NewMsgTx(wire.TxVersion)
	for _, o := range tx.outputs {
		msgtx.AddTxOut(wire.NewTxOut(int64(o.amount), o.pkScript()))
	}

	if tx.hasChange() {
		msgtx.AddTxOut(tx.changeOutput)
	}

	for _, i := range tx.inputs {
		msgtx.AddTxIn(wire.NewTxIn(&i.OutPoint, []byte{}, nil))
	}
	return msgtx
}

//addoutput向该事务添加新的输出。
func (tx *withdrawalTx) addOutput(request OutputRequest) {
	log.Debugf("Added tx output sending %s to %s", request.Amount, request.Address)
	tx.outputs = append(tx.outputs, &withdrawalTxOut{request: request, amount: request.Amount})
}

//removeoutput删除最后添加的输出并返回它。
func (tx *withdrawalTx) removeOutput() *withdrawalTxOut {
	removed := tx.outputs[len(tx.outputs)-1]
	tx.outputs = tx.outputs[:len(tx.outputs)-1]
	log.Debugf("Removed tx output sending %s to %s", removed.amount, removed.request.Address)
	return removed
}

//addinput将新输入添加到此事务。
func (tx *withdrawalTx) addInput(input credit) {
	log.Debugf("Added tx input with amount %v", input.Amount)
	tx.inputs = append(tx.inputs, input)
}

//removeInput删除最后添加的输入并返回它。
func (tx *withdrawalTx) removeInput() credit {
	removed := tx.inputs[len(tx.inputs)-1]
	tx.inputs = tx.inputs[:len(tx.inputs)-1]
	log.Debugf("Removed tx input with amount %v", removed.Amount)
	return removed
}

//addchange在付款后如果有任何剩余的satoshis，则添加一个更改输出。
//所有输出和网络费用。如果更改输出为
//补充。
//
//此方法只能调用一次，不应调用任何额外的输入/输出
//调用后添加。此外，调用站点必须确保添加更改
//输出不会导致Tx超过大小限制。
func (tx *withdrawalTx) addChange(pkScript []byte) bool {
	tx.fee = tx.calculateFee()
	change := tx.inputTotal() - tx.outputTotal() - tx.fee
	log.Debugf("addChange: input total %v, output total %v, fee %v", tx.inputTotal(),
		tx.outputTotal(), tx.fee)
	if change > 0 {
		tx.changeOutput = wire.NewTxOut(int64(change), pkScript)
		log.Debugf("Added change output with amount %v", change)
	}
	return tx.hasChange()
}

//RollbackLastOutput将回滚上次添加的输出，并可能删除
//不再需要覆盖剩余输出的输入。方法
//以相反的顺序返回已删除的输出和已删除的输入
//如果有的话。
//
//Tx需要有两个或更多输出。只有一个输出的情况必须
//单独处理（通过拆分输出过程）。
func (tx *withdrawalTx) rollBackLastOutput() ([]credit, *withdrawalTxOut, error) {
//检查前提条件：事务中至少需要两个输出。
	if len(tx.outputs) < 2 {
		str := fmt.Sprintf("at least two outputs expected; got %d", len(tx.outputs))
		return nil, nil, newError(ErrPreconditionNotMet, str, nil)
	}

	removedOutput := tx.removeOutput()

	var removedInputs []credit
//继续到SUM（入）<SUM（出）＋FEE
	for tx.inputTotal() >= tx.outputTotal()+tx.calculateFee() {
		removedInputs = append(removedInputs, tx.removeInput())
	}

//从removedinputs中重新添加最后一项，这是最后一个弹出的输入。
	tx.addInput(removedInputs[len(removedInputs)-1])
	removedInputs = removedInputs[:len(removedInputs)-1]
	return removedInputs, removedOutput, nil
}

func defaultTxOptions(tx *withdrawalTx) {}

func newWithdrawal(roundID uint32, requests []OutputRequest, inputs []credit,
	changeStart ChangeAddress) *withdrawal {
	outputs := make(map[OutBailmentID]*WithdrawalOutput, len(requests))
	for _, request := range requests {
		outputs[request.outBailmentID()] = &WithdrawalOutput{request: request}
	}
	status := &WithdrawalStatus{
		outputs:        outputs,
		nextChangeAddr: changeStart,
	}
	return &withdrawal{
		roundID:         roundID,
		pendingRequests: requests,
		eligibleInputs:  inputs,
		status:          status,
		txOptions:       defaultTxOptions,
	}
}

//StartRetraction使用完全确定的算法构造
//尽可能多地满足给定输出请求的事务。
//它返回一个包含完成
//请求的输出和规范化事务ID（ntxid）到的映射
//每个钱包的签名列表（每个钱包可用的私人钥匙一个）
//这些交易的输入。关于实际算法的更多细节可以
//可在http://opentransactions.org/wiki/index.php/startdrawing找到
//必须在地址管理器未锁定的情况下调用此方法。
func (p *Pool) StartWithdrawal(ns walletdb.ReadWriteBucket, addrmgrNs walletdb.ReadBucket, roundID uint32, requests []OutputRequest,
	startAddress WithdrawalAddress, lastSeriesID uint32, changeStart ChangeAddress,
	txStore *wtxmgr.Store, txmgrNs walletdb.ReadBucket, chainHeight int32, dustThreshold btcutil.Amount) (
	*WithdrawalStatus, error) {

	status, err := getWithdrawalStatus(p, ns, addrmgrNs, roundID, requests, startAddress, lastSeriesID,
		changeStart, dustThreshold)
	if err != nil {
		return nil, err
	}
	if status != nil {
		return status, nil
	}

	eligible, err := p.getEligibleInputs(ns, addrmgrNs, txStore, txmgrNs, startAddress, lastSeriesID, dustThreshold,
		chainHeight, eligibleInputMinConfirmations)
	if err != nil {
		return nil, err
	}

	w := newWithdrawal(roundID, requests, eligible, changeStart)
	if err := w.fulfillRequests(); err != nil {
		return nil, err
	}
	w.status.sigs, err = getRawSigs(w.transactions)
	if err != nil {
		return nil, err
	}

	serialized, err := serializeWithdrawal(requests, startAddress, lastSeriesID, changeStart,
		dustThreshold, *w.status)
	if err != nil {
		return nil, err
	}
	err = putWithdrawal(ns, p.ID, roundID, serialized)
	if err != nil {
		return nil, err
	}

	return w.status, nil
}

//popRequest从挂起的堆栈中删除并返回第一个请求
//请求。
func (w *withdrawal) popRequest() OutputRequest {
	request := w.pendingRequests[0]
	w.pendingRequests = w.pendingRequests[1:]
	return request
}

//PushRequest将新请求添加到挂起请求堆栈的顶部。
func (w *withdrawal) pushRequest(request OutputRequest) {
	w.pendingRequests = append([]OutputRequest{request}, w.pendingRequests...)
}

//popinput从符合条件的堆栈中删除并返回第一个输入
//输入。
func (w *withdrawal) popInput() credit {
	input := w.eligibleInputs[len(w.eligibleInputs)-1]
	w.eligibleInputs = w.eligibleInputs[:len(w.eligibleInputs)-1]
	return input
}

//pushinput将新输入添加到合格输入堆栈的顶部。
func (w *withdrawal) pushInput(input credit) {
	w.eligibleInputs = append(w.eligibleInputs, input)
}

//如果这个返回，意味着我们已经添加了一个输出和必要的输入来实现这一点。
//产出加上所需费用。这也意味着Tx甚至不会达到大小限制
//在我们添加一个变更输出并对所有输入进行签名之后。
func (w *withdrawal) fulfillNextRequest() error {
	request := w.popRequest()
	output := w.status.outputs[request.outBailmentID()]
//我们从成功的输出状态开始，让处理
//在特殊情况下，适当时进行更改。
	output.status = statusSuccess
	w.current.addOutput(request)

	if w.current.isTooBig() {
		return w.handleOversizeTx()
	}

	fee := w.current.calculateFee()
	for w.current.inputTotal() < w.current.outputTotal()+fee {
		if len(w.eligibleInputs) == 0 {
			log.Debug("Splitting last output because we don't have enough inputs")
			if err := w.splitLastOutput(); err != nil {
				return err
			}
			break
		}
		w.current.addInput(w.popInput())
		fee = w.current.calculateFee()

		if w.current.isTooBig() {
			return w.handleOversizeTx()
		}
	}
	return nil
}

//handleOversizeTx在事务也变为
//要么回滚输出，要么拆分输出。
func (w *withdrawal) handleOversizeTx() error {
	if len(w.current.outputs) > 1 {
		log.Debug("Rolling back last output because tx got too big")
		inputs, output, err := w.current.rollBackLastOutput()
		if err != nil {
			return newError(ErrWithdrawalProcessing, "failed to rollback last output", err)
		}
		for _, input := range inputs {
			w.pushInput(input)
		}
		w.pushRequest(output.request)
	} else if len(w.current.outputs) == 1 {
		log.Debug("Splitting last output because tx got too big...")
		w.pushInput(w.current.removeInput())
		if err := w.splitLastOutput(); err != nil {
			return err
		}
	} else {
		return newError(ErrPreconditionNotMet, "Oversize tx must have at least one output", nil)
	}
	return w.finalizeCurrentTx()
}

//finalizecurrenttx在w.current中完成事务，将其移动到
//最终交易列表，并将w.current替换为新的空
//交易。
func (w *withdrawal) finalizeCurrentTx() error {
	log.Debug("Finalizing current transaction")
	tx := w.current
	if len(tx.outputs) == 0 {
		log.Debug("Current transaction has no outputs, doing nothing")
		return nil
	}

	pkScript, err := txscript.PayToAddrScript(w.status.nextChangeAddr.addr)
	if err != nil {
		return newError(ErrWithdrawalProcessing, "failed to generate pkScript for change address", err)
	}
	if tx.addChange(pkScript) {
		var err error
		w.status.nextChangeAddr, err = nextChangeAddress(w.status.nextChangeAddr)
		if err != nil {
			return newError(ErrWithdrawalProcessing, "failed to get next change address", err)
		}
	}

	ntxid := tx.ntxid()
	for i, txOut := range tx.outputs {
		outputStatus := w.status.outputs[txOut.request.outBailmentID()]
		outputStatus.addOutpoint(
			OutBailmentOutpoint{ntxid: ntxid, index: uint32(i), amount: txOut.amount})
	}

//检查状态为=success的取款输出条目的和是否为
//它们的输出点数量与请求的数量匹配。
	for _, txOut := range tx.outputs {
//查找我们收到的原始请求，因为txout.request可能
//表示拆分请求，因此与
//原来的一个。
		outputStatus := w.status.outputs[txOut.request.outBailmentID()]
		origRequest := outputStatus.request
		amtFulfilled := btcutil.Amount(0)
		for _, outpoint := range outputStatus.outpoints {
			amtFulfilled += outpoint.amount
		}
		if outputStatus.status == statusSuccess && amtFulfilled != origRequest.Amount {
			msg := fmt.Sprintf("%s was not completely fulfilled; only %v fulfilled", origRequest,
				amtFulfilled)
			return newError(ErrWithdrawalProcessing, msg, nil)
		}
	}

	w.transactions = append(w.transactions, tx)
	w.current = newWithdrawalTx(w.txOptions)
	return nil
}

//MaybedoPrequests将检查我们在合格输入中的总金额并丢弃
//如果我们没有足够的
//把他们都完成。对于每个丢弃的输出请求，我们更新其输入
//w.status.outputs，状态字符串设置为statusspadial。
func (w *withdrawal) maybeDropRequests() {
	inputAmount := btcutil.Amount(0)
	for _, input := range w.eligibleInputs {
		inputAmount += input.Amount
	}
	outputAmount := btcutil.Amount(0)
	for _, request := range w.pendingRequests {
		outputAmount += request.Amount
	}
	sort.Sort(sort.Reverse(byAmount(w.pendingRequests)))
	for inputAmount < outputAmount {
		request := w.popRequest()
		log.Infof("Not fulfilling request to send %v to %v; not enough credits.",
			request.Amount, request.Address)
		outputAmount -= request.Amount
		w.status.outputs[request.outBailmentID()].status = statusPartial
	}
}

func (w *withdrawal) fulfillRequests() error {
	w.maybeDropRequests()
	if len(w.pendingRequests) == 0 {
		return nil
	}

//按OutbailmentID（哈希（服务器ID，Tx）对输出进行排序）
	sort.Sort(byOutBailmentID(w.pendingRequests))

	w.current = newWithdrawalTx(w.txOptions)
	for len(w.pendingRequests) > 0 {
		if err := w.fulfillNextRequest(); err != nil {
			return err
		}
		tx := w.current
		if len(w.eligibleInputs) == 0 && tx.inputTotal() <= tx.outputTotal()+tx.calculateFee() {
//我们没有更多符合条件的输入和
//当前Tx已用完。
			break
		}
	}

	if err := w.finalizeCurrentTx(); err != nil {
		return err
	}

//TODO:更新w.status.nextinputaddr。还没有实现
//我们需要了解的关于未解冻系列的条件。

	w.status.transactions = make(map[Ntxid]changeAwareTx, len(w.transactions))
	for _, tx := range w.transactions {
		w.status.updateStatusFor(tx)
		w.status.fees += tx.fee
		msgtx := tx.toMsgTx()
		changeIdx := -1
		if tx.hasChange() {
//当取款机有变化时，我们知道它将是最后一个条目。
//在生成的MSGTX中。
			changeIdx = len(msgtx.TxOut) - 1
		}
		w.status.transactions[tx.ntxid()] = changeAwareTx{
			MsgTx:     msgtx,
			changeIdx: int32(changeIdx),
		}
	}
	return nil
}

func (w *withdrawal) splitLastOutput() error {
	if len(w.current.outputs) == 0 {
		return newError(ErrPreconditionNotMet,
			"splitLastOutput requires current tx to have at least 1 output", nil)
	}

	tx := w.current
	output := tx.outputs[len(tx.outputs)-1]
	log.Debugf("Splitting tx output for %s", output.request)
	origAmount := output.amount
	spentAmount := tx.outputTotal() + tx.calculateFee() - output.amount
//这就是我们在满足除最后一个输出之外的所有输出之后剩下的量
//一个。噢，我们只剩下最后一个输出了，所以我们把它设置为
//Tx上次输出的量。
	unspentAmount := tx.inputTotal() - spentAmount
	output.amount = unspentAmount
	log.Debugf("Updated output amount to %v", output.amount)

//创建一个新的输出请求，其金额为
//原始数量以及上述Tx输出中的剩余量。
	request := output.request
	newRequest := OutputRequest{
		Server:      request.Server,
		Transaction: request.Transaction,
		Address:     request.Address,
		PkScript:    request.PkScript,
		Amount:      origAmount - output.amount}
	w.pushRequest(newRequest)
	log.Debugf("Created a new pending output request with amount %v", newRequest.Amount)

	w.status.outputs[request.outBailmentID()].status = statusPartial
	return nil
}

func (s *WithdrawalStatus) updateStatusFor(tx *withdrawalTx) {
	for _, output := range s.outputs {
		if len(output.outpoints) > 1 {
			output.status = statusSplit
		}
//TODO:更新状态为“部分-”的输出。为此，我们需要一个API
//这给了我们一个系列中的信用额度。
//http://opentransactions.org/wiki/index.php/update_状态
	}
}

//如果给定参数与此中的字段匹配，则match返回true
//提取信息。对于请求切片，项的顺序不
//物质。
func (wi *withdrawalInfo) match(requests []OutputRequest, startAddress WithdrawalAddress,
	lastSeriesID uint32, changeStart ChangeAddress, dustThreshold btcutil.Amount) bool {
//使用reflect.deepequal比较changestart和startaddress
//包含指针的结构，我们希望比较它们的内容和
//不是他们的地址。
	if !reflect.DeepEqual(changeStart, wi.changeStart) {
		log.Debugf("withdrawal changeStart does not match: %v != %v", changeStart, wi.changeStart)
		return false
	}
	if !reflect.DeepEqual(startAddress, wi.startAddress) {
		log.Debugf("withdrawal startAddr does not match: %v != %v", startAddress, wi.startAddress)
		return false
	}
	if lastSeriesID != wi.lastSeriesID {
		log.Debugf("withdrawal lastSeriesID does not match: %v != %v", lastSeriesID,
			wi.lastSeriesID)
		return false
	}
	if dustThreshold != wi.dustThreshold {
		log.Debugf("withdrawal dustThreshold does not match: %v != %v", dustThreshold,
			wi.dustThreshold)
		return false
	}
	r1 := make([]OutputRequest, len(requests))
	copy(r1, requests)
	r2 := make([]OutputRequest, len(wi.requests))
	copy(r2, wi.requests)
	sort.Sort(byOutBailmentID(r1))
	sort.Sort(byOutBailmentID(r2))
	if !reflect.DeepEqual(r1, r2) {
		log.Debugf("withdrawal requests does not match: %v != %v", requests, wi.requests)
		return false
	}
	return true
}

//GetDrawAlStatus返回给定的
//提取参数（如果存在）。必须使用
//地址管理器已解锁。
func getWithdrawalStatus(p *Pool, ns, addrmgrNs walletdb.ReadBucket, roundID uint32, requests []OutputRequest,
	startAddress WithdrawalAddress, lastSeriesID uint32, changeStart ChangeAddress,
	dustThreshold btcutil.Amount) (*WithdrawalStatus, error) {

	serialized := getWithdrawal(ns, p.ID, roundID)
	if bytes.Equal(serialized, []byte{}) {
		return nil, nil
	}
	wInfo, err := deserializeWithdrawal(p, ns, addrmgrNs, serialized)
	if err != nil {
		return nil, err
	}
	if wInfo.match(requests, startAddress, lastSeriesID, changeStart, dustThreshold) {
		return &wInfo.status, nil
	}
	return nil, nil
}

//getrawsigs迭代给定的每个事务的输入，构造
//使用我们可用的私钥为它们生成原始签名。
//它将ntxids的映射返回到签名列表。
func getRawSigs(transactions []*withdrawalTx) (map[Ntxid]TxSigs, error) {
	sigs := make(map[Ntxid]TxSigs)
	for _, tx := range transactions {
		txSigs := make(TxSigs, len(tx.inputs))
		msgtx := tx.toMsgTx()
		ntxid := tx.ntxid()
		for inputIdx, input := range tx.inputs {
			creditAddr := input.addr
			redeemScript := creditAddr.redeemScript()
			series := creditAddr.series()
//签名脚本中原始签名的顺序必须与
//赎回脚本中公钥的顺序，因此我们对公钥进行排序
//这里使用相同的API在兑换脚本中对它们进行排序并使用
//series.getprivkeyor（）查找相应的私钥。
			pubKeys, err := branchOrder(series.publicKeys, creditAddr.Branch())
			if err != nil {
				return nil, err
			}
			txInSigs := make([]RawSig, len(pubKeys))
			for i, pubKey := range pubKeys {
				var sig RawSig
				privKey, err := series.getPrivKeyFor(pubKey)
				if err != nil {
					return nil, err
				}
				if privKey != nil {
					childKey, err := privKey.Child(uint32(creditAddr.Index()))
					if err != nil {
						return nil, newError(ErrKeyChain, "failed to derive private key", err)
					}
					ecPrivKey, err := childKey.ECPrivKey()
					if err != nil {
						return nil, newError(ErrKeyChain, "failed to obtain ECPrivKey", err)
					}
					log.Debugf("Generating raw sig for input %d of tx %s with privkey of %s",
						inputIdx, ntxid, pubKey.String())
					sig, err = txscript.RawTxInSignature(
						msgtx, inputIdx, redeemScript, txscript.SigHashAll, ecPrivKey)
					if err != nil {
						return nil, newError(ErrRawSigning, "failed to generate raw signature", err)
					}
				} else {
					log.Debugf("Not generating raw sig for input %d of %s because private key "+
						"for %s is not available: %v", inputIdx, ntxid, pubKey.String(), err)
					sig = []byte{}
				}
				txInSigs[i] = sig
			}
			txSigs[inputIdx] = txInSigs
		}
		sigs[ntxid] = txSigs
	}
	return sigs, nil
}

//signtx通过查找（在地址上）对给定msgtx的每个输入进行签名
//经理）每个人的兑换脚本并构造签名
//使用它和给定的原始签名编写脚本。
//必须在管理器未锁定的情况下调用此函数。
func SignTx(msgtx *wire.MsgTx, sigs TxSigs, mgr *waddrmgr.Manager, addrmgrNs walletdb.ReadBucket, store *wtxmgr.Store, txmgrNs walletdb.ReadBucket) error {
//我们在这里使用time.now（），因为我们不打算存储新的txrecord
//任何地方——我们只需要把它传递到store.previouspkscripts（）。
	rec, err := wtxmgr.NewTxRecordFromMsgTx(msgtx, time.Now())
	if err != nil {
		return newError(ErrTxSigning, "failed to construct TxRecord for signing", err)
	}
	pkScripts, err := store.PreviousPkScripts(txmgrNs, rec, nil)
	if err != nil {
		return newError(ErrTxSigning, "failed to obtain pkScripts for signing", err)
	}
	for i, pkScript := range pkScripts {
		if err = signMultiSigUTXO(mgr, addrmgrNs, msgtx, i, pkScript, sigs[i]); err != nil {
			return err
		}
	}
	return nil
}

//getredeemscript返回给定p2sh地址的兑换脚本。它必须
//在经理解锁的情况下被呼叫。
func getRedeemScript(mgr *waddrmgr.Manager, addrmgrNs walletdb.ReadBucket, addr *btcutil.AddressScriptHash) ([]byte, error) {
	address, err := mgr.Address(addrmgrNs, addr)
	if err != nil {
		return nil, err
	}
	return address.(waddrmgr.ManagedScriptAddress).Script()
}

//signmultisgutxo通过构造一个
//包含所有给定签名的脚本加上赎回（多个SIG）脚本。这个
//通过查找给定p2sh pkscript的地址，可以获取兑换脚本。
//在地址管理器上。
//签名的顺序必须与多个SIG中的公钥的顺序匹配
//op-checkmultisig期望的脚本。
//必须在管理器未锁定的情况下调用此函数。
func signMultiSigUTXO(mgr *waddrmgr.Manager, addrmgrNs walletdb.ReadBucket, tx *wire.MsgTx, idx int, pkScript []byte, sigs []RawSig) error {
	class, addresses, _, err := txscript.ExtractPkScriptAddrs(pkScript, mgr.ChainParams())
	if err != nil {
		return newError(ErrTxSigning, "unparseable pkScript", err)
	}
	if class != txscript.ScriptHashTy {
		return newError(ErrTxSigning, fmt.Sprintf("pkScript is not P2SH: %s", class), nil)
	}
	redeemScript, err := getRedeemScript(mgr, addrmgrNs, addresses[0].(*btcutil.AddressScriptHash))
	if err != nil {
		return newError(ErrTxSigning, "unable to retrieve redeem script", err)
	}

	class, _, nRequired, err := txscript.ExtractPkScriptAddrs(redeemScript, mgr.ChainParams())
	if err != nil {
		return newError(ErrTxSigning, "unparseable redeem script", err)
	}
	if class != txscript.MultiSigTy {
		return newError(ErrTxSigning, fmt.Sprintf("redeem script is not multi-sig: %v", class), nil)
	}
	if len(sigs) < nRequired {
		errStr := fmt.Sprintf("not enough signatures; need %d but got only %d", nRequired,
			len(sigs))
		return newError(ErrTxSigning, errStr, nil)
	}

//构造解锁脚本。
//从0开始，因为比特币中的错误，然后添加需要的签名。
	unlockingScript := txscript.NewScriptBuilder().AddOp(txscript.OP_FALSE)
	for _, sig := range sigs[:nRequired] {
		unlockingScript.AddData(sig)
	}

//结合兑换脚本和解锁脚本，得到实际的签名脚本。
	sigScript := unlockingScript.AddData(redeemScript)
	script, err := sigScript.Script()
	if err != nil {
		return newError(ErrTxSigning, "error building sigscript", err)
	}
	tx.TxIn[idx].SignatureScript = script

	if err := validateSigScript(tx, idx, pkScript); err != nil {
		return err
	}
	return nil
}

//validateSigscripts使用
//给定索引，如果失败则返回错误。
func validateSigScript(msgtx *wire.MsgTx, idx int, pkScript []byte) error {
	vm, err := txscript.NewEngine(pkScript, msgtx, idx,
		txscript.StandardVerifyFlags, nil, nil, 0)
	if err != nil {
		return newError(ErrTxSigning, "cannot create script engine", err)
	}
	if err = vm.Execute(); err != nil {
		return newError(ErrTxSigning, "cannot validate tx signature", err)
	}
	return nil
}

//calculatetxsize返回对
//给定的事务。它假设所有Tx输入都是P2SH多信号。
func calculateTxSize(tx *withdrawalTx) int {
	msgtx := tx.toMsgTx()
//为了简单起见，假设总是有一个变更输出。我们
//通过简单地复制第一个输出来模拟这一点，我们所关心的就是
//其序列化窗体的大小，对于所有这些窗体都应相同
//因为它们不是p2pkh就是p2sh。
	if !tx.hasChange() {
		msgtx.AddTxOut(msgtx.TxOut[0])
	}
//为这个Tx中的每个输入创建一个带有虚拟签名的签名描述
//这样我们就可以使用msgtx.serializesize（）获取它的大小，而不需要
//依靠估计。
	for i, txin := range msgtx.TxIn {
//操作错误操作码为1字节，每个签名为73+1字节
//用它们的操作码和最后的兑换脚本+1字节
//对于它的oph-pushdata操作码和n字节对于兑换脚本的大小。
//注意，我们使用73作为签名长度，因为这是最大值
//它们的长度可能是：
//https://en.bitcoin.it/wiki/椭圆曲线\数字签名\算法
		addr := tx.inputs[i].addr
		redeemScriptLen := len(addr.redeemScript())
		n := wire.VarIntSerializeSize(uint64(redeemScriptLen))
		sigScriptLen := 1 + (74 * int(addr.series().reqSigs)) + redeemScriptLen + 1 + n
		txin.SignatureScript = bytes.Repeat([]byte{1}, sigScriptLen)
	}
	return msgtx.SerializeSize()
}

func nextChangeAddress(a ChangeAddress) (ChangeAddress, error) {
	index := a.index
	seriesID := a.seriesID
	if index == math.MaxUint32 {
		index = 0
		seriesID++
	} else {
		index++
	}
	addr, err := a.pool.ChangeAddress(seriesID, index)
	return *addr, err
}

func storeTransactions(store *wtxmgr.Store, txmgrNs walletdb.ReadWriteBucket, transactions []*changeAwareTx) error {
	for _, tx := range transactions {
		if err := tx.addSelfToStore(store, txmgrNs); err != nil {
			return err
		}
	}
	return nil
}
