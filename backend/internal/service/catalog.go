// Package service 分两块：Catalog（用户/菜谱/食材）与 Planner（排期/冰箱/清单）。
// 所有外部输入在进入 GORM 前做存在性、类型和边界校验，禁止依赖调用方“碰巧传对”。
package service

import (
	"context"
	"encoding/json"
	"strings"

	"gocooking/internal/dto"
	"gocooking/internal/model"
	"gocooking/pkg/apperr"
	"gocooking/pkg/timeutil"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Catalog struct {
	DB *gorm.DB
}

func NewCatalog(db *gorm.DB) *Catalog { return &Catalog{DB: db} }

func (s *Catalog) Authenticate(ctx context.Context, username, password string) (*model.User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, apperr.Required("username")
	}
	if password == "" {
		return nil, apperr.Required("password")
	}
	var u model.User
	if err := s.DB.WithContext(ctx).Where("username = ?", username).First(&u).Error; err != nil {
		if apperr.Cancelled(err) {
			return nil, ctx.Err()
		}
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.Unauthorized("用户名或密码错误")
		}
		return nil, apperr.Internal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, apperr.Unauthorized("用户名或密码错误")
	}
	return &u, nil
}

func (s *Catalog) ListIngredients(ctx context.Context, q, stall, category string, page, per int) ([]model.Ingredient, int64, error) {
	page, per = normalizePage(page, per)
	tx := s.DB.WithContext(ctx).Model(&model.Ingredient{})
	if q = strings.TrimSpace(q); q != "" {
		like := "%" + q + "%"
		tx = tx.Where("name ILIKE ? OR aliases ILIKE ?", like, like)
	}
	if stall != "" {
		if !model.ValidStalls[stall] {
			return nil, 0, apperr.Validation("摊位分类不合法", apperr.FieldError{Field: "stall", Message: "未知摊位", Code: "invalid"})
		}
		tx = tx.Where("stall = ?", stall)
	}
	if category != "" {
		tx = tx.Where("produce_category = ?", category)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, wrapQueryErr(err)
	}
	var rows []model.Ingredient
	if err := tx.Order("id asc").Offset((page - 1) * per).Limit(per).Find(&rows).Error; err != nil {
		return nil, 0, wrapQueryErr(err)
	}
	return rows, total, nil
}

func (s *Catalog) ListRecipes(ctx context.Context, userID uint, q, tag string, page, per int) ([]model.Recipe, int64, error) {
	page, per = normalizePage(page, per)
	tx := s.DB.WithContext(ctx).Model(&model.Recipe{}).Where("user_id IS NULL OR user_id = ?", userID)
	if q = strings.TrimSpace(q); q != "" {
		tx = tx.Where("name ILIKE ?", "%"+q+"%")
	}
	if tag = strings.TrimSpace(tag); tag != "" {
		tx = tx.Where("cuisine_tag = ?", tag)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, wrapQueryErr(err)
	}
	var rows []model.Recipe
	if err := tx.Preload("Items.Ingredient").Order("id asc").Offset((page - 1) * per).Limit(per).Find(&rows).Error; err != nil {
		return nil, 0, wrapQueryErr(err)
	}
	return rows, total, nil
}

func (s *Catalog) GetRecipe(ctx context.Context, userID, id uint) (*model.Recipe, error) {
	var r model.Recipe
	err := s.DB.WithContext(ctx).Preload("Items.Ingredient").Where("id = ? AND (user_id IS NULL OR user_id = ?)", id, userID).First(&r).Error
	if err != nil {
		if apperr.Cancelled(err) {
			return nil, ctx.Err()
		}
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.NotFound("菜谱")
		}
		return nil, apperr.Internal(err)
	}
	return &r, nil
}

