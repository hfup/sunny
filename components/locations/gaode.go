package locations

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// GeocodeRequest 地理编码请求参数
type GeocodeRequest struct {
	Key      string `json:"key"`                // 必填：高德Key
	Address  string `json:"address"`            // 必填：结构化地址信息
	City     string `json:"city,omitempty"`     // 可选：指定查询的城市
	Output   string `json:"output,omitempty"`   // 可选：返回数据格式类型，默认JSON
	Callback string `json:"callback,omitempty"` // 可选：回调函数
	Sig      string `json:"sig,omitempty"`      // 可选：数字签名
}

// GeocodeResponse 地理编码响应
type GeocodeResponse struct {
	Status   string           `json:"status"`   // 返回状态：1成功，0失败
	Count    string           `json:"count"`    // 返回结果数目
	Info     string           `json:"info"`     // 返回状态说明
	InfoCode string           `json:"infocode"` // 返回状态码
	Geocodes []map[string]any `json:"geocodes"` // 地理编码信息列表
}

// Geocode 地理编码：将地址转换为经纬度坐标
func Geocode(req *GeocodeRequest) (*GeocodeResponse, error) {
	if req.Key == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if req.Address == "" {
		return nil, fmt.Errorf("address is required")
	}

	// 设置默认输出格式
	if req.Output == "" {
		req.Output = "JSON"
	}

	// 构建请求URL
	baseURL := "https://restapi.amap.com/v3/geocode/geo"
	params := url.Values{}
	params.Set("key", req.Key)
	params.Set("address", req.Address)
	params.Set("output", req.Output)

	if req.City != "" {
		params.Set("city", req.City)
	}
	if req.Callback != "" {
		params.Set("callback", req.Callback)
	}
	if req.Sig != "" {
		params.Set("sig", req.Sig)
	}

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// 调试：输出请求URL
	fmt.Printf("Request URL: %s\n", fullURL)

	// 发送HTTP请求
	resp, err := httpClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}
	var geocodeResp GeocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&geocodeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &geocodeResp, nil
}

// GeocodeSimple 简化的地理编码方法，只需要地址和可选的城市参数
func GeocodeSimple(key string, address string, city ...string) (*GeocodeResponse, error) {
	req := &GeocodeRequest{
		Key:     key,
		Address: address,
	}

	if len(city) > 0 && city[0] != "" {
		req.City = city[0]
	}
	return Geocode(req)
}

// RegeoRequest 逆地理编码请求参数
type RegeoRequest struct {
	Key        string `json:"key"`                  // 必填：高德Key
	Location   string `json:"location"`             // 必填：经纬度坐标，格式：经度,纬度
	Poitype    string `json:"poitype,omitempty"`    // 可选：返回附近POI类型
	Radius     int    `json:"radius,omitempty"`     // 可选：搜索半径，取值范围0~3000，默认1000
	Extensions string `json:"extensions,omitempty"` // 可选：返回结果控制，base或all，默认base
	Roadlevel  int    `json:"roadlevel,omitempty"`  // 可选：道路等级，0显示所有道路，1过滤非主干道路
	Output     string `json:"output,omitempty"`     // 可选：返回数据格式类型，默认JSON
	Callback   string `json:"callback,omitempty"`   // 可选：回调函数
	Homeorcorp int    `json:"homeorcorp,omitempty"` // 可选：是否优化POI返回顺序，0不干扰，1优化居家相关，2优化公司相关
	Sig        string `json:"sig,omitempty"`        // 可选：数字签名
}

// RegeoResponse 逆地理编码响应
type RegeoResponse struct {
	Status    string         `json:"status"`    // 返回状态：1成功，0失败
	Info      string         `json:"info"`      // 返回状态说明
	InfoCode  string         `json:"infocode"`  // 返回状态码
	Regeocode map[string]any `json:"regeocode"` // 逆地理编码信息
}

// Regeo 逆地理编码：将经纬度坐标转换为地址信息
func Regeo(req *RegeoRequest) (*RegeoResponse, error) {
	if req.Key == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if req.Location == "" {
		return nil, fmt.Errorf("location is required")
	}

	// 设置默认值
	if req.Output == "" {
		req.Output = "JSON"
	}
	if req.Radius == 0 {
		req.Radius = 3000
	}
	if req.Extensions == "" {
		req.Extensions = "all"
	}

	// 构建请求URL
	baseURL := "https://restapi.amap.com/v3/geocode/regeo"
	params := url.Values{}
	params.Set("key", req.Key)
	params.Set("location", req.Location)
	params.Set("output", req.Output)
	params.Set("radius", fmt.Sprintf("%d", req.Radius))
	params.Set("extensions", req.Extensions)

	if req.Poitype != "" {
		params.Set("poitype", req.Poitype)
	}
	if req.Roadlevel != 0 {
		params.Set("roadlevel", fmt.Sprintf("%d", req.Roadlevel))
	}
	if req.Callback != "" {
		params.Set("callback", req.Callback)
	}
	if req.Homeorcorp != 0 {
		params.Set("homeorcorp", fmt.Sprintf("%d", req.Homeorcorp))
	}
	if req.Sig != "" {
		params.Set("sig", req.Sig)
	}

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// 调试：输出请求URL
	fmt.Printf("Request URL: %s\n", fullURL)

	// 发送HTTP请求
	resp, err := httpClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	println(string(bodyData))

	var regeoResp RegeoResponse
	if err := json.Unmarshal(bodyData, &regeoResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &regeoResp, nil
}

// RegeoSimple 简化的逆地理编码方法，只需要经纬度坐标和可选参数
func RegeoSimple(key string, location string, options ...RegeoOption) (*RegeoResponse, error) {
	req := &RegeoRequest{
		Key:      key,
		Location: location,
	}

	// 应用可选参数
	for _, opt := range options {
		opt(req)
	}

	return Regeo(req)
}

// RegeoOption 逆地理编码可选参数
type RegeoOption func(*RegeoRequest)

// WithRadius 设置搜索半径
func WithRadius(radius int) RegeoOption {
	return func(req *RegeoRequest) {
		req.Radius = radius
	}
}

// WithExtensions 设置返回结果控制
func WithExtensions(extensions string) RegeoOption {
	return func(req *RegeoRequest) {
		req.Extensions = extensions
	}
}

// WithPoitype 设置POI类型
func WithPoitype(poitype string) RegeoOption {
	return func(req *RegeoRequest) {
		req.Poitype = poitype
	}
}

// WithRoadlevel 设置道路等级
func WithRoadlevel(roadlevel int) RegeoOption {
	return func(req *RegeoRequest) {
		req.Roadlevel = roadlevel
	}
}

// WithHomeorcorp 设置POI返回顺序优化
func WithHomeorcorp(homeorcorp int) RegeoOption {
	return func(req *RegeoRequest) {
		req.Homeorcorp = homeorcorp
	}
}

// RegeoWithPOI 简化的逆地理编码方法，默认返回POI信息
func RegeoWithPOI(key string, location string, options ...RegeoOption) (*RegeoResponse, error) {
	req := &RegeoRequest{
		Key:        key,
		Location:   location,
		Extensions: "all", // 强制设置为 all 以返回POI信息
	}

	// 应用可选参数
	for _, opt := range options {
		opt(req)
	}

	return Regeo(req)
}

