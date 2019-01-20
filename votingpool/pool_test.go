
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package votingpool_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"

	"github.com/btcsuite/btcutil/hdkeychain"
	vp "github.com/btcsuite/btcwallet/votingpool"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/walletdb"
	_ "github.com/btcsuite/btcwallet/walletdb/bdb"
)

func TestLoadPoolAndDepositScript(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

//设置
	poolID := "test"
	pubKeys := vp.TstPubKeys[0:3]
	err = vp.LoadAndCreateSeries(ns, pool.Manager(), 1, poolID, 1, 2, pubKeys)
	if err != nil {
		t.Fatalf("Failed to create voting pool and series: %v", err)
	}

//执行
	script, err := vp.LoadAndGetDepositScript(ns, pool.Manager(), poolID, 1, 0, 0)
	if err != nil {
		t.Fatalf("Failed to get deposit script: %v", err)
	}

//验证
	strScript := hex.EncodeToString(script)
	want := "5221035e94da75731a2153b20909017f62fcd49474c45f3b46282c0dafa8b40a3a312b2102e983a53dd20b7746dd100dfd2925b777436fc1ab1dd319433798924a5ce143e32102908d52a548ee9ef6b2d0ea67a3781a0381bc3570ad623564451e63757ff9393253ae"
	if want != strScript {
		t.Fatalf("Failed to get the right deposit script. Got %v, want %v",
			strScript, want)
	}
}

func TestLoadPoolAndCreateSeries(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

	poolID := "test"

//第一次创建投票池
	pubKeys := vp.TstPubKeys[0:3]
	err = vp.LoadAndCreateSeries(ns, pool.Manager(), 1, poolID, 1, 2, pubKeys)
	if err != nil {
		t.Fatalf("Creating voting pool and Creating series failed: %v", err)
	}

//创建另一个系列，这次在其中加载投票池
	pubKeys = vp.TstPubKeys[3:6]
	err = vp.LoadAndCreateSeries(ns, pool.Manager(), 1, poolID, 2, 2, pubKeys)

	if err != nil {
		t.Fatalf("Loading voting pool and Creating series failed: %v", err)
	}
}

func TestLoadPoolAndReplaceSeries(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

//设置
	poolID := "test"
	pubKeys := vp.TstPubKeys[0:3]
	err = vp.LoadAndCreateSeries(ns, pool.Manager(), 1, poolID, 1, 2, pubKeys)
	if err != nil {
		t.Fatalf("Failed to create voting pool and series: %v", err)
	}

	pubKeys = vp.TstPubKeys[3:6]
	err = vp.LoadAndReplaceSeries(ns, pool.Manager(), 1, poolID, 1, 2, pubKeys)
	if err != nil {
		t.Fatalf("Failed to replace series: %v", err)
	}
}

func TestLoadPoolAndEmpowerSeries(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, addrmgrNs := vp.TstRWNamespaces(dbtx)

//设置
	poolID := "test"
	pubKeys := vp.TstPubKeys[0:3]
	err = vp.LoadAndCreateSeries(ns, pool.Manager(), 1, poolID, 1, 2, pubKeys)
	if err != nil {
		t.Fatalf("Creating voting pool and Creating series failed: %v", err)
	}

	vp.TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		err = vp.LoadAndEmpowerSeries(ns, pool.Manager(), poolID, 1, vp.TstPrivKeys[0])
	})
	if err != nil {
		t.Fatalf("Load voting pool and Empower series failed: %v", err)
	}
}

