
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package wallet_test

import (
	"runtime"
	"testing"

	"github.com/btcsuite/btcwallet/wallet"
)

//Harness保存正在测试的BranchRecoveryState，恢复窗口为
//使用，提供对测试对象的访问，并跟踪预期范围
//以及下一个未找到的值。
type Harness struct {
	t              *testing.T
	brs            *wallet.BranchRecoveryState
	recoveryWindow uint32
	expHorizon     uint32
	expNextUnfound uint32
}

type (
//步进器是执行操作或断言的通用接口。
//测试线束。
	Stepper interface {
//应用对分支恢复执行操作或断言
//由安全带控制的状态。步骤索引是这样提供的
//任何失败都可以报告哪个步骤失败。
		Apply(step int, harness *Harness)
	}

//InitialDelta是验证我们第一次尝试扩展
//分支恢复状态的视界告诉我们
//添加与恢复窗口相等的连衣裙。
	InitialDelta struct{}

//checkdelta是一个扩展分支恢复状态的步骤
//地平线，检查返回的三角洲是否符合我们的预期
//
	CheckDelta struct {
		delta uint32
	}

//
//state reports `total` invalid children with the current horizon.
	CheckNumInvalid struct {
		total uint32
	}

//MarkInvalid is a Step that marks the `child` as invalid in the branch
//恢复状态。
	MarkInvalid struct {
		child uint32
	}

//ReportFound is a Step that reports `child` as being found to the
//branch recovery state.
	ReportFound struct {
		child uint32
	}
)

//应用扩展分支恢复状态的当前视界，并检查
//that the returned delta is equal to the test's recovery window. 如果
//assertions pass, the harness's expected horizon is increased by the returned
//三角洲。
//
//注意：这应该在应用任何checkdelta步骤之前使用。
func (_ InitialDelta) Apply(i int, h *Harness) {
	curHorizon, delta := h.brs.ExtendHorizon()
	assertHorizon(h.t, i, curHorizon, h.expHorizon)
	assertDelta(h.t, i, delta, h.recoveryWindow)
	h.expHorizon += delta
}

//应用扩展分支恢复状态的当前视界，并检查
//返回的delta等于checkdelta的子值。
func (d CheckDelta) Apply(i int, h *Harness) {
	curHorizon, delta := h.brs.ExtendHorizon()
	assertHorizon(h.t, i, curHorizon, h.expHorizon)
	assertDelta(h.t, i, delta, d.delta)
	h.expHorizon += delta
}

//apply查询分支恢复状态中无效子级的数目
//that lie between the last found address and the current horizon, and compares
//
func (m CheckNumInvalid) Apply(i int, h *Harness) {
	assertNumInvalid(h.t, i, h.brs.NumInvalidInHorizon(), m.total)
}

//Apply marks the MarkInvalid's child index as invalid in the branch recovery
//状态，并增加线束的预期范围。
func (m MarkInvalid) Apply(i int, h *Harness) {
	h.brs.MarkInvalidChild(m.child)
	h.expHorizon++
}

//应用报告在分支恢复中找到的reportfound的子索引
//状态。If the child index meets or exceeds our expected next unfound value,
//预期值将被修改为子索引+1。之后，
//this step asserts that the branch recovery state's next reported unfound
//value matches our potentially-updated value.
func (r ReportFound) Apply(i int, h *Harness) {
	h.brs.ReportFound(r.child)
	if r.child >= h.expNextUnfound {
		h.expNextUnfound = r.child + 1
	}
	assertNextUnfound(h.t, i, h.brs.NextUnfound(), h.expNextUnfound)
}

//编译时检查以确保我们的步骤实现步骤接口。
var _ Stepper = InitialDelta{}
var _ Stepper = CheckDelta{}
var _ Stepper = CheckNumInvalid{}
var _ Stepper = MarkInvalid{}
var _ Stepper = ReportFound{}

//TestBranchRecoveryState walks the BranchRecoveryState through a sequence of
//步骤，验证：
//- the horizon is properly expanded in response to found addrs
//-报告发现低于或等于先前找到的子级不会导致更改
//-标记无效的子项可扩展范围
func TestBranchRecoveryState(t *testing.T) {

	const recoveryWindow = 10

	recoverySteps := []Stepper{
//首先，检查一下扩大我们的视野是否能准确地返回
//恢复窗口（10）。
		InitialDelta{},

//预期范围：10。

//报告找到第二个地址，这将导致我们的地平线
//扩大2。
		ReportFound{1},
		CheckDelta{2},

//预期范围：12。

//健全性检查，再次扩展报告零增量，如
//什么都没有改变。
		CheckDelta{0},

//现在，报告查找第6个地址，它应该扩展我们的
//地平线到16，Detla为4。
		ReportFound{5},
		CheckDelta{4},

//预期地平线：16。

//健全性检查，再次扩展报告零增量，如
//什么都没有改变。
		CheckDelta{0},

//再次查找子索引5，没有任何更改。
		ReportFound{5},
		CheckDelta{0},

//报告发现一个较低的索引，
//什么都不应该改变。
		ReportFound{4},
		CheckDelta{0},

//继续，报告查找第11个地址，应将其扩展
//我们的地平线是21。
		ReportFound{10},
		CheckDelta{5},

//预期范围：21。

//在测试lookahead扩展之前
//无效的子密钥，请检查我们是否正确地从
//没有无效的密钥。
		CheckNumInvalid{0},

//现在窗口已展开，请模拟派生
//为所派生的加法器范围内的键无效
//第一次。地平线将增加1，因为
//恢复管理器应尝试并至少派生
//下一个地址。
		MarkInvalid{17},
		CheckNumInvalid{1},
		CheckDelta{0},

//预期地平线：22。

//检查派生第二个无效键是否显示这两个键都无效
//当前在范围内的索引。
		MarkInvalid{18},
		CheckNumInvalid{2},
		CheckDelta{0},

//预期地平线：23。

//最后，在我们的两个地址之后立即报告查找地址
//无效的密钥。这将返回我们的无效密钥数
//在地平线上回到0。
		ReportFound{19},
		CheckNumInvalid{0},

//因为第20把钥匙刚刚被标记出来，我们的地平线需要
//扩大到30。地平线23度，三角洲返回
//应该是7。
		CheckDelta{7},
		CheckDelta{0},

//预期范围：30。
	}

	brs := wallet.NewBranchRecoveryState(recoveryWindow)
	harness := &Harness{
		t:              t,
		brs:            brs,
		recoveryWindow: recoveryWindow,
	}

	for i, step := range recoverySteps {
		step.Apply(i, harness)
	}
}

func assertHorizon(t *testing.T, i int, have, want uint32) {
	assertHaveWant(t, i, "incorrect horizon", have, want)
}

func assertDelta(t *testing.T, i int, have, want uint32) {
	assertHaveWant(t, i, "incorrect delta", have, want)
}

func assertNextUnfound(t *testing.T, i int, have, want uint32) {
	assertHaveWant(t, i, "incorrect next unfound", have, want)
}

func assertNumInvalid(t *testing.T, i int, have, want uint32) {
	assertHaveWant(t, i, "incorrect num invalid children", have, want)
}

func assertHaveWant(t *testing.T, i int, msg string, have, want uint32) {
	_, _, line, _ := runtime.Caller(2)
	if want != have {
		t.Fatalf("[line: %d, step: %d] %s: got %d, want %d",
			line, i, msg, have, want)
	}
}