func (s *Catalog) CreateRecipe(ctx context.Context, userID uint, in dto.RecipeIn) (*model.Recipe, error) {
	r, err := recipeFromInput(in)
	if err != nil {
		return nil, err
	}
	uid := userID
	r.UserID = &uid
	if err := s.validateItems(ctx, in.Items); err != nil {
		return nil, err
	}
	if err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(r).Error; err != nil {
			return err
		}
		items := itemsFromInput(r.ID, in.Items)
		if len(items) > 0 {
			return tx.Create(&items).Error
		}
		return nil
	}); err != nil {
		return nil, wrapQueryErr(err)
	}
	return s.GetRecipe(ctx, userID, r.ID)
}

func (s *Catalog) UpdateRecipe(ctx context.Context, userID, id uint, in dto.RecipeIn) (*model.Recipe, error) {
	cur, err := s.GetRecipe(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if cur.UserID == nil || *cur.UserID != userID {
		return nil, apperr.Conflict("系统菜谱不可直接改写，请先复制为副本")
	}
	next, err := recipeFromInput(in)
	if err != nil {
		return nil, err
	}
	if err := s.validateItems(ctx, in.Items); err != nil {
		return nil, err
	}
	if err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(cur).Updates(map[string]any{
			"name": next.Name, "cover_url": next.CoverURL, "cuisine_tag": next.CuisineTag,
			"servings": next.Servings, "steps_json": next.StepsJSON, "updated_at": timeutil.Now(),
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("recipe_id = ?", id).Delete(&model.RecipeItem{}).Error; err != nil {
			return err
		}
		items := itemsFromInput(id, in.Items)
		if len(items) == 0 {
			return nil
		}
		return tx.Create(&items).Error
	}); err != nil {
		return nil, wrapQueryErr(err)
	}
	return s.GetRecipe(ctx, userID, id)
}

func (s *Catalog) DeleteRecipe(ctx context.Context, userID, id uint) error {
	cur, err := s.GetRecipe(ctx, userID, id)
	if err != nil {
		return err
	}
	if cur.UserID == nil || *cur.UserID != userID {
		return apperr.Conflict("系统菜谱不可删除")
	}
	if err := s.DB.WithContext(ctx).Delete(&model.Recipe{}, id).Error; err != nil {
		return wrapQueryErr(err)
	}
	return nil
}

func (s *Catalog) DuplicateRecipe(ctx context.Context, userID, id uint) (*model.Recipe, error) {
	src, err := s.GetRecipe(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	in := dto.RecipeIn{
		Name: src.Name + "（改良）", CoverURL: src.CoverURL, CuisineTag: src.CuisineTag,
		Servings: src.Servings, Steps: DecodeSteps(src.StepsJSON),
	}
	for _, it := range src.Items {
		in.Items = append(in.Items, dto.RecipeItemIn{
			IngredientID: it.IngredientID, Quantity: it.Quantity, Unit: it.Unit, Optional: it.Optional,
		})
	}
	return s.CreateRecipe(ctx, userID, in)
}

func (s *Catalog) validateItems(ctx context.Context, items []dto.RecipeItemIn) error {
	if len(items) == 0 {
		return apperr.Validation("至少需要一味食材", apperr.FieldError{Field: "items", Message: "不能为空", Code: "required"})
	}
	ids := make([]uint, 0, len(items))
	for i, it := range items {
		if it.IngredientID == 0 {
			return apperr.Validation("食材无效", apperr.FieldError{Field: fieldItems(i, "ingredient_id"), Message: "必填", Code: "required"})
		}
		if strings.TrimSpace(it.Unit) == "" {
			return apperr.Validation("单位必填", apperr.FieldError{Field: fieldItems(i, "unit"), Message: "必填", Code: "required"})
		}
		if it.Quantity < 0 {
			return apperr.Validation("数量不能为负", apperr.FieldError{Field: fieldItems(i, "quantity"), Message: "必须 ≥ 0", Code: "out_of_range"})
		}
		ids = append(ids, it.IngredientID)
	}
	var cnt int64
	if err := s.DB.WithContext(ctx).Model(&model.Ingredient{}).Where("id IN ?", ids).Count(&cnt).Error; err != nil {
		return wrapQueryErr(err)
	}
	if int(cnt) != uniqLen(ids) {
		return apperr.Validation("存在未知食材", apperr.FieldError{Field: "items", Message: "ingredient_id 无效", Code: "invalid"})
	}
	return nil
}

func recipeFromInput(in dto.RecipeIn) (*model.Recipe, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, apperr.Required("name")
	}
	if len(name) > 128 {
		return nil, apperr.Validation("名称过长", apperr.FieldError{Field: "name", Message: "最多 128 字", Code: "too_long"})
	}
	serv := in.Servings
	if serv <= 0 {
		serv = 2
	}
	if serv > 20 {
		return nil, apperr.Validation("份数过大", apperr.FieldError{Field: "servings", Message: "范围 1–20", Code: "out_of_range"})
	}
	raw, _ := json.Marshal(in.Steps)
	return &model.Recipe{
		Name: name, CoverURL: strings.TrimSpace(in.CoverURL), CuisineTag: strings.TrimSpace(in.CuisineTag),
		Servings: serv, StepsJSON: string(raw),
	}, nil
}

func itemsFromInput(recipeID uint, in []dto.RecipeItemIn) []model.RecipeItem {
	out := make([]model.RecipeItem, 0, len(in))
	for _, it := range in {
		out = append(out, model.RecipeItem{
			RecipeID: recipeID, IngredientID: it.IngredientID,
			Quantity: it.Quantity, Unit: strings.TrimSpace(it.Unit), Optional: it.Optional,
		})
	}
	return out
}

func DecodeSteps(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var steps []string
	if err := json.Unmarshal([]byte(raw), &steps); err != nil {
		return []string{raw}
	}
	return steps
}

func DecodeAliases(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var a []string
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		parts := strings.Split(raw, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				out = append(out, t)
			}
		}
		return out
	}
	return a
}

