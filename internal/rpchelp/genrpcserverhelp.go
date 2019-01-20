
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

//+生成生成

package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcwallet/internal/rpchelp"
)

var outputFile = func() *os.File {
	fi, err := os.Create("rpcserverhelp.go")
	if err != nil {
		log.Fatal(err)
	}
	return fi
}()

func writefln(format string, args ...interface{}) {
	_, err := fmt.Fprintf(outputFile, format, args...)
	if err != nil {
		log.Fatal(err)
	}
	_, err = outputFile.Write([]byte{'\n'})
	if err != nil {
		log.Fatal(err)
	}
}

func writeLocaleHelp(locale, goLocale string, descs map[string]string) {
	funcName := "helpDescs" + goLocale
	writefln("func %s() map[string]string {", funcName)
	writefln("return map[string]string{")
	for i := range rpchelp.Methods {
		m := &rpchelp.Methods[i]
		helpText, err := btcjson.GenerateHelp(m.Method, descs, m.ResultTypes...)
		if err != nil {
			log.Fatal(err)
		}
		writefln("%q: %q,", m.Method, helpText)
	}
	writefln("}")
	writefln("}")
}

func writeLocales() {
	writefln("var localeHelpDescs = map[string]func() map[string]string{")
	for _, h := range rpchelp.HelpDescs {
		writefln("%q: helpDescs%s,", h.Locale, h.GoLocale)
	}
	writefln("}")
}

func writeUsage() {
	usageStrs := make([]string, len(rpchelp.Methods))
	var err error
	for i := range rpchelp.Methods {
		usageStrs[i], err = btcjson.MethodUsageText(rpchelp.Methods[i].Method)
		if err != nil {
			log.Fatal(err)
		}
	}
	usages := strings.Join(usageStrs, "\n")
	writefln("var requestUsages = %q", usages)
}

func main() {
	defer outputFile.Close()

	packageName := "main"
	if len(os.Args) > 1 {
		packageName = os.Args[1]
	}

writefln("//由internal/rpchelp/genrpcserverhelp.go自动生成；不编辑。“）
	writefln("")
	writefln("package %s", packageName)
	writefln("")
	for _, h := range rpchelp.HelpDescs {
		writeLocaleHelp(h.Locale, h.GoLocale, h.Descs)
		writefln("")
	}
	writeLocales()
	writefln("")
	writeUsage()
}
