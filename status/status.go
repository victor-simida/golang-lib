package status

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
	"violate/config"
	"violate/mylog"
	"violate/proxy/AES_ECB"
	"violate/proxy/KAFKA"
	"violate/status/json_private"
	"strings"
)

var ipAddress string
var projectName string

var logKafka sarama.AsyncProducer
var statusKafka sarama.AsyncProducer

func Initalize() {

	filePath, _ := exec.LookPath(os.Args[0])
	//文件名
	projectName = strings.ToLower(filepath.Base(filePath))

	ipAddress = "0.0.0.0"
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		mylog.LOG.E(err.Error())
		return
	}

	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ipAddress = ipnet.IP.String()
				break
			}
		}
	}

	logKafka, err = KAFKA.InitAsyncProducer(config.Settings.KafkaHost, "biz-log")
	if err != nil {
		panic(fmt.Sprintf("log kafka init:%s", err.Error()))
	}
	statusKafka, err = KAFKA.InitAsyncProducer(config.Settings.KafkaHost, "monitor-status")
	if err != nil {
		panic(fmt.Sprintf("status kafka init:%s", err.Error()))
	}
}

/*******************************************
*函数名：Init
*作用：填充结构体
*作者:liziang061
*时间：2017/5/11 10:55
*******************************************/
func (this *Status_t) Init(req *http.Request) {
	this.RWMutex = new(sync.RWMutex)
	this.StartT = time.Now().UnixNano() / int64(time.Millisecond)
	this.Uri = strings.Split(req.RequestURI,"?")[0]
	this.RequestMethod = req.Method
	this.InParam = make(map[string]interface{})
	this.OutParam = make(map[string]interface{})
	this.ChildStatus = make([]BasicStatus_t, 0)
	if this.getTraceInfo(req) {
		this.ServiceType = ServiceTypeHttp //如果上游有传入traceinfo，则表明为http
	} else {
		this.ServiceType = ServiceTypeUserCall //否则是用户传入的，表明为web
	}
	this.parseSpartaId(req)
	this.NewLogShell()
	this.Project = projectName
	this.ServerAddress = ipAddress + ":" + config.Settings.ServerPort
}

/*******************************************
*函数名： http 子服务的总耗时add
*作用：AddHttpMillis
*作者:liziang061
*时间：2017/5/11 10:46
*******************************************/
func (this *Status_t) AddHttpMillis(t int64) {
	this.HttpMillis += int64(t) / int64(time.Millisecond)
}

/*******************************************
*函数名：调用 sql 子服务的总耗时add
*作用：AddSqlMillis
*作者:liziang061
*时间：2017/5/11 10:46
*******************************************/
func (this *Status_t) AddSqlMillis(t int64) {
	this.SqlMillis += int64(t) / int64(time.Millisecond)
}

/*******************************************
*函数名：AddExpCount
*作用：增加异常计数
*作者:liziang061
*时间：2017/5/11 14:30
*******************************************/
func (this *BasicStatus_t) AddExpCount(count int) {
	this.ExpCount = count
}

/*******************************************
*函数名：AddExp
*作用：增加异常信息
*作者:liziang061
*时间：2017/5/11 14:31
*******************************************/
func (this *BasicStatus_t) SetExp(exp string) {
	this.Exp = exp
}

/*******************************************
*函数名：SetDurMillis
*作用：设置耗时时长
*作者:liziang061
*时间：2017/5/11 16:27
*******************************************/
func (this *BasicStatus_t) SetDurMillis(durmillis int64) {
	this.DurMillis = int64(durmillis) / int64(time.Millisecond)
}

/*******************************************
*函数名：SetStartT
*作用：设置开始时间
*作者:liziang061
*时间：2017/5/11 16:27
*******************************************/
func (this *BasicStatus_t) SetStartT(t time.Time) {
	this.StartT = t.UnixNano() / int64(time.Millisecond)
}

/*******************************************
*函数名：SetStartTWithNow
*作用：设置开始时间，无需外界传入当前时间
*作者:liziang061
*时间：2017/5/11 14:31
*******************************************/
func (this *BasicStatus_t) SetStartTWithNow() {
	this.StartT = time.Now().UnixNano() / int64(time.Millisecond)
}

