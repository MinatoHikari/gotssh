#!/bin/bash

# GotSSH 演示脚本
# 展示如何使用 GotSSH 工具

echo "=== GotSSH 功能演示 ==="
echo

# 构建项目
echo "1. 构建项目..."
go build -o gotssh
echo "✅ 构建完成"
echo

# 显示帮助信息
echo "2. 显示帮助信息..."
./gotssh --help
echo

# 测试连接到不存在的服务器
echo "3. 测试连接到不存在的服务器..."
./gotssh -a nonexistent 2>/dev/null || echo "✅ 错误处理正常"
echo

# 显示各种命令的帮助
echo "4. 显示各种命令的帮助..."
echo "--- 连接命令 ---"
./gotssh connect --help
echo

echo "--- 管理命令 ---"
./gotssh manage --help
echo

echo "--- 端口转发命令 ---"
./gotssh tunnel --help
echo

echo "--- 快速端口转发命令 ---"
./gotssh tunnel-connect --help
echo

echo "--- 凭证管理命令 ---"
./gotssh credential --help
echo

echo "=== 演示完成 ==="
echo
echo "下一步："
echo "1. 运行 './gotssh -o' 进入凭证管理界面（可选）"
echo "2. 运行 './gotssh -m' 进入交互式服务器管理界面"
echo "3. 添加服务器配置（可选择已保存的凭证）"
echo "4. 使用 './gotssh -a <server>' 连接服务器"
echo "5. 使用 './gotssh -t' 管理端口转发"
echo "6. 使用 './gotssh --at <alias>' 快速启动端口转发" 