func TestDepositScriptAddress(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

	tests := []struct {
		version uint32
		series  uint32
		reqSigs uint32
		pubKeys []string
//分支机构地图：地址（我们只检查0处的分支机构索引）
		addresses map[uint32]string
	}{
		{
			version: 1,
			series:  1,
			reqSigs: 2,
			pubKeys: vp.TstPubKeys[0:3],
			addresses: map[uint32]string{
				0: "3Hb4xcebcKg4DiETJfwjh8sF4uDw9rqtVC",
				1: "34eVkREKgvvGASZW7hkgE2uNc1yycntMK6",
				2: "3Qt1EaKRD9g9FeL2DGkLLswhK1AKmmXFSe",
				3: "3PbExiaztsSYgh6zeMswC49hLUwhTQ86XG",
			},
		},
	}

	for i, test := range tests {
		if err := pool.CreateSeries(ns, test.version, test.series,
			test.reqSigs, test.pubKeys); err != nil {
			t.Fatalf("Cannot creates series %v", test.series)
		}
		for branch, expectedAddress := range test.addresses {
			addr, err := pool.DepositScriptAddress(test.series, vp.Branch(branch), vp.Index(0))
			if err != nil {
				t.Fatalf("Failed to get DepositScriptAddress #%d: %v", i, err)
			}
			address := addr.EncodeAddress()
			if expectedAddress != address {
				t.Errorf("DepositScript #%d returned the wrong deposit script. Got %v, want %v",
					i, address, expectedAddress)
			}
		}
	}
}

func TestDepositScriptAddressForNonExistentSeries(t *testing.T) {
	tearDown, _, pool := vp.TstCreatePool(t)
	defer tearDown()

	_, err := pool.DepositScriptAddress(1, 0, 0)

	vp.TstCheckError(t, "", err, vp.ErrSeriesNotExists)
}

func TestDepositScriptAddressForHardenedPubKey(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

	if err := pool.CreateSeries(ns, 1, 1, 2, vp.TstPubKeys[0:3]); err != nil {
		t.Fatalf("Cannot creates series")
	}

//要求使用硬化儿童索引的DepositscriptAddress，这应该
//当我们使用扩展的公钥派生子项时失败。
	_, err = pool.DepositScriptAddress(1, 0, vp.Index(hdkeychain.HardenedKeyStart+1))

	vp.TstCheckError(t, "", err, vp.ErrKeyChain)
}

func TestLoadPool(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

	pool2, err := vp.Load(ns, pool.Manager(), pool.ID)
	if err != nil {
		t.Errorf("Error loading Pool: %v", err)
	}
	if !bytes.Equal(pool2.ID, pool.ID) {
		t.Errorf("Voting pool obtained from DB does not match the created one")
	}
}

func TestCreatePool(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

	pool2, err := vp.Create(ns, pool.Manager(), []byte{0x02})
	if err != nil {
		t.Errorf("Error creating Pool: %v", err)
	}
	if !bytes.Equal(pool2.ID, []byte{0x02}) {
		t.Errorf("Pool ID mismatch: got %v, want %v", pool2.ID, []byte{0x02})
	}
}

func TestCreatePoolWhenAlreadyExists(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

	_, err = vp.Create(ns, pool.Manager(), pool.ID)

	vp.TstCheckError(t, "", err, vp.ErrPoolAlreadyExists)
}

func TestCreateSeries(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

	tests := []struct {
		version uint32
		series  uint32
		reqSigs uint32
		pubKeys []string
	}{
		{
			version: 1,
			series:  1,
			reqSigs: 2,
			pubKeys: vp.TstPubKeys[0:3],
		},
		{
			version: 1,
			series:  2,
			reqSigs: 3,
			pubKeys: vp.TstPubKeys[0:5],
		},
		{
			version: 1,
			series:  3,
			reqSigs: 4,
			pubKeys: vp.TstPubKeys[0:7],
		},
		{
			version: 1,
			series:  4,
			reqSigs: 5,
			pubKeys: vp.TstPubKeys[0:9],
		},
	}

	for testNum, test := range tests {
		err := pool.CreateSeries(ns, test.version, test.series, test.reqSigs, test.pubKeys[:])
		if err != nil {
			t.Fatalf("%d: Cannot create series %d", testNum, test.series)
		}
		exists, err := pool.TstExistsSeries(dbtx, test.series)
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Errorf("%d: Series %d not in database", testNum, test.series)
		}
	}
}

