# sunny 引擎规范
1. 接口描述
    - Start(ctx context.Context,args ...string) // 启动服务
    - OnLaunch(args ...string) // 启动时初始化操作 args 启动传入的参数 ，该阶段 加载配置文件
    - UseRoles(roles ...RoleInf)
    - UseGroup(groups ...GroupInf)
    ...


### web 请求
1. HTTP请求 → 路由匹配 → 角色验证[前置钩子->handler->后置钩子] → 组处理[前置钩子->FindHandler(actionKey):Handler->后置钩子]  → 响应返回
    - 所有前置/后置 钩子 支持多个,支持排序
2. 路径 "/v1/:group/actionKey" // v1 -> role,roleLabel 取决于 [path]-> roleLabel , 存在同一个路径有多个角色
3. type MultiRoleHandlerFunc(path,group,actionKey string) string // 返回 角色
2. Context 上下文
    ```
    type Context struct{*gin.Context,....}
    func (c *Context) JsonResult(errCode int,msg string,data ...any)
    func (c *Context) HandlerAbort()
    func (c *Context) HandlerNext() bool
    ```
3. HandlerFunc(ctx *context)
4. role 角色
    - 接口描述
        - GetRoleLabel() string // 唯一标志
        - GetRemark() string // 角色备注
        - Call(c *context) // handler [前置钩子->handler->后置钩子] 在这里实现
5. group 组
    - 接口描述
        - GetRoleLabel() string // 当前角色
        - Call(c *context) // [前置钩子->FindHandler(actionKey):Handler->后置钩子] 在这里实现
        - GetGroupLabel() string
        - GetRemark() // 组备注

### 消息队列



