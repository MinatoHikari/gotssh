package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"gotssh/internal/config"
)

// 测试辅助函数
func createTestConfigManager(t *testing.T) *config.Manager {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	manager, err := config.NewManager(configPath)
	require.NoError(t, err)

	return manager
}

func createTestServerConfig(host string, port int) *config.ServerConfig {
	return &config.ServerConfig{
		ID:       "test-server-id",
		Host:     host,
		Port:     port,
		User:     "test-user",
		AuthType: config.AuthTypePassword,
		Password: "test-password",
	}
}

func createTestCredential(credType config.CredentialType) *config.CredentialConfig {
	cred := &config.CredentialConfig{
		ID:       "test-cred-id",
		Alias:    "test-cred",
		Username: "test-user",
		Type:     credType,
	}

	if credType == config.CredentialTypePassword {
		cred.Password = "test-password"
	} else if credType == config.CredentialTypeKey {
		cred.KeyPath = "/fake/key/path"
	}

	return cred
}

func generateTestSSHKey(t *testing.T) (string, string) {
	// 生成RSA私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// 编码私钥
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)

	// 生成公钥
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)

	return string(privateKeyBytes), string(publicKeyBytes)
}

func createTestKeyFile(t *testing.T, keyContent string) string {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")

	err := os.WriteFile(keyPath, []byte(keyContent), 0600)
	require.NoError(t, err)

	return keyPath
}

// TestNewClient 测试客户端创建
func TestNewClient(t *testing.T) {
	t.Run("创建客户端", func(t *testing.T) {
		manager := createTestConfigManager(t)
		serverConfig := createTestServerConfig("localhost", 22)

		client := NewClient(serverConfig, manager)

		assert.NotNil(t, client)
		assert.Equal(t, serverConfig, client.config)
		assert.Equal(t, manager, client.configManager)
		assert.Nil(t, client.credential) // 没有凭证ID
	})

	t.Run("创建带凭证的客户端", func(t *testing.T) {
		manager := createTestConfigManager(t)

		// 添加凭证
		cred := createTestCredential(config.CredentialTypePassword)
		err := manager.AddCredential(cred)
		require.NoError(t, err)

		// 创建引用凭证的服务器配置
		serverConfig := createTestServerConfig("localhost", 22)
		serverConfig.CredentialID = cred.ID

		client := NewClient(serverConfig, manager)

		assert.NotNil(t, client)
		assert.Equal(t, serverConfig, client.config)
		assert.Equal(t, manager, client.configManager)
		assert.NotNil(t, client.credential)
		assert.Equal(t, cred.ID, client.credential.ID)
	})

	t.Run("无效凭证ID", func(t *testing.T) {
		manager := createTestConfigManager(t)

		// 创建引用不存在凭证的服务器配置
		serverConfig := createTestServerConfig("localhost", 22)
		serverConfig.CredentialID = "non-existent-id"

		client := NewClient(serverConfig, manager)

		assert.NotNil(t, client)
		assert.Nil(t, client.credential) // 凭证不存在
	})
}

