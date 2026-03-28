# Architecture Review - tinyclaw

## 总体评价
架构设计清晰，核心思路正确。框架定位轻量，agent 自治——以下建议均遵循这一原则。

## 1. 消息顺序与中断

**当前方案：**
- 同一 session_key 严格串行消费
- v0 固定 append 策略

**建议：**
- v0 保持 append 策略（正确）
- 中断/取消逻辑后续再加，由 agent 自己决定如何处理，框架不介入

## 2. 休眠策略

**当前方案：**
- v0 默认软休眠（XREADGROUP BLOCK 等待）
- 硬休眠和自动销毁后置

**待办（第一期后）：**
- 补充成本估算：软休眠的资源占用（CPU/内存）
- 给出"多少并发会话数时建议切换到硬休眠"的参考阈值

## 3. 监控指标

**待办（第一期后）：**
为每个指标增加建议的告警阈值：
- `reply_e2e_ms` P95 > 10s（告警）
- `reply_error_rate` > 1%（告警）
- `agent_wakeup_success_rate` < 95%（告警）
- `session_stream_pending_depth` > 100（告警）

## 4. 安全性

**待办（第一期后）：**
增加"安全实现清单"：
- [ ] agent 不能访问其他 session 的数据
- [ ] agent 不能访问平台级别的 secret
- [ ] 所有工具调用都经过 Tool Gateway
- [ ] 敏感数据（密码、token）不能出现在日志中
- [ ] 用户上传的文件需要病毒扫描
- [ ] agent 生成的内容需要内容审核

## 5. 文档一致性

**需要修复：**
1. ARCHITECTURE_V0.md 中"OneClaw"统一改为"tinyclaw"
2. 有些地方用 `chat_id_or_user_id`，有些地方用 `chat_id`，统一术语
3. Redis key 命名统一用冒号分隔

## 总结

架构设计的核心思路是正确的：
- ✅ 会话隔离清晰
- ✅ 消息流设计合理
- ✅ 生命周期管理完整
- ✅ 分阶段落地计划清晰
- ✅ 框架轻量，agent 自治

待办（第一期后）：
1. 补充成本估算和告警阈值
2. 完善安全实现清单
3. 统一文档术语（OneClaw -> tinyclaw）
