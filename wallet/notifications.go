
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wallet

import (
	"bytes"
	"sync"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/btcsuite/btcwallet/wtxmgr"
)

//TODO:最好在创建通知期间将错误发送到RPC
//服务器而不是将它们记录在这里，这样客户机就知道钱包
//工作不正常，通知丢失。

//TODO:这里处理帐户的任何操作都很昂贵，因为数据库
//没有正确组织以获得真正的客户支持，但要做的是缓慢的事情
//而不是简单的事情，因为数据库可以稍后修复，我们希望
//API现在更正了。

//NotificationServer是感兴趣的客户机可以连接到的服务器
//收到钱包更改通知。为每个
//注册通知。保证客户端在
//订单钱包创建了它们，但在
//不同的客户。
type NotificationServer struct {
	transactions   []chan *TransactionNotifications
currentTxNtfn  *TransactionNotifications //合并这个，因为钱包不会将挖掘的TXS添加到一起。
	spentness      map[uint32][]chan *SpentnessNotifications
	accountClients []chan *AccountNotification
mu             sync.Mutex //仅保护已注册的客户端通道
wallet         *Wallet    //
}

func newNotificationServer(wallet *Wallet) *NotificationServer {
	return &NotificationServer{
		spentness: make(map[uint32][]chan *SpentnessNotifications),
		wallet:    wallet,
	}
}

func lookupInputAccount(dbtx walletdb.ReadTx, w *Wallet, details *wtxmgr.TxDetails, deb wtxmgr.DebitRecord) uint32 {
	addrmgrNs := dbtx.ReadBucket(waddrmgrNamespaceKey)
	txmgrNs := dbtx.ReadBucket(wtxmgrNamespaceKey)

//TODO:借方应记录哪个账户？他们
//从中借记，这样就不需要查询。
	prevOP := &details.MsgTx.TxIn[deb.Index].PreviousOutPoint
	prev, err := w.TxStore.TxDetails(txmgrNs, &prevOP.Hash)
	if err != nil {
		log.Errorf("Cannot query previous transaction details for %v: %v", prevOP.Hash, err)
		return 0
	}
	if prev == nil {
		log.Errorf("Missing previous transaction %v", prevOP.Hash)
		return 0
	}
	prevOut := prev.MsgTx.TxOut[prevOP.Index]
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(prevOut.PkScript, w.chainParams)
	var inputAcct uint32
	if err == nil && len(addrs) > 0 {
		_, inputAcct, err = w.Manager.AddrAccount(addrmgrNs, addrs[0])
	}
	if err != nil {
		log.Errorf("Cannot fetch account for previous output %v: %v", prevOP, err)
		inputAcct = 0
	}
	return inputAcct
}

func lookupOutputChain(dbtx walletdb.ReadTx, w *Wallet, details *wtxmgr.TxDetails,
	cred wtxmgr.CreditRecord) (account uint32, internal bool) {

	addrmgrNs := dbtx.ReadBucket(waddrmgrNamespaceKey)

	output := details.MsgTx.TxOut[cred.Index]
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(output.PkScript, w.chainParams)
	var ma waddrmgr.ManagedAddress
	if err == nil && len(addrs) > 0 {
		ma, err = w.Manager.Address(addrmgrNs, addrs[0])
	}
	if err != nil {
		log.Errorf("Cannot fetch account for wallet output: %v", err)
	} else {
		account = ma.Account()
		internal = ma.Internal()
	}
	return
}

