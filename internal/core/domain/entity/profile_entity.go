package entity

type UserHttpClientResponse struct {
	Message string              `json:"message"`
	Data    ProfileHttpResponse `json:"data"`
}

type ProfileHttpResponse struct {
	RoleName string `json:"role"`
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Lat      string `json:"lat"`
	Lng      string `json:"lng"`
	Address  string `json:"address"`
	Photo    string `json:"photo"`
}
