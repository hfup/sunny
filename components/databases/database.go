package databases

import "gorm.io/gorm"


// 数据库信息
type DatabaseInfo struct {
	Driver   string `yaml:"driver" json:"driver"` // mysql, postgres, sqlite
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	User     string `yaml:"user" json:"user"`
	Password string `yaml:"password" json:"password"`
	DbName   string `yaml:"db_name" json:"db_name"`
	Charset  string `yaml:"charset" json:"charset"` // 字符集
}


type DBRouterFunc func(key string) (*gorm.DB, error)


// 数据库管理器接口
type DatabaseMangerInf interface {
	
}










