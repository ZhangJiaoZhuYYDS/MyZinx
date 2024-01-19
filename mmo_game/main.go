// @Author zhangjiaozhu 2024/1/18 13:42:00
package main

import (
	"fmt"
	"zinx/mmo_game/api"
	"zinx/mmo_game/core"
	"zinx/ziface"
	"zinx/znet"
)

//当客户端建立连接的时候的hook函数
func OnConnecionAdd(conn ziface.IConnection)  {
	//创建一个玩家
	player := core.NewPlayer(conn)
	//同步当前的PlayerID给客户端， 走MsgID:1 消息
	player.SyncPid()
	//同步当前玩家的初始化坐标信息给客户端，走MsgID:200消息
	player.BroadCastStartPosition()

	//========将当前新上线玩家添加到worldManager中
	core.WorldMgrObj.AddPlayer(player)
	//========================================
	//将该连接绑定属性Pid
	conn.SetProperty("pid", player.Pid)

	//==============同步周边玩家上线信息，与现实周边玩家信息========
	player.SyncSurrounding()
	//=======================================================

	fmt.Println("=====> Player pidId = ", player.Pid, " arrived ====")
}
//当客户端断开连接的时候的hook函数
func OnConnectionLost(conn ziface.IConnection) {
	//获取当前连接的Pid属性
	pid, _ := conn.GetProperty("pid")

	//根据pid获取对应的玩家对象
	player := core.WorldMgrObj.GetPlayerByPid(pid.(int32))

	//触发玩家下线业务
	if pid != nil {
		player.LostConnection()
	}

	fmt.Println("====> Player ", pid , " left =====")

}

func main()  {
	s := znet.NewServer("MMO GAME")
	//注册客户端连接建立和丢失函数
	s.SetOnConnStart(OnConnecionAdd)
	//注册客户端断开连接触发下线函数
	s.SetOnConnStop(OnConnectionLost)
	//注册路由
	s.AddRouter(2, &api.WorldChatApi{})    //聊天
	s.AddRouter(3, &api.MoveApi{})		   //移动
	//启动服务
	s.Serve()
}