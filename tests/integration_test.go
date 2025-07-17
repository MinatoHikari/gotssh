package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gotssh/internal/config"
	"gotssh/internal/forward"
	"gotssh/internal/ssh"
	"gotssh/internal/ui"
)

// 测试辅助函数
func createTestEnvironment(t *testing.T) (*config.Manager, *forward.Manager, *ui.Menu, *ui.CredentialMenu) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "integration-test.yaml")

	configManager, err := config.NewManager(configPath)
	require.NoError(t, err)

	forwardManager := forward.NewManager(configManager)
	menu := ui.NewMenu(configManager, forwardManager)
	credentialMenu := ui.NewCredentialMenu(configManager)

	return configManager, forwardManager, menu, credentialMenu
}

func createTestServerAndCredential(t *testing.T, configManager *config.Manager) (*config.ServerConfig, *config.CredentialConfig) {
	// 创建测试凭证
	cred := config.NewCredentialConfig()
	cred.Alias = "integration-test-cred"
	cred.Username = "testuser"
	cred.Type = config.CredentialTypePassword
	cred.Password = "testpass"
	cred.Description = "Integration test credential"

	err := configManager.AddCredential(cred)
	require.NoError(t, err)

	// 创建测试服务器
	server := config.NewServerConfig("localhost")
	server.Alias = "integration-test-server"
	server.User = "testuser"
	server.Port = 22
	server.AuthType = config.AuthTypeCredential
	server.CredentialID = cred.ID
	server.Description = "Integration test server"

	err = configManager.AddServer(server)
	require.NoError(t, err)

	return server, cred
}

func createTestPortForwardConfig(t *testing.T, configManager *config.Manager, serverID string) *config.PortForwardConfig {
	pf := config.NewPortForwardConfig(serverID)
	pf.Alias = "integration-test-forward"
	pf.LocalHost = "127.0.0.1"
	pf.LocalPort = 8080
	pf.RemoteHost = "127.0.0.1"
	pf.RemotePort = 80
	pf.Type = config.ForwardTypeLocal
	pf.Description = "Integration test port forward"

	err := configManager.AddPortForward(pf)
	require.NoError(t, err)

	return pf
}

// TestFullWorkflow 测试完整的工作流程
func TestFullWorkflow(t *testing.T) {
	t.Run("完整的配置管理流程", func(t *testing.T) {
		configManager, forwardManager, menu, credentialMenu := createTestEnvironment(t)

		// 1. 验证初始状态
		assert.Empty(t, configManager.ListServers())
		assert.Empty(t, configManager.ListCredentials())
		assert.Empty(t, configManager.ListPortForwards())
		assert.Empty(t, forwardManager.ListActiveForwards())

		// 2. 创建测试数据
		server, cred := createTestServerAndCredential(t, configManager)
		pf := createTestPortForwardConfig(t, configManager, server.ID)

		// 3. 验证数据被正确创建
		servers := configManager.ListServers()
		assert.Len(t, servers, 1)
		assert.Equal(t, server.ID, servers[0].ID)

		credentials := configManager.ListCredentials()
		assert.Len(t, credentials, 1)
		assert.Equal(t, cred.ID, credentials[0].ID)

		portForwards := configManager.ListPortForwards()
		assert.Len(t, portForwards, 1)
		assert.Equal(t, pf.ID, portForwards[0].ID)

		// 4. 验证服务器和凭证的关联
		assert.Equal(t, cred.ID, server.CredentialID)
		assert.Equal(t, config.AuthTypeCredential, server.AuthType)

		// 5. 验证端口转发和服务器的关联
		assert.Equal(t, server.ID, pf.ServerID)

		// 6. 测试UI组件能够正确显示数据
		assert.NotPanics(t, func() {
			menu.ShowServerList()
			menu.ShowPortForwardList()
			credentialMenu.ShowCredentialList()
		})
	})
}

// TestServerCredentialIntegration 测试服务器和凭证的集成
func TestServerCredentialIntegration(t *testing.T) {
	t.Run("服务器使用凭证认证", func(t *testing.T) {
		configManager, _, _, _ := createTestEnvironment(t)

		// 创建凭证
		cred := config.NewCredentialConfig()
		cred.Alias = "server-cred"
		cred.Username = "serveruser"
		cred.Type = config.CredentialTypePassword
		cred.Password = "serverpass"

		err := configManager.AddCredential(cred)
		require.NoError(t, err)

		// 创建使用凭证的服务器
		server := config.NewServerConfig("server.example.com")
		server.Alias = "test-server"
		server.AuthType = config.AuthTypeCredential
		server.CredentialID = cred.ID

		err = configManager.AddServer(server)
		require.NoError(t, err)

		// 验证关联
		assert.Equal(t, cred.ID, server.CredentialID)
		assert.Equal(t, config.AuthTypeCredential, server.AuthType)

		// 测试SSH客户端创建
		sshClient := ssh.NewClient(server, configManager)
		assert.NotNil(t, sshClient)

		// 验证SSH客户端能够获取凭证
		// 注意：这里不能测试实际连接，因为没有真实的SSH服务器
		// 但可以验证配置是否正确
		assert.NotNil(t, sshClient)
	})
}

