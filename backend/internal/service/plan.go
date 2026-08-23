package service

import (
	"sort"
	"strings"
	"time"

	"gocooking/internal/dto"
	"gocooking/internal/engine"
	"gocooking/internal/model"
	"gocooking/pkg/apperr"
	"gocooking/pkg/timeutil"

	"gorm.io/gorm"
)

type Planner struct {
	DB *gorm.DB
}

func NewPlanner(db *gorm.DB) *Planner { return &Planner{DB: db} }

func (s *Planner) WeekPlan(userID uint, week string) (dto.WeekPlanOut, error) {
	anchor, err := timeutil.ParseDate(week)
	if err != nil {
		return dto.WeekPlanOut{}, apperr.Validation("日期格式应为 yyyy-MM-dd", apperr.FieldError{Field: "week", Message: "格式错误", Code: "invalid_format"})
	}
	start := timeutil.StartOfWeek(anchor)
	end := timeutil.EndOfWeek(anchor)
	var slots []model.MealSlot
	if err := s.DB.Preload("Recipe.Items.Ingredient").
		Where("user_id = ? AND plan_date >= ? AND plan_date <= ?", userID, start, end).
		Order("plan_date asc, slot asc, sort_order asc, id asc").
		Find(&slots).Error; err != nil {
		return dto.WeekPlanOut{}, apperr.Internal(err)
	}
	out := dto.WeekPlanOut{WeekStart: timeutil.FormatDate(start), WeekEnd: timeutil.FormatDate(end)}
	for _, sl := range slots {
		out.Slots = append(out.Slots, slotOut(sl))
	}
	if out.Slots == nil {
		out.Slots = []dto.SlotOut{}
	}
	return out, nil
}

func (s *Planner) AddSlot(userID uint, in dto.SlotIn) (dto.SlotOut, error) {
	d, err := timeutil.ParseDate(in.Date)
	if err != nil {
		return dto.SlotOut{}, apperr.Validation("日期格式应为 yyyy-MM-dd", apperr.FieldError{Field: "date", Message: "格式错误", Code: "invalid_format"})
	}
	if !model.ValidSlots[in.Slot] {
		return dto.SlotOut{}, apperr.Validation("餐位不合法", apperr.FieldError{Field: "slot", Message: "须为 breakfast/lunch/dinner", Code: "invalid"})
	}
	if in.RecipeID == 0 {
		return dto.SlotOut{}, apperr.Required("recipe_id")
	}
	mult := in.ServingsMultiplier
	if mult == 0 {
		mult = 1
	}
	if mult < 0.5 || mult > 4 {
		return dto.SlotOut{}, apperr.Validation("份数倍数超范围", apperr.FieldError{Field: "servings_multiplier", Message: "须在 0.5–4", Code: "out_of_range"})
	}
	var rec model.Recipe
	if err := s.DB.Where("id = ? AND (user_id IS NULL OR user_id = ?)", in.RecipeID, userID).First(&rec).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return dto.SlotOut{}, apperr.NotFound("菜谱")
		}
		return dto.SlotOut{}, apperr.Internal(err)
	}
	var maxOrd int
	_ = s.DB.Model(&model.MealSlot{}).
		Where("user_id = ? AND plan_date = ? AND slot = ?", userID, d, in.Slot).
		Select("COALESCE(MAX(sort_order),0)").Scan(&maxOrd)
	row := model.MealSlot{
		UserID: userID, PlanDate: d, Slot: in.Slot, RecipeID: in.RecipeID,
		ServingsMultiplier: mult, SortOrder: maxOrd + 1, CreatedAt: timeutil.Now(),
	}
	if err := s.DB.Create(&row).Error; err != nil {
		return dto.SlotOut{}, apperr.Internal(err)
	}
	if err := s.DB.Preload("Recipe.Items.Ingredient").First(&row, row.ID).Error; err != nil {
		return dto.SlotOut{}, apperr.Internal(err)
	}
	return slotOut(row), nil
}

