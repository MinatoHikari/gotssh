package cmd

import (
	"fmt"

	"gotssh/internal/ui"

	"github.com/spf13/cobra"
)

// tunnelCmd 端口转发管理命令 (-t)
var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "交互式管理SSH端口转发",
	Long: `进入交互式端口转发管理界面，可以配置和管理SSH端口转发。

这个命令等同于使用 -t 参数。

支持的功能：
- 添加端口转发配置
- 启动/停止端口转发
- 查看端口转发状态
- 删除端口转发配置
- 测试端口转发连接

支持本地端口转发和远程端口转发。`,
	Aliases: []string{"t"},
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建交互式菜单
		menu := ui.NewMenu(configManager, forwardManager)

		fmt.Println("🌐 欢迎使用 GotSSH 端口转发管理界面！")
		fmt.Println("使用方向键选择，按 Enter 确认，按 Ctrl+C 退出")
		fmt.Println()

		// 直接显示端口转发菜单
		if err := menu.ShowPortForwardMenu(); err != nil {
			return fmt.Errorf("显示端口转发管理界面失败: %w", err)
		}

		fmt.Println("👋 再见！")
		return nil
	},
}

func init() {
	// 添加-t标志
	rootCmd.Flags().BoolP("tunnel", "t", false, "交互式管理SSH端口转发")
}
