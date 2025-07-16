package cmd

import (
	"fmt"
	"os"

	"gotssh/internal/config"
	"gotssh/internal/forward"

	"github.com/spf13/cobra"
)

var (
	configManager  *config.Manager
	forwardManager *forward.Manager
)

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "gotssh",
	Short: "一个功能强大的SSH连接和端口转发管理工具",
	Long: `GotSSH是一个功能强大的SSH连接和端口转发管理工具。

支持的功能：
- 交互式服务器管理
- 多种认证方式（密码、密钥、登录凭证、每次询问）
- 代理配置支持
- 端口转发管理
- 服务器别名管理

示例：
  gotssh -m          # 进入交互式服务器管理界面
  gotssh -o          # 进入交互式凭证管理界面
  gotssh -a server1  # 连接到别名为server1的服务器
  gotssh -t          # 管理端口转发
  gotssh --at tunnel1 # 启动别名为tunnel1的端口转发`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 初始化配置管理器
		var err error
		configManager, err = config.NewManager("")
		if err != nil {
			return fmt.Errorf("初始化配置管理器失败: %w", err)
		}

		// 初始化端口转发管理器
		forwardManager = forward.NewManager(configManager)

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// 检查是否使用了标志
		if manage, _ := cmd.Flags().GetBool("manage"); manage {
			return manageCmd.RunE(cmd, args)
		}
		
		if connectTarget, _ := cmd.Flags().GetString("connect"); connectTarget != "" {
			return connectCmd.RunE(cmd, []string{connectTarget})
		}
		
		if tunnel, _ := cmd.Flags().GetBool("tunnel"); tunnel {
			return tunnelCmd.RunE(cmd, args)
		}
		
		if tunnelAlias, _ := cmd.Flags().GetString("at"); tunnelAlias != "" {
			return tunnelConnectCmd.RunE(cmd, []string{tunnelAlias})
		}
		
		if credential, _ := cmd.Flags().GetBool("credential"); credential {
			return credentialCmd.RunE(cmd, args)
		}
		
		// 如果没有指定任何标志，显示帮助信息
		cmd.Help()
		return nil
	},
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "执行命令失败: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// 添加子命令
	rootCmd.AddCommand(manageCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(tunnelCmd)
	rootCmd.AddCommand(tunnelConnectCmd)
	rootCmd.AddCommand(credentialCmd)
} 