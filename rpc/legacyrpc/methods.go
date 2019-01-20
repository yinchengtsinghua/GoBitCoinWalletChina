
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//版权所有（c）2016版权所有
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package legacyrpc

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/chain"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/wallet"
	"github.com/btcsuite/btcwallet/wallet/txrules"
	"github.com/btcsuite/btcwallet/wtxmgr"
)

//已确认检查高度txheight处的事务是否符合minconf
//Height Curheight区块链的确认。
func confirmed(minconf, txHeight, curHeight int32) bool {
	return confirms(txHeight, curHeight) >= minconf
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

//RequestHandler是一个处理程序函数，用于处理未解析和解析的
//请求进入可封送响应。如果错误为*btcjson.rpcerror
//或上述任何特殊错误类，服务器将用
//JSON-RPC应用程序错误代码。所有其他错误使用钱包
//捕获所有错误代码，btcjson.errrpcwallet。
type requestHandler func(interface{}, *wallet.Wallet) (interface{}, error)

//RequestHandlerChain是一个RequestHandler，它还接受
type requestHandlerChainRequired func(interface{}, *wallet.Wallet, *chain.RPCClient) (interface{}, error)

var rpcHandlers = map[string]struct {
	handler          requestHandler
	handlerWithChain requestHandlerChainRequired

//函数变量不能与除nil以外的任何值进行比较，因此
//使用布尔值记录是否需要生成帮助。这个
//由测试使用，以确保可以为
//实现方法。
//
//一张地图，这个布尔图在这里被使用，而不是几个地图
//对于未实现的处理程序，因此每个方法只有一个
//处理程序函数。
	noHelp bool
}{
//参考实施钱包方法（实施）
	"addmultisigaddress":     {handler: addMultiSigAddress},
	"createmultisig":         {handler: createMultiSig},
	"dumpprivkey":            {handler: dumpPrivKey},
	"getaccount":             {handler: getAccount},
	"getaccountaddress":      {handler: getAccountAddress},
	"getaddressesbyaccount":  {handler: getAddressesByAccount},
	"getbalance":             {handler: getBalance},
	"getbestblockhash":       {handler: getBestBlockHash},
	"getblockcount":          {handler: getBlockCount},
	"getinfo":                {handlerWithChain: getInfo},
	"getnewaddress":          {handler: getNewAddress},
	"getrawchangeaddress":    {handler: getRawChangeAddress},
	"getreceivedbyaccount":   {handler: getReceivedByAccount},
	"getreceivedbyaddress":   {handler: getReceivedByAddress},
	"gettransaction":         {handler: getTransaction},
	"help":                   {handler: helpNoChainRPC, handlerWithChain: helpWithChainRPC},
	"importprivkey":          {handler: importPrivKey},
	"keypoolrefill":          {handler: keypoolRefill},
	"listaccounts":           {handler: listAccounts},
	"listlockunspent":        {handler: listLockUnspent},
	"listreceivedbyaccount":  {handler: listReceivedByAccount},
	"listreceivedbyaddress":  {handler: listReceivedByAddress},
	"listsinceblock":         {handlerWithChain: listSinceBlock},
	"listtransactions":       {handler: listTransactions},
	"listunspent":            {handler: listUnspent},
	"lockunspent":            {handler: lockUnspent},
	"sendfrom":               {handlerWithChain: sendFrom},
	"sendmany":               {handler: sendMany},
	"sendtoaddress":          {handler: sendToAddress},
	"settxfee":               {handler: setTxFee},
	"signmessage":            {handler: signMessage},
	"signrawtransaction":     {handlerWithChain: signRawTransaction},
	"validateaddress":        {handler: validateAddress},
	"verifymessage":          {handler: verifyMessage},
	"walletlock":             {handler: walletLock},
	"walletpassphrase":       {handler: walletPassphrase},
	"walletpassphrasechange": {handler: walletPassphraseChange},

//参考实现方法（仍未实现）
	"backupwallet":         {handler: unimplemented, noHelp: true},
	"dumpwallet":           {handler: unimplemented, noHelp: true},
	"getwalletinfo":        {handler: unimplemented, noHelp: true},
	"importwallet":         {handler: unimplemented, noHelp: true},
	"listaddressgroupings": {handler: unimplemented, noHelp: true},

//由于以下原因，btcwallet无法实现的引用方法
//设计决策差异
	"encryptwallet": {handler: unsupported, noHelp: true},
	"move":          {handler: unsupported, noHelp: true},
	"setaccount":    {handler: unsupported, noHelp: true},

//引用客户端JSON-RPC API的扩展
	"createnewaccount": {handler: createNewAccount},
	"getbestblock":     {handler: getBestBlock},
//这是一个扩展，但引用实现将其添加为
//好吧，但是有一个不同的API（没有帐户参数）。它上市了
//这里是因为没有更新以使用引用
//实施API。
	"getunconfirmedbalance":   {handler: getUnconfirmedBalance},
	"listaddresstransactions": {handler: listAddressTransactions},
	"listalltransactions":     {handler: listAllTransactions},
	"renameaccount":           {handler: renameAccount},
	"walletislocked":          {handler: walletIsLocked},
}

//未实现处理未实现的RPC请求
//适用错误。
func unimplemented(interface{}, *wallet.Wallet) (interface{}, error) {
	return nil, &btcjson.RPCError{
		Code:    btcjson.ErrRPCUnimplemented,
		Message: "Method unimplemented",
	}
}

//不受支持的处理标准比特币RPC请求
//由于设计差异，不受btcwallet支持。
func unsupported(interface{}, *wallet.Wallet) (interface{}, error) {
	return nil, &btcjson.RPCError{
		Code:    -1,
		Message: "Request unsupported by btcwallet",
	}
}

//Lazyhandler是对请求处理程序或具有
//作为关闭的一部分，RPC服务器的钱包和链服务器变量
//语境。
type lazyHandler func() (interface{}, *btcjson.RPCError)

//LazyApplyHandler查找该方法的最佳请求处理程序func，
//返回将使用（必需）钱包执行的关闭，以及
//（可选）共识RPC服务器。如果找不到处理程序，
//chainclient不是nil，返回的处理程序执行RPC传递。
func lazyApplyHandler(request *btcjson.Request, w *wallet.Wallet, chainClient chain.Interface) lazyHandler {
	handlerData, ok := rpcHandlers[request.Method]
	if ok && handlerData.handlerWithChain != nil && w != nil && chainClient != nil {
		return func() (interface{}, *btcjson.RPCError) {
			cmd, err := btcjson.UnmarshalCmd(request)
			if err != nil {
				return nil, btcjson.ErrRPCInvalidRequest
			}
			switch client := chainClient.(type) {
			case *chain.RPCClient:
				resp, err := handlerData.handlerWithChain(cmd,
					w, client)
				if err != nil {
					return nil, jsonError(err)
				}
				return resp, nil
			default:
				return nil, &btcjson.RPCError{
					Code:    -1,
					Message: "Chain RPC is inactive",
				}
			}
		}
	}
	if ok && handlerData.handler != nil && w != nil {
		return func() (interface{}, *btcjson.RPCError) {
			cmd, err := btcjson.UnmarshalCmd(request)
			if err != nil {
				return nil, btcjson.ErrRPCInvalidRequest
			}
			resp, err := handlerData.handler(cmd, w)
			if err != nil {
				return nil, jsonError(err)
			}
			return resp, nil
		}
	}

//回退到RPC传递
	return func() (interface{}, *btcjson.RPCError) {
		if chainClient == nil {
			return nil, &btcjson.RPCError{
				Code:    -1,
				Message: "Chain RPC is inactive",
			}
		}
		switch client := chainClient.(type) {
		case *chain.RPCClient:
			resp, err := client.RawRequest(request.Method,
				request.Params)
			if err != nil {
				return nil, jsonError(err)
			}
			return &resp, nil
		default:
			return nil, &btcjson.RPCError{
				Code:    -1,
				Message: "Chain RPC is inactive",
			}
		}
	}
}

//makeresponse为结果和错误生成json-rpc响应结构
//由请求处理程序返回。返回的响应尚未准备好
//封送和发送给客户，但必须
func makeResponse(id, result interface{}, err error) btcjson.Response {
	idPtr := idPointer(id)
	if err != nil {
		return btcjson.Response{
			ID:    idPtr,
			Error: jsonError(err),
		}
	}
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return btcjson.Response{
			ID: idPtr,
			Error: &btcjson.RPCError{
				Code:    btcjson.ErrRPCInternal.Code,
				Message: "Unexpected error marshalling result",
			},
		}
	}
	return btcjson.Response{
		ID:     idPtr,
		Result: json.RawMessage(resultBytes),
	}
}

