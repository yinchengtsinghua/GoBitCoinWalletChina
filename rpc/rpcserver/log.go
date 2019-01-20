
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2016 BTCSuite开发者
//
//使用、复制、修改和分发本软件的权限
//特此授予免费或不收费的目的，前提是
//版权声明和本许可声明出现在所有副本中。
//
//本软件按“原样”提供，作者不作任何保证。
//关于本软件，包括
//适销性和适用性。在任何情况下，作者都不对
//任何特殊、直接、间接或后果性损害或任何损害
//因使用、数据或利润损失而导致的任何情况，无论是在
//合同行为、疏忽或其他侵权行为
//或与本软件的使用或性能有关。

package rpcserver

import (
	"os"
	"strings"

	"google.golang.org/grpc/grpclog"

	"github.com/btcsuite/btclog"
)

//use logger设置要用于GRPC服务器的记录器。
func UseLogger(l btclog.Logger) {
	grpclog.SetLogger(logger{l})
}

//logger使用btclog.logger来实现grpclog.logger接口。
type logger struct {
	btclog.Logger
}

//stripgrpcprefix删除对grpc的所有日志的包前缀
//记录器，因为这些已经作为btclog子系统名称包含在内。
func stripGrpcPrefix(logstr string) string {
	return strings.TrimPrefix(logstr, "grpc: ")
}

//stripgrpcprefixargs从第一个参数中删除包前缀，如果它
//存在且是一个字符串，在重新分配
//第一个ARG。
func stripGrpcPrefixArgs(args ...interface{}) []interface{} {
	if len(args) == 0 {
		return args
	}
	firstArgStr, ok := args[0].(string)
	if ok {
		args[0] = stripGrpcPrefix(firstArgStr)
	}
	return args
}

func (l logger) Fatal(args ...interface{}) {
	l.Critical(stripGrpcPrefixArgs(args)...)
	os.Exit(1)
}

func (l logger) Fatalf(format string, args ...interface{}) {
	l.Criticalf(stripGrpcPrefix(format), args...)
	os.Exit(1)
}

func (l logger) Fatalln(args ...interface{}) {
	l.Critical(stripGrpcPrefixArgs(args)...)
	os.Exit(1)
}

func (l logger) Print(args ...interface{}) {
	l.Info(stripGrpcPrefixArgs(args)...)
}

func (l logger) Printf(format string, args ...interface{}) {
	l.Infof(stripGrpcPrefix(format), args...)
}

func (l logger) Println(args ...interface{}) {
	l.Info(stripGrpcPrefixArgs(args)...)
}
