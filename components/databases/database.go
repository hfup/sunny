package databases

import (
	"context"
	"errors"

	"github.com/hfup/sunny/types"
	"gorm.io/gorm"
)

type DBRouterFunc func(key string) (*gorm.DB, error)
type DBInitHandler func(ctx context.Context, opt any) ([]*types.DBConfig, error) // 数据库初始化

// 数据库管理器接口
type DatabaseClientMangerInf interface {
	types.SubServiceInf
	GetDBFromKey(key string) (*gorm.DB, error)
	SetRouterHandler(routerHandler DBRouterFunc) // 设置路由方法
}

// 数据库管理器
type DatabaseClientManager struct {
	initHandler  DBInitHandler
	dbRouterFunc DBRouterFunc

	dbMap     map[string]*gorm.DB
	defaultDB *gorm.DB
}

func NewDatabaseClientManager(initHandler DBInitHandler) *DatabaseClientManager {
	return &DatabaseClientManager{
		initHandler: initHandler,
		dbMap:       make(map[string]*gorm.DB),
	}
}

// 设置数据库路由函数
// 参数：
//   - dbRouterFunc 数据库路由函数
func (d *DatabaseClientManager) SetDBRouterFunc(dbRouterFunc DBRouterFunc) {
	d.dbRouterFunc = dbRouterFunc
}

// 启动数据库管理器
func (d *DatabaseClientManager) Start(ctx context.Context, args any, resultChan chan<- types.Result[any]) {
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
func (d *DatabaseClientManager) GetDBFromKey(key string) (*gorm.DB, error) {
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
func (d *DatabaseClientManager) GetDefaultDB() (*gorm.DB, error) {
	if d.defaultDB == nil {
		return nil, errors.New("default database not found")
	}
	return d.defaultDB, nil
}



type LocalDatabaseClientManager struct {
	dbConfigs []*types.DBConfig
	dbMap     map[string]*gorm.DB
	defaultDB *gorm.DB
	dbRouterFunc DBRouterFunc
}

func NewLocalDatabaseClientManager(dbConfigs []*types.DBConfig) *LocalDatabaseClientManager {
	return &LocalDatabaseClientManager{
		dbConfigs: dbConfigs,
		dbMap: make(map[string]*gorm.DB),
	}
}

func (d *LocalDatabaseClientManager) SetRouterHandler(handler DBRouterFunc) {
	d.dbRouterFunc = handler
}

// 启动本地数据库管理器
func (d *LocalDatabaseClientManager) Start(ctx context.Context, args any, resultChan chan<- types.Result[any]) {
	if len(d.dbConfigs) == 0 {
		resultChan <- types.Result[any]{
			ErrCode: 1,
			Message: "database manager start error: db configs is empty",
		}
		return
	}
	isSingle := false
	if len(d.dbConfigs) == 1 {
		isSingle = true
	}
	for _, dbConfig := range d.dbConfigs {
		db, err := MysqlConnect(dbConfig)
		if err != nil {
			resultChan <- types.Result[any]{
				ErrCode: 1,
				Message: "database manager start error: " + err.Error(),
			}
			return
		}
		if isSingle {
			d.defaultDB = db
			d.dbMap["default"] = db
			break
		}else{
			d.dbMap[dbConfig.DbId] = db
		}
	}
	// 启动成功
	resultChan <- types.Result[any]{
		ErrCode: 0,
		Message: "database manager start success",
	}
}


// 获取数据库实例
func (d *LocalDatabaseClientManager) GetDBFromKey(key string) (*gorm.DB, error) {
	if key == "default" {
		return  d.defaultDB,nil
	}
	if d.dbRouterFunc != nil {
		return d.dbRouterFunc(key)
	}
	db, ok := d.dbMap[key]
	if !ok {
		return nil, errors.New("database not found")
	}
	return db, nil
}


// 遇到错误 终止整个程序执行
func (d *LocalDatabaseClientManager) IsErrorStop() bool {
	return  true
}

func (d *LocalDatabaseClientManager) ServiceName() string {
	return  "local database manager service"
}

