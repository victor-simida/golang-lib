package proxy

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
	"violate/config"
	"violate/mylog"
	"net"
	"context"
)

var httpClient *http.Client
func init() {
	httpClient = http.DefaultClient
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second, }
	transport.DisableCompression = true

	httpClient.Transport = transport
}

const PrivateTimeout = "private_time"
func Get(url string) (body []byte, statusCode int) {
	return req(url, `GET`, nil, config.Settings.HttpTimeout)
}
func GetWithHeaders(url string, headers map[string]string, timeout int64) (body []byte, statusCode int) {
	return reqJsonCustomHeader(url, "GET", nil, timeout, headers)
}

func GetWithTimeout(url string, timeout int64) (body []byte, statusCode int) {
	return req(url, `GET`, nil, timeout)
}


func Post(url string, data url.Values) (body []byte, statusCode int) {
	return req(url, "POST", data, config.Settings.HttpTimeout)
}

func PostForAops(url string, data url.Values) (body []byte, statusCode int) {
	return reqForAops(url, "POST", data, config.Settings.HttpTimeout)
}

func PostWithTimeout(url string, data url.Values, timeout int64) (body []byte, statusCode int) {
	return req(url, "POST", data, timeout)
}

func PostJson(url string, data []byte) (body []byte, statusCode int) {
	return reqJson(url, "POST", data, config.Settings.HttpTimeout)
}
func PostJsonWithTimeout(url string, data []byte, timeout int64) (body []byte, statusCode int) {
	return reqJson(url, "POST", data, timeout)
}

func PostFile(url string, data string) (body []byte, statusCode int) {
	return reqFile(url, `POST`, data, config.Settings.HttpTimeout)
}
func PostFileWithTimeout(url string, data string, timeout int64) (body []byte, statusCode int) {
	return reqFile(url, `POST`, data, timeout)
}

func PostFileWithHeader(url string, data string, timeout int64, headMethod string) (body []byte, statusCode int) {
	return reqFileCustom(url, `POST`, data, timeout, headMethod)
}

func PostJsonWithSetter(url string, data []byte, setter ...func(*http.Request)) (body []byte,  statusCode int) {
	return reqJsonSetter(url, "POST", data, setter...)
}

/*******************************************
*函数名：PostJsonServiceWindowWithTimeout
*作用：http请求 for ServiceWindow
*时间：2017/03/16 13:43
*******************************************/
func PostJsonServiceWindowWithTimeout(url string, data []byte, timeout int64) (body []byte, statusCode int) {
	return reqServiceWindowJson(url, "POST", data, timeout)
}

/*******************************************
*函数名：reqJson
*作用：http请求 for json
*时间：2016/9/20 10:43
*******************************************/
func reqJsonCustomHeader(url string, method string, data []byte, timeout int64, headers map[string]string) (body []byte, statusCode int) {
	request, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		mylog.LOG.E("reqForCustomHeader: NewRequest failed: %v", err.Error())
		return
	}
	if method == "POST" || method == "PUT" || method == "DELETE" {
		request.Header.Set("Content-Type", "application/json")
		request.Header.Add("Accept-Charset", "UTF-8")
	}

	for k, v := range headers {
		request.Header.Add(k, v)
	}
	return reqInner(request, timeout)
}


func reqJson(url string, method string, data []byte, timeout int64) (body []byte, statusCode int) {
	mylog.LOG.I("reqJson input data:%s", string(data))
	request, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		mylog.LOG.E("reqJson: NewRequest failed: %v", err.Error())
		return
	}
	if method == "POST" || method == "PUT" || method == "DELETE" {
		request.Header.Set("Content-Type", "application/json")
		request.Header.Add("Accept-Charset", "UTF-8")
	}

	return reqInner(request, timeout)
}


