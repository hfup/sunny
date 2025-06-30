package auths

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"
)

type JwtSignerResult struct {
	Signature string
	KeyIndex int
}


type Jwt struct {
	UpdateKeyPeriod time.Duration
	keyChain        [][]byte
	currentKeyIndex int
	chainCapacity int // chain  
}

func NewJwt(updateKeyPeriod time.Duration, chainCapacity int) *Jwt {
	return &Jwt{
		UpdateKeyPeriod: updateKeyPeriod,
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

