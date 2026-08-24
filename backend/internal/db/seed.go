package db

import (
	"encoding/json"
	"fmt"

	"gocooking/internal/model"
	"gocooking/pkg/logger"
	"gocooking/pkg/timeutil"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 种子策略：
//   - 用户 demo / demo123 仅在不存在时创建（bcrypt，禁止明文）
//   - 食材/菜谱按表空才注入，避免覆盖用户后改数据
//   - 演示冰箱写入鸡蛋、西红柿、生菜（临期）、豆腐、猪肉，方便验收 FEFO
//   - 菜谱名必须能被前端搜索到「西红柿炒鸡蛋」「酸菜鱼」
const demoUser = "demo"
const demoPass = "demo123"

func Seed(gdb *gorm.DB) error {
	if err := seedUser(gdb); err != nil {
		return err
	}
	if err := seedIngredients(gdb); err != nil {
		return err
	}
	if err := seedRecipes(gdb); err != nil {
		return err
	}
	if err := seedDemoPantry(gdb); err != nil {
		return err
	}
	logger.Info("seed complete")
	return nil
}

func seedUser(gdb *gorm.DB) error {
	var n int64
	if err := gdb.Model(&model.User{}).Where("username = ?", demoUser).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(demoPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u := model.User{Username: demoUser, PasswordHash: string(hash), CreatedAt: timeutil.Now()}
	return gdb.Create(&u).Error
}

func seedIngredients(gdb *gorm.DB) error {
	var n int64
	if err := gdb.Model(&model.Ingredient{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	now := timeutil.Now()
	rows := make([]model.Ingredient, 0, len(ingredientCatalog))
	for _, d := range ingredientCatalog {
		raw, _ := json.Marshal(d.Aliases)
		rows = append(rows, model.Ingredient{
			Name: d.Name, Aliases: string(raw), DefaultUnit: d.Unit, Dimension: d.Dim,
			ProduceCategory: d.Cat, Stall: d.Stall, IsStapleDefault: d.Staple, CreatedAt: now,
		})
	}
	if err := gdb.CreateInBatches(rows, 50).Error; err != nil {
		return fmt.Errorf("seed ingredients: %w", err)
	}
	logger.Info("seeded ingredients", "count", len(rows))
	return nil
}

func seedRecipes(gdb *gorm.DB) error {
	var n int64
	if err := gdb.Model(&model.Recipe{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	var ings []model.Ingredient
	if err := gdb.Find(&ings).Error; err != nil {
		return err
	}
	idByName := map[string]uint{}
	for _, ing := range ings {
		idByName[ing.Name] = ing.ID
		var aliases []string
		_ = json.Unmarshal([]byte(ing.Aliases), &aliases)
		for _, a := range aliases {
			idByName[a] = ing.ID
		}
	}
	now := timeutil.Now()
	for _, d := range recipeCatalog {
		steps, _ := json.Marshal(d.Steps)
		r := model.Recipe{
			Name: d.Name, CuisineTag: d.Tag, Servings: d.Servings, StepsJSON: string(steps),
			CoverURL: "", CreatedAt: now, UpdatedAt: now,
		}
		if err := gdb.Create(&r).Error; err != nil {
			return fmt.Errorf("seed recipe %s: %w", d.Name, err)
		}
		items := make([]model.RecipeItem, 0, len(d.Items))
		for _, it := range d.Items {
			id, ok := idByName[it.Name]
			if !ok {
				return fmt.Errorf("seed recipe %s: unknown ingredient %s", d.Name, it.Name)
			}
			items = append(items, model.RecipeItem{
				RecipeID: r.ID, IngredientID: id, Quantity: it.Qty, Unit: it.Unit, Optional: it.Opt,
			})
		}
		if err := gdb.Create(&items).Error; err != nil {
			return fmt.Errorf("seed recipe items %s: %w", d.Name, err)
		}
	}
	logger.Info("seeded recipes", "count", len(recipeCatalog))
	return nil
}

func seedDemoPantry(gdb *gorm.DB) error {
	var user model.User
	if err := gdb.Where("username = ?", demoUser).First(&user).Error; err != nil {
		return err
	}
	var n int64
	if err := gdb.Model(&model.PantryItem{}).Where("user_id = ?", user.ID).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	find := func(name string) uint {
		var ing model.Ingredient
		if err := gdb.Where("name = ?", name).First(&ing).Error; err != nil {
			return 0
		}
		return ing.ID
	}
	today := timeutil.Today()
	type p struct {
		name, unit string
		qty        float64
		days       int
	}
	samples := []p{
		{"鸡蛋", "个", 6, 8},
		{"西红柿", "个", 3, 5},
		{"生菜", "g", 200, 2},
		{"豆腐", "g", 400, 4},
		{"猪肉", "g", 250, 3},
	}
	for _, s := range samples {
		id := find(s.name)
		if id == 0 {
			continue
		}
		row := model.PantryItem{
			UserID: user.ID, IngredientID: id, Quantity: s.qty, Unit: s.unit,
			StockedAt: today.AddDate(0, 0, -1), ExpiresAt: today.AddDate(0, 0, s.days),
		}
		if err := gdb.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

type ingDef struct {
	Name    string
	Aliases []string
	Unit    string
	Dim     string
	Cat     string
	Stall   string
	Staple  bool
}

type recItemDef struct {
	Name string
	Qty  float64
	Unit string
	Opt  bool
}

type recDef struct {
	Name     string
	Tag      string
	Servings int
	Steps    []string
	Items    []recItemDef
}
