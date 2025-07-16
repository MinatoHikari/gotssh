package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// tunnelConnectCmd 快速端口转发连接命令 (-at)
var tunnelConnectCmd = &cobra.Command{
	Use:   "tunnel-connect [alias]",
	Short: "根据别名快速启动端口转发",
	Long: `根据别名快速启动已配置的端口转发隧道。

这个命令等同于使用 -at 参数。

参数：
  alias    端口转发配置的别名

示例：
  gotssh tunnel-connect mysql-tunnel
  gotssh -at mysql-tunnel

端口转发将在前台运行，使用 Ctrl+C 停止。`,
	Aliases: []string{"at"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]
		
		// 根据别名获取端口转发配置
		pf, err := configManager.GetPortForwardByAlias(alias)
		if err != nil {
			return fmt.Errorf("获取端口转发配置失败: %w", err)
		}
		
		// 获取服务器信息
		server, err := configManager.GetServer(pf.ServerID)
		if err != nil {
			return fmt.Errorf("获取服务器配置失败: %w", err)
		}
		
		// 显示端口转发信息
		fmt.Printf("🚀 正在启动端口转发: [%s]\n", alias)
		fmt.Printf("服务器: %s@%s:%d", server.User, server.Host, server.Port)
		if server.Alias != "" {
			fmt.Printf(" [%s]", server.Alias)
		}
		fmt.Println()
		
		fmt.Printf("转发配置: %s:%d -> %s:%d (%s)\n", 
			pf.LocalHost, pf.LocalPort, pf.RemoteHost, pf.RemotePort, pf.Type)
		
		if pf.Description != "" {
			fmt.Printf("描述: %s\n", pf.Description)
		}
		
		fmt.Println("使用 Ctrl+C 停止端口转发")
		fmt.Println("----------------------------------------")
		
		// 启动端口转发并处理信号
		if err := forwardManager.StartPortForwardByAliasWithSignalHandler(alias); err != nil {
			return fmt.Errorf("启动端口转发失败: %w", err)
		}
		
		return nil
	},
}

func init() {
	// 添加--at标志（不能使用短标志，因为at是两个字符）
	rootCmd.Flags().String("at", "", "根据别名快速启动端口转发")
} 