func (s *Planner) PatchSlot(userID, id uint, in dto.SlotPatch) (dto.SlotOut, error) {
	var row model.MealSlot
	if err := s.DB.Where("id = ? AND user_id = ?", id, userID).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return dto.SlotOut{}, apperr.NotFound("排期")
		}
		return dto.SlotOut{}, apperr.Internal(err)
	}
	updates := map[string]any{}
	if in.Date != nil {
		d, err := timeutil.ParseDate(*in.Date)
		if err != nil {
			return dto.SlotOut{}, apperr.Validation("日期格式错误", apperr.FieldError{Field: "date", Message: "格式错误", Code: "invalid_format"})
		}
		updates["plan_date"] = d
	}
	if in.Slot != nil {
		if !model.ValidSlots[*in.Slot] {
			return dto.SlotOut{}, apperr.Validation("餐位不合法", apperr.FieldError{Field: "slot", Message: "无效", Code: "invalid"})
		}
		updates["slot"] = *in.Slot
	}
	if in.ServingsMultiplier != nil {
		if *in.ServingsMultiplier < 0.5 || *in.ServingsMultiplier > 4 {
			return dto.SlotOut{}, apperr.Validation("份数倍数超范围", apperr.FieldError{Field: "servings_multiplier", Message: "须在 0.5–4", Code: "out_of_range"})
		}
		updates["servings_multiplier"] = *in.ServingsMultiplier
	}
	if in.SortOrder != nil {
		updates["sort_order"] = *in.SortOrder
	}
	if len(updates) > 0 {
		if err := s.DB.Model(&row).Updates(updates).Error; err != nil {
			return dto.SlotOut{}, apperr.Internal(err)
		}
	}
	if err := s.DB.Preload("Recipe.Items.Ingredient").First(&row, id).Error; err != nil {
		return dto.SlotOut{}, apperr.Internal(err)
	}
	return slotOut(row), nil
}

func (s *Planner) DeleteSlot(userID, id uint) error {
	res := s.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.MealSlot{})
	if res.Error != nil {
		return apperr.Internal(res.Error)
	}
	if res.RowsAffected == 0 {
		return apperr.NotFound("排期")
	}
	return nil
}