/*******************************************
*函数名：SetEndTime
*作用：设置结束时间，如果外部已经存在有当前时间，请使用此方法，以节约调用系统时间的开销
*作者:liziang061
*时间：2017/5/11 14:31
*******************************************/
func (this *BasicStatus_t) SetEndTime(t time.Time) {
	this.EndT = t.UnixNano() / int64(time.Millisecond)
}

func (this *TraceInfo_t) ToString() string {
	out, _ := json.Marshal(this)
	return string(out)
}

/*******************************************
*函数名：SetEndTimeWithNow
*作用：设置结束时间，无需外界传入当前时间
*作者:liziang061
*时间：2017/5/11 14:31
*******************************************/
func (this *BasicStatus_t) SetEndTimeWithNow() {
	this.EndT = time.Now().UnixNano() / int64(time.Millisecond)
}

/*******************************************
*函数名：AddChildStatus
*作用：增加子status，将index返回,方便业务方填充数据
*作者:liziang061
*时间：2017/5/11 17:28
*******************************************/
func (this *Status_t) AddChildStatus() int {
	this.Lock()
	defer this.Unlock()

	var temp BasicStatus_t
	temp.SetStartTWithNow()
	temp.TraceInfo = this.TraceInfo
	temp.TraceInfo.SpanId = fmt.Sprintf("%s.%d", this.TraceInfo.SpanId, len(this.ChildStatus)+1)
	temp.InParam = make(map[string]interface{})
	temp.OutParam = make(map[string]interface{})
	temp.RWMutex = new(sync.RWMutex)
	this.ChildStatus = append(this.ChildStatus, temp)

	return len(this.ChildStatus) - 1
}

/*******************************************
*函数名：getTraceInfo
*作用：从req中获取traceinfo，如果没有则生成之,返回bool表明了是否上游有传入accioTraceInfo头部
*作者:liziang061
*时间：2017/5/11 14:32
*******************************************/
func (this *Status_t) getTraceInfo(req *http.Request) bool {
	accioTraceInfo := req.Header.Get("accioTraceInfo")
	var temp TraceInfo_t
	if accioTraceInfo != "" {
		err := json.Unmarshal([]byte(accioTraceInfo), &temp)
		if err != nil {
			mylog.LOG.E(err.Error())
			goto NO_TRACE_INFO
		}
		this.TraceInfo = temp
		return true
	}

NO_TRACE_INFO:
	temp.TraceId = generateTraceId(req)
	temp.AopsId = req.Header.Get("aopsId")
	temp.Phone = req.Header.Get("phone") //todo 关于电话号码的头部key未定
	temp.SpanId = "1"
	temp.Project = projectName
	this.TraceInfo = temp

	return false

}

/*******************************************
*函数名：generateTraceId
*作用：生成trace id
*作者:liziang061
*时间：2017/5/11 14:28
*******************************************/
func generateTraceId(req *http.Request) string {
	now := time.Now()
	day := now.Day()
	minute := now.Minute()
	hour := now.Hour()
	second := now.Second()
	nano := now.Nanosecond()

	return fmt.Sprintf("%02d-%02d:%02d:%02d.%03d-%s-%s-%s-%p", day, hour, minute, second, nano/int(time.Millisecond), projectName,
		strings.Split(req.RequestURI,"?")[0],
		ipAddress, req)
}

/*******************************************
*函数名：parseSpartaId
*作用：从头部获取spartaid
*作者:liziang061
*时间：2017/5/11 16:06
*******************************************/
func (this *Status_t) parseSpartaId(req *http.Request) {
	src := req.Header.Get("spartaId")
	if src == "" {
		return
	}

	dst, err := base64.StdEncoding.DecodeString(src)
	if err != nil {
		return
	}

	out, err := AES_ECB.AesDecrypt(dst, []byte(config.Settings.SpartaIdAesKey))
	if err != nil {
		return
	}

	var temp OriginSpartaId
	err = json.Unmarshal(out, &temp)
	if err != nil {
		return
	}

	this.ClientInfo.Gps = temp.Gps
	this.ClientInfo.Network = temp.NetWork
	this.ClientInfo.OsVersionCode = temp.OsVersionCode
	this.ClientInfo.OsVersionName = temp.OsVersionName
	this.ClientInfo.Mnc = temp.MNC
	this.ClientInfo.VersionCode = temp.VersionCode

	/*安卓和ios的区分*/
	if temp.OsVersionName == "A" {
		this.ClientInfo.DeviceId = temp.Data.DeviceIdAndroid
	} else if temp.OsVersionName == "I" {
		this.ClientInfo.DeviceId = temp.DeviceIdIos
	}
}

