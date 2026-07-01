package dto

type ForgotPasswordDTO struct {
	Email string `json:"email" binding:"required"`
}

type ResetPasswordDTO struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required"`
}
