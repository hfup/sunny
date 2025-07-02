package databases

import (
	"context"
	"errors"

	"github.com/hfup/sunny/types"
	"gorm.io/gorm"
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

type DBConfig struct {
	DatabaseInfo *DatabaseInfo
	DbId         string `yaml:"db_id" json:"db_id"`                   // 数据库id 唯一
	MaxIdelConns int    `yaml:"max_idel_conns" json:"max_idel_conns"` // 最大空闲连接数
	MaxOpenConns int    `yaml:"max_open_conns" json:"max_open_conns"` // 最大打开连接数
	MaxLifetime  int    `yaml:"max_lifetime" json:"max_lifetime"`     // 连接最大生命周期
}

type DBRouterFunc func(key string) (*gorm.DB, error)
type DBInitHandler func(ctx context.Context, opt any) ([]*DBConfig, error) // 数据库初始化

// 数据库管理器接口
type DatabaseMangerInf interface {
	types.SubServiceInf
	GetDBFromKey(key string) (*gorm.DB, error)
}

// 数据库管理器
type DatabaseManager struct {
	initHandler  DBInitHandler
	dbRouterFunc DBRouterFunc

	dbMap     map[string]*gorm.DB
	defaultDB *gorm.DB
}

func NewDatabaseManager(initHandler DBInitHandler) *DatabaseManager {
	return &DatabaseManager{
		initHandler: initHandler,
		dbMap:       make(map[string]*gorm.DB),
	}
}

// 设置数据库路由函数
// 参数：
//   - dbRouterFunc 数据库路由函数
func (d *DatabaseManager) SetDBRouterFunc(dbRouterFunc DBRouterFunc) {
	d.dbRouterFunc = dbRouterFunc
}

// 启动数据库管理器
func (d *DatabaseManager) Start(ctx context.Context, args any, resultChan chan<- types.Result[any]) {
	dbConfigs, err := d.initHandler(ctx, args)
	if err != nil {
		resultChan <- types.Result[any]{
			ErrCode: 1,
			Message: "database manager start error: " + err.Error(),
		}
		return
	}
	for _, dbConfig := range dbConfigs {
		db, err := MysqlConnect(dbConfig)
		if err != nil {
			resultChan <- types.Result[any]{
				ErrCode: 1,
				Message: "database manager start error: " + err.Error(),
			}
		}
		if dbConfig.DbId == "default" { // 默认数据库
			d.defaultDB = db
		}
		d.dbMap[dbConfig.DbId] = db
	}
	// 启动成功
	resultChan <- types.Result[any]{
		ErrCode: 0,
		Message: "database manager start success",
	}
}

// 根据key获取数据库
func (d *DatabaseManager) GetDBFromKey(key string) (*gorm.DB, error) {
	if d.dbRouterFunc != nil {
		return d.dbRouterFunc(key)
	}
	db, ok := d.dbMap[key]
	if !ok {
		return nil, errors.New("database not found")
	}
	return db, nil
}

// 获取当前激活的数据库
func (d *DatabaseManager) GetDefaultDB() (*gorm.DB, error) {
	if d.defaultDB == nil {
		return nil, errors.New("default database not found")
	}
	return d.defaultDB, nil
}
