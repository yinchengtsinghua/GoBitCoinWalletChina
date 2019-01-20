
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

package waddrmgr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/btcsuite/btcwallet/waddrmgr"
)

//TesterRorCodeStringer测试错误代码类型的字符串化输出。
func TestErrorCodeStringer(t *testing.T) {
	tests := []struct {
		in   waddrmgr.ErrorCode
		want string
	}{
		{waddrmgr.ErrDatabase, "ErrDatabase"},
		{waddrmgr.ErrUpgrade, "ErrUpgrade"},
		{waddrmgr.ErrKeyChain, "ErrKeyChain"},
		{waddrmgr.ErrCrypto, "ErrCrypto"},
		{waddrmgr.ErrInvalidKeyType, "ErrInvalidKeyType"},
		{waddrmgr.ErrNoExist, "ErrNoExist"},
		{waddrmgr.ErrAlreadyExists, "ErrAlreadyExists"},
		{waddrmgr.ErrCoinTypeTooHigh, "ErrCoinTypeTooHigh"},
		{waddrmgr.ErrAccountNumTooHigh, "ErrAccountNumTooHigh"},
		{waddrmgr.ErrLocked, "ErrLocked"},
		{waddrmgr.ErrWatchingOnly, "ErrWatchingOnly"},
		{waddrmgr.ErrInvalidAccount, "ErrInvalidAccount"},
		{waddrmgr.ErrAddressNotFound, "ErrAddressNotFound"},
		{waddrmgr.ErrAccountNotFound, "ErrAccountNotFound"},
		{waddrmgr.ErrDuplicateAddress, "ErrDuplicateAddress"},
		{waddrmgr.ErrDuplicateAccount, "ErrDuplicateAccount"},
		{waddrmgr.ErrTooManyAddresses, "ErrTooManyAddresses"},
		{waddrmgr.ErrWrongPassphrase, "ErrWrongPassphrase"},
		{waddrmgr.ErrWrongNet, "ErrWrongNet"},
		{waddrmgr.ErrCallBackBreak, "ErrCallBackBreak"},
		{waddrmgr.ErrEmptyPassphrase, "ErrEmptyPassphrase"},
		{0xffff, "Unknown ErrorCode (65535)"},
	}
	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.String()
		if result != test.want {
			t.Errorf("String #%d\ngot: %s\nwant: %s", i, result,
				test.want)
			continue
		}
	}
}

//
func TestManagerError(t *testing.T) {
	tests := []struct {
		in   waddrmgr.ManagerError
		want string
	}{
//
		{
			waddrmgr.ManagerError{Description: "human-readable error"},
			"human-readable error",
		},

//封装数据库错误。
		{
			waddrmgr.ManagerError{
				Description: "failed to store master private " +
					"key parameters",
				ErrorCode: waddrmgr.ErrDatabase,
				Err:       fmt.Errorf("underlying db error"),
			},
			"failed to store master private key parameters: " +
				"underlying db error",
		},

//
		{
			waddrmgr.ManagerError{
				Description: "failed to derive extended key " +
					"branch 0",
				ErrorCode: waddrmgr.ErrKeyChain,
				Err:       fmt.Errorf("underlying error"),
			},
			"failed to derive extended key branch 0: underlying " +
				"error",
		},

//
		{
			waddrmgr.ManagerError{
				Description: "failed to decrypt account 0 " +
					"private key",
				ErrorCode: waddrmgr.ErrCrypto,
				Err:       fmt.Errorf("underlying error"),
			},
			"failed to decrypt account 0 private key: underlying " +
				"error",
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.Error()
		if result != test.want {
			t.Errorf("Error #%d\ngot: %s\nwant: %s", i, result,
				test.want)
			continue
		}
	}
}

//测试错误测试ISerror函数。
func TestIsError(t *testing.T) {
	tests := []struct {
		err  error
		code waddrmgr.ErrorCode
		exp  bool
	}{
		{
			err: waddrmgr.ManagerError{
				ErrorCode: waddrmgr.ErrDatabase,
			},
			code: waddrmgr.ErrDatabase,
			exp:  true,
		},
		{
//
			err: &waddrmgr.ManagerError{
				ErrorCode: waddrmgr.ErrDatabase,
			},
			code: waddrmgr.ErrDatabase,
			exp:  false,
		},
		{
			err: waddrmgr.ManagerError{
				ErrorCode: waddrmgr.ErrCrypto,
			},
			code: waddrmgr.ErrDatabase,
			exp:  false,
		},
		{
			err:  errors.New("not a ManagerError"),
			code: waddrmgr.ErrDatabase,
			exp:  false,
		},
	}

	for i, test := range tests {
		got := waddrmgr.IsError(test.err, test.code)
		if got != test.exp {
			t.Errorf("Test %d: got %v expected %v", i, got, test.exp)
		}
	}
}