func TestPoolCreateSeriesInvalidID(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

	err = pool.CreateSeries(ns, vp.CurrentVersion, 0, 1, vp.TstPubKeys[0:3])

	vp.TstCheckError(t, "", err, vp.ErrSeriesIDInvalid)
}

func TestPoolCreateSeriesWhenAlreadyExists(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

	pubKeys := vp.TstPubKeys[0:3]
	if err := pool.CreateSeries(ns, 1, 1, 1, pubKeys); err != nil {
		t.Fatalf("Cannot create series: %v", err)
	}

	err = pool.CreateSeries(ns, 1, 1, 1, pubKeys)

	vp.TstCheckError(t, "", err, vp.ErrSeriesAlreadyExists)
}

func TestPoolCreateSeriesIDNotSequential(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

	pubKeys := vp.TstPubKeys[0:4]
	if err := pool.CreateSeries(ns, 1, 1, 2, pubKeys); err != nil {
		t.Fatalf("Cannot create series: %v", err)
	}

	err = pool.CreateSeries(ns, 1, 3, 2, pubKeys)

	vp.TstCheckError(t, "", err, vp.ErrSeriesIDNotSequential)
}

func TestPutSeriesErrors(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

	tests := []struct {
		version uint32
		reqSigs uint32
		pubKeys []string
		err     vp.ErrorCode
		msg     string
	}{
		{
			pubKeys: vp.TstPubKeys[0:1],
			err:     vp.ErrTooFewPublicKeys,
			msg:     "Should return error when passed too few pubkeys",
		},
		{
			reqSigs: 5,
			pubKeys: vp.TstPubKeys[0:3],
			err:     vp.ErrTooManyReqSignatures,
			msg:     "Should return error when reqSigs > len(pubKeys)",
		},
		{
			pubKeys: []string{vp.TstPubKeys[0], vp.TstPubKeys[1], vp.TstPubKeys[2], vp.TstPubKeys[0]},
			err:     vp.ErrKeyDuplicate,
			msg:     "Should return error when passed duplicate pubkeys",
		},
		{
			pubKeys: []string{"invalidxpub1", "invalidxpub2", "invalidxpub3"},
			err:     vp.ErrKeyChain,
			msg:     "Should return error when passed invalid pubkey",
		},
		{
			pubKeys: vp.TstPrivKeys[0:3],
			err:     vp.ErrKeyIsPrivate,
			msg:     "Should return error when passed private keys",
		},
	}

	for i, test := range tests {
		err := pool.TstPutSeries(ns, test.version, uint32(i+1), test.reqSigs, test.pubKeys)
		vp.TstCheckError(t, fmt.Sprintf("Create series #%d", i), err, test.err)
	}
}

func TestCannotReplaceEmpoweredSeries(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, addrmgrNs := vp.TstRWNamespaces(dbtx)

	seriesID := uint32(1)

	if err := pool.CreateSeries(ns, 1, seriesID, 3, vp.TstPubKeys[0:4]); err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	vp.TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		if err := pool.EmpowerSeries(ns, seriesID, vp.TstPrivKeys[1]); err != nil {
			t.Fatalf("Failed to empower series: %v", err)
		}
	})

	err = pool.ReplaceSeries(ns, 1, seriesID, 2, []string{vp.TstPubKeys[0], vp.TstPubKeys[2],
		vp.TstPubKeys[3]})

	vp.TstCheckError(t, "", err, vp.ErrSeriesAlreadyEmpowered)
}

func TestReplaceNonExistingSeries(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

	pubKeys := vp.TstPubKeys[0:3]

	err = pool.ReplaceSeries(ns, 1, 1, 3, pubKeys)

	vp.TstCheckError(t, "", err, vp.ErrSeriesNotExists)
}

type replaceSeriesTestEntry struct {
	testID      int
	orig        seriesRaw
	replaceWith seriesRaw
}

