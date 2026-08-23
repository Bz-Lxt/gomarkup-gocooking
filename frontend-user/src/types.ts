export type RecipeItem = {
  id: number;
  ingredient_id: number;
  ingredient_name: string;
  quantity: number;
  unit: string;
  optional: boolean;
};

export type Recipe = {
  id: number;
  user_id?: number | null;
  name: string;
  cover_url: string;
  cuisine_tag: string;
  servings: number;
  steps: string[];
  items: RecipeItem[];
};

export type Ingredient = {
  id: number;
  name: string;
  aliases: string[];
  default_unit: string;
  dimension: string;
  produce_category: string;
  stall: string;
  is_staple_default: boolean;
};

export type Slot = {
  id: number;
  date: string;
  slot: "breakfast" | "lunch" | "dinner";
  recipe_id: number;
  recipe: Recipe;
  servings_multiplier: number;
  sort_order: number;
};

export type WeekPlan = {
  week_start: string;
  week_end: string;
  slots: Slot[];
};

export type Pantry = {
  id: number;
  ingredient_id: number;
  ingredient_name: string;
  stall: string;
  quantity: number;
  unit: string;
  stocked_at: string;
  expires_at: string;
  status: "ok" | "soon" | "expired";
  days_left: number;
};

export type Source = {
  date: string;
  slot: string;
  slot_label: string;
  recipe_name: string;
  quantity: number;
  unit: string;
};

export type ListItem = {
  ingredient_id: number;
  name: string;
  quantity: number;
  unit: string;
  check_unit: string;
  display: string;
  dimension: string;
  needs_review: boolean;
  checked: boolean;
  filtered: boolean;
  sources: Source[];
  siblings?: ListItem[];
  produce_category: string;
  stall: string;
};

export type Group = { key: string; items: ListItem[] };

export type Shopping = {
  from: string;
  to: string;
  groups_by_stall: Group[];
  groups_by_produce: Group[];
  filtered: ListItem[];
  expiry_alerts: { ingredient_id: number; name: string; expires_at: string; message: string }[];
  deducted: { ingredient_id: number; name: string; from_pantry: number; unit: string }[];
};

export type Staple = {
  ingredient_id: number;
  name: string;
  enabled: boolean;
  default_enabled: boolean;
};
