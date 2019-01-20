
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2015-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package wallet

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcwallet/internal/prompt"
	"github.com/btcsuite/btcwallet/waddrmgr"
	"github.com/btcsuite/btcwallet/walletdb"
)

const (
	walletDbName = "wallet.db"
)

var (
//ErrLoaded描述了尝试加载或
//当装载机已经这样做时，创建一个钱包。
	ErrLoaded = errors.New("wallet already loaded")

//errnotloaded描述了试图关闭
//当钱包还没有装上时，就装上了钱包。
	ErrNotLoaded = errors.New("wallet is not loaded")

//errexists描述尝试创建新的
//钱包已经存在。
	ErrExists = errors.New("wallet already exists")
)

//加载器实现了新钱包的创建和现有钱包的打开，而
//为其他子系统提供回调系统以处理
//钱包。这主要供RPC服务器使用，以启用
//
//另一个子系统。
//
//加载器对于并发访问是安全的。
type Loader struct {
	callbacks      []func(*Wallet)
	chainParams    *chaincfg.Params
	dbDirPath      string
	recoveryWindow uint32
	wallet         *Wallet
	db             walletdb.DB
	mu             sync.Mutex
}

//newloader构造具有可选恢复窗口的加载程序。如果
//恢复窗口非零，钱包将尝试恢复地址
//从上次同步到高度。
func NewLoader(chainParams *chaincfg.Params, dbDirPath string,
	recoveryWindow uint32) *Loader {

	return &Loader{
		chainParams:    chainParams,
		dbDirPath:      dbDirPath,
		recoveryWindow: recoveryWindow,
	}
}

//OnLoaded执行每个添加的回调，并阻止加载程序加载
//
func (l *Loader) onLoaded(w *Wallet, db walletdb.DB) {
	for _, fn := range l.callbacks {
		fn(w)
	}

	l.wallet = w
	l.db = db
l.callbacks = nil //
}

//runafterload添加了一个在加载程序创建或打开时要执行的函数
//钱包函数在单个goroutine中按它们的顺序执行
//补充。
func (l *Loader) RunAfterLoad(fn func(*Wallet)) {
	l.mu.Lock()
	if l.wallet != nil {
		w := l.wallet
		l.mu.Unlock()
		fn(w)
	} else {
		l.callbacks = append(l.callbacks, fn)
		l.mu.Unlock()
	}
}

//创建新钱包使用提供的公共和私人钱包创建新钱包
//口令。种子是可选的。如果非空，则地址来源于
//这个种子。如果为零，则生成安全的随机种子。
func (l *Loader) CreateNewWallet(pubPassphrase, privPassphrase, seed []byte,
	bday time.Time) (*Wallet, error) {

	defer l.mu.Unlock()
	l.mu.Lock()

	if l.wallet != nil {
		return nil, ErrLoaded
	}

	dbPath := filepath.Join(l.dbDirPath, walletDbName)
	exists, err := fileExists(dbPath)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrExists
	}

//创建由bolt db支持的钱包数据库。
	err = os.MkdirAll(l.dbDirPath, 0700)
	if err != nil {
		return nil, err
	}
	db, err := walletdb.Create("bdb", dbPath)
	if err != nil {
		return nil, err
	}

//在打开钱包之前初始化新创建的数据库。
	err = Create(
		db, pubPassphrase, privPassphrase, seed, l.chainParams, bday,
	)
	if err != nil {
		return nil, err
	}

//打开新创建的钱包。
	w, err := Open(db, pubPassphrase, nil, l.chainParams, l.recoveryWindow)
	if err != nil {
		return nil, err
	}
	w.Start()

	l.onLoaded(w, db)
	return w, nil
}

var errNoConsole = errors.New("db upgrade requires console access for additional input")

func noConsole() ([]byte, error) {
	return nil, errNoConsole
}

//打开现有钱包从加载程序的钱包数据库路径打开钱包
//以及公共密码。如果加载程序正由上下文调用，其中
//标准输入提示可在钱包升级时使用，设置
//CanconsolePrompt将启用这些提示。
func (l *Loader) OpenExistingWallet(pubPassphrase []byte, canConsolePrompt bool) (*Wallet, error) {
	defer l.mu.Unlock()
	l.mu.Lock()

	if l.wallet != nil {
		return nil, ErrLoaded
	}

//确保网络目录存在。
	if err := checkCreateDir(l.dbDirPath); err != nil {
		return nil, err
	}

//使用boltdb后端打开数据库。
	dbPath := filepath.Join(l.dbDirPath, walletDbName)
	db, err := walletdb.Open("bdb", dbPath)
	if err != nil {
		log.Errorf("Failed to open database: %v", err)
		return nil, err
	}

	var cbs *waddrmgr.OpenCallbacks
	if canConsolePrompt {
		cbs = &waddrmgr.OpenCallbacks{
			ObtainSeed:        prompt.ProvideSeed,
			ObtainPrivatePass: prompt.ProvidePrivPassphrase,
		}
	} else {
		cbs = &waddrmgr.OpenCallbacks{
			ObtainSeed:        noConsole,
			ObtainPrivatePass: noConsole,
		}
	}
	w, err := Open(db, pubPassphrase, cbs, l.chainParams, l.recoveryWindow)
	if err != nil {
//如果打开钱包失败（例如由于错误
//密码），我们必须关闭备份数据库
//允许以后调用walletdb.open（）。
		e := db.Close()
		if e != nil {
			log.Warnf("Error closing database: %v", e)
		}
		return nil, err
	}
	w.Start()

	l.onLoaded(w, db)
	return w, nil
}

//walletexists返回加载程序的数据库路径中是否存在文件。
//这可能会返回意外I/O故障的错误。
func (l *Loader) WalletExists() (bool, error) {
	dbPath := filepath.Join(l.dbDirPath, walletDbName)
	return fileExists(dbPath)
}

//loaded wallet返回已加载的钱包（如果有），并返回一个bool
//钱包是否已装好。如果是真的，钱包指针应该是安全的
//撤销引用。
func (l *Loader) LoadedWallet() (*Wallet, bool) {
	l.mu.Lock()
	w := l.wallet
	l.mu.Unlock()
	return w, w != nil
}

//UnloadWallet停止加载的钱包（如果有），并关闭钱包数据库。
//如果钱包未加载，则返回errnotloaded
//创建新钱包或加载现有钱包。如果这样，装载机可以重新使用
//函数返回时不出错。
func (l *Loader) UnloadWallet() error {
	defer l.mu.Unlock()
	l.mu.Lock()

	if l.wallet == nil {
		return ErrNotLoaded
	}

	l.wallet.Stop()
	l.wallet.WaitForShutdown()
	err := l.db.Close()
	if err != nil {
		return err
	}

	l.wallet = nil
	l.db = nil
	return nil
}

func fileExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
