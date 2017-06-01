package THREAD_MANAGER

import (
	"sync"
	"violate/mylog"
)

var getPictureThreadChan chan int
var getPictureWG sync.WaitGroup

func init() {
	getPictureThreadChan = make(chan int, 0)
}

func Stop() {
	mylog.LOG.I("getPictureThreadChan length:%v",len(getPictureThreadChan))
	close(getPictureThreadChan)
}

func IsStop() bool {
	select {
	case <-getPictureThreadChan:
		mylog.LOG.I("getPictureThreadChan Closed")
		return true
	default:
		return false
	}
}

func Add() {
	getPictureWG.Add(1)
	mylog.LOG.I("getPictureWG Add")
}

func Done() {
	getPictureWG.Done()
	mylog.LOG.I("getPictureWG Done")
}

func Wait() {
	getPictureWG.Wait()
	mylog.LOG.I("getPictureWG Wait Exited")
}
