
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

package cfgutil

import (
	"strconv"
	"strings"

	"github.com/btcsuite/btcutil"
)

//amountFlag嵌入bcutil.amount并实现Flags.Marshaler和
//取消标记接口，以便它可以用作配置结构字段。
type AmountFlag struct {
	btcutil.Amount
}

//newamountflag使用默认btcutil.amount创建amountflag。
func NewAmountFlag(defaultValue btcutil.Amount) *AmountFlag {
	return &AmountFlag{defaultValue}
}

//MarshalFlag满足Flags.Marshaler接口。
func (a *AmountFlag) MarshalFlag() (string, error) {
	return a.Amount.String(), nil
}

//unmarshalflag满足标志。unmarshaller接口。
func (a *AmountFlag) UnmarshalFlag(value string) error {
	value = strings.TrimSuffix(value, " BTC")
	valueF64, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}
	amount, err := btcutil.NewAmount(valueF64)
	if err != nil {
		return err
	}
	a.Amount = amount
	return nil
}
