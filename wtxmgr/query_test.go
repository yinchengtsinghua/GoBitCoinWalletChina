
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wtxmgr

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/walletdb"
)

type queryState struct {
//切片项按高度排序，mempool排在最后。
	blocks    [][]TxDetails
	txDetails map[chainhash.Hash][]TxDetails
}

func newQueryState() *queryState {
	return &queryState{
		txDetails: make(map[chainhash.Hash][]TxDetails),
	}
}

func (q *queryState) deepCopy() *queryState {
	cpy := newQueryState()
	for _, blockDetails := range q.blocks {
		var cpyDetails []TxDetails
		for _, detail := range blockDetails {
			cpyDetails = append(cpyDetails, *deepCopyTxDetails(&detail))
		}
		cpy.blocks = append(cpy.blocks, cpyDetails)
	}
	cpy.txDetails = make(map[chainhash.Hash][]TxDetails)
	for txHash, details := range q.txDetails {
		detailsSlice := make([]TxDetails, len(details))
		for i, detail := range details {
			detailsSlice[i] = *deepCopyTxDetails(&detail)
		}
		cpy.txDetails[txHash] = detailsSlice
	}
	return cpy
}

func deepCopyTxDetails(d *TxDetails) *TxDetails {
	cpy := *d
	cpy.MsgTx = *d.MsgTx.Copy()
	if cpy.SerializedTx != nil {
		cpy.SerializedTx = make([]byte, len(cpy.SerializedTx))
		copy(cpy.SerializedTx, d.SerializedTx)
	}
	cpy.Credits = make([]CreditRecord, len(d.Credits))
	copy(cpy.Credits, d.Credits)
	cpy.Debits = make([]DebitRecord, len(d.Debits))
	copy(cpy.Debits, d.Debits)
	return &cpy
}

func (q *queryState) compare(s *Store, ns walletdb.ReadBucket,
	changeDesc string) error {

	fwdBlocks := q.blocks
	revBlocks := make([][]TxDetails, len(q.blocks))
	copy(revBlocks, q.blocks)
	for i := 0; i < len(revBlocks)/2; i++ {
		revBlocks[i], revBlocks[len(revBlocks)-1-i] = revBlocks[len(revBlocks)-1-i], revBlocks[i]
	}
	checkBlock := func(blocks [][]TxDetails) func([]TxDetails) (bool, error) {
		return func(got []TxDetails) (bool, error) {
			if len(fwdBlocks) == 0 {
				return false, errors.New("entered range " +
					"when no more details expected")
			}
			exp := blocks[0]
			if len(got) != len(exp) {
				return false, fmt.Errorf("got len(details)=%d "+
					"in transaction range, expected %d",
					len(got), len(exp))
			}
			for i := range got {
				err := equalTxDetails(&got[i], &exp[i])
				if err != nil {
					return false, fmt.Errorf("failed "+
						"comparing range of "+
						"transaction details: %v", err)
				}
			}
			blocks = blocks[1:]
			return false, nil
		}
	}
	err := s.RangeTransactions(ns, 0, -1, checkBlock(fwdBlocks))
	if err != nil {
		return fmt.Errorf("%s: failed in RangeTransactions (forwards "+
			"iteration): %v", changeDesc, err)
	}
	err = s.RangeTransactions(ns, -1, 0, checkBlock(revBlocks))
	if err != nil {
		return fmt.Errorf("%s: failed in RangeTransactions (reverse "+
			"iteration): %v", changeDesc, err)
	}

	for txHash, details := range q.txDetails {
		for _, detail := range details {
			blk := &detail.Block.Block
			if blk.Height == -1 {
				blk = nil
			}
			d, err := s.UniqueTxDetails(ns, &txHash, blk)
			if err != nil {
				return err
			}
			if d == nil {
				return fmt.Errorf("found no matching "+
					"transaction at height %d",
					detail.Block.Height)
			}
			if err := equalTxDetails(d, &detail); err != nil {
				return fmt.Errorf("%s: failed querying latest "+
					"details regarding transaction %v",
					changeDesc, txHash)
			}
		}

//对于最近使用此哈希的Tx，请检查
//txdetails（不查找任何特定的tx
//高度）与最后一个匹配。
		detail := &details[len(details)-1]
		d, err := s.TxDetails(ns, &txHash)
		if err != nil {
			return err
		}
		if err := equalTxDetails(d, detail); err != nil {
			return fmt.Errorf("%s: failed querying latest details "+
				"regarding transaction %v", changeDesc, txHash)
		}
	}

	return nil
}

