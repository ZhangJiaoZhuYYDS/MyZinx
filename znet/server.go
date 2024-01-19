// @Author zhangjiaozhu 2024/1/17 9:44:00
package znet

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"
	"zinx/utils"
	"zinx/ziface"
)

type Server struct {
	Name string //服务器的名称
	IPVersion string //tcp4 or other
	IP string //服务绑定的IP地址
	Port int //服务绑定的端口
	//Router ziface.IRouter //当前Server由用户绑定的回调router,也就是Server注册的链接对应的处理业务
	MsgHandler ziface.IMsgHandle
	//当前Server的链接管理器
	ConnMgr ziface.IConnManager
	// =======================
	//新增两个hook函数原型

	//该Server的连接创建时Hook函数
	OnConnStart	func(conn ziface.IConnection)
	//该Server的连接断开时的Hook函数
	OnConnStop func(conn ziface.IConnection)

	// =======================
}

func (s *Server) AddRouter(msgId uint32, router ziface.IRouter) {
	s.MsgHandler.AddRouter(msgId, router)
	fmt.Println("Add Router succ! ")
}

//============== 定义当前客户端链接的handle api ===========
func CallBackToClient(conn *net.TCPConn, data []byte, cnt int) error {
	//回显业务
	fmt.Println("[Conn Handle] CallBackToClient ... ")
	if _, err := conn.Write(data[:cnt]); err !=nil {
		fmt.Println("write back buf err ", err)
		return errors.New("CallBackToClient error")
	}
	return nil
}
/*
  创建一个服务器句柄
*/
func NewServer (name string) ziface.IServer {
	//先初始化全局配置文件
	utils.GlobalObject.Reload()
	fmt.Printf("[Zinx] Version: %s, MaxConn: %d,  MaxPacketSize: %d\n",
		utils.GlobalObject.Version,
		utils.GlobalObject.MaxConn,
		utils.GlobalObject.MaxPacketSize)
	s:= &Server {
		Name :utils.GlobalObject.Name,
		IPVersion:"tcp4",
		IP:utils.GlobalObject.Host,
		Port:utils.GlobalObject.TcpPort,
		//Router: nil,
		MsgHandler: NewMsgHandle(),
		ConnMgr: NewConnManager(),
	}

	return s
}
func (s *Server) Start()  {
	log.Println("开启服务器监听，listening...............")
	// 开启一个协程
	go func() {
		//0 启动worker工作池机制
		s.MsgHandler.StartWorkerPool()

		addr, err := net.ResolveTCPAddr(s.IPVersion, fmt.Sprintf("%s:%d",s.IP,s.Port))
		if err != nil {
			log.Fatalln("服务器解析地址错误，开启服务器失败",err)
		}
		// 监听服务器地址
		listenner, err := net.ListenTCP(s.IPVersion, addr)
		if err != nil {
			log.Fatalln("服务器监听失败",err)
		}
		log.Println("服务器监听成功",s.IP,s.Port)

		//TODO server.go 应该有一个自动生成ID的方法
		var cid uint32
		cid = 0


		// 启动服务器处理网络连接服务（三次握手，四次挥手）
		for{
			conn, err := listenner.AcceptTCP()
			if err != nil {
				log.Println("和客户端建立网络连接失败",err)
				continue   // 使用continue , 继续处理网络连接，而不是return直接退出
			}
			//3.2 设置服务器最大连接控制,如果超过最大连接，那么则关闭此新的连接
			if s.ConnMgr.Len() >= utils.GlobalObject.MaxConn {
				conn.Close()
				continue
			}

			//3.3 处理该新连接请求的 业务 方法， 此时应该有 handler 和 conn是绑定的
			//dealConn := NewConntion(conn, cid, CallBackToClient)

			dealConn := NewConntion(s,conn, cid, s.MsgHandler)
			cid ++

			//3.4 启动当前链接的处理业务  为每个客户端连接创建一个协程，传入参数：连接，连接唯一id,结束连接通道
			go dealConn.Start()
			// 为每次客户端连接新建一个协程
			//go func() {
			//	//不断的循环从客户端获取数据
			//	for {
			//		buf := make([]byte, 512)
			//		log.Println("服务端开始读数据")
			//		cnt, err := conn.Read(buf)
			//		log.Println("服务端读取到了", cnt, "字节")
			//		if err != nil {
			//			fmt.Println("recv buf err ", err)
			//			continue
			//		}
			//		log.Println(123456)
			//		// TODO 使用了下面的第一行代码，服务端就发送不了数据
			//		//log.Println("服务端获取到了客户端数据",string(buf),"开始回写数据")
			//		//fmt.Printf("服务端获取到了客户端数据 %s 开始回写数据\n", string(buf))
			//		//log.Println(string(buf))
			//		log.Println(buf)
			//		log.Println("服务端获取到了客户端数据", string(buf), "开始回写数据")
			//
			//		log.Println(buf)
			//		//回显
			//		// 将日志标志设置为默认的格式（日期和时间）
			//		//log.SetFlags(log.LstdFlags)
			//		// 将日志输出设置为标准输出
			//		//log.SetOutput(os.Stdout)
			//		//log.Printf("服务端获取到了客户端数据 %s 开始回写数据\n", string(buf))
			//
			//		if _, err := conn.Write(buf[:cnt]); err != nil {
			//			fmt.Println("write back buf err ", err)
			//			continue
			//		}
			//	}
			//}()
		}

	}()
}

func (s *Server) Stop()  {
	log.Println("停止服务器")
	//将其他需要清理的连接信息或者其他信息 也要一并停止或者清理
	s.ConnMgr.ClearConn()
}
func (s *Server) Serve()  {
	s.Start()
	for  {
		time.Sleep(10*time.Second)
	}
}
//func (s *Server) AddRouter(router ziface.IRouter) {
//	s.Router = router
//	fmt.Println("服务器添加路由成功")
//}

//得到链接管理
func (s *Server) GetConnMgr() ziface.IConnManager {
	return s.ConnMgr
}
//设置该Server的连接创建时Hook函数
func (s *Server) SetOnConnStart(hookFunc func (ziface.IConnection)) {
	s.OnConnStart = hookFunc
}

//设置该Server的连接断开时的Hook函数
func (s *Server) SetOnConnStop(hookFunc func (ziface.IConnection)) {
	s.OnConnStop = hookFunc
}

//调用连接OnConnStart Hook函数
func (s *Server) CallOnConnStart(conn ziface.IConnection) {
	if s.OnConnStart != nil {
		fmt.Println("---> CallOnConnStart....")
		s.OnConnStart(conn)
	}
}

//调用连接OnConnStop Hook函数
func (s *Server) CallOnConnStop(conn ziface.IConnection) {
	if s.OnConnStop != nil {
		fmt.Println("---> CallOnConnStop....")
		s.OnConnStop(conn)
	}
}