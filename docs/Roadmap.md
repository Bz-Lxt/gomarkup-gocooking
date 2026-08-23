# GoCooking Roadmap

> 版本：v1.0 ｜ 2026-08-23 (GMT+8) ｜ Chief Architect

## 阶段顺序决策

**Logic-First（交换 Phase 2 与 Phase 3）。**  
理由：七天日历的槽位、拖拽载荷、份数倍数完全派生自 MealPlan 模型与聚合引擎契约，先写 UI 会因字段返工。

## 目录结构

```
GoCooking/
├── backend/                 # Go（模块 1.25 / 镜像 golang:1.25-alpine）+ Gin + GORM
├── frontend-user/           # React 18 + Vite + dnd-kit
├── frontend-admin/          # 本项目无独立后台，仅保留占位说明
├── frontend-mp/             # 非小程序项目，仅保留占位说明
├── docker-compose.yml
└── docs/
```

## 开发端口（随机，/deploy 前不得改为 8081）

| 服务 | 宿主端口 | 容器端口 |
|---|---|---|
| frontend-user (Nginx) | **27341** | 80 |
| backend API | **27342** | 8080 |
| PostgreSQL 16 | **27343** | 5432 |

## MVP（交付必须完成）

- [x] A-01 Git 初始化与 `.gitignore`
- [x] A-02 docker-compose + 多阶段 Dockerfile（ARM64/AMD64，国内镜像源）
- [x] L-01 北京时间 `pkg/timeutil` + slog `pkg/logger` + 统一错误码
- [x] L-02 数据模型：User / Ingredient / Recipe / MealSlot / Pantry / StapleOverride / ShoppingCheck
- [x] L-03 单位换算引擎（C-03 三层策略，禁止伪造系数）
- [x] L-04 食材聚合引擎（别名归一 + 同量纲合并）
- [x] L-05 FEFO 库存扣减（按用餐日，C-02）
- [x] L-06 常备清单过滤（按用户开关，非调料分类，C-01）
- [x] L-07 JWT 登录 + 测试账号 demo/demo123
- [x] L-08 菜谱 / 食材 / 排期 / 冰箱 / 设置 / 清单 API
- [x] L-09 种子：≥120 食材、≥30 菜谱（含西红柿炒鸡蛋、酸菜鱼）
- [x] L-10 核心引擎单测（合并 / 别名 / 换算 / FEFO / 常备）
- [x] U-01 DesignSpec + 湿市场晨光视觉
- [x] U-02 登录页
- [x] U-03 七天日历拖拽（dnd-kit，21 槽位，份数 0.5x–4x）
- [x] U-04 买菜清单双视图 + 复选框持久化 + 溯源 + 常备展开加回 + 导出
- [x] U-05 菜谱 CRUD / 冰箱库存 / 常备设置
- [x] Q-01 API smoke + 核心单测全绿，成本 ¥0
- [x] Q-02 Playwright 脚本已落盘；本轮以浏览器实操 + API smoke 验收
- [x] AUD-01 对照 audit-rules 出具 AuditReport

## V1（体验增强，实现但不阻塞核心验收）

- [x] V1-01 菜谱复制为副本
- [x] V1-02 一周排期清空 / 复制到下一周
- [x] V1-03 库存按摊位分类查看

## V2（不阻塞本次交付）

- [ ] V2-01 采购回写入库
- [ ] V2-02 PDF 打印
- [ ] V2-03 AI 菜谱解析（Mock 默认 + 真实开关，README §7）