var replaceSeriesTestData = []replaceSeriesTestEntry{
	{
		testID: 0,
		orig: seriesRaw{
			id:      1,
			version: 1,
			reqSigs: 2,
			pubKeys: vp.CanonicalKeyOrder([]string{vp.TstPubKeys[0], vp.TstPubKeys[1],
				vp.TstPubKeys[2], vp.TstPubKeys[4]}),
		},
		replaceWith: seriesRaw{
			id:      1,
			version: 1,
			reqSigs: 1,
			pubKeys: vp.CanonicalKeyOrder(vp.TstPubKeys[3:6]),
		},
	},
	{
		testID: 1,
		orig: seriesRaw{
			id:      2,
			version: 1,
			reqSigs: 2,
			pubKeys: vp.CanonicalKeyOrder(vp.TstPubKeys[0:3]),
		},
		replaceWith: seriesRaw{
			id:      2,
			version: 1,
			reqSigs: 2,
			pubKeys: vp.CanonicalKeyOrder(vp.TstPubKeys[3:7]),
		},
	},
	{
		testID: 2,
		orig: seriesRaw{
			id:      3,
			version: 1,
			reqSigs: 8,
			pubKeys: vp.CanonicalKeyOrder(vp.TstPubKeys[0:9]),
		},
		replaceWith: seriesRaw{
			id:      3,
			version: 1,
			reqSigs: 7,
			pubKeys: vp.CanonicalKeyOrder(vp.TstPubKeys[0:8]),
		},
	},
}

func TestReplaceExistingSeries(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

	for _, data := range replaceSeriesTestData {
		seriesID := data.orig.id
		testID := data.testID

		if err := pool.CreateSeries(ns, data.orig.version, seriesID, data.orig.reqSigs, data.orig.pubKeys); err != nil {
			t.Fatalf("Test #%d: failed to create series in replace series setup: %v",
				testID, err)
		}

		if err := pool.ReplaceSeries(ns, data.replaceWith.version, seriesID,
			data.replaceWith.reqSigs, data.replaceWith.pubKeys); err != nil {
			t.Errorf("Test #%d: replaceSeries failed: %v", testID, err)
		}

		validateReplaceSeries(t, pool, testID, data.replaceWith)
	}
}

//validateReplaceSeries验证存储在系统中的已创建序列
//对应于我们用原始序列替换的序列。
func validateReplaceSeries(t *testing.T, pool *vp.Pool, testID int, replacedWith seriesRaw) {
	seriesID := replacedWith.id
	series := pool.Series(seriesID)
	if series == nil {
		t.Fatalf("Test #%d Series #%d: series not found", testID, seriesID)
	}

	pubKeys := series.TstGetRawPublicKeys()
//检查公钥是否与我们期望的相符。
	if !reflect.DeepEqual(replacedWith.pubKeys, pubKeys) {
		t.Errorf("Test #%d, series #%d: pubkeys mismatch. Got %v, want %v",
			testID, seriesID, pubKeys, replacedWith.pubKeys)
	}

//检查所需信号的编号。
	if replacedWith.reqSigs != series.TstGetReqSigs() {
		t.Errorf("Test #%d, series #%d: required signatures mismatch. Got %d, want %d",
			testID, seriesID, series.TstGetReqSigs(), replacedWith.reqSigs)
	}

//检查序列是否未被授权。
	if series.IsEmpowered() {
		t.Errorf("Test #%d, series #%d: series is empowered but should not be",
			testID, seriesID)
	}
}

func TestEmpowerSeries(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, addrmgrNs := vp.TstRWNamespaces(dbtx)

	seriesID := uint32(1)
	if err := pool.CreateSeries(ns, 1, seriesID, 2, vp.TstPubKeys[0:3]); err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	vp.TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
		if err := pool.EmpowerSeries(ns, seriesID, vp.TstPrivKeys[0]); err != nil {
			t.Errorf("Failed to empower series: %v", err)
		}
	})
}

