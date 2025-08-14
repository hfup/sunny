package auths

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/hfup/sunny/types"
	"github.com/hfup/sunny/utils"
	"github.com/sirupsen/logrus"
)

type JwtSignerResult struct {
	Signature string
	KeyIndex int
}


type Jwt struct {
	keyChain [][]byte // 
	currentKeyIndex int
}

func NewJwt() *Jwt {
	return &Jwt{
		keyChain: make([][]byte,0),
		currentKeyIndex: 0,
	}
}

// GenerateSignature 生成签名
// 参数:
//   - data: 要签名的数据
// 返回:
//   - string 签名结果
//   - error 错误信息
func (j *Jwt) GenerateSignature(data map[string]string) (*JwtSignerResult, error) {
	// 获取当前密钥
	key, err := j.GetKey(j.currentKeyIndex)
	if err != nil {
		return nil, err
	}
	// 对 map 进行字典排序并转换为字符串
	sortedStr := j.sortMapToString(data)
	
	// 使用 HMAC-SHA256 生成签名
	h := hmac.New(sha256.New, key)
	h.Write([]byte(sortedStr))
	signature := h.Sum(nil)

	// 返回十六进制编码的签名
	return &JwtSignerResult{
		Signature: hex.EncodeToString(signature),
		KeyIndex:  j.currentKeyIndex,
	}, nil
}


// sortMapToString 将 map 按字典序排序并转换为字符串
// 参数:
//   - data: map[string]string 要排序的数据
// 返回:
//   - string 排序后的字符串
func (j *Jwt) sortMapToString(data map[string]string) string {
	// 获取所有键并排序
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 按排序后的键顺序构建 key=value&key=value 格式的字符串
	var result strings.Builder
	
	for i, k := range keys {
		if i > 0 {
			result.WriteString("&")
		}
		
		result.WriteString(k)
		result.WriteString("=")
		result.WriteString(data[k])
	}
	
	return result.String()
}


// Reset 重置密钥链
// 参数:
//   - keyChain: 密钥链
//   - currentKeyIndex: 当前密钥索引
// 返回:
//   - error 错误信息
func (j *Jwt) Reset(keyChain [][]byte,currentKeyIndex int) error{
	if currentKeyIndex > len(keyChain) - 1  {
		return errors.New("currentKeyIndex out of range")
	}
	j.keyChain = keyChain
	j.currentKeyIndex = currentKeyIndex
	return nil
}


// AddNewKey 添加新密钥
// 如果 index 大于 当前密钥链长度,则直接添加到末尾
// 如果 index 小于 当前密钥链长度,则替换当前索引的密钥
// 参数:
//   - key: 新密钥
//   - index: 新密钥索引
// 返回:
//   - error 错误信息
func (j *Jwt) AddNewKey(key []byte,index int) error{
	if index < 0 {
		return errors.New("index out of range")
	}
	if index >= len(j.keyChain) {
		j.keyChain = append(j.keyChain, key)
	}else{
		j.keyChain[index] = key
	}
	j.currentKeyIndex = index
	return nil
}



// GetKey 获取密钥
// 参数:
//   - index: 密钥索引
// 返回:
//   - []byte 密钥
//   - error 错误信息
func (j *Jwt) GetKey(index int) ([]byte, error) {
	if index < 0 || index >= len(j.keyChain) {
		return nil, errors.New("index out of range")
	}
	key := j.keyChain[index]
	if key == nil {	
		return nil, errors.New("key is nil")
	}
	return key, nil
}



// JwtKeyManager 密钥管理器
type JwtKeyManager struct {
	keyUpdatePeriod time.Duration // 密钥更新周期
	currentKeyIndex int // 当前密钥索引
	keyUpdateHandler KeyUpdateHandler // 密钥更新处理函数
	keysChain [9][]byte // 密钥链，固定容量为9
	keyInitHandler KeyInitHandler // 密钥初始化处理函数
	keyCount int // 已初始化的密钥数量
}

// KeyUpdateHandler 密钥更新处理函数
// 参数:
//   - ctx: 上下文
//   - key: 密钥
//   - curIndex: 当前密钥索引
// 返回:
//   - error 错误信息
// 注意: 这个函数执行的时候 如果要通知其他服务,保证其他服务已启动, 建议使用 消息队列 通知其他服务
// 新生成的key 应该要持久化到 数据库中, 并设置到 密钥管理器中,重启的时候重新加载
type KeyUpdateHandler func(ctx context.Context,key []byte,curIndex int) error

// KeyInitHandler 密钥初始化处理函数
// 当服务服务器重启的时候,应该要从 持久化的密钥 中获取 密钥信息, 并设置到 密钥管理器中
// 参数:
//   - ctx: 上下文
// 返回:
//   - []byte 密钥
//   - int 当前密钥索引
//   - error 错误信息
type KeyInitHandler func(ctx context.Context) ([][]byte,int,error)


