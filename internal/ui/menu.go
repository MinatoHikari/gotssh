package ui

import (
	"fmt"
	"strconv"
	"strings"

	"gotssh/internal/config"
	"gotssh/internal/forward"
	"gotssh/internal/ssh"

	"github.com/manifoldco/promptui"
)

// Menu 主菜单结构
type Menu struct {
	configManager  *config.Manager
	forwardManager *forward.Manager
}

// NewMenu 创建新的菜单实例
func NewMenu(configManager *config.Manager, forwardManager *forward.Manager) *Menu {
	return &Menu{
		configManager:  configManager,
		forwardManager: forwardManager,
	}
}

// ShowServerMenu 显示服务器管理菜单
func (m *Menu) ShowServerMenu() error {
	for {
		prompt := promptui.Select{
			Label: "服务器管理",
			Items: []string{
				"添加服务器",
				"查看服务器列表",
				"连接服务器",
				"编辑服务器",
				"删除服务器",
				"测试连接",
				"退出",
			},
		}

		_, result, err := prompt.Run()
		if err != nil {
			return fmt.Errorf("选择服务器管理菜单失败: %w", err)
		}

		switch result {
		case "添加服务器":
			if err := m.AddServer(); err != nil {
				fmt.Printf("添加服务器失败: %v\n", err)
			}
		case "查看服务器列表":
			m.ShowServerList()
		case "连接服务器":
			if err := m.ConnectServer(); err != nil {
				fmt.Printf("连接服务器失败: %v\n", err)
			}
		case "编辑服务器":
			if err := m.EditServer(); err != nil {
				fmt.Printf("编辑服务器失败: %v\n", err)
			}
		case "删除服务器":
			if err := m.DeleteServer(); err != nil {
				fmt.Printf("删除服务器失败: %v\n", err)
			}
		case "测试连接":
			if err := m.TestServerConnection(); err != nil {
				fmt.Printf("测试连接失败: %v\n", err)
			}
		case "退出":
			return nil
		}
	}
}