func makeTxSummary(dbtx walletdb.ReadTx, w *Wallet, details *wtxmgr.TxDetails) TransactionSummary {
	serializedTx := details.SerializedTx
	if serializedTx == nil {
		var buf bytes.Buffer
		err := details.MsgTx.Serialize(&buf)
		if err != nil {
			log.Errorf("Transaction serialization: %v", err)
		}
		serializedTx = buf.Bytes()
	}
	var fee btcutil.Amount
	if len(details.Debits) == len(details.MsgTx.TxIn) {
		for _, deb := range details.Debits {
			fee += deb.Amount
		}
		for _, txOut := range details.MsgTx.TxOut {
			fee -= btcutil.Amount(txOut.Value)
		}
	}
	var inputs []TransactionSummaryInput
	if len(details.Debits) != 0 {
		inputs = make([]TransactionSummaryInput, len(details.Debits))
		for i, d := range details.Debits {
			inputs[i] = TransactionSummaryInput{
				Index:           d.Index,
				PreviousAccount: lookupInputAccount(dbtx, w, details, d),
				PreviousAmount:  d.Amount,
			}
		}
	}
	outputs := make([]TransactionSummaryOutput, 0, len(details.MsgTx.TxOut))
	for i := range details.MsgTx.TxOut {
		credIndex := len(outputs)
		mine := len(details.Credits) > credIndex && details.Credits[credIndex].Index == uint32(i)
		if !mine {
			continue
		}
		acct, internal := lookupOutputChain(dbtx, w, details, details.Credits[credIndex])
		output := TransactionSummaryOutput{
			Index:    uint32(i),
			Account:  acct,
			Internal: internal,
		}
		outputs = append(outputs, output)
	}
	return TransactionSummary{
		Hash:        &details.Hash,
		Transaction: serializedTx,
		MyInputs:    inputs,
		MyOutputs:   outputs,
		Fee:         fee,
		Timestamp:   details.Received.Unix(),
	}
}

func totalBalances(dbtx walletdb.ReadTx, w *Wallet, m map[uint32]btcutil.Amount) error {
	addrmgrNs := dbtx.ReadBucket(waddrmgrNamespaceKey)
	unspent, err := w.TxStore.UnspentOutputs(dbtx.ReadBucket(wtxmgrNamespaceKey))
	if err != nil {
		return err
	}
	for i := range unspent {
		output := &unspent[i]
		var outputAcct uint32
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(
			output.PkScript, w.chainParams)
		if err == nil && len(addrs) > 0 {
			_, outputAcct, err = w.Manager.AddrAccount(addrmgrNs, addrs[0])
		}
		if err == nil {
			_, ok := m[outputAcct]
			if ok {
				m[outputAcct] += output.Amount
			}
		}
	}
	return nil
}

func flattenBalanceMap(m map[uint32]btcutil.Amount) []AccountBalance {
	s := make([]AccountBalance, 0, len(m))
	for k, v := range m {
		s = append(s, AccountBalance{Account: k, TotalBalance: v})
	}
	return s
}

func relevantAccounts(w *Wallet, m map[uint32]btcutil.Amount, txs []TransactionSummary) {
	for _, tx := range txs {
		for _, d := range tx.MyInputs {
			m[d.PreviousAccount] = 0
		}
		for _, c := range tx.MyOutputs {
			m[c.Account] = 0
		}
	}
}

func (s *NotificationServer) notifyUnminedTransaction(dbtx walletdb.ReadTx, details *wtxmgr.TxDetails) {
//健全性检查：当前不应合并的通知
//在通知未链接的Tx的同时挖掘事务。
	if s.currentTxNtfn != nil {
		log.Errorf("Notifying unmined tx notification (%s) while creating notification for blocks",
			details.Hash)
	}

	defer s.mu.Unlock()
	s.mu.Lock()
	clients := s.transactions
	if len(clients) == 0 {
		return
	}

	unminedTxs := []TransactionSummary{makeTxSummary(dbtx, s.wallet, details)}
	unminedHashes, err := s.wallet.TxStore.UnminedTxHashes(dbtx.ReadBucket(wtxmgrNamespaceKey))
	if err != nil {
		log.Errorf("Cannot fetch unmined transaction hashes: %v", err)
		return
	}
	bals := make(map[uint32]btcutil.Amount)
	relevantAccounts(s.wallet, bals, unminedTxs)
	err = totalBalances(dbtx, s.wallet, bals)
	if err != nil {
		log.Errorf("Cannot determine balances for relevant accounts: %v", err)
		return
	}
	n := &TransactionNotifications{
		UnminedTransactions:      unminedTxs,
		UnminedTransactionHashes: unminedHashes,
		NewBalances:              flattenBalanceMap(bals),
	}
	for _, c := range clients {
		c <- n
	}
}

