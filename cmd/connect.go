package cmd

import (
	"fmt"

	"gotssh/internal/config"
	"gotssh/internal/ssh"

	"github.com/spf13/cobra"
)

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
		
		// 查找服务器
		servers, err := configManager.FindServer(serverQuery)
		if err != nil {
			return fmt.Errorf("查找服务器失败: %w", err)
		}
		
		var server *config.ServerConfig
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

func init() {
	// 添加-a标志
	rootCmd.Flags().StringP("connect", "a", "", "连接到服务器 (IP或别名)")
} 