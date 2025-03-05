package entities

type Currency struct {
	Code    uint   `json:"code" gorm:"primaryKey;autoIncrement:false"`
	Name    string `json:"name" gorm:"index"`
	StrCode string `json:"strCode"`

	FireflyId *string `json:"-" gorm:"<-:false;index"`
}

func (c Currency) TableName() string {
	return "currencies"
}
