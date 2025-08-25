package config

import (
	"os"
	"time"

	"github.com/habbazettt/muraja-server/models"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDB() *gorm.DB {
	LoadEnv()
	InitLogger()

	dsn := os.Getenv("DB_URL")
	logrus.Info("Berhasil Menggunakan database production")

	gormLogger := logger.New(
		logrus.StandardLogger(),
		logger.Config{
			LogLevel: logger.Info,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: false,
		Logger:      gormLogger,
	})
	if err != nil {
		logrus.WithError(err).Fatal("❌ Gagal terhubung ke database!")
	}

	sqlDB, err := db.DB()
	if err != nil {
		logrus.WithError(err).Fatal("❌ Gagal mendapatkan database instance!")
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logrus.Info("✅ Berhasil terhubung ke database!")
	DB = db

	return DB
}

func MigrateDB() {
	if DB == nil {
		logrus.Fatal("❌ Database belum terhubung! Jalankan ConnectDB() terlebih dahulu.")
	}

	err := DB.AutoMigrate(
		&models.User{},
		&models.LogHarian{},
		&models.DetailLog{},
		&models.JadwalPersonal{},
		&models.JadwalRekomendasi{},
	)
	if err != nil {
		logrus.WithError(err).Error("❌ Gagal melakukan migrasi database!")
	}

	logrus.Info("✅ Database berhasil dimigrasi!")
}

func CloseDB() {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			logrus.WithError(err).Error("❌ Gagal mendapatkan database instance untuk ditutup!")
			return
		}
		if err := sqlDB.Close(); err != nil {
			logrus.WithError(err).Error("❌ Gagal menutup koneksi database!")
		} else {
			logrus.Info("✅ Koneksi database berhasil ditutup!")
		}
	}
}
