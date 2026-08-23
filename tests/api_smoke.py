#!/usr/bin/env python3
"""API smoke（Mock/离线，成本 ¥0）。默认打 docker 映射端口 27342。"""
import json
import os
import sys
import urllib.error
import urllib.request

BASE = os.environ.get("API_BASE", "http://127.0.0.1:27342/api/v1")


def req(method, path, body=None, token=None):
    data = None if body is None else json.dumps(body).encode()
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    r = urllib.request.Request(BASE + path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(r, timeout=15) as resp:
            raw = resp.read().decode()
            return resp.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        raw = e.read().decode()
        return e.code, json.loads(raw) if raw else {"error": {"message": raw}}


def ok(cond, msg):
    if not cond:
        print("[FAIL]", msg)
        sys.exit(1)
    print("[PASS]", msg)


def main():
    st, body = req("GET", "/health")
    ok(st == 200 and body.get("data", {}).get("status") == "ok", "Health Check")

    st, body = req("POST", "/auth/login", {"username": "demo", "password": "wrong"})
    ok(st == 401, "Bad password rejected")

    st, body = req("POST", "/auth/login", {"username": "demo", "password": "demo123"})
    token = body.get("data", {}).get("token")
    ok(st == 200 and token, "Login demo")

    st, body = req("GET", "/recipes?per_page=80", token=token)
    recipes = body.get("data") or []
    ok(st == 200 and len(recipes) >= 30, f"Seed recipes >= 30 (got {len(recipes)})")
    names = {r["name"] for r in recipes}
    ok("西红柿炒鸡蛋" in names and "酸菜鱼" in names, "Named recipes exist")

    st, body = req("GET", "/ingredients?per_page=200", token=token)
    ings = body.get("data") or []
    ok(st == 200 and len(ings) >= 120, f"Seed ingredients >= 120 (got {len(ings)})")
    tomato = next(i for i in ings if i["name"] == "西红柿")
    ok("番茄" in tomato.get("aliases", []), "Alias 番茄")

    egg = next(r for r in recipes if r["name"] == "西红柿炒鸡蛋")
    rice = next(r for r in recipes if r["name"] == "蛋炒饭")
    st, week = req("GET", "/meal-plan?week=2026-08-24", token=token)
    ok(st == 200, "Week plan readable")

    req("POST", "/meal-plan/clear", {"week": "2026-08-24"}, token)
    st, _ = req("POST", "/meal-plan/slots", {
        "date": "2026-08-24", "slot": "dinner", "recipe_id": egg["id"], "servings_multiplier": 1,
    }, token)
    ok(st == 201, "Drop 西红柿炒鸡蛋 to Monday dinner")
    st, _ = req("POST", "/meal-plan/slots", {
        "date": "2026-08-26", "slot": "lunch", "recipe_id": rice["id"], "servings_multiplier": 1,
    }, token)
    ok(st == 201, "Drop 蛋炒饭 to Wednesday lunch")

    st, body = req("POST", "/shopping-lists/generate", {"from": "2026-08-24", "to": "2026-08-30"}, token)
    ok(st == 200, "Generate shopping list")
    data = body.get("data") or {}
    items = [it for g in data.get("groups_by_stall", []) for it in g["items"]]
    rice = [it for it in items if it["name"] == "大米"]
    ok(len(rice) == 1, "大米出现在清单（无库存）")
    scallion = [it for it in items if it["name"] == "葱"]
    ok(len(scallion) == 1 and scallion[0]["quantity"] >= 2, "两道菜的葱合并为一条")
    # 演示冰箱有未过期鸡蛋，两道菜鸡蛋需求应被扣完（冗余采购率 0%）
    eggs_buy = [it for it in items if it["name"] == "鸡蛋"]
    ok(len(eggs_buy) == 0, "库存鸡蛋已扣减，清单不再出现")
    ok(any(d["name"] == "鸡蛋" for d in data.get("deducted", [])), "鸡蛋进入 deducted")
    salt_in = any(it["name"] == "盐" for it in items)
    ok(not salt_in, "Staple 盐 filtered from buy list")
    ok(any(it["name"] == "盐" for it in data.get("filtered", [])), "盐 appears in filtered")
    ok(len(data.get("groups_by_produce", [])) >= 1, "Produce grouping present")

    target = rice[0]
    st, _ = req("PATCH", "/shopping-lists/checks", {
        "from": "2026-08-24", "to": "2026-08-30",
        "ingredient_id": target["ingredient_id"], "unit": target.get("check_unit") or target["unit"],
        "dimension": target["dimension"], "checked": True,
    }, token)
    ok(st == 200, "Checkbox persisted")
    st, body = req("POST", "/shopping-lists/generate", {"from": "2026-08-24", "to": "2026-08-30"}, token)
    items = [it for g in body["data"]["groups_by_stall"] for it in g["items"]]
    ok(any(it["name"] == "大米" and it["checked"] for it in items), "Checkbox survives regenerate")

    print("[PASS] Mock API Response")
    print("Cost ¥0")


if __name__ == "__main__":
    main()
