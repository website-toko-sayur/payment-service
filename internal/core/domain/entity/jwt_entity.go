package entity

type JwtUserData struct {
	CreatedAt string `json:"created_at"`
	Email     string `json:"email"`
	LoggedIn  bool   `json:"logged_in"`
	Name      string `json:"name"`
	Token     string `json:"token"`
	UserID    int64  `json:"user_id"`
	RoleName  string `json:"role_name"`
}
