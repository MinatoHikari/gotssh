package ui

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gotssh/internal/config"
)

// 测试辅助函数
func createTestCredentialMenu(t *testing.T) (*CredentialMenu, *config.Manager) {
	tempDir := t.TempDir()
	configPath := tempDir + "/test-config.yaml"

	configManager, err := config.NewManager(configPath)
	require.NoError(t, err)

	credentialMenu := NewCredentialMenu(configManager)

	return credentialMenu, configManager
}

func createTestCredentialForMenu(t *testing.T, configManager *config.Manager, alias string) *config.CredentialConfig {
	cred := config.NewCredentialConfig()
	cred.Alias = alias
	cred.Username = "testuser"
	cred.Type = config.CredentialTypePassword
	cred.Password = "testpass"
	cred.Description = "Test credential"

	err := configManager.AddCredential(cred)
	require.NoError(t, err)

	return cred
}

func createTestKeyCredential(t *testing.T, configManager *config.Manager, alias string) *config.CredentialConfig {
	cred := config.NewCredentialConfig()
	cred.Alias = alias
	cred.Username = "testuser"
	cred.Type = config.CredentialTypeKey
	cred.KeyPath = "/fake/path/to/key"
	cred.KeyPassphrase = "keypass"
	cred.Description = "Test key credential"

	err := configManager.AddCredential(cred)
	require.NoError(t, err)

	return cred
}

// TestNewCredentialMenu 测试凭证菜单创建
func TestNewCredentialMenu(t *testing.T) {
	t.Run("创建凭证菜单实例", func(t *testing.T) {
		credentialMenu, configManager := createTestCredentialMenu(t)

		assert.NotNil(t, credentialMenu)
		assert.Equal(t, configManager, credentialMenu.configManager)
	})

	t.Run("使用nil配置管理器", func(t *testing.T) {
		credentialMenu := NewCredentialMenu(nil)

		assert.NotNil(t, credentialMenu)
		assert.Nil(t, credentialMenu.configManager)
	})
}

// TestCredentialMenuStructure 测试凭证菜单结构
func TestCredentialMenuStructure(t *testing.T) {
	t.Run("菜单字段验证", func(t *testing.T) {
		credentialMenu, configManager := createTestCredentialMenu(t)

		// 验证菜单字段
		assert.NotNil(t, credentialMenu.configManager)

		// 验证字段类型
		assert.IsType(t, &config.Manager{}, credentialMenu.configManager)

		// 验证实例是否正确
		assert.Same(t, configManager, credentialMenu.configManager)
	})
}

// TestCredentialMenuShowCredentialList 测试显示凭证列表功能
func TestCredentialMenuShowCredentialList(t *testing.T) {
	t.Run("空凭证列表", func(t *testing.T) {
		credentialMenu, _ := createTestCredentialMenu(t)

		// 测试显示空列表
		assert.NotPanics(t, func() {
			credentialMenu.ShowCredentialList()
		})
	})

	t.Run("有凭证的列表", func(t *testing.T) {
		credentialMenu, configManager := createTestCredentialMenu(t)

		// 添加测试凭证
		createTestCredentialForMenu(t, configManager, "test-cred")

		// 测试显示有内容的列表
		assert.NotPanics(t, func() {
			credentialMenu.ShowCredentialList()
		})
	})

	t.Run("多个凭证的列表", func(t *testing.T) {
		credentialMenu, configManager := createTestCredentialMenu(t)

		// 添加多个测试凭证
		createTestCredentialForMenu(t, configManager, "test-cred-1")
		createTestCredentialForMenu(t, configManager, "test-cred-2")
		createTestKeyCredential(t, configManager, "test-key-cred")

		// 测试显示有内容的列表
		assert.NotPanics(t, func() {
			credentialMenu.ShowCredentialList()
		})
	})
}

// TestCredentialDataValidation 测试凭证数据验证
func TestCredentialDataValidation(t *testing.T) {
	t.Run("验证密码凭证", func(t *testing.T) {
		credentialMenu, configManager := createTestCredentialMenu(t)

		// 创建密码凭证
		cred := createTestCredentialForMenu(t, configManager, "test-pass-cred")

		// 验证凭证配置
		assert.Equal(t, "test-pass-cred", cred.Alias)
		assert.Equal(t, "testuser", cred.Username)
		assert.Equal(t, config.CredentialTypePassword, cred.Type)
		assert.Equal(t, "testpass", cred.Password)
		assert.Equal(t, "Test credential", cred.Description)

		// 验证凭证存在于配置中
		credentials := credentialMenu.configManager.ListCredentials()
		assert.Len(t, credentials, 1)
		assert.Equal(t, cred.ID, credentials[0].ID)
	})

	t.Run("验证密钥凭证", func(t *testing.T) {
		credentialMenu, configManager := createTestCredentialMenu(t)

		// 创建密钥凭证
		cred := createTestKeyCredential(t, configManager, "test-key-cred")

		// 验证凭证配置
		assert.Equal(t, "test-key-cred", cred.Alias)
		assert.Equal(t, "testuser", cred.Username)
		assert.Equal(t, config.CredentialTypeKey, cred.Type)
		assert.Equal(t, "/fake/path/to/key", cred.KeyPath)
		assert.Equal(t, "keypass", cred.KeyPassphrase)
		assert.Equal(t, "Test key credential", cred.Description)

		// 验证凭证存在于配置中
		credentials := credentialMenu.configManager.ListCredentials()
		assert.Len(t, credentials, 1)
		assert.Equal(t, cred.ID, credentials[0].ID)
	})
}

