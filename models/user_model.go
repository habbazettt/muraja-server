package models

import "time"

type User struct {
	ID                   uint   `gorm:"primaryKey" json:"id"`
	Nama                 string `gorm:"type:varchar(255);not null" json:"nama"`
	Email                string `gorm:"type:varchar(255);not null;unique" json:"email"`
	Password             string `gorm:"not null" json:"-"`
	IsDataMurojaahFilled bool   `gorm:"default:false" json:"is_data_murojaah_filled"`
	UserType             string `gorm:"type:varchar(255);not null" json:"user_type"`

	JadwalPersonal     *JadwalPersonal     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;" json:"jadwal_personal,omitempty"`
	LogHarians         []LogHarian         `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;" json:"log_harians,omitempty"`
	JadwalRekomendasis []JadwalRekomendasi `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;" json:"jadwal_rekomendasi,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
