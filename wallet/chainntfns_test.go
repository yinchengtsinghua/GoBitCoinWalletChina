
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package wallet

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/waddrmgr"
	_ "github.com/btcsuite/btcwallet/walletdb/bdb"
)

var (
//chainparams是整个钱包中使用的链参数。
//测验。
	chainParams = chaincfg.MainNetParams

//BlockInterval是模拟中任意两个块之间的时间间隔
//链。
	blockInterval = 10 * time.Minute
)

//mock chainconn是chainconn接口的模拟内存实现
//这将用于生日块健全性检查测试。结构是
//
//情节。
type mockChainConn struct {
	chainTip    uint32
	blockHashes map[uint32]chainhash.Hash
	blocks      map[chainhash.Hash]*wire.MsgBlock
}

var _ chainConn = (*mockChainConn)(nil)

//CreateMockChainConn创建由链支持的新模拟链连接
//用n个块。每个块都有一个时间戳，正好在10分钟后
//上一个块的时间戳。
func createMockChainConn(genesis *wire.MsgBlock, n uint32) *mockChainConn {
	c := &mockChainConn{
		chainTip:    n,
		blockHashes: make(map[uint32]chainhash.Hash),
		blocks:      make(map[chainhash.Hash]*wire.MsgBlock),
	}

	genesisHash := genesis.BlockHash()
	c.blockHashes[0] = genesisHash
	c.blocks[genesisHash] = genesis

	for i := uint32(1); i <= n; i++ {
		prevTimestamp := c.blocks[c.blockHashes[i-1]].Header.Timestamp
		block := &wire.MsgBlock{
			Header: wire.BlockHeader{
				Timestamp: prevTimestamp.Add(blockInterval),
			},
		}

		blockHash := block.BlockHash()
		c.blockHashes[i] = blockHash
		c.blocks[blockHash] = block
	}

	return c
}

//GetBestBlock返回已知最佳块的哈希和高度
//后端。
func (c *mockChainConn) GetBestBlock() (*chainhash.Hash, int32, error) {
	bestHash, ok := c.blockHashes[c.chainTip]
	if !ok {
		return nil, 0, fmt.Errorf("block with height %d not found",
			c.chainTip)
	}

	return &bestHash, int32(c.chainTip), nil
}

//GetBlockHash返回具有给定高度的块的哈希。
func (c *mockChainConn) GetBlockHash(height int64) (*chainhash.Hash, error) {
	hash, ok := c.blockHashes[uint32(height)]
	if !ok {
		return nil, fmt.Errorf("block with height %d not found", height)
	}

	return &hash, nil
}

//GetBlockHeader返回具有给定哈希的块的头。
func (c *mockChainConn) GetBlockHeader(hash *chainhash.Hash) (*wire.BlockHeader, error) {
	block, ok := c.blocks[*hash]
	if !ok {
		return nil, fmt.Errorf("header for block %v not found", hash)
	}

	return &block.Header, nil
}

//
//这将用于生日块健全性检查测试。
type mockBirthdayStore struct {
	birthday              time.Time
	birthdayBlock         *waddrmgr.BlockStamp
	birthdayBlockVerified bool
	syncedTo              waddrmgr.BlockStamp
}

var _ birthdayStore = (*mockBirthdayStore)(nil)

//生日返回钱包的生日时间戳。
func (s *mockBirthdayStore) Birthday() time.Time {
	return s.birthday
}

//BirthDayBlock返回钱包的生日块。
func (s *mockBirthdayStore) BirthdayBlock() (waddrmgr.BlockStamp, bool, error) {
	if s.birthdayBlock == nil {
		err := waddrmgr.ManagerError{
			ErrorCode: waddrmgr.ErrBirthdayBlockNotSet,
		}
		return waddrmgr.BlockStamp{}, false, err
	}

	return *s.birthdayBlock, s.birthdayBlockVerified, nil
}

//setbirthdayblock将钱包的生日块更新为给定的块。
//布尔值可用于指示是否应检查此块的健全性。
//下次钱包启动时。
func (s *mockBirthdayStore) SetBirthdayBlock(block waddrmgr.BlockStamp) error {
	s.birthdayBlock = &block
	s.birthdayBlockVerified = true
	s.syncedTo = block
	return nil
}

//
//如果生日块最初不存在，则完成。
func TestBirthdaySanityCheckEmptyBirthdayBlock(t *testing.T) {
	t.Parallel()

	chainConn := &mockChainConn{}

//我们的生日商店会反映出我们没有生日街区
//设置，因此我们不应尝试进行健全性检查。
	birthdayStore := &mockBirthdayStore{}

	birthdayBlock, err := birthdaySanityCheck(chainConn, birthdayStore)
	if !waddrmgr.IsError(err, waddrmgr.ErrBirthdayBlockNotSet) {
		t.Fatalf("expected ErrBirthdayBlockNotSet, got %v", err)
	}

	if birthdayBlock != nil {
		t.Fatalf("expected birthday block to be nil due to not being "+
			"set, got %v", *birthdayBlock)
	}
}