func equalTxDetails(got, exp *TxDetails) error {
//需要避免对切片使用reflect.deepequal，因为它
//对于零与非零零零长度切片，返回false。
	if err := equalTxs(&got.MsgTx, &exp.MsgTx); err != nil {
		return err
	}

	if got.Hash != exp.Hash {
		return fmt.Errorf("found mismatched hashes: got %v, expected %v",
			got.Hash, exp.Hash)
	}
	if got.Received != exp.Received {
		return fmt.Errorf("found mismatched receive time: got %v, "+
			"expected %v", got.Received, exp.Received)
	}
	if !bytes.Equal(got.SerializedTx, exp.SerializedTx) {
		return fmt.Errorf("found mismatched serialized txs: got %v, "+
			"expected %v", got.SerializedTx, exp.SerializedTx)
	}
	if got.Block != exp.Block {
		return fmt.Errorf("found mismatched block meta: got %v, "+
			"expected %v", got.Block, exp.Block)
	}
	if len(got.Credits) != len(exp.Credits) {
		return fmt.Errorf("credit slice lengths differ: got %d, "+
			"expected %d", len(got.Credits), len(exp.Credits))
	}
	for i := range got.Credits {
		if got.Credits[i] != exp.Credits[i] {
			return fmt.Errorf("found mismatched credit[%d]: got %v, "+
				"expected %v", i, got.Credits[i], exp.Credits[i])
		}
	}
	if len(got.Debits) != len(exp.Debits) {
		return fmt.Errorf("debit slice lengths differ: got %d, "+
			"expected %d", len(got.Debits), len(exp.Debits))
	}
	for i := range got.Debits {
		if got.Debits[i] != exp.Debits[i] {
			return fmt.Errorf("found mismatched debit[%d]: got %v, "+
				"expected %v", i, got.Debits[i], exp.Debits[i])
		}
	}

	return nil
}

func equalTxs(got, exp *wire.MsgTx) error {
	var bufGot, bufExp bytes.Buffer
	err := got.Serialize(&bufGot)
	if err != nil {
		return err
	}
	err = exp.Serialize(&bufExp)
	if err != nil {
		return err
	}
	if !bytes.Equal(bufGot.Bytes(), bufExp.Bytes()) {
		return fmt.Errorf("found unexpected wire.MsgTx: got: %v, "+
			"expected %v", got, exp)
	}

	return nil
}

//返回time.now（）和秒分辨率，这是存储所保存的。
func timeNow() time.Time {
	return time.Unix(time.Now().Unix(), 0)
}

//返回不带序列化Tx的TxRecord的副本。
func stripSerializedTx(rec *TxRecord) *TxRecord {
	ret := *rec
	ret.SerializedTx = nil
	return &ret
}

func makeBlockMeta(height int32) BlockMeta {
	if height == -1 {
		return BlockMeta{Block: Block{Height: -1}}
	}

	b := BlockMeta{
		Block: Block{Height: height},
		Time:  timeNow(),
	}
//给它一个根据高度和时间创建的假块散列。
	binary.LittleEndian.PutUint32(b.Hash[0:4], uint32(height))
	binary.LittleEndian.PutUint64(b.Hash[4:12], uint64(b.Time.Unix()))
	return b
}

func TestStoreQueries(t *testing.T) {
	t.Parallel()

	type queryTest struct {
		desc    string
		updates func(ns walletdb.ReadWriteBucket) error
		state   *queryState
	}
	var tests []queryTest

//创建存储并测试初始状态。
	s, db, teardown, err := testStore()
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}
	lastState := newQueryState()
	tests = append(tests, queryTest{
		desc:    "initial store",
		updates: func(walletdb.ReadWriteBucket) error { return nil },
		state:   lastState,
	})

