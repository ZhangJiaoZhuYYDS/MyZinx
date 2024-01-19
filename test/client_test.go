// @Author zhangjiaozhu 2024/1/17 9:25:00
package test

import (
	"fmt"
	"io"
	"log"
	"net"
	"testing"
	"time"
	"zinx/ziface"
	"zinx/znet"
)

// 客户端
func ClientClient(){
	log.Println("客户端测试开始。。。。")
	//3秒之后发起测试请求，给服务端开启服务的机会
	time.Sleep(3*time.Second)
	conn , err := net.Dial("tcp","127.0.0.1:7777")
	if err!= nil {
		log.Fatalln("客户端启动失败")
	}
	for {
		//发封包message消息
		dp := znet.NewDataPack()
		msg, _ := dp.Pack(znet.NewMsgPackage(0,[]byte("Zinx V0.5 Client Test Message")))
		_, err := conn.Write(msg)
		if err !=nil {
			fmt.Println("write error err ", err)
			return
		}

		//先读出流中的head部分
		headData := make([]byte, dp.GetHeadLen())
		_, err = io.ReadFull(conn, headData) //ReadFull 会把msg填充满为止
		if err != nil {
			fmt.Println("read head error")
			break
		}
		//将headData字节流 拆包到msg中
		msgHead, err := dp.Unpack(headData)
		if err != nil {
			fmt.Println("server unpack err:", err)
			return
		}
		if msgHead.GetDataLen() > 0 {
			//msg 是有data数据的，需要再次读取data数据
			msg := msgHead.(*znet.Message)
			msg.Data = make([]byte, msg.GetDataLen())

			//根据dataLen从io中读取字节流
			_, err := io.ReadFull(conn, msg.Data)
			if err != nil {
				fmt.Println("server unpack data err:", err)
				return
			}

			fmt.Println("==> Recv Msg: ID=", msg.Id, ", len=", msg.DataLen, ", data=", string(msg.Data))
		}

		time.Sleep(1*time.Second)
	}
}

//ping test 自定义路由
type PingRouter struct {
	znet.BaseRouter
}
//Test PreHandle
func (this *PingRouter) PreHandle(request ziface.IRequest) {
	fmt.Println("前置路由")
	//_, err := request.GetConnection().GetTCPConnection().Write([]byte("Before ping ....\n"))
	//if err !=nil {
	//	fmt.Println("call back ping ping ping error")
	//}
}
//Test Handle
func (this *PingRouter) Handle(request ziface.IRequest) {
	fmt.Println("Call PingRouter Handle")
	//先读取客户端的数据，再回写ping...ping...ping
	fmt.Println("recv from client : msgId=", request.GetMsgID(), ", data=", string(request.GetData()))

	err := request.GetConnection().SendBuffMsg(0, []byte("ping...ping...ping"))
	if err != nil {
		fmt.Println(err)
	}
}

type HelloZinxRouter struct {
	znet.BaseRouter
}

//HelloZinxRouter Handle
func (this *HelloZinxRouter) Handle(request ziface.IRequest) {
	fmt.Println("Call HelloZinxRouter Handle")
	//先读取客户端的数据，再回写ping...ping...ping   获取连接属性
	fmt.Println("recv from client : msgId=", request.GetMsgID(), ", data=", string(request.GetData()))

	err := request.GetConnection().SendBuffMsg(1, []byte("Hello Zinx Router V0.10"))
	if err != nil {
		fmt.Println(err)
	}
}


//Test PostHandle
func (this *PingRouter) PostHandle(request ziface.IRequest) {
	fmt.Println("后置路由")
	//_, err := request.GetConnection().GetTCPConnection().Write([]byte("After ping .....\n"))
	//if err != nil {
	//	fmt.Println("call back ping ping ping error")
	//}
}

//创建连接的时候执行
func DoConnectionBegin(conn ziface.IConnection) {
	fmt.Println("DoConnecionBegin is Called ... ")

	//=============设置两个链接属性，在连接创建之后===========
	fmt.Println("Set conn Name, Home done!")
	conn.SetProperty("Name", "Aceld")
	conn.SetProperty("Home", "https://www.jianshu.com/u/35261429b7f1")
	//===================================================

	err := conn.SendMsg(2, []byte("DoConnection BEGIN..."))
	if err != nil {
		fmt.Println(err)
	}
}

//连接断开的时候执行
func DoConnectionLost(conn ziface.IConnection) {
	//============在连接销毁之前，查询conn的Name，Home属性=====
	if name, err:= conn.GetProperty("Name"); err == nil {
		fmt.Println("Conn Property Name = ", name)
	}

	if home, err := conn.GetProperty("Home"); err == nil {
		fmt.Println("Conn Property Home = ", home)
	}
	//===================================================

	fmt.Println("DoConneciotnLost is Called ... ")
}

func TestClient(t *testing.T)  {
	s := znet.NewServer("zinx 0.1")
	s.AddRouter(0,&PingRouter{})
	go ClientClient()
	s.Serve()
}
