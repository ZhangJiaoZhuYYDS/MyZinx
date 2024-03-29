// @Author zhangjiaozhu 2024/1/17 10:40:00
package znet

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"zinx/utils"
	"zinx/ziface"
)

type Connection struct {
	//当前Conn属于哪个Server
	TcpServer	ziface.IServer       //当前conn属于哪个server，在conn初始化的时候添加即可
	Conn *net.TCPConn //当前连接的socket TCP套接字
	ConnID uint32 //当前连接的ID 也可以称作为SessionID，ID全局唯一
	isClosed bool //当前连接的关闭状态

	//handleAPI ziface.HandFunc //该连接的处理方法api

	ExitBuffChan chan bool //告知该链接已经退出/停止的channel


	//Router  ziface.IRouter  //该连接的处理方法router
	MsgHandler ziface.IMsgHandle

	//无缓冲管道，用于读、写两个goroutine之间的消息通信
	msgChan		chan []byte
	//有缓冲管道，用于读、写两个goroutine之间的消息通信
	msgBuffChan chan []byte

	// ================================
	//链接属性
	property     map[string]interface{}
	//保护链接属性修改的锁
	propertyLock sync.RWMutex
	// ================================
}



//创建连接的方法
//func NewConntion(conn *net.TCPConn, connID uint32, callback_api ziface.HandFunc) *Connection{
//	c := &Connection{
//		Conn:     conn,
//		ConnID:   connID,
//		isClosed: false,
//		handleAPI: callback_api,
//		ExitBuffChan: make(chan bool, 1),
//	}
//	return c
//}
func NewConntion(server ziface.IServer,conn *net.TCPConn, connID uint32, router ziface.IMsgHandle) *Connection{
	c := &Connection{
		TcpServer: server,
		Conn:     conn,
		ConnID:   connID,
		isClosed: false,
		MsgHandler: router,
		ExitBuffChan: make(chan bool, 1),
		msgChan:make(chan []byte), //msgChan初始化
		msgBuffChan: make(chan []byte,utils.GlobalObject.MaxMsgChanLen),
		property:     make(map[string]interface{}), //对链接属性map初始化
	}
	//将新创建的Conn添加到链接管理中
	c.TcpServer.GetConnMgr().Add(c) //将当前新创建的连接添加到ConnManager中
	return c
}
/* 处理conn读数据的Goroutine */
func (c *Connection) StartReader() {
	log.Println("读协程正在从连接中读取数据")
	defer log.Println(c.Conn.RemoteAddr().String(),"读取数据协程结束")
	defer c.Stop()
	for{
		// 创建拆包解包的对象
 		dp := NewDataPack()

		//读取客户端的Msg head
		headData := make([]byte, dp.GetHeadLen())
		if _,err := io.ReadFull(c.GetTCPConnection(),headData);err != nil {
			log.Println("读取客户端数据头信息失败",err)
			c.ExitBuffChan <- true
			continue
		}

		//拆包，得到msgid 和 datalen 放在msg中
		msg , err := dp.Unpack(headData)
		if err != nil {
			fmt.Println("unpack error ", err)
			c.ExitBuffChan <- true
			continue
		}
		//根据 dataLen 读取 data，放在msg.Data中
		var data []byte
		if msg.GetDataLen() > 0 {
			data = make([]byte, msg.GetDataLen())
			if _, err := io.ReadFull(c.GetTCPConnection(), data); err != nil {
				fmt.Println("read msg data error ", err)
				c.ExitBuffChan <- true
				continue
			}
		}
		msg.SetData(data)

		//buf := make([]byte, utils.GlobalObject.MaxPacketSize)
		//cnt, err := c.Conn.Read(buf)
		//if err != nil {
		//	log.Println("读协程读取数据失败",err)
		//	c.ExitBuffChan <- true
		//	continue
		//}
		//log.Println("服务端的连接的读协程读取到了",cnt)

		//得到当前客户端请求的Request数据
		req := Request{conn: c,msg: msg}
		//从路由Routers 中找到注册绑定Conn的对应Handle
		//go func(request ziface.IRequest) {
		//	c.Router.PreHandle(request)
		//	c.Router.Handle(request)
		//	c.Router.PostHandle(request)
		//}(&req)
		/*这里并没有强制使用多任务Worker机制，而是判断用户配置WorkerPoolSize的个数，
		如果大于0，那么我就启动多任务机制处理链接请求消息，
		如果=0或者<0那么，我们依然只是之前的开启一个临时的Goroutine处理客户端请求消息。*/
		if utils.GlobalObject.WorkerPoolSize > 0 {
			//已经启动工作池机制，将消息交给Worker处理
			c.MsgHandler.SendMsgToTaskQueue(&req)
		}else {
			//从绑定好的消息和对应的处理方法中执行对应的Handle方法
			go c.MsgHandler.DoMsgHandler(&req)
		}

		//go c.MsgHandler.DoMsgHandler(&req)

		//调用当前链接业务(这里执行的是当前conn的绑定的handle方法)
		//if err := c.handleAPI(c.Conn,buf,cnt);err != nil {
		//	log.Println(c.ConnID,"handle错误",err)
		//	c.ExitBuffChan <- true
		//	return
		//}
	}
}