//插入未限定的事务。没有学分。
	txA := spendOutput(&chainhash.Hash{}, 0, 100e8)
	recA, err := NewTxRecordFromMsgTx(txA, timeNow())
	if err != nil {
		t.Fatal(err)
	}
	newState := lastState.deepCopy()
	newState.blocks = [][]TxDetails{
		{
			{
				TxRecord: *stripSerializedTx(recA),
				Block:    BlockMeta{Block: Block{Height: -1}},
			},
		},
	}
	newState.txDetails[recA.Hash] = []TxDetails{
		newState.blocks[0][0],
	}
	lastState = newState
	tests = append(tests, queryTest{
		desc: "insert tx A unmined",
		updates: func(ns walletdb.ReadWriteBucket) error {
			return s.InsertTx(ns, recA, nil)
		},
		state: newState,
	})

//添加txa:0作为更改信用证。
	newState = lastState.deepCopy()
	newState.blocks[0][0].Credits = []CreditRecord{
		{
			Index:  0,
			Amount: btcutil.Amount(recA.MsgTx.TxOut[0].Value),
			Spent:  false,
			Change: true,
		},
	}
	newState.txDetails[recA.Hash][0].Credits = newState.blocks[0][0].Credits
	lastState = newState
	tests = append(tests, queryTest{
		desc: "mark unconfirmed txA:0 as credit",
		updates: func(ns walletdb.ReadWriteBucket) error {
			return s.AddCredit(ns, recA, nil, 0, true)
		},
		state: newState,
	})

//插入另一个花费txa:0的未限定事务，拆分
//输出量为40和60 BTC。
	txB := spendOutput(&recA.Hash, 0, 40e8, 60e8)
	recB, err := NewTxRecordFromMsgTx(txB, timeNow())
	if err != nil {
		t.Fatal(err)
	}
	newState = lastState.deepCopy()
	newState.blocks[0][0].Credits[0].Spent = true
	newState.blocks[0] = append(newState.blocks[0], TxDetails{
		TxRecord: *stripSerializedTx(recB),
		Block:    BlockMeta{Block: Block{Height: -1}},
		Debits: []DebitRecord{
			{
				Amount: btcutil.Amount(recA.MsgTx.TxOut[0].Value),
Index:  0, //recb.msgtx.txin索引
			},
		},
	})
	newState.txDetails[recA.Hash][0].Credits[0].Spent = true
	newState.txDetails[recB.Hash] = []TxDetails{newState.blocks[0][1]}
	lastState = newState
	tests = append(tests, queryTest{
		desc: "insert tx B unmined",
		updates: func(ns walletdb.ReadWriteBucket) error {
			return s.InsertTx(ns, recB, nil)
		},
		state: newState,
	})
	newState = lastState.deepCopy()
	newState.blocks[0][1].Credits = []CreditRecord{
		{
			Index:  0,
			Amount: btcutil.Amount(recB.MsgTx.TxOut[0].Value),
			Spent:  false,
			Change: false,
		},
	}
	newState.txDetails[recB.Hash][0].Credits = newState.blocks[0][1].Credits
	lastState = newState
	tests = append(tests, queryTest{
		desc: "mark txB:0 as non-change credit",
		updates: func(ns walletdb.ReadWriteBucket) error {
			return s.AddCredit(ns, recB, nil, 0, false)
		},
		state: newState,
	})

//我的TX A在100区。使tx b保持未链接状态。
	b100 := makeBlockMeta(100)
	newState = lastState.deepCopy()
	newState.blocks[0] = newState.blocks[0][:1]
	newState.blocks[0][0].Block = b100
	newState.blocks = append(newState.blocks, lastState.blocks[0][1:])
	newState.txDetails[recA.Hash][0].Block = b100
	lastState = newState
	tests = append(tests, queryTest{
		desc: "mine tx A",
		updates: func(ns walletdb.ReadWriteBucket) error {
			return s.InsertTx(ns, recA, &b100)
		},
		state: newState,
	})

