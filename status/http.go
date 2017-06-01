package status

import (
	"bytes"
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"violate/config"
	"violate/mylog"
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
		TLSHandshakeTimeout: 10 * time.Second}
	transport.DisableCompression = true

	httpClient.Transport = transport
}

func (this *Status_t) Get(url string, setters ...func(*http.Request)) (body []byte, statusCode int) {
	return this.req(url, `GET`, nil, config.Settings.HttpTimeout, setters...)
}

func (this *Status_t) GetWithTimeout(url string, timeout int64, setters ...func(*http.Request)) (body []byte, statusCode int) {
	return this.req(url, `GET`, nil, timeout, setters...)
}

func (this *Status_t) Post(url string, data url.Values, setters ...func(*http.Request)) (body []byte, statusCode int) {
	return this.req(url, "POST", data, config.Settings.HttpTimeout, setters...)
}

func (this *Status_t) PostWithTimeout(url string, data url.Values, timeout int64, setters ...func(*http.Request)) (body []byte, statusCode int) {
	return this.req(url, "POST", data, timeout, setters...)
}

func (this *Status_t) PostJson(url string, data []byte, setters ...func(*http.Request)) (body []byte, statusCode int) {
	return this.reqJson(url, "POST", data, config.Settings.HttpTimeout, setters...)
}
func (this *Status_t) PostJsonWithTimeout(url string, data []byte, timeout int64, setters ...func(*http.Request)) (body []byte, statusCode int) {
	return this.reqJson(url, "POST", data, timeout, setters...)
}

func (this *Status_t) PostFile(url string, data string, setters ...func(*http.Request)) (body []byte, statusCode int) {
	return this.reqFile(url, `POST`, data, config.Settings.HttpTimeout, setters...)
}
func (this *Status_t) PostFileWithTimeout(url string, data string, timeout int64, setters ...func(*http.Request)) (body []byte, statusCode int) {
	return this.reqFile(url, `POST`, data, timeout, setters...)
}

func (this *Status_t) reqJson(url string, method string, data []byte, timeout int64, setters ...func(*http.Request)) (body []byte, statusCode int) {
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

    /*status统计*/
    ts := time.Now()
    childStatusIndex := this.AddChildStatus()
	if data != nil {
		this.ChildStatus[childStatusIndex].InParam[ParamSingle] = string(data)
	}
    request.Header.Add("accioTraceInfo", this.ChildStatus[childStatusIndex].TraceInfo.ToString())
	body, statusCode ,err = this.reqInner(request, timeout)
	defer  this.httpReport(ts, childStatusIndex, request, body, statusCode, err)
	return
}

/*******************************************
*函数名：req
*作用：http请求
*时间：2016/9/20 10:43
*******************************************/
func (this *Status_t) req(url string, method string, data url.Values, timeout int64, setters ...func(*http.Request)) (body []byte, statusCode int) {
	request, err := http.NewRequest(method, url, strings.NewReader(data.Encode()))
	if err != nil {
		return
	}
	if method == "POST" || method == "PUT" {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Add("Accept-Charset", "UTF-8")
	}

    for _, setter := range setters {
        setter(request)
    }

    /*status统计*/
    ts := time.Now()
    childStatusIndex := this.AddChildStatus()
	if data != nil {
		this.ChildStatus[childStatusIndex].WriteInParam(data)
	} else if method == "GET" {
		value := request.URL.Query()
		this.ChildStatus[childStatusIndex].WriteInParam(value)
	}
    request.Header.Add("accioTraceInfo", this.ChildStatus[childStatusIndex].TraceInfo.ToString())

	body, statusCode ,err = this.reqInner(request, timeout)
	defer  this.httpReport(ts, childStatusIndex, request, body, statusCode, err)
	return
}

func (this *Status_t) reqFile(url string, method string, data string, timeout int64, setters ...func(*http.Request)) (body []byte, statusCode int) {

    request, err := http.NewRequest(method, url, strings.NewReader(data))
    if err != nil {
        mylog.LOG.E("reqFile err:%s. %s url:%s, data:%s", err, method, url, data)
        return
    }

    /*status统计*/
    ts := time.Now()
    childStatusIndex := this.AddChildStatus()
    this.ChildStatus[childStatusIndex].InParam[ParamSingle] = string(data)
    request.Header.Add("accioTraceInfo", this.ChildStatus[childStatusIndex].TraceInfo.ToString())

	request.Header.Set("Content-Type", `text/xml;charset=UTF-8`)
	request.Header.Add("Accept", `application/soap+xml, application/dime, multipart/related, text/*`)
	body, statusCode ,err = this.reqInner(request, timeout)
	defer  this.httpReport(ts, childStatusIndex, request, body, statusCode, err)
	return
}


func (this *Status_t) reqInner(request *http.Request, timeout int64) (body []byte, statusCode int,err error) {
	var (
		res *http.Response
	)

	if timeout == 0 {
		timeout = config.Settings.HttpTimeout
	}

	mylog.LOG.I("reqInner: request url %v", request.URL.String())

	ctx, _ := context.WithTimeout(context.TODO(), time.Duration(timeout)*time.Second)
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
		return
	}

	mylog.LOG.I("reqInner Resbody: %v", string(body))
	statusCode = res.StatusCode

	return
}


func (this *Status_t)httpReport(startTime time.Time, childStatusIndex int, request *http.Request, output []byte, statusCode int, err error) {
    if this != nil {
        durMillis := int64(time.Since(startTime))
        this.AddHttpMillis(durMillis)
        this.ChildStatus[childStatusIndex].Uri = strings.Split(request.URL.String(), "?")[0]
        this.ChildStatus[childStatusIndex].RequestMethod = request.Method
		if output != nil {
			this.ChildStatus[childStatusIndex].OutParam[ParamSingle] = string(output)
			this.ChildStatus[childStatusIndex].OutParam[OutParamStatusCode] = statusCode
		}
		if err != nil {
			this.ChildStatus[childStatusIndex].SetExp(err.Error())
			this.ChildStatus[childStatusIndex].AddExpCount(1)
		}
        this.ChildStatus[childStatusIndex].SetEndTimeWithNow()
        this.ChildStatus[childStatusIndex].SetDurMillis(durMillis)
        this.ChildStatus[childStatusIndex].ServiceType = ServiceTypeHttp
    }
}