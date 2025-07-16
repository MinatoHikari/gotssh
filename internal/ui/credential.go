package ui

import (
	"fmt"
	"os"
	"strings"

	"gotssh/internal/config"

	"github.com/manifoldco/promptui"
)

// CredentialMenu 凭证管理菜单结构
type CredentialMenu struct {
	configManager *config.Manager
}

// NewCredentialMenu 创建新的凭证管理菜单实例
func NewCredentialMenu(configManager *config.Manager) *CredentialMenu {
	return &CredentialMenu{
		configManager: configManager,
	}
}

// ShowCredentialMenu 显示凭证管理菜单
func (cm *CredentialMenu) ShowCredentialMenu() error {
	for {
		prompt := promptui.Select{
			Label: "凭证管理",
			Items: []string{
				"添加凭证",
				"查看凭证列表",
				"编辑凭证",
				"删除凭证",
				"退出",
			},
		}

		_, result, err := prompt.Run()
		if err != nil {
			return fmt.Errorf("选择凭证管理菜单失败: %w", err)
		}

		switch result {
		case "添加凭证":
			if err := cm.AddCredential(); err != nil {
				fmt.Printf("添加凭证失败: %v\n", err)
			}
		case "查看凭证列表":
			cm.ShowCredentialList()
		case "编辑凭证":
			if err := cm.EditCredential(); err != nil {
				fmt.Printf("编辑凭证失败: %v\n", err)
			}
		case "删除凭证":
			if err := cm.DeleteCredential(); err != nil {
				fmt.Printf("删除凭证失败: %v\n", err)
			}
		case "退出":
			return nil
		}
	}
}

// AddCredential 添加凭证
func (cm *CredentialMenu) AddCredential() error {
	cred := config.NewCredentialConfig()

	// 别名
	aliasPrompt := promptui.Prompt{
		Label: "凭证别名",
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("别名不能为空")
			}
			return nil
		},
	}
	alias, err := aliasPrompt.Run()
	if err != nil {
		return err
	}
	cred.Alias = alias

	// 用户名
	userPrompt := promptui.Prompt{
		Label: "用户名",
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("用户名不能为空")
			}
			return nil
		},
	}
	username, err := userPrompt.Run()
	if err != nil {
		return err
	}
	cred.Username = username

	// 凭证类型
	typePrompt := promptui.Select{
		Label: "凭证类型",
		Items: []string{
			"密码凭证",
			"SSH密钥凭证",
		},
	}
	_, typeResult, err := typePrompt.Run()
	if err != nil {
		return err
	}

	switch typeResult {
	case "密码凭证":
		cred.Type = config.CredentialTypePassword
		passwordPrompt := promptui.Prompt{
			Label: "密码",
			Mask:  '*',
			Validate: func(input string) error {
				if strings.TrimSpace(input) == "" {
					return fmt.Errorf("密码不能为空")
				}
				return nil
			},
		}
		password, err := passwordPrompt.Run()
		if err != nil {
			return err
		}
		cred.Password = password

	case "SSH密钥凭证":
		cred.Type = config.CredentialTypeKey
		
		// 选择密钥输入方式
		inputMethodPrompt := promptui.Select{
			Label: "密钥输入方式",
			Items: []string{
				"密钥文件路径",
				"直接输入密钥内容",
			},
		}
		_, inputMethod, err := inputMethodPrompt.Run()
		if err != nil {
			return err
		}

		if inputMethod == "密钥文件路径" {
			keyPathPrompt := promptui.Prompt{
				Label: "密钥文件路径",
				Validate: func(input string) error {
					if strings.TrimSpace(input) == "" {
						return fmt.Errorf("密钥文件路径不能为空")
					}
					// 检查文件是否存在
					if _, err := os.Stat(input); os.IsNotExist(err) {
						return fmt.Errorf("密钥文件不存在")
					}
					return nil
				},
			}
			keyPath, err := keyPathPrompt.Run()
			if err != nil {
				return err
			}
			cred.KeyPath = keyPath
		} else {
			keyContentPrompt := promptui.Prompt{
				Label: "密钥内容 (多行输入，输入END结束)",
			}
			fmt.Println("请输入SSH私钥内容，最后一行输入END结束:")
			var keyContent strings.Builder
			for {
				line, err := keyContentPrompt.Run()
				if err != nil {
					return err
				}
				if line == "END" {
					break
				}
				keyContent.WriteString(line + "\n")
			}
			
			if keyContent.Len() == 0 {
				return fmt.Errorf("密钥内容不能为空")
			}
			cred.KeyContent = keyContent.String()
		}

		// 密钥密码短语（可选）
		passphrasePrompt := promptui.Prompt{
			Label: "密钥密码短语 (可选，直接回车跳过)",
			Mask:  '*',
		}
		passphrase, err := passphrasePrompt.Run()
		if err != nil {
			return err
		}
		cred.KeyPassphrase = passphrase
	}

	// 描述（可选）
	descPrompt := promptui.Prompt{
		Label: "描述 (可选)",
	}
	desc, err := descPrompt.Run()
	if err != nil {
		return err
	}
	cred.Description = desc

	// 保存凭证
	if err := cm.configManager.AddCredential(cred); err != nil {
		return fmt.Errorf("保存凭证失败: %w", err)
	}

	fmt.Printf("✅ 凭证 '%s' 添加成功！\n", cred.Alias)
	return nil
}

