#!/bin/bash

# GoTSSH 测试运行脚本
# 用于运行所有测试，包括单元测试、集成测试和性能测试

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_msg() {
    local color=$1
    local msg=$2
    echo -e "${color}${msg}${NC}"
}

# 打印标题
print_title() {
    print_msg "$BLUE" "=========================================="
    print_msg "$BLUE" "$1"
    print_msg "$BLUE" "=========================================="
}

# 运行命令并检查结果
run_and_check() {
    local cmd="$1"
    local desc="$2"
    
    print_msg "$YELLOW" "运行: $desc"
    echo "执行命令: $cmd"
    
    if eval "$cmd"; then
        print_msg "$GREEN" "✅ $desc 成功"
        return 0
    else
        print_msg "$RED" "❌ $desc 失败"
        return 1
    fi
}

# 检查Go环境
check_go_env() {
    print_title "检查Go环境"
    
    if ! command -v go &> /dev/null; then
        print_msg "$RED" "错误: Go未安装或不在PATH中"
        exit 1
    fi
    
    GO_VERSION=$(go version)
    print_msg "$GREEN" "Go版本: $GO_VERSION"
    
    # 检查Go模块
    if [ ! -f "go.mod" ]; then
        print_msg "$RED" "错误: 未找到go.mod文件"
        exit 1
    fi
    
    print_msg "$GREEN" "Go环境检查通过"
}

# 下载依赖
download_deps() {
    print_title "下载依赖"
    run_and_check "go mod download" "下载Go模块依赖"
    run_and_check "go mod tidy" "整理Go模块依赖"
}

# 代码格式化检查
check_format() {
    print_title "代码格式化检查"
    
    # 检查代码格式
    UNFORMATTED=$(gofmt -l .)
    if [ -n "$UNFORMATTED" ]; then
        print_msg "$RED" "以下文件需要格式化:"
        echo "$UNFORMATTED"
        print_msg "$YELLOW" "运行 'gofmt -w .' 来格式化代码"
        return 1
    else
        print_msg "$GREEN" "所有代码格式正确"
    fi
}

# 代码静态检查
static_check() {
    print_title "静态代码检查"
    
    # 使用go vet检查
    run_and_check "go vet ./..." "Go vet静态检查"
    
    # 如果安装了golint，也运行它
    if command -v golint &> /dev/null; then
        run_and_check "golint ./..." "Golint代码风格检查"
    else
        print_msg "$YELLOW" "Golint未安装，跳过代码风格检查"
    fi
}

# 运行单元测试
run_unit_tests() {
    print_title "运行单元测试"
    
    # 运行所有包的单元测试
    local packages=(
        "./cmd"
        "./internal/config"
        "./internal/ssh"
        "./internal/forward"
        "./internal/ui"
    )
    
    for package in "${packages[@]}"; do
        if [ -d "$package" ]; then
            print_msg "$YELLOW" "测试包: $package"
            run_and_check "go test -v $package" "单元测试 $package"
        else
            print_msg "$YELLOW" "跳过不存在的包: $package"
        fi
    done
}

# 运行集成测试
run_integration_tests() {
    print_title "运行集成测试"
    
    if [ -d "./tests" ]; then
        run_and_check "go test -v ./tests" "集成测试"
    else
        print_msg "$YELLOW" "未找到集成测试目录，跳过集成测试"
    fi
}

# 运行主程序测试
run_main_tests() {
    print_title "运行主程序测试"
    
    if [ -f "./main_test.go" ]; then
        run_and_check "go test -v ." "主程序测试"
    else
        print_msg "$YELLOW" "未找到主程序测试文件，跳过主程序测试"
    fi
}

# 运行性能测试
run_benchmark_tests() {
    print_title "运行性能测试"
    
    print_msg "$YELLOW" "运行所有包的性能测试"
    run_and_check "go test -bench=. -benchmem ./..." "性能测试"
}

