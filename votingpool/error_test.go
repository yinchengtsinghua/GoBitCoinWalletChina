
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
	"testing"

	vp "github.com/btcsuite/btcwallet/votingpool"
)

//TesterRorCodeStringer测试所有错误代码都有文本
//表示法和文本表示法仍然正确，
//即，错误代码的重构和重命名没有
//偏离了文本的表述。
func TestErrorCodeStringer(t *testing.T) {
//所有的错误
	tests := []struct {
		in   vp.ErrorCode
		want string
	}{
		{vp.ErrInputSelection, "ErrInputSelection"},
		{vp.ErrWithdrawalProcessing, "ErrWithdrawalProcessing"},
		{vp.ErrUnknownPubKey, "ErrUnknownPubKey"},
		{vp.ErrSeriesSerialization, "ErrSeriesSerialization"},
		{vp.ErrSeriesVersion, "ErrSeriesVersion"},
		{vp.ErrSeriesNotExists, "ErrSeriesNotExists"},
		{vp.ErrSeriesAlreadyExists, "ErrSeriesAlreadyExists"},
		{vp.ErrSeriesAlreadyEmpowered, "ErrSeriesAlreadyEmpowered"},
		{vp.ErrSeriesIDNotSequential, "ErrSeriesIDNotSequential"},
		{vp.ErrSeriesIDInvalid, "ErrSeriesIDInvalid"},
		{vp.ErrSeriesNotActive, "ErrSeriesNotActive"},
		{vp.ErrKeyIsPrivate, "ErrKeyIsPrivate"},
		{vp.ErrKeyIsPublic, "ErrKeyIsPublic"},
		{vp.ErrKeyNeuter, "ErrKeyNeuter"},
		{vp.ErrKeyMismatch, "ErrKeyMismatch"},
		{vp.ErrKeysPrivatePublicMismatch, "ErrKeysPrivatePublicMismatch"},
		{vp.ErrKeyDuplicate, "ErrKeyDuplicate"},
		{vp.ErrTooFewPublicKeys, "ErrTooFewPublicKeys"},
		{vp.ErrPoolAlreadyExists, "ErrPoolAlreadyExists"},
		{vp.ErrPoolNotExists, "ErrPoolNotExists"},
		{vp.ErrScriptCreation, "ErrScriptCreation"},
		{vp.ErrTooManyReqSignatures, "ErrTooManyReqSignatures"},
		{vp.ErrInvalidBranch, "ErrInvalidBranch"},
		{vp.ErrInvalidValue, "ErrInvalidValue"},
		{vp.ErrDatabase, "ErrDatabase"},
		{vp.ErrKeyChain, "ErrKeyChain"},
		{vp.ErrCrypto, "ErrCrypto"},
		{vp.ErrRawSigning, "ErrRawSigning"},
		{vp.ErrPreconditionNotMet, "ErrPreconditionNotMet"},
		{vp.ErrTxSigning, "ErrTxSigning"},
		{vp.ErrInvalidScriptHash, "ErrInvalidScriptHash"},
		{vp.ErrWithdrawFromUnusedAddr, "ErrWithdrawFromUnusedAddr"},
		{vp.ErrWithdrawalTxStorage, "ErrWithdrawalTxStorage"},
		{vp.ErrWithdrawalStorage, "ErrWithdrawalStorage"},
		{0xffff, "Unknown ErrorCode (65535)"},
	}

	if int(vp.TstLastErr) != len(tests)-1 {
		t.Errorf("Wrong number of errorCodeStrings. Got: %d, want: %d",
			int(vp.TstLastErr), len(tests))
	}

	for i, test := range tests {
		result := test.in.String()
		if result != test.want {
			t.Errorf("String #%d\ngot: %s\nwant: %s", i, result,
				test.want)
		}
	}
}
