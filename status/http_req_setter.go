package status

import (
	"net/http"
	"violate/config"
)

/*******************************************
*函数名：AopsHeaderSetter
*作用：设置aops请求认证头部setter
*作者:liziang061
*时间：2017/5/22 11:10
*******************************************/
func AopsHeaderSetter(request *http.Request) {
	request.Header.Add("Authorization", "Basic "+config.Settings.Authorization)
}