// NewJwtKeyManager 创建密钥管理器
// 参数:
//   - keyUpdatePeriod: 密钥更新周期
//   - keyInitHandler: 密钥初始化处理函数
//   - keyUpdateHandler: 密钥更新处理函数
// 返回:
//   - *JwtKeyManager 密钥管理器
func NewJwtKeyManager(keyUpdatePeriod time.Duration,keyInitHandler KeyInitHandler,keyUpdateHandler KeyUpdateHandler) *JwtKeyManager {
	return &JwtKeyManager{
		keyUpdatePeriod: keyUpdatePeriod,
		keyInitHandler: keyInitHandler,
		keyUpdateHandler: keyUpdateHandler,
		currentKeyIndex: 0,
		keyCount: 0,
	}
}

func (j *JwtKeyManager) Start(ctx context.Context,args any,resultChan chan<- types.Result[any]) {
	// 初始化密钥 存在重启的情况
	keyList,index,err := j.keyInitHandler(ctx) // 这里的返回的排序 asc 
	if err != nil {
		resultChan <- types.Result[any]	{
			ErrCode: 1,
			Message: "初始化密钥失败",
			Data: nil,
		}
		return
	}
	if len(keyList) > 0  {
		// 初始化密钥链，最多取9个
		for i, key := range keyList {
			if i >= 9 {
				break
			}
			j.keysChain[i] = key
			j.keyCount++
		}
		// 确保索引在有效范围内
		if index >= 0 && index < j.keyCount {
			j.currentKeyIndex = index
		} else {
			j.currentKeyIndex = 0
		}
	}else{
		// 没有密钥时生成第一个
		key := utils.RandBytes(32)
		j.keysChain[0] = key
		j.currentKeyIndex = 0
		j.keyCount = 1

		// 更新密钥
		err = j.keyUpdateHandler(ctx,key,j.currentKeyIndex)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"err": err,
				"service": "jwt_key_manager",
				"key": key,
				"index": j.currentKeyIndex,
			}).Warn("更新密钥失败")
		}
	}

	if j.keyUpdatePeriod == 0 {
		// 默认 3 天 
		j.keyUpdatePeriod = 3 * 24 * time.Hour
	}

	ticker := time.NewTicker(j.keyUpdatePeriod) // 定时器
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 添加新密钥，使用循环索引
			newKey := utils.RandBytes(32)
			
			// 计算下一个索引：当前索引+1，超过8就从0开始
			nextIndex := (j.currentKeyIndex + 1) % 9
			
			// 设置新密钥
			j.keysChain[nextIndex] = newKey
			j.currentKeyIndex = nextIndex
			
			// 如果还没有填满9个位置，增加计数
			if j.keyCount < 9 && nextIndex >= j.keyCount {
				j.keyCount = nextIndex + 1
			}

			err = j.keyUpdateHandler(ctx,newKey,j.currentKeyIndex) // 更新密钥
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"err": err,
					"service": "jwt_key_manager",
					"nextIndex": nextIndex,
				}).Warn("更新密钥失败")
			}
		}
	}
}

func (j *JwtKeyManager) IsErrorStop() bool {
	return true
}

func (j *JwtKeyManager) ServiceName() string {
	return "JwtKeyManager"
}


func (j *JwtKeyManager) GetCurrentKey() []byte {
	if j.keyCount == 0 {
		return nil
	}
	return j.keysChain[j.currentKeyIndex]
}

func (j *JwtKeyManager) GetCurrentKeyIndex() int {
	return j.currentKeyIndex
}

func (j *JwtKeyManager) GetKeysChain() [][]byte {
	result := make([][]byte, j.keyCount)
	for i := 0; i < j.keyCount; i++ {
		result[i] = j.keysChain[i]
	}
	return result
}

func (j *JwtKeyManager) GetKeysChainCapacity() int {
	return 9
}

// GetKeyCount 获取已初始化的密钥数量
// 返回:
//   - int 已初始化的密钥数量
func (j *JwtKeyManager) GetKeyCount() int {
	return j.keyCount
}

// GetKeyByIndex 获取指定索引的密钥
// 参数:
//   - index: 密钥索引
// 返回:
//   - []byte 密钥
//   - error 错误信息
func (j *JwtKeyManager) GetKeyByIndex(index int) ([]byte, error) {
	if index < 0 || index >= j.keyCount {
		return nil, errors.New("index out of range")
	}
	key := j.keysChain[index]
	if key == nil {	
		return nil, errors.New("key is nil")
	}
	return key, nil
}
