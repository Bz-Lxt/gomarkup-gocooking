# GoCooking API

Base URL: `/api/v1`  
认证：除 `/health` 与 `/auth/login` 外，请求头 `Authorization: Bearer <token>`  
时间：用户可见日期为 `yyyy-MM-dd`，时间戳为北京时间 `yyyy-MM-dd HH:mm:ss`。

## 统一响应

成功：

```json
{ "data": {} }
```

列表：

```json
{ "data": [], "meta": { "total": 0, "page": 1, "per_page": 20 } }
```

错误：

```json
{
  "error": {
    "code": "validation_error",
    "message": "请求校验失败",
    "details": [{ "field": "username", "message": "必填", "code": "required" }]
  }
}
```

## 错误码

| HTTP | code | 含义 |
|---|---|---|
| 400 | invalid_json | 请求体不是合法 JSON |
| 400 / 422 | validation_error | 字段缺失、类型或边界错误 |
| 401 | unauthorized | 未登录或 token 失效 |
| 404 | not_found | 资源不存在 |
| 409 | conflict | 名称冲突 / 状态冲突 |
| 500 | internal_error | 未预期错误（不泄露内部细节） |

## 端点

### GET /health

```json
{ "data": { "status": "ok", "time": "2026-08-23 16:00:00" } }
```

### POST /auth/login

请求：`{ "username": "demo", "password": "demo123" }`  
响应：`{ "data": { "token": "eyJ...", "username": "demo" } }`

### 菜谱

- `GET /recipes?q=&tag=&page=&per_page=`
- `GET /recipes/:id`
- `POST /recipes`
- `PUT /recipes/:id`
- `DELETE /recipes/:id`
- `POST /recipes/:id/duplicate`（V1）

创建示例：

```json
{
  "name": "西红柿炒鸡蛋",
  "cover_url": "",
  "cuisine_tag": "家常",
  "servings": 2,
  "steps": ["鸡蛋打散", "西红柿切块", "热锅炒制"],
  "items": [
    { "ingredient_id": 1, "quantity": 2, "unit": "个", "optional": false }
  ]
}
```

### 食材

- `GET /ingredients?q=&stall=&category=&page=&per_page=`

### 周计划

- `GET /meal-plan?week=2026-08-24`（任意落在该周的日期，周一为周始）
- `POST /meal-plan/slots`
- `PATCH /meal-plan/slots/:id`
- `DELETE /meal-plan/slots/:id`
- `POST /meal-plan/clear` `{ "week": "2026-08-24" }`
- `POST /meal-plan/copy-next` `{ "week": "2026-08-24" }`

创建槽位：

```json
{
  "date": "2026-08-24",
  "slot": "dinner",
  "recipe_id": 1,
  "servings_multiplier": 1
}
```

`slot` ∈ `breakfast|lunch|dinner`，`servings_multiplier` ∈ `[0.5, 4]`。

### 冰箱

- `GET /pantry`
- `POST /pantry`
- `PUT /pantry/:id`
- `DELETE /pantry/:id`

```json
{
  "ingredient_id": 12,
  "quantity": 6,
  "unit": "个",
  "stocked_at": "2026-08-20",
  "expires_at": "2026-08-27"
}
```

### 买菜清单

- `POST /shopping-lists/generate` `{ "from": "2026-08-24", "to": "2026-08-30" }`
- `PATCH /shopping-lists/checks` `{ "from":"...","to":"...","ingredient_id":1,"unit":"个","dimension":"count","checked":true }`
- `POST /shopping-lists/restore` `{ "from":"...","to":"...","ingredient_id":9,"unit":"g","dimension":"weight" }`

生成响应关键字段：`groups_by_stall`、`groups_by_produce`、`filtered`、`expiry_alerts`、`deducted`、`sources`、`needs_review`。

### 常备设置

- `GET /settings/staples`
- `PUT /settings/staples` `{ "items": [{ "ingredient_id": 9, "enabled": false }] }`
- `POST /settings/staples` `{ "ingredient_id": 88, "enabled": true }`