func (s *Planner) ClearWeek(userID uint, week string) error {
	anchor, err := timeutil.ParseDate(week)
	if err != nil {
		return apperr.Validation("日期格式错误", apperr.FieldError{Field: "week", Message: "格式错误", Code: "invalid_format"})
	}
	start, end := timeutil.StartOfWeek(anchor), timeutil.EndOfWeek(anchor)
	if err := s.DB.Where("user_id = ? AND plan_date >= ? AND plan_date <= ?", userID, start, end).
		Delete(&model.MealSlot{}).Error; err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (s *Planner) CopyNext(userID uint, week string) error {
	anchor, err := timeutil.ParseDate(week)
	if err != nil {
		return apperr.Validation("日期格式错误", apperr.FieldError{Field: "week", Message: "格式错误", Code: "invalid_format"})
	}
	start, end := timeutil.StartOfWeek(anchor), timeutil.EndOfWeek(anchor)
	nextStart := start.AddDate(0, 0, 7)
	nextEnd := end.AddDate(0, 0, 7)
	var src []model.MealSlot
	if err := s.DB.Where("user_id = ? AND plan_date >= ? AND plan_date <= ?", userID, start, end).Find(&src).Error; err != nil {
		return apperr.Internal(err)
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND plan_date >= ? AND plan_date <= ?", userID, nextStart, nextEnd).
			Delete(&model.MealSlot{}).Error; err != nil {
			return err
		}
		for _, sl := range src {
			n := sl
			n.ID = 0
			n.PlanDate = sl.PlanDate.AddDate(0, 0, 7)
			n.CreatedAt = timeutil.Now()
			if err := tx.Create(&n).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Planner) ListPantry(userID uint) ([]dto.PantryOut, error) {
	var rows []model.PantryItem
	if err := s.DB.Preload("Ingredient").Where("user_id = ?", userID).Order("expires_at asc").Find(&rows).Error; err != nil {
		return nil, apperr.Internal(err)
	}
	today := timeutil.Today()
	out := make([]dto.PantryOut, 0, len(rows))
	for _, r := range rows {
		out = append(out, pantryOut(r, today))
	}
	return out, nil
}

func (s *Planner) CreatePantry(userID uint, in dto.PantryIn) (dto.PantryOut, error) {
	row, err := pantryFromInput(userID, in)
	if err != nil {
		return dto.PantryOut{}, err
	}
	if err := s.mustIngredient(in.IngredientID); err != nil {
		return dto.PantryOut{}, err
	}
	if err := s.DB.Create(row).Error; err != nil {
		return dto.PantryOut{}, apperr.Internal(err)
	}
	if err := s.DB.Preload("Ingredient").First(row, row.ID).Error; err != nil {
		return dto.PantryOut{}, apperr.Internal(err)
	}
	return pantryOut(*row, timeutil.Today()), nil
}

func (s *Planner) UpdatePantry(userID, id uint, in dto.PantryIn) (dto.PantryOut, error) {
	var cur model.PantryItem
	if err := s.DB.Where("id = ? AND user_id = ?", id, userID).First(&cur).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return dto.PantryOut{}, apperr.NotFound("库存")
		}
		return dto.PantryOut{}, apperr.Internal(err)
	}
	next, err := pantryFromInput(userID, in)
	if err != nil {
		return dto.PantryOut{}, err
	}
	if err := s.mustIngredient(in.IngredientID); err != nil {
		return dto.PantryOut{}, err
	}
	if err := s.DB.Model(&cur).Updates(map[string]any{
		"ingredient_id": next.IngredientID, "quantity": next.Quantity, "unit": next.Unit,
		"stocked_at": next.StockedAt, "expires_at": next.ExpiresAt, "updated_at": timeutil.Now(),
	}).Error; err != nil {
		return dto.PantryOut{}, apperr.Internal(err)
	}
	if err := s.DB.Preload("Ingredient").First(&cur, id).Error; err != nil {
		return dto.PantryOut{}, apperr.Internal(err)
	}
	return pantryOut(cur, timeutil.Today()), nil
}

func (s *Planner) DeletePantry(userID, id uint) error {
	res := s.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.PantryItem{})
	if res.Error != nil {
		return apperr.Internal(res.Error)
	}
	if res.RowsAffected == 0 {
		return apperr.NotFound("库存")
	}
	return nil
}

func (s *Planner) Staples(userID uint) ([]dto.StapleItem, error) {
	var ings []model.Ingredient
	if err := s.DB.Order("is_staple_default desc, id asc").Find(&ings).Error; err != nil {
		return nil, apperr.Internal(err)
	}
	var ov []model.StapleOverride
	_ = s.DB.Where("user_id = ?", userID).Find(&ov)
	om := map[uint]bool{}
	for _, o := range ov {
		om[o.IngredientID] = o.Enabled
	}
	out := make([]dto.StapleItem, 0, len(ings))
	for _, ing := range ings {
		en := ing.IsStapleDefault
		if v, ok := om[ing.ID]; ok {
			en = v
		}
		if !ing.IsStapleDefault {
			if _, custom := om[ing.ID]; !custom {
				continue
			}
		}
		out = append(out, dto.StapleItem{
			IngredientID: ing.ID, Name: ing.Name, Enabled: en, DefaultEnabled: ing.IsStapleDefault,
		})
	}
	return out, nil
}