// TestBuildSSHConfig 测试SSH配置构建
func TestBuildSSHConfig(t *testing.T) {
	t.Run("密码认证", func(t *testing.T) {
		manager := createTestConfigManager(t)
		serverConfig := createTestServerConfig("localhost", 22)
		serverConfig.AuthType = config.AuthTypePassword
		serverConfig.Password = "test-password"

		client := NewClient(serverConfig, manager)

		sshConfig, err := client.buildSSHConfig()
		assert.NoError(t, err)
		assert.NotNil(t, sshConfig)
		assert.Equal(t, "test-user", sshConfig.User)
		assert.Len(t, sshConfig.Auth, 1)
		assert.Equal(t, 30*time.Second, sshConfig.Timeout)
	})

	t.Run("密码认证无密码", func(t *testing.T) {
		manager := createTestConfigManager(t)
		serverConfig := createTestServerConfig("localhost", 22)
		serverConfig.AuthType = config.AuthTypePassword
		serverConfig.Password = ""

		client := NewClient(serverConfig, manager)

		_, err := client.buildSSHConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "密码认证需要提供密码")
	})

	t.Run("凭证认证-密码", func(t *testing.T) {
		manager := createTestConfigManager(t)

		// 添加密码凭证
		cred := createTestCredential(config.CredentialTypePassword)
		err := manager.AddCredential(cred)
		require.NoError(t, err)

		serverConfig := createTestServerConfig("localhost", 22)
		serverConfig.AuthType = config.AuthTypeCredential
		serverConfig.CredentialID = cred.ID

		client := NewClient(serverConfig, manager)

		sshConfig, err := client.buildSSHConfig()
		assert.NoError(t, err)
		assert.NotNil(t, sshConfig)
		assert.Equal(t, "test-user", sshConfig.User)
		assert.Len(t, sshConfig.Auth, 1)
	})

	t.Run("凭证认证-无凭证", func(t *testing.T) {
		manager := createTestConfigManager(t)

		serverConfig := createTestServerConfig("localhost", 22)
		serverConfig.AuthType = config.AuthTypeCredential
		serverConfig.CredentialID = ""

		client := NewClient(serverConfig, manager)

		sshConfig, err := client.buildSSHConfig()
		// 应该有后备的认证方式（SSH代理或默认密钥）
		assert.NoError(t, err)
		assert.NotNil(t, sshConfig)
	})

	t.Run("密钥认证无密钥文件", func(t *testing.T) {
		manager := createTestConfigManager(t)
		serverConfig := createTestServerConfig("localhost", 22)
		serverConfig.AuthType = config.AuthTypeKey
		serverConfig.KeyPath = ""

		client := NewClient(serverConfig, manager)

		_, err := client.buildSSHConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "密钥认证需要提供密钥文件路径")
	})

	t.Run("用户名优先级", func(t *testing.T) {
		manager := createTestConfigManager(t)

		// 添加凭证，凭证有自己的用户名
		cred := createTestCredential(config.CredentialTypePassword)
		cred.Username = "cred-user"
		err := manager.AddCredential(cred)
		require.NoError(t, err)

		serverConfig := createTestServerConfig("localhost", 22)
		serverConfig.User = "server-user"
		serverConfig.AuthType = config.AuthTypeCredential
		serverConfig.CredentialID = cred.ID

		client := NewClient(serverConfig, manager)

		sshConfig, err := client.buildSSHConfig()
		assert.NoError(t, err)
		assert.NotNil(t, sshConfig)
		// 应该优先使用凭证中的用户名
		assert.Equal(t, "cred-user", sshConfig.User)
	})
}

// TestGetKeyAuth 测试密钥认证
func TestGetKeyAuth(t *testing.T) {
	t.Run("有效私钥文件", func(t *testing.T) {
		manager := createTestConfigManager(t)

		// 生成测试密钥
		privateKey, _ := generateTestSSHKey(t)
		keyPath := createTestKeyFile(t, privateKey)

		serverConfig := createTestServerConfig("localhost", 22)
		serverConfig.KeyPath = keyPath

		client := NewClient(serverConfig, manager)

		authMethod, err := client.getKeyAuth()
		assert.NoError(t, err)
		assert.NotNil(t, authMethod)
	})

	t.Run("无效私钥文件", func(t *testing.T) {
		manager := createTestConfigManager(t)

		// 创建无效密钥文件
		keyPath := createTestKeyFile(t, "invalid key content")

		serverConfig := createTestServerConfig("localhost", 22)
		serverConfig.KeyPath = keyPath

		client := NewClient(serverConfig, manager)

		_, err := client.getKeyAuth()
		assert.Error(t, err)
	})

	t.Run("不存在的密钥文件", func(t *testing.T) {
		manager := createTestConfigManager(t)

		serverConfig := createTestServerConfig("localhost", 22)
		serverConfig.KeyPath = "/nonexistent/key/path"

		client := NewClient(serverConfig, manager)

		_, err := client.getKeyAuth()
		assert.Error(t, err)
	})
}

// TestClose 测试客户端关闭
func TestClose(t *testing.T) {
	t.Run("关闭未连接的客户端", func(t *testing.T) {
		manager := createTestConfigManager(t)
		serverConfig := createTestServerConfig("localhost", 22)

		client := NewClient(serverConfig, manager)

		// 关闭未连接的客户端不应该panic
		assert.NotPanics(t, func() {
			client.Close()
		})
	})
}

// TestConnectViaProxy 测试代理连接配置
func TestConnectViaProxy(t *testing.T) {
	t.Run("SOCKS5代理配置", func(t *testing.T) {
		manager := createTestConfigManager(t)
		serverConfig := createTestServerConfig("localhost", 22)
		serverConfig.Proxy = &config.ProxyConfig{
			Type:     "socks5",
			Host:     "127.0.0.1",
			Port:     1080,
			Username: "proxy-user",
			Password: "proxy-pass",
		}

		client := NewClient(serverConfig, manager)

		// 由于没有实际的代理服务器，这个测试会失败
		// 但我们可以验证配置是否正确构建
		assert.NotNil(t, client.config.Proxy)
		assert.Equal(t, "socks5", client.config.Proxy.Type)
		assert.Equal(t, "127.0.0.1", client.config.Proxy.Host)
		assert.Equal(t, 1080, client.config.Proxy.Port)
	})
}

