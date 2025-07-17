package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gotssh/internal/config"
	"gotssh/internal/ssh"

	"github.com/spf13/cobra"
)

// parseServerQuery 解析服务器查询字符串
// 支持格式：host, user@host, host:port, user@host:port
func parseServerQuery(query string) (user, host string, port int, err error) {
	// 默认值
	user = "root"
	port = 22

	// 分离用户名和主机部分
	var hostPart string
	if strings.Contains(query, "@") {
		parts := strings.SplitN(query, "@", 2)
		if len(parts) != 2 {
			err = fmt.Errorf("无效的用户名@主机格式")
			return
		}
		user = parts[0]
		hostPart = parts[1]
	} else {
		hostPart = query
	}

	// 分离主机和端口
	if strings.Contains(hostPart, ":") {
		parts := strings.SplitN(hostPart, ":", 2)
		if len(parts) != 2 {
			err = fmt.Errorf("无效的主机:端口格式")
			return
		}
		host = parts[0]
		if parts[1] != "" {
			port, err = strconv.Atoi(parts[1])
			if err != nil {
				err = fmt.Errorf("无效的端口号: %s", parts[1])
				return
			}
		}
	} else {
		host = hostPart
	}

	// 验证主机名不为空
	if host == "" {
		err = fmt.Errorf("主机名不能为空")
		return
	}

	return
}

// connectCmd 连接命令 (-a)
var connectCmd = &cobra.Command{
	Use:   "connect [server]",
	Short: "连接到服务器",
	Long: `根据IP地址或别名连接到已保存的服务器。

这个命令等同于使用 -a 参数。

参数：
  server    服务器IP地址或别名

示例：
  gotssh connect 192.168.1.100
  gotssh connect myserver
  gotssh -a 192.168.1.100
  gotssh -a myserver`,
	Aliases: []string{"a"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverQuery := args[0]

		// 先尝试查找保存的服务器
		servers, err := configManager.FindServer(serverQuery)

		var server *config.ServerConfig

		if err != nil {
			// 找不到保存的服务器，尝试解析为直接连接
			fmt.Printf("未找到保存的服务器 '%s'，尝试直接连接...\n", serverQuery)

			user, host, port, parseErr := parseServerQuery(serverQuery)
			if parseErr != nil {
				return fmt.Errorf("解析服务器地址失败: %w", parseErr)
			}

			// 创建临时服务器配置
			server = &config.ServerConfig{
				ID:       "temp-" + fmt.Sprintf("%d", time.Now().Unix()),
				Host:     host,
				Port:     port,
				User:     user,
				AuthType: config.AuthTypeAsk,
			}

			fmt.Printf("将使用交互式认证连接到 %s@%s:%d\n", user, host, port)
		} else {
			// 找到保存的服务器
			if len(servers) == 1 {
				server = servers[0]
			} else {
				// 多个匹配结果，让用户选择
				fmt.Printf("找到 %d 个匹配的服务器：\n", len(servers))
				for i, s := range servers {
					fmt.Printf("%d. ", i+1)
					if s.Alias != "" {
						fmt.Printf("[%s] ", s.Alias)
					}
					fmt.Printf("%s@%s:%d", s.User, s.Host, s.Port)
					if s.Description != "" {
						fmt.Printf(" - %s", s.Description)
					}
					fmt.Println()
				}

				var choice int
				fmt.Print("请选择服务器 (1-" + fmt.Sprintf("%d", len(servers)) + "): ")
				if _, err := fmt.Scanf("%d", &choice); err != nil {
					return fmt.Errorf("读取选择失败: %w", err)
				}

				if choice < 1 || choice > len(servers) {
					return fmt.Errorf("无效的选择")
				}

				server = servers[choice-1]
			}
		}

		// 显示连接信息
		fmt.Printf("正在连接到 %s@%s:%d", server.User, server.Host, server.Port)
		if server.Alias != "" {
			fmt.Printf(" [%s]", server.Alias)
		}
		fmt.Println()

		// 创建SSH客户端
		client := ssh.NewClient(server, configManager)

		// 连接到服务器
		if err := client.Connect(); err != nil {
			return fmt.Errorf("连接失败: %w", err)
		}
		defer client.Close()

		fmt.Println("✅ 连接成功！正在启动Shell...")

		// 启动交互式Shell
		if err := client.Shell(); err != nil {
			return fmt.Errorf("Shell启动失败: %w", err)
		}

		return nil
	},
}

