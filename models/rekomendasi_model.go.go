package models

import "time"

type JadwalRekomendasi struct {
	ID                        uint     `gorm:"primaryKey" json:"id"`
	UserID                    uint     `gorm:"not null" json:"user_id"`
	State                     string   `gorm:"not null" json:"state"`
	RekomendasiJadwal         string   `gorm:"not null" json:"rekomendasi_jadwal"`
	TipeRekomendasi           string   `gorm:"not null" json:"tipe_rekomendasi"`
	EstimasiQValue            *float64 `gorm:"null" json:"estimasi_q_value"`
	PersentaseEfektifHistoris *float64 `gorm:"null" json:"persentase_efektif_historis"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;" json:"user"`
}
