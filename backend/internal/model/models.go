// Package model 是 GoCooking 的持久化模型。
//
// User              JWT 登录主体，密码只存 bcrypt
// Ingredient        食材主数据：双维度分类 + 别名 JSON + 默认常备标记
// Recipe / RecipeItem  菜谱及用料；UserID=nil 表示系统种子
// MealSlot          七天×三餐槽位，一份菜谱可重复拖入
// PantryItem        冰箱批次，扣减按用餐日 FEFO
// StapleOverride    用户级常备开关，覆盖 IsStapleDefault
// ShoppingCheck     清单勾选与「加回常备」的持久化键
package model

import (
	"time"

	"gocooking/pkg/timeutil"

	"gorm.io/gorm"
)

const (
	SlotBreakfast = "breakfast"
	SlotLunch     = "lunch"
	SlotDinner    = "dinner"

	DimWeight        = "weight"
	DimVolume        = "volume"
	DimCount         = "count"
	DimDimensionless = "dimensionless"
	DimUnknown       = "unknown"

	StallVeg    = "蔬菜摊"
	StallMeat   = "肉摊"
	StallAqua   = "水产摊"
	StallSpice  = "干货调料摊"
	StallDairy  = "蛋奶摊"

	CatLeaf   = "叶菜"
	CatRoot   = "根茎"
	CatMelon  = "瓜果"
	CatMushroom = "菌菇"
	CatBean   = "豆制品"
	CatMeat   = "肉类"
	CatAqua   = "水产"
	CatEgg    = "蛋奶"
	CatDry    = "干货"
	CatSpice  = "调料"
	CatOther  = "其他"
)

var ValidSlots = map[string]bool{
	SlotBreakfast: true,
	SlotLunch:     true,
	SlotDinner:    true,
}

var ValidStalls = map[string]bool{
	StallVeg: true, StallMeat: true, StallAqua: true, StallSpice: true, StallDairy: true,
}

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"size:128;not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	now := timeutil.Now()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	u.UpdatedAt = now
	return nil
}

func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = timeutil.Now()
	return nil
}

type Ingredient struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"size:64;uniqueIndex;not null" json:"name"`
	Aliases         string    `gorm:"type:text" json:"aliases"`
	DefaultUnit     string    `gorm:"size:16;not null" json:"default_unit"`
	Dimension       string    `gorm:"size:24;not null;index" json:"dimension"`
	ProduceCategory string    `gorm:"size:32;not null;index" json:"produce_category"`
	Stall           string    `gorm:"size:32;not null;index" json:"stall"`
	IsStapleDefault bool      `gorm:"not null;default:false" json:"is_staple_default"`
	CreatedAt       time.Time `json:"created_at"`
}

type Recipe struct {
	ID         uint         `gorm:"primaryKey" json:"id"`
	UserID     *uint        `gorm:"index" json:"user_id"`
	Name       string       `gorm:"size:128;not null;index" json:"name"`
	CoverURL   string       `gorm:"size:512" json:"cover_url"`
	CuisineTag string       `gorm:"size:32;index" json:"cuisine_tag"`
	Servings   int          `gorm:"not null;default:2" json:"servings"`
	StepsJSON  string       `gorm:"type:text" json:"-"`
	Items      []RecipeItem `gorm:"constraint:OnDelete:CASCADE" json:"items"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

func (r *Recipe) BeforeCreate(tx *gorm.DB) error {
	now := timeutil.Now()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	r.UpdatedAt = now
	return nil
}

func (r *Recipe) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = timeutil.Now()
	return nil
}

type RecipeItem struct {
	ID           uint        `gorm:"primaryKey" json:"id"`
	RecipeID     uint        `gorm:"index;not null" json:"recipe_id"`
	IngredientID uint        `gorm:"index;not null" json:"ingredient_id"`
	Ingredient   *Ingredient `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"ingredient,omitempty"`
	Quantity     float64     `gorm:"not null" json:"quantity"`
	Unit         string      `gorm:"size:16;not null" json:"unit"`
	Optional     bool        `gorm:"not null;default:false" json:"optional"`
}

type MealSlot struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	UserID             uint      `gorm:"index;not null" json:"user_id"`
	PlanDate           time.Time `gorm:"type:date;index;not null" json:"plan_date"`
	Slot               string    `gorm:"size:16;not null;index" json:"slot"`
	RecipeID           uint      `gorm:"index;not null" json:"recipe_id"`
	Recipe             *Recipe   `json:"recipe,omitempty"`
	ServingsMultiplier float64   `gorm:"not null;default:1" json:"servings_multiplier"`
	SortOrder          int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt          time.Time `json:"created_at"`
}

type PantryItem struct {
	ID           uint        `gorm:"primaryKey" json:"id"`
	UserID       uint        `gorm:"index;not null" json:"user_id"`
	IngredientID uint        `gorm:"index;not null" json:"ingredient_id"`
	Ingredient   *Ingredient `json:"ingredient,omitempty"`
	Quantity     float64     `gorm:"not null" json:"quantity"`
	Unit         string      `gorm:"size:16;not null" json:"unit"`
	StockedAt    time.Time   `gorm:"type:date;not null" json:"stocked_at"`
	ExpiresAt    time.Time   `gorm:"type:date;not null;index" json:"expires_at"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

func (p *PantryItem) BeforeCreate(tx *gorm.DB) error {
	now := timeutil.Now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	return nil
}

func (p *PantryItem) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = timeutil.Now()
	return nil
}

type StapleOverride struct {
	ID           uint `gorm:"primaryKey"`
	UserID       uint `gorm:"uniqueIndex:ux_staple_user_ing;not null"`
	IngredientID uint `gorm:"uniqueIndex:ux_staple_user_ing;not null"`
	Enabled      bool `gorm:"not null"`
	UpdatedAt    time.Time
}

type ShoppingCheck struct {
	ID           uint   `gorm:"primaryKey"`
	UserID       uint   `gorm:"uniqueIndex:ux_check_key;not null"`
	RangeFrom    string `gorm:"size:16;uniqueIndex:ux_check_key;not null"`
	RangeTo      string `gorm:"size:16;uniqueIndex:ux_check_key;not null"`
	IngredientID uint   `gorm:"uniqueIndex:ux_check_key;not null"`
	Unit         string `gorm:"size:16;uniqueIndex:ux_check_key;not null"`
	Dimension    string `gorm:"size:24;uniqueIndex:ux_check_key;not null"`
	Checked      bool   `gorm:"not null;default:false"`
	Restored     bool   `gorm:"not null;default:false"`
	UpdatedAt    time.Time
}

func SlotLabel(slot string) string {
	switch slot {
	case SlotBreakfast:
		return "早餐"
	case SlotLunch:
		return "午餐"
	case SlotDinner:
		return "晚餐"
	default:
		return slot
	}
}