// ShowCredentialList 显示凭证列表
func (cm *CredentialMenu) ShowCredentialList() {
	credentials := cm.configManager.ListCredentials()
	if len(credentials) == 0 {
		fmt.Println("暂无凭证配置")
		return
	}

	fmt.Println("\n=== 凭证列表 ===")
	for i, cred := range credentials {
		fmt.Printf("%d. [%s] %s (%s)", i+1, cred.Alias, cred.Username, cred.Type)
		if cred.Description != "" {
			fmt.Printf(" - %s", cred.Description)
		}
		fmt.Printf(" [创建时间: %s]", cred.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}
	fmt.Println()
}

// EditCredential 编辑凭证
func (cm *CredentialMenu) EditCredential() error {
	credentials := cm.configManager.ListCredentials()
	if len(credentials) == 0 {
		fmt.Println("暂无凭证配置")
		return nil
	}

	// 选择要编辑的凭证
	var items []string
	for _, cred := range credentials {
		item := fmt.Sprintf("[%s] %s (%s)", cred.Alias, cred.Username, cred.Type)
		items = append(items, item)
	}

	prompt := promptui.Select{
		Label: "选择要编辑的凭证",
		Items: items,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return err
	}

	cred := credentials[index]
	originalCred := *cred // 复制原始配置

	fmt.Printf("正在编辑凭证: [%s] %s\n", cred.Alias, cred.Username)

	// 编辑别名
	aliasPrompt := promptui.Prompt{
		Label:   "凭证别名",
		Default: cred.Alias,
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("别名不能为空")
			}
			return nil
		},
	}
	alias, err := aliasPrompt.Run()
	if err != nil {
		return err
	}
	cred.Alias = alias

	// 编辑用户名
	userPrompt := promptui.Prompt{
		Label:   "用户名",
		Default: cred.Username,
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("用户名不能为空")
			}
			return nil
		},
	}
	username, err := userPrompt.Run()
	if err != nil {
		return err
	}
	cred.Username = username

	// 编辑凭证类型
	typeItems := []string{"密码凭证", "SSH密钥凭证"}
	var currentTypeIndex int
	if cred.Type == config.CredentialTypeKey {
		currentTypeIndex = 1
	}

	typePrompt := promptui.Select{
		Label:     "凭证类型",
		Items:     typeItems,
		CursorPos: currentTypeIndex,
	}
	_, typeResult, err := typePrompt.Run()
	if err != nil {
		return err
	}

	switch typeResult {
	case "密码凭证":
		cred.Type = config.CredentialTypePassword
		passwordPrompt := promptui.Prompt{
			Label: "密码",
			Mask:  '*',
			Validate: func(input string) error {
				if strings.TrimSpace(input) == "" {
					return fmt.Errorf("密码不能为空")
				}
				return nil
			},
		}
		password, err := passwordPrompt.Run()
		if err != nil {
			return err
		}
		cred.Password = password
		// 清空密钥相关字段
		cred.KeyPath = ""
		cred.KeyContent = ""
		cred.KeyPassphrase = ""

	case "SSH密钥凭证":
		cred.Type = config.CredentialTypeKey
		
		// 选择密钥输入方式
		inputMethods := []string{"密钥文件路径", "直接输入密钥内容"}
		var currentInputIndex int
		if cred.KeyContent != "" {
			currentInputIndex = 1
		}

		inputMethodPrompt := promptui.Select{
			Label:     "密钥输入方式",
			Items:     inputMethods,
			CursorPos: currentInputIndex,
		}
		_, inputMethod, err := inputMethodPrompt.Run()
		if err != nil {
			return err
		}

		if inputMethod == "密钥文件路径" {
			keyPathPrompt := promptui.Prompt{
				Label:   "密钥文件路径",
				Default: cred.KeyPath,
				Validate: func(input string) error {
					if strings.TrimSpace(input) == "" {
						return fmt.Errorf("密钥文件路径不能为空")
					}
					if _, err := os.Stat(input); os.IsNotExist(err) {
						return fmt.Errorf("密钥文件不存在")
					}
					return nil
				},
			}
			keyPath, err := keyPathPrompt.Run()
			if err != nil {
				return err
			}
			cred.KeyPath = keyPath
			cred.KeyContent = "" // 清空密钥内容
		} else {
			fmt.Println("请输入SSH私钥内容，最后一行输入END结束:")
			if cred.KeyContent != "" {
				fmt.Printf("当前密钥内容预览:\n%s...\n", cred.KeyContent[:min(100, len(cred.KeyContent))])
			}
			
			keyContentPrompt := promptui.Prompt{
				Label: "密钥内容 (多行输入，输入END结束)",
			}
			var keyContent strings.Builder
			for {
				line, err := keyContentPrompt.Run()
				if err != nil {
					return err
				}
				if line == "END" {
					break
				}
				keyContent.WriteString(line + "\n")
			}
			
			if keyContent.Len() == 0 {
				return fmt.Errorf("密钥内容不能为空")
			}
			cred.KeyContent = keyContent.String()
			cred.KeyPath = "" // 清空密钥路径
		}

		// 密钥密码短语
		passphrasePrompt := promptui.Prompt{
			Label:   "密钥密码短语 (可选)",
			Default: cred.KeyPassphrase,
			Mask:    '*',
		}
		passphrase, err := passphrasePrompt.Run()
		if err != nil {
			return err
		}
		cred.KeyPassphrase = passphrase
		// 清空密码字段
		cred.Password = ""
	}

	// 编辑描述
	descPrompt := promptui.Prompt{
		Label:   "描述",
		Default: cred.Description,
	}
	desc, err := descPrompt.Run()
	if err != nil {
		return err
	}
	cred.Description = desc

	// 确认保存
	confirmPrompt := promptui.Select{
		Label: "确定保存修改吗？",
		Items: []string{"是", "否"},
	}
	_, confirmResult, err := confirmPrompt.Run()
	if err != nil {
		return err
	}

	if confirmResult == "是" {
		if err := cm.configManager.UpdateCredential(cred.ID, cred); err != nil {
			return fmt.Errorf("保存凭证配置失败: %w", err)
		}
		fmt.Printf("✅ 凭证 '%s' 配置已更新！\n", cred.Alias)
	} else {
		// 还原配置
		*cred = originalCred
		fmt.Println("❌ 取消编辑")
	}

	return nil
}

// DeleteCredential 删除凭证
func (cm *CredentialMenu) DeleteCredential() error {
	credentials := cm.configManager.ListCredentials()
	if len(credentials) == 0 {
		fmt.Println("暂无凭证配置")
		return nil
	}

	// 选择要删除的凭证
	var items []string
	for _, cred := range credentials {
		item := fmt.Sprintf("[%s] %s (%s)", cred.Alias, cred.Username, cred.Type)
		items = append(items, item)
	}

	prompt := promptui.Select{
		Label: "选择要删除的凭证",
		Items: items,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return err
	}

	cred := credentials[index]

	// 确认删除
	confirmPrompt := promptui.Select{
		Label: fmt.Sprintf("确定要删除凭证 '%s' 吗？", cred.Alias),
		Items: []string{"是", "否"},
	}

	_, confirmResult, err := confirmPrompt.Run()
	if err != nil {
		return err
	}

	if confirmResult == "是" {
		if err := cm.configManager.DeleteCredential(cred.ID); err != nil {
			return fmt.Errorf("删除凭证失败: %w", err)
		}
		fmt.Printf("✅ 凭证 '%s' 已删除\n", cred.Alias)
	}

	return nil
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
} 