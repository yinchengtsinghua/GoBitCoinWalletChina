
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

package wtxmgr

import "github.com/btcsuite/btcd/chaincfg/chainhash"

type graphNode struct {
	value    *TxRecord
	outEdges []*chainhash.Hash
	inDegree int
}

type hashGraph map[chainhash.Hash]graphNode

func makeGraph(set map[chainhash.Hash]*TxRecord) hashGraph {
	graph := make(hashGraph)

	for _, rec := range set {
//为每个事务记录添加一个节点。输出边
//通过对每个记录的
//以下输入。
		if _, ok := graph[rec.Hash]; !ok {
			graph[rec.Hash] = graphNode{value: rec}
		}

	inputLoop:
		for _, input := range rec.MsgTx.TxIn {
//引用非交易记录的交易记录输入
//包含在集合中不创建任何（本地）图表
//边缘。
			if _, ok := set[input.PreviousOutPoint.Hash]; !ok {
				continue
			}

			inputNode := graph[input.PreviousOutPoint.Hash]

//跳过重复边。
			for _, outEdge := range inputNode.outEdges {
				if *outEdge == input.PreviousOutPoint.Hash {
					continue inputLoop
				}
			}

//标记上一个事务的定向边缘
//散列到此事务记录并增加
//输入此记录节点的度数。
			inputRec := inputNode.value
			if inputRec == nil {
				inputRec = set[input.PreviousOutPoint.Hash]
			}
			graph[input.PreviousOutPoint.Hash] = graphNode{
				value:    inputRec,
				outEdges: append(inputNode.outEdges, &rec.Hash),
				inDegree: inputNode.inDegree,
			}
			node := graph[rec.Hash]
			graph[rec.Hash] = graphNode{
				value:    rec,
				outEdges: node.outEdges,
				inDegree: node.inDegree + 1,
			}
		}
	}

	return graph
}

//graph roots返回图的根。也就是说，它返回节点的
//包含0的输入阶数的所有节点的值。
func graphRoots(graph hashGraph) []*TxRecord {
	roots := make([]*TxRecord, 0, len(graph))
	for _, node := range graph {
		if node.inDegree == 0 {
			roots = append(roots, node.value)
		}
	}
	return roots
}

//DependencySort拓扑排序一组事务记录
//依赖项顺序。它是使用Kahn的算法实现的。
func dependencySort(txs map[chainhash.Hash]*TxRecord) []*TxRecord {
	graph := makeGraph(txs)
	s := graphRoots(graph)

//如果没有边（每个边都没有来自映射引用的事务
//其他），那么Kahn的算法是不必要的。
	if len(s) == len(txs) {
		return s
	}

	sorted := make([]*TxRecord, 0, len(txs))
	for len(s) != 0 {
		rec := s[0]
		s = s[1:]
		sorted = append(sorted, rec)

		n := graph[rec.Hash]
		for _, mHash := range n.outEdges {
			m := graph[*mHash]
			if m.inDegree != 0 {
				m.inDegree--
				graph[*mHash] = m
				if m.inDegree == 0 {
					s = append(s, m.value)
				}
			}
		}
	}
	return sorted
}
