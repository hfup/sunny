package sunny

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"errors"
)


var (
	ErrDBNotFound = errors.New("database not found")
)

type ActionHanderFunc func(c *Context)

type ActionHandlerWithOrder struct {
	ActionHandlerFunc ActionHanderFunc // 动作处理函数
	Order int // 执行顺序
}

type ActionHandlerWithOrderList []ActionHandlerWithOrder


type Context struct {
	*gin.Context
	curDB *gorm.DB
	actionNext bool // 是否执行下一个中间件
	roleLabel string // 角色标签
	groupLabel string // 组标签
	actionLabel string // 动作标签
}

// 获取当前数据库
func (c *Context) GetDB() (*gorm.DB, error) {
	if c.curDB == nil {
		return nil, ErrDBNotFound
	}
	return  c.curDB,nil
}

// 设置当前数据库
func (c *Context) SetDB(db *gorm.DB) {
	c.curDB = db
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

// JsonResult 返回json结果
func (c *Context) JsonResult(code int, msg string, data ...any) {
	result := gin.H{
		"err_code": code,
		"message":    msg,
	}

	if len(data) == 1 {
		result["data"] = data[0]
	} else if len(data) > 1 {
		result["data"] = data
	}
	c.JSON(200, result)
}