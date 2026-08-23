import { test, expect } from "@playwright/test";

const BASE = process.env.E2E_BASE || "http://127.0.0.1:27341";

test("登录 → 拖拽排餐 → 生成清单 → 勾选 → 导出", async ({ page }) => {
  await page.goto(BASE + "/login");
  await page.getByLabel("用户名").fill("demo");
  await page.getByLabel("密码").fill("demo123");
  await page.getByRole("button", { name: "进入厨房" }).click();
  await expect(page.getByRole("heading", { name: "本周吃什么" })).toBeVisible();

  const recipe = page.getByText("西红柿炒鸡蛋", { exact: true }).first();
  await expect(recipe).toBeVisible();
  const mondayDinner = page.locator("text=早餐").first();
  const box = page.locator("div").filter({ hasText: /^早餐$/ }).first();
  await recipe.dragTo(box);

  await page.getByRole("link", { name: "买菜清单" }).click();
  await expect(page.getByRole("heading", { name: "买菜清单" })).toBeVisible();
  await page.getByRole("button", { name: "重新生成" }).click();
  await expect(page.getByText("西红柿").first()).toBeVisible({ timeout: 8000 });

  const box2 = page.locator('input[type="checkbox"]').first();
  await box2.check();
  await page.getByRole("button", { name: "复制文本" }).click();
  await expect(page.getByText("已复制为纯文本")).toBeVisible();
  await page.getByRole("button", { name: "下载 Markdown" }).click();
});
