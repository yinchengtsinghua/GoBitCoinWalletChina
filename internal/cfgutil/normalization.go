
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

package cfgutil

import "net"

//NormalizedAddress返回地址的规范化形式，并添加默认值
//必要时连接端口。如果地址，即使没有端口，
//无效。
func NormalizeAddress(addr string, defaultPort string) (hostport string, err error) {
//如果第一个splithostport由于缺少端口而出错，而不是
//对于无效主机，请添加端口。如果第二个splithostport
//失败，则端口不丢失，原始错误应为
//返回。
	host, port, origErr := net.SplitHostPort(addr)
	if origErr == nil {
		return net.JoinHostPort(host, port), nil
	}
	addr = net.JoinHostPort(addr, defaultPort)
	_, _, err = net.SplitHostPort(addr)
	if err != nil {
		return "", origErr
	}
	return addr, nil
}

//normalizeadresss返回一个新切片，其中包含所有传递的对等地址
//使用给定的默认端口进行规范化，并删除所有重复项。
func NormalizeAddresses(addrs []string, defaultPort string) ([]string, error) {
	var (
		normalized = make([]string, 0, len(addrs))
		seenSet    = make(map[string]struct{})
	)

	for _, addr := range addrs {
		normalizedAddr, err := NormalizeAddress(addr, defaultPort)
		if err != nil {
			return nil, err
		}
		_, seen := seenSet[normalizedAddr]
		if !seen {
			normalized = append(normalized, normalizedAddr)
			seenSet[normalizedAddr] = struct{}{}
		}
	}

	return normalized, nil
}
