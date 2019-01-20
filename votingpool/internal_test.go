
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

package votingpool

import (
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/walletdb"
)

var TstLastErr = lastErr

const TstEligibleInputMinConfirmations = eligibleInputMinConfirmations

//tstputseries透明地包装投票池putseries方法。
func (vp *Pool) TstPutSeries(ns walletdb.ReadWriteBucket, version, seriesID, reqSigs uint32, inRawPubKeys []string) error {
	return vp.putSeries(ns, version, seriesID, reqSigs, inRawPubKeys)
}

var TstBranchOrder = branchOrder

//tsexistsseries检查序列是否存储在数据库中。
func (vp *Pool) TstExistsSeries(dbtx walletdb.ReadTx, seriesID uint32) (bool, error) {
	ns, _ := TstRNamespaces(dbtx)
	poolBucket := ns.NestedReadBucket(vp.ID)
	if poolBucket == nil {
		return false, nil
	}
	bucket := poolBucket.NestedReadBucket(seriesBucketName)
	if bucket == nil {
		return false, nil
	}
	return bucket.Get(uint32ToBytes(seriesID)) != nil, nil
}

//tstGetRawPublickeys获取字符串格式的序列公钥。
func (s *SeriesData) TstGetRawPublicKeys() []string {
	rawKeys := make([]string, len(s.publicKeys))
	for i, key := range s.publicKeys {
		rawKeys[i] = key.String()
	}
	return rawKeys
}

//tstgetrawprivatekeys获取字符串格式的系列私钥。
func (s *SeriesData) TstGetRawPrivateKeys() []string {
	rawKeys := make([]string, len(s.privateKeys))
	for i, key := range s.privateKeys {
		if key != nil {
			rawKeys[i] = key.String()
		}
	}
	return rawKeys
}

//tstgetreqsigs公开series reqsigs属性。
func (s *SeriesData) TstGetReqSigs() uint32 {
	return s.reqSigs
}

//tstEmptySriesLookup清空投票池序列查找属性。
func (vp *Pool) TstEmptySeriesLookup() {
	vp.seriesLookup = make(map[uint32]*SeriesData)
}

//tstdecryptextendedkey公开decryptextendedkey方法。
func (vp *Pool) TstDecryptExtendedKey(keyType waddrmgr.CryptoKeyType, encrypted []byte) (*hdkeychain.ExtendedKey, error) {
	return vp.decryptExtendedKey(keyType, encrypted)
}

//tstgetmsgtx返回具有给定的
//NTXID。
func (s *WithdrawalStatus) TstGetMsgTx(ntxid Ntxid) *wire.MsgTx {
	return s.transactions[ntxid].MsgTx.Copy()
}
