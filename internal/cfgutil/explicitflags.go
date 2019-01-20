
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package cfgutil

//explicitString是实现Flags.Marshaler和
//标记。取消标记接口，以便它可以用作配置结构字段。它
//记录该值是否由Flags包显式设置。这是
//当必须根据标志是否由
//用户或保留为默认值。如果不录下来的话
//无法确定带有默认值的标志是否未被修改或
//显式设置为默认值。
type ExplicitString struct {
	Value         string
	explicitlySet bool
}

//newExplicitString使用提供的默认值创建字符串标志。
func NewExplicitString(defaultValue string) *ExplicitString {
	return &ExplicitString{Value: defaultValue, explicitlySet: false}
}

//explicitly set返回标志是否通过
//flags.unmasheler接口。
func (e *ExplicitString) ExplicitlySet() bool { return e.explicitlySet }

//MarshalFlag实现Flags.Marshaler接口。
func (e *ExplicitString) MarshalFlag() (string, error) { return e.Value, nil }

//UnmarshalFlag实现标志。Unmarshaller接口。
func (e *ExplicitString) UnmarshalFlag(value string) error {
	e.Value = value
	e.explicitlySet = true
	return nil
}
