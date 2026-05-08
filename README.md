# HTTPBench

轻量级 HTTP 压测工具，支持自定义并发数、持续时间、请求速率，实时输出进度，并统计 P50/P95/P99 延迟分布。

## 功能

- 自定义并发数 `-c`
- 按总请求数 `-n` 或持续时间 `-d` 两种模式执行
- 速率限制 `-rate`
- 支持 GET / POST / 自定义方法
- 支持 `-body` / `-body-file`
- 支持多次 `-H` 自定义请求头
- text / JSON 输出
- 每秒实时打印当前 completed / errors / qps
- `-n` 模式显示完成数和百分比，`-d` 模式显示 elapsed / remaining
- Ctrl+C 优雅退出并输出已收集结果
- 自定义 Transport，启用 keep-alive 和连接池调优

## 安装与运行

```bash
go mod tidy
go build -o httpbench .
```

## 使用示例

### 固定请求数

```bash
./httpbench -url http://127.0.0.1:8080 -c 20 -n 1000
```

### 持续时间模式

```bash
./httpbench -url http://127.0.0.1:8080 -c 50 -d 30s -rate 200
```

### POST + JSON Body

```bash
./httpbench \
  -url http://127.0.0.1:8080/api/test \
  -method POST \
  -body '{"ok":true}' \
  -H 'Content-Type: application/json' \
  -c 10 \
  -n 100
```

### 从文件读取 Body

```bash
./httpbench -url http://127.0.0.1:8080 -method POST -body-file payload.json -c 10 -n 100
```

### JSON 输出

```bash
./httpbench -url http://127.0.0.1:8080 -c 20 -n 1000 -o json
```

## 参数

- `-url`：目标地址，必填
- `-method`：HTTP 方法，默认 `GET`
- `-c`：并发数，默认 `50`
- `-n`：总请求数
- `-d`：持续时间，例如 `30s`
- `-rate`：每秒最大请求数，`0` 表示不限速
- `-H`：自定义 Header，可重复指定
- `-body`：直接传入请求体
- `-body-file`：从文件读取请求体
- `-o`：输出格式，`text` 或 `json`
- `-timeout`：单个请求超时，默认 `30s`

## 输出说明

text 输出包含：

- 总请求数
- 成功/失败/错误数
- 总耗时
- QPS
- Bytes
- 平均延迟、最小/最大延迟
- P50 / P95 / P99
- 状态码分布
- 错误聚合

## 架构

### 1. Worker Pool + context

每个 worker 循环发送请求，通过 `context` 统一控制退出：

- `-n` 模式下用原子计数分配请求配额
- `-d` 模式下使用 `context.WithTimeout`
- Ctrl+C 通过 `signal.NotifyContext` 触发取消

### 2. 结果收集

每次请求生成一个 `Result`，由 collector 汇总成最终 `Stats`：

- 状态码统计
- 错误聚合
- 字节数统计
- 延迟切片排序后计算百分位

### 3. 连接复用

基于 `net/http.Transport` 调整：

- `MaxIdleConns`
- `MaxIdleConnsPerHost`
- `MaxConnsPerHost`
- Keep-Alive

并在每次请求后完整 drain response body，确保连接可复用。

## 测试

```bash
go test ./...
```
