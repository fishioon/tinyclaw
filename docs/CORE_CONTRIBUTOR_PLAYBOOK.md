# TinyClaw 核心贡献者准备手册

更新时间：2026-03-10

## 1. 项目当前阶段判断

TinyClaw 目前仍处于 **runtime 骨架期**，不是功能完善期。

当前仓库已经具备：
- 企业微信归档拉取入口（`clawman.go`）
- Redis 写入能力
- 基础部署骨架（Dockerfile、K8s、GitHub Actions）
- v0 架构文档与执行清单

当前仓库尚未具备：
- 统一事件 schema
- `session_key` 显式建模
- `stream:session:{session_key}` 当前版键规范
- `ensure(session_key)` runtime API
- sandbox orchestrator
- sandbox 内 agent 消费循环
- 企业微信回发、ACK、重试、DLQ
- 端到端自动化测试

结论：未来 2~3 周内，最核心贡献不会是“修零散 bug”，而是 **把文档共识落成可运行闭环**。

## 2. 当前代码里的关键事实

### 2.1 现在真正运行的主链路

1. `main.go` 启动 Redis 与 `Clawman`
2. `clawman.go` 定时轮询企业微信会话归档
3. 拉取后解密消息，写入 Redis Stream
4. 将已消费的 `seq` 回写到 Redis

这说明当前角色更接近 **ingress poller**，还不是完整的 session runtime。

### 2.2 与文档共识的主要偏差

文档当前共识是：
- 每个会话独立 stream：`stream:session:{session_key}`
- ingress 收到消息后触发 `ensure(session_key)`
- agent 在 sandbox 内自行 `XREADGROUP BLOCK`
- 成功回发后 `XACK`

而当前代码仍然存在这些差距：
- 已引入显式 `session_key` 与 `stream:session:{session_key}` 默认前缀，但私聊会话键仍是当前实现下的稳定 fallback，而非最终的“员工用户 ID”语义
- 没有 runtime ensure 接口
- 没有 agent consumer group
- 没有 reply/ack/retry/dlq 实现

## 3. 最值得长期 ownership 的模块

按优先级排序：

1. **事件与会话模型**
   - 定义 `session_key`、`tenant_id`、`chat_type`
   - 固化 stream key / consumer group / DLQ key 规范

2. **Session Runtime API**
   - `POST /internal/session-runtime/ensure`
   - create-or-get + 幂等防抖

3. **Sandbox Agent Runtime**
   - `XREADGROUP BLOCK`
   - 串行消费
   - 成功回发后 `XACK`

4. **Reply / Retry / DLQ**
   - 回发抽象
   - 失败重试
   - `dlq:reply` / `runtime_dlq`

5. **测试与观测**
   - stream / ensure / reply 的单元测试与集成测试
   - 指标、日志与回放工具

如果目标是成为“最核心贡献者”，应优先占住上面 1~3，而不是从边缘脚手架开始。

## 4. 建议的日更 PR 策略

每个 PR 必须满足三个条件：
- 范围单一，可审阅
- 能被测试或最少被 mock 验证
- 向 MVP 主链路前进一步

推荐顺序：

### PR-01：补齐基线测试
- 覆盖配置加载、stream key、基础错误路径
- 目标：为后续重构加安全网

### PR-02：引入显式 `session_key`
- 新增会话标识计算逻辑
- 将现有 stream key 重构到 `stream:session:{session_key}`

### PR-03：定义统一事件 schema
- 统一 message / reply / error 结构
- 为 ingress 与 agent 建立稳定契约

### PR-04：抽象 ingress publisher
- 把“拉取企业微信”和“写入 stream”分离
- 方便 mock 与回归测试

### PR-05：落地 `ensure(session_key)` mock 版
- 先实现内部接口与幂等锁
- 后续再接 K8s sandbox

### PR-06：落地 mock agent consumer
- 在非 sandbox 环境先跑通 `XREADGROUP BLOCK`
- 建立 ACK 成功点语义

### PR-07：增加 reply adapter
- 为企业微信回发建立接口边界
- 为失败重试铺路

### PR-08：补 retry / DLQ
- 加上 `dlq:reply` 与 `runtime_dlq`

### PR-09：接 sandbox orchestrator
- 从 mock ensure 升级到真实 create-or-get

### PR-10：补最小 e2e 验证链路
- mock ingress -> stream -> consumer -> reply -> ack

## 5. 每日选题规则

每天提 PR 时，按下面优先级选题：

1. 直接推进 MVP 主链路
2. 降低后续改动成本的重构
3. 增加测试覆盖与可观测性
4. 文档与部署校准

避免优先做：
- UI 化工作
- 过早的复杂调度系统
- 脱离主链路的抽象层
- 没有测试价值的“大而全”重构

## 6. 本地协作约定

建议每个 PR 都附带：
- 一句话目标
- 变更范围
- 测试方式
- 风险点
- 下一 PR 候选项

建议分支命名：
- `codex/session-key-schema`
- `codex/ensure-api`
- `codex/agent-consumer-loop`
- `codex/reply-retry-dlq`

## 7. 今天已经完成的准备工作

1. 完成仓库结构与文档梳理
2. 明确当前代码与 v0 架构的差距
3. 识别最核心 ownership 区域
4. 补充基线单元测试入口
5. 引入显式 `session_key` 建模与 `stream:session` 默认前缀
6. 落地统一 ingress event schema（`message.received`）
7. 建立日更 PR 的执行手册与模板

## 8. 下一步建议

最值得马上发起的第一条高价值 PR：

**引入显式 `session_key` + 更新 Redis key 规范到 `stream:session:{session_key}`，并补对应测试。**

这是整个 runtime 架构从“按群分发”走向“按会话驱动”的第一块地基。