// AddServer 添加服务器
func (m *Menu) AddServer() error {
	// 主机地址
	hostPrompt := promptui.Prompt{
		Label: "主机地址",
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("主机地址不能为空")
			}
			return nil
		},
	}
	host, err := hostPrompt.Run()
	if err != nil {
		return err
	}

	// 端口
	portPrompt := promptui.Prompt{
		Label:   "端口 (默认22)",
		Default: "22",
		Validate: func(input string) error {
			if input == "" {
				return nil
			}
			port, err := strconv.Atoi(input)
			if err != nil || port < 1 || port > 65535 {
				return fmt.Errorf("端口必须是1-65535之间的数字")
			}
			return nil
		},
	}
	portStr, err := portPrompt.Run()
	if err != nil {
		return err
	}
	port, _ := strconv.Atoi(portStr)

	// 用户名
	userPrompt := promptui.Prompt{
		Label:   "用户名 (默认root)",
		Default: "root",
	}
	user, err := userPrompt.Run()
	if err != nil {
		return err
	}

	// 别名
	aliasPrompt := promptui.Prompt{
		Label: "别名 (可选)",
	}
	alias, err := aliasPrompt.Run()
	if err != nil {
		return err
	}

	// 认证类型
	authPrompt := promptui.Select{
		Label: "认证类型",
		Items: []string{
			"每次询问",
			"密码认证",
			"密钥认证",
			"登录凭证",
		},
	}
	_, authResult, err := authPrompt.Run()
	if err != nil {
		return err
	}

	// 创建服务器配置
	server := config.NewServerConfig(host)
	server.Port = port
	server.User = user
	server.Alias = alias

	switch authResult {
	case "密码认证":
		server.AuthType = config.AuthTypePassword
		passwordPrompt := promptui.Prompt{
			Label: "密码",
			Mask:  '*',
		}
		password, err := passwordPrompt.Run()
		if err != nil {
			return err
		}
		server.Password = password

	case "密钥认证":
		server.AuthType = config.AuthTypeKey
		keyPrompt := promptui.Prompt{
			Label: "密钥文件路径",
			Validate: func(input string) error {
				if strings.TrimSpace(input) == "" {
					return fmt.Errorf("密钥文件路径不能为空")
				}
				return nil
			},
		}
		keyPath, err := keyPrompt.Run()
		if err != nil {
			return err
		}
		server.KeyPath = keyPath

		// 密钥密码短语（可选）
		passphrasePrompt := promptui.Prompt{
			Label: "密钥密码短语 (可选)",
			Mask:  '*',
		}
		passphrase, err := passphrasePrompt.Run()
		if err != nil {
			return err
		}
		server.KeyPassphrase = passphrase

	case "登录凭证":
		server.AuthType = config.AuthTypeCredential
		// 选择凭证
		credentials := m.configManager.ListCredentials()
		if len(credentials) > 0 {
			var credItems []string
			credItems = append(credItems, "不使用凭证")
			for _, cred := range credentials {
				item := fmt.Sprintf("[%s] %s (%s)", cred.Alias, cred.Username, cred.Type)
				credItems = append(credItems, item)
			}

			credPrompt := promptui.Select{
				Label: "选择凭证",
				Items: credItems,
			}
			credIndex, _, err := credPrompt.Run()
			if err != nil {
				return err
			}

			if credIndex > 0 {
				server.CredentialID = credentials[credIndex-1].ID
			}
		} else {
			fmt.Println("提示: 暂无可用凭证，请先使用 -o 命令添加凭证")
		}

	case "每次询问":
		server.AuthType = config.AuthTypeAsk
	}

	// 代理配置（可选）
	proxyPrompt := promptui.Select{
		Label: "是否配置代理",
		Items: []string{"是", "否"},
	}
	_, proxyResult, err := proxyPrompt.Run()
	if err != nil {
		return err
	}
	useProxy := proxyResult == "是"

	if useProxy {
		proxyConfig, err := m.configureProxy()
		if err != nil {
			return err
		}
		server.Proxy = proxyConfig
	}

	// 启动脚本（可选）
	scriptPrompt := promptui.Prompt{
		Label: "启动脚本 (可选)",
	}
	script, err := scriptPrompt.Run()
	if err != nil {
		return err
	}
	server.StartupScript = script

	// 描述（可选）
	descPrompt := promptui.Prompt{
		Label: "描述 (可选)",
	}
	desc, err := descPrompt.Run()
	if err != nil {
		return err
	}
	server.Description = desc

	// 保存服务器配置
	if err := m.configManager.AddServer(server); err != nil {
		return fmt.Errorf("保存服务器配置失败: %w", err)
	}

	fmt.Printf("服务器 %s 添加成功！\n", server.Host)
	return nil
}

// configureProxy 配置代理
func (m *Menu) configureProxy() (*config.ProxyConfig, error) {
	// 代理类型
	typePrompt := promptui.Select{
		Label: "代理类型",
		Items: []string{"http", "socks5"},
	}
	_, proxyType, err := typePrompt.Run()
	if err != nil {
		return nil, err
	}

	// 代理主机
	hostPrompt := promptui.Prompt{
		Label: "代理主机",
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("代理主机不能为空")
			}
			return nil
		},
	}
	host, err := hostPrompt.Run()
	if err != nil {
		return nil, err
	}

	// 代理端口
	portPrompt := promptui.Prompt{
		Label: "代理端口",
		Validate: func(input string) error {
			port, err := strconv.Atoi(input)
			if err != nil || port < 1 || port > 65535 {
				return fmt.Errorf("端口必须是1-65535之间的数字")
			}
			return nil
		},
	}
	portStr, err := portPrompt.Run()
	if err != nil {
		return nil, err
	}
	port, _ := strconv.Atoi(portStr)

	// 代理用户名（可选）
	userPrompt := promptui.Prompt{
		Label: "代理用户名 (可选)",
	}
	username, err := userPrompt.Run()
	if err != nil {
		return nil, err
	}

	// 代理密码（可选）
	var password string
	if username != "" {
		passwordPrompt := promptui.Prompt{
			Label: "代理密码 (可选)",
			Mask:  '*',
		}
		password, err = passwordPrompt.Run()
		if err != nil {
			return nil, err
		}
	}

	return &config.ProxyConfig{
		Type:     proxyType,
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}, nil
}

