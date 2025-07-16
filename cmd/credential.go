package cmd

import (
	"fmt"

	"gotssh/internal/ui"

	"github.com/spf13/cobra"
)

// credentialCmd 凭证管理命令 (-o)
var credentialCmd = &cobra.Command{
	Use:   "credential",
	Short: "交互式管理登录凭证",
	Long: `进入交互式凭证管理界面，可以添加、编辑、删除登录凭证。

这个命令等同于使用 -o 参数。

支持的凭证类型：
- 密码凭证：存储用户名和密码
- SSH密钥凭证：存储用户名和SSH私钥

凭证可以设置别名，方便在添加服务器时快速选择。`,
	Aliases: []string{"o"},
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建凭证管理菜单
		credentialMenu := ui.NewCredentialMenu(configManager)
		
		fmt.Println("🔐 欢迎使用 GotSSH 凭证管理界面！")
		fmt.Println("使用方向键选择，按 Enter 确认，按 Ctrl+C 退出")
		fmt.Println()
		
		// 显示凭证管理菜单
		if err := credentialMenu.ShowCredentialMenu(); err != nil {
			return fmt.Errorf("显示凭证管理界面失败: %w", err)
		}
		
		fmt.Println("👋 再见！")
		return nil
	},
}

func init() {
	// 添加-o标志
	rootCmd.Flags().BoolP("credential", "o", false, "交互式管理登录凭证")
} 