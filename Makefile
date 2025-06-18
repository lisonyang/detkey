# DetKey Makefile
# 用于构建确定性SSH密钥生成器

# 变量定义
BINARY_NAME=detkey
GO_FILES=$(shell find . -name "*.go" -type f)
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

# 默认目标
.PHONY: build
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# 清理构建产物
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f *.exe

# 安装依赖
.PHONY: deps
deps:
	go mod tidy
	go mod download

# 格式化代码
.PHONY: fmt
fmt:
	go fmt ./...

# 运行测试
.PHONY: test
test:
	go test -v ./...

# 静态分析
.PHONY: vet
vet:
	go vet ./...

# 完整检查
.PHONY: check
check: fmt vet test

# 跨平台编译
.PHONY: build-all
build-all: build-linux build-darwin build-windows

.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 .

.PHONY: build-darwin
build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .

.PHONY: build-windows
build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe .

# 安装到本地系统
.PHONY: install
install: build
	cp $(BINARY_NAME) /usr/local/bin/

# 卸载
.PHONY: uninstall
uninstall:
	rm -f /usr/local/bin/$(BINARY_NAME)

# 显示帮助
.PHONY: help
help:
	@echo "DetKey 构建工具"
	@echo ""
	@echo "可用命令:"
	@echo "  build        构建本地平台的可执行文件"
	@echo "  build-all    构建所有平台的可执行文件"
	@echo "  build-linux  构建 Linux 平台的可执行文件"
	@echo "  build-darwin 构建 macOS 平台的可执行文件"
	@echo "  build-windows 构建 Windows 平台的可执行文件"
	@echo "  clean        清理构建产物"
	@echo "  deps         安装/更新依赖"
	@echo "  fmt          格式化代码"
	@echo "  test         运行测试"
	@echo "  vet          静态分析"
	@echo "  check        运行所有检查 (fmt + vet + test)"
	@echo "  install      安装到系统路径"
	@echo "  uninstall    从系统路径卸载"
	@echo "  help         显示此帮助信息" 