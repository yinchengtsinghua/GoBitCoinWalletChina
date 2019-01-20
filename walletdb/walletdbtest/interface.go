
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package walletdbtest

import (
	"fmt"
	"os"
	"reflect"

	"github.com/btcsuite/btcwallet/walletdb"
)

//errSubestFail用于指示子测试返回了false。
var errSubTestFail = fmt.Errorf("sub test failure")

//test context用于存储有关正在运行的测试的上下文信息，该测试
//传递到helper函数中。
type testContext struct {
	t           Tester
	db          walletdb.DB
	bucketDepth int
	isWritable  bool
}

//RollbackValues返回所提供映射的副本，其中所有值都设置为
//空字符串。这用于测试值是否正确回滚。
func rollbackValues(values map[string]string) map[string]string {
	retMap := make(map[string]string, len(values))
	for k := range values {
		retMap[k] = ""
	}
	return retMap
}

//testGetValues检查提供的所有键/值对是否可以
//从数据库检索，检索到的值与提供的
//价值观。
func testGetValues(tc *testContext, bucket walletdb.ReadBucket, values map[string]string) bool {
	for k, v := range values {
		var vBytes []byte
		if v != "" {
			vBytes = []byte(v)
		}

		gotValue := bucket.Get([]byte(k))
		if !reflect.DeepEqual(gotValue, vBytes) {
			tc.t.Errorf("Get: unexpected value - got %s, want %s",
				gotValue, vBytes)
			return false
		}
	}

	return true
}

//testputvalues将提供的所有键/值对存储在
//同时检查错误。
func testPutValues(tc *testContext, bucket walletdb.ReadWriteBucket, values map[string]string) bool {
	for k, v := range values {
		var vBytes []byte
		if v != "" {
			vBytes = []byte(v)
		}
		if err := bucket.Put([]byte(k), vBytes); err != nil {
			tc.t.Errorf("Put: unexpected error: %v", err)
			return false
		}
	}

	return true
}

//testDeleteValues从
//提供桶。
func testDeleteValues(tc *testContext, bucket walletdb.ReadWriteBucket, values map[string]string) bool {
	for k := range values {
		if err := bucket.Delete([]byte(k)); err != nil {
			tc.t.Errorf("Delete: unexpected error: %v", err)
			return false
		}
	}

	return true
}

//TestNestedReadWriteBucket针对嵌套的Bucket重新运行TestBucketInterface
//用计数器只测试几个水平深度。
func testNestedReadWriteBucket(tc *testContext, testBucket walletdb.ReadWriteBucket) bool {
//不要超过2层嵌套深度。
	if tc.bucketDepth > 1 {
		return true
	}

	tc.bucketDepth++
	defer func() {
		tc.bucketDepth--
	}()
	if !testReadWriteBucketInterface(tc, testBucket) {
		return false
	}

	return true
}