// ShowServerList 显示服务器列表
func (m *Menu) ShowServerList() {
	if m.configManager == nil {
		fmt.Println("配置管理器未初始化")
		return
	}

	servers := m.configManager.ListServers()
	if len(servers) == 0 {
		fmt.Println("暂无服务器配置")
		return
	}

	fmt.Println("\n=== 服务器列表 ===")
	for i, server := range servers {
		fmt.Printf("%d. ", i+1)
		if server.Alias != "" {
			fmt.Printf("[%s] ", server.Alias)
		}
		fmt.Printf("%s@%s:%d", server.User, server.Host, server.Port)
		if server.Description != "" {
			fmt.Printf(" - %s", server.Description)
		}
		fmt.Printf(" (认证: %s)", server.AuthType)
		if server.Proxy != nil {
			fmt.Printf(" [代理: %s]", server.Proxy.Type)
		}
		fmt.Printf(" [创建时间: %s]", server.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}
	fmt.Println()
}

// EditServer 编辑服务器
func (m *Menu) EditServer() error {
	servers := m.configManager.ListServers()
	if len(servers) == 0 {
		fmt.Println("暂无服务器配置")
		return nil
	}

	// 选择要编辑的服务器
	var items []string
	for _, server := range servers {
		item := fmt.Sprintf("%s@%s:%d", server.User, server.Host, server.Port)
		if server.Alias != "" {
			item = fmt.Sprintf("[%s] %s", server.Alias, item)
		}
		items = append(items, item)
	}

	prompt := promptui.Select{
		Label: "选择要编辑的服务器",
		Items: items,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return err
	}

	server := servers[index]
	originalServer := *server // 复制原始配置

	fmt.Printf("正在编辑服务器: %s@%s:%d\n", server.User, server.Host, server.Port)

	// 编辑主机地址
	hostPrompt := promptui.Prompt{
		Label:   "主机地址",
		Default: server.Host,
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("主机地址不能为空")
			}
			return nil
		},
	}
	host, err := hostPrompt.Run()
	if err != nil {
		return err
	}
	server.Host = host

	// 编辑端口
	portPrompt := promptui.Prompt{
		Label:   "端口",
		Default: fmt.Sprintf("%d", server.Port),
		Validate: func(input string) error {
			if input == "" {
				return nil
			}
			port, err := strconv.Atoi(input)
			if err != nil || port < 1 || port > 65535 {
				return fmt.Errorf("端口必须是1-65535之间的数字")
			}
			return nil
		},
	}
	portStr, err := portPrompt.Run()
	if err != nil {
		return err
	}
	server.Port, _ = strconv.Atoi(portStr)

	// 编辑用户名
	userPrompt := promptui.Prompt{
		Label:   "用户名",
		Default: server.User,
	}
	user, err := userPrompt.Run()
	if err != nil {
		return err
	}
	server.User = user

	// 编辑别名
	aliasPrompt := promptui.Prompt{
		Label:   "别名",
		Default: server.Alias,
	}
	alias, err := aliasPrompt.Run()
	if err != nil {
		return err
	}
	server.Alias = alias

	// 编辑描述
	descPrompt := promptui.Prompt{
		Label:   "描述",
		Default: server.Description,
	}
	desc, err := descPrompt.Run()
	if err != nil {
		return err
	}
	server.Description = desc

	// 编辑启动脚本
	scriptPrompt := promptui.Prompt{
		Label:   "启动脚本",
		Default: server.StartupScript,
	}
	script, err := scriptPrompt.Run()
	if err != nil {
		return err
	}
	server.StartupScript = script

	// 编辑认证类型
	authTypes := []string{"每次询问", "密码认证", "密钥认证", "登录凭证"}
	var currentAuthIndex int
	switch server.AuthType {
	case config.AuthTypeAsk:
		currentAuthIndex = 0
	case config.AuthTypePassword:
		currentAuthIndex = 1
	case config.AuthTypeKey:
		currentAuthIndex = 2
	case config.AuthTypeCredential:
		currentAuthIndex = 3
	}

	authPrompt := promptui.Select{
		Label:     "认证类型",
		Items:     authTypes,
		CursorPos: currentAuthIndex,
	}
	_, authResult, err := authPrompt.Run()
	if err != nil {
		return err
	}

	switch authResult {
	case "密码认证":
		server.AuthType = config.AuthTypePassword
		passwordPrompt := promptui.Prompt{
			Label:   "密码",
			Mask:    '*',
			Default: server.Password,
		}
		password, err := passwordPrompt.Run()
		if err != nil {
			return err
		}
		server.Password = password
		server.CredentialID = ""

	case "密钥认证":
		server.AuthType = config.AuthTypeKey
		keyPrompt := promptui.Prompt{
			Label:   "密钥文件路径",
			Default: server.KeyPath,
			Validate: func(input string) error {
				if strings.TrimSpace(input) == "" {
					return fmt.Errorf("密钥文件路径不能为空")
				}
				return nil
			},
		}
		keyPath, err := keyPrompt.Run()
		if err != nil {
			return err
		}
		server.KeyPath = keyPath

		passphrasePrompt := promptui.Prompt{
			Label:   "密钥密码短语 (可选)",
			Mask:    '*',
			Default: server.KeyPassphrase,
		}
		passphrase, err := passphrasePrompt.Run()
		if err != nil {
			return err
		}
		server.KeyPassphrase = passphrase
		server.CredentialID = ""

	case "登录凭证":
		server.AuthType = config.AuthTypeCredential
		// 选择凭证
		credentials := m.configManager.ListCredentials()
		if len(credentials) > 0 {
			var credItems []string
			credItems = append(credItems, "不使用凭证")
			for _, cred := range credentials {
				item := fmt.Sprintf("[%s] %s (%s)", cred.Alias, cred.Username, cred.Type)
				credItems = append(credItems, item)
			}

			credPrompt := promptui.Select{
				Label: "选择凭证",
				Items: credItems,
			}
			credIndex, _, err := credPrompt.Run()
			if err != nil {
				return err
			}

			if credIndex == 0 {
				server.CredentialID = ""
			} else {
				server.CredentialID = credentials[credIndex-1].ID
			}
		} else {
			server.CredentialID = ""
		}

	case "每次询问":
		server.AuthType = config.AuthTypeAsk
		server.CredentialID = ""
	}

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
		if err := m.configManager.UpdateServer(server.ID, server); err != nil {
			return fmt.Errorf("保存服务器配置失败: %w", err)
		}
		fmt.Printf("✅ 服务器 %s 配置已更新！\n", server.Host)
	} else {
		// 还原配置
		*server = originalServer
		fmt.Println("❌ 取消编辑")
	}

	return nil
}

