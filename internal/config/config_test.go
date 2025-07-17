package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试辅助函数
func createTempConfigFile(t *testing.T) string {
	tempDir := t.TempDir()
	return filepath.Join(tempDir, "test-config.yaml")
}

// TestNewConfig 测试配置创建
func TestNewConfig(t *testing.T) {
	t.Run("创建新配置", func(t *testing.T) {
		config := NewConfig()

		assert.NotNil(t, config)
		assert.Equal(t, 1, config.ConfigVersion)
		assert.NotNil(t, config.Servers)
		assert.NotNil(t, config.PortForwards)
		assert.NotNil(t, config.Credentials)
		assert.NotNil(t, config.Settings)

		// 验证默认设置
		assert.Equal(t, "info", config.Settings.LogLevel)
		assert.Equal(t, 30, config.Settings.ConnectTimeout)
		assert.Equal(t, "root", config.Settings.DefaultUser)
		assert.Equal(t, 22, config.Settings.DefaultPort)
		assert.Equal(t, "ask", config.Settings.DefaultAuthType)
	})
}

// TestNewServerConfig 测试服务器配置创建
func TestNewServerConfig(t *testing.T) {
	t.Run("创建新服务器配置", func(t *testing.T) {
		host := "192.168.1.100"
		server := NewServerConfig(host)

		assert.NotNil(t, server)
		assert.NotEmpty(t, server.ID)
		assert.Equal(t, host, server.Host)
		assert.Equal(t, 22, server.Port)
		assert.Equal(t, "root", server.User)
		assert.Equal(t, AuthTypeAsk, server.AuthType)
		assert.NotNil(t, server.Tags)
		assert.Empty(t, server.Tags)
		assert.WithinDuration(t, time.Now(), server.CreatedAt, time.Second)
		assert.WithinDuration(t, time.Now(), server.UpdatedAt, time.Second)
	})
}

// TestNewPortForwardConfig 测试端口转发配置创建
func TestNewPortForwardConfig(t *testing.T) {
	t.Run("创建新端口转发配置", func(t *testing.T) {
		serverID := "test-server-id"
		pf := NewPortForwardConfig(serverID)

		assert.NotNil(t, pf)
		assert.NotEmpty(t, pf.ID)
		assert.Equal(t, serverID, pf.ServerID)
		assert.Equal(t, ForwardTypeLocal, pf.Type)
		assert.Equal(t, "127.0.0.1", pf.LocalHost)
		assert.Equal(t, "127.0.0.1", pf.RemoteHost)
		assert.WithinDuration(t, time.Now(), pf.CreatedAt, time.Second)
		assert.WithinDuration(t, time.Now(), pf.UpdatedAt, time.Second)
	})
}

// TestNewCredentialConfig 测试凭证配置创建
func TestNewCredentialConfig(t *testing.T) {
	t.Run("创建新凭证配置", func(t *testing.T) {
		cred := NewCredentialConfig()

		assert.NotNil(t, cred)
		assert.NotEmpty(t, cred.ID)
		assert.Equal(t, CredentialTypePassword, cred.Type)
		assert.WithinDuration(t, time.Now(), cred.CreatedAt, time.Second)
		assert.WithinDuration(t, time.Now(), cred.UpdatedAt, time.Second)
	})
}

// TestManagerNew 测试配置管理器创建
func TestManagerNew(t *testing.T) {
	t.Run("创建配置管理器", func(t *testing.T) {
		configPath := createTempConfigFile(t)

		manager, err := NewManager(configPath)
		require.NoError(t, err)
		assert.NotNil(t, manager)

		// 验证配置文件被创建
		_, err = os.Stat(configPath)
		assert.NoError(t, err)
	})

	t.Run("使用默认配置路径", func(t *testing.T) {
		manager, err := NewManager("")
		require.NoError(t, err)
		assert.NotNil(t, manager)
	})
}

// TestManagerSaveAndLoad 测试配置保存和加载
func TestManagerSaveAndLoad(t *testing.T) {
	t.Run("保存和加载配置", func(t *testing.T) {
		configPath := createTempConfigFile(t)

		manager, err := NewManager(configPath)
		require.NoError(t, err)

		// 添加一个服务器
		server := NewServerConfig("test.example.com")
		server.Alias = "test-server"
		server.User = "admin"
		server.Port = 2222

		err = manager.AddServer(server)
		require.NoError(t, err)

		// 保存配置
		err = manager.Save()
		require.NoError(t, err)

		// 创建新的管理器实例并加载配置
		manager2, err := NewManager(configPath)
		require.NoError(t, err)

		// 验证配置被正确加载
		servers := manager2.ListServers()
		assert.Len(t, servers, 1)
		assert.Equal(t, "test-server", servers[0].Alias)
		assert.Equal(t, "admin", servers[0].User)
		assert.Equal(t, 2222, servers[0].Port)
	})
}