/*******************************************
*函数名：reqServiceWindowJson
*作用：http请求 for ServiceWindow
*时间：2017/03/16 13:43
*******************************************/
func reqServiceWindowJson(url string, method string, data []byte, timeout int64) (body []byte, statusCode int) {
	mylog.LOG.I("reqJson input data:%s", string(data))
	request, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		mylog.LOG.E("reqJson: NewRequest failed: %v", err.Error())
		return
	}
	if method == "POST" || method == "PUT" || method == "DELETE" {
		request.Header.Set("Content-Type", "application/json")
		request.Header.Add("Accept-Charset", "UTF-8")
		request.Header.Add("access_token", config.Settings.SWAccessToken)
		request.Header["access_token"] = []string{config.Settings.SWAccessToken}
	}

	return reqInner(request, timeout)
}

/*******************************************
*函数名：reqJsonSetter
*作用：http请求 for json
*时间：2016/9/20 10:43
*******************************************/
func reqJsonSetter(url string, method string, data []byte, setters ...func(*http.Request)) (body []byte, statusCode int) {
	mylog.LOG.I("reqJson input data:%s", string(data))
	request, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		mylog.LOG.E("reqJson: NewRequest failed: %v", err.Error())
		return
	}
	if method == "POST" || method == "PUT" || method == "DELETE" {
		request.Header.Set("Content-Type", "application/json")
		request.Header.Add("Accept-Charset", "UTF-8")
	}

	for _, setter := range setters {
		setter(request)
	}

	return reqInner(request, 0)
}

/*******************************************
*函数名：req
*作用：http请求
*时间：2016/9/20 10:43
*******************************************/
func req(url string, method string, data url.Values, timeout int64) (body []byte, statusCode int) {
	request, err := http.NewRequest(method, url, strings.NewReader(data.Encode()))
	if err != nil {
		return
	}
	if method == "POST" || method == "PUT" {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Add("Accept-Charset", "UTF-8")
	}
	return reqInner(request, timeout)
}


/*******************************************
*函数名：reqForAops
*作用：aops接口需要增加认证头部
*时间：2016/9/20 10:43
*******************************************/
func reqForAops(url string, method string, data url.Values, timeout int64) (body []byte, statusCode int) {
	request, err := http.NewRequest(method, url, strings.NewReader(data.Encode()))
	if err != nil {
		return
	}
	if method == "POST" || method == "PUT" {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Add("Accept-Charset", "UTF-8")
	}

	/*aops需要携带下面的认证信息*/
	request.Header.Add("Authorization", "Basic " + config.Settings.Authorization)
	return reqInner(request, timeout)
}

func reqFile(url string, method string, data string, timeout int64) (body []byte, statusCode int) {
	request, err := http.NewRequest(method, url, strings.NewReader(data))
	if err != nil {
		mylog.LOG.E("reqFile err:%s. %s url:%s, data:%s", err, method, url, data)
		return
	}

	request.Header.Set("Content-Type", `text/xml;charset=UTF-8`)
	request.Header.Add("Accept", `application/soap+xml, application/dime, multipart/related, text/*`)
	return reqInner(request, timeout)
}

func reqFileCustom(url string, method string, data string, timeout int64, headMethod string) (body []byte, statusCode int) {
	request, err := http.NewRequest(method, url, strings.NewReader(data))
	if err != nil {
		mylog.LOG.E("reqFile err:%s. %s url:%s, data:%s", err, method, url, data)
		return
	}

	request.Header.Set("Content-Type", `text/xml;charset=UTF-8`)
	request.Header.Add("Accept", `application/soap+xml, application/dime, multipart/related, text/*`)
	request.Header.Add("SOAPAction", headMethod)
	return reqInner(request, timeout)
}

func reqInner(request *http.Request, timeout int64) (body []byte, statusCode int) {
	var (
		err error
		res *http.Response
	)

	if timeout == 0 {
		timeout = config.Settings.HttpTimeout
	}

	mylog.LOG.I("reqInner: request url %v", request.URL.String())

	ctx, _ := context.WithTimeout(context.TODO(), time.Duration(timeout) * time.Second)
	request = request.WithContext(ctx)
	res, err = httpClient.Do(request)
	if err != nil {
		mylog.LOG.E("reqInner httpClient.Do failed: %v", err.Error())
		return
	}

	body, err = ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		mylog.LOG.E("reqInner ioutil.ReadAll(res.Body) failed: %v", err.Error())
	}

	mylog.LOG.I("reqInner Resbody: %v", string(body))
	statusCode = res.StatusCode

	return
}
