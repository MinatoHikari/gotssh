# GotSSH

一个功能强大的SSH命令行工具，支持服务器管理、端口转发管理等功能。

## 功能

- 🔧 **交互式管理界面**: 使用 `-m` 参数进入交互式服务器管理界面
- 🚀 **快速连接**: 使用 `-a` 参数根据IP地址或别名快速连接服务器
- 🔄 **端口转发管理**: 使用 `-t` 参数管理SSH端口转发配置
- ⚡ **快速端口转发**: 使用 `--at` 参数根据别名快速建立端口转发隧道
- 🔐 **凭证管理**: 使用 `-o` 参数交互式管理登录凭证
- 🎯 **指定凭证连接**: 使用 `-a` 和 `-o` 参数组合，直接使用指定凭证连接服务器

## 服务器配置支持

- 代理配置（每个服务器单独配置）
- 端口、用户名、别名设置
- 多种登录方式：密码、密钥、登录凭证、每次询问
- 启动脚本配置
- 支持选择预保存的登录凭证

## 端口转发配置

- 本地端口转发和远程端口转发
- 端口转发别名管理
- 快速隧道建立

## 凭证管理

- 密码凭证：存储用户名和密码
- SSH密钥凭证：存储用户名和SSH私钥（支持文件路径或直接内容）
- 凭证别名管理，方便快速选择
- 添加服务器时可选择已保存的凭证
- 支持与连接命令组合使用，实现快速认证连接

### 凭证与连接组合使用

当同时使用 `-a` 和 `-o` 参数时，系统会：

1. **优先使用指定凭证**: 无论服务器配置中设置了什么认证方式，都会直接使用 `-o` 参数指定的凭证
2. **用户名优先级**: 如果凭证中包含用户名，会覆盖服务器配置或解析出的用户名
3. **支持直接连接**: 即使服务器未保存在配置中，也可以解析连接字符串并使用指定凭证
4. **灵活的格式支持**: 支持 `host`、`user@host`、`host:port`、`user@host:port` 等多种格式

#### 使用示例

```bash
# 使用凭证连接已保存的服务器（支持两种语法）
./gotssh -a myserver -o mycred
./gotssh -a myserver -o=mycred

# 使用凭证直接连接IP地址
./gotssh -a 192.168.1.100 -o mycred

# 使用凭证连接指定用户和端口
./gotssh -a admin@192.168.1.100:2222 -o mycred

# 凭证中的用户名会覆盖连接字符串中的用户名
./gotssh -a root@192.168.1.100 -o mycred  # 实际使用mycred中的用户名

# 注意：支持两种语法形式
# 1. 空格形式：-o credential
# 2. 等号形式：-o=credential
```

## 安装和使用

### 构建

```bash
# 克隆项目
git clone <repository-url>
cd gotssh

# 下载依赖
go mod tidy

# 构建 (当前平台)
go build -o gotssh
```

### 跨平台编译

Go支持交叉编译，可以编译到不同的平台和架构：

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build -o gotssh-linux-amd64

# Linux arm64
GOOS=linux GOARCH=arm64 go build -o gotssh-linux-arm64

# macOS amd64 (Intel)
GOOS=darwin GOARCH=amd64 go build -o gotssh-darwin-amd64

# macOS arm64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o gotssh-darwin-arm64

# Windows amd64
GOOS=windows GOARCH=amd64 go build -o gotssh-windows-amd64.exe

# Windows arm64
GOOS=windows GOARCH=arm64 go build -o gotssh-windows-arm64.exe

# FreeBSD amd64
GOOS=freebsd GOARCH=amd64 go build -o gotssh-freebsd-amd64

# OpenBSD amd64
GOOS=openbsd GOARCH=amd64 go build -o gotssh-openbsd-amd64
```

#### 批量编译脚本

创建一个 `build.sh` 脚本来批量编译多个平台：

```bash
#!/bin/bash

# 定义版本
VERSION=${1:-v1.0.0}

# 定义平台和架构
PLATFORMS=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64" "windows/arm64" "freebsd/amd64" "openbsd/amd64")

# 创建构建目录
mkdir -p build

