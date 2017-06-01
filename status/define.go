package status

import (
	"violate/mylog"
	"sync"
	"database/sql"
)

const (
	ParamSingle = "singleParam(string)"
	OutParamStatusCode = "statusCode(int)"
	OutParamRowsAffected = "rowsAffected(int)"
)

const (
	ServiceTypeSql      = "sql"
	ServiceTypeUserCall = "web"  //用户调用
	ServiceTypeHttp     = "http" //http调用，存在上游
)

/*客户端上报的sparta id*/
type OriginSpartaId struct {
	MNC           string `json:"b4"`
	NetWork       string `json:"b1"`
	OsVersionCode string `json:"a5"`
	OsVersionName string `json:"a4"`
	VersionCode   string `json:"a2"`
	Gps           string `json:"a9"`
	Data          struct {
		DeviceIdAndroid string `json:"d8"`
	} `json:"b9"`
	DeviceIdIos string `json:"g4"`
}

type Status_t struct {
	BasicStatus_t                 /*基础数据类型*/
	Project       string          `json:"project"`       /*微服务名称*/
	ClientInfo    ClientInfo_t    `json:"clientInfo"`    /*客户端通用信息，从请求头部获取和 spartaId 解析获得*/
	ChildStatus   []BasicStatus_t `json:"childStatus"`   /*当前服务调用的子服务 list*/
	ServerAddress string          `json:"serverAddress"` /*机器地址，ip:port*/
	HttpMillis    int64           `json:"httpMillis"`    /*调用 http 子服务的总耗时*/
	SqlMillis     int64           `json:"sqlMillis"`     /*调用 sql 子服务的总耗时*/
	thisLog       LogShell
	thisMysql     mysqlStruct
}

type BasicStatus_t struct {
	*sync.RWMutex                        /*增加读写锁*/
	TraceInfo     TraceInfo_t            `json:"traceInfo"`     /*请求链信息，需要传递给子服务，header 统一叫 "accioTraceInfo"，并添加在日志前缀*/
	DurMillis     int64                  `json:"durMillis"`     /*整个请求总耗时,毫秒*/
	EndT          int64                  `json:"endT"`          /*请求结束时间戳*/
	Exp           string                 `json:"exp"`           /*请求异常信息，格式 "包名.类名.方法名(文件名:行号)>异常信息"	*/
	ExpCount      int                    `json:"expCount"`      /*为了方便后续合统计，有异常为 1，没有异常为 0	*/
	StartT        int64                  `json:"startT"`        /*请求开始时间戳*/
	ServiceType   string                 `json:"serviceType"`   /*当前服务的类型*/
	Uri           string                 `json:"uri"`           /*HTTP 协议规定的 URI*/
	InParam       map[string]interface{} `json:"inParam"`       /*请求的入参，若返回结果是基础数据类型或 String 类型，则手动添加一个 key 是 "singleParam"，value 是返回结果；其他类型解析成可读的 JSON 结构	*/
	OutParam      map[string]interface{} `json:"outParam"`      /*请求的出参，若返回结果是基础数据类型或 String 类型，则手动添加一个 key 是 "singleParam"，value 是返回结果；其他类型解析成可读的 JSON 结构	*/
	RequestMethod string                 `json:"requestMethod"` /*HTTP 方法，如 GET，POST*/
}

type TraceInfo_t struct {
	TraceId  string `json:"traceId"`  /*请求链 id，格式："dd-HH:mm:ss.SSS-服务名称-uri-机器ip-内存地址"	*/
	SpanId   string `json:"spanId"`   /*服务 id，根服务取值为 1，子服务取值由根服务生成，再传递给子服务，子 spanId 生成规则：父级的 spanId + "." + 子服务在 childStatus 的顺序	*/
	AopsId   string `json:"aopsId"`   /*aopsid*/
	Phone    string `json:"phone"`    /*电话号码*/
	SourceIp string `json:"sourceIp"` /*客户端的 ip*/
	Project  string `json:"project"`  /*微服务名称*/
}

type ClientInfo_t struct {
	Mnc           string `json:"mnc"`           /*从 http header 中提取 spartaId 字段，从中提取运营商	*/
	Network       string `json:"network"`       /*从 http header 中提取 spartaId 字段，从中提取网络类型	*/
	OsVersionCode string `json:"osVersionCode"` /*从 http header 中提取 spartaId 字段，从中提取系统版本号	*/
	OsVersionName string `json:"osVersionName"` /*从 http header 中提取 spartaId 字段，从中提取系统名称	*/
	VersionCode   string `json:"versionCode"`   /*从 http header 中提取 spartaId 字段，从中提取 app 版本号	*/
	Gps           string `json:"gps"`           /*从 http header 中提取 spartaId 字段，从中提取经纬度	*/
	DeviceId      string `json:"deviceId"`      /*从 http header 中提取 spartaId 字段，从中提取设备号，Android 是 d8 IMEI，IOS 是 g4 IDFA	*/
}

type LogShell struct {
	log    mylog.ILog
	parent *BasicStatus_t
}

type mysqlStruct struct {
	db *sql.DB
}