/*
	写消息Goroutine， 用户将数据发送给客户端
*/
func (c *Connection) StartWriter() {

	fmt.Println("[Writer Goroutine is running]")
	defer fmt.Println(c.RemoteAddr().String(), "[conn Writer exit!]")

	for {
		select {
		case data := <-c.msgChan:
			//有数据要写给客户端
			if _, err := c.Conn.Write(data); err != nil {
				fmt.Println("Send Data error:, ", err, " Conn Writer exit")
				return
			}
			//针对有缓冲channel需要些的数据处理
		case data, ok:= <-c.msgBuffChan:
			if ok {
				//有数据要写给客户端
				if _, err := c.Conn.Write(data); err != nil {
					fmt.Println("Send Buff Data error:, ", err, " Conn Writer exit")
					return
				}
			} else {
				break
				fmt.Println("msgBuffChan is Closed")
			}
		case <- c.ExitBuffChan:
			//conn已经关闭
			return
		}
	}
}
//启动连接，让当前连接开始工作
func (c *Connection) Start() {

	//开启处理该链接读取到客户端数据之后的请求业务
	go c.StartReader()
	//2 开启用于写回客户端数据流程的Goroutine
	go c.StartWriter()
	//==================
	//按照用户传递进来的创建连接时需要处理的业务，执行钩子方法
	c.TcpServer.CallOnConnStart(c)
	//==================

	for {
		select {
		case <- c.ExitBuffChan:
			//得到退出消息，不再阻塞，结束Start方法
			return
		}
	}
}
//停止连接，结束当前连接状态M
func (c *Connection) Stop() {
	fmt.Println("Conn Stop()...ConnID = ", c.ConnID)
	//1. 如果当前链接已经关闭
	if c.isClosed == true {
		return
	}
	c.isClosed = true

	//TODO Connection Stop() 如果用户注册了该链接的关闭回调业务，那么在此刻应该显示调用
	//==================
	//如果用户注册了该链接的关闭回调业务，那么在此刻应该显示调用
	c.TcpServer.CallOnConnStop(c)
	//==================

	// 关闭socket链接
	c.Conn.Close()

	//通知从缓冲队列读数据的业务，该链接已经关闭
	c.ExitBuffChan <- true

	//将链接从连接管理器中删除
	c.TcpServer.GetConnMgr().Remove(c)  //删除conn从ConnManager中

	//关闭该链接全部管道
	close(c.ExitBuffChan)
	close(c.msgBuffChan)
}
//从当前连接获取原始的socket TCPConn
func (c *Connection) GetTCPConnection() *net.TCPConn {
	return c.Conn
}

//获取当前连接ID
func (c *Connection) GetConnID() uint32{
	return c.ConnID
}

//获取远程客户端地址信息
func (c *Connection) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}
//直接将Message数据发送数据给远程的TCP客户端
func (c *Connection) SendMsg(msgId uint32, data []byte) error {
	if c.isClosed == true {
		return errors.New("Connection closed when send msg")
	}
	//将data封包，并且发送
	dp := NewDataPack()
	msg, err := dp.Pack(NewMsgPackage(msgId, data))
	if err != nil {
		fmt.Println("Pack error msg id = ", msgId)
		return  errors.New("Pack error msg ")
	}

	//写回客户端
	//if _, err := c.Conn.Write(msg); err != nil {
	//	fmt.Println("Write msg id ", msgId, " error ")
	//	c.ExitBuffChan <- true
	//	return errors.New("conn Write error")
	//}
	//写回客户端
	c.msgChan <- msg   //将之前直接回写给conn.Write的方法 改为 发送给Channel 供Writer读取

	return nil
}
func (c *Connection) SendBuffMsg(msgId uint32, data []byte) error {
	if c.isClosed == true {
		return errors.New("Connection closed when send buff msg")
	}
	//将data封包，并且发送
	dp := NewDataPack()
	msg, err := dp.Pack(NewMsgPackage(msgId, data))
	if err != nil {
		fmt.Println("Pack error msg id = ", msgId)
		return  errors.New("Pack error msg ")
	}

	//写回客户端
	c.msgBuffChan <- msg

	return nil
}
//设置链接属性
func (c *Connection) SetProperty(key string, value interface{}) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()

	c.property[key] = value
}

//获取链接属性
func (c *Connection) GetProperty(key string) (interface{}, error) {
	c.propertyLock.RLock()
	defer c.propertyLock.RUnlock()

	if value, ok := c.property[key]; ok  {
		return value, nil
	} else {
		return nil, errors.New("no property found")
	}
}

//移除链接属性
func (c *Connection) RemoveProperty(key string) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()

	delete(c.property, key)
}