// TestManagerServerOperations 测试服务器操作
func TestManagerServerOperations(t *testing.T) {
	configPath := createTempConfigFile(t)
	manager, err := NewManager(configPath)
	require.NoError(t, err)

	t.Run("添加服务器", func(t *testing.T) {
		server := NewServerConfig("192.168.1.100")
		server.Alias = "test-server"

		err := manager.AddServer(server)
		assert.NoError(t, err)

		// 验证服务器被添加
		servers := manager.ListServers()
		assert.Len(t, servers, 1)
		assert.Equal(t, "test-server", servers[0].Alias)
	})

	t.Run("获取服务器", func(t *testing.T) {
		servers := manager.ListServers()
		require.Len(t, servers, 1)

		server, err := manager.GetServer(servers[0].ID)
		assert.NoError(t, err)
		assert.Equal(t, "test-server", server.Alias)
	})

	t.Run("查找服务器", func(t *testing.T) {
		// 按别名查找
		servers, err := manager.FindServer("test-server")
		assert.NoError(t, err)
		assert.Len(t, servers, 1)
		assert.Equal(t, "test-server", servers[0].Alias)

		// 按主机名查找
		servers, err = manager.FindServer("192.168.1.100")
		assert.NoError(t, err)
		assert.Len(t, servers, 1)
		assert.Equal(t, "192.168.1.100", servers[0].Host)
	})

	t.Run("更新服务器", func(t *testing.T) {
		servers := manager.ListServers()
		require.Len(t, servers, 1)

		server := servers[0]
		server.Description = "Updated description"

		err = manager.UpdateServer(server.ID, server)
		assert.NoError(t, err)

		// 验证更新
		updatedServer, err := manager.GetServer(server.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated description", updatedServer.Description)
	})

	t.Run("删除服务器", func(t *testing.T) {
		servers := manager.ListServers()
		require.Len(t, servers, 1)

		err = manager.DeleteServer(servers[0].ID)
		assert.NoError(t, err)

		// 验证删除
		servers = manager.ListServers()
		assert.Len(t, servers, 0)
	})
}

// TestManagerCredentialOperations 测试凭证操作
func TestManagerCredentialOperations(t *testing.T) {
	configPath := createTempConfigFile(t)
	manager, err := NewManager(configPath)
	require.NoError(t, err)

	t.Run("添加凭证", func(t *testing.T) {
		cred := NewCredentialConfig()
		cred.Alias = "test-cred"
		cred.Username = "admin"
		cred.Password = "secret"

		err := manager.AddCredential(cred)
		assert.NoError(t, err)

		// 验证凭证被添加
		creds := manager.ListCredentials()
		assert.Len(t, creds, 1)
		assert.Equal(t, "test-cred", creds[0].Alias)
	})

	t.Run("获取凭证", func(t *testing.T) {
		creds := manager.ListCredentials()
		require.Len(t, creds, 1)

		cred, err := manager.GetCredential(creds[0].ID)
		assert.NoError(t, err)
		assert.Equal(t, "test-cred", cred.Alias)
	})

	t.Run("根据别名获取凭证", func(t *testing.T) {
		cred, err := manager.GetCredentialByAlias("test-cred")
		assert.NoError(t, err)
		assert.Equal(t, "test-cred", cred.Alias)
		assert.Equal(t, "admin", cred.Username)
	})

	t.Run("更新凭证", func(t *testing.T) {
		creds := manager.ListCredentials()
		require.Len(t, creds, 1)

		cred := creds[0]
		cred.Description = "Updated credential"

		err = manager.UpdateCredential(cred.ID, cred)
		assert.NoError(t, err)

		// 验证更新
		updatedCred, err := manager.GetCredential(cred.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated credential", updatedCred.Description)
	})

	t.Run("删除凭证", func(t *testing.T) {
		creds := manager.ListCredentials()
		require.Len(t, creds, 1)

		err = manager.DeleteCredential(creds[0].ID)
		assert.NoError(t, err)

		// 验证删除
		creds = manager.ListCredentials()
		assert.Len(t, creds, 0)
	})
}

// TestManagerPortForwardOperations 测试端口转发操作
func TestManagerPortForwardOperations(t *testing.T) {
	configPath := createTempConfigFile(t)
	manager, err := NewManager(configPath)
	require.NoError(t, err)

	// 先添加一个服务器
	server := NewServerConfig("test.example.com")
	err = manager.AddServer(server)
	require.NoError(t, err)

	t.Run("添加端口转发", func(t *testing.T) {
		pf := NewPortForwardConfig(server.ID)
		pf.Alias = "test-forward"
		pf.LocalPort = 8080
		pf.RemotePort = 80

		err := manager.AddPortForward(pf)
		assert.NoError(t, err)

		// 验证端口转发被添加
		forwards := manager.ListPortForwards()
		assert.Len(t, forwards, 1)
		assert.Equal(t, "test-forward", forwards[0].Alias)
	})

	t.Run("获取端口转发", func(t *testing.T) {
		forwards := manager.ListPortForwards()
		require.Len(t, forwards, 1)

		pf, err := manager.GetPortForward(forwards[0].ID)
		assert.NoError(t, err)
		assert.Equal(t, "test-forward", pf.Alias)
	})

	t.Run("根据别名获取端口转发", func(t *testing.T) {
		pf, err := manager.GetPortForwardByAlias("test-forward")
		assert.NoError(t, err)
		assert.Equal(t, "test-forward", pf.Alias)
		assert.Equal(t, 8080, pf.LocalPort)
		assert.Equal(t, 80, pf.RemotePort)
	})

	t.Run("更新端口转发", func(t *testing.T) {
		forwards := manager.ListPortForwards()
		require.Len(t, forwards, 1)

		pf := forwards[0]
		pf.Description = "Updated port forward"

		err = manager.UpdatePortForward(pf.ID, pf)
		assert.NoError(t, err)

		// 验证更新
		updatedPf, err := manager.GetPortForward(pf.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated port forward", updatedPf.Description)
	})

	t.Run("删除端口转发", func(t *testing.T) {
		forwards := manager.ListPortForwards()
		require.Len(t, forwards, 1)

		err = manager.DeletePortForward(forwards[0].ID)
		assert.NoError(t, err)

		// 验证删除
		forwards = manager.ListPortForwards()
		assert.Len(t, forwards, 0)
	})
}

// TestAuthType 测试认证类型
func TestAuthType(t *testing.T) {
	t.Run("认证类型常量", func(t *testing.T) {
		assert.Equal(t, AuthType("password"), AuthTypePassword)
		assert.Equal(t, AuthType("key"), AuthTypeKey)
		assert.Equal(t, AuthType("credential"), AuthTypeCredential)
		assert.Equal(t, AuthType("ask"), AuthTypeAsk)
	})
}

// TestForwardType 测试端口转发类型
func TestForwardType(t *testing.T) {
	t.Run("端口转发类型常量", func(t *testing.T) {
		assert.Equal(t, ForwardType("local"), ForwardTypeLocal)
		assert.Equal(t, ForwardType("remote"), ForwardTypeRemote)
	})
}

// TestCredentialType 测试凭证类型
func TestCredentialType(t *testing.T) {
	t.Run("凭证类型常量", func(t *testing.T) {
		assert.Equal(t, CredentialType("password"), CredentialTypePassword)
		assert.Equal(t, CredentialType("key"), CredentialTypeKey)
	})
}

// TestGenerateID 测试ID生成
func TestGenerateID(t *testing.T) {
	t.Run("生成唯一ID", func(t *testing.T) {
		id1 := generateID()
		id2 := generateID()

		assert.NotEmpty(t, id1)
		assert.NotEmpty(t, id2)
		assert.NotEqual(t, id1, id2)

		// 验证ID格式 (时间戳-随机字符串)
		assert.Contains(t, id1, "-")
		assert.Contains(t, id2, "-")
	})
}

// TestRandomString 测试随机字符串生成
func TestRandomString(t *testing.T) {
	t.Run("生成随机字符串", func(t *testing.T) {
		str1 := randomString(6)
		str2 := randomString(6)

		assert.Len(t, str1, 6)
		assert.Len(t, str2, 6)
		assert.NotEqual(t, str1, str2)

		// 验证字符串只包含字母和数字
		for _, char := range str1 {
			assert.True(t, (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9'))
		}
	})
}

// TestConfigErrorHandling 测试错误处理
func TestConfigErrorHandling(t *testing.T) {
	configPath := createTempConfigFile(t)
	manager, err := NewManager(configPath)
	require.NoError(t, err)

	t.Run("获取不存在的服务器", func(t *testing.T) {
		_, err := manager.GetServer("non-existent-id")
		assert.Error(t, err)
	})

	t.Run("获取不存在的凭证", func(t *testing.T) {
		_, err := manager.GetCredential("non-existent-id")
		assert.Error(t, err)
	})

	t.Run("获取不存在的端口转发", func(t *testing.T) {
		_, err := manager.GetPortForward("non-existent-id")
		assert.Error(t, err)
	})

	t.Run("根据别名获取不存在的凭证", func(t *testing.T) {
		_, err := manager.GetCredentialByAlias("non-existent-alias")
		assert.Error(t, err)
	})

	t.Run("根据别名获取不存在的端口转发", func(t *testing.T) {
		_, err := manager.GetPortForwardByAlias("non-existent-alias")
		assert.Error(t, err)
	})

	t.Run("查找不存在的服务器", func(t *testing.T) {
		_, err := manager.FindServer("non-existent-server")
		assert.Error(t, err)
	})
}

// BenchmarkNewConfig 性能测试
func BenchmarkNewConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewConfig()
	}
}

// BenchmarkNewServerConfig 性能测试
func BenchmarkNewServerConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewServerConfig("test.example.com")
	}
}

// BenchmarkGenerateID 性能测试
func BenchmarkGenerateID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateID()
	}
}