// TestProxyURLBuilding 测试代理URL构建
func TestProxyURLBuilding(t *testing.T) {
	t.Run("构建代理URL", func(t *testing.T) {
		manager := createTestConfigManager(t)
		serverConfig := createTestServerConfig("localhost", 22)
		serverConfig.Proxy = &config.ProxyConfig{
			Type:     "socks5",
			Host:     "127.0.0.1",
			Port:     1080,
			Username: "user",
			Password: "pass",
		}

		client := NewClient(serverConfig, manager)

		// 测试代理连接（会失败，但至少能验证配置）
		_, err := client.connectViaProxy("target:22")
		assert.Error(t, err) // 预期会失败，因为没有真实的代理服务器
	})
}

// TestSSHConfigValidation 测试SSH配置验证
func TestSSHConfigValidation(t *testing.T) {
	t.Run("验证超时设置", func(t *testing.T) {
		manager := createTestConfigManager(t)
		serverConfig := createTestServerConfig("localhost", 22)

		client := NewClient(serverConfig, manager)

		sshConfig, err := client.buildSSHConfig()
		assert.NoError(t, err)
		assert.Equal(t, 30*time.Second, sshConfig.Timeout)
	})

	t.Run("验证主机密钥回调", func(t *testing.T) {
		manager := createTestConfigManager(t)
		serverConfig := createTestServerConfig("localhost", 22)

		client := NewClient(serverConfig, manager)

		sshConfig, err := client.buildSSHConfig()
		assert.NoError(t, err)
		assert.NotNil(t, sshConfig.HostKeyCallback)
	})
}

// TestClientConfiguration 测试客户端配置
func TestClientConfiguration(t *testing.T) {
	t.Run("验证默认配置", func(t *testing.T) {
		manager := createTestConfigManager(t)
		serverConfig := createTestServerConfig("localhost", 22)

		client := NewClient(serverConfig, manager)

		assert.Equal(t, "localhost", client.config.Host)
		assert.Equal(t, 22, client.config.Port)
		assert.Equal(t, "test-user", client.config.User)
		assert.Equal(t, config.AuthTypePassword, client.config.AuthType)
	})

	t.Run("验证凭证覆盖", func(t *testing.T) {
		manager := createTestConfigManager(t)

		// 添加凭证
		cred := createTestCredential(config.CredentialTypePassword)
		cred.Username = "override-user"
		err := manager.AddCredential(cred)
		require.NoError(t, err)

		serverConfig := createTestServerConfig("localhost", 22)
		serverConfig.CredentialID = cred.ID

		client := NewClient(serverConfig, manager)

		assert.Equal(t, "override-user", client.credential.Username)
	})
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	t.Run("空认证方式", func(t *testing.T) {
		manager := createTestConfigManager(t)
		serverConfig := createTestServerConfig("localhost", 22)
		serverConfig.AuthType = ""

		client := NewClient(serverConfig, manager)

		_, err := client.buildSSHConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "未找到有效的认证方式")
	})

	t.Run("无效认证类型", func(t *testing.T) {
		manager := createTestConfigManager(t)
		serverConfig := createTestServerConfig("localhost", 22)
		serverConfig.AuthType = "invalid-auth-type"

		client := NewClient(serverConfig, manager)

		_, err := client.buildSSHConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "未找到有效的认证方式")
	})
}

// BenchmarkNewClient 性能测试
func BenchmarkNewClient(b *testing.B) {
	manager := createTestConfigManager(&testing.T{})
	serverConfig := createTestServerConfig("localhost", 22)

	for i := 0; i < b.N; i++ {
		NewClient(serverConfig, manager)
	}
}

// BenchmarkBuildSSHConfig 性能测试
func BenchmarkBuildSSHConfig(b *testing.B) {
	manager := createTestConfigManager(&testing.T{})
	serverConfig := createTestServerConfig("localhost", 22)
	client := NewClient(serverConfig, manager)

	for i := 0; i < b.N; i++ {
		client.buildSSHConfig()
	}
}
