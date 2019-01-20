
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2014-2016 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package waddrmgr

import (
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/btcsuite/btcwallet/internal/zero"
	"github.com/btcsuite/btcwallet/snacl"
	"github.com/btcsuite/btcwallet/walletdb"
)

const (
//
//之所以选择，是因为账户是硬化的儿童，因此不能
//
//支持导入范围顶部的预留账户
//地址。
MaxAccountNum = hdkeychain.HardenedKeyStart - 2 //2 ^ 31—2

//MaxAddressPeracCount是允许的最大地址数。
//
//
	MaxAddressesPerAccount = hdkeychain.HardenedKeyStart - 1

//
//
//根层次确定键和导入的地址
//
ImportedAddrAccount = MaxAccountNum + 1 //

//
	ImportedAddrAccountName = "imported"

//DefaultAccountNum是默认帐户的编号。
	DefaultAccountNum = 0

//
//默认帐户可以重命名，并且不是保留名称，
//因此，默认帐户可能不会被命名为“默认”和“非默认”
//帐户可以命名为“默认”。
//
//帐号永远不会改变，所以默认的accountNum应该是
//
	defaultAccountName = "default"

//bip0043描述的层次结构是：
 /*
 
 
 / /
 

 
 
 //基础层次确定性键的限制
 / /推导。
 

 
 
 
 外部分支uint32=0

 
 
 //分支。
 

 
 
 






 
}

//如果保留帐号，isreservedacountnum将返回true。

func isreservedacountnum（acct uint32）bool_
 返回帐户==importedAddrAccount




键入scryptoptions结构
 
}





 //obtainseed是一个回调函数，在
 
 //来自用户（或调用方认为合适的任何其他机制）。
 获取种子获取用户输入

 //obtainPrivatePass是一个可能被调用的回调函数
 
 //来自用户（或调用方的任何其他机制）的私有密码短语
 
 obtainprivatepass obtainuserinputfunc


//default scrypt options是用于scrypt的默认选项。
var defaultscryptoptions=scryptoptions_
 
 R：8，
 p：1，







//accountinfo存储内部和外部分支机构的当前状态
//一个帐户以及派生新密钥所需的扩展密钥。它
//还通过保持序列化的加密版本来处理锁定

//当地址管理器被锁定时。
键入accountinfo struct_
 

 
 
 //当地址管理器被锁定时。
 acctkeyencrypted[]字节
 acctkeypriv*hdkeychain.extended键
 

 
 
 
 

 
 
 
 
}


//派生和导入密钥的帐户名、编号和nubmer。

 
 帐户名字符串
 
 
 
}

//UnlockDeriveInfo包含派生私钥所需的信息

//管理器结构中的DeriveOnUnlock字段，以获取有关如何执行此操作的详细信息

类型unlockderiveinfo struct_
 
 
 索引UInt32





 passphrase*[]byte，config*scryptoptions）（*snacl.secretkey，错误）



 
 


var
 
 
 secretkeygen=默认newsecretkey

 
 
 
）




 
 oldkeygen：=secretkeygen
 secretkeygen=keygen
 

 
}

//new secret key使用活动secretkeygen生成新的密钥。
func newsecretkey（密码*[]字节，
 config*scryptoptions）（*snacl.secretkey，错误）

 
 
 返回secretkeygen（passphrase，config）



//我们的测试可以使用依赖注入来强制它们需要的行为。

 
 解密（以[]字节为单位）（[]字节，错误）
 
 
 零（）
}

//cryptokey扩展snacl.cryptokey来实现encryptorDecryptor。
类型CryptoKey结构
 加密密钥
}

//byte s返回此加密密钥的字节片的副本。

 返回ck.cryptokey[：]




 



func defaultNewCryptoKey（）（EncryptorDecryptor，错误）
 键，错误：=snacl.GenerateCryptoKey（）。
 如果犯错！= nIL{
  返回零
 }
 
}

//CryptoKeyType用于区分



//加密密钥类型。
康斯特
 
 //密钥材料，如派生的扩展私钥和导入的
 //私钥。
 cktprivate cryptokeytype=iota

 
 CKTScript

 //cktpublic指定用于公共加密的密钥
 //关键材料，如Dervied扩展公共密钥和导入公共密钥
 //键。
 
）


//用于使测试可以提供测试错误失败的版本的函数
//路径。
var newCryptoKey=默认newCryptoKey


//密钥存储。
类型管理器结构
 mtx同步.rwmutex

 
 
 ScopedManager映射[KeyScope]*ScopedKeyManager

 ExternalAddrschemas映射[地址类型][]键作用域
 InternalAddrschemas映射[地址类型][]键作用域

 
 
 
 
 闭式吊杆
 chainParams*链配置参数

 //masterkeypub是用于保护cryptokeypub密钥的密钥
 //而masterkeypriv是用于保护cryptokeypriv的密钥
 
 
 //提供未来的灵活性。
 / /
 //注意：这与扩展的bip0032主节点不同
 //KEY。
 / /
 
 //管理器已锁定。
 主密钥pub*snacl.secretkey
 主密钥priv*snacl.secretkey

 //cryptokeypub是用于加密公共扩展密钥和
 /地址。
 CryptoKeyPub加密机加密机加密机

 //cryptokeypriv是用于加密私有数据的密钥，如
 //主层次确定性扩展键。
 / /
 
 CryptoKeyPrivencrypted[]字节
 

 //cryptokeyscript是用于加密脚本数据的密钥。
 / /
 //当地址管理器被锁定时，此键将归零。
 
 CryptoKeyscript加密机加密机

 
 //在管理器解锁时检测到正确的密码短语，
 //管理器已解锁。散列在每个锁上归零。
 
 
}


/否则。
func（m*manager）watchonly（）bool_
 
 

 




/ /


 
  
  
   
    
   
   
  
 }

 
 
  
   
   
    锁定（）
   
    锁定（）
   
  
 

 
 
 m.cryptokeypriv.zero（）。
 

 
 零.bytea64（&m.hashedprivpassphrase）

 
 
 
 / /锁定。

 






 
 

 
  返回
 

 
  
  
 

 
 
  
 

 
 
 

 
 返回
}

//new scoped key manager从根管理器创建新的范围键管理器。一
//作用域密钥管理器是一个子管理器，它只有硬币类型的密钥
//特定的硬币类型和bip0043用途。这是有用的，因为它可以
//调用方创建具有独立模式的任意bip0043类似架构
/经理。请注意，如果钱包是
//只需注意，管理器尚未解锁，或根密钥已解锁。
//从数据库中和。
/ /
//TODO（roasbef）：原始密钥的addrType意味着它将在脚本中查找
//标记为Gucci？
func（m*manager）newscopedkeymanager（ns walletdb.readwritebucket，作用域keyscope，
 addrschema scopeaddrschema）（*scopedkeymanager，错误）

 
 

 
 
  
 

 
 
 
 / /
 //请注意，硬币类型的路径需要经过强化派生，
 //因此，只有当钱包的根密钥没有
 
 masterrootprivenc，u，err：=fetchmasterhdkeys（ns）
 如果犯错！= nIL{
  返回零
 }

 //如果在数据库中找不到主根私钥，但是
 //我们需要保释在这里，因为没有
 
 
  返回nil，managererror（errWatchingOnly，“”，nil）
 }

 
 
 
 如果犯错！= nIL{
  
  返回nil，managererror（errlocked，str，err）
 }

 //现在我们知道了根priv在数据库中，我们将解码
 
 rootpriv，错误：=hdkeychain.newkeyfromstring（
  
 ）
 
 如果犯错！= nIL{
  str：=fmt.sprintf（“无法创建主扩展私钥”）
  
 }

 //现在我们有了根私钥，我们将获取scope bucket
 //这样我们就可以创建正确的内部名称空间。
 scopebucket:=ns.nestedReadWriteBucket（scopebucketname）

 //现在我们知道可以实际创建一个新的范围
 //管理器，我们将在数据库中划分出它的bucket空间。
 如果错误：=CreateScopedManagers（ScopeBucket，&Scope）；错误！= nIL{
  返回零
 

 //创建数据库状态后，我们现在将写下地址
 
 
 如果scopeschemas==nil
  str：=“找不到范围架构存储桶”
  
 }
 scopekey：=scopetobytes（&scope）
 
 err=scopeschemas.put（scopekey[：]，schemabytes）
 如果犯错！= nIL{
  返回零
 }

 
 //使用主HD私钥，然后将其与
 
 
  
 ）
 如果犯错！= nIL{
  返回零
 }

 
 /经理。
 m.scopedmanager[scope]=和scopedkeymanager_
  
  
  根经理：
  地址：make（map[addrkey]managedAddress）
  
 }
 m.externalAddrschemas[addrschema.externalAddrType]=附加（
  m.externalAddrschemas[addrschema.externalAddrType]，作用域，
 ）
 
  
 ）

 返回m.scopedmanagers[scope]，无
}

//FetchScopedKeyManager尝试根据




 M. MTX
 

 sm，确定：=m.scopedmanagers[范围]
 如果！好吧{
  str：=fmt.sprintf（“找不到作用域%v”，作用域）
  返回nil，managererror（errscopenotfound，str，nil）
 }

 
}

//ActiveScopedKeyManagers返回所有活动范围键的切片
//根密钥管理器当前已知的管理器。
func（m*manager）activescopedkeymanagers（）[]*scopedkeymanager_
 M. MTX
 延迟m.mtx.runlock（）

 var ScopedManagers[]*ScopedKeyManager
 对于uuSMgr：=range m.scopedmanagers
  
 

 
}


//生成目标地址类型作为外部地址。
func（m*管理器）scopesforexternaladdrType（addrtype addressType）[]keyscope_
 M. MTX
 延迟m.mtx.runlock（）

 作用域，：=m.externalAddrschemas[addrType]
 返回范围
}


//生成目标地址类型作为内部地址。
func（m*manager）scopesforinternaladdrtypes（addrtype addresstype）[]keyscope_
 M. MTX
 延迟m.mtx.runlock（）

 作用域，：=m.InternalAddrschemas[AddrType]
 返回范围
}

//Neuterrootkey是一种特殊方法，一旦调用方

//将*从数据库中删除*加密的主HD根私钥。
func（m*manager）neuterrootkey（ns walletdb.readwritebucket）错误
 
 

 //首先，我们将从数据库中获取当前的主HD键。
 masterrootprivenc，u，err：=fetchmasterhdkeys（ns）
 如果犯错！= nIL{
  
 

 //如果根主私钥已经为零，则返回
 
 / /阉割。
 如果masterrootprivenc==nil
  返回零
 }
 

 //否则，我们将通过删除
 //数据库中加密的主HD密钥。
 
}

//address返回给定传递地址的托管地址（如果已知的话）
//地址管理器。托管地址与中传递的地址不同

//事务，如支付到PubKey的关联私钥和

//付费脚本哈希地址。
func（m*manager）地址（ns walletdb.readbucket，
 

 M. MTX
 延迟m.mtx.runlock（）

 //我们将遍历每个已知范围的管理器，并查看
 
 
  地址，错误：=scopedmgr.address（ns，address）
  如果犯错！= nIL{
   持续
  }

  
 }

 //如果任何作用域管理器都不知道地址，则
 
 str：=fmt.sprintf（“找不到地址%v”的键，地址）
 返回nil，managererror（errAddressNotFound，str，nil）
}


func（m*manager）markused（ns walletdb.readwritebucket，address btcutil.address）错误
 M. MTX
 延迟m.mtx.runlock（）

 
 //每个地址所用的地址。

 //首先，我们将找出这个地址属于哪个范围的管理器。
 
  如果uu，错误：=scopedmgr.address（ns，address）；错误！= nIL{
   持续
  }

  
  
  返回scopedmgr.markused（ns，address）
 

 //如果我们到了这一点，就无法在
 
 str：=fmt.sprintf（“找不到地址%v”的键，地址）
 返回管理器错误（errAddressNotFound、str、nil）
}

//addraccount返回给定地址所属的帐户。我们也
//返回拥有addr+帐户组合的作用域管理器。
func（m*管理器）addraccount（ns walletdb.readbucket，
 地址bcutil.address）（*ScopedKeyManager，uint32，错误）

 M. MTX
 延迟m.mtx.runlock（）

 对于uu，scopedmgr：=range m.scopedmanagers
  如果uu，错误：=scopedmgr.address（ns，address）；错误！= nIL{
   持续
  }

  
  //可以与管理器一起检索地址的帐户
  //地址所属的。
  
  如果犯错！= nIL{
   返回nil，0，err
  }

  
 }

 //如果我们到了这一点，就无法在
 
 str：=fmt.sprintf（“找不到地址%v”的键，地址）
 返回nil，0，managererror（errAddressNotFound，str，nil）
}

//foreachActiveAccountAddress调用给定函数，每个函数都处于活动状态
//存储在管理器中的给定帐户的地址，跨越所有活动
//作用域，出错时提前中断。
/ /


 帐户uint32，fn func（maddr managedAddress）错误

 M. MTX
 延迟m.mtx.runlock（）

 对于uu，scopedmgr：=range m.scopedmanagers
  错误：=scopedmgr.forEachActiveAccountAddress（ns，account，fn）
  如果犯错！= nIL{
   返回错误
  
 }

 返回零
}

//foreachActiveAddress使用每个活动地址调用给定函数
//存储在管理器中，出错时提前中断。
func（m*manager）foreachactiveaddress（ns walletdb.readbucket，fn func（addr btcutil.address）error）错误
 M. MTX
 延迟m.mtx.runlock（）

 对于uu，scopedmgr：=range m.scopedmanagers
  错误：=scopedmgr.foreachActiveAddress（ns，fn）
  如果犯错！= nIL{
   返回错误
  }
 }

 返回零
}

//foreachaccountaddress调用给定函数，每个地址为

func（m*manager）foreachaccountaddress（ns walletdb.readbucket，帐户uint32，
 fn func（maddr managedAddress）错误

 M. MTX
 延迟m.mtx.runlock（）

 对于uu，scopedmgr：=range m.scopedmanagers
  错误：=scopedmgr.foreachaccountaddress（ns，account，fn）
  如果犯错！= nIL{
   返回错误
  }
 }

 返回零
}

//chainParams返回此地址管理器的链参数。
func（m*manager）chainparams（）*chaincfg.params_
 //注意：这里不需要mutex，因为net字段没有更改
 //创建管理器实例后。

 
}

//changepassphrase将public或private passphrase更改为
//提供的值取决于私有标志。为了改变
//私有密码，地址管理器不能只监视。新的
//passphrase键是使用选项中的scrypt参数派生的，因此
//更改密码短语可能会增加计算难度
//需要强制使用密码短语。

 newpassphrase[]byte，private bool，config*scryptoptions）错误

 
 如果是private&&m.watchingOnly
  返回管理器错误（errWatchingOnly、errWatchingOnly、nil）
 }

 M.MTX.LoCK（）
 延迟m.mtx.unlock（）

 //确保提供的旧密码正确。这张支票办完了
 //使用相应的主密钥的副本取决于
 //确保当前状态不被更改的标志。临时密钥是
 //完成后清除，以避免在内存中保留副本。
 var keyname字符串
 secretkey：=snacl.secretkey key:&snacl.cryptokey
 如果私有{
  keyname=“私人”
  secretkey.parameters=m.masterkeypriv.parameters.
 }否则{
  keyname=“公共”
  secretkey.parameters=m.masterkeypub.parameters
 }
 如果错误：=secretkey.derivekey（&oldpassphrase）；错误！= nIL{
  if err==snacl.errInvalidPassword_
   str:=fmt.sprintf（“%s master的密码无效”+
    
   返回管理器错误（错误密码、str、nil）
  }

  str：=fmt.sprintf（“未能派生%s主密钥”，keyname）
  返回管理器错误（errCrypto、str、err）
 }
 延迟secretkey.zero（）

 //从用于安全的密码短语生成新的主密钥
 
 newmasterkey，错误：=newsecretkey（&newpassphrase，config）
 如果犯错！= nIL{
  str：=“无法创建新的主私钥”
  返回管理器错误（errCrypto、str、err）
 }
 newkeyparams:=newmasterkey.marshal（）。

 如果私有{
  //从技术上讲，这里只能检查锁定状态
  
  
  
  //速度快，而且简单解密的循环复杂性更小
  //无论哪种情况。

  //创建一个新的salt，用于散列新的
  
  var passphrasesalt[saltsize]字节
  错误：=rand.read（passphrasesalt[：]）
  如果犯错！= nIL{
   str：=“读取passhbrase salt的随机源失败”
   返回管理器错误（errCrypto、str、err）
  }

  //使用新的主密钥重新加密加密加密私钥
  //私钥。
  decpriv，错误：=secretkey.decrypt（m.cryptokeyprivEncrypted）
  如果犯错！= nIL{
   str：=“解密加密私钥失败”
   返回管理器错误（errCrypto、str、err）
  }
  encpriv，错误：=newmasterkey.encrypt（decpriv）
  零字节（decpriv）
  如果犯错！= nIL{
   str：=“加密加密私钥失败”
   返回管理器错误（errCrypto、str、err）
  }

  
  //私钥。
  decscript，错误：=secretkey.decrypt（m.cryptokeyscriptencrypted）
  如果犯错！= nIL{
   str：=“解密加密脚本密钥失败”
   返回管理器错误（errCrypto、str、err）
  }
  encscript，错误：=newmasterkey.encrypt（decscript）
  
  如果犯错！= nIL{
   str：=“加密加密脚本密钥失败”
   返回管理器错误（errCrypto、str、err）
  }

  //当管理器被锁定时，确保新的明文主控形状
  //由于不再需要密钥，该密钥已从内存中清除。
  //如果解锁，则使用新密码创建新密码哈希
  //密码和salt。
  var hashedpassphrase[sha512.size]字节
  如果M.锁定
   newmasterkey.zero（）。
  }否则{
   saltedpassphrase:=附加（passphrasesalt[：]，
    新密码…）
   hashedpassphrase=sha512.sum512（saltedpassphrase）
   零字节（saltedpassphrase）
  }

  //将新的键和参数单独保存到数据库中
  / /事务。
  err=输入密码键（ns、nil、encpriv、encscript）
  如果犯错！= nIL{
   返回maybeconvertdberror（err）
  }

  错误=putmasterkeyparams（ns，nil，newkeyparams）
  如果犯错！= nIL{
   返回maybeconvertdberror（err）
  }

  //现在数据库已成功更新，请清除旧数据库
  //键并设置新的。
  复制（m.cryptokeyprivencrypted[：]，encpriv）
  复制（m.cryptokeyscriptencrypted[：]，encscript）
  m.masterkeypriv.zero（）//清除旧密钥。
  m.masterkeypriv=新主密钥
  m.privPassphrasesalt=密码alt
  m.hashedprivpassphrase=hashedpassphrase
 }否则{
  //使用新的master public重新加密加密公钥
  //KEY。
  encryptedpub，错误：=newmasterkey.encrypt（m.cryptokeypb.bytes（））
  如果犯错！= nIL{
   str：=“加密加密公钥失败”
   返回管理器错误（errCrypto、str、err）
  }

  //将新的键和参数保存到数据库中
  / /事务。
  
  如果犯错！= nIL{
   返回maybeconvertdberror（err）
  }

  错误=putmasterkeyparams（ns，newkeyparams，nil）
  如果犯错！= nIL{
   返回maybeconvertdberror（err）
  }

  //现在数据库已成功更新，请清除旧数据库
  //键并设置新的。
  m.masterkeypub.zero（）。
  m.masterkeypub=新主密钥
 }

 返回零
}

//converttowatchingOnly将当前地址管理器转换为锁定的

/ /
//警告：此函数从现有地址管理器中删除私钥


//表示永久丢失任何导入的私钥和脚本。
/ /
//在只监视的管理器上执行此函数将
/没有效果。
func（m*manager）converttowatchingonly（ns walletdb.readwritebucket）错误
 M.MTX.LoCK（）
 延迟m.mtx.unlock（）

 //如果经理已经在监视，则立即退出。
 如果M.watchingonly
  返回零
 }

 无功误差

 //删除所有私钥材料并将新数据库标记为
 //仅观看。
 如果错误：=deleteprivatekeys（ns）；错误！= nIL{
  返回maybeconvertdberror（err）
 }

 错误=PutWatchingOnly（ns，true）
 如果犯错！= nIL{
  返回maybeconvertdberror（err）
 }

 
 //内存（如果需要）。
 如果！M.锁定
  锁（）
 }

 //此节清除并删除加密的私钥材料
 //通常用于解锁管理器。自从经理
 //正在转换为仅监视加密的私钥
 //不再需要材料。

 //清除并删除所有加密的帐户私钥。
 经理：=range m.scopedmanagers
  对于u，acctinfo：=range manager.acctinfo
   零字节（acctinfo.acctkeyEncrypted）
   acctinfo.acctkeyEncrypted=nil
  }
 }

 //清除并删除加密的私钥和加密的脚本
 
 经理：=range m.scopedmanagers
  对于uuma：=range manager.addrs
   开关地址：=ma.（类型）
   案例*管理者地址：
    零字节（addr.privkeyencrypted）
    addr.privkeyencrypted=nil
   案例*脚本地址：
    0.bytes（addr.scriptEncrypted）
    addr.scriptcencrypted=nil
   }
  }
 }

 //清除并删除加密的私钥和脚本加密密钥。
 0.bytes（m.cryptokeyscriptencrypted）
 m.cryptokeyscriptencrypted=nil
 m.cryptokeyscript=无
 零字节（m.CryptoKeyPrivencrypted）
 m.cryptokeyprivEncrypted=nil
 m.cryptokeypriv=零

 //当管理器
 //未锁定，因此没有加密版本为零。然而，
 //它不再需要了，所以不需要了。
 m.masterkeypriv=无

 //只标记经理正在监视。
 m.watchingOnly=真
 返回零

}


//未锁定，解密用于签名的私钥所需的解密密钥
//在内存中。
func（m*manager）islocked（）bool_
 M. MTX
 延迟m.mtx.runlock（）

 返回m.islocked（）。
}

//IsLocked是一个内部方法，返回地址管理器
//通过未受保护的读取被锁定。
/ /
//注意：调用方*必须*在调用之前获取管理器的互斥体
//避免数据争用。

 返回M锁定
}

//lock会尽最大努力删除所有关联的密钥并将其归零。
//使用地址管理器。
/ /
//如果在仅监视的地址上调用此函数，则此函数将返回错误
/经理。
func（m*manager）lock（）错误
 //不能锁定只监视的地址管理器。
 如果M.watchingonly
  返回管理器错误（errWatchingOnly、errWatchingOnly、nil）
 }

 M.MTX.LoCK（）
 延迟m.mtx.unlock（）

 //试图锁定已锁定的管理器时出错。
 如果M.锁定
  返回管理器错误（errlocked、errlocked、nil）
 }

 锁（）
 返回零
}

//unlock从指定的密码短语派生主私钥。安
//无效的密码将返回错误。否则，派生的密钥
//存储在内存中，直到地址管理器被锁定。任何故障

//即使在调用此函数之前它已经被解锁。
/ /
//如果在仅监视的地址上调用此函数，则此函数将返回错误
/经理。
func（m*manager）unlock（ns walletdb.readbucket，passphrase[]byte）错误
 
 如果M.watchingonly
  返回管理器错误（errWatchingOnly、errWatchingOnly、nil）
 }

 M.MTX.LoCK（）
 延迟m.mtx.unlock（）

 //如果管理器已解锁，则避免实际解锁
 //密码短语匹配。
 如果！M.锁定
  saltedpassphrase:=附加（m.privpassphrasesalt[：]，
   口令……
  hashedpassphrase:=sha512.sum512（saltedpassphrase）
  零字节（saltedpassphrase）
  如果是hashedpassphrase！=m.hashedprivpassphrase_
   锁（）
   
   返回管理器错误（错误密码、str、nil）
  }
  返回零
 }

 //使用提供的密码短语派生主私钥。
 如果错误：=m.masterkeypriv.derivekey（&passphrase）；错误！= nIL{
  锁（）
  if err==snacl.errInvalidPassword_
   str:=“主私钥的密码无效”
   返回管理器错误（错误密码、str、nil）
  }

  str：=“无法派生主私钥”
  返回管理器错误（errCrypto、str、err）
 }

 
 decryptedkey，错误：=m.masterkeypriv.decrypt（m.cryptokeyprivencrypted）
 如果犯错！= nIL{
  锁（）
  str：=“解密加密私钥失败”
  返回管理器错误（errCrypto、str、err）
 }
 m.cryptokeypriv.copybytes（解密密钥）
 零字节（解密密钥）

 //使用加密私钥解密所有帐户private
 //扩展键。
 经理：=range m.scopedmanagers
  
   已解密，错误：=m.CryptoKeyPriv.Decrypt（AcctInfo.AcctKeyEncrypted）
   如果犯错！= nIL{
    锁（）
    str:=fmt.sprintf（“解密帐户%d失败”+
     “私钥”，帐户）
    返回管理器错误（errCrypto、str、err）
   }

   acctkeypriv，错误：=hdkeychain.newkeypromstring（字符串（已解密））。
   0.字节（已解密）
   如果犯错！= nIL{
    锁（）
    str：=fmt.sprintf（“无法重新生成帐户%d”+
     “扩展密钥”，帐户）
    
   }
   acctinfo.acctkeypriv=acctkeypriv
  }

  //我们还将派生由于
  //它们是在地址管理器被锁定时创建的。
  对于u，信息：=Range Manager.DeriveOnUnlock
   AddressKey，错误：=Manager.DeriveKeyFromPath（）。
    ns，info.managedAddr.account（），info.branch，信息分支，
    信息索引，真的，
   ）
   如果犯错！= nIL{
    锁（）
    返回错误
   }

   //可以忽略这里的错误，因为它只能
   //如果扩展密钥不是私有的，则失败，但是
   //只是作为私钥派生的。
   privkey，：=addresskey.ecprivkey（）。
   地址键.zero（）

   privKeyBytes：=privKey.serialize（）。
   privkeyencrypted，错误：=m.cryptokeypriv.encrypt（privkeybytes）
   零.bigint（privkey.d）
   如果犯错！= nIL{
    锁（）
    str:=fmt.sprintf（“加密“+的私钥失败”
     “地址%s”，info.managedAddr.address（））
    返回管理器错误（errCrypto、str、err）
   }

   开关A：=info.managedaddr.（类型）
   案例*管理者地址：
    a.privkeyEncrypted=privkeyEncrypted
    a.privkeyct=privkebytes
   案例*脚本地址：
   }

   //避免在后续解锁时重新派生此密钥。
   manager.deriveOnUnlock[0]=无
   manager.deriveOnUnlock=manager.deriveOnUnlock[1:]
  }
 }

 m.locked=假
 saltedpassphrase:=append（m.privpassphrasesalt[：]，passphrase…）
 m.hashedprivpassphrase=sha512.sum512（saltedpassphrase）
 零字节（saltedpassphrase）
 返回零
}

//validateCountName验证给定的帐户名并返回错误（如果有）。
func validateCountName（name string）错误
 如果name＝“”{
  
  
 }
 如果是IsReservedAcountName（名称）
  
  退货经理错误（errInvalidAccount、str、nil）
 }
 返回零
}

//selectcryptokey根据密钥类型选择适当的加密密钥。安
//当指定的键类型或请求的键无效时返回错误
//要求管理器在未锁定时解锁。
/ /
//必须在保持管理器锁可读取的情况下调用此函数。
func（m*manager）selectcryptokey（keytype cryptokeytype）（encryptordecryptor，错误）
 
  //必须解锁管理器才能使用私钥。
  如果m.locked m.watchingOnly
   返回nil，managererror（errlocked，errlocked，nil）
  }
 }

 var密码密钥加密机加密机
 
 
  cryptokey=m.cryptokeypriv
 CKTScript案：
  cryptokey=m.cryptokeyscript
 CKTPublic案：
  cryptokey=m.cryptokeypub
 违约：
  返回nil，managererror（errInvalidKeyType，“无效密钥类型”，
   零）
 }

 返回密码键，零
}

//使用key type指定的加密密钥类型进行加密。
func（m*manager）encrypt（keytype cryptokeytype，in[]byte）（[]byte，error）
 //必须在管理器mutex下执行加密，因为
 //当管理器被锁定时，键被清除。
 M.MTX.LoCK（）
 延迟m.mtx.unlock（）

 cryptokey，错误：=m.selectcryptokey（keytype）
 如果犯错！= nIL{
  返回零
 }

 
 如果犯错！= nIL{
  返回nil，managererror（errCrypto，“加密失败”，err）
 }
 返回加密，无
}

//使用key type指定的加密密钥类型解密。
func（m*manager）解密（keytype cryptokeytype，in[]byte）（[]byte，error）
 //由于密钥
 //在管理器被锁定时被清除。
 M.MTX.LoCK（）
 延迟m.mtx.unlock（）

 cryptokey，错误：=m.selectcryptokey（keytype）
 如果犯错！= nIL{
  返回零
 }

 已解密，错误：=cryptokey.decrypt（in）
 如果犯错！= nIL{
  返回nil，managererror（errCrypto，“解密失败”，err）
 }
 返回已解密，无
}

//new manager返回一个具有给定参数的新锁定地址管理器。
func newmanager（chainparams*chaincfg.params，masterkeypub*snacl.secretkey，
 masterkeypriv*snacl.secretkey，cryptokeypub加密机或加密机，
 CryptoKeyprivEncrypted，CryptoKeyscriptEncrypted[]字节，SyncInfo*同步状态，
 生日时间。时间，privpassphrasesalt[saltsize]字节，
 ScopedManagers映射[keyscope]*ScopedKeyManager）*经理

 M: =＆经理{
  链参数：链参数，
  同步状态：*同步信息，
  锁定：正确，
  生日：生日，
  masterkeypub:masterkeypub，
  主钥匙私人：主钥匙私人，
  cryptokeypub：cryptokeypub，
  cryptokeyprivencrypted:加密的cryptokeyprivencrypted，
  cryptokeypriv:&cryptokey，
  cryptokeyscriptencrypted：cryptokeyscriptencrypted，
  cryptokeyscript:&cryptokey，
  
  ScopedManagers：ScopedManagers，
  ExternalAddrschemas:Make（映射[AddressType][]键作用域），
  InternalAddrschemas:make（映射[地址类型][]键作用域）
 }

 对于uuSMgr：=range m.scopedmanagers
  
  内部类型：=smgr.addrschema（）.InternalAddrType
  范围：=smgr.scope（））

  m.externalAddrschemas[externalType]=附加（
   M.ExternalAddrschemas[外部类型]，范围，
  ）
  m.InternalAddrschemas[InternalType]=附加（
   M.InternalAddrschemas[内部类型]，范围，
  ）
 }

 返回M
}

//DeriveCointypeKey派生可用于派生
// extended key for an account according to the hierarchy described by BIP0044
//给出了硬币类型的密钥。
/ /
//特别是这是层次确定的扩展密钥路径：
//m/用途“/<硬币类型>”
func-deriveCententypekey（masternode*hdkeychain.extendedkey，
 作用域keyscope）（*hdkeychain.extendedkey，错误）

 //强制使用最大硬币类型。
 如果scope.coin>maxcointype
  错误：=管理器错误（errcintypetoohigh，errcintypetoohigh，nil）
  返回零
 }

 //bip0043描述的层次结构是：
 //m/<目的>'/*
 / /
 //bip0044进一步扩展到：
 //m/44'/<coin type>'/<account>'/<branch>/<address index>
 / /
 //但是，因为这是任何bip0044系列的通用密钥存储
 //标准，我们将使用自定义作用域来控制我们的密钥派生。
 / /
 //分支的外部地址为0，内部地址为1。

 //将purpose键作为master节点的子级派生。
 用途，错误：=masternode.child（scope.purpose+hdkeychain.hardenedkeystart）
 如果犯错！= nIL{
  返回零
 }

 //将coin类型的键派生为purpose键的子级。
 cointypekey，err:=目的.child（scope.coin+hdkeychain.hardenedkeystart）
 如果犯错！= nIL{
  返回零
 }

 返回cointypekey，零
}

//DeriveAcountKey根据
//给定主节点的bip0044描述的层次结构。
/ /
//特别是这是层次确定的扩展密钥路径：

func derivateCountkey（cointypekey*hdkeychain.extendedkey，
 账户uint32）（*hdkeychain.extendedkey，错误）

 //强制使用最大帐号。
 
  错误：=managerError（errAccountNumTooHigh，erracttooHigh，nil）
  返回零
 }

 //将帐户密钥派生为硬币类型密钥的子项。
 返回cointypekey.child（account+hdkeychain.hardenedkeystart）
}

//checkBranchKeys确保派生内部和
//给定帐户密钥的外部分支不会导致无效的子级
//错误，这意味着所选种子不可用。这符合
//只要帐户密钥为
//已相应派生。
/ /
//特别是这是层次确定的扩展密钥路径：
//m/purpose'/<coin type>'/<account>'/<branch>
/ /
//分支的外部地址为0，内部地址为1。
func checkbranchkeys（acctkey*hdkeychain.extendedkey）错误
 //将外部分支派生为帐户密钥的第一个子级。
 如果uuErr：=acctkey.child（externalBranch）；Err！= nIL{
  返回错误
 }

 //将外部分支派生为帐户密钥的第二个子级。
 
 返回错误
}

//LoadManager返回一个新的地址管理器，该地址管理器是从
//传递的已打开数据库。解密需要公用密码短语
//公钥。

 chainparams*chaincfg.params）（*管理器，错误）

 //验证版本既不是太旧也不是太新。
 版本，错误：=fetchmanagerversion（ns）
 如果犯错！= nIL{
  str：=“无法获取更新版本”
  返回nil，managererror（errdatabase，str，err）
 }
 如果version<latestmgrversion
  
  返回nil，managererror（errUpgrade，str，nil）
 
  str：=“数据库版本大于最新的可理解版本”
  返回nil，managererror（errUpgrade，str，nil）
 }

 //加载管理器是否仅从数据库监视。
 watchingOnly，错误：=fetchwatchingOnly（ns）
 如果犯错！= nIL{
  返回nil，maybeconvertdberror（err）
 }

 //从数据库加载主密钥参数。
 masterkeypubparams，masterkeyprivparams，错误：=fetchmasterkeyparams（ns）
 如果犯错！= nIL{
  返回nil，maybeconvertdberror（err）
 }

 //从数据库加载加密密钥。
 cryptokeypubunc，cryptokeyprivenc，cryptokeyscriptenc，错误：=
  密码键（ns）
 如果犯错！= nIL{
  返回nil，maybeconvertdberror（err）
 }

 //从数据库加载同步状态。
 同步到，错误：=fetchsyncedto（ns）
 如果犯错！= nIL{
  返回nil，maybeconvertdberror（err）
 }
 startblock，错误：=fetchstartblock（ns）
 如果犯错！= nIL{
  返回nil，maybeconvertdberror（err）
 }
 生日，错误：=FetchBirthday（ns）
 如果犯错！= nIL{
  返回nil，maybeconvertdberror（err）
 }

 //当不是仅监视管理器时，设置主私钥参数，
 //但现在不要派生它，因为管理器开始锁定。
 var主密钥priv snacl.secretkey
 如果！只观察{
  
  如果犯错！= nIL{
   str：=“无法取消主私钥的标记”
   返回nil，managererror（errCrypto，str，err）
  }
 }

 
 //口令。
 var主密钥pub snacl.secretkey
 如果错误：=masterkeypub.unmashal（masterkeypubparams）；错误！= nIL{
  str：=“无法取消标记主公钥”
  返回nil，managererror（errCrypto，str，err）
 }
 如果错误：=masterkeypub.derivekey（&pubpassphrase）；错误！= nIL{
  str:=“主公钥的密码无效”
  返回nil，managererror（errwronpassphrase，str，nil）
 }

 //使用主公钥解密加密公钥。
 cryptokeypub：=&cryptokey snacl.cryptokey
 cryptokeypubct，错误：=masterkeypub.decrypt（cryptokeypubnc）
 如果犯错！= nIL{
  str：=“解密加密公钥失败”
  返回nil，managererror（errCrypto，str，err）
 }
 CryptoKeyPub.CopyBytes（CryptoKeyPubct）
 零字节（cryptokeypubct）

 //创建同步状态结构。
 同步信息：=newsyncstate（startblock，syncedto）

 //生成私有密码短语salt。
 var privpassphrasesalt[saltsize]字节
 _u，err=rand.read（privapassphrasesalt[：]）
 如果犯错！= nIL{
  str:=“读取密码短语salt的随机源失败”
  返回nil，managererror（errCrypto，str，err）
 }

 //接下来，我们需要从磁盘加载所有已知的管理器作用域。各
 //作用域位于我们的HD密钥链中一个独特的顶级路径上。
 ScopedManager:=Make（映射[KeyScope]*ScopedKeyManager）
 err=foreachkeyscope（ns，func（scope keyscope）错误
  scopeschema，err：=fetchscopeaddrschema（ns，&scope）
  如果犯错！= nIL{
   返回错误
  }

  scopedManagers[scope]=和scopedKeyManager_
   范围：范围，
   
   地址：make（map[addrkey]managedAddress）
   
  }

  返回零
 }）
 如果犯错！= nIL{
  返回零
 }

 
 //重写其他字段的默认值，这些字段不是
 //在调用new时使用从中加载的值指定
 //数据库。
 经理：=newmanager（
  链参数、MasterKeyPub和MasterKeyPriv，
  cryptokeypub、cryptokeyprivenc、cryptokeyscriptenc、同步信息，
  生日，私人口令，ScopedManagers，
 ）
 mgr.watchingonly=仅监视

 对于uu，scopedmanager：=范围scopedmanager
  scopedmanager.rootmanager=经理
 }

 返回MGR
}

//open从给定的命名空间加载现有的地址管理器。公众
//需要密码短语来解密用于保护公共密钥的公共密钥
//地址等信息。这很重要，因为访问bip0032
//扩展键意味着可以生成所有未来的地址。
/ /
//如果将配置结构传递给函数，则该配置将
//重写默认值。
/ /
//如果
//传递的管理器在指定的命名空间中不存在。
func open（ns walletdb.readbucket，pubppasshrase[]字节，
 chainparams*chaincfg.params）（*管理器，错误）

 //如果尚未在中创建管理器，则返回错误
 //给定的数据库命名空间。
 存在：=managerxists（ns）
 如果！存在{
  str：=“指定的地址管理器不存在”
  返回nil，managererror（errnoexist，str，nil）
 }

 返回loadmanager（ns、pubpassphrase、chainparams）
}

//createManagerKeyScope为目标管理器的作用域创建一个新的键作用域。

//同时维护多个地址派生方案。
func createManagerKeyScope（ns walletdb.readWriteBucket，
 作用域键作用域，根*hdkeychain.extendedkey，
 cryptokeypub，cryptokeypriv encryptordecryptor）错误

 //根据传递的作用域派生coinType键。
 cointypekeypriv，err:=派生类型键（根，作用域）
 如果犯错！= nIL{
  str：=“未能派生coinType扩展密钥”
  返回管理器错误（errkeychain、str、err）
 }
 推迟cointypekeypriv.zero（）。

 //根据我们的
 //类似bip0044的派生。
 acctkeypriv，err:=派生计数键（cointypekeypriv，0）
 如果犯错！= nIL{
  //如果
  //由于子级无效，无法派生所需的层次结构。
  if err==hdkeychain.errInvalidChild_
   str：=“提供的种子不可用”
   返回管理器错误（errkeychain，str，
    hd钥匙链。错误不可用）
  }

  返回错误
 }

 
 //到我们的bip0044类派生。
 如果错误：=checkbranchkeys（acctkeypriv）；错误！= nIL{
  //如果
  //由于子级无效，无法派生所需的层次结构。
  if err==hdkeychain.errInvalidChild_
   str：=“提供的种子不可用”
   返回管理器错误（errkeychain，str，
    hd钥匙链。错误不可用）
  }

  返回错误
 }

 //地址管理器需要该帐户的公共扩展密钥。
 acctkeypub，错误：=acctkeypriv.neuter（）
 如果犯错！= nIL{
  str：=“无法转换帐户0的私钥”
  返回管理器错误（errkeychain、str、err）
 }

 //使用关联的加密密钥加密cointype密钥。
 cointypekeypub，错误：=cointypekeypriv.neuter（）
 如果犯错！= nIL{
  str：=“无法转换coinType私钥”
  返回管理器错误（errkeychain、str、err）
 }
 cointypepupnc，错误：=cryptokeypub.encrypt（[]byte（cointypekeypub.string（））
 如果犯错！= nIL{
  str：=“加密CoinType公钥失败”
  返回管理器错误（errCrypto、str、err）
 }
 cointypeprivenc，错误：=cryptokeypriv.encrypt（[]byte（cointypekeypriv.string（））
 如果犯错！= nIL{
  str：=“加密CoinType私钥失败”
  返回管理器错误（errCrypto、str、err）
 }

 //使用关联的加密密钥加密默认帐户密钥。
 acctpubenc，错误：=cryptokeypub.encrypt（[]byte（acctkeypub.string（））
 如果犯错！= nIL{
  
  返回管理器错误（errCrypto、str、err）
 }
 
 如果犯错！= nIL{
  str：=“加密帐户0的私钥失败”
  返回管理器错误（errCrypto、str、err）
 }

 //将加密的cointype密钥保存到数据库。
 err=putcintypekeys（ns和scope，cointypepupnc，cointypeprivenc）
 如果犯错！= nIL{
  返回错误
 }

 //将默认帐户的信息保存到数据库中。
 
  ns和scope，默认accountnum，acctpubenc，acctprivenc，0，0，
  默认帐户名，
 ）
 如果犯错！= nIL{
  返回错误
 }

 返回putaccountinfo（
  ns和scope，importedAddAccount，nil，nil，0，0，
  导入的地址帐户名，
 ）
}

//create在给定的命名空间中创建新的地址管理器。种子必须
//符合hdkeychain.newmaster中描述的标准，将使用
//创建主根节点，所有层次结构都从该节点确定
//地址是派生的。这允许地址中的所有链接地址
//要使用相同种子恢复的管理器。
/ /
//所有私钥和公钥以及信息都受密钥保护
//从提供的私有和公共密码短语派生。公众
//在随后打开地址管理器时需要密码短语，并且
//需要使用专用密码来解锁地址管理器，以便
//访问任何私钥和信息。
/ /
//如果将配置结构传递给函数，则该配置将
//重写默认值。
/ /
//将返回错误代码为erralreadyexists的managerError
//指定的命名空间中已存在地址管理器。
func create（ns walletdb.readwritebucket，seed，pubpassphrase，privpassphrase[]byte，
 chainparams*chaincfg.params，config*scryptoptions，
 生日时间（time）错误

 //如果管理器已在中创建，则返回错误
 //给定的数据库命名空间。
 存在：=managerxists（ns）
 如果存在{
  返回管理器错误（erralreadyexists、erralreadyexists、nil）
 }

 //确保私有密码短语不为空。
 如果len（privpassphrase）==0
  str:=“私人密码不能为空”
  返回管理器错误（errEmptypassPhrase、str、nil）
 }

 //执行初始bucket创建和数据库命名空间设置。
 如果错误：=createManagerns（ns，scopeAddrMap）；错误！= nIL{
  返回maybeconvertdberror（err）
 }

 如果config==nil
  config=默认密码选项（&D）
 }

 //生成新的主密钥。这些主密钥用于保护
 //下一步将生成的加密密钥。
 masterkeypub，错误：=newsecretkey（&pubppasshrase，config）
 如果犯错！= nIL{
  str：=“无法掌握公钥”
  返回管理器错误（errCrypto、str、err）
 }
 masterkeypriv，err：=newsecretkey（&privpassphrase，config）
 如果犯错！= nIL{
  str：=“无法主控私钥”
  返回管理器错误（errCrypto、str、err）
 }
 推迟masterkeypriv.zero（）。

 //生成私有密码短语salt。这是在散列时使用的
 //当管理器
 //已解锁。
 var privpassphrasesalt[saltsize]字节
 _u，err=rand.read（privapassphrasesalt[：]）
 如果犯错！= nIL{
  str:=“读取密码短语salt的随机源失败”
  返回管理器错误（errCrypto、str、err）
 }

 //生成新的加密公钥、私钥和脚本密钥。这些钥匙是
 //用于保护实际的公共和私有数据，如地址，
 //扩展键和脚本。
 CryptoKeyPub，错误：=NewCryptoKey（）。
 如果犯错！= nIL{
  str：=“无法生成加密公钥”
  返回管理器错误（errCrypto、str、err）
 }
 CryptoKeyPriv，错误：=NewCryptoKey（）。
 如果犯错！= nIL{
  str：=“无法生成加密私钥”
  返回管理器错误（errCrypto、str、err）
 }
 推迟cryptokeypriv.zero（）。
 CryptoKeyScript，错误：=NewCryptoKey（）。
 如果犯错！= nIL{
  str：=“未能生成加密脚本密钥”
  返回管理器错误（errCrypto、str、err）
 }
 推迟cryptokeyscript.zero（）。

 //用关联的主密钥加密加密加密密钥。
 cryptokeypubunc，错误：=masterkeypub.encrypt（cryptokeypub.bytes（））
 如果犯错！= nIL{
  str：=“加密加密公钥失败”
  返回管理器错误（errCrypto、str、err）
 }
 cryptokeyprivenc，错误：=masterkeypriv.encrypt（cryptokeypriv.bytes（））
 如果犯错！= nIL{
  str：=“加密加密私钥失败”
  返回管理器错误（errCrypto、str、err）
 }
 cryptokeyscriptenc，错误：=masterkeypriv.encrypt（cryptokeyscript.bytes（））
 如果犯错！= nIL{
  str：=“加密加密脚本密钥失败”
  返回管理器错误（errCrypto、str、err）
 }

 //将传递链的Genesis块用作在块中创建的
 //默认值。
 createdat：=&blockstamp hash:*chainParams.genesHash，height:0

 //创建初始同步状态。
 同步信息：=newsyncstate（createdat，createdat）

 //将主密钥参数保存到数据库中。
 pubparams：=masterkeypub.marshal（）。
 privParams：=masterkeypriv.marshal（）。
 错误=putmasterkeyparams（ns、pubparams、privparams）
 如果犯错！= nIL{
  返回maybeconvertdberror（err）
 }

 //生成bip0044 hd密钥结构，以确保提供的种子
 //可以生成所需的结构，没有问题。

 //从种子派生主扩展键。
 rootkey，错误：=hdkeychain.newmaster（seed，chainparams）
 如果犯错！= nIL{
  str：=“无法派生主扩展密钥”
  返回管理器错误（errkeychain、str、err）
 }
 rootpubkey，错误：=rootkey.neuter（）。
 如果犯错！= nIL{
  str：=“无法使主扩展密钥中性化”
  返回管理器错误（errkeychain、str、err）
 }

 //接下来，对于每个寄存器的默认管理器作用域，我们将创建
 //它的硬CoinType键，以及第一个默认帐户。
 对于u，默认范围：=range defaultkeyscopes
  错误：=CreateManagerKeyScope（
   ns、defaultscope、rootkey、cryptokeypub、cryptokeypriv、
  ）
  如果犯错！= nIL{
   返回maybeconvertdberror（err）
  }
 }

 //在继续之前，我们还将存储根主私钥
 //以加密格式在数据库中。这是必需的，如
 //将来，我们可能需要创建额外的作用域密钥管理器。
 
 如果犯错！= nIL{
  返回maybeconvertdberror（err）
 }
 masterhdpubkeyenc，错误：=cryptokeypub.encrypt（[]byte（rootpubkey.string（）））
 如果犯错！= nIL{
  返回maybeconvertdberror（err）
 }
 错误=putmasterhdkeys（ns、masterhdprivkeenc、masterhdpubkeenc）
 如果犯错！= nIL{
  返回maybeconvertdberror（err）
 }

 //将加密的加密密钥保存到数据库。
 err=输入密码键（ns，cryptokeypubunc，cryptokeyprivenc，
  密码字）
 如果犯错！= nIL{
  返回maybeconvertdberror（err）
 }

 //将这不是只监视地址管理器的事实保存到
 //数据库。
 错误=putWatchingOnly（ns，false）
 如果犯错！= nIL{
  返回maybeconvertdberror（err）
 }

 //保存初始同步到状态。
 err=putsyncedto（ns，&syncinfo.syncedto）
 如果犯错！= nIL{
  返回maybeconvertdberror（err）
 }
 err=putstartblock（ns和syncinfo.startblock）
 如果犯错！= nIL{
  返回maybeconvertdberror（err）
 }

 //钱包生日时使用48小时作为安全时间。
 返回putBirthday（ns，birthday.add（-48*time.hour））。
}
