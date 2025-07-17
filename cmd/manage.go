package cmd

import (
	"fmt"

	"gotssh/internal/ui"

	"github.com/spf13/cobra"
)

// manageCmd 管理命令 (-m)
var manageCmd = &cobra.Command{
	Use:   "manage",
	Short: "进入交互式管理界面",
	Long: `进入交互式管理界面，可以管理服务器配置。

这个命令等同于使用 -m 参数。`,
	Aliases: []string{"m"},
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建交互式菜单
		menu := ui.NewMenu(configManager, forwardManager)

		fmt.Println("🚀 欢迎使用 GotSSH 服务器管理界面！")
		fmt.Println("使用方向键选择，按 Enter 确认，按 Ctrl+C 退出")
		fmt.Println()

		// 直接显示服务器管理菜单
		if err := menu.ShowServerMenu(); err != nil {
			return fmt.Errorf("显示服务器管理界面失败: %w", err)
		}

		fmt.Println("👋 再见！")
		return nil
	},
}

func init() {
	// 添加-m标志
	rootCmd.Flags().BoolP("manage", "m", false, "进入交互式管理界面")
}
