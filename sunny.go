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
)


// Sunny 主服务
type Sunny struct {
	configPath string // 配置文件路径
	activeEnv string // 激活环境 dev,test prod
	runPath string // 运行路径
	config *Config // 配置
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
	return nil
}


// 加载配置文件
func (s *Sunny) loadConfig() error{
	configer := &Config{}
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

	logrus.Info("load config success")
	
	return nil
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