//jsonError从Go错误创建一个json-rpc错误。
func jsonError(err error) *btcjson.RPCError {
	if err == nil {
		return nil
	}

	code := btcjson.ErrRPCWallet
	switch e := err.(type) {
	case btcjson.RPCError:
		return &e
	case *btcjson.RPCError:
		return e
	case DeserializationError:
		code = btcjson.ErrRPCDeserialization
	case InvalidParameterError:
		code = btcjson.ErrRPCInvalidParameter
	case ParseError:
		code = btcjson.ErrRPCParse.Code
	case waddrmgr.ManagerError:
		switch e.ErrorCode {
		case waddrmgr.ErrWrongPassphrase:
			code = btcjson.ErrRPCWalletPassphraseIncorrect
		}
	}
	return &btcjson.RPCError{
		Code:    code,
		Message: err.Error(),
	}
}

//makemultisigscript是一个帮助函数，用于组合
//添加multisig并创建multisig。
func makeMultiSigScript(w *wallet.Wallet, keys []string, nRequired int) ([]byte, error) {
	keysesPrecious := make([]*btcutil.AddressPubKey, len(keys))

//地址列表将由addreseses（pubkey散列）中的任意一个组成，用于
//我们需要查钱包里的钥匙，直接的钥匙，或者
//两者的混合物。
	for i, a := range keys {
//尝试解析为pubkey地址
		a, err := decodeAddress(a, w.ChainParams())
		if err != nil {
			return nil, err
		}

		switch addr := a.(type) {
		case *btcutil.AddressPubKey:
			keysesPrecious[i] = addr
		default:
			pubKey, err := w.PubKeyForAddress(addr)
			if err != nil {
				return nil, err
			}
			pubKeyAddr, err := btcutil.NewAddressPubKey(
				pubKey.SerializeCompressed(), w.ChainParams())
			if err != nil {
				return nil, err
			}
			keysesPrecious[i] = pubKeyAddr
		}
	}

	return txscript.MultiSigScript(keysesPrecious, nRequired)
}

//addmultisigaddress通过添加
//指定钱包的多搜索地址。
func addMultiSigAddress(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.AddMultisigAddressCmd)

//如果指定了帐户，请确保该帐户是导入的帐户。
	if cmd.Account != nil && *cmd.Account != waddrmgr.ImportedAddrAccountName {
		return nil, &ErrNotImportedAccount
	}

	secp256k1Addrs := make([]btcutil.Address, len(cmd.Keys))
	for i, k := range cmd.Keys {
		addr, err := decodeAddress(k, w.ChainParams())
		if err != nil {
			return nil, ParseError{err}
		}
		secp256k1Addrs[i] = addr
	}

	script, err := w.MakeMultiSigScript(secp256k1Addrs, cmd.NRequired)
	if err != nil {
		return nil, err
	}

	p2shAddr, err := w.ImportP2SHRedeemScript(script)
	if err != nil {
		return nil, err
	}

	return p2shAddr.EncodeAddress(), nil
}

//createMutilsig通过返回
//给定输入的多组地址。
func createMultiSig(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.CreateMultisigCmd)

	script, err := makeMultiSigScript(w, cmd.Keys, cmd.NRequired)
	if err != nil {
		return nil, ParseError{err}
	}

	address, err := btcutil.NewAddressScriptHash(script, w.ChainParams())
	if err != nil {
//上面是一个有效的脚本，不应该发生。
		return nil, err
	}

	return btcjson.CreateMultiSigResult{
		Address:      address.EncodeAddress(),
		RedeemScript: hex.EncodeToString(script),
	}, nil
}

//dumpprivkey使用私钥处理dumpprivkey请求
//对于单个地址，或者如果钱包
//被锁定。
func dumpPrivKey(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.DumpPrivKeyCmd)

	addr, err := decodeAddress(cmd.Address, w.ChainParams())
	if err != nil {
		return nil, err
	}

	key, err := w.DumpWIFPrivateKey(addr)
	if waddrmgr.IsError(err, waddrmgr.ErrLocked) {
//找到了地址，但私钥没有
//可接近的。
		return nil, &ErrWalletUnlockNeeded
	}
	return key, err
}

//dumpwallet通过返回所有私人信息来处理dumpwallet请求
//钱包中的钥匙，或者钱包被锁定时的适当错误。
//TODO:通过将转储文件写入文件来完成此操作以匹配比特币。
func dumpWallet(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	keys, err := w.DumpPrivKeys()
	if waddrmgr.IsError(err, waddrmgr.ErrLocked) {
		return nil, &ErrWalletUnlockNeeded
	}

	return keys, err
}

//GetAddressByAcCount通过返回
//帐户的所有地址，或者如果请求的帐户
//
func getAddressesByAccount(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.GetAddressesByAccountCmd)

	account, err := w.AccountNumber(waddrmgr.KeyScopeBIP0044, cmd.Account)
	if err != nil {
		return nil, err
	}

	addrs, err := w.AccountAddresses(account)
	if err != nil {
		return nil, err
	}

	addrStrs := make([]string, len(addrs))
	for i, a := range addrs {
		addrStrs[i] = a.EncodeAddress()
	}
	return addrStrs, nil
}

//
//帐户（钱包），或者如果请求的帐户没有
//存在。
func getBalance(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.GetBalanceCmd)

	var balance btcutil.Amount
	var err error
	accountName := "*"
	if cmd.Account != nil {
		accountName = *cmd.Account
	}
	if accountName == "*" {
		balance, err = w.CalculateBalance(int32(*cmd.MinConf))
		if err != nil {
			return nil, err
		}
	} else {
		var account uint32
		account, err = w.AccountNumber(waddrmgr.KeyScopeBIP0044, accountName)
		if err != nil {
			return nil, err
		}
		bals, err := w.CalculateAccountBalances(account, int32(*cmd.MinConf))
		if err != nil {
			return nil, err
		}
		balance = bals.Spendable
	}
	return balance.ToBTC(), nil
}

//GetBestBlock通过返回JSON对象来处理GetBestBlock请求
//具有最近处理的块的高度和哈希。
func getBestBlock(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	blk := w.Manager.SyncedTo()
	result := &btcjson.GetBestBlockResult{
		Hash:   blk.Hash.String(),
		Height: blk.Height,
	}
	return result, nil
}

//GetBestBlockHash通过返回哈希来处理GetBestBlockHash请求
//最近处理的块。
func getBestBlockHash(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	blk := w.Manager.SyncedTo()
	return blk.Hash.String(), nil
}