//testreadwritebucketinterface通过以下方式确保bucket接口正常工作：
//行使其所有职能。
func testReadWriteBucketInterface(tc *testContext, bucket walletdb.ReadWriteBucket) bool {
//keyValues保存放置时要使用的键和值
//值进入存储桶。
	var keyValues = map[string]string{
		"bucketkey1": "foo1",
		"bucketkey2": "foo2",
		"bucketkey3": "foo3",
	}
	if !testPutValues(tc, bucket, keyValues) {
		return false
	}

	if !testGetValues(tc, bucket, keyValues) {
		return false
	}

//迭代使用foreach的所有键，同时确保
//存储的值是预期值。
	keysFound := make(map[string]struct{}, len(keyValues))
	err := bucket.ForEach(func(k, v []byte) error {
		ks := string(k)
		wantV, ok := keyValues[ks]
		if !ok {
			return fmt.Errorf("ForEach: key '%s' should "+
				"exist", ks)
		}

		if !reflect.DeepEqual(v, []byte(wantV)) {
			return fmt.Errorf("ForEach: value for key '%s' "+
				"does not match - got %s, want %s",
				ks, v, wantV)
		}

		keysFound[ks] = struct{}{}
		return nil
	})
	if err != nil {
		tc.t.Errorf("%v", err)
		return false
	}

//确保所有键都已迭代。
	for k := range keyValues {
		if _, ok := keysFound[k]; !ok {
			tc.t.Errorf("ForEach: key '%s' was not iterated "+
				"when it should have been", k)
			return false
		}
	}

//删除密钥并确保它们已被删除。
	if !testDeleteValues(tc, bucket, keyValues) {
		return false
	}
	if !testGetValues(tc, bucket, rollbackValues(keyValues)) {
		return false
	}

//确保创建新存储桶按预期工作。
	testBucketName := []byte("testbucket")
	testBucket, err := bucket.CreateBucket(testBucketName)
	if err != nil {
		tc.t.Errorf("CreateBucket: unexpected error: %v", err)
		return false
	}
	if !testNestedReadWriteBucket(tc, testBucket) {
		return false
	}

//确保创建已存在的bucket失败
//期望误差。
	wantErr := walletdb.ErrBucketExists
	if _, err := bucket.CreateBucket(testBucketName); err != wantErr {
		tc.t.Errorf("CreateBucket: unexpected error - got %v, "+
			"want %v", err, wantErr)
		return false
	}

//确保CreateBacketifNotexists返回现有Bucket。
	testBucket, err = bucket.CreateBucketIfNotExists(testBucketName)
	if err != nil {
		tc.t.Errorf("CreateBucketIfNotExists: unexpected "+
			"error: %v", err)
		return false
	}
	if !testNestedReadWriteBucket(tc, testBucket) {
		return false
	}

//确保回收和现有桶按预期工作。
	testBucket = bucket.NestedReadWriteBucket(testBucketName)
	if !testNestedReadWriteBucket(tc, testBucket) {
		return false
	}

//确保删除存储桶按预期工作。
	if err := bucket.DeleteNestedBucket(testBucketName); err != nil {
		tc.t.Errorf("DeleteNestedBucket: unexpected error: %v", err)
		return false
	}
	if b := bucket.NestedReadWriteBucket(testBucketName); b != nil {
		tc.t.Errorf("DeleteNestedBucket: bucket '%s' still exists",
			testBucketName)
		return false
	}

//确保删除不存在的存储桶返回
//期望误差。
	wantErr = walletdb.ErrBucketNotFound
	if err := bucket.DeleteNestedBucket(testBucketName); err != wantErr {
		tc.t.Errorf("DeleteNestedBucket: unexpected error - got %v, "+
			"want %v", err, wantErr)
		return false
	}

//确保CreateBacketifNotexists在以下情况下创建新bucket：
//它还不存在。
	testBucket, err = bucket.CreateBucketIfNotExists(testBucketName)
	if err != nil {
		tc.t.Errorf("CreateBucketIfNotExists: unexpected "+
			"error: %v", err)
		return false
	}
	if !testNestedReadWriteBucket(tc, testBucket) {
		return false
	}

//删除测试存储桶以避免将来留下它
//电话。
	if err := bucket.DeleteNestedBucket(testBucketName); err != nil {
		tc.t.Errorf("DeleteNestedBucket: unexpected error: %v", err)
		return false
	}
	if b := bucket.NestedReadWriteBucket(testBucketName); b != nil {
		tc.t.Errorf("DeleteNestedBucket: bucket '%s' still exists",
			testBucketName)
		return false
	}
	return true
}