/*******************************************
*函数名：ToString
*作用：转字符串方法
*作者:liziang061
*时间：2017/5/11 17:36
*******************************************/
func (this *Status_t) ToString() string {
	this.RLock()
	defer this.RUnlock()
	out, _ := json.Marshal(this)
	return string(out)
}

/*******************************************
*函数名：Info
*作用：此函数会记录Info log并使用对应的baseStatus的数据结构上报Info log信息到kafka
*作者:liziang061
*时间：2017/5/17 20:14
*******************************************/
func (this *Status_t) Info(format string, params ...interface{}) {
	this.thisLog.log.Info(format, params...)
	_, filename, line, _ := runtime.Caller(1)
	logStr := fmt.Sprintf(format, params...)
	now := time.Now()
	KAFKA.SendMsg(logKafka, "biz-log", fmt.Sprintf("%s %s.%03d [*] INFO [%v:%v] %s", this.TraceInfo.ToString(), now.Format("2006-01-02 15:04:05"),
		now.Nanosecond()/int(time.Millisecond), filepath.Base(filename), line, logStr))
}

func (this *Status_t) I(format string, params ...interface{}) {
	this.thisLog.log.Info(format, params...)
	_, filename, line, _ := runtime.Caller(1)
	logStr := fmt.Sprintf(format, params...)
	now := time.Now()
	KAFKA.SendMsg(logKafka, "biz-log", fmt.Sprintf("%s %s.%03d [*] INFO [%v:%v] %s", this.TraceInfo.ToString(), now.Format("2006-01-02 15:04:05"),
		now.Nanosecond()/int(time.Millisecond), filepath.Base(filename), line, logStr))
}

/*******************************************
*函数名：Error
*作用：此函数会记录Error log并使用对应的baseStatus的数据结构上报Error log信息到kafka
*作者:liziang061
*时间：2017/5/17 20:14
*******************************************/
func (this *Status_t) Error(format string, params ...interface{}) {
	this.thisLog.log.Error(format, params...)
	//todo kafka
	_, filename, line, _ := runtime.Caller(1)
	logStr := fmt.Sprintf(format, params...)
	now := time.Now()
	KAFKA.SendMsg(logKafka, "biz-log", fmt.Sprintf("%s %s.%03d [*] ERROR [%v:%v] %s", this.TraceInfo.ToString(), now.Format("2006-01-02 15:04:05"),
		now.Nanosecond()/int(time.Millisecond), filepath.Base(filename), line, logStr))
}

func (this *Status_t) E(format string, params ...interface{}) {
	this.thisLog.log.Error(format, params...)
	//todo kafka
	_, filename, line, _ := runtime.Caller(1)
	logStr := fmt.Sprintf(format, params...)
	now := time.Now()
	KAFKA.SendMsg(logKafka, "biz-log", fmt.Sprintf("%s %s.%03d [*] ERROR [%v:%v] %s", this.TraceInfo.ToString(), now.Format("2006-01-02 15:04:05"),
		now.Nanosecond()/int(time.Millisecond), filepath.Base(filename), line, logStr))
}

/*******************************************
*函数名：Warn
*作用：Warn log记录及kafka上报
*作者:liziang061
*时间：2017/5/19 9:37
*******************************************/
func (this *Status_t) Warn(format string, params ...interface{}) {
	this.thisLog.log.Warn(format, params...)
	_, filename, line, _ := runtime.Caller(1)
	logStr := fmt.Sprintf(format, params...)
	now := time.Now()
	KAFKA.SendMsg(logKafka, "biz-log", fmt.Sprintf("%s %s.%03d [*] WARN [%v:%v] %s", this.TraceInfo.ToString(), now.Format("2006-01-02 15:04:05"),
		now.Nanosecond()/int(time.Millisecond), filepath.Base(filename), line, logStr))
}