# 编译每个平台
for platform in "${PLATFORMS[@]}"; do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    output_name="gotssh-${GOOS}-${GOARCH}"
    
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi
    
    echo "Building for $GOOS/$GOARCH..."
    env GOOS=$GOOS GOARCH=$GOARCH go build -o build/$output_name .
    
    if [ $? -eq 0 ]; then
        echo "✅ Successfully built: build/$output_name"
    else
        echo "❌ Failed to build for $GOOS/$GOARCH"
    fi
done

echo "Build complete! Check the build/ directory for binaries."
```

使用方法：
```bash
# 给脚本执行权限
chmod +x build.sh

# 运行批量编译
./build.sh

# 或指定版本
./build.sh v1.0.1
```

### 命令参考

| 参数组合 | 功能 | 示例 |
|---------|------|------|
| `-m` | 进入交互式服务器管理界面 | `./gotssh -m` |
| `-o` | 进入交互式凭证管理界面 | `./gotssh -o` |
| `-a <server>` | 连接到服务器 | `./gotssh -a myserver` |
| `-a <server> -o <cred>` | 使用指定凭证连接服务器 | `./gotssh -a myserver -o mycred` |
| `-t` | 管理端口转发 | `./gotssh -t` |
| `--at <alias>` | 快速启动端口转发 | `./gotssh --at tunnel1` |

### 使用方法

#### 1. 交互式管理界面
```bash
./gotssh -m
# 或
./gotssh manage
```

#### 2. 连接服务器
```bash
# 普通连接
./gotssh -a <ip/alias>
# 或
./gotssh connect <ip/alias>

# 使用指定凭证连接
./gotssh -a <ip/alias> -o <credential_alias>
```

#### 3. 端口转发管理
```bash
./gotssh -t
# 或
./gotssh tunnel
```

#### 4. 快速端口转发
```bash
./gotssh --at <alias>
# 或
./gotssh tunnel-connect <alias>
```

#### 5. 凭证管理
```bash
# 进入凭证管理界面
./gotssh -o
# 或
./gotssh credential

# 使用指定凭证连接服务器（与-a组合使用）
./gotssh -a server1 -o mycred
```

### 配置文件

配置文件默认保存在 `~/.config/gotssh/config.yaml`

### 示例使用流程

1. 首先管理登录凭证（可选）：
   ```bash
   ./gotssh -o
   ```

2. 进入管理界面添加服务器：
   ```bash
   ./gotssh -m
   ```
   
3. 添加服务器后，可以快速连接：
   ```bash
   # 普通连接
   ./gotssh -a myserver
   
   # 使用指定凭证连接
   ./gotssh -a myserver -o mycred
   
   # 直接连接到IP地址并使用指定凭证
   ./gotssh -a root@192.168.1.100:22 -o mycred
   ```

4. 配置端口转发：
   ```bash
   ./gotssh -t
   ```

5. 使用别名快速启动端口转发：
   ```bash
   ./gotssh --at mysql-tunnel
   ```

## 项目结构

```
gotssh/
├── cmd/                      # 命令行接口
│   ├── root.go              # 根命令
│   ├── manage.go            # 管理命令 (-m)
│   ├── connect.go           # 连接命令 (-a)
│   ├── tunnel.go            # 端口转发管理 (-t)
│   ├── tunnel-connect.go    # 快速端口转发 (--at)
│   └── credential.go        # 凭证管理 (-o)
├── internal/                # 内部实现
│   ├── config/             # 配置管理
│   │   ├── types.go        # 数据结构定义
│   │   └── manager.go      # 配置管理器
│   ├── ssh/                # SSH客户端
│   │   └── client.go       # SSH连接和操作
│   ├── forward/            # 端口转发
│   │   └── manager.go      # 端口转发管理器
│   └── ui/                 # 用户界面
│       ├── menu.go         # 交互式菜单
│       └── credential.go   # 凭证管理界面
├── main.go                 # 主程序入口
├── go.mod                  # Go模块定义
├── go.sum                  # 依赖锁定
├── demo.sh                 # 演示脚本
├── example-config.yaml     # 示例配置
├── LICENSE                 # 许可证
└── README.md              # 项目说明
```

## 特性

- ✅ 交互式服务器管理
- ✅ 多种SSH认证方式支持
- ✅ 登录凭证管理系统
- ✅ 代理服务器支持
- ✅ 端口转发管理
- ✅ 配置文件持久化
- ✅ 别名快速访问
- ✅ 启动脚本支持
- ✅ 直观的命令行界面
- ✅ 完整的编辑功能

## 依赖项

- `github.com/manifoldco/promptui` - 交互式命令行界面
- `github.com/spf13/cobra` - 命令行工具框架
- `golang.org/x/crypto` - SSH协议支持
- `golang.org/x/net` - 网络代理支持
- `golang.org/x/term` - 终端操作
- `gopkg.in/yaml.v3` - YAML配置文件解析

## 高级用法：凭证与连接组合

### 场景1：快速连接已保存的服务器

如果你已经保存了服务器配置和凭证，可以直接使用别名快速连接：

```bash
# 假设你有一个名为 "webserver" 的服务器和一个名为 "mykey" 的SSH密钥凭证
./gotssh -a webserver -o mykey
```

### 场景2：直接连接未保存的服务器

即使服务器没有保存在配置中，也可以直接连接：

```bash
# 使用密码凭证连接
./gotssh -a 192.168.1.100 -o mypassword

