// @Author zhangjiaozhu 2024/1/17 16:23:00
package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"zinx/ziface"
)

type GlobalObj struct {
	TcpServer ziface.IServer //当前Zinx的全局Server对象
	Host      string         //当前服务器主机IP
	TcpPort   int            //当前服务器主机监听端口号
	Name      string         //当前服务器名称
	Version   string         //当前Zinx版本号

	MaxPacketSize uint32 //都需数据包的最大值
	MaxConn       int    //当前服务器主机允许的最大链接个数
	WorkerPoolSize   uint32 //业务工作Worker池的数量
	MaxWorkerTaskLen uint32 //业务工作Worker对应负责的任务队列最大任务存储数量
	ConfFilePath string
	MaxMsgChanLen int //连接的读写通道缓冲区大小
}

/*
	定义一个全局的对象
*/
var GlobalObject *GlobalObj


// 读取配置文件
func (g *GlobalObj)Reload()  {
	dir, _ := os.Getwd()
	fmt.Println(dir)
	file, err := ioutil.ReadFile(dir +"/conf/zinx.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(file,&GlobalObject)
	if err != nil {
		panic(err)
	}
}

/*
	提供init方法，默认加载
*/
func init() {
	//初始化GlobalObject变量，设置一些默认值
	GlobalObject = &GlobalObj{
		Name:    "ZinxServerApp",
		Version: "V0.4",
		TcpPort: 7777,
		Host:    "0.0.0.0",
		MaxConn: 12000,
		MaxPacketSize:4096,
		ConfFilePath:  "../conf/zinx.json",
		WorkerPoolSize: 10,
		MaxWorkerTaskLen: 1024,
		MaxMsgChanLen: 50,
	}

	//从配置文件中加载一些用户配置的参数
	GlobalObject.Reload()
}