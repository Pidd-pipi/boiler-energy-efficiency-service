# 工业锅炉能效与运行监测服务（boiler-energy-efficiency-service）

## 一、项目概述

基于 Go 实现的工业锅炉能效 Web 项目，一款后端服务，完成锅炉运行数据采集、能效指标计算、燃烧工况诊断、排污管理提示与运行日报生成。

项目类型：**全栈 Web 应用**（Go 后端服务 + `go:embed` 内嵌前端页面）。

## 二、业务背景与领域规则

工业园区锅炉房有多台工业锅炉（蒸汽/热水），需要监测运行效率：燃料消耗、蒸汽产量、排烟温度、氧含量、给水流量等参数决定锅炉热效率。系统实时采集运行数据，计算热效率与单位蒸汽煤耗，诊断燃烧工况（过量空气系数过大/过小），提示排污与清灰时机。数据异常（如排烟温度突升）要及时告警，防止安全事故。

关键领域规则（这些规则是后续埋 bug 验证跨文件改动的核心约束，必须真实实现）：

1. 锅炉状态机：停炉(stopped) → 启动中(starting) → 运行(running) → 压火(firing_down) → 停炉确认(stopped)；运行中禁止直接进入停炉，必须先压火。
2. 能效计算：热效率 = 有效利用热量 / 输入热量（按正平衡法），单位蒸汽煤耗 = 燃料量 / 蒸汽量；计算输入缺失（缺燃料量或蒸汽量）时拒绝生成能效记录。
3. 燃烧工况诊断：过量空气系数 = 21/(21-氧含量)，低于 1.2 判为缺氧燃烧、高于 1.8 判为过剩空气；诊断结果写入工况记录并给出调整建议。
4. 异常告警：排烟温度相对基线突升超 30℃ 或氧含量异常波动，生成告警并要求司炉工确认；告警未确认超 1 小时自动升级。
5. 排污管理：按累计运行时长与水质硬度提示排污时机，排污执行后记录；超 2 倍排污周期未排污进入「需关注」。
6. 运行日报：每日按锅炉聚合产汽量、煤耗、平均热效率、告警次数，生成日报数据；日报数据用于月度能效考核。

## 三、核心实体（≥3 个，必须贯穿全栈）

每个实体必须贯穿「数据库/存储表 → domain model → repository → service → handler → 前端 API 层 → 前端页面/组件」全链路。

| 实体 | 关键字段 | 业务动作 |
|---|---|---|
| 锅炉 Boiler | id、类型(蒸汽/热水)、额定蒸发量、状态 | 台账、状态查询 |
| 运行数据 RunData | id、锅炉id、燃料量、蒸汽量、给水流量、排烟温度、氧含量、时间戳 | 采集、计算 |
| 能效记录 EfficiencyRecord | id、锅炉id、热效率、单位煤耗、过量空气系数、时间 | 能效计算、诊断 |
| 燃烧工况 CombustionStatus | id、锅炉id、过量空气系数、诊断结论(缺氧/正常/过剩)、建议 | 诊断 |
| 运行告警 RunAlert | id、锅炉id、类型、级别、状态、确认人、说明 | 生成、确认、升级 |
| 排污记录 BlowdownRecord | id、锅炉id、执行时间、时长、执行人 | 排污管理 |

## 四、核心页面与 API

### 前端页面（≥4 个路由，至少 2 个页面共用同一个业务组件）

| 项目 | 说明 |
|---|---|
| / 锅炉房总览 | 锅炉运行状态 + 实时热效率 + 未确认告警 | Boiler、EfficiencyRecord |
| /boilers/{id} 锅炉详情 | 运行趋势 + 能效曲线 + 工况诊断 | RunData、EfficiencyRecord |
| /diagnosis 燃烧工况 | 诊断列表 + 调整建议 | CombustionStatus |
| /alerts 运行告警 | 告警列表 + 确认处置 | RunAlert |
| /blowdown 排污管理 | 排污计划提示 + 执行记录 | BlowdownRecord |

### 后端 REST API（与页面一一对应，命中真实业务链路）

| 项目 | 说明 |
|---|---|
| POST /api/boilers/{id}/run | 运行数据上报（能效计算 + 工况诊断 + 告警判定） |
| POST /api/boilers/{id}/transition | 锅炉状态迁移 |
| GET /api/boilers/{id}/efficiency | 能效记录 |
| GET /api/boilers/{id}/diagnosis | 工况诊断列表 |
| POST /api/alerts/{id}/ack | 确认告警 |
| POST /api/boilers/{id}/blowdown | 排污执行记录 |
| GET /api/boilers/{id}/daily-report | 运行日报 |
| GET /api/overview | 总览聚合 |
| GET /api/healthz | 健康检查 |

## 五、横切关注点（≥2 个）

1. 操作审计日志：状态迁移、告警确认、排污执行全部留痕；触达 handler → service → audit store。
2. 告警升级扫描定时任务：每 10 分钟扫描未确认超时告警并升级；触达 service → store → 告警页。
3. 全局错误处理与统一响应格式。

## 六、共享枚举/常量（≥2 组）

枚举/常量要求前后端各自定义且保持一致，README 中列出所有出现位置。

1. 锅炉状态 BoilerStatus：stopped / starting / running / firing_down。
2. 告警类型 AlertType：flue_temp_spike / oxygen_abnormal / pressure_high / water_low。
3. 诊断结论 DiagnosisResult：under_air / normal / excess_air。