func (s *NotificationServer) notifyDetachedBlock(hash *chainhash.Hash) {
	if s.currentTxNtfn == nil {
		s.currentTxNtfn = &TransactionNotifications{}
	}
	s.currentTxNtfn.DetachedBlocks = append(s.currentTxNtfn.DetachedBlocks, hash)
}

func (s *NotificationServer) notifyMinedTransaction(dbtx walletdb.ReadTx, details *wtxmgr.TxDetails, block *wtxmgr.BlockMeta) {
	if s.currentTxNtfn == nil {
		s.currentTxNtfn = &TransactionNotifications{}
	}
	n := len(s.currentTxNtfn.AttachedBlocks)
	if n == 0 || *s.currentTxNtfn.AttachedBlocks[n-1].Hash != block.Hash {
		s.currentTxNtfn.AttachedBlocks = append(s.currentTxNtfn.AttachedBlocks, Block{
			Hash:      &block.Hash,
			Height:    block.Height,
			Timestamp: block.Time.Unix(),
		})
		n++
	}
	txs := s.currentTxNtfn.AttachedBlocks[n-1].Transactions
	s.currentTxNtfn.AttachedBlocks[n-1].Transactions =
		append(txs, makeTxSummary(dbtx, s.wallet, details))
}

func (s *NotificationServer) notifyAttachedBlock(dbtx walletdb.ReadTx, block *wtxmgr.BlockMeta) {
	if s.currentTxNtfn == nil {
		s.currentTxNtfn = &TransactionNotifications{}
	}

//如果以前没有包含块详细信息，则添加块详细信息
//
	n := len(s.currentTxNtfn.AttachedBlocks)
	if n == 0 || *s.currentTxNtfn.AttachedBlocks[n-1].Hash != block.Hash {
		s.currentTxNtfn.AttachedBlocks = append(s.currentTxNtfn.AttachedBlocks, Block{
			Hash:      &block.Hash,
			Height:    block.Height,
			Timestamp: block.Time.Unix(),
		})
	}

//现在（直到不需要合并通知为止）只需使用
//链长决定这是否是新的最佳块。
	if s.wallet.ChainSynced() {
		if len(s.currentTxNtfn.DetachedBlocks) >= len(s.currentTxNtfn.AttachedBlocks) {
			return
		}
	}

	defer s.mu.Unlock()
	s.mu.Lock()
	clients := s.transactions
	if len(clients) == 0 {
		s.currentTxNtfn = nil
		return
	}

//“UnlinedTransactions”字段是故意不设置的。自从
//报告所有分离块的哈希，以及所有事务
//从采空区移回未确认区
//UnlinedTransactionHashes切片或由于与冲突而不存在
//在新的最佳链中挖掘的事务，不可能
//出现在未确认中的新的、以前未看到的事务。

	txmgrNs := dbtx.ReadBucket(wtxmgrNamespaceKey)
	unminedHashes, err := s.wallet.TxStore.UnminedTxHashes(txmgrNs)
	if err != nil {
		log.Errorf("Cannot fetch unmined transaction hashes: %v", err)
		return
	}
	s.currentTxNtfn.UnminedTransactionHashes = unminedHashes

	bals := make(map[uint32]btcutil.Amount)
	for _, b := range s.currentTxNtfn.AttachedBlocks {
		relevantAccounts(s.wallet, bals, b.Transactions)
	}
	err = totalBalances(dbtx, s.wallet, bals)
	if err != nil {
		log.Errorf("Cannot determine balances for relevant accounts: %v", err)
		return
	}
	s.currentTxNtfn.NewBalances = flattenBalanceMap(bals)

	for _, c := range clients {
		c <- s.currentTxNtfn
	}
	s.currentTxNtfn = nil
}

//交易通知是钱包更改的通知。
//交易集和钱包被视为
//
//他们的地雷区。
//
//在链切换期间，所有移除的块散列都包含在内。独立的
//块的排序顺序与开采顺序相反。附加块是
//按开采顺序分类。
//
//包括所有新添加的未关联交易。未开采的
//未明确包括交易。相反，所有的哈希
//
//
//如果涉及任何交易，每个受影响账户的新总余额
//包括在内。
//
//托多：因为这包括了关于块的东西，可以不用任何
//对事务进行更改，它需要更好的名称。
type TransactionNotifications struct {
	AttachedBlocks           []Block
	DetachedBlocks           []*chainhash.Hash
	UnminedTransactions      []TransactionSummary
	UnminedTransactionHashes []*chainhash.Hash
	NewBalances              []AccountBalance
}