func TestEmpowerSeriesErrors(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

	seriesID := uint32(1)
	if err := pool.CreateSeries(ns, 1, seriesID, 2, vp.TstPubKeys[0:3]); err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	tests := []struct {
		seriesID uint32
		key      string
		err      vp.ErrorCode
	}{
		{
			seriesID: 2,
			key:      vp.TstPrivKeys[0],
//无效的系列。
			err: vp.ErrSeriesNotExists,
		},
		{
			seriesID: seriesID,
			key:      "NONSENSE",
//私钥无效。
			err: vp.ErrKeyChain,
		},
		{
			seriesID: seriesID,
			key:      vp.TstPubKeys[5],
//键类型错误。
			err: vp.ErrKeyIsPublic,
		},
		{
			seriesID: seriesID,
			key:      vp.TstPrivKeys[5],
//密钥与公钥不对应。
			err: vp.ErrKeysPrivatePublicMismatch,
		},
	}

	for i, test := range tests {
		err := pool.EmpowerSeries(ns, test.seriesID, test.key)
		vp.TstCheckError(t, fmt.Sprintf("EmpowerSeries #%d", i), err, test.err)
	}

}

func TestPoolSeries(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

	expectedPubKeys := vp.CanonicalKeyOrder(vp.TstPubKeys[0:3])
	if err := pool.CreateSeries(ns, vp.CurrentVersion, 1, 2, expectedPubKeys); err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	series := pool.Series(1)

	if series == nil {
		t.Fatal("Series() returned nil")
	}
	pubKeys := series.TstGetRawPublicKeys()
	if !reflect.DeepEqual(pubKeys, expectedPubKeys) {
		t.Errorf("Series pubKeys mismatch. Got %v, want %v", pubKeys, expectedPubKeys)
	}
}

type seriesRaw struct {
	id       uint32
	version  uint32
	reqSigs  uint32
	pubKeys  []string
	privKeys []string
}

type testLoadAllSeriesTest struct {
	id     int
	series []seriesRaw
}

var testLoadAllSeriesTests = []testLoadAllSeriesTest{
	{
		id: 1,
		series: []seriesRaw{
			{
				id:      1,
				version: 1,
				reqSigs: 2,
				pubKeys: vp.TstPubKeys[0:3],
			},
			{
				id:       2,
				version:  1,
				reqSigs:  2,
				pubKeys:  vp.TstPubKeys[3:6],
				privKeys: vp.TstPrivKeys[4:5],
			},
			{
				id:       3,
				version:  1,
				reqSigs:  3,
				pubKeys:  vp.TstPubKeys[0:5],
				privKeys: []string{vp.TstPrivKeys[0], vp.TstPrivKeys[2]},
			},
		},
	},
	{
		id: 2,
		series: []seriesRaw{
			{
				id:      1,
				version: 1,
				reqSigs: 2,
				pubKeys: vp.TstPubKeys[0:3],
			},
		},
	},
}

func setUpLoadAllSeries(t *testing.T, dbtx walletdb.ReadWriteTx, mgr *waddrmgr.Manager,
	test testLoadAllSeriesTest) *vp.Pool {
	ns, addrmgrNs := vp.TstRWNamespaces(dbtx)
	pool, err := vp.Create(ns, mgr, []byte{byte(test.id + 1)})
	if err != nil {
		t.Fatalf("Voting Pool creation failed: %v", err)
	}

	for _, series := range test.series {
		err := pool.CreateSeries(ns, series.version, series.id,
			series.reqSigs, series.pubKeys)
		if err != nil {
			t.Fatalf("Test #%d Series #%d: failed to create series: %v",
				test.id, series.id, err)
		}

		for _, privKey := range series.privKeys {
			vp.TstRunWithManagerUnlocked(t, mgr, addrmgrNs, func() {
				if err := pool.EmpowerSeries(ns, series.id, privKey); err != nil {
					t.Fatalf("Test #%d Series #%d: empower with privKey %v failed: %v",
						test.id, series.id, privKey, err)
				}
			})
		}
	}
	return pool
}

