package storages

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hfup/sunny/types"
	"github.com/tencentyun/cos-go-sdk-v5"
)



// 创建cos存储
// 参数：
//  - bucketLabel 桶标签
//  - getCosInfoFunc 获取桶信息函数
// 返回：
//  - cos存储
func NewCosStorage(cosInfo *types.CloudStorageConf) *CosStorage {
	return &CosStorage{
		cosInfo: cosInfo,
	}
}


type CosStorage struct {
	cosInfo *types.CloudStorageConf	// 桶信息
	cosClient *cos.Client	// 桶客户端
}

func (c *CosStorage) GetType() string {
	return "cos"
}


// 获取bucket client 后面如果存在 不同桶 需要区分
func (cs *CosStorage) getBucketClient() (*cos.Client,error) {
	if cs.cosClient != nil {
		return cs.cosClient,nil
	}
	uri:=fmt.Sprintf("https://%s.cos.%s.myqcloud.com",cs.cosInfo.Bucket,cs.cosInfo.Region)
	url,err:=url.Parse(uri)
	if err!=nil{
		return nil,err
	}
	su, _ := url.Parse(fmt.Sprintf("https://cos.%s.myqcloud.com",cs.cosInfo.Region))
	u:=&cos.BaseURL{	
		BucketURL: url,
		ServiceURL: su,
	}
	cs.cosClient = cos.NewClient(u, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  cs.cosInfo.SecretId,
			SecretKey: cs.cosInfo.SecretKey,
		},
	})
	return cs.cosClient,nil
}


// 上传文件
// 参数：
//  - ctx 上下文
//  - objtectKey 对象key
//  - data 数据
// 返回：
//  - 错误
func (cs *CosStorage) Upload(ctx context.Context,objtectKey string,data io.Reader) error {
	cosClient,err:=cs.getBucketClient()
	if err!=nil{
		return err
	}
	_,err=cosClient.Object.Put(ctx,objtectKey,data,nil)
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
func (cs *CosStorage) Download(ctx context.Context,objtectKey string) (io.ReadCloser,error) {
	cosClient,err:=cs.getBucketClient()
	if err!=nil{
		return nil,err
	}
	resp,err:=cosClient.Object.Get(ctx,objtectKey,nil)
	if err!=nil{
		return nil,err
	}
	return resp.Body,nil
}