//GetBlockCount通过返回链高度来处理GetBlockCount请求
//最近处理的块。
func getBlockCount(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	blk := w.Manager.SyncedTo()
	return blk.Height, nil
}

//GetInfo通过返回包含
//有关btcwallet当前状态的信息。
//存在。
func getInfo(icmd interface{}, w *wallet.Wallet, chainClient *chain.RPCClient) (interface{}, error) {
//调用btcd获取此命令中已知的所有信息
//由他们。
	info, err := chainClient.GetInfo()
	if err != nil {
		return nil, err
	}

	bal, err := w.CalculateBalance(1)
	if err != nil {
		return nil, err
	}

//TODO（Davec）：这应该有一个相反的数据库版本
//使用管理器版本。
	info.WalletVersion = int32(waddrmgr.LatestMgrVersion)
	info.Balance = bal.ToBTC()
	info.PaytxFee = float64(txrules.DefaultRelayFeePerKb)
//我们不设置以下内容，因为它们在
//钱包结构：
//-解锁\直到
//-错误

	return info, nil
}

func decodeAddress(s string, params *chaincfg.Params) (btcutil.Address, error) {
	addr, err := btcutil.DecodeAddress(s, params)
	if err != nil {
		msg := fmt.Sprintf("Invalid address %q: decode failed with %#q", s, err)
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidAddressOrKey,
			Message: msg,
		}
	}
	if !addr.IsForNet(params) {
		msg := fmt.Sprintf("Invalid address %q: not intended for use on %s",
			addr, params.Name)
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidAddressOrKey,
			Message: msg,
		}
	}
	return addr, nil
}

//GetAccount通过返回帐户名来处理GetAccount请求
//与单个地址关联。
func getAccount(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.GetAccountCmd)

	addr, err := decodeAddress(cmd.Address, w.ChainParams())
	if err != nil {
		return nil, err
	}

//获取关联帐户
	account, err := w.AccountOfAddress(addr)
	if err != nil {
		return nil, &ErrAddressNotInWallet
	}

	acctName, err := w.AccountName(waddrmgr.KeyScopeBIP0044, account)
	if err != nil {
		return nil, &ErrAccountNameNotFound
	}
	return acctName, nil
}

//GetAccountAddress通过返回
//最近创建的链接地址尚未使用（尚未使用
//出现在区块链中，或任何已到达BTCD内存池的Tx）。
//如果使用了最近请求的地址，则为
//使用键池中的下一个链接地址）。如果密钥池
//耗尽（如果发生这种情况，将返回btcjson.errrpcwalletkeypoolranout）。
func getAccountAddress(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.GetAccountAddressCmd)

	account, err := w.AccountNumber(waddrmgr.KeyScopeBIP0044, cmd.Account)
	if err != nil {
		return nil, err
	}
	addr, err := w.CurrentAddress(account, waddrmgr.KeyScopeBIP0044)
	if err != nil {
		return nil, err
	}

	return addr.EncodeAddress(), err
}

//getUnconfirmedBalance处理getUnconfirmedBalance扩展请求
//通过返回帐户的当前未确认余额。
func getUnconfirmedBalance(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.GetUnconfirmedBalanceCmd)

	acctName := "default"
	if cmd.Account != nil {
		acctName = *cmd.Account
	}
	account, err := w.AccountNumber(waddrmgr.KeyScopeBIP0044, acctName)
	if err != nil {
		return nil, err
	}
	bals, err := w.CalculateAccountBalances(account, 1)
	if err != nil {
		return nil, err
	}

	return (bals.Total - bals.Spendable).ToBTC(), nil
}

//importprivkey通过分析处理importprivkey请求
//WIF编码的私钥，并将其添加到帐户中。
func importPrivKey(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.ImportPrivKeyCmd)

//确保只将私钥导入到正确的帐户。
//
//是的，label是帐户名。
	if cmd.Label != nil && *cmd.Label != waddrmgr.ImportedAddrAccountName {
		return nil, &ErrNotImportedAccount
	}

	wif, err := btcutil.DecodeWIF(cmd.PrivKey)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidAddressOrKey,
			Message: "WIF decode failed: " + err.Error(),
		}
	}
	if !wif.IsForNet(w.ChainParams()) {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidAddressOrKey,
			Message: "Key is not intended for " + w.ChainParams().Name,
		}
	}

//导入私钥，处理所有错误。
	_, err = w.ImportPrivateKey(waddrmgr.KeyScopeBIP0044, wif, nil, *cmd.Rescan)
	switch {
	case waddrmgr.IsError(err, waddrmgr.ErrDuplicateAddress):
//不要向客户端返回重复的密钥错误。
		return nil, nil
	case waddrmgr.IsError(err, waddrmgr.ErrLocked):
		return nil, &ErrWalletUnlockNeeded
	}

	return nil, err
}

//keypoolrefill处理keypoolrefill命令。因为我们处理了钥匙池
//自动这不起作用，因为从来没有手动要求重新加注。
func keypoolRefill(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	return nil, nil
}

//CreateNewAccount通过创建和
//返回新帐户。如果最后一个帐户没有交易历史记录
//根据BIP 0044，无法创建新帐户，因此将返回错误。
func createNewAccount(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.CreateNewAccountCmd)

//通配符*由具有特殊含义的RPC服务器保留
//“所有帐户”中，因此不允许将命名帐户用于此字符串。
	if cmd.Account == "*" {
		return nil, &ErrReservedAccountName
	}

	_, err := w.NextAccount(waddrmgr.KeyScopeBIP0044, cmd.Account)
	if waddrmgr.IsError(err, waddrmgr.ErrLocked) {
		return nil, &btcjson.RPCError{
			Code: btcjson.ErrRPCWalletUnlockNeeded,
			Message: "Creating an account requires the wallet to be unlocked. " +
				"Enter the wallet passphrase with walletpassphrase to unlock",
		}
	}
	return nil, err
}

//RenameAccount通过重命名帐户来处理RenameAccount请求。
//如果帐户不存在，将返回一个适当的错误。
func renameAccount(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.RenameAccountCmd)

//通配符*由具有特殊含义的RPC服务器保留
//“所有帐户”中，因此不允许将命名帐户用于此字符串。
	if cmd.NewAccount == "*" {
		return nil, &ErrReservedAccountName
	}

//检查给定帐户是否存在
	account, err := w.AccountNumber(waddrmgr.KeyScopeBIP0044, cmd.OldAccount)
	if err != nil {
		return nil, err
	}
	return nil, w.RenameAccount(waddrmgr.KeyScopeBIP0044, account, cmd.NewAccount)
}

//GetNewAddress通过返回新的
//帐户地址。如果该帐户不存在，则为
//返回错误。
//TODO:遵循BIP 0044，如果未使用的地址数超过了，则发出警告
//差距限制。
func getNewAddress(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.GetNewAddressCmd)

	acctName := "default"
	if cmd.Account != nil {
		acctName = *cmd.Account
	}
	account, err := w.AccountNumber(waddrmgr.KeyScopeBIP0044, acctName)
	if err != nil {
		return nil, err
	}
	addr, err := w.NewAddress(account, waddrmgr.KeyScopeBIP0044)
	if err != nil {
		return nil, err
	}

//返回新的付款地址字符串。
	return addr.EncodeAddress(), nil
}

//GetRawChangeAddress通过创建
//并返回帐户的新更改地址。
//
//注意：比特币允许将帐户指定为可选参数，
//但忽略参数。
func getRawChangeAddress(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.GetRawChangeAddressCmd)

	acctName := "default"
	if cmd.Account != nil {
		acctName = *cmd.Account
	}
	account, err := w.AccountNumber(waddrmgr.KeyScopeBIP0044, acctName)
	if err != nil {
		return nil, err
	}
	addr, err := w.NewChangeAddress(account, waddrmgr.KeyScopeBIP0044)
	if err != nil {
		return nil, err
	}