//块包含附加的属性和所有相关事务
//块。
type Block struct {
	Hash         *chainhash.Hash
	Height       int32
	Timestamp    int64
	Transactions []TransactionSummary
}

//交易摘要包含与钱包和标记相关的交易
//哪些输入和输出是相关的。
type TransactionSummary struct {
	Hash        *chainhash.Hash
	Transaction []byte
	MyInputs    []TransactionSummaryInput
	MyOutputs   []TransactionSummaryOutput
	Fee         btcutil.Amount
	Timestamp   int64
}

//TransactionSummaryInput描述与
//钱包。索引字段标记事务的事务输入索引
//（此处不包括）。PreviousAccount和PreviousMount字段描述
//这个输入从钱包帐户中扣除多少。
type TransactionSummaryInput struct {
	Index           uint32
	PreviousAccount uint32
	PreviousAmount  btcutil.Amount
}

//TransactionSummaryOutput描述交易输出的钱包属性
//由钱包控制。索引字段标记事务输出索引
//
type TransactionSummaryOutput struct {
	Index    uint32
	Account  uint32
	Internal bool
}

//accountBalance将总（零确认）余额与
//帐户。其他最低确认计数的余额需要更多
//昂贵的逻辑，不清楚客户感兴趣的最小值是多少，
//所以不包括在内。
type AccountBalance struct {
	Account      uint32
	TotalBalance btcutil.Amount
}

//TransactionNotificationClient接收来自
//通道C上的通知服务器。
type TransactionNotificationsClient struct {
	C      <-chan *TransactionNotifications
	server *NotificationServer
}

//
//通道上的TransactionNotifications通知。通道是
//无缓冲的
//
//完成后，应在客户端上调用Done方法以解除关联
//它来自服务器。
func (s *NotificationServer) TransactionNotifications() TransactionNotificationsClient {
	c := make(chan *TransactionNotifications)
	s.mu.Lock()
	s.transactions = append(s.transactions, c)
	s.mu.Unlock()
	return TransactionNotificationsClient{
		C:      c,
		server: s,
	}
}

//完成从服务器注销客户端并排出所有剩余的
//信息。客户端完成后必须调用一次
//正在接收通知。
func (c *TransactionNotificationsClient) Done() {
	go func() {
//在从中删除客户端通道之前排出通知
//服务器已关闭。
		for range c.C {
		}
	}()
	go func() {
		s := c.server
		s.mu.Lock()
		clients := s.transactions
		for i, ch := range clients {
			if c.C == ch {
				clients[i] = clients[len(clients)-1]
				s.transactions = clients[:len(clients)-1]
				close(ch)
				break
			}
		}
		s.mu.Unlock()
	}()
}

//SpentnessNotifications是为事务激发的通知
//由某个帐户的密钥控制的输出。通知可能是关于
//新添加的未暂停事务输出或以前未暂停的输出
//现在花了。当支出时，通知包括支出交易的
//哈希和输入索引。
type SpentnessNotifications struct {
	hash         *chainhash.Hash
	spenderHash  *chainhash.Hash
	index        uint32
	spenderIndex uint32
}

//hash返回已用输出的事务哈希。
func (n *SpentnessNotifications) Hash() *chainhash.Hash {
	return n.hash
}

//index返回已用输出的事务输出索引。
func (n *SpentnessNotifications) Index() uint32 {
	return n.index
}

//Spender返回支出转换的哈希和输入索引（如果有）。如果
//输出未暂停，最终bool返回为假。
func (n *SpentnessNotifications) Spender() (*chainhash.Hash, uint32, bool) {
	return n.spenderHash, n.spenderIndex, n.spenderHash != nil
}

//notifyunspentoutput通知注册的客户端新的未使用的输出
//由钱包控制。
func (s *NotificationServer) notifyUnspentOutput(account uint32, hash *chainhash.Hash, index uint32) {
	defer s.mu.Unlock()
	s.mu.Lock()
	clients := s.spentness[account]
	if len(clients) == 0 {
		return
	}
	n := &SpentnessNotifications{
		hash:  hash,
		index: index,
	}
	for _, c := range clients {
		c <- n
	}
}

