package dto

type UserResponse struct {
	ID                   uint                    `json:"id"`
	Nama                 string                  `json:"nama"`
	Email                string                  `json:"email"`
	UserType             string                  `json:"user_type"`
	IsDataMurojaahFilled bool                    `json:"is_data_murojaah_filled"`
	JadwalPersonal       *JadwalPersonalResponse `json:"jadwal_personal,omitempty"`
}

type UpdateUserRequest struct {
	Nama     *string `json:"nama,omitempty"`
	Email    *string `json:"email,omitempty"`
	UserType *string `json:"user_type,omitempty"`
}
