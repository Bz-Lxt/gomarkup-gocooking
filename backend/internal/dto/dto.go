// Package dto 定义 HTTP 契约。日期一律 yyyy-MM-dd，时间戳 yyyy-MM-dd HH:mm:ss（北京时间）。
// 列表走 {data, meta}，写操作走 {data}，错误走 {error:{code,message,details}}。
package dto

// 下面的结构与 docs/API.md 保持同步。若增减字段必须同时改前端 types.ts。


type Envelope struct {
	Data any `json:"data"`
}

type ListEnvelope struct {
	Data any      `json:"data"`
	Meta ListMeta `json:"meta"`
}

type ListMeta struct {
	Total   int64 `json:"total"`
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
}

type ErrorBody struct {
	Error ErrorObj `json:"error"`
}

type ErrorObj struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResp struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

type RecipeItemIn struct {
	IngredientID uint    `json:"ingredient_id"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
	Optional     bool    `json:"optional"`
}

type RecipeIn struct {
	Name       string          `json:"name"`
	CoverURL   string          `json:"cover_url"`
	CuisineTag string          `json:"cuisine_tag"`
	Servings   int             `json:"servings"`
	Steps      []string        `json:"steps"`
	Items      []RecipeItemIn  `json:"items"`
}

type RecipeItemOut struct {
	ID             uint    `json:"id"`
	IngredientID   uint    `json:"ingredient_id"`
	IngredientName string  `json:"ingredient_name"`
	Quantity       float64 `json:"quantity"`
	Unit           string  `json:"unit"`
	Optional       bool    `json:"optional"`
}

type RecipeOut struct {
	ID         uint            `json:"id"`
	UserID     *uint           `json:"user_id"`
	Name       string          `json:"name"`
	CoverURL   string          `json:"cover_url"`
	CuisineTag string          `json:"cuisine_tag"`
	Servings   int             `json:"servings"`
	Steps      []string        `json:"steps"`
	Items      []RecipeItemOut `json:"items"`
	CreatedAt  string          `json:"created_at"`
	UpdatedAt  string          `json:"updated_at"`
}

type IngredientOut struct {
	ID              uint     `json:"id"`
	Name            string   `json:"name"`
	Aliases         []string `json:"aliases"`
	DefaultUnit     string   `json:"default_unit"`
	Dimension       string   `json:"dimension"`
	ProduceCategory string   `json:"produce_category"`
	Stall           string   `json:"stall"`
	IsStapleDefault bool     `json:"is_staple_default"`
}

type SlotIn struct {
	Date               string  `json:"date"`
	Slot               string  `json:"slot"`
	RecipeID           uint    `json:"recipe_id"`
	ServingsMultiplier float64 `json:"servings_multiplier"`
}

type SlotPatch struct {
	Date               *string  `json:"date"`
	Slot               *string  `json:"slot"`
	ServingsMultiplier *float64 `json:"servings_multiplier"`
	SortOrder          *int     `json:"sort_order"`
}

type SlotOut struct {
	ID                 uint      `json:"id"`
	Date               string    `json:"date"`
	Slot               string    `json:"slot"`
	RecipeID           uint      `json:"recipe_id"`
	Recipe             RecipeOut `json:"recipe"`
	ServingsMultiplier float64   `json:"servings_multiplier"`
	SortOrder          int       `json:"sort_order"`
}

type WeekPlanOut struct {
	WeekStart string    `json:"week_start"`
	WeekEnd   string    `json:"week_end"`
	Slots     []SlotOut `json:"slots"`
}

type WeekOp struct {
	Week string `json:"week"`
}

type PantryIn struct {
	IngredientID uint    `json:"ingredient_id"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
	StockedAt    string  `json:"stocked_at"`
	ExpiresAt    string  `json:"expires_at"`
}

type PantryOut struct {
	ID             uint    `json:"id"`
	IngredientID   uint    `json:"ingredient_id"`
	IngredientName string  `json:"ingredient_name"`
	Stall          string  `json:"stall"`
	Quantity       float64 `json:"quantity"`
	Unit           string  `json:"unit"`
	StockedAt      string  `json:"stocked_at"`
	ExpiresAt      string  `json:"expires_at"`
	Status         string  `json:"status"`
	DaysLeft       int     `json:"days_left"`
}

type GenerateReq struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type CheckReq struct {
	From         string `json:"from"`
	To           string `json:"to"`
	IngredientID uint   `json:"ingredient_id"`
	Unit         string `json:"unit"`
	Dimension    string `json:"dimension"`
	Checked      bool   `json:"checked"`
}

type RestoreReq struct {
	From         string `json:"from"`
	To           string `json:"to"`
	IngredientID uint   `json:"ingredient_id"`
	Unit         string `json:"unit"`
	Dimension    string `json:"dimension"`
}

type StapleItem struct {
	IngredientID   uint   `json:"ingredient_id"`
	Name           string `json:"name"`
	Enabled        bool   `json:"enabled"`
	DefaultEnabled bool   `json:"default_enabled"`
}

type StaplesPut struct {
	Items []struct {
		IngredientID uint `json:"ingredient_id"`
		Enabled      bool `json:"enabled"`
	} `json:"items"`
}

type StapleAdd struct {
	IngredientID uint `json:"ingredient_id"`
	Enabled      bool `json:"enabled"`
}

type SourceOut struct {
	Date       string  `json:"date"`
	Slot       string  `json:"slot"`
	SlotLabel  string  `json:"slot_label"`
	RecipeName string  `json:"recipe_name"`
	Quantity   float64 `json:"quantity"`
	Unit       string  `json:"unit"`
}

type ListItemOut struct {
	IngredientID uint        `json:"ingredient_id"`
	Name         string      `json:"name"`
	Quantity     float64     `json:"quantity"`
	Unit         string      `json:"unit"`
	CheckUnit    string      `json:"check_unit"`
	Display      string      `json:"display"`
	Dimension    string      `json:"dimension"`
	NeedsReview  bool        `json:"needs_review"`
	Checked      bool        `json:"checked"`
	Filtered     bool        `json:"filtered"`
	Sources      []SourceOut `json:"sources"`
	Siblings     []ListItemOut `json:"siblings,omitempty"`
	ProduceCategory string   `json:"produce_category"`
	Stall        string      `json:"stall"`
}

type GroupOut struct {
	Key   string        `json:"key"`
	Items []ListItemOut `json:"items"`
}

type DeductOut struct {
	IngredientID uint    `json:"ingredient_id"`
	Name         string  `json:"name"`
	FromPantry   float64 `json:"from_pantry"`
	Unit         string  `json:"unit"`
}

type AlertOut struct {
	IngredientID uint   `json:"ingredient_id"`
	Name         string `json:"name"`
	ExpiresAt    string `json:"expires_at"`
	Message      string `json:"message"`
}

type ShoppingOut struct {
	From            string      `json:"from"`
	To              string      `json:"to"`
	GroupsByStall   []GroupOut  `json:"groups_by_stall"`
	GroupsByProduce []GroupOut  `json:"groups_by_produce"`
	Filtered        []ListItemOut `json:"filtered"`
	ExpiryAlerts    []AlertOut  `json:"expiry_alerts"`
	Deducted        []DeductOut `json:"deducted"`
}
