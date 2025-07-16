#!/bin/bash

# GotSSH 发布脚本
# 发布到 Homebrew, Scoop 等平台

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查必要的工具
check_requirements() {
    echo -e "${BLUE}🔍 检查必要工具...${NC}"
    
    if ! command -v git &> /dev/null; then
        echo -e "${RED}❌ Git 未安装${NC}"
        exit 1
    fi
    
    if ! command -v gh &> /dev/null; then
        echo -e "${YELLOW}⚠️  GitHub CLI (gh) 未安装，将跳过自动发布到 GitHub${NC}"
        GITHUB_CLI_AVAILABLE=false
    else
        GITHUB_CLI_AVAILABLE=true
    fi
    
    if ! command -v shasum &> /dev/null; then
        echo -e "${RED}❌ shasum 未安装${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ 工具检查完成${NC}"
}

# 获取版本信息
get_version() {
    if [ -z "$1" ]; then
        echo -e "${RED}❌ 请提供版本号${NC}"
        echo "使用方法: $0 <version>"
        echo "例如: $0 v1.0.0"
        exit 1
    fi
    
    VERSION=$1
    # 移除 v 前缀（如果有）
    VERSION_WITHOUT_V=${VERSION#v}
    
    echo -e "${BLUE}📦 发布版本: ${VERSION}${NC}"
}

# 编译所有平台
build_all_platforms() {
    echo -e "${BLUE}🔨 开始编译所有平台...${NC}"
    
    # 清理之前的构建
    rm -rf build releases
    mkdir -p build releases
    
    # 定义平台和架构
    PLATFORMS=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64" "windows/arm64")
    
    for platform in "${PLATFORMS[@]}"; do
        platform_split=(${platform//\// })
        GOOS=${platform_split[0]}
        GOARCH=${platform_split[1]}
        output_name="gotssh-${GOOS}-${GOARCH}"
        
        if [ $GOOS = "windows" ]; then
            output_name+='.exe'
        fi
        
        echo -e "${YELLOW}📦 编译 $GOOS/$GOARCH...${NC}"
        env GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w -X main.version=${VERSION}" -o build/$output_name .
        
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✅ 编译成功: $output_name${NC}"
            
            # 创建压缩包
            cd build
            if [ $GOOS = "windows" ]; then
                zip "../releases/gotssh-${VERSION_WITHOUT_V}-${GOOS}-${GOARCH}.zip" $output_name
            else
                tar -czf "../releases/gotssh-${VERSION_WITHOUT_V}-${GOOS}-${GOARCH}.tar.gz" $output_name
            fi
            cd ..
        else
            echo -e "${RED}❌ 编译失败: $GOOS/$GOARCH${NC}"
            exit 1
        fi
    done
    
    echo -e "${GREEN}🎉 所有平台编译完成！${NC}"
}

# 生成 SHA256 校验和
generate_checksums() {
    echo -e "${BLUE}🔐 生成 SHA256 校验和...${NC}"
    
    cd releases
    shasum -a 256 *.tar.gz *.zip > SHA256SUMS
    cd ..
    
    echo -e "${GREEN}✅ SHA256 校验和生成完成${NC}"
}

# 生成 Homebrew formula
generate_homebrew_formula() {
    echo -e "${BLUE}🍺 生成 Homebrew formula...${NC}"
    
    # 计算 macOS 版本的 SHA256
    MACOS_AMD64_SHA256=$(shasum -a 256 releases/gotssh-${VERSION_WITHOUT_V}-darwin-amd64.tar.gz | cut -d' ' -f1)
    MACOS_ARM64_SHA256=$(shasum -a 256 releases/gotssh-${VERSION_WITHOUT_V}-darwin-arm64.tar.gz | cut -d' ' -f1)
    
    # 生成 Homebrew formula
    cat > Formula/gotssh.rb << EOF
class Gotssh < Formula
  desc "功能强大的SSH连接和端口转发管理工具"
  homepage "https://github.com/MinatoHikari/gotssh"
  version "${VERSION_WITHOUT_V}"
  
  if Hardware::CPU.arm?
    url "https://github.com/MinatoHikari/gotssh/releases/download/${VERSION}/gotssh-${VERSION_WITHOUT_V}-darwin-arm64.tar.gz"
    sha256 "${MACOS_ARM64_SHA256}"
  else
    url "https://github.com/MinatoHikari/gotssh/releases/download/${VERSION}/gotssh-${VERSION_WITHOUT_V}-darwin-amd64.tar.gz"
    sha256 "${MACOS_AMD64_SHA256}"
  end
  
  depends_on "go" => :build
  
  def install
    bin.install "gotssh-darwin-arm64" => "gotssh" if Hardware::CPU.arm?
    bin.install "gotssh-darwin-amd64" => "gotssh" if Hardware::CPU.intel?
  end
  
  test do
    system "#{bin}/gotssh", "-h"
  end
end
EOF
    
    echo -e "${GREEN}✅ Homebrew formula 生成完成: Formula/gotssh.rb${NC}"
}

# 生成 Scoop manifest
generate_scoop_manifest() {
    echo -e "${BLUE}🥄 生成 Scoop manifest...${NC}"
    
    # 计算 Windows 版本的 SHA256
    WINDOWS_AMD64_SHA256=$(shasum -a 256 releases/gotssh-${VERSION_WITHOUT_V}-windows-amd64.zip | cut -d' ' -f1)
    WINDOWS_ARM64_SHA256=$(shasum -a 256 releases/gotssh-${VERSION_WITHOUT_V}-windows-arm64.zip | cut -d' ' -f1)
    
    # 创建 Scoop manifest 目录
    mkdir -p Scoop
    
    # 生成 Scoop manifest
    cat > Scoop/gotssh.json << EOF
{
    "version": "${VERSION_WITHOUT_V}",
    "description": "功能强大的SSH连接和端口转发管理工具",
    "homepage": "https://github.com/MinatoHikari/gotssh",
    "license": "MIT",
    "architecture": {
        "64bit": {
            "url": "https://github.com/MinatoHikari/gotssh/releases/download/${VERSION}/gotssh-${VERSION_WITHOUT_V}-windows-amd64.zip",
            "hash": "${WINDOWS_AMD64_SHA256}",
            "extract_dir": ".",
            "bin": "gotssh-windows-amd64.exe"
        },
        "arm64": {
            "url": "https://github.com/MinatoHikari/gotssh/releases/download/${VERSION}/gotssh-${VERSION_WITHOUT_V}-windows-arm64.zip",
            "hash": "${WINDOWS_ARM64_SHA256}",
            "extract_dir": ".",
            "bin": "gotssh-windows-arm64.exe"
        }
    },
    "checkver": {
        "github": "https://github.com/MinatoHikari/gotssh"
    },
    "autoupdate": {
        "architecture": {
            "64bit": {
                "url": "https://github.com/MinatoHikari/gotssh/releases/download/v\$version/gotssh-\$version-windows-amd64.zip"
            },
            "arm64": {
                "url": "https://github.com/MinatoHikari/gotssh/releases/download/v\$version/gotssh-\$version-windows-arm64.zip"
            }
        }
    }
}
EOF
    
    echo -e "${GREEN}✅ Scoop manifest 生成完成: Scoop/gotssh.json${NC}"
}

# 生成 AUR PKGBUILD
generate_aur_pkgbuild() {
    echo -e "${BLUE}🐧 生成 AUR PKGBUILD...${NC}"
    
    # 计算 Linux 版本的 SHA256
    LINUX_AMD64_SHA256=$(shasum -a 256 releases/gotssh-${VERSION_WITHOUT_V}-linux-amd64.tar.gz | cut -d' ' -f1)
    
    # 创建 AUR 目录
    mkdir -p AUR
    
    # 生成 PKGBUILD
    cat > AUR/PKGBUILD << EOF
# Maintainer: Your Name <your.email@example.com>
pkgname=gotssh
pkgver=${VERSION_WITHOUT_V}
pkgrel=1
pkgdesc="功能强大的SSH连接和端口转发管理工具"
arch=('x86_64')
url="https://github.com/MinatoHikari/gotssh"
license=('MIT')
depends=('glibc')
source=("https://github.com/MinatoHikari/gotssh/releases/download/${VERSION}/gotssh-${VERSION_WITHOUT_V}-linux-amd64.tar.gz")
sha256sums=('${LINUX_AMD64_SHA256}')

package() {
    install -Dm755 "\$srcdir/gotssh-linux-amd64" "\$pkgdir/usr/bin/gotssh"
}
EOF
    
    echo -e "${GREEN}✅ AUR PKGBUILD 生成完成: AUR/PKGBUILD${NC}"
}

# 创建 GitHub Release
create_github_release() {
    if [ "$GITHUB_CLI_AVAILABLE" = true ]; then
        echo -e "${BLUE}🚀 创建 GitHub Release...${NC}"
        
        # 检查是否已登录 GitHub CLI
        if ! gh auth status &> /dev/null; then
            echo -e "${RED}❌ 请先登录 GitHub CLI: gh auth login${NC}"
            exit 1
        fi
        
        # 创建 release
        gh release create "${VERSION}" \
            --title "Release ${VERSION}" \
            --notes "Release ${VERSION} of GotSSH" \
            --draft \
            || echo -e "${YELLOW}⚠️  Release 可能已存在${NC}"
            
        # 上传文件
        cd releases
        for file in *.tar.gz *.zip; do
            [ -f "$file" ] && gh release upload "${VERSION}" "$file" || true
        done
        cd ..
        
        echo -e "${GREEN}✅ GitHub Release 创建完成${NC}"
    else
        echo -e "${YELLOW}⚠️  跳过 GitHub Release 创建（GitHub CLI 不可用）${NC}"
    fi
}

# 生成发布说明
generate_release_notes() {
    echo -e "${BLUE}📝 生成发布说明...${NC}"
    
    cat > RELEASE_NOTES.md << EOF
# GotSSH ${VERSION} 发布说明

## 下载

### 二进制文件
- [Linux amd64](https://github.com/MinatoHikari/gotssh/releases/download/${VERSION}/gotssh-${VERSION_WITHOUT_V}-linux-amd64.tar.gz)
- [Linux arm64](https://github.com/MinatoHikari/gotssh/releases/download/${VERSION}/gotssh-${VERSION_WITHOUT_V}-linux-arm64.tar.gz)
- [macOS amd64](https://github.com/MinatoHikari/gotssh/releases/download/${VERSION}/gotssh-${VERSION_WITHOUT_V}-darwin-amd64.tar.gz)
- [macOS arm64](https://github.com/MinatoHikari/gotssh/releases/download/${VERSION}/gotssh-${VERSION_WITHOUT_V}-darwin-arm64.tar.gz)
- [Windows amd64](https://github.com/MinatoHikari/gotssh/releases/download/${VERSION}/gotssh-${VERSION_WITHOUT_V}-windows-amd64.zip)
- [Windows arm64](https://github.com/MinatoHikari/gotssh/releases/download/${VERSION}/gotssh-${VERSION_WITHOUT_V}-windows-arm64.zip)

### 包管理器安装

#### Homebrew (macOS/Linux)
\`\`\`bash
brew tap MinatoHikari/tap
brew install gotssh
\`\`\`

#### Scoop (Windows)
\`\`\`bash
scoop bucket add gotssh https://github.com/MinatoHikari/scoop-bucket
scoop install gotssh
\`\`\`

#### AUR (Arch Linux)
\`\`\`bash
yay -S gotssh
# 或
paru -S gotssh
\`\`\`

## 校验和

请使用以下校验和验证下载的文件：

\`\`\`
$(cat releases/SHA256SUMS)
\`\`\`

## 更新内容

- 请填写具体的更新内容
- 修复的问题
- 新增的功能
- 性能改进

## 安装和使用

请参考 [README.md](https://github.com/MinatoHikari/gotssh/blob/main/README.md) 获取详细的安装和使用说明。
EOF
    
    echo -e "${GREEN}✅ 发布说明生成完成: RELEASE_NOTES.md${NC}"
}

# 显示发布后的操作指南
show_post_release_guide() {
    echo -e "${BLUE}📋 发布后操作指南:${NC}"
    echo ""
    echo -e "${YELLOW}1. 更新 Homebrew Tap:${NC}"
    echo "   - 如果还没有 Homebrew tap，创建一个新的仓库: https://github.com/MinatoHikari/homebrew-tap"
    echo "   - 将 Formula/gotssh.rb 复制到 tap 仓库的 Formula/ 目录"
    echo "   - 提交并推送到 GitHub"
    echo ""
    echo -e "${YELLOW}2. 更新 Scoop Bucket:${NC}"
    echo "   - 如果还没有 Scoop bucket，创建一个新的仓库: https://github.com/MinatoHikari/scoop-bucket"
    echo "   - 将 Scoop/gotssh.json 复制到 bucket 仓库的根目录"
    echo "   - 提交并推送到 GitHub"
    echo ""
    echo -e "${YELLOW}3. 发布到 AUR:${NC}"
    echo "   - 克隆 AUR 仓库: git clone ssh://aur@aur.archlinux.org/gotssh.git"
    echo "   - 将 AUR/PKGBUILD 复制到 AUR 仓库"
    echo "   - 更新 .SRCINFO: makepkg --printsrcinfo > .SRCINFO"
    echo "   - 提交并推送到 AUR"
    echo ""
    echo -e "${YELLOW}4. 更新文档:${NC}"
    echo "   - 将 MinatoHikari 替换为实际的 GitHub 用户名"
    echo "   - 更新 README.md 中的安装说明"
    echo "   - 更新 RELEASE_NOTES.md 中的具体更新内容"
    echo ""
    echo -e "${GREEN}🎉 发布完成！${NC}"
}

# 主函数
main() {
    echo -e "${BLUE}🚀 GotSSH 发布脚本${NC}"
    echo ""
    
    get_version $1
    check_requirements
    build_all_platforms
    generate_checksums
    
    # 创建必要的目录
    mkdir -p Formula Scoop AUR
    
    generate_homebrew_formula
    generate_scoop_manifest
    generate_aur_pkgbuild
    create_github_release
    generate_release_notes
    
    echo ""
    echo -e "${GREEN}✅ 发布准备完成！${NC}"
    echo ""
    
    show_post_release_guide
}

# 运行主函数
main "$@" 