// DeleteServer 删除服务器
func (m *Menu) DeleteServer() error {
	servers := m.configManager.ListServers()
	if len(servers) == 0 {
		fmt.Println("暂无服务器配置")
		return nil
	}

	// 选择要删除的服务器
	var items []string
	for _, server := range servers {
		item := fmt.Sprintf("%s@%s:%d", server.User, server.Host, server.Port)
		if server.Alias != "" {
			item = fmt.Sprintf("[%s] %s", server.Alias, item)
		}
		items = append(items, item)
	}

	prompt := promptui.Select{
		Label: "选择要删除的服务器",
		Items: items,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return err
	}

	server := servers[index]

	// 确认删除
	confirmPrompt := promptui.Select{
		Label: fmt.Sprintf("确定要删除服务器 %s@%s:%d 吗？", server.User, server.Host, server.Port),
		Items: []string{"是", "否"},
	}

	_, confirmResult, err := confirmPrompt.Run()
	if err != nil {
		return err
	}

	if confirmResult == "是" {
		if err := m.configManager.DeleteServer(server.ID); err != nil {
			return fmt.Errorf("删除服务器失败: %w", err)
		}
		fmt.Printf("服务器 %s@%s:%d 已删除\n", server.User, server.Host, server.Port)
	}

	return nil
}