//notifySpendOutput通知注册客户端以前未使用的
//
//通知。
func (s *NotificationServer) notifySpentOutput(account uint32, op *wire.OutPoint, spenderHash *chainhash.Hash, spenderIndex uint32) {
	defer s.mu.Unlock()
	s.mu.Lock()
	clients := s.spentness[account]
	if len(clients) == 0 {
		return
	}
	n := &SpentnessNotifications{
		hash:         &op.Hash,
		index:        op.Index,
		spenderHash:  spenderHash,
		spenderIndex: spenderIndex,
	}
	for _, c := range clients {
		c <- n
	}
}

//SpentnessNotificationsClient从
//通道C上的通知服务器。
type SpentnessNotificationsClient struct {
	C       <-chan *SpentnessNotifications
	account uint32
	server  *NotificationServer
}

//AccountSpentnessNotifications registers a client for spentness changes of
//输出由帐户控制。
func (s *NotificationServer) AccountSpentnessNotifications(account uint32) SpentnessNotificationsClient {
	c := make(chan *SpentnessNotifications)
	s.mu.Lock()
	s.spentness[account] = append(s.spentness[account], c)
	s.mu.Unlock()
	return SpentnessNotificationsClient{
		C:       c,
		account: account,
		server:  s,
	}
}

//完成从服务器注销客户端并排出所有剩余的
//信息。客户端完成后必须调用一次
//正在接收通知。
func (c *SpentnessNotificationsClient) Done() {
	go func() {
//在从中删除客户端通道之前排出通知
//服务器已关闭。
		for range c.C {
		}
	}()
	go func() {
		s := c.server
		s.mu.Lock()
		clients := s.spentness[c.account]
		for i, ch := range clients {
			if c.C == ch {
				clients[i] = clients[len(clients)-1]
				s.spentness[c.account] = clients[:len(clients)-1]
				close(ch)
				break
			}
		}
		s.mu.Unlock()
	}()
}

//AccountNotification contains properties regarding an account, such as its
//派生键和导入键的名称和数目。当这些
//属性更改后，通知被触发。
type AccountNotification struct {
	AccountNumber    uint32
	AccountName      string
	ExternalKeyCount uint32
	InternalKeyCount uint32
	ImportedKeyCount uint32
}

func (s *NotificationServer) notifyAccountProperties(props *waddrmgr.AccountProperties) {
	defer s.mu.Unlock()
	s.mu.Lock()
	clients := s.accountClients
	if len(clients) == 0 {
		return
	}
	n := &AccountNotification{
		AccountNumber:    props.AccountNumber,
		AccountName:      props.AccountName,
		ExternalKeyCount: props.ExternalKeyCount,
		InternalKeyCount: props.InternalKeyCount,
		ImportedKeyCount: props.ImportedKeyCount,
	}
	for _, c := range clients {
		c <- n
	}
}

//accountnotificationsclient通过通道C接收accountnotifications。
type AccountNotificationsClient struct {
	C      chan *AccountNotification
	server *NotificationServer
}

//会计通知返回客户端接收会计通知。
//一个频道。通道没有缓冲。完成后，客户完成
//应调用方法以解除客户端与服务器的关联。
func (s *NotificationServer) AccountNotifications() AccountNotificationsClient {
	c := make(chan *AccountNotification)
	s.mu.Lock()
	s.accountClients = append(s.accountClients, c)
	s.mu.Unlock()
	return AccountNotificationsClient{
		C:      c,
		server: s,
	}
}

//完成从服务器注销客户端并排出所有剩余的
//信息。客户端完成后必须调用一次
//正在接收通知。
func (c *AccountNotificationsClient) Done() {
	go func() {
		for range c.C {
		}
	}()
	go func() {
		s := c.server
		s.mu.Lock()
		clients := s.accountClients
		for i, ch := range clients {
			if c.C == ch {
				clients[i] = clients[len(clients)-1]
				s.accountClients = clients[:len(clients)-1]
				close(ch)
				break
			}
		}
		s.mu.Unlock()
	}()
}
