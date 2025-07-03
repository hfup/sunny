package databases

import (
	"fmt"

	"github.com/hfup/sunny/types"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
)


func MysqlConnect(dataInfo *types.DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		dataInfo.DatabaseInfo.User, dataInfo.DatabaseInfo.Password, dataInfo.DatabaseInfo.Host, dataInfo.DatabaseInfo.Port, dataInfo.DatabaseInfo.DbName, dataInfo.DatabaseInfo.Charset)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if dataInfo.MaxIdelConns > 0 {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		sqlDB.SetMaxIdleConns(dataInfo.MaxIdelConns)
	}
	if dataInfo.MaxOpenConns > 0 {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		sqlDB.SetMaxOpenConns(dataInfo.MaxOpenConns)
	}
	if dataInfo.MaxLifetime > 0 {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		sqlDB.SetConnMaxLifetime(time.Duration(dataInfo.MaxLifetime) * time.Second)
	}
	return db, nil
}