//在101区开采TX B。
	b101 := makeBlockMeta(101)
	newState = lastState.deepCopy()
	newState.blocks[1][0].Block = b101
	newState.txDetails[recB.Hash][0].Block = b101
	lastState = newState
	tests = append(tests, queryTest{
		desc: "mine tx B",
		updates: func(ns walletdb.ReadWriteBucket) error {
			return s.InsertTx(ns, recB, &b101)
		},
		state: newState,
	})

	for _, tst := range tests {
		err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
			ns := tx.ReadWriteBucket(namespaceKey)
			if err := tst.updates(ns); err != nil {
				return err
			}
			return tst.state.compare(s, ns, tst.desc)
		})
		if err != nil {
			t.Fatal(err)
		}
	}

//使用当前存储的状态运行一些其他查询测试：
//-验证查询不在存储区中的事务是否返回
//没有失败。
//-验证在错误的块中查询唯一事务
//无故障返回零。
//-验证在RangeTransactions上提前中断是否会进一步停止
//迭代。

	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := tx.ReadWriteBucket(namespaceKey)

		missingTx := spendOutput(&recB.Hash, 0, 40e8)
		missingRec, err := NewTxRecordFromMsgTx(missingTx, timeNow())
		if err != nil {
			return err
		}
		missingBlock := makeBlockMeta(102)
		missingDetails, err := s.TxDetails(ns, &missingRec.Hash)
		if err != nil {
			return err
		}
		if missingDetails != nil {
			return fmt.Errorf("Expected no details, found details "+
				"for tx %v", missingDetails.Hash)
		}
		missingUniqueTests := []struct {
			hash  *chainhash.Hash
			block *Block
		}{
			{&missingRec.Hash, &b100.Block},
			{&missingRec.Hash, &missingBlock.Block},
			{&missingRec.Hash, nil},
			{&recB.Hash, &b100.Block},
			{&recB.Hash, &missingBlock.Block},
			{&recB.Hash, nil},
		}
		for _, tst := range missingUniqueTests {
			missingDetails, err = s.UniqueTxDetails(ns, tst.hash, tst.block)
			if err != nil {
				t.Fatal(err)
			}
			if missingDetails != nil {
				t.Errorf("Expected no details, found details for tx %v", missingDetails.Hash)
			}
		}

		iterations := 0
		err = s.RangeTransactions(ns, 0, -1, func([]TxDetails) (bool, error) {
			iterations++
			return true, nil
		})
		if iterations != 1 {
			t.Errorf("RangeTransactions (forwards) ran func %d times", iterations)
		}
		iterations = 0
		err = s.RangeTransactions(ns, -1, 0, func([]TxDetails) (bool, error) {
			iterations++
			return true, nil
		})
		if iterations != 1 {
			t.Errorf("RangeTransactions (reverse) ran func %d times", iterations)
		}
