package utils

import (
	"math/rand"
	"sync"
	"time"

	"crypto/md5"
	"encoding/hex"
	"regexp"
	"crypto/sha256"
)

var (
	sendTime    = time.Now().UnixNano()
	sendLock    = sync.Mutex{}
	currentStep = 1
)

// RandStr 随机字符串
func RandStr(lenStr int, isNum bool) string {
	sendLock.Lock()
	defer sendLock.Unlock()
	currentStep++
	sendNum := sendTime + int64(currentStep)
	chars := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if isNum {
		chars = "0123456789"
	}
	bytes := []byte(chars)
	result := []byte{}
	r := rand.New(rand.NewSource(sendNum))
	for i := 0; i < lenStr; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}


func RandBytes(lenBytes int) []byte {
	sendLock.Lock()
	defer sendLock.Unlock()
	currentStep++
	sendNum := sendTime + int64(currentStep)
	bytes := make([]byte, lenBytes)
	r := rand.New(rand.NewSource(sendNum))
	for i := 0; i < lenBytes; i++ {
		bytes[i] = byte(r.Intn(256))
	}
	return bytes
}


// IsEmail 是否是邮箱
func IsEmail(email string) bool {
	regexpStr := `^([a-zA-Z0-9_-])+@([a-zA-Z0-9_-])+((\.[a-zA-Z0-9_-]{2,3}){1,2})$`
	regexps := regexp.MustCompile(regexpStr)
	return regexps.MatchString(email)
}


func MD5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func Sha256(str string) string {
	h := sha256.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}