//返回新的付款地址字符串。
	return addr.EncodeAddress(), nil
}

//GetReceiveDByaccount通过返回来处理GetReceiveDByaccount请求
//按帐户地址接收的总金额。
func getReceivedByAccount(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.GetReceivedByAccountCmd)

	account, err := w.AccountNumber(waddrmgr.KeyScopeBIP0044, cmd.Account)
	if err != nil {
		return nil, err
	}

//托多：这比可能的效率低，但整个
//算法已经由读取
//钱包的历史。
	results, err := w.TotalReceivedForAccounts(
		waddrmgr.KeyScopeBIP0044, int32(*cmd.MinConf),
	)
	if err != nil {
		return nil, err
	}
	acctIndex := int(account)
	if account == waddrmgr.ImportedAddrAccount {
		acctIndex = len(results) - 1
	}
	return results[acctIndex].TotalReceived.ToBTC(), nil
}

//GetReceiveDBYAddress通过返回来处理GetReceiveDBYAddress请求
//单个地址收到的总金额。
func getReceivedByAddress(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.GetReceivedByAddressCmd)

	addr, err := decodeAddress(cmd.Address, w.ChainParams())
	if err != nil {
		return nil, err
	}
	total, err := w.TotalReceivedForAddr(addr, int32(*cmd.MinConf))
	if err != nil {
		return nil, err
	}

	return total.ToBTC(), nil
}

//GetTransaction通过返回有关的详细信息来处理GetTransaction请求
//用钱包存钱的单笔交易。
func getTransaction(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.GetTransactionCmd)

	txHash, err := chainhash.NewHashFromStr(cmd.Txid)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDecodeHexString,
			Message: "Transaction hash string decode failed: " + err.Error(),
		}
	}

	details, err := wallet.UnstableAPI(w).TxDetails(txHash)
	if err != nil {
		return nil, err
	}
	if details == nil {
		return nil, &ErrNoTransactionInfo
	}

	syncBlock := w.Manager.SyncedTo()

//TODO:序列化事务已在数据库中，因此
//在这里可以避免再结晶。
	var txBuf bytes.Buffer
	txBuf.Grow(details.MsgTx.SerializeSize())
	err = details.MsgTx.Serialize(&txBuf)
	if err != nil {
		return nil, err
	}

//TODO:向此结果类型添加“已生成”字段。生成的：真
//仅当事务是CoinBase时才添加。
	ret := btcjson.GetTransactionResult{
		TxID:            cmd.Txid,
		Hex:             hex.EncodeToString(txBuf.Bytes()),
		Time:            details.Received.Unix(),
		TimeReceived:    details.Received.Unix(),
WalletConflicts: []string{}, //未保存
//生成：区块链.iscoinbasetx（&details.msgtx）
	}

	if details.Block.Height != -1 {
		ret.BlockHash = details.Block.Hash.String()
		ret.BlockTime = details.Block.Time.Unix()
		ret.Confirmations = int64(confirms(details.Block.Height, syncBlock.Height))
	}

	var (
		debitTotal  btcutil.Amount
creditTotal btcutil.Amount //排除变化
		fee         btcutil.Amount
		feeF64      float64
	)
	for _, deb := range details.Debits {
		debitTotal += deb.Amount
	}
	for _, cred := range details.Credits {
		if !cred.Change {
			creditTotal += cred.Amount
		}
	}
//只有当每个输入都是借方时，才能确定费用。
	if len(details.Debits) == len(details.MsgTx.TxIn) {
		var outputTotal btcutil.Amount
		for _, output := range details.MsgTx.TxOut {
			outputTotal += btcutil.Amount(output.Value)
		}
		fee = debitTotal - outputTotal
		feeF64 = fee.ToBTC()
	}

	if len(details.Debits) == 0 {
//学分必须晚一点设置，但因为我们知道全部长度
//在细节部分，用正确的帽子分配它。
		ret.Details = make([]btcjson.GetTransactionDetailsResult, 0, len(details.Credits))
	} else {
		ret.Details = make([]btcjson.GetTransactionDetailsResult, 1, len(details.Credits)+1)

		ret.Details[0] = btcjson.GetTransactionDetailsResult{
//字段左零：
//仅限InvolvesWatchOnly
//账户
//地址
//VOUT
//
//TODO（JRICK）：应始终设置地址和凭证，
//但我们在这里做错事，因为不匹配
//核心。相反，getTransaction应该只添加
//事务输出的详细信息，就像
//ListTransactions（但使用短结果格式）。
			Category: "send",
Amount:   (-debitTotal).ToBTC(), //因为它是发送的，所以是否定的
			Fee:      &feeF64,
		}
		ret.Fee = feeF64
	}

	credCat := wallet.RecvCategory(details, syncBlock.Height, w.ChainParams()).String()
	for _, cred := range details.Credits {
//更改被忽略。
		if cred.Change {
			continue
		}

		var address string
		var accountName string
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(
			details.MsgTx.TxOut[cred.Index].PkScript, w.ChainParams())
		if err == nil && len(addrs) == 1 {
			addr := addrs[0]
			address = addr.EncodeAddress()
			account, err := w.AccountOfAddress(addr)
			if err == nil {
				name, err := w.AccountName(waddrmgr.KeyScopeBIP0044, account)
				if err == nil {
					accountName = name
				}
			}
		}

		ret.Details = append(ret.Details, btcjson.GetTransactionDetailsResult{
//字段左零：
//仅限InvolvesWatchOnly
//费用
			Account:  accountName,
			Address:  address,
			Category: credCat,
			Amount:   cred.Amount.ToBTC(),
			Vout:     cred.Index,
		})
	}

	ret.Amount = creditTotal.ToBTC()
	return ret, nil
}

//这些生成器在此包中创建以下全局变量：
//
//var localeHelpDescs映射[string]func（）映射[string]string
//var requestusages字符串
//
//localeHelpDescs从locale字符串（例如“en-us”）映射到
//为每个RPC服务器方法构建帮助文本的映射。这会阻止帮助
//每个区域设置映射的文本映射，这些映射是在init期间根目录和创建的。
//相反，当首先需要帮助文本时，将查找appropiate函数。
//使用当前区域设置并保存到下面的全局区域以供进一步重用。
//
//请求用法包含每个支持的请求的单行用法，
//用换行符分隔。它是在初始化期间设置的。这些用法适用于所有
//场所。
//
//go:生成go run../。/internal/rpchelp/genrpcserverhelp.go legacyrpc
//go：生成gofmt-w rpcserverhelp.go

var helpDescs map[string]string
var helpDescsMu sync.Mutex //帮助可能同时执行，因此同步访问。

//helpWithChainRPC在RPC服务器
//与共识RPC客户端关联。附加的RPC客户端用于
//包括共识服务器通过RPC实现的方法的帮助消息
//通过。
func helpWithChainRPC(icmd interface{}, w *wallet.Wallet, chainClient *chain.RPCClient) (interface{}, error) {
	return help(icmd, w, chainClient)
}

//helpnochainrpc在rpc服务器尚未启动时处理帮助请求
//与共识RPC客户端关联。不包括帮助消息
//直通请求。
func helpNoChainRPC(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	return help(icmd, w, nil)
}