// ConnectServer 连接服务器
func (m *Menu) ConnectServer() error {
	servers := m.configManager.ListServers()
	if len(servers) == 0 {
		fmt.Println("暂无服务器配置")
		return nil
	}

	// 选择要连接的服务器
	var items []string
	for _, server := range servers {
		item := fmt.Sprintf("%s@%s:%d", server.User, server.Host, server.Port)
		if server.Alias != "" {
			item = fmt.Sprintf("[%s] %s", server.Alias, item)
		}
		items = append(items, item)
	}

	prompt := promptui.Select{
		Label: "选择要连接的服务器",
		Items: items,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return err
	}

	server := servers[index]

	fmt.Printf("正在连接到 %s@%s:%d", server.User, server.Host, server.Port)
	if server.Alias != "" {
		fmt.Printf(" [%s]", server.Alias)
	}
	fmt.Println()

	// 创建SSH客户端
	client := ssh.NewClient(server, m.configManager)

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
}

// TestServerConnection 测试服务器连接
func (m *Menu) TestServerConnection() error {
	servers := m.configManager.ListServers()
	if len(servers) == 0 {
		fmt.Println("暂无服务器配置")
		return nil
	}

	// 选择要测试的服务器
	var items []string
	for _, server := range servers {
		item := fmt.Sprintf("%s@%s:%d", server.User, server.Host, server.Port)
		if server.Alias != "" {
			item = fmt.Sprintf("[%s] %s", server.Alias, item)
		}
		items = append(items, item)
	}

	prompt := promptui.Select{
		Label: "选择要测试的服务器",
		Items: items,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return err
	}

	server := servers[index]

	fmt.Printf("正在测试连接到 %s@%s:%d ...\n", server.User, server.Host, server.Port)

	// 创建SSH客户端进行测试
	client := ssh.NewClient(server, m.configManager)
	err = client.Connect()
	if err != nil {
		fmt.Printf("❌ 连接失败: %v\n", err)
		return nil
	}
	defer client.Close()

	// 执行简单命令测试
	output, err := client.ExecuteCommand("echo 'Hello from gotssh'")
	if err != nil {
		fmt.Printf("❌ 命令执行失败: %v\n", err)
		return nil
	}

	fmt.Printf("✅ 连接成功!\n")
	fmt.Printf("测试命令输出: %s\n", strings.TrimSpace(output))

	return nil
}

