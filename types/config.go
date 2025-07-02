package types


type Config struct {
	ActiveEnv string `yaml:"active_env"`
	EnvArgs   map[string]string `yaml:"env_args" json:"env_args"` // 环境参数
	WebRoutes []*WebRouterInfo  `yaml:"web_routes" json:"web_routes"` // 路由信息
	Services  []*ServiceInfo    `yaml:"services" json:"services"`     // 服务信息
	Redis     *RedisConfig      `yaml:"redis" json:"redis"`             // redis 配置

}

// 路由信息
type WebRouterInfo struct {
	Path     string `yaml:"path" json:"path"`
	Role     string `yaml:"role" json:"role"`
	RoleDesc string `yaml:"role_desc" json:"role_desc"` // 角色描述
}

// 服务信息
type ServiceInfo struct {
	Protocol    string `yaml:"protocol" json:"protocol"`           // http, https ,grpc,beechat 自定义 beechat 自定义协议
	Port        int    `yaml:"port" json:"port"`                   // 服务端口
	Host        string `yaml:"host" json:"host"`                   // 服务地址
	CertPemPath string `yaml:"cert_pem_path" json:"cert_pem_path"` // 证书文件
	KeyPemPath  string `yaml:"key_pem_path" json:"key_pem_path"`   // 私钥文件
}


// redis 配置
type RedisConfig struct {
	IsCluster bool `yaml:"is_cluster" json:"is_cluster"` // 是否集群
	Addrs     []string `yaml:"addrs" json:"addrs"` // 集群地址 如果是单机，则只有一个地址
	PoolSize  int `yaml:"pool_size" json:"pool_size"` // 连接池大小
	MaxIdle   int `yaml:"max_idle" json:"max_idle"` // 最大空闲连接数
	MaxActive int `yaml:"max_active" json:"max_active"` // 最大活跃连接数
	MinIdleConns int `yaml:"min_idle_conns" json:"min_idle_conns"` // 最小空闲连接数
	Password  string `yaml:"password" json:"password"` // 密码
	DB        int `yaml:"db" json:"db"` // 数据库
}
