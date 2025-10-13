package databases

import (
	"context"
	"errors"

	"github.com/hfup/sunny/types"
	"gorm.io/gorm"
	"github.com/sirupsen/logrus"
)


type DBRouterFunc func(key string) (string, error) // key -> areaKey -> gorm.DB 数据库实例

// 数据库管理器接口
type DatabaseClientMangerInf interface {
	types.RunAbleInf // 初始化操作
	GetDBFromKey(key string) (*gorm.DB, error) // key -> gorm.DB 数据库实例
	SetRouterHandler(routerHandler DBRouterFunc) // 设置路由处理函数
	AddDBConfigs(dbConfigs []*types.DatabaseInfo) // 添加数据库配置
	IsDebug() bool // 是否开启调试
}

// 数据库管理器
type DatabaseClientManager struct {
	dbRouterFunc DBRouterFunc
	dbConfigs []*types.DatabaseInfo
	dbMap     map[string]*gorm.DB
	isDebug bool
}

func NewDatabaseClientManager(isDebug bool) *DatabaseClientManager {
	return &DatabaseClientManager{
		dbMap: make(map[string]*gorm.DB), // key -> gorm.DB 数据库实例
		isDebug: isDebug,
	}
}


func (d *DatabaseClientManager) IsDebug() bool {
	return d.isDebug
}

// 设置数据库路由函数
// 参数：
//   - dbRouterFunc 数据库路由函数
func (d *DatabaseClientManager) SetRouterHandler(dbRouterFunc DBRouterFunc) {
	d.dbRouterFunc = dbRouterFunc
}

// 添加数据库配置
// 参数：
//   - dbConfigs 数据库配置
// 返回：
//   - 错误
func (d *DatabaseClientManager) AddDBConfigs(dbConfigs []*types.DatabaseInfo) {
	d.dbConfigs = append(d.dbConfigs, dbConfigs...)
}

// 是否开启调试
// 返回：
//   - 是否开启调试
func (d *DatabaseClientManager) Debug() bool {
	return d.isDebug
}


// 启动数据库管理器
func (d *DatabaseClientManager) Run(ctx context.Context, app any) error {
	if len(d.dbConfigs) == 0 {
		logrus.Info("database manager start error: db configs is empty") // 没有配置数据库
		return nil
	}
	isSingle := len(d.dbConfigs) == 1
	for _, dbConfig := range d.dbConfigs {
		if isSingle && dbConfig.AreaKey == "" { // 单个数据库配置 没有配置areaKey 则设置为default
			dbConfig.AreaKey = "default"
		}
		if dbConfig.AreaKey == "" { // 多个数据库配置 没有配置areaKey 则返回错误
			return errors.New("database manager start error: db config areaKey is empty")
		}
		db, err := MysqlConnect(dbConfig)
		if err != nil {
			return err
		}
		if d.Debug() {
			db = db.Debug()
		}
		d.dbMap[dbConfig.AreaKey] = db
	}
	return nil
}

// 根据key获取数据库
func (d *DatabaseClientManager) GetDBFromKey(key string) (*gorm.DB, error) {
	if d.dbRouterFunc != nil {
		areaKey, err := d.dbRouterFunc(key)
		if err != nil {
			return nil, err
		}
		db, ok := d.dbMap[areaKey]
		if !ok {
			return nil, errors.New("database not found")
		}
		return db, nil
	}
	db, ok := d.dbMap[key]
	if !ok {
		return nil, errors.New("database not found")
	}
	return db, nil
}

// 获取当前激活的数据库
func (d *DatabaseClientManager) GetDefaultDB() (*gorm.DB, error) {
	db, ok := d.dbMap["default"]
	if !ok {
		return nil, errors.New("default database not found")
	}
	return db, nil
}


func (d *DatabaseClientManager) Description() string {
	return "数据库管理器"
}






