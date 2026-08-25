# BENZHI_README

基于 Go 实现的seed-vigor-gate Web 项目，一款后端服务，面向种质资源保存机构的入库前种子活力鉴定工作台，完整覆盖批次建档、代表性抽样、不可变观测修订、确定性偏差检测与处置、质量复核、证据冻结、适格凭据签发和验真，并由本地 SQLite 保存版本化聚合与连续审计记录。

## 项目说明
- 项目：benzhi-project-f7391c8f-791c-44c3-8d9e-2752702b26bb
- 项目用途：面向种质资源保存机构的入库前种子活力鉴定工作台，完整覆盖批次建档、代表性抽样、不可变观测修订、确定性偏差检测与处置、质量复核、证据冻结、适格凭据签发和验真，并由本地 SQLite 保存版本化聚合与连续审计记录。
- Go 工具链：`golang:1.22`
- 前端工具链：原生 HTML、CSS 和 JavaScript，由 Go 服务直接提供

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run ./cmd/server -selfcheck -addr=127.0.0.1:19081
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-f7391c8f-791c-44c3-8d9e-2752702b26bb-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-f7391c8f-791c-44c3-8d9e-2752702b26bb-arm64 linux/arm64
docker run -it benzhi-project-f7391c8f-791c-44c3-8d9e-2752702b26bb-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/server -selfcheck -addr=127.0.0.1:19081`
