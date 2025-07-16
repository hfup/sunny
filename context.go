package sunny

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

var (
	ErrDBNotFound = errors.New("database not found")
	ErrRedisClientNotFound = errors.New("redis client not found")
)

type ActionHandlerFunc func(c *Context)

type ActionHandlerWithOrder struct {
	ActionHandlerFunc ActionHandlerFunc // 动作处理函数
	Order             int               // 执行顺序
}

type ActionHandlerWithOrderList []ActionHandlerWithOrder

type Context struct {
	*gin.Context
	curDB       *gorm.DB
	curRedisClient redis.UniversalClient
	actionNext  bool   // 是否执行下一个中间件
	roleLabel   string // 角色标签
	groupLabel  string // 组标签
	actionLabel string // 动作标签
}

// 获取当前数据库
func (c *Context) GetDB() (*gorm.DB, error) {
	if c.curDB == nil {
		return nil, ErrDBNotFound
	}
	return c.curDB, nil
}

// 设置当前数据库
func (c *Context) SetDB(db *gorm.DB) {
	c.curDB = db
}

// 获取当前 redis 客户端
func (c *Context) GetRedisClient() (redis.UniversalClient, error) {
	if c.curRedisClient == nil {
		return nil, ErrRedisClientNotFound
	}
	return c.curRedisClient, nil
}

// 获取 redis 客户端
func (c *Context) GetRedisClientFromKey(key string) (redis.UniversalClient, error) {
	redisClient, err := c.GetRedisClient()
	if err != nil {
		return nil, err
	}
	return redisClient, nil
}

// 设置当前 redis 客户端
func (c *Context) SetRedisClient(redisClient redis.UniversalClient) {
	c.curRedisClient = redisClient
}


func (c *Context) ActionNext() bool {
	return c.actionNext
}

func (c *Context) ActionLabel() string {
	return c.actionLabel
}

func (c *Context) RoleLabel() string {
	return c.roleLabel
}

func (c *Context) GroupLabel() string {
	return c.groupLabel
}

func (c *Context) Clone() *Context {
	// 复制 gin.Context
	ginCtx := c.Context.Copy()
	newCtx := &Context{
		Context:     ginCtx,
		curDB:       c.curDB,
		actionNext:  c.actionNext,
		roleLabel:   c.roleLabel,
		groupLabel:  c.groupLabel,
		actionLabel: c.actionLabel,
	}
	return newCtx
}

// JsonResult 返回json结果
func (c *Context) JsonResult(code int, msg string, data ...any) {
	result := gin.H{
		"err_code": code,
		"message":  msg,
	}

	if len(data) == 1 {
		result["data"] = data[0]
	} else if len(data) > 1 {
		result["data"] = data
	}
	c.JSON(200, result)
}