//testBirthDaySanityCheckVerifiedBirthDayBlock确保
//如果已验证生日块，则不执行。
func TestBirthdaySanityCheckVerifiedBirthdayBlock(t *testing.T) {
	t.Parallel()

	const chainTip = 5000
	chainConn := createMockChainConn(chainParams.GenesisBlock, chainTip)
	expectedBirthdayBlock := waddrmgr.BlockStamp{Height: 1337}

//我们的生日商店反映出我们的生日街区已经
//已验证，不需要进行健全性检查。
	birthdayStore := &mockBirthdayStore{
		birthdayBlock:         &expectedBirthdayBlock,
		birthdayBlockVerified: true,
		syncedTo: waddrmgr.BlockStamp{
			Height: chainTip,
		},
	}

//现在，我们将进行健康检查。我们应该看看生日
//块未更改。
	birthdayBlock, err := birthdaySanityCheck(chainConn, birthdayStore)
	if err != nil {
		t.Fatalf("unable to sanity check birthday block: %v", err)
	}
	if !reflect.DeepEqual(*birthdayBlock, expectedBirthdayBlock) {
		t.Fatalf("expected birthday block %v, got %v",
			expectedBirthdayBlock, birthdayBlock)
	}

//为了确保健康检查没有进行，我们将检查同步到的
//高度，因为如果新的候选对象
//被发现。
	if birthdayStore.syncedTo.Height != chainTip {
		t.Fatalf("expected synced height remain the same (%d), got %d",
			chainTip, birthdayStore.syncedTo.Height)
	}
}

//testBirthDaysSanityCheckLowerEstimate确保我们可以正确定位
//如果我们的估计正好是太远的话，最好是生日档的候选人。
//链条。
func TestBirthdaySanityCheckLowerEstimate(t *testing.T) {
	t.Parallel()

//我们将从定义生日时间戳开始，
//1337块的时间戳。
	genesisTimestamp := chainParams.GenesisBlock.Header.Timestamp
	birthday := genesisTimestamp.Add(1337 * blockInterval)

//我们将建立到5000个区块的模拟链的连接。
	chainConn := createMockChainConn(chainParams.GenesisBlock, 5000)

//我们的生日商店将反映出我们的生日街区目前
//
//通过健全性检查进行调整。
	birthdayStore := &mockBirthdayStore{
		birthday: birthday,
		birthdayBlock: &waddrmgr.BlockStamp{
			Hash:      *chainParams.GenesisHash,
			Height:    0,
			Timestamp: genesisTimestamp,
		},
		birthdayBlockVerified: false,
		syncedTo: waddrmgr.BlockStamp{
			Height: 5000,
		},
	}

//我们将进行健康检查并确定我们是否能够
//找一个更好的生日蛋糕候选人。
	birthdayBlock, err := birthdaySanityCheck(chainConn, birthdayStore)
	if err != nil {
		t.Fatalf("unable to sanity check birthday block: %v", err)
	}
	if birthday.Sub(birthdayBlock.Timestamp) >= birthdayBlockDelta {
		t.Fatalf("expected birthday block timestamp=%v to be within "+
			"%v of birthday timestamp=%v", birthdayBlock.Timestamp,
			birthdayBlockDelta, birthday)
	}

//
//阻止以确保钱包不会错过任何事件
//向前地。
	if !reflect.DeepEqual(birthdayStore.syncedTo, *birthdayBlock) {
		t.Fatalf("expected syncedTo and birthday block to match: "+
			"%v vs %v", birthdayStore.syncedTo, birthdayBlock)
	}
}

//testBirthDaysSanityCheckHigherestimate确保我们可以正确定位
//如果我们的估计值正好太高，
//链。
func TestBirthdaySanityCheckHigherEstimate(t *testing.T) {
	t.Parallel()

//我们将从定义生日时间戳开始，
//1337块的时间戳。
	genesisTimestamp := chainParams.GenesisBlock.Header.Timestamp
	birthday := genesisTimestamp.Add(1337 * blockInterval)

//我们将建立到5000个区块的模拟链的连接。
	chainConn := createMockChainConn(chainParams.GenesisBlock, 5000)

//我们的生日商店将反映出我们的生日街区目前
//设置为链尖。该值太高，应进行调整
//通过健康检查。
	bestBlock := chainConn.blocks[chainConn.blockHashes[5000]]
	birthdayStore := &mockBirthdayStore{
		birthday: birthday,
		birthdayBlock: &waddrmgr.BlockStamp{
			Hash:      bestBlock.BlockHash(),
			Height:    5000,
			Timestamp: bestBlock.Header.Timestamp,
		},
		birthdayBlockVerified: false,
		syncedTo: waddrmgr.BlockStamp{
			Height: 5000,
		},
	}

//我们将进行健康检查并确定我们是否能够
//找一个更好的生日蛋糕候选人。
	birthdayBlock, err := birthdaySanityCheck(chainConn, birthdayStore)
	if err != nil {
		t.Fatalf("unable to sanity check birthday block: %v", err)
	}
	if birthday.Sub(birthdayBlock.Timestamp) >= birthdayBlockDelta {
		t.Fatalf("expected birthday block timestamp=%v to be within "+
			"%v of birthday timestamp=%v", birthdayBlock.Timestamp,
			birthdayBlockDelta, birthday)
	}

//
//阻止以确保钱包不会错过任何事件
//向前地。
	if !reflect.DeepEqual(birthdayStore.syncedTo, *birthdayBlock) {
		t.Fatalf("expected syncedTo and birthday block to match: "+
			"%v vs %v", birthdayStore.syncedTo, birthdayBlock)
	}
}