//确保它也在一次迭代之后通过未链接的事务提前中断。
		if err := s.Rollback(ns, b101.Height); err != nil {
			return err
		}
		iterations = 0
		err = s.RangeTransactions(ns, -1, 0, func([]TxDetails) (bool, error) {
			iterations++
			return true, nil
		})
		if iterations != 1 {
			t.Errorf("RangeTransactions (reverse) ran func %d times", iterations)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

//以上测试都没有测试具有多个
//每个块的txs，现在就这样做。首先将Tx B移动到块100
//（与TXA相同的块），然后从块100开始回滚，因此
//这两个都是未连接的。
	newState = lastState.deepCopy()
	newState.blocks[0] = append(newState.blocks[0], newState.blocks[1]...)
	newState.blocks[0][1].Block = b100
	newState.blocks = newState.blocks[:1]
	newState.txDetails[recB.Hash][0].Block = b100
	lastState = newState
	tests = append(tests[:0:0], queryTest{
		desc: "move tx B to block 100",
		updates: func(ns walletdb.ReadWriteBucket) error {
			return s.InsertTx(ns, recB, &b100)
		},
		state: newState,
	})
	newState = lastState.deepCopy()
	newState.blocks[0][0].Block = makeBlockMeta(-1)
	newState.blocks[0][1].Block = makeBlockMeta(-1)
	newState.txDetails[recA.Hash][0].Block = makeBlockMeta(-1)
	newState.txDetails[recB.Hash][0].Block = makeBlockMeta(-1)
	lastState = newState
	tests = append(tests, queryTest{
		desc: "rollback block 100",
		updates: func(ns walletdb.ReadWriteBucket) error {
			return s.Rollback(ns, b100.Height)
		},
		state: newState,
	})

	for _, tst := range tests {
		err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
			ns := tx.ReadWriteBucket(namespaceKey)
			if err := tst.updates(ns); err != nil {
				return err
			}
			return tst.state.compare(s, ns, tst.desc)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestPreviousPkScripts(t *testing.T) {
	t.Parallel()

	s, db, teardown, err := testStore()
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

//脚本无效，但足以进行测试。
	var (
		scriptA0 = []byte("tx A output 0")
		scriptA1 = []byte("tx A output 1")
		scriptB0 = []byte("tx B output 0")
		scriptB1 = []byte("tx B output 1")
		scriptC0 = []byte("tx C output 0")
		scriptC1 = []byte("tx C output 1")
	)

//创建一个事务，花费两个以前的输出并生成两个
//新输出传递的pkscipts。花费PrevHash的输出0和1。
	buildTx := func(prevHash *chainhash.Hash, script0, script1 []byte) *wire.MsgTx {
		return &wire.MsgTx{
			TxIn: []*wire.TxIn{
				{PreviousOutPoint: wire.OutPoint{
					Hash:  *prevHash,
					Index: 0,
				}},
				{PreviousOutPoint: wire.OutPoint{
					Hash: *prevHash, Index: 1,
				}},
			},
			TxOut: []*wire.TxOut{
				{Value: 1e8, PkScript: script0},
				{Value: 1e8, PkScript: script1},
			},
		}
	}

	newTxRecordFromMsgTx := func(tx *wire.MsgTx) *TxRecord {
		rec, err := NewTxRecordFromMsgTx(tx, timeNow())
		if err != nil {
			t.Fatal(err)
		}
		return rec
	}

//使用假输出脚本创建事务。
	var (
		txA  = buildTx(&chainhash.Hash{}, scriptA0, scriptA1)
		recA = newTxRecordFromMsgTx(txA)
		txB  = buildTx(&recA.Hash, scriptB0, scriptB1)
		recB = newTxRecordFromMsgTx(txB)
		txC  = buildTx(&recB.Hash, scriptC0, scriptC1)
		recC = newTxRecordFromMsgTx(txC)
		txD  = buildTx(&recC.Hash, nil, nil)
		recD = newTxRecordFromMsgTx(txD)
	)

	insertTx := func(ns walletdb.ReadWriteBucket, rec *TxRecord, block *BlockMeta) {
		err := s.InsertTx(ns, rec, block)
		if err != nil {
			t.Fatal(err)
		}
	}
	addCredit := func(ns walletdb.ReadWriteBucket, rec *TxRecord, block *BlockMeta, index uint32) {
		err := s.AddCredit(ns, rec, block, index, false)
		if err != nil {
			t.Fatal(err)
		}
	}

	type scriptTest struct {
		rec     *TxRecord
		block   *Block
		scripts [][]byte
	}
	runTest := func(ns walletdb.ReadWriteBucket, tst *scriptTest) {
		scripts, err := s.PreviousPkScripts(ns, tst.rec, tst.block)
		if err != nil {
			t.Fatal(err)
		}
		height := int32(-1)
		if tst.block != nil {
			height = tst.block.Height
		}
		if len(scripts) != len(tst.scripts) {
			t.Errorf("Transaction %v height %d: got len(scripts)=%d, expected %d",
				tst.rec.Hash, height, len(scripts), len(tst.scripts))
			return
		}
		for i := range scripts {
			if !bytes.Equal(scripts[i], tst.scripts[i]) {
//用%s格式化脚本，因为它们是（应该是）ASCII。
				t.Errorf("Transaction %v height %d script %d: got '%s' expected '%s'",
					tst.rec.Hash, height, i, scripts[i], tst.scripts[i])
			}
		}
	}

	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		t.Fatal(err)
	}
	defer dbtx.Commit()
	ns := dbtx.ReadWriteBucket(namespaceKey)

//插入交易A-C，但不标记信用。直到
//这些标记为信用证，previouspkscript不应返回
//他们。
	insertTx(ns, recA, nil)
	insertTx(ns, recB, nil)
	insertTx(ns, recC, nil)

	b100 := makeBlockMeta(100)
	b101 := makeBlockMeta(101)

	tests := []scriptTest{
		{recA, nil, nil},
		{recA, &b100.Block, nil},
		{recB, nil, nil},
		{recB, &b100.Block, nil},
		{recC, nil, nil},
		{recC, &b100.Block, nil},
	}
	for _, tst := range tests {
		runTest(ns, &tst)
	}
	if t.Failed() {
		t.Fatal("Failed after unmined tx inserts")
	}

//马克学分。Tx C输出1未标记为信用：Tx D将花费
//稍后，但在挖掘C时，输出1的脚本不应
//返回。
	addCredit(ns, recA, nil, 0)
	addCredit(ns, recA, nil, 1)
	addCredit(ns, recB, nil, 0)
	addCredit(ns, recB, nil, 1)
	addCredit(ns, recC, nil, 0)
	tests = []scriptTest{
		{recA, nil, nil},
		{recA, &b100.Block, nil},
		{recB, nil, [][]byte{scriptA0, scriptA1}},
		{recB, &b100.Block, nil},
		{recC, nil, [][]byte{scriptB0, scriptB1}},
		{recC, &b100.Block, nil},
	}
	for _, tst := range tests {
		runTest(ns, &tst)
	}
	if t.Failed() {
		t.Fatal("Failed after marking unmined credits")
	}

//我的TX A在100区。测试结果应相同。
	insertTx(ns, recA, &b100)
	for _, tst := range tests {
		runTest(ns, &tst)
	}
	if t.Failed() {
		t.Fatal("Failed after mining tx A")
	}

//在101区开采Tx B。
	insertTx(ns, recB, &b101)
	tests = []scriptTest{
		{recA, nil, nil},
		{recA, &b100.Block, nil},
		{recB, nil, nil},
		{recB, &b101.Block, [][]byte{scriptA0, scriptA1}},
		{recC, nil, [][]byte{scriptB0, scriptB1}},
		{recC, &b101.Block, nil},
	}
	for _, tst := range tests {
		runTest(ns, &tst)
	}
	if t.Failed() {
		t.Fatal("Failed after mining tx B")
	}

//在101块（与TX B相同的块）中挖掘TX C，以测试
//同一块。
	insertTx(ns, recC, &b101)
	tests = []scriptTest{
		{recA, nil, nil},
		{recA, &b100.Block, nil},
		{recB, nil, nil},
		{recB, &b101.Block, [][]byte{scriptA0, scriptA1}},
		{recC, nil, nil},
		{recC, &b101.Block, [][]byte{scriptB0, scriptB1}},
	}
	for _, tst := range tests {
		runTest(ns, &tst)
	}
	if t.Failed() {
		t.Fatal("Failed after mining tx C")
	}

//插入tx d，花费c:0和c:1。但是，只标记了C:0
//作为奖励，只应返回输出脚本。
	insertTx(ns, recD, nil)
	tests = append(tests, scriptTest{recD, nil, [][]byte{scriptC0}})
	tests = append(tests, scriptTest{recD, &b101.Block, nil})
	for _, tst := range tests {
		runTest(ns, &tst)
	}
	if t.Failed() {
		t.Fatal("Failed after inserting tx D")
	}
}
