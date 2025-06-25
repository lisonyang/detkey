# DetKey Makefile
# 用于构建确定性SSH密钥生成器

# 变量定义
BINARY_NAME=detkey

# 默认目标
.PHONY: build
build:
	go build -o $(BINARY_NAME) ./cmd/detkey

# 清理构建产物
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f dist/*

# 安装依赖
.PHONY: deps
	go mod tidy
	go mod download

# 格式化代码
.PHONY: fmt
	go fmt ./...

# 运行测试
.PHONY: test
	go test -v ./...

# 静态分析
.PHONY: vet
	go vet ./...

# 完整检查
.PHONY: check
	check: fmt vet test

# 安装到本地系统
.PHONY: install
install: build
	cp $(BINARY_NAME) /usr/local/bin/

# 卸载
.PHONY: uninstall
	rm -f /usr/local/bin/$(BINARY_NAME)

# 显示帮助
.PHONY: help
	@echo "DetKey 构建工具"
	@echo ""
	@echo "可用命令:"
	@echo "  build        构建本地平台的可执行文件"
	@echo "  clean        清理构建产物"
	@echo "  deps         安装/更新依赖"
	@echo "  fmt          格式化代码"
	@echo "  test         运行测试"
	@echo "  vet          静态分析"
	@echo "  check        运行所有检查 (fmt + vet + test)"
	@echo "  install      安装到系统路径"
	@echo "  uninstall    从系统路径卸载"
	@echo "  help         显示此帮助信息"
 