// TestCredentialMenuIntegration 测试凭证菜单集成
func TestCredentialMenuIntegration(t *testing.T) {
	t.Run("配置管理器集成", func(t *testing.T) {
		credentialMenu, configManager := createTestCredentialMenu(t)

		// 验证配置管理器功能
		assert.NotNil(t, credentialMenu.configManager)
		assert.Same(t, configManager, credentialMenu.configManager)

		// 测试配置管理器操作
		credentials := credentialMenu.configManager.ListCredentials()
		assert.NotNil(t, credentials)
		assert.Empty(t, credentials)
	})

	t.Run("凭证CRUD操作", func(t *testing.T) {
		credentialMenu, configManager := createTestCredentialMenu(t)

		// 添加凭证
		cred := createTestCredentialForMenu(t, configManager, "test-cred")

		// 验证凭证被添加
		credentials := credentialMenu.configManager.ListCredentials()
		assert.Len(t, credentials, 1)
		assert.Equal(t, cred.ID, credentials[0].ID)

		// 获取凭证
		retrievedCred, err := credentialMenu.configManager.GetCredential(cred.ID)
		assert.NoError(t, err)
		assert.Equal(t, cred.Alias, retrievedCred.Alias)

		// 删除凭证
		err = credentialMenu.configManager.DeleteCredential(cred.ID)
		assert.NoError(t, err)

		// 验证凭证被删除
		credentials = credentialMenu.configManager.ListCredentials()
		assert.Empty(t, credentials)
	})
}

// TestCredentialMenuStateManagement 测试凭证菜单状态管理
func TestCredentialMenuStateManagement(t *testing.T) {
	t.Run("凭证状态一致性", func(t *testing.T) {
		credentialMenu, configManager := createTestCredentialMenu(t)

		// 验证初始状态
		assert.Empty(t, credentialMenu.configManager.ListCredentials())

		// 添加凭证
		cred1 := createTestCredentialForMenu(t, configManager, "cred-1")
		cred2 := createTestKeyCredential(t, configManager, "cred-2")

		// 验证凭证存在
		credentials := credentialMenu.configManager.ListCredentials()
		assert.Len(t, credentials, 2)

		// 验证凭证内容
		credentialMap := make(map[string]*config.CredentialConfig)
		for _, cred := range credentials {
			credentialMap[cred.ID] = cred
		}

		assert.Contains(t, credentialMap, cred1.ID)
		assert.Contains(t, credentialMap, cred2.ID)
		assert.Equal(t, cred1.Alias, credentialMap[cred1.ID].Alias)
		assert.Equal(t, cred2.Alias, credentialMap[cred2.ID].Alias)
	})
}

// TestCredentialMenuErrorHandling 测试凭证菜单错误处理
func TestCredentialMenuErrorHandling(t *testing.T) {
	t.Run("nil配置管理器", func(t *testing.T) {
		credentialMenu := NewCredentialMenu(nil)

		assert.NotNil(t, credentialMenu)
		assert.Nil(t, credentialMenu.configManager)

		// 尝试显示列表应该不会panic（但可能会出错）
		assert.NotPanics(t, func() {
			credentialMenu.ShowCredentialList()
		})
	})

	t.Run("重复别名处理", func(t *testing.T) {
		credentialMenu, configManager := createTestCredentialMenu(t)

		// 添加第一个凭证
		cred1 := createTestCredentialForMenu(t, configManager, "duplicate-alias")

		// 尝试添加相同别名的凭证
		cred2 := config.NewCredentialConfig()
		cred2.Alias = "duplicate-alias"
		cred2.Username = "testuser2"
		cred2.Type = config.CredentialTypePassword
		cred2.Password = "testpass2"

		err := configManager.AddCredential(cred2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "别名")

		// 验证只有第一个凭证存在
		credentials := credentialMenu.configManager.ListCredentials()
		assert.Len(t, credentials, 1)
		assert.Equal(t, cred1.ID, credentials[0].ID)
	})
}

