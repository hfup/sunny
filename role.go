package sunny

import (
	"sort"

	"github.com/sirupsen/logrus"
)

type RoleInf interface {
	RoleLabel() string
	Description() string
	UseBefore(order int, handlerFunc ActionHandlerFunc) // 使用前置动作
	UseAfter(order int, handlerFunc ActionHandlerFunc) // 使用后置动作
	RunBefore(c *Context) // 执行前置动作
	RunAfter(c *Context) // 执行后置动作
}

type Role struct {
	roleLabel string
	description string
	beforeActions []ActionHandlerWithOrder
	afterActions []ActionHandlerWithOrder
}

func NewRole(roleLabel string, description string) *Role {
	return &Role{
		roleLabel: roleLabel,
		description: description,
	}
}

func (r *Role) RoleLabel() string {
	return r.roleLabel
}

func (r *Role) Description() string {
	return r.description
}

func (r *Role) UseBefore(order int, handlerFunc ActionHandlerFunc) {
	if r.beforeActions == nil {
		r.beforeActions = []ActionHandlerWithOrder{}
	}
	r.beforeActions = append(r.beforeActions, ActionHandlerWithOrder{ActionHandlerFunc: handlerFunc, Order: order})
	sort.Slice(r.beforeActions, func(i, j int) bool {
		return r.beforeActions[i].Order < r.beforeActions[j].Order
	})
}

func (r *Role) UseAfter(order int, handlerFunc ActionHandlerFunc) {
	if r.afterActions == nil {
		r.afterActions = []ActionHandlerWithOrder{}
	}
	r.afterActions = append(r.afterActions, ActionHandlerWithOrder{ActionHandlerFunc: handlerFunc, Order: order})
	sort.Slice(r.afterActions, func(i, j int) bool {
		return r.afterActions[i].Order < r.afterActions[j].Order
	})
}

func (r *Role) RunBefore(c *Context) {
	if len(r.beforeActions) > 0 {
		logrus.WithFields(logrus.Fields{
			"role": r.roleLabel,
			"before_actions": len(r.beforeActions),
		}).Info("run before actions")
		for _, action := range r.beforeActions {
			action.ActionHandlerFunc(c)
			if !c.ActionNext() {
				return
			}
		}
	}
}

func (r *Role) RunAfter(c *Context) {
	if len(r.afterActions) > 0 {
		for _, action := range r.afterActions {
			action.ActionHandlerFunc(c)
			if !c.ActionNext() {
				return
			}
		}
	}
}