# 使用SSH密钥凭证连接
./gotssh -a admin@192.168.1.100:2222 -o mykey

# 连接到非标准端口
./gotssh -a user@example.com:2222 -o mykey
```

### 场景3：覆盖服务器配置的认证方式

如果服务器配置中设置了密码认证，但你想使用SSH密钥：

```bash
# 服务器配置使用密码认证，但强制使用SSH密钥
./gotssh -a myserver -o mykey
```

### 场景4：用户名优先级

凭证中的用户名会覆盖连接字符串中的用户名：

```bash
# 假设 "mykey" 凭证中的用户名是 "admin"
# 即使连接字符串中写的是 "root"，实际会使用 "admin" 用户名
./gotssh -a root@192.168.1.100 -o mykey
```

### 错误处理

如果指定的凭证不存在，系统会提示错误：

```bash
$ ./gotssh -a server1 -o nonexistent
获取凭证失败: credential not found: nonexistent
```

如果服务器地址格式不正确，系统会提示解析错误：

```bash
$ ./gotssh -a invalid:host:format -o mycred
解析服务器地址失败: 无效的主机:端口格式
```

## 发布和分发

### 发布脚本

项目提供了自动化发布脚本，可以一键发布到多个平台：

```bash
# 发布新版本
./release.sh v1.0.0
```

发布脚本会自动：
- 🔨 编译所有平台的二进制文件
- 📦 创建发布包和校验和
- 🍺 生成 Homebrew Formula
- 🥄 生成 Scoop Manifest  
- 🐧 生成 AUR PKGBUILD
- 🚀 创建 GitHub Release

详细的发布流程请参考 [RELEASE_GUIDE.md](RELEASE_GUIDE.md)。

### 包管理器安装

#### Homebrew (macOS/Linux)
```bash
brew tap MinatoHikari/tap
brew install gotssh
```

#### Scoop (Windows)
```bash
scoop bucket add gotssh https://github.com/MinatoHikari/scoop-bucket
scoop install gotssh
```

#### AUR (Arch Linux)
```bash
yay -S gotssh
# 或
paru -S gotssh
```

## 更新日志

### 最新更新：凭证与连接组合功能

**新增功能：**
- ✨ 支持 `-a` 和 `-o` 参数组合使用
- 🎯 可以直接使用指定凭证连接服务器
- 🔧 改进了 `-o` 参数的处理逻辑，支持可选参数

**使用方式：**
```bash
# 单独使用 -o 进入凭证管理界面
./gotssh -o

# 组合使用 -a 和 -o 直接连接（支持两种语法）
./gotssh -a server1 -o mycred
./gotssh -a server1 -o=mycred
./gotssh -a 192.168.1.100 -o mykey
```

**功能特点：**
1. **优先级明确**：凭证中的认证信息会覆盖服务器配置
2. **用户名优先**：凭证中的用户名会覆盖连接字符串中的用户名
3. **灵活连接**：支持已保存服务器和直接IP地址连接
4. **错误处理**：提供清晰的错误提示和使用指导

**语法说明：**
- 支持两种语法：`-o credential` 和 `-o=credential`
- 单独使用 `-o` 进入凭证管理界面
- 必须与 `-a` 参数组合使用才能指定凭证连接