// TestCredentialMenuTypes 测试凭证类型处理
func TestCredentialMenuTypes(t *testing.T) {
	t.Run("密码凭证类型", func(t *testing.T) {
		_, configManager := createTestCredentialMenu(t)

		// 创建密码凭证
		cred := createTestCredentialForMenu(t, configManager, "password-cred")

		// 验证凭证类型
		assert.Equal(t, config.CredentialTypePassword, cred.Type)
		assert.NotEmpty(t, cred.Password)
		assert.Empty(t, cred.KeyPath)
		assert.Empty(t, cred.KeyPassphrase)
	})

	t.Run("密钥凭证类型", func(t *testing.T) {
		_, configManager := createTestCredentialMenu(t)

		// 创建密钥凭证
		cred := createTestKeyCredential(t, configManager, "key-cred")

		// 验证凭证类型
		assert.Equal(t, config.CredentialTypeKey, cred.Type)
		assert.Empty(t, cred.Password)
		assert.NotEmpty(t, cred.KeyPath)
		assert.NotEmpty(t, cred.KeyPassphrase)
	})
}

// TestCredentialMenuPersistence 测试凭证持久化
func TestCredentialMenuPersistence(t *testing.T) {
	t.Run("凭证保存和加载", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := tempDir + "/persistence-test.yaml"

		// 创建第一个管理器实例
		configManager1, err := config.NewManager(configPath)
		require.NoError(t, err)

		credentialMenu1 := NewCredentialMenu(configManager1)

		// 添加测试凭证
		cred1 := createTestCredentialForMenu(t, configManager1, "persist-cred-1")
		cred2 := createTestKeyCredential(t, configManager1, "persist-cred-2")

		// 验证凭证存在
		credentials := credentialMenu1.configManager.ListCredentials()
		assert.Len(t, credentials, 2)

		// 创建第二个管理器实例（加载相同配置）
		configManager2, err := config.NewManager(configPath)
		require.NoError(t, err)

		credentialMenu2 := NewCredentialMenu(configManager2)

		// 验证凭证被正确加载
		credentials = credentialMenu2.configManager.ListCredentials()
		assert.Len(t, credentials, 2)

		// 验证凭证内容
		credentialMap := make(map[string]*config.CredentialConfig)
		for _, cred := range credentials {
			credentialMap[cred.Alias] = cred
		}

		assert.Contains(t, credentialMap, cred1.Alias)
		assert.Contains(t, credentialMap, cred2.Alias)
		assert.Equal(t, cred1.Type, credentialMap[cred1.Alias].Type)
		assert.Equal(t, cred2.Type, credentialMap[cred2.Alias].Type)
	})
}

// TestCredentialMenuSearch 测试凭证搜索功能
func TestCredentialMenuSearch(t *testing.T) {
	t.Run("按别名搜索", func(t *testing.T) {
		credentialMenu, configManager := createTestCredentialMenu(t)

		// 添加测试凭证
		cred := createTestCredentialForMenu(t, configManager, "search-cred")

		// 按别名搜索
		foundCred, err := credentialMenu.configManager.GetCredentialByAlias("search-cred")
		assert.NoError(t, err)
		assert.Equal(t, cred.ID, foundCred.ID)
		assert.Equal(t, cred.Alias, foundCred.Alias)

		// 搜索不存在的别名
		_, err = credentialMenu.configManager.GetCredentialByAlias("non-existent")
		assert.Error(t, err)
	})
}

// BenchmarkCredentialMenuCreation 性能测试
func BenchmarkCredentialMenuCreation(b *testing.B) {
	tempDir := b.TempDir()
	configPath := tempDir + "/benchmark-config.yaml"

	configManager, err := config.NewManager(configPath)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		NewCredentialMenu(configManager)
	}
}

// BenchmarkShowCredentialList 性能测试
func BenchmarkShowCredentialList(b *testing.B) {
	tempDir := b.TempDir()
	configPath := tempDir + "/benchmark-config.yaml"

	configManager, err := config.NewManager(configPath)
	if err != nil {
		b.Fatal(err)
	}

	// 添加一些测试凭证
	for i := 0; i < 10; i++ {
		cred := config.NewCredentialConfig()
		cred.Alias = fmt.Sprintf("benchmark-cred-%d", i)
		cred.Username = "testuser"
		cred.Type = config.CredentialTypePassword
		cred.Password = "testpass"
		configManager.AddCredential(cred)
	}

	credentialMenu := NewCredentialMenu(configManager)

	for i := 0; i < b.N; i++ {
		credentialMenu.ShowCredentialList()
	}
}
