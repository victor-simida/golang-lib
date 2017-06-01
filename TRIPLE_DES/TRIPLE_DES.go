package TRIPLE_DES

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
	"encoding/base64"
	"strings"
	"violate/mylog"
)

/*******************************************
*函数名：TripleDesEncrypt
*作用：3DES加密,CBC方式
*时间：2016/6/29 14:30
*******************************************/
func TripleDesEncrypt(origData, key, iv []byte) ([]byte, error) {
	block, err := des.NewTripleDESCipher(key)
	if err != nil {
		return nil, err
	}
	origData = pkCS5Padding(origData, block.BlockSize())
	// origData = ZeroPadding(origData, block.BlockSize())
	blockMode := cipher.NewCBCEncrypter(block, iv)
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)

	result := base64.StdEncoding.EncodeToString(crypted)
	return []byte(result), nil
}

/*******************************************
*函数名：TripleDesDecrypt
*作用：3DES解密,CBC方式
*时间：2016/6/29 14:30
*******************************************/
func TripleDesDecrypt(crypted, key, iv []byte) ([]byte, error) {
	//去除无效字符空格，目前golang base64包存在此问题，没有将空格删除，导致b64解码的失败
	str := strings.Replace(string(crypted), " ", "", -1)

	crypted, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		mylog.LOG.E("DecodeString Error:%s", err.Error())
		return nil, err
	}

	block, err := des.NewTripleDESCipher(key)
	if err != nil {
		mylog.LOG.E("NewTripleDESCipher Error:%s", err.Error())
		return nil, err
	}
	blockMode := cipher.NewCBCDecrypter(block, iv)
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = pkCS5UnPadding(origData)
	return origData, nil
}

func pkCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext) % blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func pkCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	// 去掉最后一个字节 unpadding 次
	unpadding := int(origData[length - 1])
	return origData[:(length - unpadding)]
}
