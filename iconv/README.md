# iconv
```go
/*******************************************
*函数名：IconvGbkToUtf8
*作用：转换gbk字符至utf-8
*时间：2017/2/7 14:21
*******************************************/
func IconvGbkToUtf8(input string) (string, error) {
	cd, err := iconv.Open("utf-8", "gbk")
	if err != nil {
		mylog.LOG.E("IconvGbkToUtf8 Error:%s", err.Error())
		return "", err
	}
	defer cd.Close()
	result := cd.ConvString(input)
	return result, nil
}


/*******************************************
*函数名：IconvUtf8toGbk
*作用：转换utf-8字符至gbk
*时间：2017/2/7 14:21
*******************************************/
func IconvUtf8toGbk(input string) (string, error) {
	cd, err := iconv.Open("gbk", "utf-8")
	if err != nil {
		mylog.LOG.E("IconvUtf8toGbk Error:%s", err.Error())
		return "", err
	}
	defer cd.Close()
	result := cd.ConvString(input)
	return result, nil
}

```