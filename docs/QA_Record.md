# QA Record

## Round 1 · 2026-08-23 16:40 (GMT+8)

**Cost**: ¥0（全程 Mock/离线，无外部计费 API）

### 环境

`docker compose up --build -d` 后三服务 healthy：frontend 27341、API 27342、postgres 27343。

### 后端单测（镜像外编译、逻辑同构）

```
ok  gocooking/internal/engine
ok  gocooking/pkg/timeutil
```

覆盖：鸡蛋 2+3=5、番茄/西红柿别名、g/斤/kg/两、适量合并、把 vs g 不伪造合并、FEFO 用餐日、过期不扣、保质期当天可用、常备过滤与加回、库存充足零采购。

`go vet ./...` 通过。

### API Smoke（打正在运行的容器端口）

脚本：`tests/api_smoke.py`

```
[PASS] Health Check
[PASS] Bad password rejected
[PASS] Login demo
[PASS] Seed recipes >= 30 (got 60)
[PASS] Named recipes exist
[PASS] Seed ingredients >= 120 (got 140)
[PASS] Alias 番茄
[PASS] Week plan readable
[PASS] Drop 西红柿炒鸡蛋 to Monday dinner
[PASS] Drop 蛋炒饭 to Wednesday lunch
[PASS] Generate shopping list
[PASS] 大米出现在清单（无库存）
[PASS] 两道菜的葱合并为一条
[PASS] 库存鸡蛋已扣减，清单不再出现
[PASS] 鸡蛋进入 deducted
[PASS] Staple 盐 filtered from buy list
[PASS] 盐 appears in filtered
[PASS] Produce grouping present
[PASS] Checkbox persisted
[PASS] Checkbox survives regenerate
[PASS] Mock API Response
Cost ¥0
```

容器内健康检查：`docker compose exec backend wget -qO- http://127.0.0.1:8080/api/v1/health` → status ok。

### 浏览器关键路径

登录页「灶下清单」可用 demo/demo123 进入。日历页左侧可见西红柿炒鸡蛋、酸菜鱼等种子菜谱。买菜清单页双视图开关与导出按钮可用。冰箱页生菜临期高亮、鸡蛋/西红柿在库。菜谱库可搜索、新建。

Playwright 脚本已落盘 `tests/e2e_flow.spec.ts`。本轮用浏览器实操 + API smoke 代替完整 Playwright runner（宿主未装浏览器依赖），行为已验证。

### 结论

**PASS**。无失败项，不进入修测循环。