# 运行覆盖率测试
run_coverage_tests() {
    print_title "代码覆盖率测试"
    
    print_msg "$YELLOW" "生成覆盖率报告"
    run_and_check "go test -coverprofile=coverage.out ./..." "生成覆盖率文件"
    
    if [ -f "coverage.out" ]; then
        print_msg "$YELLOW" "显示覆盖率报告"
        go tool cover -func=coverage.out
        
        # 生成HTML覆盖率报告
        print_msg "$YELLOW" "生成HTML覆盖率报告"
        go tool cover -html=coverage.out -o coverage.html
        print_msg "$GREEN" "HTML覆盖率报告已生成: coverage.html"
    fi
}

# 运行竞态条件检测
run_race_tests() {
    print_title "竞态条件检测"
    
    print_msg "$YELLOW" "运行竞态条件检测"
    run_and_check "go test -race ./..." "竞态条件检测"
}

# 构建测试
build_test() {
    print_title "构建测试"
    
    print_msg "$YELLOW" "测试构建可执行文件"
    run_and_check "go build -o gotssh_test ." "构建可执行文件"
    
    if [ -f "gotssh_test" ]; then
        print_msg "$GREEN" "构建成功，清理测试文件"
        rm -f gotssh_test
    fi
}

# 显示测试统计
show_test_stats() {
    print_title "测试统计"
    
    echo "测试文件统计:"
    find . -name "*_test.go" -type f | wc -l | xargs echo "测试文件数量:"
    find . -name "*_test.go" -type f | xargs wc -l | tail -1 | awk '{print "测试代码行数: " $1}'
    
    echo ""
    echo "源代码文件统计:"
    find . -name "*.go" -not -name "*_test.go" -type f | wc -l | xargs echo "源文件数量:"
    find . -name "*.go" -not -name "*_test.go" -type f | xargs wc -l | tail -1 | awk '{print "源代码行数: " $1}'
}

# 清理测试文件
cleanup() {
    print_title "清理测试文件"
    
    print_msg "$YELLOW" "清理临时文件"
    rm -f coverage.out coverage.html
    rm -f gotssh_test
    
    print_msg "$GREEN" "清理完成"
}

# 显示帮助信息
show_help() {
    echo "GoTSSH 测试脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  all          运行所有测试 (默认)"
    echo "  unit         只运行单元测试"
    echo "  integration  只运行集成测试"
    echo "  benchmark    只运行性能测试"
    echo "  coverage     只运行覆盖率测试"
    echo "  race         只运行竞态条件检测"
    echo "  build        只运行构建测试"
    echo "  format       只检查代码格式"
    echo "  static       只运行静态检查"
    echo "  stats        显示测试统计信息"
    echo "  clean        清理测试文件"
    echo "  help         显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0           # 运行所有测试"
    echo "  $0 unit      # 只运行单元测试"
    echo "  $0 coverage  # 只运行覆盖率测试"
}

# 主函数
main() {
    local option=${1:-all}
    
    case $option in
        "all")
            check_go_env
            download_deps
            check_format
            static_check
            run_unit_tests
            run_integration_tests
            run_main_tests
            build_test
            show_test_stats
            print_msg "$GREEN" "所有测试完成!"
            ;;
        "unit")
            check_go_env
            run_unit_tests
            ;;
        "integration")
            check_go_env
            run_integration_tests
            ;;
        "benchmark")
            check_go_env
            run_benchmark_tests
            ;;
        "coverage")
            check_go_env
            run_coverage_tests
            ;;
        "race")
            check_go_env
            run_race_tests
            ;;
        "build")
            check_go_env
            build_test
            ;;
        "format")
            check_format
            ;;
        "static")
            check_go_env
            static_check
            ;;
        "stats")
            show_test_stats
            ;;
        "clean")
            cleanup
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            print_msg "$RED" "未知选项: $option"
            show_help
            exit 1
            ;;
    esac
}

# 捕获Ctrl+C信号
trap cleanup EXIT

# 运行主函数
main "$@" 