func TestLoadAllSeries(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, addrmgrNs := vp.TstRWNamespaces(dbtx)

	for _, test := range testLoadAllSeriesTests {
		pool := setUpLoadAllSeries(t, dbtx, pool.Manager(), test)
		pool.TstEmptySeriesLookup()
		vp.TstRunWithManagerUnlocked(t, pool.Manager(), addrmgrNs, func() {
			if err := pool.LoadAllSeries(ns); err != nil {
				t.Fatalf("Test #%d: failed to load voting pool: %v", test.id, err)
			}
		})
		for _, seriesData := range test.series {
			validateLoadAllSeries(t, pool, test.id, seriesData)
		}
	}
}

func validateLoadAllSeries(t *testing.T, pool *vp.Pool, testID int, seriesData seriesRaw) {
	series := pool.Series(seriesData.id)

//检查序列是否存在。
	if series == nil {
		t.Errorf("Test #%d, series #%d: series not found", testID, seriesData.id)
	}

//检查reqsigs是否是我们插入的。
	if seriesData.reqSigs != series.TstGetReqSigs() {
		t.Errorf("Test #%d, series #%d: required sigs are different. Got %d, want %d",
			testID, seriesData.id, series.TstGetReqSigs(), seriesData.reqSigs)
	}

//检查pubkeys和privkeys的长度是否相同。
	publicKeys := series.TstGetRawPublicKeys()
	privateKeys := series.TstGetRawPrivateKeys()
	if len(privateKeys) != len(publicKeys) {
		t.Errorf("Test #%d, series #%d: wrong number of private keys. Got %d, want %d",
			testID, seriesData.id, len(privateKeys), len(publicKeys))
	}

	sortedKeys := vp.CanonicalKeyOrder(seriesData.pubKeys)
	if !reflect.DeepEqual(publicKeys, sortedKeys) {
		t.Errorf("Test #%d, series #%d: public keys mismatch. Got %v, want %v",
			testID, seriesData.id, sortedKeys, publicKeys)
	}

//检查私钥是否是我们插入的（长度和内容）。
	foundPrivKeys := make([]string, 0, len(seriesData.pubKeys))
	for _, privateKey := range privateKeys {
		if privateKey != "" {
			foundPrivKeys = append(foundPrivKeys, privateKey)
		}
	}
	foundPrivKeys = vp.CanonicalKeyOrder(foundPrivKeys)
	privKeys := vp.CanonicalKeyOrder(seriesData.privKeys)
	if !reflect.DeepEqual(privKeys, foundPrivKeys) {
		t.Errorf("Test #%d, series #%d: private keys mismatch. Got %v, want %v",
			testID, seriesData.id, foundPrivKeys, privKeys)
	}
}

func reverse(inKeys []*hdkeychain.ExtendedKey) []*hdkeychain.ExtendedKey {
	revKeys := make([]*hdkeychain.ExtendedKey, len(inKeys))
	max := len(inKeys)
	for i := range inKeys {
		revKeys[i] = inKeys[max-i-1]
	}
	return revKeys
}

func TestBranchOrderZero(t *testing.T) {
//0-10键的测试更改地址分支（0）
	for i := 0; i < 10; i++ {
		inKeys := createTestPubKeys(t, i, 0)
		wantKeys := reverse(inKeys)
		resKeys, err := vp.TstBranchOrder(inKeys, 0)
		if err != nil {
			t.Fatalf("Error ordering keys: %v", err)
		}

		if len(resKeys) != len(wantKeys) {
			t.Errorf("BranchOrder: wrong no. of keys. Got: %d, want %d",
				len(resKeys), len(inKeys))
			return
		}

		for keyIdx := 0; i < len(inKeys); i++ {
			if resKeys[keyIdx] != wantKeys[keyIdx] {
				t.Errorf("BranchOrder(keys, 0): got %v, want %v",
					resKeys[i], wantKeys[i])
			}
		}
	}
}