// connectWithCredentialCmd 使用指定凭证连接到服务器
var connectWithCredentialCmd = &cobra.Command{
	Use:   "connect-with-credential [server] [credential]",
	Short: "使用指定凭证连接到服务器",
	Long: `使用指定的凭证连接到服务器。

这个命令用于处理同时使用 -a 和 -o 参数的情况。

参数：
  server      服务器IP地址或别名
  credential  凭证别名

示例：
  gotssh -a server1 -o mycred`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverQuery := args[0]
		credentialAlias := args[1]

		// 获取指定的凭证
		credential, err := configManager.GetCredentialByAlias(credentialAlias)
		if err != nil {
			return fmt.Errorf("获取凭证失败: %w", err)
		}

		// 先尝试查找保存的服务器
		servers, err := configManager.FindServer(serverQuery)

		var server *config.ServerConfig

		if err != nil {
			// 找不到保存的服务器，尝试解析为直接连接
			fmt.Printf("未找到保存的服务器 '%s'，尝试直接连接...\n", serverQuery)

			user, host, port, parseErr := parseServerQuery(serverQuery)
			if parseErr != nil {
				return fmt.Errorf("解析服务器地址失败: %w", parseErr)
			}

			// 创建临时服务器配置，使用凭证认证
			server = &config.ServerConfig{
				ID:           "temp-" + fmt.Sprintf("%d", time.Now().Unix()),
				Host:         host,
				Port:         port,
				User:         user,
				AuthType:     config.AuthTypeCredential,
				CredentialID: credential.ID,
			}

			// 如果凭证中有用户名，优先使用凭证中的用户名
			if credential.Username != "" {
				server.User = credential.Username
			}

			fmt.Printf("将使用凭证 '%s' 连接到 %s@%s:%d\n", credentialAlias, server.User, host, port)
		} else {
			// 找到保存的服务器
			if len(servers) == 1 {
				server = servers[0]
			} else {
				// 多个匹配结果，让用户选择
				fmt.Printf("找到 %d 个匹配的服务器：\n", len(servers))
				for i, s := range servers {
					fmt.Printf("%d. ", i+1)
					if s.Alias != "" {
						fmt.Printf("[%s] ", s.Alias)
					}
					fmt.Printf("%s@%s:%d", s.User, s.Host, s.Port)
					if s.Description != "" {
						fmt.Printf(" - %s", s.Description)
					}
					fmt.Println()
				}

				var choice int
				fmt.Print("请选择服务器 (1-" + fmt.Sprintf("%d", len(servers)) + "): ")
				if _, err := fmt.Scanf("%d", &choice); err != nil {
					return fmt.Errorf("读取选择失败: %w", err)
				}

				if choice < 1 || choice > len(servers) {
					return fmt.Errorf("无效的选择")
				}

				server = servers[choice-1]
			}

			// 修改服务器配置以使用指定的凭证
			server.AuthType = config.AuthTypeCredential
			server.CredentialID = credential.ID

			// 如果凭证中有用户名，优先使用凭证中的用户名
			if credential.Username != "" {
				server.User = credential.Username
			}

			fmt.Printf("将使用凭证 '%s' 连接到 %s@%s:%d", credentialAlias, server.User, server.Host, server.Port)
			if server.Alias != "" {
				fmt.Printf(" [%s]", server.Alias)
			}
			fmt.Println()
		}

		// 创建SSH客户端
		client := ssh.NewClient(server, configManager)

		// 连接到服务器
		if err := client.Connect(); err != nil {
			return fmt.Errorf("连接失败: %w", err)
		}
		defer client.Close()

		fmt.Println("✅ 连接成功！正在启动Shell...")

		// 启动交互式Shell
		if err := client.Shell(); err != nil {
			return fmt.Errorf("Shell启动失败: %w", err)
		}

		return nil
	},
}

func init() {
	// 添加-a标志
	rootCmd.Flags().StringP("connect", "a", "", "连接到服务器 (IP或别名)")
}
