
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package chain

import (
	"container/list"
)

//ConcurrentQueue是一个具有无限容量的并发安全FIFO队列。
//客户机通过将项目推送到in通道和
//从输出通道弹出项目。有一个Goroutine负责移动
//以正确的顺序从输入通道到输出通道的项目必须
//通过调用Start（）启动。
type ConcurrentQueue struct {
	chanIn   chan interface{}
	chanOut  chan interface{}
	quit     chan struct{}
	overflow *list.List
}

//NewConcurrentQueue构造一个ConcurrentQueue。bufferSize参数是
//输出通道的容量。当队列的大小低于此值时
//阈值，推送不会产生效率较低的溢出的开销
//结构。
func NewConcurrentQueue(bufferSize int) *ConcurrentQueue {
	return &ConcurrentQueue{
		chanIn:   make(chan interface{}),
		chanOut:  make(chan interface{}, bufferSize),
		quit:     make(chan struct{}),
		overflow: list.New(),
	}
}

//chanin返回一个可用于将新项目推入队列的通道。
func (cq *ConcurrentQueue) ChanIn() chan<- interface{} {
	return cq.chanIn
}

//chanout返回可用于从队列中弹出项目的通道。
func (cq *ConcurrentQueue) ChanOut() <-chan interface{} {
	return cq.chanOut
}

//Start开始一个Goroutine，该Goroutine管理将项目从In通道移动到
//输出通道。队列尝试将项目直接移动到输出通道
//尽可能减少开销，但如果输出通道已满，则会将项推送到
//溢出队列。在使用队列之前必须调用此函数。
func (cq *ConcurrentQueue) Start() {
	go func() {
		for {
			nextElement := cq.overflow.Front()
			if nextElement == nil {
//溢出队列为空，因此传入
//项目可以直接推送到输出
//通道。但是，如果输出通道已满，
//我们将推到溢出列表。
				select {
				case item := <-cq.chanIn:
					select {
					case cq.chanOut <- item:
					case <-cq.quit:
						return
					default:
						cq.overflow.PushBack(item)
					}
				case <-cq.quit:
					return
				}
			} else {
//溢出队列不是空的，因此
//物品被推到后面保存
//秩序。
				select {
				case item := <-cq.chanIn:
					cq.overflow.PushBack(item)
				case cq.chanOut <- nextElement.Value:
					cq.overflow.Remove(nextElement)
				case <-cq.quit:
					return
				}
			}
		}
	}()
}

//stop结束将项目从in通道移动到out的goroutine
//通道。
func (cq *ConcurrentQueue) Stop() {
	close(cq.quit)
}
