.PHONY: help test build clean run docker-influx benchmark

# 加载 .env 文件（如果存在）
ifneq (,$(wildcard .env))
    include .env
    export
endif

# 默认目标
help:
	@echo "GInflux 测试工具"
	@echo ""
	@echo "可用命令:"
	@echo "  make test             - 运行功能测试"
	@echo "  make benchmark        - 运行性能基准测试"
	@echo "  make build            - 编译测试程序"
	@echo "  make build-benchmark  - 编译基准测试程序"
	@echo "  make run              - 运行编译后的测试程序"
	@echo "  make clean            - 清理编译文件"
	@echo "  make docker-influx - 启动本地 InfluxDB Docker 容器"
	@echo "  make stop-influx   - 停止 InfluxDB Docker 容器"
	@echo "  make logs-influx   - 查看 InfluxDB 日志"
	@echo ""

# 运行测试
test:
	@echo "运行 GInflux 测试..."
	@if [ -z "$$INFLUX_TOKEN" ]; then \
		echo "警告: 环境变量未设置，使用默认配置"; \
		echo "建议运行: source ~/.bashrc 或 source ~/.zshrc"; \
	fi
	@go run cmd/test/main.go

# 运行基准测试
benchmark:
	@echo "运行 GInflux 性能基准测试..."
	@go run cmd/benchmark/benchmark.go

# 编译
build:
	@echo "编译测试程序..."
	@go build -o ginflux-test cmd/test/main.go
	@echo "编译完成: ginflux-test"

# 编译基准测试
build-benchmark:
	@echo "编译基准测试程序..."
	@go build -o ginflux-benchmark cmd/benchmark/benchmark.go
	@echo "编译完成: ginflux-benchmark"

# 运行编译后的程序
run: build
	@echo "运行测试..."
	@./ginflux-test

# 清理
clean:
	@echo "清理编译文件..."
	@rm -f ginflux-test ginflux-benchmark
	@echo "清理完成"

# 启动 InfluxDB Docker 容器 (用于本地测试)
docker-influx:
	@echo "启动 InfluxDB 2.x Docker 容器..."
	@docker run -d \
		--name influxdb-test \
		-p 8086:8086 \
		-e DOCKER_INFLUXDB_INIT_MODE=setup \
		-e DOCKER_INFLUXDB_INIT_USERNAME=admin \
		-e DOCKER_INFLUXDB_INIT_PASSWORD=password123 \
		-e DOCKER_INFLUXDB_INIT_ORG=test-org \
		-e DOCKER_INFLUXDB_INIT_BUCKET=test-bucket \
		-e DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=test-token-123456789 \
		influxdb:2.7
	@echo ""
	@echo "InfluxDB 已启动!"
	@echo "URL: http://localhost:8086"
	@echo "用户名: admin"
	@echo "密码: password123"
	@echo "组织: test-org"
	@echo "Bucket: test-bucket"
	@echo "Token: test-token-123456789"
	@echo ""
	@echo "请设置环境变量:"
	@echo "  export INFLUX_URL=http://localhost:8086"
	@echo "  export INFLUX_TOKEN=test-token-123456789"
	@echo "  export INFLUX_ORG=test-org"
	@echo "  export INFLUX_BUCKET=test-bucket"

# 停止 InfluxDB 容器
stop-influx:
	@echo "停止 InfluxDB 容器..."
	@docker stop influxdb-test
	@docker rm influxdb-test
	@echo "InfluxDB 容器已停止并删除"

# 查看 InfluxDB 日志
logs-influx:
	@docker logs -f influxdb-test

# 快速测试 (启动 InfluxDB + 运行测试)
quick-test: docker-influx
	@echo "等待 InfluxDB 启动..."
	@sleep 5
	@INFLUX_URL=http://localhost:8086 \
	 INFLUX_TOKEN=test-token-123456789 \
	 INFLUX_ORG=test-org \
	 INFLUX_BUCKET=test-bucket \
	 go run cmd/test/main.go
