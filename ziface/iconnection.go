// @Author zhangjiaozhu 2024/1/17 10:38:00
package ziface

import "net"

type IConnection interface {
	Start() //启动连接，让当前连接开始工作
	Stop()  //停止连接，结束当前连接状态M
	GetConnID() uint32 //获取当前连接ID
	GetTCPConnection() *net.TCPConn // //从当前连接获取原始的socket TCPConn
	//获取远程客户端地址信息
	RemoteAddr() net.Addr
	//直接将Message数据发送数据给远程的TCP客户端(无缓冲)
	SendMsg(msgId uint32, data []byte) error
	//直接将Message数据发送给远程的TCP客户端(有缓冲)
	SendBuffMsg(msgId uint32, data []byte) error   //添加带缓冲发送消息接口
	//设置链接属性
	SetProperty(key string, value interface{})
	//获取链接属性
	GetProperty(key string)(interface{}, error)
	//移除链接属性
	RemoveProperty(key string)
}

type HandFunc func(*net.TCPConn , []byte , int)error