func TestBranchOrderNonZero(t *testing.T) {
	maxBranch := 5
	maxTail := 4
//分支编号大于0的测试分支重新排序。我们测试所有分支值
//在[1，5]以内，最多9片（maxbranch-1+branch pivot+
//密钥）。希望这涵盖了所有的组合和边缘情况。
//我们在别处测试分支编号为0的情况。
	for branch := 1; branch <= maxBranch; branch++ {
		for j := 0; j <= maxTail; j++ {
			first := createTestPubKeys(t, branch-1, 0)
			pivot := createTestPubKeys(t, 1, branch)
			last := createTestPubKeys(t, j, branch+1)

			inKeys := append(append(first, pivot...), last...)
			wantKeys := append(append(pivot, first...), last...)
			resKeys, err := vp.TstBranchOrder(inKeys, vp.Branch(branch))
			if err != nil {
				t.Fatalf("Error ordering keys: %v", err)
			}

			if len(resKeys) != len(inKeys) {
				t.Errorf("BranchOrder: wrong no. of keys. Got: %d, want %d",
					len(resKeys), len(inKeys))
			}

			for idx := 0; idx < len(inKeys); idx++ {
				if resKeys[idx] != wantKeys[idx] {
					o, w, g := branchErrorFormat(inKeys, wantKeys, resKeys)
					t.Errorf("Branch: %d\nOrig: %v\nGot: %v\nWant: %v", branch, o, g, w)
				}
			}
		}
	}
}

func TestBranchOrderNilKeys(t *testing.T) {
	_, err := vp.TstBranchOrder(nil, 1)

	vp.TstCheckError(t, "", err, vp.ErrInvalidValue)
}

func TestBranchOrderInvalidBranch(t *testing.T) {
	_, err := vp.TstBranchOrder(createTestPubKeys(t, 3, 0), 4)

	vp.TstCheckError(t, "", err, vp.ErrInvalidBranch)
}

func branchErrorFormat(orig, want, got []*hdkeychain.ExtendedKey) (origOrder, wantOrder, gotOrder []int) {
	origOrder = []int{}
	origMap := make(map[*hdkeychain.ExtendedKey]int)
	for i, key := range orig {
		origMap[key] = i + 1
		origOrder = append(origOrder, i+1)
	}

	wantOrder = []int{}
	for _, key := range want {
		wantOrder = append(wantOrder, origMap[key])
	}

	gotOrder = []int{}
	for _, key := range got {
		gotOrder = append(gotOrder, origMap[key])
	}

	return origOrder, wantOrder, gotOrder
}

func createTestPubKeys(t *testing.T, number, offset int) []*hdkeychain.ExtendedKey {
	xpubRaw := "xpub661MyMwAqRbcFwdnYF5mvCBY54vaLdJf8c5ugJTp5p7PqF9J1USgBx12qYMnZ9yUiswV7smbQ1DSweMqu8wn7Jociz4PWkuJ6EPvoVEgMw7"
	xpubKey, err := hdkeychain.NewKeyFromString(xpubRaw)
	if err != nil {
		t.Fatalf("Failed to generate new key: %v", err)
	}

	keys := make([]*hdkeychain.ExtendedKey, number)
	for i := uint32(0); i < uint32(len(keys)); i++ {
		chPubKey, err := xpubKey.Child(i + uint32(offset))
		if err != nil {
			t.Fatalf("Failed to generate child key: %v", err)
		}
		keys[i] = chPubKey
	}
	return keys
}

func TestReverse(t *testing.T) {
//测试用于反转公钥列表的实用程序功能。
//11是任意的。
	for numKeys := 0; numKeys < 11; numKeys++ {
		keys := createTestPubKeys(t, numKeys, 0)
		revRevKeys := reverse(reverse(keys))
		if len(keys) != len(revRevKeys) {
			t.Errorf("Reverse(Reverse(x)): the no. pubkeys changed. Got %d, want %d",
				len(revRevKeys), len(keys))
		}

		for i := 0; i < len(keys); i++ {
			if keys[i] != revRevKeys[i] {
				t.Errorf("Reverse(Reverse(x)) != x. Got %v, want %v",
					revRevKeys[i], keys[i])
			}
		}
	}
}