func (this *Status_t) W(format string, params ...interface{}) {
	this.thisLog.log.Warn(format, params...)
	_, filename, line, _ := runtime.Caller(1)
	logStr := fmt.Sprintf(format, params...)
	now := time.Now()
	KAFKA.SendMsg(logKafka, "biz-log", fmt.Sprintf("%s %s.%03d [*] WARN [%v:%v] %s", this.TraceInfo.ToString(), now.Format("2006-01-02 15:04:05"),
		now.Nanosecond()/int(time.Millisecond), filepath.Base(filename), line, logStr))
}

/*******************************************
*函数名：Critical
*作用：Critical log记录及kafka上报
*作者:liziang061
*时间：2017/5/19 9:37
*******************************************/
func (this *Status_t) Critical(format string, params ...interface{}) {
	this.thisLog.log.Critical(format, params...)
	_, filename, line, _ := runtime.Caller(1)
	logStr := fmt.Sprintf(format, params...)
	now := time.Now()
	KAFKA.SendMsg(logKafka, "biz-log", fmt.Sprintf("%s %s.%03d [*] CRITICAL [%v:%v] %s", this.TraceInfo.ToString(), now.Format("2006-01-02 15:04:05"),
		now.Nanosecond()/int(time.Millisecond), filepath.Base(filename), line, logStr))
}

func (this *Status_t) C(format string, params ...interface{}) {
	this.thisLog.log.Critical(format, params...)
	_, filename, line, _ := runtime.Caller(1)
	logStr := fmt.Sprintf(format, params...)
	now := time.Now()
	KAFKA.SendMsg(logKafka, "biz-log", fmt.Sprintf("%s %s.%03d [*] CRITICAL [%v:%v] %s", this.TraceInfo.ToString(), now.Format("2006-01-02 15:04:05"),
		now.Nanosecond()/int(time.Millisecond), filepath.Base(filename), line, logStr))
}

/*******************************************
*函数名：Debug
*作用：Debug log记录及kafka上报
*作者:liziang061
*时间：2017/5/19 9:44
*******************************************/
func (this *Status_t) Debug(format string, params ...interface{}) {
	this.thisLog.log.Debug(format, params...)
	_, filename, line, _ := runtime.Caller(1)
	logStr := fmt.Sprintf(format, params...)
	now := time.Now()
	KAFKA.SendMsg(logKafka, "biz-log", fmt.Sprintf("%s %s.%03d [*] DEBUG [%v:%v] %s", this.TraceInfo.ToString(), now.Format("2006-01-02 15:04:05"),
		now.Nanosecond()/int(time.Millisecond), filepath.Base(filename), line, logStr))
}

func (this *Status_t) D(format string, params ...interface{}) {
	this.thisLog.log.Debug(format, params...)
	_, filename, line, _ := runtime.Caller(1)
	logStr := fmt.Sprintf(format, params...)
	now := time.Now()
	KAFKA.SendMsg(logKafka, "biz-log", fmt.Sprintf("%s %s.%03d [*] DEBUG [%v:%v] %s", this.TraceInfo.ToString(), now.Format("2006-01-02 15:04:05"),
		now.Nanosecond()/int(time.Millisecond), filepath.Base(filename), line, logStr))
}

func (this *Status_t) NewLogShell() {
	var temp LogShell
	temp.log = mylog.LOG //todo
	temp.parent = &this.BasicStatus_t
	this.thisLog = temp
}

func (this *Status_t) SendToKafka() {
	KAFKA.SendMsg(statusKafka, "monitor-status", this.ToString())
}

/*******************************************
*函数名：WriteOutParam
*作用：填充出参
*作者:liziang061
*时间：2017/5/24 17:29
*******************************************/
func (this *BasicStatus_t) WriteParam(out interface{}) {
	if out != nil {
		outByte , err := json_private.Marshal(out)
		if err != nil {
			return
		}
		temp := make(map[string]interface{})
		err = json_private.Unmarshal(outByte, &temp)
		if err != nil {
			return
		}
		this.OutParam = temp
	}
}

func (this *BasicStatus_t) WriteInParam(in interface{}) {
	if in != nil {
		inByte, err := json_private.Marshal(in)
		if err != nil {
			return
		}
		temp := make(map[string]interface{})
		err = json_private.Unmarshal(inByte, &temp)
		if err != nil {
			return
		}
		this.InParam = temp
	}
}