//帮助通过返回所有可用的一行用法来处理帮助请求
//方法，或特定方法的完全帮助。chainclient是可选的，
//这只是helpNoChainRpc的一个助手函数，
//helpWithChainRPC处理程序。
func help(icmd interface{}, w *wallet.Wallet, chainClient *chain.RPCClient) (interface{}, error) {
	cmd := icmd.(*btcjson.HelpCmd)

//btcd根据
//客户端正在使用的连接。只有可用于HTTP Post的方法
//客户可供钱包客户使用，即使
//钱包本身就是一个btcd的websocket客户端。因此，创建一个
//根据需要发布客户端。
//
//如果chainclient当前为nil或存在错误，则返回nil
//正在创建客户端。
//
//这是一种黑客行为，通过公开帮助用法可能更好地处理它。
//非内部BTCD包中的文本。
	postClient := func() *rpcclient.Client {
		if chainClient == nil {
			return nil
		}
		c, err := chainClient.POSTClient()
		if err != nil {
			return nil
		}
		return c
	}
	if cmd.Command == nil || *cmd.Command == "" {
//如果链服务器可用，则提前使用。
		usages := requestUsages
		client := postClient()
		if client != nil {
			rawChainUsage, err := client.RawRequest("help", nil)
			var chainUsage string
			if err == nil {
				_ = json.Unmarshal([]byte(rawChainUsage), &chainUsage)
			}
			if chainUsage != "" {
				usages = "Chain server usage:\n\n" + chainUsage + "\n\n" +
					"Wallet server usage (overrides chain requests):\n\n" +
					requestUsages
			}
		}
		return usages, nil
	}

	defer helpDescsMu.Unlock()
	helpDescsMu.Lock()

	if helpDescs == nil {
//TODO:允许通过config或detemine设置其他区域设置
//这来自环境变量。现在，硬编码我们
//英语。
		helpDescs = localeHelpDescs["en_US"]()
	}

	helpText, ok := helpDescs[*cmd.Command]
	if ok {
		return helpText, nil
	}

//如果可能，返回链服务器的详细帮助。
	var chainHelp string
	client := postClient()
	if client != nil {
		param := make([]byte, len(*cmd.Command)+2)
		param[0] = '"'
		copy(param[1:], *cmd.Command)
		param[len(param)-1] = '"'
		rawChainHelp, err := client.RawRequest("help", []json.RawMessage{param})
		if err == nil {
			_ = json.Unmarshal([]byte(rawChainHelp), &chainHelp)
		}
	}
	if chainHelp != "" {
		return chainHelp, nil
	}
	return nil, &btcjson.RPCError{
		Code:    btcjson.ErrRPCInvalidParameter,
		Message: fmt.Sprintf("No help for method '%s'", *cmd.Command),
	}
}

//listaccounts通过返回帐户映射来处理listaccounts请求
//姓名与余额相符。
func listAccounts(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.ListAccountsCmd)

	accountBalances := map[string]float64{}
	results, err := w.AccountBalances(waddrmgr.KeyScopeBIP0044, int32(*cmd.MinConf))
	if err != nil {
		return nil, err
	}
	for _, result := range results {
		accountBalances[result.AccountName] = result.AccountBalance.ToBTC()
	}
//把地图还给我。这将被封送到JSON对象中。
	return accountBalances, nil
}

//listlockunspent通过返回
//所有锁定的输出点。
func listLockUnspent(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	return w.LockedOutpoints(), nil
}

//ListReceiveDByaCount通过返回来处理ListReceiveDByaCount请求
//对象切片，每个切片包含：
//“账户”：收款账户；
//“金额”：账户收到的总金额；
//“确认”：最近交易的确认数。
//它需要两个参数：
//“minconf”：考虑交易的最小确认数-
//默认值：1；
//“includeEmpty”：是否包括没有交易的地址-
//默认值：false。
func listReceivedByAccount(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.ListReceivedByAccountCmd)

	results, err := w.TotalReceivedForAccounts(
		waddrmgr.KeyScopeBIP0044, int32(*cmd.MinConf),
	)
	if err != nil {
		return nil, err
	}

	jsonResults := make([]btcjson.ListReceivedByAccountResult, 0, len(results))
	for _, result := range results {
		jsonResults = append(jsonResults, btcjson.ListReceivedByAccountResult{
			Account:       result.AccountName,
			Amount:        result.TotalReceived.ToBTC(),
			Confirmations: uint64(result.LastConfirmation),
		})
	}
	return jsonResults, nil
}

//lisreceivedbyaddress通过返回来处理lisreceivedbyaddress请求
//对象切片，每个切片包含：
//“账户”：收货地址的账户；
//“地址”：接收地址；
//“金额”：地址收到的总金额；
//“确认”：最近交易的确认数。
//它需要两个参数：
//“minconf”：考虑交易的最小确认数-
//默认值：1；
//“includeEmpty”：是否包括没有交易的地址-
//默认值：false。
func listReceivedByAddress(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.ListReceivedByAddressCmd)

//每个地址的中间数据。
	type AddrData struct {
//收到的总金额。
		amount btcutil.Amount
//上次交易的确认数。
		confirmations int32
//包含支付到地址的输出的事务散列
		tx []string
//地址所属帐户
		account string
	}

	syncBlock := w.Manager.SyncedTo()

//所有地址的中间数据。
	allAddrData := make(map[string]AddrData)
//为帐户中的每个活动地址创建一个addrdata条目。
//否则，我们稍后将从事务中获取地址。
	sortedAddrs, err := w.SortedActivePaymentAddresses()
	if err != nil {
		return nil, err
	}
	for _, address := range sortedAddrs {
//可能有重复项，只需覆盖它们即可。
		allAddrData[address] = AddrData{}
	}

	minConf := *cmd.MinConf
	var endHeight int32
	if minConf == 0 {
		endHeight = -1
	} else {
		endHeight = syncBlock.Height - int32(minConf) + 1
	}
	err = wallet.UnstableAPI(w).RangeTransactions(0, endHeight, func(details []wtxmgr.TxDetails) (bool, error) {
		confirmations := confirms(details[0].Block.Height, syncBlock.Height)
		for _, tx := range details {
			for _, cred := range tx.Credits {
				pkScript := tx.MsgTx.TxOut[cred.Index].PkScript
				_, addrs, _, err := txscript.ExtractPkScriptAddrs(
					pkScript, w.ChainParams())
				if err != nil {
//非标准脚本，跳过。
					continue
				}
				for _, addr := range addrs {
					addrStr := addr.EncodeAddress()
					addrData, ok := allAddrData[addrStr]
					if ok {
						addrData.amount += cred.Amount
//总是用新确认覆盖确认。
						addrData.confirmations = confirmations
					} else {
						addrData = AddrData{
							amount:        cred.Amount,
							confirmations: confirmations,
						}
					}
					addrData.tx = append(addrData.tx, tx.Hash.String())
					allAddrData[addrStr] = addrData
				}
			}
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}

//将地址数据按摩成输出格式。
	numAddresses := len(allAddrData)
	ret := make([]btcjson.ListReceivedByAddressResult, numAddresses, numAddresses)
	idx := 0
	for address, addrData := range allAddrData {
		ret[idx] = btcjson.ListReceivedByAddressResult{
			Address:       address,
			Amount:        addrData.amount.ToBTC(),
			Confirmations: uint64(addrData.confirmations),
			TxIDs:         addrData.tx,
		}
		idx++
	}
	return ret, nil
}

//listsinceblock通过返回映射数组来处理listsinceblock请求
//提供自给定数据块以来发送和接收的钱包交易的详细信息。
func listSinceBlock(icmd interface{}, w *wallet.Wallet, chainClient *chain.RPCClient) (interface{}, error) {
	cmd := icmd.(*btcjson.ListSinceBlockCmd)

	syncBlock := w.Manager.SyncedTo()
	targetConf := int64(*cmd.TargetConfirmations)

//对于结果，我们需要最后一个已计数的块的块散列
//在区块链中，由于确认。我们现在把这个寄出去
//它可以异步到达，而我们可以计算出其余的。
	gbh := chainClient.GetBlockHashAsync(int64(syncBlock.Height) + 1 - targetConf)

	var start int32
	if cmd.BlockHash != nil {
		hash, err := chainhash.NewHashFromStr(*cmd.BlockHash)
		if err != nil {
			return nil, DeserializationError{err}
		}
		block, err := chainClient.GetBlockVerboseTx(hash)
		if err != nil {
			return nil, err
		}
		start = int32(block.Height) + 1
	}

	txInfoList, err := w.ListSinceBlock(start, -1, syncBlock.Height)
	if err != nil {
		return nil, err
	}

//完成工作，得到回应。
	blockHash, err := gbh.Receive()
	if err != nil {
		return nil, err
	}

	res := btcjson.ListSinceBlockResult{
		Transactions: txInfoList,
		LastBlock:    blockHash.String(),
	}
	return res, nil
}

//ListTransactions通过返回
//包含已发送和已接收钱包交易详细信息的地图阵列。
func listTransactions(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.ListTransactionsCmd)

//TODO:ListTransactions当前不了解差异
//在与另一个账户有关的交易之间。这个
//将在wtxmgr与waddrmgr命名空间组合时解析。

	if cmd.Account != nil && *cmd.Account != "*" {
//现在，如果用户
//指定一个帐户，因为这不可能
//有效）计算。
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCWallet,
			Message: "Transactions are not yet grouped by account",
		}
	}

	return w.ListTransactions(*cmd.From, *cmd.Count)
}

