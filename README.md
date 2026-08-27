# blast-permit

`blast-permit` 是面向地下工程爆破技术人员的振动控制审查服务。它把保护对象基线、不可变设计修订、振动预测、风险整改、监测布点复核、证据冻结和点火许可放在同一条可追溯流程中，防止尚未完成技术闭环的方案进入现场。

服务只提供版本化 JSON HTTP API。业务数据保存在本地 SQLite，案卷变更使用 `expectedVersion` 做乐观并发控制，使用 `idempotencyKey` 保证重试不重复创建修订、许可或审计记录。每次成功变更会在同一事务追加带连续序号和链式摘要的审计事件。

## 构建、运行与测试

要求 Go 1.23 或更高版本。

```text
go build ./cmd/blastpermit
go test ./...
go run ./cmd/blastpermit -selfcheck -addr=127.0.0.1:19081
```

普通服务默认监听 `127.0.0.1:19081`，数据库默认写入 `blast-permit.db`：

```text
go run ./cmd/blastpermit
go run ./cmd/blastpermit -addr=127.0.0.1:19181 -db=./data/blast-permit.db
```

也可以把 `PORT` 设置为端口号，服务会绑定到 `127.0.0.1:<PORT>`。`-addr` 必须是明确的回环 IP 和有效端口；服务拒绝 `0.0.0.0`、主机名以及非回环地址。`-selfcheck` 会创建临时 SQLite 数据库，真实监听给定回环地址，通过公开 API 执行完整流程，并在成功或失败后自行关闭和清理。

## 角色与并发协议

变更请求使用以下请求头标识操作者：

- `X-Actor-Role: designer`：建档、登记基线、提交修订和整改。
- `X-Actor-Role: reviewer`：提交监测计划并批准或退回。
- `X-Actor-Role: safety_officer`：冻结证据并签发许可。
- `X-Actor-Name`：必填的操作者姓名。

`expectedVersion` 和 `idempotencyKey` 可以放在 JSON 正文中，也可以分别通过 `Expected-Version`、`Idempotency-Key` 请求头传入；两处同时提供时必须一致。幂等键长度为 8 到 128 个字符。

## API 流程

所有写入使用 `Content-Type: application/json`，请求体上限为 1 MiB，未知 JSON 字段会被拒绝。主要路由如下：

1. `POST /api/v1/cases` 创建 `draft` 案卷。
2. `POST /api/v1/cases/{caseId}/targets` 原子批量登记保护对象；同类型、规范化名称相同的批内或存量对象会返回带项序号、冲突字段和既有对象编号的 `target_conflict`。
3. `GET /api/v1/cases/{caseId}/baseline/precheck` 只读检查三类对象、字段完整性和基线余量分级；`POST /api/v1/cases/{caseId}/baseline/complete` 仅在预检无阻断时进入 `baseline_ready`，并保存确认摘要。
4. `POST /api/v1/cases/{caseId}/revisions` 提交不可变修订并自动评估。响应包含相对父修订的逐字段差异、影响范围、逐目标允许药量、控制目标和当前药量余量；业务参数没有变化的空修订会被拒绝。
5. `POST /api/v1/cases/{caseId}/remediations` 使用 `findingResolutions` 逐项引用当前阻断编号并提供 `handlingNote`。响应和案卷详情保留关闭、遗留、替代与新增阻断关系；仍有阻断时保持 `remediation_required`。
6. `POST /api/v1/cases/{caseId}/reviews` 提交监测计划及 `approve` 或 `reject` 决定。系统校验传感器身份、控制/临界对象覆盖和触发阈值；退回时 `reasons` 必须包含结构化类别、说明、`targetId` 或 `parameter` 及 `requiredChange`。再次批准通过 `reasonResolutions` 逐项确认，所有历史轮次保持不可变。
7. `GET /api/v1/cases/{caseId}/permit/precheck` 只读核对批准状态、修订评估对应关系、获批监测计划、四类组件摘要和审计链；`POST /api/v1/cases/{caseId}/permit` 复用该预检结果冻结证据并签发许可，案卷进入 `frozen` 后拒绝任何业务修改。
8. `GET /api/v1/cases/{caseId}` 查询完整案卷和差异、评估、阻断处置及复核历史，`GET /api/v1/cases/{caseId}/audit` 查询经连续性校验的审计时间线。
9. `GET /api/v1/permits/{permitNumber}/verify` 公开验真，稳定区分 `valid`、`expired`、`permit_digest_mismatch`、`frozen_evidence_mismatch` 和 `audit_chain_broken`，并只返回故障组件编号。

健康检查为 `GET /healthz`。应用错误统一返回 `{"error":{"code":"...","message":"..."}}`，版本冲突、非法状态和冻结写入使用 HTTP `409`，字段错误使用 HTTP `400`，角色错误使用 HTTP `403`。

## 振动预测与冻结证据

预测采用固定公式版本 `scaled-distance-v1`：依据单段最大药量、保护对象距离以及 `propagationK`、`propagationAlpha` 计算峰值质点速度，并以同一公式反算逐目标单段允许药量。快照保存结构化输入摘要、逐目标预测值、安全裕度、控制药量和稳定阻断编号；每次修订都新增记录，不覆盖历史修订或评估。

许可签发时会分别对保护对象基线、当前获批修订、评估快照和带覆盖校验的监测计划生成确定性 SHA-256 组件摘要，再生成总证据摘要和许可记录摘要。验真接口会重新读取冻结案卷、逐组件复算并校验冻结时的审计链位置。
