package config

import "time"

// AuthType 认证类型
type AuthType string

const (
	AuthTypePassword    AuthType = "password"    // 密码认证
	AuthTypeKey         AuthType = "key"         // 密钥认证
	AuthTypeCredential  AuthType = "credential"  // 登录凭证
	AuthTypeAsk         AuthType = "ask"         // 每次询问
)

// ForwardType 端口转发类型
type ForwardType string

const (
	ForwardTypeLocal  ForwardType = "local"  // 本地端口转发
	ForwardTypeRemote ForwardType = "remote" // 远程端口转发
)

// CredentialType 凭证类型
type CredentialType string

const (
	CredentialTypePassword CredentialType = "password" // 密码凭证
	CredentialTypeKey      CredentialType = "key"      // SSH密钥凭证
)

// CredentialConfig 凭证配置
type CredentialConfig struct {
	ID           string         `yaml:"id"`           // 凭证ID
	Alias        string         `yaml:"alias"`        // 别名
	Username     string         `yaml:"username"`     // 用户名
	Type         CredentialType `yaml:"type"`         // 凭证类型
	Password     string         `yaml:"password"`     // 密码（如果是密码类型）
	KeyPath      string         `yaml:"key_path"`     // 密钥文件路径（如果是密钥类型）
	KeyContent   string         `yaml:"key_content"`  // 密钥内容（如果是密钥类型）
	KeyPassphrase string        `yaml:"key_passphrase"` // 密钥密码
	Description  string         `yaml:"description"`  // 描述
	CreatedAt    time.Time      `yaml:"created_at"`   // 创建时间
	UpdatedAt    time.Time      `yaml:"updated_at"`   // 更新时间
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Type     string `yaml:"type"`     // 代理类型 (http, socks5)
	Host     string `yaml:"host"`     // 代理主机
	Port     int    `yaml:"port"`     // 代理端口
	Username string `yaml:"username"` // 代理用户名
	Password string `yaml:"password"` // 代理密码
}

// ServerConfig 服务器配置
type ServerConfig struct {
	ID            string       `yaml:"id"`            // 服务器ID
	Alias         string       `yaml:"alias"`         // 别名
	Host          string       `yaml:"host"`          // 主机地址
	Port          int          `yaml:"port"`          // SSH端口
	User          string       `yaml:"user"`          // 用户名
	AuthType      AuthType     `yaml:"auth_type"`     // 认证类型
	CredentialID  string       `yaml:"credential_id"` // 引用的凭证ID
	Password      string       `yaml:"password"`      // 密码（如果使用密码认证）
	KeyPath       string       `yaml:"key_path"`      // 密钥文件路径
	KeyPassphrase string       `yaml:"key_passphrase"` // 密钥密码
	StartupScript string       `yaml:"startup_script"` // 启动脚本
	Proxy         *ProxyConfig `yaml:"proxy"`         // 代理配置
	Tags          []string     `yaml:"tags"`          // 标签
	Description   string       `yaml:"description"`   // 描述
	CreatedAt     time.Time    `yaml:"created_at"`    // 创建时间
	UpdatedAt     time.Time    `yaml:"updated_at"`    // 更新时间
}

// PortForwardConfig 端口转发配置
type PortForwardConfig struct {
	ID          string      `yaml:"id"`          // 转发ID
	Alias       string      `yaml:"alias"`       // 别名
	ServerID    string      `yaml:"server_id"`   // 服务器ID
	Type        ForwardType `yaml:"type"`        // 转发类型
	LocalHost   string      `yaml:"local_host"`  // 本地主机
	LocalPort   int         `yaml:"local_port"`  // 本地端口
	RemoteHost  string      `yaml:"remote_host"` // 远程主机
	RemotePort  int         `yaml:"remote_port"` // 远程端口
	Description string      `yaml:"description"` // 描述
	CreatedAt   time.Time   `yaml:"created_at"`  // 创建时间
	UpdatedAt   time.Time   `yaml:"updated_at"`  // 更新时间
}

// Config 主配置
type Config struct {
	ConfigVersion int                          `yaml:"config_version"` // 配置版本
	Servers       map[string]*ServerConfig     `yaml:"servers"`        // 服务器配置
	PortForwards  map[string]*PortForwardConfig `yaml:"port_forwards"`  // 端口转发配置
	Credentials   map[string]*CredentialConfig `yaml:"credentials"`    // 凭证配置
	Settings      *Settings                    `yaml:"settings"`       // 全局设置
}

// Settings 全局设置
type Settings struct {
	ConfigDir       string `yaml:"config_dir"`       // 配置目录
	LogLevel        string `yaml:"log_level"`        // 日志级别
	ConnectTimeout  int    `yaml:"connect_timeout"`  // 连接超时时间（秒）
	DefaultUser     string `yaml:"default_user"`     // 默认用户名
	DefaultPort     int    `yaml:"default_port"`     // 默认端口
	DefaultAuthType string `yaml:"default_auth_type"` // 默认认证类型
}

// NewConfig 创建新的配置实例
func NewConfig() *Config {
	return &Config{
		ConfigVersion: 1,
		Servers:       make(map[string]*ServerConfig),
		PortForwards:  make(map[string]*PortForwardConfig),
		Credentials:   make(map[string]*CredentialConfig),
		Settings: &Settings{
			LogLevel:        "info",
			ConnectTimeout:  30,
			DefaultUser:     "root",
			DefaultPort:     22,
			DefaultAuthType: "ask",
		},
	}
}

// NewServerConfig 创建新的服务器配置
func NewServerConfig(host string) *ServerConfig {
	now := time.Now()
	return &ServerConfig{
		ID:        generateID(),
		Host:      host,
		Port:      22,
		User:      "root",
		AuthType:  AuthTypeAsk,
		Tags:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewPortForwardConfig 创建新的端口转发配置
func NewPortForwardConfig(serverID string) *PortForwardConfig {
	now := time.Now()
	return &PortForwardConfig{
		ID:         generateID(),
		ServerID:   serverID,
		Type:       ForwardTypeLocal,
		LocalHost:  "127.0.0.1",
		RemoteHost: "127.0.0.1",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// NewCredentialConfig 创建新的凭证配置
func NewCredentialConfig() *CredentialConfig {
	now := time.Now()
	return &CredentialConfig{
		ID:        generateID(),
		Type:      CredentialTypePassword,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// generateID 生成唯一ID
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

// randomString 生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
} 