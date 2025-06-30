package databases

import (
	"context"
	"gorm.io/gorm"
	"github.com/hfup/sunny"
)


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
type DBInitHandler func(ctx context.Context) ([]*gorm.DB, error) // 数据库初始化


// 数据库管理器接口
type DatabaseMangerInf interface {
	sunny.SubServiceInf
}


// 数据库管理器
type DatabaseManager struct {
	initHandler DBInitHandler
	dbRouterFunc DBRouterFunc
}

// 启动数据库管理器
func (d *DatabaseManager) Start(ctx context.Context,args any,resultChan chan<- sunny.Result) {
	dbs,err := d.initHandler(ctx)
	if err != nil {
		resultChan <- sunny.Result{
			Code: 1,
			Msg: "database manager start error: " + err.Error(),
		}
	}
	logrus.Infof("database manager start success, db count: %d", len(dbs))
}









