package storages

import (
	"io"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"sync"
	"context"
)


type OssInfo struct {
	Region string `json:"region"`
	AccessKeyId string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	Bucket string `json:"bucket"`
}

type GetOssInfoFunc func() (*OssInfo,error) // 获取oss信息函数


// 创建oss存储
// 参数：
//  - bucketLabel 桶标签
//  - getOssInfoFunc 获取oss信息函数
// 返回：
//  - oss存储
func NewOssStorage(bucketLabel string,getOssInfoFunc GetOssInfoFunc) *OssStorage {
	return &OssStorage{
		bucketLabel: bucketLabel,
		getOssInfoFunc: getOssInfoFunc,
	}
}


type OssStorage struct {
	bucketLabel string	// 桶标签
	ossInfo *OssInfo	// oss信息
	ossClient *oss.Client	// oss客户端
	getOssInfoFunc GetOssInfoFunc // 获取oss信息函数

	mu sync.Mutex
}


// 获取阿里云oss客户端
func (o *OssStorage) getClient() (*oss.Client, error) {
	if o.ossClient != nil {
		return o.ossClient, nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()

	// double check
	if o.ossClient != nil {
		return o.ossClient, nil
	}

	if o.ossInfo == nil {
		// 网络请求 grpc
		ossInfo,err:=o.getOssInfoFunc()
		if err != nil {
			return nil,err
		}
		o.ossInfo = ossInfo
	}
	endpoint := "https://" + o.ossInfo.Region + ".aliyuncs.com"
	client, err := oss.New(endpoint, o.ossInfo.AccessKeyId, o.ossInfo.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	o.ossClient = client
	return client, nil
}


func (o *OssStorage) GetType() string {	
	return o.bucketLabel
}

// 上传文件
// 参数：
//  - ctx 上下文
//  - objtectKey 对象key
//  - data 数据
// 返回：
//  - 错误
func (o *OssStorage) Upload(ctx context.Context,objtectKey string,data io.Reader) error {
	client,err:=o.getClient()
	if err!=nil{
		return err
	}
	bucket, err := client.Bucket(o.ossInfo.Bucket)
	if err != nil {
		return err
	}
	err=bucket.PutObject(objtectKey, data)
	if err!=nil{
		return err
	}
	return nil
}


// 下载文件
// 参数：
//  - ctx 上下文
//  - objtectKey 对象key
// 返回：
//  - 数据
//  - 错误
func (o *OssStorage) Download(ctx context.Context,objtectKey string) (io.ReadCloser,error) {
	client,err:=o.getClient()
	if err!=nil{
		return nil,err
	}
	bucket, err := client.Bucket(o.ossInfo.Bucket)
	if err != nil {
		return nil,err
	}
	return bucket.GetObject(objtectKey)
}