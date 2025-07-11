package models

import (
	"time"
)

type JadwalPersonal struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            uint      `gorm:"unique;not null" json:"user_id"`
	TotalHafalan      int       `gorm:"not null" json:"total_hafalan"`
	Kesibukan         string    `gorm:"type:varchar(255);not null" json:"kesibukan"`
	Jadwal            string    `gorm:"type:varchar(255)" json:"jadwal"`
	EfektifitasJadwal int       `gorm:"not null" json:"efektifitas_jadwal"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;" json:"user"`
}
