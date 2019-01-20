
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package migration_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/btcsuite/btcwallet/walletdb"
	"github.com/btcsuite/btcwallet/walletdb/migration"
	"github.com/davecgh/go-spew/spew"
)

type mockMigrationManager struct {
	currentVersion uint32
	versions       []migration.Version
}

var _ migration.Manager = (*mockMigrationManager)(nil)

func (m *mockMigrationManager) Name() string {
	return "mock"
}

func (m *mockMigrationManager) Namespace() walletdb.ReadWriteBucket {
	return nil
}

func (m *mockMigrationManager) CurrentVersion(_ walletdb.ReadBucket) (uint32, error) {
	return m.currentVersion, nil
}

func (m *mockMigrationManager) SetVersion(_ walletdb.ReadWriteBucket, version uint32) error {
	m.currentVersion = version
	return nil
}

func (m *mockMigrationManager) Versions() []migration.Version {
	return m.versions
}

//testGetLatestVersion确保我们可以正确检索最新版本
//从一个版本切片。
func TestGetLatestVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		versions      []migration.Version
		latestVersion uint32
	}{
		{
			versions:      []migration.Version{},
			latestVersion: 0,
		},
		{
			versions: []migration.Version{
				{
					Number:    1,
					Migration: nil,
				},
			},
			latestVersion: 1,
		},
		{
			versions: []migration.Version{
				{
					Number:    1,
					Migration: nil,
				},
				{
					Number:    2,
					Migration: nil,
				},
			},
			latestVersion: 2,
		},
		{
			versions: []migration.Version{
				{
					Number:    2,
					Migration: nil,
				},
				{
					Number:    0,
					Migration: nil,
				},
				{
					Number:    1,
					Migration: nil,
				},
			},
			latestVersion: 2,
		},
	}

	for i, test := range tests {
		latestVersion := migration.GetLatestVersion(test.versions)
		if latestVersion != test.latestVersion {
			t.Fatalf("test %d: expected latest version %d, got %d",
				i, test.latestVersion, latestVersion)
		}
	}
}

//TestVersionsToApply确保需要应用的正确版本
//返回当前版本。
func TestVersionsToApply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		currentVersion  uint32
		versions        []migration.Version
		versionsToApply []migration.Version
	}{
		{
			currentVersion: 0,
			versions: []migration.Version{
				{
					Number:    0,
					Migration: nil,
				},
			},
			versionsToApply: nil,
		},
		{
			currentVersion: 1,
			versions: []migration.Version{
				{
					Number:    0,
					Migration: nil,
				},
			},
			versionsToApply: nil,
		},
		{
			currentVersion: 0,
			versions: []migration.Version{
				{
					Number:    0,
					Migration: nil,
				},
				{
					Number:    1,
					Migration: nil,
				},
				{
					Number:    2,
					Migration: nil,
				},
			},
			versionsToApply: []migration.Version{
				{
					Number:    1,
					Migration: nil,
				},
				{
					Number:    2,
					Migration: nil,
				},
			},
		},
		{
			currentVersion: 0,
			versions: []migration.Version{
				{
					Number:    2,
					Migration: nil,
				},
				{
					Number:    0,
					Migration: nil,
				},
				{
					Number:    1,
					Migration: nil,
				},
			},
			versionsToApply: []migration.Version{
				{
					Number:    1,
					Migration: nil,
				},
				{
					Number:    2,
					Migration: nil,
				},
			},
		},
	}

	for i, test := range tests {
		versionsToApply := migration.VersionsToApply(
			test.currentVersion, test.versions,
		)

		if !reflect.DeepEqual(versionsToApply, test.versionsToApply) {
			t.Fatalf("test %d: versions to apply mismatch\n"+
				"expected: %v\ngot: %v", i,
				spew.Sdump(test.versionsToApply),
				spew.Sdump(versionsToApply))
		}
	}
}

//testUpgradeRevert确保我们无法还原到上一个
//版本。
func TestUpgradeRevert(t *testing.T) {
	t.Parallel()

	m := &mockMigrationManager{
		currentVersion: 1,
		versions: []migration.Version{
			{
				Number:    0,
				Migration: nil,
			},
		},
	}

	if err := migration.Upgrade(m); err != migration.ErrReversion {
		t.Fatalf("expected Upgrade to fail with ErrReversion, got %v",
			err)
	}
}

//testupgradesameversion确保在当前版本
//匹配最新版本。
func TestUpgradeSameVersion(t *testing.T) {
	t.Parallel()

	m := &mockMigrationManager{
		currentVersion: 1,
		versions: []migration.Version{
			{
				Number:    0,
				Migration: nil,
			},
			{
				Number: 1,
				Migration: func(walletdb.ReadWriteBucket) error {
					return errors.New("migration should " +
						"not happen due to already " +
						"being on the latest version")
				},
			},
		},
	}

	if err := migration.Upgrade(m); err != nil {
		t.Fatalf("unable to upgrade: %v", err)
	}
}

//testupgradenewversion确保我们可以正确升级到较新的版本
//如果有的话。
func TestUpgradeNewVersion(t *testing.T) {
	t.Parallel()

	versions := []migration.Version{
		{
			Number:    0,
			Migration: nil,
		},
		{
			Number: 1,
			Migration: func(walletdb.ReadWriteBucket) error {
				return nil
			},
		},
	}

	m := &mockMigrationManager{
		currentVersion: 0,
		versions:       versions,
	}

	if err := migration.Upgrade(m); err != nil {
		t.Fatalf("unable to upgrade: %v", err)
	}

	latestVersion := migration.GetLatestVersion(versions)
	if m.currentVersion != latestVersion {
		t.Fatalf("expected current version to match latest: "+
			"current=%d vs latest=%d", m.currentVersion,
			latestVersion)
	}
}

//testupgradeMultipleverations确保我们可以进行多次升级
//以达到最新版本。
func TestUpgradeMultipleVersions(t *testing.T) {
	t.Parallel()

	previousVersion := uint32(0)
	versions := []migration.Version{
		{
			Number:    previousVersion,
			Migration: nil,
		},
		{
			Number: 1,
			Migration: func(walletdb.ReadWriteBucket) error {
				if previousVersion != 0 {
					return fmt.Errorf("expected previous "+
						"version to be %d, got %d", 0,
						previousVersion)
				}

				previousVersion = 1
				return nil
			},
		},
		{
			Number: 2,
			Migration: func(walletdb.ReadWriteBucket) error {
				if previousVersion != 1 {
					return fmt.Errorf("expected previous "+
						"version to be %d, got %d", 1,
						previousVersion)
				}

				previousVersion = 2
				return nil
			},
		},
	}

	m := &mockMigrationManager{
		currentVersion: 0,
		versions:       versions,
	}

	if err := migration.Upgrade(m); err != nil {
		t.Fatalf("unable to upgrade: %v", err)
	}

	latestVersion := migration.GetLatestVersion(versions)
	if m.currentVersion != latestVersion {
		t.Fatalf("expected current version to match latest: "+
			"current=%d vs latest=%d", m.currentVersion,
			latestVersion)
	}
}
