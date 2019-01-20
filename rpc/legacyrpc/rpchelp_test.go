
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

package legacyrpc

import (
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcwallet/internal/rpchelp"
)

func serverMethods() map[string]struct{} {
	m := make(map[string]struct{})
	for method, handlerData := range rpcHandlers {
		if !handlerData.noHelp {
			m[method] = struct{}{}
		}
	}
	return m
}

//TestRpcMethodHelpGeneration确保可以为每个
//每个支持的区域设置的RPC服务器的方法。
func TestRPCMethodHelpGeneration(t *testing.T) {
	needsGenerate := false

	defer func() {
		if needsGenerate && !t.Failed() {
			t.Error("Generated help texts are out of date: run 'go generate'")
			return
		}
		if t.Failed() {
			t.Log("Regenerate help texts with 'go generate' after fixing")
		}
	}()

	for i := range rpchelp.HelpDescs {
		svrMethods := serverMethods()
		locale := rpchelp.HelpDescs[i].Locale
		generatedDescs := localeHelpDescs[locale]()
		for _, m := range rpchelp.Methods {
			delete(svrMethods, m.Method)

			helpText, err := btcjson.GenerateHelp(m.Method, rpchelp.HelpDescs[i].Descs, m.ResultTypes...)
			if err != nil {
				t.Errorf("Cannot generate '%s' help for method '%s': missing description for '%s'",
					locale, m.Method, err)
				continue
			}
			if !needsGenerate && helpText != generatedDescs[m.Method] {
				needsGenerate = true
			}
		}

		for m := range svrMethods {
			t.Errorf("Missing '%s' help for method '%s'", locale, m)
		}
	}
}

//testrpcmethodusagegeneration确保单行用法文本可以
//为RPC服务器的每个支持请求生成。
func TestRPCMethodUsageGeneration(t *testing.T) {
	needsGenerate := false

	defer func() {
		if needsGenerate && !t.Failed() {
			t.Error("Generated help usages are out of date: run 'go generate'")
			return
		}
		if t.Failed() {
			t.Log("Regenerate help usage with 'go generate' after fixing")
		}
	}()

	svrMethods := serverMethods()
	usageStrs := make([]string, 0, len(rpchelp.Methods))
	for _, m := range rpchelp.Methods {
		delete(svrMethods, m.Method)

		usage, err := btcjson.MethodUsageText(m.Method)
		if err != nil {
			t.Errorf("Cannot generate single line usage for method '%s': %v",
				m.Method, err)
		}

		if !t.Failed() {
			usageStrs = append(usageStrs, usage)
		}
	}

	if !t.Failed() {
		usages := strings.Join(usageStrs, "\n")
		needsGenerate = usages != requestUsages
	}
}
