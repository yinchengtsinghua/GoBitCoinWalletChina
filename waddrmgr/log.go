
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package waddrmgr

import "github.com/btcsuite/btclog"

//日志是一个没有输出过滤器初始化的日志程序。这个
//意味着在调用方之前，包默认不会执行任何日志记录
//请求它。
var log btclog.Logger

//默认的日志记录量为“无”。
func init() {
	DisableLog()
}

//DisableLog禁用所有库日志输出。日志记录输出被禁用
//默认情况下，直到调用uselogger或setlogwriter。
func DisableLog() {
	UseLogger(btclog.Disabled)
}

//uselogger使用指定的记录器输出包日志信息。
//如果调用方还
//使用BTCULG。
func UseLogger(logger btclog.Logger) {
	log = logger
}

//LogClosing是一个可以用%v打印的闭包，用于
//为详细的日志级别创建数据并避免
//数据未打印时的工作。
type logClosure func() string

//字符串调用日志闭包并返回结果字符串。
func (c logClosure) String() string {
	return c()
}

//newlogclosure返回传递函数的新闭包，该函数允许
//在日志函数中用作参数，该函数仅在
//日志级别是这样的，消息将被实际记录。
func newLogClosure(c func() string) logClosure {
	return logClosure(c)
}
