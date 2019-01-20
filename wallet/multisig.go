
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2017 BTCSuite开发者
//版权所有（c）2016版权所有
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wallet

import (
	"errors"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/walletdb"
)

//makemultisigscript创建一个可以用
//传递的密钥和地址的必需签名。如果地址是
//p2pkh地址，如有可能，钱包会查找相关的公钥，
//否则，会为缺少的pubkey返回错误。
//
//
func (w *Wallet) MakeMultiSigScript(addrs []btcutil.Address, nRequired int) ([]byte, error) {
	pubKeys := make([]*btcutil.AddressPubKey, len(addrs))

	var dbtx walletdb.ReadTx
	var addrmgrNs walletdb.ReadBucket
	defer func() {
		if dbtx != nil {
			dbtx.Rollback()
		}
	}()

//地址列表将由addreseses（pubkey散列）中的任意一个组成，用于
//我们需要查钱包里的钥匙，直接的钥匙，或者
//两者的混合物。
	for i, addr := range addrs {
		switch addr := addr.(type) {
		default:
			return nil, errors.New("cannot make multisig script for " +
				"a non-secp256k1 public key or P2PKH address")

		case *btcutil.AddressPubKey:
			pubKeys[i] = addr

		case *btcutil.AddressPubKeyHash:
			if dbtx == nil {
				var err error
				dbtx, err = w.db.BeginReadTx()
				if err != nil {
					return nil, err
				}
				addrmgrNs = dbtx.ReadBucket(waddrmgrNamespaceKey)
			}
			addrInfo, err := w.Manager.Address(addrmgrNs, addr)
			if err != nil {
				return nil, err
			}
			serializedPubKey := addrInfo.(waddrmgr.ManagedPubKeyAddress).
				PubKey().SerializeCompressed()

			pubKeyAddr, err := btcutil.NewAddressPubKey(
				serializedPubKey, w.chainParams)
			if err != nil {
				return nil, err
			}
			pubKeys[i] = pubKeyAddr
		}
	}

	return txscript.MultiSigScript(pubKeys, nRequired)
}

//importp2shredeemscript将p2sh兑换脚本添加到钱包中。
func (w *Wallet) ImportP2SHRedeemScript(script []byte) (*btcutil.AddressScriptHash, error) {
	var p2shAddr *btcutil.AddressScriptHash
	err := walletdb.Update(w.db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)

//todo（oga）blockstamp当前块？
		bs := &waddrmgr.BlockStamp{
			Hash:   *w.ChainParams().GenesisHash,
			Height: 0,
		}

//因为这是一个普通的p2sh脚本，所以我们将把它导入
//BIP044范围。
		bip44Mgr, err := w.Manager.FetchScopedKeyManager(
			waddrmgr.KeyScopeBIP0084,
		)
		if err != nil {
			return err
		}

		addrInfo, err := bip44Mgr.ImportScript(addrmgrNs, script, bs)
		if err != nil {
//不在乎它是否已经在那里，但仍然必须
//设置p2shaddr，因为地址管理器没有
//归还任何有用的东西。
			if waddrmgr.IsError(err, waddrmgr.ErrDuplicateAddress) {
//这个函数永远不会出错
//将脚本散列到正确的长度。
				p2shAddr, _ = btcutil.NewAddressScriptHash(script,
					w.chainParams)
				return nil
			}
			return err
		}

		p2shAddr = addrInfo.Address().(*btcutil.AddressScriptHash)
		return nil
	})
	return p2shAddr, err
}
