package auths

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/hfup/sunny"
	"github.com/hfup/sunny/utils"
	"github.com/sirupsen/logrus"
)

type JwtSignerResult struct {
	Signature string
	KeyIndex int
}


type Jwt struct {
	keyChain        [][]byte
	currentKeyIndex int
	chainCapacity int // chain  
}

func NewJwt(chainCapacity int) *Jwt {
	return &Jwt{
		chainCapacity:   chainCapacity,
	}
}

// GenerateSignature 生成签名
// 参数:
//   - data: 要签名的数据
// 返回:
//   - string 签名结果
//   - error 错误信息
func (j *Jwt) GenerateSignature(data map[string]any) (*JwtSignerResult, error) {
	// 获取当前密钥
	key, err := j.GetKey(j.currentKeyIndex)
	if err != nil {
		return nil, err
	}
	// 对 map 进行字典排序并转换为字符串
	sortedStr, err := j.sortMapToString(data)
	if err != nil {
		return nil, err
	}
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
//   - data: map[string]any 要排序的数据
// 返回:
//   - string 排序后的字符串
//   - error 错误信息
func (j *Jwt) sortMapToString(data map[string]any) (string, error) {
	// 获取所有键并排序
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 构建排序后的 map
	sortedMap := make(map[string]any)
	for _, k := range keys {
		sortedMap[k] = data[k]
	}

	// 转换为 JSON 字符串
	jsonBytes, err := json.Marshal(sortedMap)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}


// SetKeyChain 设置密钥链
// 参数:
//   - keyChain: 密钥链
func (j *Jwt) SetKeyChain(keyChain [][]byte) {
	j.keyChain = keyChain
}

// SetCurrentKeyIndex 设置当前密钥索引
// 参数:
//   - index: 当前密钥索引
func (j *Jwt) SetCurrentKeyIndexAndKey(index int, key []byte) error{
	if index < 0 || index >= j.chainCapacity {
		return errors.New("index out of range")
	}
	
	// 初始化 keyChain 如果为空
	if j.keyChain == nil {
		j.keyChain = make([][]byte, j.chainCapacity)
	}
	
	// 确保 keyChain 长度足够
	if len(j.keyChain) < j.chainCapacity {
		newChain := make([][]byte, j.chainCapacity)
		copy(newChain, j.keyChain)
		j.keyChain = newChain
	}
	
	j.currentKeyIndex = index
	j.keyChain[index] = key

	return nil
}

// GetKey 获取密钥
// 参数:
//   - index: 密钥索引
// 返回:
//   - []byte 密钥
//   - error 错误信息
func (j *Jwt) GetKey(index int) ([]byte, error) {
	if index < 0 || index >= j.chainCapacity {
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
	keysChain [][]byte // 密钥链
	chainCapacity int // 密钥链容量
	keyInitHandler KeyInitHandler // 密钥初始化处理函数
}

// KeyUpdateHandler 密钥更新处理函数
// 参数:
//   - ctx: 上下文
//   - key: 密钥
//   - curIndex: 当前密钥索引
// 返回:
//   - error 错误信息
// 注意: 这个函数执行的时候 如果要通知其他服务,保证其他服务已启动, 建议使用 消息队列 通知其他服务
type KeyUpdateHandler func(ctx context.Context,key []byte,curIndex int) error

// KeyInitHandler 密钥初始化处理函数
// 参数:
//   - ctx: 上下文
// 返回:
//   - []byte 密钥
//   - int 当前密钥索引
//   - error 错误信息
type KeyInitHandler func(ctx context.Context) ([][]byte,int,error)


// NewJwtKeyManager 创建密钥管理器
// 参数:
//   - chainCapacity: 密钥链容量
//   - keyUpdatePeriod: 密钥更新周期
//   - keyInitHandler: 密钥初始化处理函数
//   - keyUpdateHandler: 密钥更新处理函数
// 返回:
//   - *JwtKeyManager 密钥管理器
func NewJwtKeyManager(chainCapacity int,keyUpdatePeriod time.Duration,keyInitHandler KeyInitHandler,keyUpdateHandler KeyUpdateHandler) *JwtKeyManager {
	return &JwtKeyManager{
		keyUpdatePeriod: keyUpdatePeriod,
		keyInitHandler: keyInitHandler,
		keyUpdateHandler: keyUpdateHandler,
		chainCapacity: chainCapacity,
		keysChain: make([][]byte,0,chainCapacity),
	}
}

func (j *JwtKeyManager) Start(ctx context.Context,args any,resultChan chan<- sunny.Result) {
	// 初始化密钥
	keyList,index,err := j.keyInitHandler(ctx) // 这里的返回的排序 asc
	if err != nil {
		resultChan <- sunny.Result{
			Code:    -1,
			Msg: "初始化密钥失败",
		}
		return
	}
	if len(keyList) > 0  {
		// 这里要判断 最大取 j.chainCapacity 个
		if len(keyList) > j.chainCapacity {
			keyList = keyList[:j.chainCapacity]
		}
		j.keysChain = append(j.keysChain, keyList...)
		j.currentKeyIndex = index

	}else{
		key := utils.RandBytes(32)
		j.keysChain = append(j.keysChain, key)
		j.currentKeyIndex = 0

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
		// 默认 7 天
		j.keyUpdatePeriod = 7 * 24 * time.Hour
	}

	ticker := time.NewTicker(j.keyUpdatePeriod) // 定时器
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 更新密钥
			newKey := utils.RandBytes(32)
			
			// 确保 keysChain 已初始化且长度足够
			if len(j.keysChain) == 0 {
				logrus.WithFields(logrus.Fields{
					"service": "jwt_key_manager",
				}).Error("keysChain 为空，无法更新密钥")
				continue
			}
			
			// 先判断当前激活的index 是否是最后一个
			if j.currentKeyIndex >= len(j.keysChain) - 1 {
				// 是最后一个 则更新为第一个
				j.currentKeyIndex = 0
			}else{
				// 不是最后一个 则更新为下一个
				j.currentKeyIndex++
			}
			
			// 确保索引在有效范围内
			if j.currentKeyIndex >= 0 && j.currentKeyIndex < len(j.keysChain) {
				j.keysChain[j.currentKeyIndex] = newKey
			} else {
				logrus.WithFields(logrus.Fields{
					"service": "jwt_key_manager",
					"index": j.currentKeyIndex,
					"chainLen": len(j.keysChain),
				}).Error("密钥索引超出范围")
				continue
			}

			err = j.keyUpdateHandler(ctx,newKey,j.currentKeyIndex) // 更新密钥
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"err": err,
					"service": "jwt_key_manager",
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
	return j.keysChain[j.currentKeyIndex]
}

func (j *JwtKeyManager) GetCurrentKeyIndex() int {
	return j.currentKeyIndex
}

func (j *JwtKeyManager) GetKeysChain() [][]byte {
	return j.keysChain
}

func (j *JwtKeyManager) GetKeysChainCapacity() int {
	return j.chainCapacity
}