//TestManualTxInterface确保手动事务按预期工作。
func testManualTxInterface(tc *testContext, bucketKey []byte) bool {
	db := tc.db

//填充值的PopulateValues测试按预期工作。
//
//当可写标志为false时，将创建只读转换，
//执行只读事务的标准存储桶测试，以及
//检查commit函数以确保它按预期失败。
//
//否则，将创建读写事务，值为
//读写事务的标准存储桶测试是
//执行，然后提交或滚动事务
//返回取决于标志。
	populateValues := func(writable, rollback bool, putValues map[string]string) bool {
		var dbtx walletdb.ReadTx
		var rootBucket walletdb.ReadBucket
		var err error
		if writable {
			dbtx, err = db.BeginReadWriteTx()
			if err != nil {
				tc.t.Errorf("BeginReadWriteTx: unexpected error %v", err)
				return false
			}
			rootBucket = dbtx.(walletdb.ReadWriteTx).ReadWriteBucket(bucketKey)
		} else {
			dbtx, err = db.BeginReadTx()
			if err != nil {
				tc.t.Errorf("BeginReadTx: unexpected error %v", err)
				return false
			}
			rootBucket = dbtx.ReadBucket(bucketKey)
		}
		if rootBucket == nil {
			tc.t.Errorf("ReadWriteBucket/ReadBucket: unexpected nil root bucket")
			_ = dbtx.Rollback()
			return false
		}

		if writable {
			tc.isWritable = writable
			if !testReadWriteBucketInterface(tc, rootBucket.(walletdb.ReadWriteBucket)) {
				_ = dbtx.Rollback()
				return false
			}
		}

		if !writable {
//回滚事务。
			if err := dbtx.Rollback(); err != nil {
				tc.t.Errorf("Commit: unexpected error %v", err)
				return false
			}
		} else {
			rootBucket := rootBucket.(walletdb.ReadWriteBucket)
			if !testPutValues(tc, rootBucket, putValues) {
				return false
			}

			if rollback {
//回滚事务。
				if err := dbtx.Rollback(); err != nil {
					tc.t.Errorf("Rollback: unexpected "+
						"error %v", err)
					return false
				}
			} else {
//承诺应该成功。
				if err := dbtx.(walletdb.ReadWriteTx).Commit(); err != nil {
					tc.t.Errorf("Commit: unexpected error "+
						"%v", err)
					return false
				}
			}
		}

		return true
	}

//checkvalues启动一个只读事务并检查
//ExpectedValues参数中指定的键/值对匹配
//数据库中有什么。
	checkValues := func(expectedValues map[string]string) bool {
//开始另一个只读事务以确保…
		dbtx, err := db.BeginReadTx()
		if err != nil {
			tc.t.Errorf("BeginReadTx: unexpected error %v", err)
			return false
		}

		rootBucket := dbtx.ReadBucket(bucketKey)
		if rootBucket == nil {
			tc.t.Errorf("ReadBucket: unexpected nil root bucket")
			_ = dbtx.Rollback()
			return false
		}

		if !testGetValues(tc, rootBucket, expectedValues) {
			_ = dbtx.Rollback()
			return false
		}

//回滚只读事务。
		if err := dbtx.Rollback(); err != nil {
			tc.t.Errorf("Commit: unexpected error %v", err)
			return false
		}

		return true
	}

//DeleteValues启动读写事务并删除键
//在传递的键/值对中。
	deleteValues := func(values map[string]string) bool {
		dbtx, err := db.BeginReadWriteTx()
		if err != nil {
			tc.t.Errorf("BeginReadWriteTx: unexpected error %v", err)
			_ = dbtx.Rollback()
			return false
		}

		rootBucket := dbtx.ReadWriteBucket(bucketKey)
		if rootBucket == nil {
			tc.t.Errorf("RootBucket: unexpected nil root bucket")
			_ = dbtx.Rollback()
			return false
		}

//删除密钥并确保它们已被删除。
		if !testDeleteValues(tc, rootBucket, values) {
			_ = dbtx.Rollback()
			return false
		}
		if !testGetValues(tc, rootBucket, rollbackValues(values)) {
			_ = dbtx.Rollback()
			return false
		}

//提交更改并确保成功。
		if err := dbtx.Commit(); err != nil {
			tc.t.Errorf("Commit: unexpected error %v", err)
			return false
		}

		return true
	}

//keyValues保存放置值时要使用的键和值
//变成一个桶。
	var keyValues = map[string]string{
		"umtxkey1": "foo1",
		"umtxkey2": "foo2",
		"umtxkey3": "foo3",
	}

//确保尝试使用只读填充值
//事务按预期失败。
	if !populateValues(false, true, keyValues) {
		return false
	}
	if !checkValues(rollbackValues(keyValues)) {
		return false
	}

//确保尝试使用读写填充值
//事务，然后将其回滚，得到预期值。
	if !populateValues(true, true, keyValues) {
		return false
	}
	if !checkValues(rollbackValues(keyValues)) {
		return false
	}

//确保尝试使用读写填充值
//事务，然后提交它来存储期望的值。
	if !populateValues(true, false, keyValues) {
		return false
	}
	if !checkValues(keyValues) {
		return false
	}

//把钥匙清理干净。
	if !deleteValues(keyValues) {
		return false
	}

	return true
}

