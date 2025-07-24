package databases

import (
	"context"
	"errors"

	"github.com/hfup/sunny/types"
	"gorm.io/gorm"
)


type DBRouterFunc func(key string) (*gorm.DB, error)
// 数据库初始化
// map[string]*types.DBConfig 数据库配置 key 为 areaKey
type DBInitHandler func(ctx context.Context, opt any) (map[string]*types.DatabaseInfo, error) // 数据库初始化

// 数据库管理器接口
type DatabaseClientMangerInf interface {
	types.RunAbleInf
	GetDBFromKey(key string) (*gorm.DB, error)
	SetRouterHandler(routerHandler DBRouterFunc) // 设置路由方法
}

// 数据库管理器
type DatabaseClientManager struct {
	initHandler  DBInitHandler
	dbRouterFunc DBRouterFunc

	dbMap     map[string]*gorm.DB
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

func (d *DatabaseClientManager) Description() string {
	return "数据库管理器"
}

// 启动数据库管理器
func (d *DatabaseClientManager) Run(ctx context.Context, app any) error {
	dbConfigs, err := d.initHandler(ctx, app)
	if err != nil {
		return err
	}
	isSingle := len(dbConfigs) == 1
	for key, dbConfig := range dbConfigs {
		if isSingle && dbConfig.AreaKey == "" {
			dbConfig.AreaKey = "default"
		}
		if dbConfig.AreaKey == "" {
			return errors.New("database manager start error: db config areaKey is empty")
		}
		db, err := MysqlConnect(dbConfig)
		if err != nil {
			return err
		}
		d.dbMap[key] = db
	}
	return nil
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
	if d.dbRouterFunc != nil {
		return d.dbRouterFunc("default")
	}
	db, ok := d.dbMap["default"]
	if !ok {
		return nil, errors.New("default database not found")
	}
	return db, nil
}


// 本地数据库管理器
type LocalDatabaseClientManager struct {
	dbConfigs []*types.DatabaseInfo
	dbMap     map[string]*gorm.DB
	dbRouterFunc DBRouterFunc
}

// 本地数据库管理器
func NewLocalDatabaseClientManager(dbConfigs []*types.DatabaseInfo) *LocalDatabaseClientManager {
	return &LocalDatabaseClientManager{
		dbConfigs: dbConfigs,
		dbMap: make(map[string]*gorm.DB),
	}
}

func (d *LocalDatabaseClientManager) SetRouterHandler(handler DBRouterFunc) {
	d.dbRouterFunc = handler
}


// 启动数据库管理器
func (d *LocalDatabaseClientManager) Run(ctx context.Context, app any) error {
	if len(d.dbConfigs) == 0 {
		return errors.New("database manager start error: db configs is empty")
	}
	isSingle := len(d.dbConfigs) == 1
	for _, dbConfig := range d.dbConfigs {
		if isSingle && dbConfig.AreaKey == "" {
			dbConfig.AreaKey = "default"
		}
		if dbConfig.AreaKey == "" {
			return errors.New("database manager start error: db config areaKey is empty")
		}
		db, err := MysqlConnect(dbConfig)
		if err != nil {
			return err
		}
		d.dbMap[dbConfig.AreaKey] = db
	}
	return nil
}

func (d *LocalDatabaseClientManager) Description() string {
	return "本地数据库管理器"
}


// 获取数据库实例
func (d *LocalDatabaseClientManager) GetDBFromKey(key string) (*gorm.DB, error) {
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
	return  "local database 客户端管理器"
}


// 远程请求数据库配置
// 参数:
//   - ctx 上下文
//   - serviceMark 服务标识 当前服务的标识
// 返回:
//   - map[string]*types.DatabaseInfo 数据库配置 key 为 areaKey
//   - error 错误
type RemoteDatabaseHandlerFunc func (ctx context.Context,serviceMark string) (map[string]*types.DatabaseInfo,error) // 请求数据库配置

type RemoteDatabaseClientManagerInf interface {
	types.SubServiceInf
	GetDBFromKey(key string) (*gorm.DB, error)
	SetRouterHandler(routerHandler DBRouterFunc) // 设置路由方法
}


// 远程同步数据库 管理器
type RemoteDatabaseClientManager struct {
	dbMap     map[string]*gorm.DB
	requestHandler RemoteDatabaseHandlerFunc
}


// 创建远程数据库管理器
// 参数:
//   - requestHandler 请求数据库配置
// 返回:
//   - *RemoteDatabaseClientManager 远程数据库管理器
func NewRemoteDatabaseClientManager(requestHandler RemoteDatabaseHandlerFunc) *RemoteDatabaseClientManager {
	return &RemoteDatabaseClientManager{
		requestHandler: requestHandler,
		dbMap: make(map[string]*gorm.DB),
	}
}


// 启动远程数据库管理器
// 参数:
//   - ctx 上下文
//   - serivceMark 服务标识
//   - resultChan 结果通道
func (d *RemoteDatabaseClientManager) Start(ctx context.Context, serivceMark any, resultChan chan<- types.Result[any]) {
	serviceMarkStr,ok:=serivceMark.(string)
	if !ok {
		resultChan <- types.Result[any]{
			ErrCode: 1,
			Message: "database manager start error: service mark is not string",
		}
		return
	}
	dbConfigs, err := d.requestHandler(ctx, serviceMarkStr)
	if err != nil {
		resultChan <- types.Result[any]{
			ErrCode: 1,
			Message: "database manager start error: " + err.Error(),
		}
		return
	}
	for key, dbConfig := range dbConfigs {
		db, err := MysqlConnect(dbConfig) // 数据库连接
		if err != nil {
			resultChan <- types.Result[any]{
				ErrCode: 1,
				Message: "database manager start error: " + err.Error(),
			}
		}
		d.dbMap[key] = db
	}
}


// 根据key获取数据库
func (d *RemoteDatabaseClientManager) GetDBFromKey(areaKey string) (*gorm.DB, error) {
	db, ok := d.dbMap[areaKey]
	if !ok {
		return nil, errors.New("database not found")
	}
	return db, nil
}