func TestEmpowerSeriesNeuterFailed(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, _ := vp.TstRWNamespaces(dbtx)

	seriesID := uint32(1)
	err = pool.CreateSeries(ns, 1, seriesID, 2, vp.TstPubKeys[0:3])
	if err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

//具有错误版本（0xffffffff）的私钥将触发
//（k*extendedkey）.neuter中的错误和相关的错误路径
//在授权系列中。
	badKey := "wM5uZBNTYmaYGiK8VaGi7zPGbZGLuQgDiR2Zk4nGfbRFLXwHGcMUdVdazRpNHFSR7X7WLmzzbAq8dA1ViN6eWKgKqPye1rJTDQTvBiXvZ7E3nmdx"
	err = pool.EmpowerSeries(ns, seriesID, badKey)

	vp.TstCheckError(t, "", err, vp.ErrKeyNeuter)
}

func TestDecryptExtendedKeyCannotCreateResultKey(t *testing.T) {
	tearDown, _, pool := vp.TstCreatePool(t)
	defer tearDown()

//明文不是base58编码的，会触发错误
	cipherText, err := pool.Manager().Encrypt(waddrmgr.CKTPublic, []byte("not-base58-encoded"))
	if err != nil {
		t.Fatalf("Failed to encrypt plaintext: %v", err)
	}

	_, err = pool.TstDecryptExtendedKey(waddrmgr.CKTPublic, cipherText)

	vp.TstCheckError(t, "", err, vp.ErrKeyChain)
}

func TestDecryptExtendedKeyCannotDecrypt(t *testing.T) {
	tearDown, _, pool := vp.TstCreatePool(t)
	defer tearDown()

	_, err := pool.TstDecryptExtendedKey(waddrmgr.CKTPublic, []byte{})

	vp.TstCheckError(t, "", err, vp.ErrCrypto)
}

func TestPoolChangeAddress(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()

	pubKeys := vp.TstPubKeys[1:4]
	vp.TstCreateSeries(t, dbtx, pool, []vp.TstSeriesDef{{ReqSigs: 2, PubKeys: pubKeys, SeriesID: 1}})

	addr := vp.TstNewChangeAddress(t, pool, 1, 0)
	checkPoolAddress(t, addr, 1, 0, 0)

//当序列不活动时，我们应该得到一个错误。
	pubKeys = vp.TstPubKeys[3:6]
	vp.TstCreateSeries(t, dbtx, pool,
		[]vp.TstSeriesDef{{ReqSigs: 2, PubKeys: pubKeys, SeriesID: 2, Inactive: true}})
	_, err = pool.ChangeAddress(2, 0)
	vp.TstCheckError(t, "", err, vp.ErrSeriesNotActive)
}

func TestPoolWithdrawalAddress(t *testing.T) {
	tearDown, db, pool := vp.TstCreatePool(t)
	defer tearDown()

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns, addrmgrNs := vp.TstRWNamespaces(dbtx)

	pubKeys := vp.TstPubKeys[1:4]
	vp.TstCreateSeries(t, dbtx, pool, []vp.TstSeriesDef{{ReqSigs: 2, PubKeys: pubKeys, SeriesID: 1}})
	addr := vp.TstNewWithdrawalAddress(t, dbtx, pool, 1, 0, 0)
	checkPoolAddress(t, addr, 1, 0, 0)

//当请求的地址不在已用地址集中时
//对于那个池，我们应该得到一个错误。
	_, err = pool.WithdrawalAddress(ns, addrmgrNs, 1, 2, 3)
	vp.TstCheckError(t, "", err, vp.ErrWithdrawFromUnusedAddr)
}

func checkPoolAddress(t *testing.T, addr vp.PoolAddress, seriesID uint32, branch vp.Branch,
	index vp.Index) {

	if addr.SeriesID() != seriesID {
		t.Fatalf("Wrong SeriesID; got %d, want %d", addr.SeriesID(), seriesID)
	}
	if addr.Branch() != branch {
		t.Fatalf("Wrong Branch; got %d, want %d", addr.Branch(), branch)
	}
	if addr.Index() != index {
		t.Fatalf("Wrong Index; got %d, want %d", addr.Index(), index)
	}
}