//testnamespaceandxinterfaces使用提供的键和
//测试IT接口以及事务和bucket的所有方面
//它下面的接口。
func testNamespaceAndTxInterfaces(tc *testContext, namespaceKey string) bool {
	namespaceKeyBytes := []byte(namespaceKey)
	err := walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		_, err := tx.CreateTopLevelBucket(namespaceKeyBytes)
		return err
	})
	if err != nil {
		tc.t.Errorf("CreateTopLevelBucket: unexpected error: %v", err)
		return false
	}
	defer func() {
//现在已经完成了对名称空间的测试，请删除该名称空间。
		err := walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
			return tx.DeleteTopLevelBucket(namespaceKeyBytes)
		})
		if err != nil {
			tc.t.Errorf("DeleteTopLevelBucket: unexpected error: %v", err)
			return
		}
	}()

	if !testManualTxInterface(tc, namespaceKeyBytes) {
		return false
	}

//keyValues保存放置值时要使用的键和值
//变成一个桶。
	var keyValues = map[string]string{
		"mtxkey1": "foo1",
		"mtxkey2": "foo2",
		"mtxkey3": "foo3",
	}

//通过托管只读事务测试bucket接口。
	err = walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
		rootBucket := tx.ReadBucket(namespaceKeyBytes)
		if rootBucket == nil {
			return fmt.Errorf("ReadBucket: unexpected nil root bucket")
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("%v", err)
		}
		return false
	}

//通过托管读写事务测试bucket接口。
//另外，放置一系列值并强制回滚，因此
//代码可以确保没有存储值。
	forceRollbackError := fmt.Errorf("force rollback")
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		rootBucket := tx.ReadWriteBucket(namespaceKeyBytes)
		if rootBucket == nil {
			return fmt.Errorf("ReadWriteBucket: unexpected nil root bucket")
		}

		tc.isWritable = true
		if !testReadWriteBucketInterface(tc, rootBucket) {
			return errSubTestFail
		}

		if !testPutValues(tc, rootBucket, keyValues) {
			return errSubTestFail
		}

//返回一个错误以强制回滚。
		return forceRollbackError
	})
	if err != forceRollbackError {
		if err == errSubTestFail {
			return false
		}

		tc.t.Errorf("Update: inner function error not returned - got "+
			"%v, want %v", err, forceRollbackError)
		return false
	}

//确保由于强制
//上面的回滚实际上没有存储。
	err = walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
		rootBucket := tx.ReadBucket(namespaceKeyBytes)
		if rootBucket == nil {
			return fmt.Errorf("ReadBucket: unexpected nil root bucket")
		}

		if !testGetValues(tc, rootBucket, rollbackValues(keyValues)) {
			return errSubTestFail
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("%v", err)
		}
		return false
	}

//通过托管读写事务存储一系列值。
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		rootBucket := tx.ReadWriteBucket(namespaceKeyBytes)
		if rootBucket == nil {
			return fmt.Errorf("ReadWriteBucket: unexpected nil root bucket")
		}

		if !testPutValues(tc, rootBucket, keyValues) {
			return errSubTestFail
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("%v", err)
		}
		return false
	}

