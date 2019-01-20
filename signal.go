
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2014 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package main

import (
	"os"
	"os/signal"
)

//InterruptChannel用于接收SIGINT（ctrl+c）信号。
var interruptChannel chan os.Signal

//addhandlerChannel用于将中断处理程序添加到处理程序列表中
//在SIGINT（ctrl+c）信号上调用。
var addHandlerChannel = make(chan func())

//在所有中断处理程序运行第一个
//发出中断信号的时间。
var interruptHandlersDone = make(chan struct{})

var simulateInterruptChannel = make(chan struct{}, 1)

//信号定义处理的信号以进行干净的关机。
//条件编译还用于在UNIX上包含sigterm。
var signals = []os.Signal{os.Interrupt}

//SimulateTinterrupt请求通过
//内部组件而不是sigint。
func simulateInterrupt() {
	select {
	case simulateInterruptChannel <- struct{}{}:
	default:
	}
}

//MainInterruptHandler在
//InterruptChannel并相应地调用注册的InterruptCallbacks。
//它还监听回调注册。它必须像野人一样运作。
func mainInterruptHandler() {
//InterruptCallbacks是当
//接收到sigint（ctrl+c）。
	var interruptCallbacks []func()
	invokeCallbacks := func() {
//按后进先出顺序运行处理程序。
		for i := range interruptCallbacks {
			idx := len(interruptCallbacks) - 1 - i
			interruptCallbacks[idx]()
		}
		close(interruptHandlersDone)
	}

	for {
		select {
		case sig := <-interruptChannel:
			log.Infof("Received signal (%s).  Shutting down...", sig)
			invokeCallbacks()
			return
		case <-simulateInterruptChannel:
			log.Info("Received shutdown request.  Shutting down...")
			invokeCallbacks()
			return

		case handler := <-addHandlerChannel:
			interruptCallbacks = append(interruptCallbacks, handler)
		}
	}
}

//当sigint（ctrl+c）为
//收到。
func addInterruptHandler(handler func()) {
//创建通道并启动调用
//所有其他回调和退出（如果尚未完成）。
	if interruptChannel == nil {
		interruptChannel = make(chan os.Signal, 1)
		signal.Notify(interruptChannel, signals...)
		go mainInterruptHandler()
	}

	addHandlerChannel <- handler
}
