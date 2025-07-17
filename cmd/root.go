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

// preprocessCredentialFlag 预处理 -o 参数，支持 -o value 和 -o=value 两种形式
func preprocessCredentialFlag() {
	// 检查命令行参数中是否有 -o 后跟着值的情况
	args := os.Args[1:]
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-o" && i+1 < len(args) && args[i+1] != "" && args[i+1][0] != '-' {
			// 将 -o value 转换为 -o=value
			os.Args[i+1] = "-o=" + args[i+1]
			// 移除原来的 value 参数
			os.Args = append(os.Args[:i+2], os.Args[i+3:]...)
			break
		}
	}
}

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
  gotssh -m                    # 进入交互式服务器管理界面
  gotssh -o                    # 进入交互式凭证管理界面
  gotssh -a server1            # 连接到别名为server1的服务器
  gotssh -a server1 -o mycred  # 使用指定凭证连接到服务器
  gotssh -a 192.168.1.100 -o mycred  # 使用凭证直接连接IP地址
  gotssh -t                    # 管理端口转发
  gotssh --at tunnel1          # 启动别名为tunnel1的端口转发`,
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

		// 检查是否同时使用了 -a 和 -o 参数
		connectTarget, _ := cmd.Flags().GetString("connect")
		credentialAlias, _ := cmd.Flags().GetString("credential")

		// 检查 -o 标志是否被设置（即使值为空）
		credentialFlagSet := cmd.Flags().Changed("credential")

		if connectTarget != "" && credentialAlias != "" && credentialAlias != "CREDENTIAL_MANAGEMENT_MODE" {
			// 同时使用 -a 和 -o 参数，使用指定凭证连接
			return connectWithCredentialCmd.RunE(cmd, []string{connectTarget, credentialAlias})
		}

		if connectTarget != "" {
			return connectCmd.RunE(cmd, []string{connectTarget})
		}

		if tunnel, _ := cmd.Flags().GetBool("tunnel"); tunnel {
			return tunnelCmd.RunE(cmd, args)
		}

		if tunnelAlias, _ := cmd.Flags().GetString("at"); tunnelAlias != "" {
			return tunnelConnectCmd.RunE(cmd, []string{tunnelAlias})
		}

		if credentialFlagSet {
			// 如果 -o 标志被设置
			if credentialAlias == "" || credentialAlias == "CREDENTIAL_MANAGEMENT_MODE" {
				// 单独使用 -o 参数（没有值或默认值），进入凭证管理界面
				return credentialCmd.RunE(cmd, args)
			} else {
				// -o 参数有具体值但没有 -a 参数，这是无效的使用方式
				return fmt.Errorf("使用 -o 参数指定凭证时，必须同时使用 -a 参数指定服务器")
			}
		}

		// 如果没有指定任何标志，显示帮助信息
		cmd.Help()
		return nil
	},
}

// Execute 执行根命令
func Execute() {
	// 预处理参数：将 -o value 转换为 -o=value 以支持两种语法
	preprocessCredentialFlag()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "执行命令失败: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// 添加子命令
	rootCmd.AddCommand(manageCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(connectWithCredentialCmd)
	rootCmd.AddCommand(tunnelCmd)
	rootCmd.AddCommand(tunnelConnectCmd)
	rootCmd.AddCommand(credentialCmd)
}