func RecipeOut(r model.Recipe) dto.RecipeOut {
	items := make([]dto.RecipeItemOut, 0, len(r.Items))
	for _, it := range r.Items {
		name := ""
		if it.Ingredient != nil {
			name = it.Ingredient.Name
		}
		items = append(items, dto.RecipeItemOut{
			ID: it.ID, IngredientID: it.IngredientID, IngredientName: name,
			Quantity: it.Quantity, Unit: it.Unit, Optional: it.Optional,
		})
	}
	return dto.RecipeOut{
		ID: r.ID, UserID: r.UserID, Name: r.Name, CoverURL: r.CoverURL, CuisineTag: r.CuisineTag,
		Servings: r.Servings, Steps: DecodeSteps(r.StepsJSON), Items: items,
		CreatedAt: timeutil.FormatDateTime(r.CreatedAt), UpdatedAt: timeutil.FormatDateTime(r.UpdatedAt),
	}
}

func IngredientOut(ing model.Ingredient) dto.IngredientOut {
	return dto.IngredientOut{
		ID: ing.ID, Name: ing.Name, Aliases: DecodeAliases(ing.Aliases),
		DefaultUnit: ing.DefaultUnit, Dimension: ing.Dimension,
		ProduceCategory: ing.ProduceCategory, Stall: ing.Stall, IsStapleDefault: ing.IsStapleDefault,
	}
}

func normalizePage(page, per int) (int, int) {
	if page < 1 {
		page = 1
	}
	if per < 1 {
		per = 50
	}
	if per > 200 {
		per = 200
	}
	return page, per
}

func fieldItems(i int, f string) string {
	return "items." + itoa(i) + "." + f
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [12]byte
	n := 11
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		b[n] = byte('0' + i%10)
		n--
		i /= 10
	}
	if neg {
		b[n] = '-'
		n--
	}
	return string(b[n+1:])
}

func uniqLen(ids []uint) int {
	seen := map[uint]struct{}{}
	for _, id := range ids {
		seen[id] = struct{}{}
	}
	return len(seen)
}

// wrapQueryErr 把 GORM/pqx 返回的 error 归类：
// 若是 context 取消/超时，直接返回 ctx 的错误（让 handler 判断是否需要写响应）；
// 否则包装成 500 内部错误。
func wrapQueryErr(err error) error {
	if apperr.Cancelled(err) {
		return err
	}
	return apperr.Internal(err)
}