func (s *Planner) PutStaples(userID uint, in dto.StaplesPut) ([]dto.StapleItem, error) {
	if len(in.Items) == 0 {
		return nil, apperr.Validation("items 不能为空", apperr.FieldError{Field: "items", Message: "必填", Code: "required"})
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		for _, it := range in.Items {
			if it.IngredientID == 0 {
				return apperr.Required("items.ingredient_id")
			}
			row := model.StapleOverride{UserID: userID, IngredientID: it.IngredientID, Enabled: it.Enabled, UpdatedAt: timeutil.Now()}
			if err := tx.Where("user_id = ? AND ingredient_id = ?", userID, it.IngredientID).
				Assign(row).FirstOrCreate(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if ae, ok := apperr.As(err); ok {
			return nil, ae
		}
		return nil, apperr.Internal(err)
	}
	return s.Staples(userID)
}

func (s *Planner) AddStaple(userID uint, in dto.StapleAdd) ([]dto.StapleItem, error) {
	if in.IngredientID == 0 {
		return nil, apperr.Required("ingredient_id")
	}
	if err := s.mustIngredient(in.IngredientID); err != nil {
		return nil, err
	}
	row := model.StapleOverride{UserID: userID, IngredientID: in.IngredientID, Enabled: in.Enabled, UpdatedAt: timeutil.Now()}
	if err := s.DB.Where("user_id = ? AND ingredient_id = ?", userID, in.IngredientID).
		Assign(row).FirstOrCreate(&row).Error; err != nil {
		return nil, apperr.Internal(err)
	}
	return s.Staples(userID)
}

// Generate 编排引擎六步并挂上勾选/加回状态。
// 1) 展开排期 2) 按日聚合 3) FEFO 扣库存 4) 跨日再合并
// 5) 常备过滤 6) 按摊位/蔬菜分类归组。
// 勾选键 = ingredient_id + dimension + base unit（check_unit），避免「斤/g」展示单位对不上。
func (s *Planner) Generate(userID uint, fromS, toS string) (dto.ShoppingOut, error) {
	from, to, err := parseRange(fromS, toS)
	if err != nil {
		return dto.ShoppingOut{}, err
	}
	var slots []model.MealSlot
	if err := s.DB.Preload("Recipe.Items.Ingredient").
		Where("user_id = ? AND plan_date >= ? AND plan_date <= ?", userID, from, to).
		Find(&slots).Error; err != nil {
		return dto.ShoppingOut{}, apperr.Internal(err)
	}
	var pantry []model.PantryItem
	if err := s.DB.Preload("Ingredient").Where("user_id = ?", userID).Find(&pantry).Error; err != nil {
		return dto.ShoppingOut{}, apperr.Internal(err)
	}
	lines := expandSlots(slots)
	daily := engine.AggregateDaily(lines)
	lots := make([]engine.Lot, 0, len(pantry))
	for _, p := range pantry {
		name := ""
		if p.Ingredient != nil {
			name = p.Ingredient.Name
		}
		lots = append(lots, engine.Lot{
			ID: p.ID, IngredientID: p.IngredientID, Name: name,
			Quantity: p.Quantity, Unit: p.Unit, ExpiresAt: p.ExpiresAt,
		})
	}
	ded := engine.DeductFEFO(daily, lots, timeutil.Today())
	merged := engine.MergeAcrossDays(ded.Remaining)
	on, err := s.stapleMap(userID)
	if err != nil {
		return dto.ShoppingOut{}, err
	}
	keep, filtered := engine.FilterStaples(merged, on)

	var checks []model.ShoppingCheck
	_ = s.DB.Where("user_id = ? AND range_from = ? AND range_to = ?", userID, timeutil.FormatDate(from), timeutil.FormatDate(to)).Find(&checks)
	restoredIDs := map[string]bool{}
	checked := map[string]bool{}
	for _, c := range checks {
		k := checkKey(c.IngredientID, c.Dimension, c.Unit)
		checked[k] = c.Checked
		if c.Restored {
			restoredIDs[k] = true
		}
	}
	var stillFiltered []engine.MergedItem
	for _, it := range filtered {
		k := checkKey(it.IngredientID, string(it.Dimension), it.BaseUnit)
		if restoredIDs[k] {
			keep = append(keep, it)
			continue
		}
		stillFiltered = append(stillFiltered, it)
	}

	meta, err := s.ingMeta()
	if err != nil {
		return dto.ShoppingOut{}, err
	}
	out := dto.ShoppingOut{
		From: timeutil.FormatDate(from), To: timeutil.FormatDate(to),
		GroupsByStall:   groupBy(keep, meta, checked, false, func(m ingMeta) string { return m.Stall }),
		GroupsByProduce: groupBy(keep, meta, checked, false, func(m ingMeta) string { return m.Produce }),
		Filtered:        toItems(stillFiltered, meta, checked, true),
	}
	for _, a := range ded.Alerts {
		out.ExpiryAlerts = append(out.ExpiryAlerts, dto.AlertOut{
			IngredientID: a.IngredientID, Name: a.Name, ExpiresAt: timeutil.FormatDate(a.ExpiresAt), Message: a.Message,
		})
	}
	agg := map[uint]*dto.DeductOut{}
	for _, d := range ded.Deducted {
		if cur, ok := agg[d.IngredientID]; ok {
			cur.FromPantry += d.FromPantry
			continue
		}
		cp := dto.DeductOut{IngredientID: d.IngredientID, Name: d.Name, FromPantry: d.FromPantry, Unit: d.Unit}
		agg[d.IngredientID] = &cp
	}
	for _, v := range agg {
		out.Deducted = append(out.Deducted, *v)
	}
	if out.ExpiryAlerts == nil {
		out.ExpiryAlerts = []dto.AlertOut{}
	}
	if out.Deducted == nil {
		out.Deducted = []dto.DeductOut{}
	}
	if out.Filtered == nil {
		out.Filtered = []dto.ListItemOut{}
	}
	return out, nil
}

func (s *Planner) SetCheck(userID uint, in dto.CheckReq) error {
	from, to, err := parseRange(in.From, in.To)
	if err != nil {
		return err
	}
	if in.IngredientID == 0 {
		return apperr.Required("ingredient_id")
	}
	row := model.ShoppingCheck{
		UserID: userID, RangeFrom: timeutil.FormatDate(from), RangeTo: timeutil.FormatDate(to),
		IngredientID: in.IngredientID, Unit: engine.NormalizeUnit(in.Unit), Dimension: in.Dimension,
		Checked: in.Checked, UpdatedAt: timeutil.Now(),
	}
	return upsertCheck(s.DB, row, map[string]any{"checked": in.Checked, "updated_at": timeutil.Now()})
}

func (s *Planner) Restore(userID uint, in dto.RestoreReq) error {
	from, to, err := parseRange(in.From, in.To)
	if err != nil {
		return err
	}
	if in.IngredientID == 0 {
		return apperr.Required("ingredient_id")
	}
	row := model.ShoppingCheck{
		UserID: userID, RangeFrom: timeutil.FormatDate(from), RangeTo: timeutil.FormatDate(to),
		IngredientID: in.IngredientID, Unit: engine.NormalizeUnit(in.Unit), Dimension: in.Dimension,
		Restored: true, UpdatedAt: timeutil.Now(),
	}
	return upsertCheck(s.DB, row, map[string]any{"restored": true, "updated_at": timeutil.Now()})
}

func upsertCheck(db *gorm.DB, row model.ShoppingCheck, assign map[string]any) error {
	if err := db.Where(
		"user_id = ? AND range_from = ? AND range_to = ? AND ingredient_id = ? AND unit = ? AND dimension = ?",
		row.UserID, row.RangeFrom, row.RangeTo, row.IngredientID, row.Unit, row.Dimension,
	).Assign(assign).FirstOrCreate(&row).Error; err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (s *Planner) mustIngredient(id uint) error {
	var n int64
	if err := s.DB.Model(&model.Ingredient{}).Where("id = ?", id).Count(&n).Error; err != nil {
		return apperr.Internal(err)
	}
	if n == 0 {
		return apperr.NotFound("食材")
	}
	return nil
}

func (s *Planner) stapleMap(userID uint) (map[uint]bool, error) {
	var ings []model.Ingredient
	if err := s.DB.Find(&ings).Error; err != nil {
		return nil, apperr.Internal(err)
	}
	on := map[uint]bool{}
	for _, ing := range ings {
		if ing.IsStapleDefault {
			on[ing.ID] = true
		}
	}
	var ov []model.StapleOverride
	if err := s.DB.Where("user_id = ?", userID).Find(&ov).Error; err != nil {
		return nil, apperr.Internal(err)
	}
	for _, o := range ov {
		on[o.IngredientID] = o.Enabled
	}
	return on, nil
}

type ingMeta struct {
	Name    string
	Stall   string
	Produce string
}

func (s *Planner) ingMeta() (map[uint]ingMeta, error) {
	var ings []model.Ingredient
	if err := s.DB.Find(&ings).Error; err != nil {
		return nil, apperr.Internal(err)
	}
	m := map[uint]ingMeta{}
	for _, ing := range ings {
		m[ing.ID] = ingMeta{Name: ing.Name, Stall: ing.Stall, Produce: ing.ProduceCategory}
	}
	return m, nil
}

func expandSlots(slots []model.MealSlot) []engine.Line {
	var lines []engine.Line
	for _, sl := range slots {
		if sl.Recipe == nil {
			continue
		}
		for _, it := range sl.Recipe.Items {
			name := ""
			if it.Ingredient != nil {
				name = it.Ingredient.Name
			}
			lines = append(lines, engine.Line{
				IngredientID: it.IngredientID, Name: name, Quantity: it.Quantity, Unit: it.Unit,
				MealDate: sl.PlanDate, Slot: sl.Slot, RecipeName: sl.Recipe.Name,
				Multiplier: sl.ServingsMultiplier, Optional: it.Optional,
			})
		}
	}
	return lines
}

func groupBy(items []engine.MergedItem, meta map[uint]ingMeta, checked map[string]bool, filtered bool, keyFn func(ingMeta) string) []dto.GroupOut {
	buckets := map[string][]dto.ListItemOut{}
	order := []string{}
	seen := map[string]bool{}
	for _, it := range items {
		m := meta[it.IngredientID]
		k := keyFn(m)
		if k == "" {
			k = model.CatOther
		}
		if !seen[k] {
			seen[k] = true
			order = append(order, k)
		}
		buckets[k] = append(buckets[k], itemOut(it, m, checked, filtered))
	}
	sort.Strings(order)
	out := make([]dto.GroupOut, 0, len(order))
	for _, k := range order {
		out = append(out, dto.GroupOut{Key: k, Items: buckets[k]})
	}
	return out
}

func toItems(items []engine.MergedItem, meta map[uint]ingMeta, checked map[string]bool, filtered bool) []dto.ListItemOut {
	out := make([]dto.ListItemOut, 0, len(items))
	for _, it := range items {
		out = append(out, itemOut(it, meta[it.IngredientID], checked, filtered))
	}
	return out
}

func itemOut(it engine.MergedItem, m ingMeta, checked map[string]bool, filtered bool) dto.ListItemOut {
	q, u := engine.PrettyQty(it.Dimension, it.BaseQty, it.BaseUnit)
	row := dto.ListItemOut{
		IngredientID: it.IngredientID, Name: it.Name, Quantity: q, Unit: u, CheckUnit: it.BaseUnit,
		Display: it.Display(), Dimension: string(it.Dimension), NeedsReview: it.NeedsReview,
		Checked: checked[checkKey(it.IngredientID, string(it.Dimension), it.BaseUnit)],
		Filtered: filtered, ProduceCategory: m.Produce, Stall: m.Stall,
	}
	for _, src := range it.Sources {
		row.Sources = append(row.Sources, dto.SourceOut{
			Date: timeutil.FormatDate(src.Date), Slot: src.Slot, SlotLabel: model.SlotLabel(src.Slot),
			RecipeName: src.RecipeName, Quantity: src.Quantity, Unit: src.Unit,
		})
	}
	for _, sib := range it.Siblings {
		row.Siblings = append(row.Siblings, itemOut(sib, m, checked, filtered))
	}
	if row.Sources == nil {
		row.Sources = []dto.SourceOut{}
	}
	return row
}

func checkKey(id uint, dim, unit string) string {
	return strings.Join([]string{utoa(id), dim, engine.NormalizeUnit(unit)}, "|")
}

func utoa(id uint) string {
	return itoa(int(id))
}

func parseRange(fromS, toS string) (time.Time, time.Time, error) {
	f, e1 := timeutil.ParseDate(fromS)
	if e1 != nil {
		return time.Time{}, time.Time{}, apperr.Validation("起始日期格式错误", apperr.FieldError{Field: "from", Message: "yyyy-MM-dd", Code: "invalid_format"})
	}
	t, e2 := timeutil.ParseDate(toS)
	if e2 != nil {
		return time.Time{}, time.Time{}, apperr.Validation("结束日期格式错误", apperr.FieldError{Field: "to", Message: "yyyy-MM-dd", Code: "invalid_format"})
	}
	if t.Before(f) {
		return time.Time{}, time.Time{}, apperr.Validation("日期范围颠倒", apperr.FieldError{Field: "to", Message: "不得早于 from", Code: "invalid"})
	}
	if timeutil.DaysUntil(f, t) > 31 {
		return time.Time{}, time.Time{}, apperr.Validation("范围过大", apperr.FieldError{Field: "to", Message: "最多 31 天", Code: "out_of_range"})
	}
	return f, t, nil
}

func slotOut(sl model.MealSlot) dto.SlotOut {
	out := dto.SlotOut{
		ID: sl.ID, Date: timeutil.FormatDate(sl.PlanDate), Slot: sl.Slot,
		RecipeID: sl.RecipeID, ServingsMultiplier: sl.ServingsMultiplier, SortOrder: sl.SortOrder,
	}
	if sl.Recipe != nil {
		out.Recipe = RecipeOut(*sl.Recipe)
	}
	return out
}

func pantryFromInput(userID uint, in dto.PantryIn) (*model.PantryItem, error) {
	if in.IngredientID == 0 {
		return nil, apperr.Required("ingredient_id")
	}
	if strings.TrimSpace(in.Unit) == "" {
		return nil, apperr.Required("unit")
	}
	if in.Quantity <= 0 {
		return nil, apperr.Validation("数量必须大于 0", apperr.FieldError{Field: "quantity", Message: "必须 > 0", Code: "out_of_range"})
	}
	stocked, err := timeutil.ParseDate(in.StockedAt)
	if err != nil {
		return nil, apperr.Validation("入库日期格式错误", apperr.FieldError{Field: "stocked_at", Message: "yyyy-MM-dd", Code: "invalid_format"})
	}
	exp, err := timeutil.ParseDate(in.ExpiresAt)
	if err != nil {
		return nil, apperr.Validation("保质期格式错误", apperr.FieldError{Field: "expires_at", Message: "yyyy-MM-dd", Code: "invalid_format"})
	}
	if exp.Before(stocked) {
		return nil, apperr.Validation("保质期早于入库日", apperr.FieldError{Field: "expires_at", Message: "不得早于入库日", Code: "invalid"})
	}
	return &model.PantryItem{
		UserID: userID, IngredientID: in.IngredientID, Quantity: in.Quantity,
		Unit: strings.TrimSpace(in.Unit), StockedAt: stocked, ExpiresAt: exp,
	}, nil
}

func pantryOut(r model.PantryItem, today time.Time) dto.PantryOut {
	name, stall := "", ""
	if r.Ingredient != nil {
		name = r.Ingredient.Name
		stall = r.Ingredient.Stall
	}
	days := timeutil.DaysUntil(today, r.ExpiresAt)
	status := "ok"
	if days < 0 {
		status = "expired"
	} else if days <= 3 {
		status = "soon"
	}
	return dto.PantryOut{
		ID: r.ID, IngredientID: r.IngredientID, IngredientName: name, Stall: stall,
		Quantity: r.Quantity, Unit: r.Unit, StockedAt: timeutil.FormatDate(r.StockedAt),
		ExpiresAt: timeutil.FormatDate(r.ExpiresAt), Status: status, DaysLeft: days,
	}
}