//ListAddressTransactions处理ListAddressTransactions请求的依据
//返回一系列地图，其中包含已用和已收钱包的详细信息
//交易。回复的形式与ListTransactions相同，
//但数组元素仅限于事务细节
//关于请求中包含的addresess。
func listAddressTransactions(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.ListAddressTransactionsCmd)

	if cmd.Account != nil && *cmd.Account != "*" {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "Listing transactions for addresses may only be done for all accounts",
		}
	}

//解码地址。
	hash160Map := make(map[string]struct{})
	for _, addrStr := range cmd.Addresses {
		addr, err := decodeAddress(addrStr, w.ChainParams())
		if err != nil {
			return nil, err
		}
		hash160Map[string(addr.ScriptAddress())] = struct{}{}
	}

	return w.ListAddressTransactions(hash160Map)
}

//ListAllTransactions通过返回来处理ListAllTransactions请求
//带有已发送和已接收钱包交易详细信息的地图。这是
//类似于ListTransactions，只是它只需要一个可选的
//帐户名的参数和所有事务的答复。
func listAllTransactions(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.ListAllTransactionsCmd)

	if cmd.Account != nil && *cmd.Account != "*" {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "Listing all transactions may only be done for all accounts",
		}
	}

	return w.ListAllTransactions()
}

//listunspent处理listunspent命令。
func listUnspent(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.ListUnspentCmd)

	var addresses map[string]struct{}
	if cmd.Addresses != nil {
		addresses = make(map[string]struct{})
//确认它们都是好的：
		for _, as := range *cmd.Addresses {
			a, err := decodeAddress(as, w.ChainParams())
			if err != nil {
				return nil, err
			}
			addresses[a.EncodeAddress()] = struct{}{}
		}
	}

	return w.ListUnspent(int32(*cmd.MinConf), int32(*cmd.MaxConf), addresses)
}

//lockunspent处理lockunspent命令。
func lockUnspent(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.LockUnspentCmd)

	switch {
	case cmd.Unlock && len(cmd.Transactions) == 0:
		w.ResetLockedOutpoints()
	default:
		for _, input := range cmd.Transactions {
			txHash, err := chainhash.NewHashFromStr(input.Txid)
			if err != nil {
				return nil, ParseError{err}
			}
			op := wire.OutPoint{Hash: *txHash, Index: input.Vout}
			if cmd.Unlock {
				w.UnlockOutpoint(op)
			} else {
				w.LockOutpoint(op)
			}
		}
	}
	return true, nil
}

//makeoutputs从一对地址创建事务输出切片
//金额字符串。这用于创建新的
//从描述输出目标的JSON对象创建事务
//金额。
func makeOutputs(pairs map[string]btcutil.Amount, chainParams *chaincfg.Params) ([]*wire.TxOut, error) {
	outputs := make([]*wire.TxOut, 0, len(pairs))
	for addrStr, amt := range pairs {
		addr, err := btcutil.DecodeAddress(addrStr, chainParams)
		if err != nil {
			return nil, fmt.Errorf("cannot decode address: %s", err)
		}

		pkScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return nil, fmt.Errorf("cannot create txout script: %s", err)
		}

		outputs = append(outputs, wire.NewTxOut(int64(amt), pkScript))
	}
	return outputs, nil
}

//sendpairs创建并发送付款交易。
//成功后返回字符串格式的事务哈希
//所有错误都以btcjson.rpcerror格式返回
func sendPairs(w *wallet.Wallet, amounts map[string]btcutil.Amount,
	account uint32, minconf int32, feeSatPerKb btcutil.Amount) (string, error) {

	outputs, err := makeOutputs(amounts, w.ChainParams())
	if err != nil {
		return "", err
	}
	tx, err := w.SendOutputs(outputs, account, minconf, feeSatPerKb)
	if err != nil {
		if err == txrules.ErrAmountNegative {
			return "", ErrNeedPositiveAmount
		}
		if waddrmgr.IsError(err, waddrmgr.ErrLocked) {
			return "", &ErrWalletUnlockNeeded
		}
		switch err.(type) {
		case btcjson.RPCError:
			return "", err
		}

		return "", &btcjson.RPCError{
			Code:    btcjson.ErrRPCInternal.Code,
			Message: err.Error(),
		}
	}

	txHashStr := tx.TxHash().String()
	log.Infof("Successfully sent transaction %v", txHashStr)
	return txHashStr, nil
}

func isNilOrEmpty(s *string) bool {
	return s == nil || *s == ""
}

//sendfrom通过创建新事务处理sendfrom rpc请求
//将钱包的未用交易输出用于其他付款
//地址。未发送到付款地址的剩余输入或
//矿工被送回钱包里的新地址。成功后，
//将返回所创建事务的TxID。
func sendFrom(icmd interface{}, w *wallet.Wallet, chainClient *chain.RPCClient) (interface{}, error) {
	cmd := icmd.(*btcjson.SendFromCmd)

//尚不支持事务注释。错误而不是
//假装救他们。
	if !isNilOrEmpty(cmd.Comment) || !isNilOrEmpty(cmd.CommentTo) {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCUnimplemented,
			Message: "Transaction comments are not yet supported",
		}
	}

	account, err := w.AccountNumber(
		waddrmgr.KeyScopeBIP0044, cmd.FromAccount,
	)
	if err != nil {
		return nil, err
	}

//检查有符号整数参数是否为正。
	if cmd.Amount < 0 {
		return nil, ErrNeedPositiveAmount
	}
	minConf := int32(*cmd.MinConf)
	if minConf < 0 {
		return nil, ErrNeedPositiveMinconf
	}
//创建地址和金额对的映射。
	amt, err := btcutil.NewAmount(cmd.Amount)
	if err != nil {
		return nil, err
	}
	pairs := map[string]btcutil.Amount{
		cmd.ToAddress: amt,
	}

	return sendPairs(w, pairs, account, minConf,
		txrules.DefaultRelayFeePerKb)
}

