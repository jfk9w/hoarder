package entities

type User struct {
	Phone string `gorm:"primaryKey"`
	Name  string `gorm:"index"`
}

func (u User) TableName() string {
	return "users"
}

type Session struct {
	UserPhone string `json:"-" gorm:"primaryKey"`
	User      User   `json:"-" gorm:"constraint:OnDelete:CASCADE"`

	ID string
}
