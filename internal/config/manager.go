package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Manager 配置管理器
type Manager struct {
	configPath string
	config     *Config
}

// NewManager 创建新的配置管理器
func NewManager(configPath string) (*Manager, error) {
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("获取用户主目录失败: %w", err)
		}
		configPath = filepath.Join(homeDir, ".config", "gotssh", "config.yaml")
	}

	manager := &Manager{
		configPath: configPath,
		config:     NewConfig(),
	}

	// 尝试加载现有配置
	if err := manager.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("加载配置失败: %w", err)
		}
		// 配置文件不存在，创建默认配置
		if err := manager.Save(); err != nil {
			return nil, fmt.Errorf("保存默认配置失败: %w", err)
		}
	}

	return manager, nil
}

// Load 从文件加载配置
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	config := NewConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	m.config = config
	return nil
}

// Save 保存配置到文件
func (m *Manager) Save() error {
	// 确保配置目录存在
	configDir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 更新配置文件的设置
	if m.config.Settings.ConfigDir == "" {
		m.config.Settings.ConfigDir = configDir
	}

	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0600); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// GetConfig 获取配置
func (m *Manager) GetConfig() *Config {
	return m.config
}

// AddServer 添加服务器配置
func (m *Manager) AddServer(server *ServerConfig) error {
	if server.ID == "" {
		server.ID = generateID()
	}

	// 检查是否已存在相同的主机+端口+用户组合
	for _, existing := range m.config.Servers {
		if existing.Host == server.Host && existing.Port == server.Port && existing.User == server.User {
			return fmt.Errorf("服务器 %s@%s:%d 已存在", server.User, server.Host, server.Port)
		}
	}

	// 检查别名是否唯一
	if server.Alias != "" {
		for _, existing := range m.config.Servers {
			if existing.Alias == server.Alias {
				return fmt.Errorf("别名 '%s' 已存在", server.Alias)
			}
		}
	}

	now := time.Now()
	server.CreatedAt = now
	server.UpdatedAt = now

	m.config.Servers[server.ID] = server
	return m.Save()
}

// UpdateServer 更新服务器配置
func (m *Manager) UpdateServer(serverID string, server *ServerConfig) error {
	if _, exists := m.config.Servers[serverID]; !exists {
		return fmt.Errorf("服务器 %s 不存在", serverID)
	}

	// 检查别名是否唯一（排除当前服务器）
	if server.Alias != "" {
		for id, existing := range m.config.Servers {
			if id != serverID && existing.Alias == server.Alias {
				return fmt.Errorf("别名 '%s' 已存在", server.Alias)
			}
		}
	}

	server.ID = serverID
	server.UpdatedAt = time.Now()
	
	m.config.Servers[serverID] = server
	return m.Save()
}

// DeleteServer 删除服务器配置
func (m *Manager) DeleteServer(serverID string) error {
	if _, exists := m.config.Servers[serverID]; !exists {
		return fmt.Errorf("服务器 %s 不存在", serverID)
	}

	// 删除相关的端口转发配置
	for id, pf := range m.config.PortForwards {
		if pf.ServerID == serverID {
			delete(m.config.PortForwards, id)
		}
	}

	delete(m.config.Servers, serverID)
	return m.Save()
}

// GetServer 获取服务器配置
func (m *Manager) GetServer(serverID string) (*ServerConfig, error) {
	server, exists := m.config.Servers[serverID]
	if !exists {
		return nil, fmt.Errorf("服务器 %s 不存在", serverID)
	}
	return server, nil
}

// GetServerByAlias 根据别名获取服务器配置
func (m *Manager) GetServerByAlias(alias string) (*ServerConfig, error) {
	for _, server := range m.config.Servers {
		if server.Alias == alias {
			return server, nil
		}
	}
	return nil, fmt.Errorf("别名 '%s' 不存在", alias)
}

// GetServerByHost 根据主机地址获取服务器配置
func (m *Manager) GetServerByHost(host string) ([]*ServerConfig, error) {
	var servers []*ServerConfig
	for _, server := range m.config.Servers {
		if server.Host == host {
			servers = append(servers, server)
		}
	}
	
	if len(servers) == 0 {
		return nil, fmt.Errorf("主机 '%s' 不存在", host)
	}
	
	return servers, nil
}

// FindServer 查找服务器配置（支持IP、别名、模糊匹配）
func (m *Manager) FindServer(query string) ([]*ServerConfig, error) {
	var servers []*ServerConfig
	
	// 精确匹配别名
	if server, err := m.GetServerByAlias(query); err == nil {
		return []*ServerConfig{server}, nil
	}
	
	// 精确匹配主机
	if results, err := m.GetServerByHost(query); err == nil {
		return results, nil
	}
	
	// 模糊匹配
	query = strings.ToLower(query)
	for _, server := range m.config.Servers {
		if strings.Contains(strings.ToLower(server.Host), query) ||
			strings.Contains(strings.ToLower(server.Alias), query) ||
			strings.Contains(strings.ToLower(server.Description), query) {
			servers = append(servers, server)
		}
	}
	
	if len(servers) == 0 {
		return nil, fmt.Errorf("未找到匹配的服务器: %s", query)
	}
	
	return servers, nil
}

// ListServers 列出所有服务器
func (m *Manager) ListServers() []*ServerConfig {
	var servers []*ServerConfig
	for _, server := range m.config.Servers {
		servers = append(servers, server)
	}
	return servers
}

