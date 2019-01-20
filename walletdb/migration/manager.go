
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package migration

import (
	"errors"
	"sort"

	"github.com/btcsuite/btcwallet/walletdb"
)

var (
//errReversion是尝试还原到
//检测到以前的版本。这样做是为了为用户提供安全
//因为有些升级可能不向后兼容。
	ErrReversion = errors.New("reverting to a previous version is not " +
		"supported")
)

//version表示数据库的版本号。可以使用迁移
//将数据库的前一个版本带到后一个版本。
type Version struct {
//数字表示此版本的编号。
	Number uint32

//迁移表示修改数据库的迁移函数
//状态。必须小心，以便随后的迁移建立在
//前一个是为了确保数据库的一致性。
	Migration func(walletdb.ReadWriteBucket) error
}

//管理器是一个接口，它公开必要的方法，以便
//迁移/升级服务。每个服务（即
//接口）然后可以使用升级功能执行任何所需的数据库
//迁移。
type Manager interface {
//name返回我们将尝试升级的服务的名称。
	Name() string

//命名空间返回服务的顶级存储桶。
	Namespace() walletdb.ReadWriteBucket

//current version返回服务数据库的当前版本。
	CurrentVersion(walletdb.ReadBucket) (uint32, error)

//setversion设置服务数据库的版本。
	SetVersion(walletdb.ReadWriteBucket, uint32) error

//版本返回的所有可用数据库版本
//服务。
	Versions() []Version
}

//GetLatestVersion返回给定切片中可用的最新版本。
func GetLatestVersion(versions []Version) uint32 {
	if len(versions) == 0 {
		return 0
	}

//在确定最新版本号之前，我们将对切片进行排序
//确保它反映了最后一个元素。
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Number < versions[j].Number
	})

	return versions[len(versions)-1].Number
}

//versionsToApply确定应将哪些版本应用于迁移
//基于当前版本。
func VersionsToApply(currentVersion uint32, versions []Version) []Version {
//假设迁移版本的顺序在增加，我们将应用
//任何版本号低于当前版本号的迁移。
	var upgradeVersions []Version
	for _, version := range versions {
		if version.Number > currentVersion {
			upgradeVersions = append(upgradeVersions, version)
		}
	}

//返回之前，我们将按其版本号对切片进行排序
//确保按预期顺序应用迁移。
	sort.Slice(upgradeVersions, func(i, j int) bool {
		return upgradeVersions[i].Number < upgradeVersions[j].Number
	})

	return upgradeVersions
}

//升级尝试升级通过管理器公开的一组服务
//接口。每个服务都将检查其可用版本并确定
//是否需要应用。
//
//注意：为了保证容错性，每次服务升级都应该
//在同一数据库事务中发生。
func Upgrade(mgrs ...Manager) error {
	for _, mgr := range mgrs {
		if err := upgrade(mgr); err != nil {
			return err
		}
	}

	return nil
}

//升级尝试升级通过其实现公开的服务
//管理器界面。此函数将确定是否有任何新版本
//需要根据服务的当前版本和最新版本应用
//可用的一个。
func upgrade(mgr Manager) error {
//我们将从获取服务的当前和最新版本开始。
	ns := mgr.Namespace()
	currentVersion, err := mgr.CurrentVersion(ns)
	if err != nil {
		return err
	}
	versions := mgr.Versions()
	latestVersion := GetLatestVersion(versions)

	switch {
//如果当前版本大于最新版本，则该服务
//正在尝试还原到可能
//向后不兼容。为了防止这种情况发生，我们将返回一个错误
//表示如此。
	case currentVersion > latestVersion:
		return ErrReversion

//如果当前版本落后于最新版本，我们需要
//应用所有较新版本以赶上最新版本。
	case currentVersion < latestVersion:
		versions := VersionsToApply(currentVersion, versions)
		mgrName := mgr.Name()
		ns := mgr.Namespace()

		for _, version := range versions {
			log.Infof("Applying %v migration #%d", mgrName,
				version.Number)

//如果有可用的迁移，我们将只运行迁移
//对于此版本。
			if version.Migration != nil {
				err := version.Migration(ns)
				if err != nil {
					log.Errorf("Unable to apply %v "+
						"migration #%d: %v", mgrName,
						version.Number, err)
					return err
				}
			}
		}

//应用所有版本后，我们现在可以反映
//服务的最新版本。
		if err := mgr.SetVersion(ns, latestVersion); err != nil {
			return err
		}

//如果当前版本与最新版本匹配，则没有升级
//需要，我们可以安全地离开。
	case currentVersion == latestVersion:
	}

	return nil
}