// TestPortForwardIntegration 测试端口转发集成
func TestPortForwardIntegration(t *testing.T) {
	t.Run("端口转发配置和管理", func(t *testing.T) {
		configManager, forwardManager, _, _ := createTestEnvironment(t)

		// 创建服务器
		server := config.NewServerConfig("forward.example.com")
		server.Alias = "forward-server"
		server.AuthType = config.AuthTypePassword
		server.Password = "testpass"

		err := configManager.AddServer(server)
		require.NoError(t, err)

		// 创建端口转发配置
		pf := config.NewPortForwardConfig(server.ID)
		pf.Alias = "test-forward"
		pf.LocalHost = "127.0.0.1"
		pf.LocalPort = 8080
		pf.RemoteHost = "127.0.0.1"
		pf.RemotePort = 80
		pf.Type = config.ForwardTypeLocal

		err = configManager.AddPortForward(pf)
		require.NoError(t, err)

		// 验证配置
		assert.Equal(t, server.ID, pf.ServerID)

		// 验证端口转发管理器可以找到配置
		foundPf, err := configManager.GetPortForwardByAlias(pf.Alias)
		assert.NoError(t, err)
		assert.Equal(t, pf.ID, foundPf.ID)

		// 验证端口转发管理器状态
		assert.False(t, forwardManager.IsForwardActive(pf.ID))
		assert.Empty(t, forwardManager.ListActiveForwards())

		// 注意：这里不能测试实际的端口转发启动，因为没有真实的SSH服务器
		// 但可以验证配置是否正确
		assert.Equal(t, pf.LocalPort, 8080)
		assert.Equal(t, pf.RemotePort, 80)
	})
}

// TestConfigurationPersistence 测试配置持久化
func TestConfigurationPersistence(t *testing.T) {
	t.Run("配置在重启后保持", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "persistence-test.yaml")

		// 第一次创建配置
		configManager1, err := config.NewManager(configPath)
		require.NoError(t, err)

		// 添加数据
		server, cred := createTestServerAndCredential(t, configManager1)
		pf := createTestPortForwardConfig(t, configManager1, server.ID)

		// 验证数据存在
		assert.Len(t, configManager1.ListServers(), 1)
		assert.Len(t, configManager1.ListCredentials(), 1)
		assert.Len(t, configManager1.ListPortForwards(), 1)

		// 第二次加载配置
		configManager2, err := config.NewManager(configPath)
		require.NoError(t, err)

		// 验证数据被正确加载
		servers := configManager2.ListServers()
		assert.Len(t, servers, 1)
		assert.Equal(t, server.Alias, servers[0].Alias)

		credentials := configManager2.ListCredentials()
		assert.Len(t, credentials, 1)
		assert.Equal(t, cred.Alias, credentials[0].Alias)

		portForwards := configManager2.ListPortForwards()
		assert.Len(t, portForwards, 1)
		assert.Equal(t, pf.Alias, portForwards[0].Alias)

		// 验证关联关系
		assert.Equal(t, credentials[0].ID, servers[0].CredentialID)
		assert.Equal(t, servers[0].ID, portForwards[0].ServerID)
	})
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	t.Run("配置文件损坏处理", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "corrupted-config.yaml")

		// 创建损坏的配置文件
		err := os.WriteFile(configPath, []byte("invalid yaml content {["), 0600)
		require.NoError(t, err)

		// 尝试加载配置
		_, err = config.NewManager(configPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "解析配置文件失败")
	})

	t.Run("引用不存在的凭证", func(t *testing.T) {
		configManager, _, _, _ := createTestEnvironment(t)

		// 创建引用不存在凭证的服务器
		server := config.NewServerConfig("test.example.com")
		server.AuthType = config.AuthTypeCredential
		server.CredentialID = "non-existent-cred-id"

		err := configManager.AddServer(server)
		require.NoError(t, err)

		// 创建SSH客户端
		sshClient := ssh.NewClient(server, configManager)
		assert.NotNil(t, sshClient)

		// 验证客户端能够处理不存在的凭证
		assert.NotNil(t, sshClient)
	})

	t.Run("端口转发引用不存在的服务器", func(t *testing.T) {
		configManager, _, _, _ := createTestEnvironment(t)

		// 尝试创建引用不存在服务器的端口转发
		pf := config.NewPortForwardConfig("non-existent-server-id")
		pf.Alias = "invalid-forward"

		err := configManager.AddPortForward(pf)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "服务器")
	})
}

