package sunny

import "sort"

type GroupInf interface {
	GroupLabel() string
	RoleLabel() string
	Call(c *Context)
	BindAction(actionLabel string, handlerFunc ActionHandlerFunc, actionDesc string)
	UseBefore(order int, handlerFunc ActionHandlerFunc)
	UseAfter(order int, handlerFunc ActionHandlerFunc)
	Description() string // 描述
}


type Group struct {
	groupLabel string
	roleLabel string
	actions map[string]ActionHandlerFunc
	beforeActions []ActionHandlerWithOrder // 前置动作 在action执行前执行
	afterActions []ActionHandlerWithOrder // 后置动作 在action执行后执行 这里不要输出结果内容
	description string // 描述
}

// 创建组
func NewGroup(roleLabel string,groupLabel string, description string) *Group {
	return &Group{
		groupLabel: groupLabel,
		roleLabel: roleLabel,
		actions: make(map[string]ActionHandlerFunc),
		description: description,
	}
}

func (g *Group) GroupLabel() string {
	return g.groupLabel
}

func (g *Group) RoleLabel() string {
	return g.roleLabel
}

func (g *Group) Description() string {
	return g.description
}

// 绑定动作 actionLabel 动作标签 handlerFunc 处理函数 actionDesc 动作描述
func (g *Group) BindAction(actionLabel string, handlerFunc ActionHandlerFunc, actionDesc string) {
	g.actions[actionLabel] = handlerFunc
}

// 使用前置动作
func (g *Group) UseBefore(order int, handlerFunc ActionHandlerFunc) {
	if g.beforeActions == nil {
		g.beforeActions = []ActionHandlerWithOrder{}
	}
	g.beforeActions = append(g.beforeActions, ActionHandlerWithOrder{ActionHandlerFunc: handlerFunc, Order: order})
	
	// 排序 order 越小越先执行
	sort.Slice(g.beforeActions, func(i, j int) bool {
		return g.beforeActions[i].Order < g.beforeActions[j].Order
	})
}

// 使用后置动作
func (g *Group) UseAfter(order int, handlerFunc ActionHandlerFunc) {
	if g.afterActions == nil {
		g.afterActions = []ActionHandlerWithOrder{}
	}
	g.afterActions = append(g.afterActions, ActionHandlerWithOrder{ActionHandlerFunc: handlerFunc, Order: order})

	// 排序 order 越小越先执行
	sort.Slice(g.afterActions, func(i, j int) bool {
		return g.afterActions[i].Order < g.afterActions[j].Order
	})
}


// 调用动作
func (g *Group) Call(c *Context) {
	if len(g.beforeActions) > 0 {
		for _, action := range g.beforeActions {
			action.ActionHandlerFunc(c)
			if !c.ActionNext() {
				return 
			}
		}
	}
	handlerFunc, ok := g.actions[c.ActionLabel()]
	if !ok {
		c.JsonResult(404, "action not found")
		return
	}

	handlerFunc(c)

	if len(g.afterActions) > 0 {
		for _, action := range g.afterActions {
			action.ActionHandlerFunc(c)
			if !c.ActionNext() {
				return
			}	
		}
	}
}
