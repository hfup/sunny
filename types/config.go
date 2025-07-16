package types


type Config struct {
	ActiveEnv string `yaml:"active_env"`
	EnvArgs   map[string]string `yaml:"env_args" json:"env_args"` // 环境参数
	WebRoutes []*WebRouterInfo  `yaml:"web_routes" json:"web_routes"` // 路由信息
	Services  []*ServiceInfo    `yaml:"services" json:"services"`     // 服务信息
	DatabaseClientManager *DatabaseClientManagerInfo `yaml:"database_client_manager" json:"database_client_manager"` // 数据库管理器配置
	Redis *RedisConfig `yaml:"redis" json:"redis"` // redis 配置
	Mq *MqConfig `yaml:"mq" json:"mq"` // mq 配置
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
	DbId      string `yaml:"db_id" json:"db_id"` // 数据库id 唯一
}


// 数据库信息
type DatabaseInfo struct {
	Driver   string `yaml:"driver" json:"driver"` // mysql, postgres, sqlite
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	User     string `yaml:"user" json:"user"`
	Password string `yaml:"password" json:"password"`
	DbName   string `yaml:"db_name" json:"db_name"`
	Charset  string `yaml:"charset" json:"charset"` // 字符集
	MaxIdelConns int    `yaml:"max_idel_conns" json:"max_idel_conns"` // 最大空闲连接数
	MaxOpenConns int    `yaml:"max_open_conns" json:"max_open_conns"` // 最大打开连接数
	MaxLifetime  int    `yaml:"max_lifetime" json:"max_lifetime"`     // 连接最大生命周期

	AreaKey string `yaml:"area_key" json:"area_key"` // 区域key 用于区分不同的数据库
}


// 数据库管理器配置
type DatabaseClientManagerInfo struct {
	// 本地配置文件如果配置了 Db 默认就是从本地配置获取 远程拉取配置 通过延迟 注入
	IsDebug int8 `yaml:"is_debug,omitempty" json:"is_debug,omitempty"` // 1 打印日志 0 不打印日志
	DBs []*DatabaseInfo `yaml:"dbs,omitempty" json:"dbs,omitempty"` // 数据库配置
}


type MqConfig struct {
	Type string `yaml:"type" json:"type"` // rabbitmq, kafka
	RabbitMQ *RabbitMQConfig `yaml:"rabbitmq" json:"rabbitmq"` // rabbitmq 配置
	Kafka *KafkaConfig `yaml:"kafka" json:"kafka"` // kafka 配置
}


type RabbitMQConfig struct {
	Host string `yaml:"host" json:"host"` // 连接地址
	Port int `yaml:"port" json:"port"` // 连接端口
	Username string `yaml:"username" json:"username"` // 用户名
	Password string `yaml:"password" json:"password"` // 密码
	MaxRetries int `yaml:"max_retries" json:"max_retries"` // 最大重试次数
	RetryInterval int `yaml:"retry_interval" json:"retry_interval"` // 重试间隔
	ReconnectDelay int `yaml:"reconnect_delay" json:"reconnect_delay"` // 重连延迟
	ChannelPoolSize int `yaml:"channel_pool_size" json:"channel_pool_size"` // 连接池大小
}

type KafkaConfig struct {
	Brokers []string `yaml:"brokers" json:"brokers"` // 连接地址
	MaxRetries int `yaml:"max_retries" json:"max_retries"` // 最大重试次数
	RetryInterval int `yaml:"retry_interval" json:"retry_interval"` // 重试间隔
	ReconnectDelay int `yaml:"reconnect_delay" json:"reconnect_delay"` // 重连延迟
	SecurityProtocol string `yaml:"security_protocol" json:"security_protocol"` // 安全协议
	SASLMechanism string `yaml:"sasl_mechanism" json:"sasl_mechanism"` // SASL 机制
	SASLUsername string `yaml:"sasl_username" json:"sasl_username"` // SASL 用户名
	SASLPassword string `yaml:"sasl_password" json:"sasl_password"` // SASL 密码
}