//sendmany通过创建新事务处理sendmany RPC请求
//将钱包的未用交易输出花费到任意数量
//付款地址。未发送到付款地址的剩余输入
//或者，矿工的费用被送回钱包中的新地址。
//成功后，将返回所创建事务的TxID。
func sendMany(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.SendManyCmd)

//尚不支持事务注释。错误而不是
//假装救他们。
	if !isNilOrEmpty(cmd.Comment) {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCUnimplemented,
			Message: "Transaction comments are not yet supported",
		}
	}

	account, err := w.AccountNumber(waddrmgr.KeyScopeBIP0044, cmd.FromAccount)
	if err != nil {
		return nil, err
	}

//检查minconf是否为阳性。
	minConf := int32(*cmd.MinConf)
	if minConf < 0 {
		return nil, ErrNeedPositiveMinconf
	}

//使用dcrutil.amount重新创建地址/金额对。
	pairs := make(map[string]btcutil.Amount, len(cmd.Amounts))
	for k, v := range cmd.Amounts {
		amt, err := btcutil.NewAmount(v)
		if err != nil {
			return nil, err
		}
		pairs[k] = amt
	}

	return sendPairs(w, pairs, account, minConf, txrules.DefaultRelayFeePerKb)
}

//sendToAddress通过创建新的
//交易支出未使用的交易输出钱包给另一个
//付款地址。未发送到付款地址或费用的剩余输入
//因为矿工被送回钱包里的新地址。成功后，
//将返回所创建事务的TxID。
func sendToAddress(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.SendToAddressCmd)

//尚不支持事务注释。错误而不是
//假装救他们。
	if !isNilOrEmpty(cmd.Comment) || !isNilOrEmpty(cmd.CommentTo) {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCUnimplemented,
			Message: "Transaction comments are not yet supported",
		}
	}

	amt, err := btcutil.NewAmount(cmd.Amount)
	if err != nil {
		return nil, err
	}

//检查有符号整数参数是否为正。
	if amt < 0 {
		return nil, ErrNeedPositiveAmount
	}

//地址和金额对的模拟图。
	pairs := map[string]btcutil.Amount{
		cmd.Address: amt,
	}

//sendToAddress始终使用默认帐户，这与比特币匹配
	return sendPairs(w, pairs, waddrmgr.DefaultAccountNum, 1,
		txrules.DefaultRelayFeePerKb)
}

//setxfee设置添加到事务中的每千字节的事务费。
func setTxFee(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.SetTxFeeCmd)

//检查金额是否为负数。
	if cmd.Amount < 0 {
		return nil, ErrNeedPositiveAmount
	}

//成功后返回布尔真结果。
	return true, nil
}

//signmessage使用给定消息的私钥对给定消息进行签名
//地址
func signMessage(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.SignMessageCmd)

	addr, err := decodeAddress(cmd.Address, w.ChainParams())
	if err != nil {
		return nil, err
	}

	privKey, err := w.PrivKeyForAddress(addr)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	wire.WriteVarString(&buf, 0, "Bitcoin Signed Message:\n")
	wire.WriteVarString(&buf, 0, cmd.Message)
	messageHash := chainhash.DoubleHashB(buf.Bytes())
	sigbytes, err := btcec.SignCompact(btcec.S256(), privKey,
		messageHash, true)
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.EncodeToString(sigbytes), nil
}

//signrawtransaction处理signrawtransaction命令。
func signRawTransaction(icmd interface{}, w *wallet.Wallet, chainClient *chain.RPCClient) (interface{}, error) {
	cmd := icmd.(*btcjson.SignRawTransactionCmd)

	serializedTx, err := decodeHexStr(cmd.RawTx)
	if err != nil {
		return nil, err
	}
	var tx wire.MsgTx
	err = tx.Deserialize(bytes.NewBuffer(serializedTx))
	if err != nil {
		e := errors.New("TX decode failed")
		return nil, DeserializationError{e}
	}

	var hashType txscript.SigHashType
	switch *cmd.Flags {
	case "ALL":
		hashType = txscript.SigHashAll
	case "NONE":
		hashType = txscript.SigHashNone
	case "SINGLE":
		hashType = txscript.SigHashSingle
	case "ALL|ANYONECANPAY":
		hashType = txscript.SigHashAll | txscript.SigHashAnyOneCanPay
	case "NONE|ANYONECANPAY":
		hashType = txscript.SigHashNone | txscript.SigHashAnyOneCanPay
	case "SINGLE|ANYONECANPAY":
		hashType = txscript.SigHashSingle | txscript.SigHashAnyOneCanPay
	default:
		e := errors.New("Invalid sighash parameter")
		return nil, InvalidParameterError{e}
	}

//托多：真的吗？我们还是应该用BTCD查一下这些
//确保它们与区块链匹配（如果存在）。
	inputs := make(map[wire.OutPoint][]byte)
	scripts := make(map[string][]byte)
	var cmdInputs []btcjson.RawTxInput
	if cmd.Inputs != nil {
		cmdInputs = *cmd.Inputs
	}
	for _, rti := range cmdInputs {
		inputHash, err := chainhash.NewHashFromStr(rti.Txid)
		if err != nil {
			return nil, DeserializationError{err}
		}

		script, err := decodeHexStr(rti.ScriptPubKey)
		if err != nil {
			return nil, err
		}

//redeemscript仅在用户提供的情况下实际使用
//私钥。在这种情况下，它用于获取脚本
//签署。如果用户没有提供密钥，那么我们总是
//从钱包里拿剧本。
//空字符串适用于此字符串，十六进制解码字符串将
//DTRT。
		if cmd.PrivKeys != nil && len(*cmd.PrivKeys) != 0 {
			redeemScript, err := decodeHexStr(rti.RedeemScript)
			if err != nil {
				return nil, err
			}

			addr, err := btcutil.NewAddressScriptHash(redeemScript,
				w.ChainParams())
			if err != nil {
				return nil, DeserializationError{err}
			}
			scripts[addr.String()] = redeemScript
		}
		inputs[wire.OutPoint{
			Hash:  *inputHash,
			Index: rti.Vout,
		}] = script
	}

//现在我们去寻找没有提供的任何输入
//使用GetRawTransaction查询BTCD。我们排队等候一堆异步
//在我们检查完
//争论。
	requested := make(map[wire.OutPoint]rpcclient.FutureGetTxOutResult)
	for _, txIn := range tx.TxIn {
//我们从争论中得到了这个前哨点吗？
		if _, ok := inputs[txIn.PreviousOutPoint]; ok {
			continue
		}

//异步请求输出脚本。
		requested[txIn.PreviousOutPoint] = chainClient.GetTxOutAsync(
			&txIn.PreviousOutPoint.Hash, txIn.PreviousOutPoint.Index,
			true)
	}

//分析私钥列表（如果存在）。如果这里有钥匙的话
//它们是我们可以用来签名的钥匙。如果是空的，我们会的
//使用我们已知的任何钥匙。
	var keys map[string]*btcutil.WIF
	if cmd.PrivKeys != nil {
		keys = make(map[string]*btcutil.WIF)

		for _, key := range *cmd.PrivKeys {
			wif, err := btcutil.DecodeWIF(key)
			if err != nil {
				return nil, DeserializationError{err}
			}

			if !wif.IsForNet(w.ChainParams()) {
				s := "key network doesn't match wallet's"
				return nil, DeserializationError{errors.New(s)}
			}

			addr, err := btcutil.NewAddressPubKey(wif.SerializePubKey(),
				w.ChainParams())
			if err != nil {
				return nil, DeserializationError{err}
			}
			keys[addr.EncodeAddress()] = wif
		}
	}

//我们已经检查了其余的参数。现在我们可以收集异步
//TXS。托多：如果我们不介意浪费工作的可能性，我们可以
//移动等待到下面的循环，并稍微异步一点。
	for outPoint, resp := range requested {
		result, err := resp.Receive()
		if err != nil {
			return nil, err
		}
		script, err := hex.DecodeString(result.ScriptPubKey.Hex)
		if err != nil {
			return nil, err
		}
		inputs[outPoint] = script
	}

//已收集所有参数。现在我们可以签署所有我们可以签署的输入。
//“完成”表示我们成功地签署了所有输出，并且
//所有脚本将运行到完成。这是作为
//回答。
	signErrs, err := w.SignTransaction(&tx, hashType, inputs, keys, scripts)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.Grow(tx.SerializeSize())

//在
//字节。意外写入缓冲区。
	if err = tx.Serialize(&buf); err != nil {
		panic(err)
	}

	signErrors := make([]btcjson.SignRawTransactionError, 0, len(signErrs))
	for _, e := range signErrs {
		input := tx.TxIn[e.InputIndex]
		signErrors = append(signErrors, btcjson.SignRawTransactionError{
			TxID:      input.PreviousOutPoint.Hash.String(),
			Vout:      input.PreviousOutPoint.Index,
			ScriptSig: hex.EncodeToString(input.SignatureScript),
			Sequence:  input.Sequence,
			Error:     e.Error.Error(),
		})
	}

	return btcjson.SignRawTransactionResult{
		Hex:      hex.EncodeToString(buf.Bytes()),
		Complete: len(signErrors) == 0,
		Errors:   signErrors,
	}, nil
}