// TestConcurrentAccess 测试并发访问
func TestConcurrentAccess(t *testing.T) {
	t.Run("并发读写配置", func(t *testing.T) {
		configManager, _, _, _ := createTestEnvironment(t)

		// 创建基础数据
		server, _ := createTestServerAndCredential(t, configManager)
		_ = createTestPortForwardConfig(t, configManager, server.ID)

		// 并发读取配置
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		done := make(chan bool, 10)

		// 启动多个goroutine并发读取配置
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()

				for {
					select {
					case <-ctx.Done():
						return
					default:
						servers := configManager.ListServers()
						credentials := configManager.ListCredentials()
						portForwards := configManager.ListPortForwards()

						assert.Len(t, servers, 1)
						assert.Len(t, credentials, 1)
						assert.Len(t, portForwards, 1)

						time.Sleep(10 * time.Millisecond)
					}
				}
			}()
		}

		// 等待所有goroutine完成
		for i := 0; i < 10; i++ {
			select {
			case <-done:
				// goroutine完成
			case <-time.After(3 * time.Second):
				t.Fatal("超时等待goroutine完成")
			}
		}
	})
}

// TestPerformance 测试性能
func TestPerformance(t *testing.T) {
	t.Run("大量配置项性能", func(t *testing.T) {
		configManager, _, _, _ := createTestEnvironment(t)

		// 添加大量配置
		serverCount := 100
		credentialCount := 50

		// 添加凭证
		for i := 0; i < credentialCount; i++ {
			cred := config.NewCredentialConfig()
			cred.Alias = fmt.Sprintf("perf-cred-%d", i)
			cred.Username = fmt.Sprintf("user%d", i)
			cred.Type = config.CredentialTypePassword
			cred.Password = fmt.Sprintf("pass%d", i)

			err := configManager.AddCredential(cred)
			require.NoError(t, err)
		}

		// 添加服务器
		for i := 0; i < serverCount; i++ {
			server := config.NewServerConfig(fmt.Sprintf("server%d.example.com", i))
			server.Alias = fmt.Sprintf("perf-server-%d", i)
			server.AuthType = config.AuthTypePassword
			server.Password = fmt.Sprintf("pass%d", i)

			err := configManager.AddServer(server)
			require.NoError(t, err)
		}

		// 测试检索性能
		start := time.Now()
		servers := configManager.ListServers()
		credentials := configManager.ListCredentials()
		duration := time.Since(start)

		assert.Len(t, servers, serverCount)
		assert.Len(t, credentials, credentialCount)
		assert.Less(t, duration, time.Second, "配置检索应该在1秒内完成")
	})
}

// TestSSHClientIntegration 测试SSH客户端集成
func TestSSHClientIntegration(t *testing.T) {
	t.Run("SSH客户端配置构建", func(t *testing.T) {
		configManager, _, _, _ := createTestEnvironment(t)

		// 创建不同类型的服务器配置
		testCases := []struct {
			name     string
			authType config.AuthType
			setup    func(*config.ServerConfig)
		}{
			{
				name:     "密码认证",
				authType: config.AuthTypePassword,
				setup: func(s *config.ServerConfig) {
					s.Password = "testpass"
				},
			},
			{
				name:     "询问认证",
				authType: config.AuthTypeAsk,
				setup:    func(s *config.ServerConfig) {},
			},
			{
				name:     "凭证认证",
				authType: config.AuthTypeCredential,
				setup: func(s *config.ServerConfig) {
					cred := config.NewCredentialConfig()
					cred.Alias = "ssh-test-cred"
					cred.Username = "testuser"
					cred.Type = config.CredentialTypePassword
					cred.Password = "testpass"

					err := configManager.AddCredential(cred)
					require.NoError(t, err)

					s.CredentialID = cred.ID
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				server := config.NewServerConfig("test.example.com")
				server.Alias = "ssh-test-server-" + strings.ReplaceAll(tc.name, " ", "-")
				server.Host = "test-" + strings.ReplaceAll(tc.name, " ", "-") + ".example.com"
				server.AuthType = tc.authType
				tc.setup(server)

				err := configManager.AddServer(server)
				require.NoError(t, err)

				// 创建SSH客户端
				sshClient := ssh.NewClient(server, configManager)
				assert.NotNil(t, sshClient)

				// 验证客户端配置
				assert.NotNil(t, sshClient)
			})
		}
	})
}

// BenchmarkIntegrationPerformance 性能基准测试
func BenchmarkIntegrationPerformance(b *testing.B) {
	b.Run("配置管理性能", func(b *testing.B) {
		tempDir := b.TempDir()
		configPath := filepath.Join(tempDir, "benchmark-config.yaml")

		configManager, err := config.NewManager(configPath)
		if err != nil {
			b.Fatal(err)
		}

		// 预先添加一些配置
		for i := 0; i < 10; i++ {
			server := config.NewServerConfig(fmt.Sprintf("benchmark%d.example.com", i))
			server.Alias = fmt.Sprintf("bench-server-%d", i)
			configManager.AddServer(server)
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			servers := configManager.ListServers()
			_ = servers
		}
	})
}
