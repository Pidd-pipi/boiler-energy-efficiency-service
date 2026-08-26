# 工业锅炉能效与运行监测服务（boiler-energy-efficiency-service）

基于 Go 实现的工业锅炉能效 Web 项目，一款后端服务，完成锅炉运行数据采集、能效指标计算（正平衡法）、燃烧工况诊断、排污管理提示与运行日报生成。前端为 `go:embed` 内嵌的原生 HTML/CSS/JS SPA，无外部 CDN，离线可运行。

## 一、功能总览

| 模块 | 说明 |
|---|---|
| 锅炉台账 | 蒸汽/热水锅炉创建、查询、状态机迁移（stopped/starting/running/firing_down） |
| 运行数据采集 | `POST /api/boilers/{id}/run` 上报燃料量、产汽量、排烟温度、氧含量等 |
| 能效计算 | 正平衡法热效率、单位煤耗、过量空气系数；计算输入缺失时拒绝生成能效记录 |
| 燃烧工况诊断 | 过量空气系数 α=21/(21-O₂%)，<1.2 缺氧 / >1.8 过剩 / 其余正常，并给出调整建议 |
| 异常告警 | 排烟温度突升>30℃、氧含量越限/突跳、压力过高、水位过低；超 1 小时未确认自动升级 |
| 排污管理 | 按累计运行时长与水质硬度提示排污时机，超 2 倍周期未排污进入「需关注」 |
| 运行日报 | 每日按锅炉聚合产汽量、煤耗、平均热效率、告警次数，用于月度能效考核 |
| 审计日志 | 状态迁移、告警确认、排污执行、接口请求全程留痕 |

## 二、架构与分层

```
HTTP 请求
  → middleware.RequestID        （生成/透传 X-Request-Id）
  → middleware.SecurityHeaders  （安全头 + API no-store）
  → middleware.AuditLogger      （审计留痕 + slog 结构化访问日志）
  → middleware.ErrorHandler     （panic 恢复 → 500 JSON + 堆栈日志）
  → httpapi 路由/handler        （入参校验、分页、统一响应 envelope）
  → service 业务用例层          （状态机迁移、能效/诊断/告警联动、日报、排污）
  → store 内存仓储              （RWMutex 串行化，getter 返回副本避免数据竞争）
  → JSONPersister               （临时文件 + fsync + rename 原子持久化）
```

- **domain**：领域实体与业务规则，只依赖 `config`，不感知 HTTP/存储。
- **store**：内存仓储 + 快照持久化；单一 `sync.RWMutex` 保护全部共享集合，所有 getter 返回副本。
- **service**：组装领域规则与仓储，承载审计、定时升级扫描、演示数据。
- **httpapi**：只做协议转换与入参校验，不承载业务逻辑。
- **web**：原生 JS SPA，被 `go:embed` 内嵌，运行时由后端同一端口提供。

## 三、目录结构

```
boiler-energy-efficiency-service/
├── go.mod
├── main.go                    # 装配配置/存储/服务/路由；全超时 HTTP Server；优雅关闭
├── runtime_smoke.json         # mode/start/ready_url/project_intro
├── Dockerfile                 # 多阶段构建，非 root，HEALTHCHECK /healthz
├── .dockerignore
├── Makefile                   # build/test/vet/fmt/run/docker-build/docker-run
├── config/
│   ├── config.go              # 业务系数/阈值/日志级别，Validate()
│   └── env.go                 # 环境变量解析（PORT/DATA_DIR/LOG_LEVEL 等）
├── domain/                    # Boiler/RunData/Efficiency/Combustion/Alert/Blowdown/Report/Audit
├── store/                     # 内存仓储 + 原子持久化 + 并发安全 getter
├── service/                   # 采集联动/能效/诊断/告警/排污/日报/审计/扫描
├── httpapi/                   # router + handler + 分页 + 统一响应
├── middleware/                # requestID/security/audit(访问日志)/errorHandler
└── web/                       # index.html/app.js/api.js/enums.js/components/hooks
```

## 四、快速运行

```bash
cd boiler-energy-efficiency-service

# 默认监听 8080，启用内存演示数据与 data/ 持久化
go run .

# 指定端口与数据目录
PORT=19003 DATA_DIR=/tmp/bess go run .
```

访问：

- 页面首页：`http://localhost:19003/`
- 健康检查：`http://localhost:19003/healthz`
- 就绪检查：`http://localhost:19003/readyz`

## 五、Docker 部署

```bash
make docker-build          # 多阶段构建镜像
make docker-run            # 以 PORT=8080 在本地运行容器

# 或手动指定端口
docker run --rm -p 19003:8080 -e PORT=8080 boiler-energy-efficiency-service:latest
```

