
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
	"os"
	"reflect"
	"runtime"
	"testing"

	"github.com/btcsuite/btclog"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/walletdb"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

//启用日志记录（调试级别）以帮助调试失败的测试。
	logger := btclog.NewBackend(os.Stdout).Logger("TEST")
	logger.SetLevel(btclog.LevelDebug)
	UseLogger(logger)
}

//TSTCheckError确保传递的错误是VotingPool。出错时出错
//与传递的错误代码匹配的代码。
func TstCheckError(t *testing.T, testName string, gotErr error, wantErrCode ErrorCode) {
	vpErr, ok := gotErr.(Error)
	if !ok {
		t.Errorf("%s: unexpected error type - got %T (%s), want %T",
			testName, gotErr, gotErr, Error{})
	}
	if vpErr.ErrorCode != wantErrCode {
		t.Errorf("%s: unexpected error code - got %s (%s), want %s",
			testName, vpErr.ErrorCode, vpErr, wantErrCode)
	}
}

//tstrunwithmanagerunlocked在管理器解锁的情况下调用给定的回调，
//回来前再锁上。
func TstRunWithManagerUnlocked(t *testing.T, mgr *waddrmgr.Manager, addrmgrNs walletdb.ReadBucket, callback func()) {
	if err := mgr.Unlock(addrmgrNs, privPassphrase); err != nil {
		t.Fatal(err)
	}
	defer mgr.Lock()
	callback()
}

//tstcheckdrawintalstatusmatches使用reflect.deepequal比较s1和s2
//如果它们不相同，则调用t.fatal（）。
func TstCheckWithdrawalStatusMatches(t *testing.T, s1, s2 WithdrawalStatus) {
	if s1.Fees() != s2.Fees() {
		t.Fatalf("Wrong amount of network fees; want %d, got %d", s1.Fees(), s2.Fees())
	}

	if !reflect.DeepEqual(s1.Sigs(), s2.Sigs()) {
		t.Fatalf("Wrong tx signatures; got %x, want %x", s1.Sigs(), s2.Sigs())
	}

	if !reflect.DeepEqual(s1.NextInputAddr(), s2.NextInputAddr()) {
		t.Fatalf("Wrong NextInputAddr; got %v, want %v", s1.NextInputAddr(), s2.NextInputAddr())
	}

	if !reflect.DeepEqual(s1.NextChangeAddr(), s2.NextChangeAddr()) {
		t.Fatalf("Wrong NextChangeAddr; got %v, want %v", s1.NextChangeAddr(), s2.NextChangeAddr())
	}

	if !reflect.DeepEqual(s1.Outputs(), s2.Outputs()) {
		t.Fatalf("Wrong WithdrawalOutputs; got %v, want %v", s1.Outputs(), s2.Outputs())
	}

	if !reflect.DeepEqual(s1.transactions, s2.transactions) {
		t.Fatalf("Wrong transactions; got %v, want %v", s1.transactions, s2.transactions)
	}

//上述支票可以用这张支票代替，但如果支票不合格
//失败消息不会给我们太多关于什么是不平等的线索，所以我们这样做。
//个人在上面检查，并将此检查用作“全部捕获”检查，以防
//我们忘记检查任何单个字段。
	if !reflect.DeepEqual(s1, s2) {
		t.Fatalf("Wrong WithdrawalStatus; got %v, want %v", s1, s2)
	}
}
