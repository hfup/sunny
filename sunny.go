package sunny

import (
	"context"
	"os"
	"strings"

	"github.com/hfup/sunny/utils"
	"github.com/sirupsen/logrus"
	"github.com/go-redis/redis/v8"
	"github.com/hfup/sunny/components/databases"
	"github.com/hfup/sunny/types"

	"github.com/gin-gonic/gin"
)


type MultiRoleHandler func(groupLabel, actionLabel string) string

// Sunny 主服务
type Sunny struct {
	*gin.Engine
	configPath string // 配置文件路径
	activeEnv string // 激活环境 dev,test prod
	runPath string // 运行路径
	config *types.Config // 配置

	roles map[string]RoleInf // 角色
	groups map[string]GroupInf // 组

	pathRoles map[string]string // 路径与角色映射
	multiRoleHandlers map[string]MultiRoleHandler // 多角色处理器  存在同一个路径，但是有多个角色

	rolesBeforeHandlers map[string][]ActionHandlerWithOrder // 角色前置处理器
	rolesAfterHandlers map[string][]ActionHandlerWithOrder // 角色后置处理器

	subServices []types.SubServiceInf
	redisClient redis.UniversalClient
	databaseManager *databases.DatabaseManager

	// 同步执行的 RunAble
	syncRunAbles []types.RunAbleInf
	asyncRunAbles []types.RunAbleInf
	requiredSubSrvSuccessCount int // 必须成功启动的子服务数量
	currentSubSrvSuccessCount int // 当前成功启动的子服务数量
}

// 初始化
// 参数：
//  - configPath 配置文件路径
//  - activeEnv 激活环境 dev,test prod
// 返回：
//  - 错误
func (s *Sunny) Init(configPath,activeEnv string) error{
	s.configPath = configPath
	s.activeEnv = activeEnv

	runPath,err := os.Getwd()
	if err != nil{
		return err
	}
	s.runPath = runPath

	// 读取配置文件
	if s.configPath == ""{
		s.configPath = s.runPath + "/resources/"
	}

	err = s.loadConfig()
	if err != nil{
		panic("load config file error: " + err.Error())
	}
	logrus.Info("load config file success")

	if len(s.config.WebRoutes) > 0 {
		s.initWebRoutes(s.config.WebRoutes)
	}


	return nil
}


// 加载配置文件
func (s *Sunny) loadConfig() error{
	configer := &types.Config{}
	if s.configPath == ""{
		s.configPath = s.runPath + "/resources/"
	}
	if !strings.HasSuffix(s.configPath, "/") {
		s.configPath += "/"
	}

	baseConfigFile := s.configPath + "application.yaml"
	err := utils.ReadYamlFile(baseConfigFile, configer)
	if err != nil {
		return err
	}

	activeEnv := configer.ActiveEnv
	if s.activeEnv != ""{
		activeEnv = s.activeEnv // 如果传入了激活环境，则使用传入的激活环境
	}

	if activeEnv != ""{
		envConfigFile := s.configPath + "application-" + activeEnv + ".yaml"
		err = utils.ReadYamlFile(envConfigFile, configer)
		if err != nil {
			return err
		}
	}

	s.config = configer	
	return nil
}


func (r *Sunny) initWebRoutes(routes []*types.WebRouterInfo){
	ginHandlerFunc := func(c *gin.Context) {
		fullPath := c.FullPath()
		paths := strings.Split(fullPath, "/")
		if len(paths) < 3 {
			c.JSON(200, gin.H{"err_code": -1, "message": "path not found"})
			c.Abort()
			return
		}

		findPath := "/" + paths[1]

		groupLabel := c.Param("group")
		actionLabel := c.Param("action")
		roleLabel := r.pathRoles[findPath]

		// if multi role handler exists
		if r.multiRoleHandlers[findPath] != nil {
			roleLabel = r.multiRoleHandlers[findPath](groupLabel, actionLabel)
		}

		ctx := &Context{
			Context:     c,
			roleLabel:   roleLabel,
			groupLabel:  groupLabel,
			actionLabel: actionLabel,
			actionNext:  true,
		}

		roleInfo, ok := r.roles[roleLabel]
		if !ok {
			c.JSON(200, gin.H{"err_code": -1, "message": "role not found"})
			c.Abort()
			return
		}

		roleInfo.RunBefore(ctx)
		if !ctx.actionNext {
			return
		}
		groupKey := roleLabel + ":" + groupLabel
		groupInfo, ok := r.groups[groupKey]
		if !ok {
			c.JSON(200, gin.H{"err_code": -1, "message": "group not found .."})
			return
		}
		groupInfo.Call(ctx)
		roleInfo.RunAfter(ctx)
	}

	// 初始化路由
	for _, route := range routes {
		r.pathRoles[route.Path] = route.Role
		// check role label
		if _, ok := r.roles[route.Role]; !ok {
			// 如果角色不存在 则 created
			r.roles[route.Role] = NewRole(route.Role, route.RoleDesc)
		}

		// 注入角色前置处理器
		if handlers, ok := r.rolesBeforeHandlers[route.Role]; ok {
			for _, handler := range handlers {
				r.roles[route.Role].UseBefore(handler.Order, handler.ActionHandlerFunc)
			}
		}

		// 注入角色后置处理器
		if handlers, ok := r.rolesAfterHandlers[route.Role]; ok {
			for _, handler := range handlers {
				r.roles[route.Role].UseAfter(handler.Order, handler.ActionHandlerFunc)
			}
		}

		r.GET(route.Path+"/:group/:action", ginHandlerFunc)  // 注册GET方法
		r.POST(route.Path+"/:group/:action", ginHandlerFunc) // 注册POST方法
	}
}

// 添加子服务
// 参数：
//  - srvs 子服务
// 返回：
//  - 错误	
func (s *Sunny) AddSubServices(srvs ...types.SubServiceInf) error{
	for _,srv := range srvs{
		if srv.IsErrorStop(){
			s.requiredSubSrvSuccessCount += 1
		}
		s.subServices = append(s.subServices,srv)
	}
	return nil
}

// 添加同步执行的 RunAble
// 参数：
//  - srvs 同步执行的 RunAble
// 返回：
//  - 错误
func (s *Sunny) AddSyncRunAbles(srvs ...types.RunAbleInf) error{
	s.syncRunAbles = append(s.syncRunAbles,srvs...)
	return nil
}

// 添加异步执行的 RunAble
// 参数：
//  - srvs 异步执行的 RunAble
// 返回：
//  - 错误
func (s *Sunny) AddAsyncRunAbles(srvs ...types.RunAbleInf) error{
	s.asyncRunAbles = append(s.asyncRunAbles,srvs...)
	return nil
}


// Start 启动 Sunny
// 参数：
//  - ctx 上下文
//  - args 参数
// 返回：
//  - 错误
func (s *Sunny) Start(ctx context.Context,args ...string) error{

	return nil
}


