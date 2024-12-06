package huya

import (
	"fmt"
	"github.com/goccy/go-json"
	"io/ioutil"
	"net/http"
)

// Response struct 用于解析JSON响应
type Response struct {
	Code int `json:"code"`
	Data struct {
		UID    int    `json:"uid"`
		Avatar string `json:"avatar"`
	} `json:"data"`
	Message string `json:"message"`
}

// GetAvatarByUID 通过UID获取用户头像URL
func GetAvatarByUID(uid string) (string, error) {
	url := fmt.Sprintf("https://user.huya.com/user/getUserInfo?uid=%s", uid)

	// 发送GET请求
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 解析JSON响应
	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	// 检查返回code是否为200
	if response.Code != 200 {
		return "", fmt.Errorf("request failed with message: %s", response.Message)
	}

	return response.Data.Avatar, nil
}