// AddPortForward 添加端口转发配置
func (m *Manager) AddPortForward(pf *PortForwardConfig) error {
	if pf.ID == "" {
		pf.ID = generateID()
	}

	// 检查服务器是否存在
	if _, exists := m.config.Servers[pf.ServerID]; !exists {
		return fmt.Errorf("服务器 %s 不存在", pf.ServerID)
	}

	// 检查别名是否唯一
	if pf.Alias != "" {
		for _, existing := range m.config.PortForwards {
			if existing.Alias == pf.Alias {
				return fmt.Errorf("端口转发别名 '%s' 已存在", pf.Alias)
			}
		}
	}

	now := time.Now()
	pf.CreatedAt = now
	pf.UpdatedAt = now

	m.config.PortForwards[pf.ID] = pf
	return m.Save()
}

// UpdatePortForward 更新端口转发配置
func (m *Manager) UpdatePortForward(pfID string, pf *PortForwardConfig) error {
	if _, exists := m.config.PortForwards[pfID]; !exists {
		return fmt.Errorf("端口转发 %s 不存在", pfID)
	}

	// 检查别名是否唯一（排除当前端口转发）
	if pf.Alias != "" {
		for id, existing := range m.config.PortForwards {
			if id != pfID && existing.Alias == pf.Alias {
				return fmt.Errorf("端口转发别名 '%s' 已存在", pf.Alias)
			}
		}
	}

	pf.ID = pfID
	pf.UpdatedAt = time.Now()
	
	m.config.PortForwards[pfID] = pf
	return m.Save()
}

// DeletePortForward 删除端口转发配置
func (m *Manager) DeletePortForward(pfID string) error {
	if _, exists := m.config.PortForwards[pfID]; !exists {
		return fmt.Errorf("端口转发 %s 不存在", pfID)
	}

	delete(m.config.PortForwards, pfID)
	return m.Save()
}

// GetPortForward 获取端口转发配置
func (m *Manager) GetPortForward(pfID string) (*PortForwardConfig, error) {
	pf, exists := m.config.PortForwards[pfID]
	if !exists {
		return nil, fmt.Errorf("端口转发 %s 不存在", pfID)
	}
	return pf, nil
}

// GetPortForwardByAlias 根据别名获取端口转发配置
func (m *Manager) GetPortForwardByAlias(alias string) (*PortForwardConfig, error) {
	for _, pf := range m.config.PortForwards {
		if pf.Alias == alias {
			return pf, nil
		}
	}
	return nil, fmt.Errorf("端口转发别名 '%s' 不存在", alias)
}

// ListPortForwards 列出所有端口转发
func (m *Manager) ListPortForwards() []*PortForwardConfig {
	var pfs []*PortForwardConfig
	for _, pf := range m.config.PortForwards {
		pfs = append(pfs, pf)
	}
	return pfs
}

// ListPortForwardsByServer 列出指定服务器的端口转发
func (m *Manager) ListPortForwardsByServer(serverID string) []*PortForwardConfig {
	var pfs []*PortForwardConfig
	for _, pf := range m.config.PortForwards {
		if pf.ServerID == serverID {
			pfs = append(pfs, pf)
		}
	}
	return pfs
}

// AddCredential 添加凭证配置
func (m *Manager) AddCredential(cred *CredentialConfig) error {
	if cred.ID == "" {
		cred.ID = generateID()
	}

	// 检查别名是否唯一
	if cred.Alias != "" {
		for _, existing := range m.config.Credentials {
			if existing.Alias == cred.Alias {
				return fmt.Errorf("凭证别名 '%s' 已存在", cred.Alias)
			}
		}
	}

	now := time.Now()
	cred.CreatedAt = now
	cred.UpdatedAt = now

	m.config.Credentials[cred.ID] = cred
	return m.Save()
}

// UpdateCredential 更新凭证配置
func (m *Manager) UpdateCredential(credID string, cred *CredentialConfig) error {
	if _, exists := m.config.Credentials[credID]; !exists {
		return fmt.Errorf("凭证 %s 不存在", credID)
	}

	// 检查别名是否唯一（排除当前凭证）
	if cred.Alias != "" {
		for id, existing := range m.config.Credentials {
			if id != credID && existing.Alias == cred.Alias {
				return fmt.Errorf("凭证别名 '%s' 已存在", cred.Alias)
			}
		}
	}

	cred.ID = credID
	cred.UpdatedAt = time.Now()
	
	m.config.Credentials[credID] = cred
	return m.Save()
}

// DeleteCredential 删除凭证配置
func (m *Manager) DeleteCredential(credID string) error {
	if _, exists := m.config.Credentials[credID]; !exists {
		return fmt.Errorf("凭证 %s 不存在", credID)
	}

	// 检查是否有服务器在使用此凭证
	for _, server := range m.config.Servers {
		if server.CredentialID == credID {
			return fmt.Errorf("凭证 '%s' 正在被服务器 '%s' 使用，无法删除", credID, server.Host)
		}
	}

	delete(m.config.Credentials, credID)
	return m.Save()
}

// GetCredential 获取凭证配置
func (m *Manager) GetCredential(credID string) (*CredentialConfig, error) {
	cred, exists := m.config.Credentials[credID]
	if !exists {
		return nil, fmt.Errorf("凭证 %s 不存在", credID)
	}
	return cred, nil
}

// GetCredentialByAlias 根据别名获取凭证配置
func (m *Manager) GetCredentialByAlias(alias string) (*CredentialConfig, error) {
	for _, cred := range m.config.Credentials {
		if cred.Alias == alias {
			return cred, nil
		}
	}
	return nil, fmt.Errorf("凭证别名 '%s' 不存在", alias)
}

// ListCredentials 列出所有凭证
func (m *Manager) ListCredentials() []*CredentialConfig {
	var creds []*CredentialConfig
	for _, cred := range m.config.Credentials {
		creds = append(creds, cred)
	}
	return creds
} 