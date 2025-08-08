package dto

type ChangePasswordDTO struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}