//validateAddress处理validateAddress命令。
func validateAddress(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.ValidateAddressCmd)

	result := btcjson.ValidateAddressWalletResult{}
	addr, err := decodeAddress(cmd.Address, w.ChainParams())
	if err != nil {
//使用结果零值（isvalid=false）。
		return result, nil
	}

//我们可以在这里输入地址是否为脚本，
//但是，通过检查“addr”的类型，
//如果脚本是
//“ismine”，我们遵循这种行为。
	result.Address = addr.EncodeAddress()
	result.IsValid = true

	ainfo, err := w.AddressInfo(addr)
	if err != nil {
		if waddrmgr.IsError(err, waddrmgr.ErrAddressNotFound) {
//没有关于地址的其他信息。
			return result, nil
		}
		return nil, err
	}

//地址查找成功，这意味着
//有关于它的信息，它是“我的”。
	result.IsMine = true
	acctName, err := w.AccountName(waddrmgr.KeyScopeBIP0044, ainfo.Account())
	if err != nil {
		return nil, &ErrAccountNameNotFound
	}
	result.Account = acctName

	switch ma := ainfo.(type) {
	case waddrmgr.ManagedPubKeyAddress:
		result.IsCompressed = ma.Compressed()
		result.PubKey = ma.ExportPubKey()

	case waddrmgr.ManagedScriptAddress:
		result.IsScript = true

//只有当管理器未锁定时，脚本才可用，因此
//如果有错误，现在就爆发吧。
		script, err := ma.Script()
		if err != nil {
			break
		}
		result.Hex = hex.EncodeToString(script)

//这通常不会失败，除非一个无效的脚本
//进口。但是，如果由于任何原因失败，则没有
//更多信息，所以只需设置脚本类型
//一个不标准的，现在爆发了。
		class, addrs, reqSigs, err := txscript.ExtractPkScriptAddrs(
			script, w.ChainParams())
		if err != nil {
			result.Script = txscript.NonStandardTy.String()
			break
		}

		addrStrings := make([]string, len(addrs))
		for i, a := range addrs {
			addrStrings[i] = a.EncodeAddress()
		}
		result.Addresses = addrStrings

//多签名脚本还提供所需的数量
//签名。
		result.Script = class.String()
		if class == txscript.MultiSigTy {
			result.SigsRequired = int32(reqSigs)
		}
	}

	return result, nil
}

//verifymessage通过验证提供的
//给定地址和消息的紧凑签名。
func verifyMessage(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.VerifyMessageCmd)

	addr, err := decodeAddress(cmd.Address, w.ChainParams())
	if err != nil {
		return nil, err
	}

//解码base64签名
	sig, err := base64.StdEncoding.DecodeString(cmd.Signature)
	if err != nil {
		return nil, err
	}

//验证签名-这表明它是有效的。
//我们将它与下一个键进行比较。
	var buf bytes.Buffer
	wire.WriteVarString(&buf, 0, "Bitcoin Signed Message:\n")
	wire.WriteVarString(&buf, 0, cmd.Message)
	expectedMessageHash := chainhash.DoubleHashB(buf.Bytes())
	pk, wasCompressed, err := btcec.RecoverCompact(btcec.S256(), sig,
		expectedMessageHash)
	if err != nil {
		return nil, err
	}

	var serializedPubKey []byte
	if wasCompressed {
		serializedPubKey = pk.SerializeCompressed()
	} else {
		serializedPubKey = pk.SerializeUncompressed()
	}
//
	switch checkAddr := addr.(type) {
case *btcutil.AddressPubKeyHash: //好啊
		return bytes.Equal(btcutil.Hash160(serializedPubKey), checkAddr.Hash160()[:]), nil
case *btcutil.AddressPubKey: //好啊
		return string(serializedPubKey) == checkAddr.String(), nil
	default:
		return nil, errors.New("address type not supported")
	}
}

//walletislocked处理walletislocked扩展请求
//返回当前锁定状态（解锁为假，锁定为真）
//记帐的
func walletIsLocked(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	return w.Locked(), nil
}

//walletlock通过锁定all帐户来处理walletlock请求
//钱包，如果钱包未加密，则返回错误（例如，
//只看电视的钱包）。
func walletLock(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	w.Lock()
	return nil, nil
}

//walletpassphrase通过解锁响应walletpassphrase请求
//钱包。解密密钥保存在钱包中直到超时
//秒后，钱包将被锁定。
func walletPassphrase(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.WalletPassphraseCmd)

	timeout := time.Second * time.Duration(cmd.Timeout)
	var unlockAfter <-chan time.Time
	if timeout != 0 {
		unlockAfter = time.After(timeout)
	}
	err := w.Unlock([]byte(cmd.Passphrase), unlockAfter)
	return nil, err
}

//walletpassphrasechange响应walletpassphrasechange请求
//通过使用提供的旧密码解锁所有帐户，以及
//使用从新密钥派生的AES密钥重新加密每个私钥
//口令。
//
//如果旧密码正确且密码已更改，则
//钱包将立即锁定。
func walletPassphraseChange(icmd interface{}, w *wallet.Wallet) (interface{}, error) {
	cmd := icmd.(*btcjson.WalletPassphraseChangeCmd)

	err := w.ChangePrivatePassphrase([]byte(cmd.OldPassphrase),
		[]byte(cmd.NewPassphrase))
	if waddrmgr.IsError(err, waddrmgr.ErrWrongPassphrase) {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCWalletPassphraseIncorrect,
			Message: "Incorrect passphrase",
		}
	}
	return nil, err
}

//decodeHexStr解码字符串的十六进制编码，可能在
//如果十六进制字符串中的字节数为奇数，则以“0”开头。
//这是为了防止使用奇数时十六进制字符串无效的错误。
//调用hex.decode时的字节数。
func decodeHexStr(hexStr string) ([]byte, error) {
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDecodeHexString,
			Message: "Hex string decode failed: " + err.Error(),
		}
	}
	return decoded, nil
}
