# seed-vigor-gate

`seed-vigor-gate` 是面向种质资源保存机构的入库前活力鉴定工作台。接收员可建立种子批次并确认代表性抽样，试验员逐日提交发芽观测和偏差处置，质量复核员批准后冻结证据包并签发不可变的入库适格凭据。浏览器工作台和 JSON HTTP API 由同一个 Go 服务提供，所有数据保存在本地 SQLite，不依赖外部系统或 Node 构建链。

## 构建、运行与测试

标准构建：

```bash
go build ./cmd/server
```

标准运行：

```bash
go run ./cmd/server
```

服务默认仅监听 `127.0.0.1:19081`，工作台地址为 `http://127.0.0.1:19081/`，默认数据库为 `./data/seed-vigor-gate.db`。可以显式指定回环地址和数据库：

```bash
go run ./cmd/server -addr=127.0.0.1:19181 -db=./data/seed-vigor-gate.db
```

未传入 `-addr` 时，也可设置 `PORT` 为端口号，服务会绑定 `127.0.0.1:<PORT>`。程序拒绝 `0.0.0.0` 和其他非回环监听地址。

运行全部测试：

```bash
go test ./...
```

运行真实 HTTP 主流程自检：

```bash
go run ./cmd/server -selfcheck -addr=127.0.0.1:19081
```

`-selfcheck` 会在配置的真实监听地址启动服务，使用临时 SQLite 数据库依次完成建档、抽样、四组观测、活力分析、批准、证据冻结、凭据签发和凭据验真，并在 20 秒边界内关闭服务后自行退出。

## 业务状态流转

批次遵循以下受控状态：

1. `draft`：已建档，等待确认抽样方案。
2. `sampling_confirmed`：抽样方案已确认，可以录入观测。
3. `observing`：观测、分析或偏差处置进行中。自动分析发现偏差时保持此状态。
4. `pending_review`：最新分析完整且没有检测项，等待质量复核。
5. `returned`：复核员带理由退回；补录观测后回到 `observing`，旧修订和旧复核决定继续保留。
6. `approved`：复核员批准鉴定结论，下一步只能冻结证据。
7. `frozen`：抽样方案、全部观测修订、采用的分析、偏差处置和复核决定已形成 SHA-256 摘要的只读证据包。
8. `credential_issued`：已从冻结包签发唯一入库适格凭据，业务数据不再允许修改。

每个写命令都要求 `expectedVersion` 和至少 8 位的 `idempotencyKey`。版本不匹配返回 `VERSION_CONFLICT`；相同幂等键和命令会复用原结果，不会重复追加证据或审计事件。抽样确认会锁定按名称稳定排序的单元配额、重复组分配、抽样占比和剩余留样量。观测以“试验 + 重复组 + 观测日 + 修订号”追加保存，一次可原子提交最多 20 行；正式提交、草稿和后续修订均不覆盖历史记录。

## 活力分析与偏差

分析模块以确认的抽样方案和每组最新正式观测为输入，确定性计算有效样本数、发芽率和组间差异，并检测：

- 重复组缺测或终判日不足；
- 四类计数不守恒；
- 培养温度超出抽样方案范围；
- 污染率超过协议阈值；
- 重复组发芽率差异超限；
- 总发芽率低于协议阈值。
- 累计发芽数跨日倒退或未发芽数反向增加；
- 已开始观测的重复组存在中间观测日缺口；
- 连续多个观测日温度越界。

分析只采用每个重复组、观测日的最新正式修订，草稿和同日旧修订不参与计算；逐组轨迹保存发芽率、相邻日增量和平台期长度。每个检测结果都会形成包含规则代码、严重度和前后证据引用的偏差。严重偏差的补充试验由系统在当前批次内创建，一对一关联原偏差，使用同一观测入口保存独立修订；证据齐全且独立分析通过后才关闭原偏差。

## 主要 HTTP 路由

- `GET /`、`GET /workbench`：响应式浏览器工作台。
- `POST /api/cases`、`GET /api/cases`、`GET /api/cases/{caseId}`：批次建档、待办列表和工作台详情。
- `POST /api/cases/{caseId}/sampling-plan/confirm`：确认抽样方案。
- `POST /api/cases/{caseId}/observations`：原子保存至多 20 行草稿或正式观测；行可携带系统生成的 `supplementalTrialId`。
- `POST /api/cases/{caseId}/analysis`：执行原试验分析，或携带 `supplementalTrialId` 执行补充试验独立分析。
- `POST /api/cases/{caseId}/deviations/{deviationId}/resolve`：提交偏差说明、纠正措施，并以 `startSupplementalTrial` 请求系统创建补充试验。
- `POST /api/cases/{caseId}/review`：带理由退回或批准。
- `POST /api/cases/{caseId}/freeze`：冻结带摘要哈希的证据包。
- `POST /api/cases/{caseId}/credential`：签发不可变适格凭据。
- `GET /api/credentials/{credentialNo}`、`GET /api/credentials/{credentialNo}/verify`：按编号只读验真，返回存储摘要、实时重算摘要、关系检查及审计顺序检查。
- `GET /api/cases/{caseId}/timeline`：查看连续审计轨迹。
- `GET /healthz`、`GET /readyz`：健康与就绪检查。

写请求必须使用 `Content-Type: application/json`，请求体上限为 1 MiB，未知 JSON 字段会被拒绝。错误响应包含稳定 `code`、中文 `message`、可选 `field` 和 `requestId`。

## 持久化与不可变性

SQLite 使用单连接单写事务，聚合条件版本更新、批量不可变证据追加、幂等结果和连续审计序号在同一事务提交。启动时执行 `schemaVersion` 迁移和 `PRAGMA quick_check`。原试验及补充试验的观测修订和分析快照、`evidence_bundles` 与 `credentials` 表均受不可变触发器保护；冻结包和凭据只能读取。凭据验真从冻结包内容按冻结时的规范字段和稳定排序规则重新计算 SHA-256，不会修复数据、重签凭据或追加审计。
