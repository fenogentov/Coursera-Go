package jsn

//easyjson:json
type User struct {
	Browsers []string `json:"browsers"`
	Name     string   `json:"name"`
	Mail     string   `json:"email"`
}