//确保按预期提交以上存储的值。
	err = walletdb.View(tc.db, func(tx walletdb.ReadTx) error {
		rootBucket := tx.ReadBucket(namespaceKeyBytes)
		if rootBucket == nil {
			return fmt.Errorf("ReadBucket: unexpected nil root bucket")
		}

		if !testGetValues(tc, rootBucket, keyValues) {
			return errSubTestFail
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("%v", err)
		}
		return false
	}

//清除托管读写事务中存储在上面的值。
	err = walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
		rootBucket := tx.ReadWriteBucket(namespaceKeyBytes)
		if rootBucket == nil {
			return fmt.Errorf("ReadWriteBucket: unexpected nil root bucket")
		}

		if !testDeleteValues(tc, rootBucket, keyValues) {
			return errSubTestFail
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("%v", err)
		}
		return false
	}

	return true
}

//TestAdditionalErrors对未覆盖的错误情况执行一些测试
//在测试的其他地方，因此提高了负测试覆盖率。
func testAdditionalErrors(tc *testContext) bool {
	ns3Key := []byte("ns3")

	err := walletdb.Update(tc.db, func(tx walletdb.ReadWriteTx) error {
//创建新的命名空间
		rootBucket, err := tx.CreateTopLevelBucket(ns3Key)
		if err != nil {
			return fmt.Errorf("CreateTopLevelBucket: unexpected error: %v", err)
		}

//确保CreateBucket在没有bucket时返回预期错误
//已指定密钥。
		wantErr := walletdb.ErrBucketNameRequired
		if _, err := rootBucket.CreateBucket(nil); err != wantErr {
			return fmt.Errorf("CreateBucket: unexpected error - "+
				"got %v, want %v", err, wantErr)
		}

//确保当没有存储桶时，DeleteNestedBucket返回预期错误
//已指定密钥。
		wantErr = walletdb.ErrIncompatibleValue
		if err := rootBucket.DeleteNestedBucket(nil); err != wantErr {
			return fmt.Errorf("DeleteNestedBucket: unexpected error - "+
				"got %v, want %v", err, wantErr)
		}

//确保Put在没有键时返回预期错误
//明确规定。
		wantErr = walletdb.ErrKeyRequired
		if err := rootBucket.Put(nil, nil); err != wantErr {
			return fmt.Errorf("Put: unexpected error - got %v, "+
				"want %v", err, wantErr)
		}

		return nil
	})
	if err != nil {
		if err != errSubTestFail {
			tc.t.Errorf("%v", err)
		}
		return false
	}

//确保尝试回滚或提交的事务
//已关闭返回预期的错误。
	tx, err := tc.db.BeginReadWriteTx()
	if err != nil {
		tc.t.Errorf("Begin: unexpected error: %v", err)
		return false
	}
	if err := tx.Rollback(); err != nil {
		tc.t.Errorf("Rollback: unexpected error: %v", err)
		return false
	}
	wantErr := walletdb.ErrTxClosed
	if err := tx.Rollback(); err != wantErr {
		tc.t.Errorf("Rollback: unexpected error - got %v, want %v", err,
			wantErr)
		return false
	}
	if err := tx.Commit(); err != wantErr {
		tc.t.Errorf("Commit: unexpected error - got %v, want %v", err,
			wantErr)
		return false
	}

	return true
}

//testinterface执行此数据库驱动程序的所有接口测试。
func TestInterface(t Tester, dbType, dbPath string) {
	db, err := walletdb.Create(dbType, dbPath)
	if err != nil {
		t.Errorf("Failed to create test database (%s) %v", dbType, err)
		return
	}
	defer os.Remove(dbPath)
	defer db.Close()

//对数据库运行所有接口测试。
//创建要传递的测试上下文。
	context := testContext{t: t, db: db}

//创建一个名称空间并测试它的接口。
	if !testNamespaceAndTxInterfaces(&context, "ns1") {
		return
	}

//创建第二个命名空间并测试其接口。
	if !testNamespaceAndTxInterfaces(&context, "ns2") {
		return
	}

//再检查一些其他地方没有涉及的错误情况。
	if !testAdditionalErrors(&context) {
		return
	}
}