镜像特点：`golang:1.23-alpine` 构建 → `alpine:3.20` 运行，`CGO_ENABLED=0` 静态编译，非 root 用户，`EXPOSE 8080`，`HEALTHCHECK` 使用 `wget` 探测 `/healthz`，并尊重 `PORT` 环境变量。

## 六、环境变量完整表

| 环境变量 | 默认值 | 说明 |
|---|---|---|
| `PORT` | `8080` | 监听端口（优先级高于 `LISTEN_ADDR`） |
| `LISTEN_ADDR` | `0.0.0.0:8080` | 监听地址；仅在 `PORT` 为空时生效 |
| `LOG_LEVEL` | `info` | 日志级别：`debug` / `info` / `warn` / `error` |
| `DATA_DIR` | `data` | JSON 快照持久化目录；为空则仅内存、不落盘 |
| `SEED_DEMO` | `true` | 仓库为空时是否写入演示锅炉与近 3 天运行数据 |
| `STEAM_ENTHALPY` | `2760.0` | 饱和蒸汽焓 kJ/kg |
| `FEED_WATER_ENTHALPY` | `293.0` | 给水焓 kJ/kg |
| `COAL_LHV` | `20934.0` | 燃料低位发热量 kJ/kg |
| `WATER_SPECIFIC_HEAT` | `4.1868` | 水的比热容 kJ/(kg·℃) |
| `HOT_WATER_SUPPLY_TEMP` | `85.0` | 热水锅炉默认出水温度 ℃ |
| `HOT_WATER_RETURN_TEMP` | `60.0` | 热水锅炉默认回水温度 ℃ |
| `EXCESS_AIR_LOW` | `1.2` | 过量空气系数下限（低于判缺氧） |
| `EXCESS_AIR_HIGH` | `1.8` | 过量空气系数上限（高于判过剩） |
| `FLUE_TEMP_SPIKE_DELTA` | `30.0` | 排烟温度相对基线突升告警阈值 ℃ |
| `FLUE_TEMP_CRITICAL_DELTA` | `60.0` | 排烟温度突升达到严重告警阈值 ℃ |
| `FLUE_TEMP_BASELINE_WINDOW` | `10` | 排烟温度基线滑动窗口条数 |
| `OXYGEN_LOW` | `3.0` | 氧含量下限 % |
| `OXYGEN_HIGH` | `15.0` | 氧含量上限 % |
| `OXYGEN_JUMP_DELTA` | `3.0` | 氧含量相对上一采样点突跳阈值（百分点） |
| `PRESSURE_HIGH` | `1.6` | 蒸汽压力过高阈值 MPa |
| `WATER_LOW` | `40.0` | 水位过低阈值 % |
| `ALERT_ESCALATION_AFTER` | `1h` | 告警未确认超时升级时长 |
| `SWEEP_INTERVAL` | `10m` | 告警升级扫描周期 |
| `BLOWDOWN_BASE_INTERVAL_HOURS` | `48.0` | 基准排污周期（小时） |
| `BLOWDOWN_REFERENCE_HARDNESS` | `2.0` | 基准水质硬度 mmol/L |
| `BLOWDOWN_MAX_INTERVAL_HOURS` | `720.0` | 排污周期上限（小时） |
| `BLOWDOWN_MIN_INTERVAL_HOURS` | `8.0` | 排污周期下限（小时） |
| `BLOWDOWN_MISS_FACTOR` | `2.0` | 超期倍数，超过 2 倍周期未排污进入「需关注」 |
| `SAMPLE_INTERVAL_MINUTES` | `5.0` | 运行数据默认采样周期（分钟） |

所有配置在 `config.Load()` 时解析并在 `Config.Validate()` 中校验；非法值会导致进程启动失败，不会静默回退。

## 七、REST API