// ShowPortForwardMenu 显示端口转发管理菜单
func (m *Menu) ShowPortForwardMenu() error {
	for {
		prompt := promptui.Select{
			Label: "端口转发管理",
			Items: []string{
				"添加端口转发",
				"查看端口转发列表",
				"启动端口转发",
				"停止端口转发",
				"删除端口转发",
				"测试端口转发",
				"返回主菜单",
			},
		}

		_, result, err := prompt.Run()
		if err != nil {
			return fmt.Errorf("选择端口转发管理菜单失败: %w", err)
		}

		switch result {
		case "添加端口转发":
			if err := m.AddPortForward(); err != nil {
				fmt.Printf("添加端口转发失败: %v\n", err)
			}
		case "查看端口转发列表":
			m.ShowPortForwardList()
		case "启动端口转发":
			if err := m.StartPortForward(); err != nil {
				fmt.Printf("启动端口转发失败: %v\n", err)
			}
		case "停止端口转发":
			if err := m.StopPortForward(); err != nil {
				fmt.Printf("停止端口转发失败: %v\n", err)
			}
		case "删除端口转发":
			if err := m.DeletePortForward(); err != nil {
				fmt.Printf("删除端口转发失败: %v\n", err)
			}
		case "测试端口转发":
			if err := m.TestPortForward(); err != nil {
				fmt.Printf("测试端口转发失败: %v\n", err)
			}
		case "返回主菜单":
			return nil
		}
	}
}

// AddPortForward 添加端口转发
func (m *Menu) AddPortForward() error {
	// 选择服务器
	servers := m.configManager.ListServers()
	if len(servers) == 0 {
		fmt.Println("请先添加服务器")
		return nil
	}

	var serverItems []string
	for _, server := range servers {
		item := fmt.Sprintf("%s@%s:%d", server.User, server.Host, server.Port)
		if server.Alias != "" {
			item = fmt.Sprintf("[%s] %s", server.Alias, item)
		}
		serverItems = append(serverItems, item)
	}

	serverPrompt := promptui.Select{
		Label: "选择服务器",
		Items: serverItems,
	}

	serverIndex, _, err := serverPrompt.Run()
	if err != nil {
		return err
	}

	server := servers[serverIndex]

	// 转发类型
	typePrompt := promptui.Select{
		Label: "转发类型",
		Items: []string{
			"本地端口转发",
			"远程端口转发",
		},
	}
	_, typeResult, err := typePrompt.Run()
	if err != nil {
		return err
	}

	// 创建端口转发配置
	pf := config.NewPortForwardConfig(server.ID)

	if typeResult == "本地端口转发" {
		pf.Type = config.ForwardTypeLocal
	} else {
		pf.Type = config.ForwardTypeRemote
	}

	// 本地主机
	localHostPrompt := promptui.Prompt{
		Label:   "本地主机 (默认127.0.0.1)",
		Default: "127.0.0.1",
	}
	localHost, err := localHostPrompt.Run()
	if err != nil {
		return err
	}
	pf.LocalHost = localHost

	// 本地端口
	localPortPrompt := promptui.Prompt{
		Label: "本地端口",
		Validate: func(input string) error {
			port, err := strconv.Atoi(input)
			if err != nil || port < 1 || port > 65535 {
				return fmt.Errorf("端口必须是1-65535之间的数字")
			}
			return nil
		},
	}
	localPortStr, err := localPortPrompt.Run()
	if err != nil {
		return err
	}
	pf.LocalPort, _ = strconv.Atoi(localPortStr)

	// 远程主机
	remoteHostPrompt := promptui.Prompt{
		Label:   "远程主机 (默认127.0.0.1)",
		Default: "127.0.0.1",
	}
	remoteHost, err := remoteHostPrompt.Run()
	if err != nil {
		return err
	}
	pf.RemoteHost = remoteHost

	// 远程端口
	remotePortPrompt := promptui.Prompt{
		Label: "远程端口",
		Validate: func(input string) error {
			port, err := strconv.Atoi(input)
			if err != nil || port < 1 || port > 65535 {
				return fmt.Errorf("端口必须是1-65535之间的数字")
			}
			return nil
		},
	}
	remotePortStr, err := remotePortPrompt.Run()
	if err != nil {
		return err
	}
	pf.RemotePort, _ = strconv.Atoi(remotePortStr)

	// 别名
	aliasPrompt := promptui.Prompt{
		Label: "别名 (可选)",
	}
	alias, err := aliasPrompt.Run()
	if err != nil {
		return err
	}
	pf.Alias = alias

	// 描述
	descPrompt := promptui.Prompt{
		Label: "描述 (可选)",
	}
	desc, err := descPrompt.Run()
	if err != nil {
		return err
	}
	pf.Description = desc

	// 保存端口转发配置
	if err := m.configManager.AddPortForward(pf); err != nil {
		return fmt.Errorf("保存端口转发配置失败: %w", err)
	}

	fmt.Printf("端口转发配置已添加: %s:%d -> %s:%d\n",
		pf.LocalHost, pf.LocalPort, pf.RemoteHost, pf.RemotePort)
	return nil
}

