// models/user.go
package models

type User struct {
    ID    int    `json:"id"`
    Username  string `json:"username"`
    Email string `json:"email"`
	CompanyName string `json:"company_name"`
	
}