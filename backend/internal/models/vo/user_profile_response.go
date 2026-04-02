package vo

type UserProfileResponse struct {
	Id           int    `gorm:"primaryKey;autoIncrement" json:"id" form:"id"`
	Username     string `gorm:"size:50;unique;not null" json:"username" form:"username"` // 用户名
	Email        string `gorm:"size:100;unique;not null" json:"email" form:"email"`
	Phone        string `gorm:"size:20;unique" json:"phone" form:"phone"`
	Avatar       string `gorm:"size:255" json:"avatar" form:"avatar"`
	OpenId       string `gorm:"size:255" json:"openId"`
	RegisterTime string `json:"registerTime" form:"registerTime"`
	RootFolderId string `json:"rootFolderId" form:"rootFolderId"`
}