// ShowPortForwardList 显示端口转发列表
func (m *Menu) ShowPortForwardList() {
	if m.configManager == nil {
		fmt.Println("配置管理器未初始化")
		return
	}

	pfs := m.configManager.ListPortForwards()
	if len(pfs) == 0 {
		fmt.Println("暂无端口转发配置")
		return
	}

	fmt.Println("\n=== 端口转发列表 ===")
	for i, pf := range pfs {
		fmt.Printf("%d. ", i+1)
		if pf.Alias != "" {
			fmt.Printf("[%s] ", pf.Alias)
		}
		fmt.Printf("%s:%d -> %s:%d",
			pf.LocalHost, pf.LocalPort, pf.RemoteHost, pf.RemotePort)
		fmt.Printf(" (%s)", pf.Type)

		// 显示状态
		status := m.forwardManager.GetForwardStatus(pf.ID)
		fmt.Printf(" [状态: %s]", status)

		if pf.Description != "" {
			fmt.Printf(" - %s", pf.Description)
		}
		fmt.Printf(" [创建时间: %s]", pf.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}
	fmt.Println()
}

// StartPortForward 启动端口转发
func (m *Menu) StartPortForward() error {
	pfs := m.configManager.ListPortForwards()
	if len(pfs) == 0 {
		fmt.Println("暂无端口转发配置")
		return nil
	}

	// 过滤出未启动的端口转发
	var availablePFs []*config.PortForwardConfig
	for _, pf := range pfs {
		if !m.forwardManager.IsForwardActive(pf.ID) {
			availablePFs = append(availablePFs, pf)
		}
	}

	if len(availablePFs) == 0 {
		fmt.Println("所有端口转发都已启动")
		return nil
	}

	// 选择要启动的端口转发
	var items []string
	for _, pf := range availablePFs {
		item := fmt.Sprintf("%s:%d -> %s:%d (%s)",
			pf.LocalHost, pf.LocalPort, pf.RemoteHost, pf.RemotePort, pf.Type)
		if pf.Alias != "" {
			item = fmt.Sprintf("[%s] %s", pf.Alias, item)
		}
		items = append(items, item)
	}

	prompt := promptui.Select{
		Label: "选择要启动的端口转发",
		Items: items,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return err
	}

	pf := availablePFs[index]

	fmt.Printf("正在启动端口转发: %s:%d -> %s:%d ...\n",
		pf.LocalHost, pf.LocalPort, pf.RemoteHost, pf.RemotePort)

	// 启动端口转发
	if err := m.forwardManager.StartPortForward(pf); err != nil {
		return fmt.Errorf("启动端口转发失败: %w", err)
	}

	fmt.Printf("✅ 端口转发已启动！\n")
	fmt.Printf("使用 Ctrl+C 或停止命令来停止转发\n")

	return nil
}

// StopPortForward 停止端口转发
func (m *Menu) StopPortForward() error {
	activeForwards := m.forwardManager.ListActiveForwards()
	if len(activeForwards) == 0 {
		fmt.Println("暂无运行中的端口转发")
		return nil
	}

	// 选择要停止的端口转发
	var items []string
	for _, forward := range activeForwards {
		pf := forward.Config
		item := fmt.Sprintf("%s:%d -> %s:%d (%s)",
			pf.LocalHost, pf.LocalPort, pf.RemoteHost, pf.RemotePort, pf.Type)
		if pf.Alias != "" {
			item = fmt.Sprintf("[%s] %s", pf.Alias, item)
		}
		items = append(items, item)
	}

	prompt := promptui.Select{
		Label: "选择要停止的端口转发",
		Items: items,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return err
	}

	forward := activeForwards[index]

	fmt.Printf("正在停止端口转发: %s:%d -> %s:%d ...\n",
		forward.Config.LocalHost, forward.Config.LocalPort,
		forward.Config.RemoteHost, forward.Config.RemotePort)

	// 停止端口转发
	if err := m.forwardManager.StopPortForward(forward.ID); err != nil {
		return fmt.Errorf("停止端口转发失败: %w", err)
	}

	fmt.Printf("✅ 端口转发已停止！\n")
	return nil
}

// DeletePortForward 删除端口转发
func (m *Menu) DeletePortForward() error {
	pfs := m.configManager.ListPortForwards()
	if len(pfs) == 0 {
		fmt.Println("暂无端口转发配置")
		return nil
	}

	// 选择要删除的端口转发
	var items []string
	for _, pf := range pfs {
		item := fmt.Sprintf("%s:%d -> %s:%d (%s)",
			pf.LocalHost, pf.LocalPort, pf.RemoteHost, pf.RemotePort, pf.Type)
		if pf.Alias != "" {
			item = fmt.Sprintf("[%s] %s", pf.Alias, item)
		}
		items = append(items, item)
	}

	prompt := promptui.Select{
		Label: "选择要删除的端口转发",
		Items: items,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return err
	}

	pf := pfs[index]

	// 确认删除
	confirmPrompt := promptui.Select{
		Label: fmt.Sprintf("确定要删除端口转发 %s:%d -> %s:%d 吗？",
			pf.LocalHost, pf.LocalPort, pf.RemoteHost, pf.RemotePort),
		Items: []string{"是", "否"},
	}

	_, confirmResult, err := confirmPrompt.Run()
	if err != nil {
		return err
	}

	if confirmResult == "是" {
		// 如果端口转发正在运行，先停止它
		if m.forwardManager.IsForwardActive(pf.ID) {
			if err := m.forwardManager.StopPortForward(pf.ID); err != nil {
				fmt.Printf("警告: 停止端口转发失败: %v\n", err)
			}
		}

		if err := m.configManager.DeletePortForward(pf.ID); err != nil {
			return fmt.Errorf("删除端口转发失败: %w", err)
		}
		fmt.Printf("端口转发 %s:%d -> %s:%d 已删除\n",
			pf.LocalHost, pf.LocalPort, pf.RemoteHost, pf.RemotePort)
	}

	return nil
}

// TestPortForward 测试端口转发
func (m *Menu) TestPortForward() error {
	pfs := m.configManager.ListPortForwards()
	if len(pfs) == 0 {
		fmt.Println("暂无端口转发配置")
		return nil
	}

	// 选择要测试的端口转发
	var items []string
	for _, pf := range pfs {
		item := fmt.Sprintf("%s:%d -> %s:%d (%s)",
			pf.LocalHost, pf.LocalPort, pf.RemoteHost, pf.RemotePort, pf.Type)
		if pf.Alias != "" {
			item = fmt.Sprintf("[%s] %s", pf.Alias, item)
		}
		items = append(items, item)
	}

	prompt := promptui.Select{
		Label: "选择要测试的端口转发",
		Items: items,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return err
	}

	pf := pfs[index]

	fmt.Printf("正在测试端口转发配置: %s:%d -> %s:%d ...\n",
		pf.LocalHost, pf.LocalPort, pf.RemoteHost, pf.RemotePort)

	// 测试端口转发配置
	if err := m.forwardManager.TestPortForward(pf); err != nil {
		fmt.Printf("❌ 测试失败: %v\n", err)
		return nil
	}

	fmt.Printf("✅ 端口转发配置测试成功！\n")
	return nil
}