统一响应：`{code, message, data, trace_id}`；`code=0` 表示成功。

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/healthz` `/api/healthz` `/readyz` | 健康/就绪检查（200） |
| POST | `/api/boilers` | 创建锅炉 |
| GET | `/api/boilers` | 锅炉列表 |
| GET | `/api/boilers/{id}` | 锅炉详情（含最新能效/未确认告警/排污计划/今日日报） |
| GET | `/api/boilers/{id}/transitions` | 当前状态允许的迁移目标 |
| POST | `/api/boilers/{id}/transition` | 状态迁移 `{"target":"running","operator":"..."}` |
| POST | `/api/boilers/{id}/run` | 运行数据上报（能效+诊断+告警联动） |
| GET | `/api/boilers/{id}/run` | 运行数据列表 |
| GET | `/api/boilers/{id}/efficiency` | 能效记录 |
| GET | `/api/boilers/{id}/diagnosis` | 某锅炉工况诊断列表 |
| GET | `/api/diagnosis` | 全部诊断 + 结论统计 |
| GET | `/api/alerts` | 告警列表（支持 status/boiler_id/type 过滤） |
| POST | `/api/alerts/{id}/ack` | 确认告警 |
| POST | `/api/alerts/{id}/escalate` | 升级告警 |
| POST | `/api/alerts/{id}/resolve` | 处置告警 |
| GET | `/api/boilers/{id}/blowdown` | 排污计划 + 记录 |
| POST | `/api/boilers/{id}/blowdown` | 排污执行 |
| GET | `/api/blowdown` | 全部锅炉排污计划 |
| GET | `/api/boilers/{id}/daily-report` | 单锅炉日报（未生成自动生成） |
| GET | `/api/daily-reports` | 日报列表 |
| POST | `/api/daily-reports/generate` | 为全部锅炉生成日报 |
| GET | `/api/overview` | 总览聚合 |
| GET | `/api/audit-logs` | 审计日志（支持 action/entity_type/entity_id 过滤） |

### 分页约定

所有 list 接口均支持 `limit` 与 `offset` 查询参数，并有默认值与上限：

- `limit`：返回条数，缺省使用各接口默认值，超过上限自动截断到上限。
- `offset`：从“最新一端”向前跳过的条数，缺省为 `0`（即默认返回最新 `limit` 条）。
- `total`：总条数通过响应头 `X-Total-Count` 返回；同时回写 `X-Limit`、`X-Offset`。

示例：

```bash
curl -s -D - "http://localhost:19003/api/alerts?limit=10&offset=0" -o /tmp/alerts.json
# 响应头包含：X-Total-Count: <total>
```

## 八、共享枚举/常量位置

| 枚举 | 后端定义 | 前端定义 |
|---|---|---|
| `BoilerStatus`（stopped/starting/running/firing_down） | `domain/boiler.go` | `web/enums.js` |
| `BoilerType`（steam/hot_water） | `domain/boiler.go` | `web/enums.js` |
| `AlertType`（flue_temp_spike/oxygen_abnormal/pressure_high/water_low） | `domain/alert.go` | `web/enums.js` |
| `AlertLevel`（warning/critical） | `domain/alert.go` | `web/enums.js` |
| `AlertStatus`（open/acknowledged/escalated/resolved） | `domain/alert.go` | `web/enums.js` |
| `DiagnosisResult`（under_air/normal/excess_air） | `domain/combustion.go` | `web/enums.js` |
| `AuditAction` | `domain/audit.go` | `web/enums.js` |

修改任意枚举时，必须同步后端 `domain/*.go`、前端 `web/enums.js` 与 README 本表。

## 九、测试与质量

```bash
make fmt          # gofmt -w .
make vet          # go vet ./...
make test         # go test ./...
make test-race    # go test -race ./...
```

- 所有领域规则（状态机、能效计算、工况诊断、告警、排污、日报）均有单元测试。
- `go test -race ./...` 用于验证 store 并发安全；getter 返回副本，避免共享指针被锁外修改。
- 验证镜像构建命令：

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/boiler-linux-amd64 .
```

## 十、健康检查

| 路径 | 说明 |
|---|---|
| `GET /healthz` | 存活检查，始终返回 200（`{"status":"ok",...}`） |
| `GET /readyz` | 就绪检查，返回 200 |
| `GET /api/healthz` | 与 `/healthz` 等价 |

Docker 镜像 `HEALTHCHECK` 使用 `wget` 探测 `/healthz`。

## 十一、故障排查

| 现象 | 排查方向 |
|---|---|
| 启动失败：`环境变量 XXX 无法解析` | 检查对应环境变量是否为合法数字/时长 |
| 启动失败：`config.Validate` 报错 | 检查阈值配置是否满足约束（如 `EXCESS_AIR_LOW < EXCESS_AIR_HIGH`） |
| 日志出现 `快照文件损坏，将备份后降级为空库启动` | `data/store.json` 已损坏，系统已自动备份为 `data/store.json.bak` 并空库启动 |
| 前端页面打不开，但 `/healthz` 正常 | 确认 SPA 路由回退可用，检查浏览器 Console 与网络请求 |
| 告警未自动升级 | 确认 `SWEEP_INTERVAL`/`ALERT_ESCALATION_AFTER` 配置，查看日志中扫描器是否启动 |
| 无法绑定端口 | 释放被占用的端口，或通过 `PORT` 修改监听端口 |
