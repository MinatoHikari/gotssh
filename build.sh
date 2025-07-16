#!/bin/bash

# GotSSH 跨平台编译脚本
# 使用方法: ./build.sh [version]

# 定义版本
VERSION=${1:-v1.0.0}

# 定义平台和架构
PLATFORMS=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64" "windows/arm64" "freebsd/amd64" "openbsd/amd64")

# 创建构建目录
mkdir -p build

echo "🚀 开始编译 GotSSH $VERSION"
echo "目标平台: $(printf '%s ' "${PLATFORMS[@]}")"
echo ""

# 编译每个平台
for platform in "${PLATFORMS[@]}"; do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    output_name="gotssh-${GOOS}-${GOARCH}"
    
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi
    
    echo "📦 正在编译 $GOOS/$GOARCH..."
    env GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w" -o build/$output_name .
    
    if [ $? -eq 0 ]; then
        # 获取文件大小
        if [ $GOOS = "darwin" ]; then
            size=$(stat -f%z build/$output_name)
        else
            size=$(stat -c%s build/$output_name 2>/dev/null || echo "unknown")
        fi
        
        if [ "$size" != "unknown" ]; then
            # 转换为 MB
            size_mb=$(echo "scale=2; $size / 1024 / 1024" | bc 2>/dev/null || echo "$(($size / 1024 / 1024))")
            echo "✅ 编译成功: build/$output_name (${size_mb}MB)"
        else
            echo "✅ 编译成功: build/$output_name"
        fi
    else
        echo "❌ 编译失败: $GOOS/$GOARCH"
    fi
done

echo ""
echo "🎉 编译完成！"
echo "📁 查看 build/ 目录获取所有编译的二进制文件"
echo ""
echo "生成的文件:"
ls -la build/ 