## 七、共享前端组件与 hooks（组件 ≥3 个、hooks ≥2 个）

### 共享组件（放 `web/components/`）

1. BoilerCard：锅炉状态卡片，被总览与详情共用。
2. EfficiencyChart：能效曲线组件，被详情与日报页共用。
3. AlertTable：运行告警表格，被总览与告警页共用。

### 自定义 hooks（放 `web/hooks/`）

1. useBoilers(poll)：锅炉数据轮询，被总览与详情共用。
2. useAlerts(filter)：告警列表，被告警页与总览共用。

## 八、后端中间件（≥2 个）

1. auditLogger：审计日志中间件。
2. errorHandler：统一错误/panic 处理中间件。
3. requestID：trace id 注入中间件。

## 九、技术要求

- 语言：**Go 1.23**（go.mod 声明 `go 1.23`，module 路径 `example.com/boiler-energy-efficiency-service`）
- 运行：`go run .` 默认监听 `8080`，支持 `PORT` 环境变量覆盖
- 存储：SQLite（`modernc.org/sqlite` 纯 Go 驱动，CGO 关闭）或内置内存仓储 + JSON 文件持久化，二选一，必须可重复构建、无外部服务依赖
- 前端：纯原生 HTML/CSS/JS，`go:embed` 内嵌 `web/` 静态资源，禁止引入外部 CDN 依赖（离线可跑）
- 服务入口：`GET /healthz` 返回 200；页面 `GET /` 可访问
- 根目录必须包含 `runtime_smoke.json`：`mode: service` + `start: go run .` + `ready_url: /healthz`；`project_intro` 一句话简介必须包含项目类型（如「基于 Go 实现的XXX Web 项目，一款后端服务，完成……」）
- 根目录必须包含 `README.md`：项目说明、目录结构、运行与测试命令、环境变量说明
- 构建：`go build ./...` 与 `go test ./...` 必须全部通过（基线干净、无 bug）

## 十、文件结构强制清单（规模目标：≥2000 行 Go 功能代码、≥20 个 `.go` 文件）

```
backend/
├── go.mod
├── main.go
├── config/
│   └── config.go            # 能效公式系数、工况阈值、告警阈值
├── domain/
│   ├── boiler.go            # 锅炉实体 + 状态机
│   ├── rundata.go           # 运行数据
│   ├── efficiency.go        # 能效计算
│   ├── combustion.go        # 燃烧工况诊断
│   ├── alert.go             # 运行告警
│   └── blowdown.go          # 排污管理
├── store/
│   ├── boiler_store.go
│   ├── rundata_store.go
│   ├── efficiency_store.go
│   ├── combustion_store.go
│   ├── alert_store.go
│   ├── blowdown_store.go
│   └── audit_store.go
├── service/
│   ├── ingest_service.go    # 采集 + 能效 + 诊断 + 告警
│   ├── efficiency_service.go
│   ├── diagnosis_service.go # 工况诊断
│   ├── alert_service.go     # 确认/升级
│   ├── blowdown_service.go
│   ├── report_service.go    # 运行日报
│   ├── sweeper.go           # 告警升级扫描
│   └── audit_service.go
├── httpapi/
│   ├── router.go
│   ├── boiler_handler.go
│   ├── run_handler.go
│   ├── diagnosis_handler.go
│   ├── alert_handler.go
│   ├── blowdown_handler.go
│   ├── report_handler.go
│   └── health_handler.go
├── middleware/
│   ├── audit.go
│   ├── error_handler.go
│   └── request_id.go
└── web/
    ├── index.html
    ├── app.js
    ├── style.css
    ├── components/
    └── hooks/
```

**严禁合并职责到单一文件**：handler、service、repository、domain 必须分层；禁止把所有逻辑塞进 `main.go` 或一个 `handlers.go`。目标规模下限 2000 行 / 20 个 `.go` 文件，实际建议做到 3000 行以上 / 30 个文件以上，保证每个业务模块（实体、状态机、联动、报表）都有独立文件。

## 十一、运行、测试与交付要求

1. `go build ./...` 通过；`go test ./...` 全绿（含各业务模块的单元测试，测试文件不计入规模）。
2. `go run .` 后 `GET /healthz` 返回 200，前端页面 `GET /` 可打开且核心接口可用。
3. 每个核心业务动作都要有可复现的输入（API 请求/页面操作），方便后续构造缺陷与验证命令。
4. 代码中不得出现任何「故意埋错」「TODO bug」类注释；交付为干净基线。

## 十二、质量红线

1. **天然多文件、多层耦合**：任何一个小改动（如给某状态新增一个合法迁移）都应触达 3-5 个文件（domain + repository + service + handler + 前端组件 + 枚举定义）。
2. 业务规则必须具体、可验证：状态机迁移表、联动逻辑、校验边界、生命周期管理必须真实存在，禁止空壳 CRUD。
3. 本项目用于评测跨文件协同改动能力，禁止做成本目录、对账/财务、库存盘点、电商订单、预约挂号、工单客服、数据可视化报表类业务。
4. 前端页面必须真实消费后端接口，禁止纯静态假页面。

---
*生成说明：本提示词面向 Go 标注数据流水线 2000 行档位，主题已对照禁选题材清单核验。*
