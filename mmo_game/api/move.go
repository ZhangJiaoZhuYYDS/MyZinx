// @Author zhangjiaozhu 2024/1/18 21:39:00
package api

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"zinx/mmo_game/core"
	"zinx/mmo_game/pb"
	"zinx/ziface"
	"zinx/znet"
)

type MoveApi struct {
	znet.BaseRouter
}

func (*MoveApi) Handle(request ziface.IRequest) {
	msg := &pb.Position{}
	err := proto.Unmarshal(request.GetData(),msg)
	if err != nil {
		fmt.Println("Move: Position Unmarshal error ", err)
		return
	}
	pid, err := request.GetConnection().GetProperty("pid")
	if err != nil {
		fmt.Println("GetProperty pid error", err)
		request.GetConnection().Stop()
		return
	}
	fmt.Printf("user pid = %d , move(%f,%f,%f,%f)", pid, msg.X, msg.Y, msg.Z, msg.V)
	//3. 根据pid得到player对象
	player := core.WorldMgrObj.GetPlayerByPid(pid.(int32))

	//4. 让player对象发起移动位置信息广播
	player.UpdatePos(msg.X, msg.Y, msg.Z, msg.V)
}