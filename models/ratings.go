package models

type Ratings struct {
	Restaurant_id int   `json:"restaurant_id"`
	Reviews       []int `json:"reviews"`
}
