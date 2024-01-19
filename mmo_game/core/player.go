// @Author zhangjiaozhu 2024/1/18 19:44:00
// 玩家对象
package core

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"math/rand"
	"sync"
	"zinx/mmo_game/pb"
	"zinx/ziface"
)

type Player struct {
	Pid  int32              //玩家ID
	Conn ziface.IConnection //当前玩家的连接
	X    float32            //平面x坐标
	Y    float32            //高度
	Z    float32            //平面y坐标 (注意不是Y)
	V    float32            //旋转0-360度
}

/*
Player ID 生成器
*/
var PidGen int32 = 1  //用来生成玩家ID的计数器
var IdLock sync.Mutex //保护PidGen的互斥机制

// 创建一个玩家对象
func NewPlayer(conn ziface.IConnection) *Player {
	// 生成一个PID
	IdLock.Lock()
	id := PidGen
	PidGen++
	IdLock.Unlock()

	p := &Player{
		Pid:  id,
		Conn: conn,
		X:    float32(160 + rand.Intn(10)), //随机在160坐标点 基于X轴偏移若干坐标
		Y:    0,                            //高度为0
		Z:    float32(134 + rand.Intn(17)), //随机在134坐标点 基于Y轴偏移若干坐标
		V:    0,                            //角度为0，尚未实现
	}

	return p
}
/*
	发送消息给客户端，
	主要是将pb的protobuf数据序列化之后发送
*/
func (p *Player) SendMsg(msgId uint32, data proto.Message) {
	fmt.Printf("before Marshal data = %+v\n", data)
	//将proto Message结构体序列化
	msg, err := proto.Marshal(data)
	if err != nil {
		fmt.Println("marshal msg err: ", err)
		return
	}
	fmt.Printf("after Marshal data = %+v\n", msg)

	if p.Conn == nil {
		fmt.Println("connection in player is nil")
		return
	}

	//调用Zinx框架的SendMsg发包
	if err := p.Conn.SendMsg(msgId, msg); err != nil {
		fmt.Println("Player SendMsg error !")
		return
	}

	return
}
//告知客户端pid,同步已经生成的玩家ID给客户端
func (p *Player) SyncPid() {
	//组建MsgId0 proto数据
	data := &pb.SyncPid{
		Pid:p.Pid,
	}

	//发送数据给客户端
	p.SendMsg(1, data)
}
//广播玩家自己的出生地点
func (p *Player) BroadCastStartPosition() {

	msg := &pb.BroadCast{
		Pid:p.Pid,
		Tp:2,//TP2 代表广播坐标
		Data: &pb.BroadCast_P{
			&pb.Position{
				X:p.X,
				Y:p.Y,
				Z:p.Z,
				V:p.V,
			},
		},
	}

	p.SendMsg(200, msg)
}
//广播玩家聊天
func (p *Player) Talk(content string) {
	//1. 组建MsgId200 proto数据
	msg := &pb.BroadCast{
		Pid:p.Pid,
		Tp:1,//TP 1 代表聊天广播
		Data: &pb.BroadCast_Content{
			Content: content,
		},
	}

	//2. 得到当前世界所有的在线玩家
	players := WorldMgrObj.GetAllPlayers()

	//3. 向所有的玩家发送MsgId:200消息
	for _, player := range players {
		player.SendMsg(200, msg)
	}
}

//给当前玩家周边的(九宫格内)玩家广播自己的位置，让他们显示自己
func (p *Player) SyncSurrounding() {
	//1 根据自己的位置，获取周围九宫格内的玩家pid
	pids := WorldMgrObj.AoiMgr.GetPIDsByPos(p.X, p.Z)
	//2 根据pid得到所有玩家对象
	players := make([]*Player, 0, len(pids))
	//3 给这些玩家发送MsgID:200消息，让自己出现在对方视野中
	for _, pid := range pids {
		players = append(players, WorldMgrObj.GetPlayerByPid(int32(pid)))
	}
	//3.1 组建MsgId200 proto数据
	msg := &pb.BroadCast{
		Pid:p.Pid,
		Tp:2,//TP2 代表广播坐标
		Data: &pb.BroadCast_P{
			P:&pb.Position{
				X:p.X,
				Y:p.Y,
				Z:p.Z,
				V:p.V,
			},
		},
	}
	//3.2 每个玩家分别给对应的客户端发送200消息，显示人物
	for _, player := range players {
		player.SendMsg(200, msg)
	}
	//4 让周围九宫格内的玩家出现在自己的视野中
	//4.1 制作Message SyncPlayers 数据
	playersData := make([]*pb.Player, 0, len(players))
	for _, player := range players {
		p := &pb.Player{
			Pid:player.Pid,
			P:&pb.Position{
				X:player.X,
				Y:player.Y,
				Z:player.Z,
				V:player.V,
			},
		}
		playersData = append(playersData, p)
	}

	//4.2 封装SyncPlayer protobuf数据
	SyncPlayersMsg := &pb.SyncPlayers{
		Ps:playersData[:],
	}

	//4.3 给当前玩家发送需要显示周围的全部玩家数据
	p.SendMsg(202, SyncPlayersMsg)
}
//玩家下线
func (p *Player) LostConnection() {
	//1 获取周围AOI九宫格内的玩家
	players := p.GetSurroundingPlayers()

	//2 封装MsgID:201消息
	msg := &pb.SyncPid{
		Pid:p.Pid,
	}

	//3 向周围玩家发送消息
	for _, player := range players {
		player.SendMsg(201, msg)
	}

	//4 世界管理器将当前玩家从AOI中摘除
	WorldMgrObj.AoiMgr.RemoveFromGridByPos(int(p.Pid), p.X, p.Z)
	WorldMgrObj.RemovePlayerByPid(p.Pid)
}

//广播玩家位置移动
//func (p *Player) UpdatePos(x float32, y float32, z float32, v float32) {
//	//更新玩家的位置信息
//	p.X = x
//	p.Y = y
//	p.Z = z
//	p.V = v
//	//组装protobuf协议，发送位置给周围玩家
//	msg := &pb.BroadCast{
//		Pid:p.Pid,
//		Tp:4,     //4 - 移动之后的坐标信息
//		Data: &pb.BroadCast_P{
//			P:&pb.Position{
//				X:p.X,
//				Y:p.Y,
//				Z:p.Z,
//				V:p.V,
//			},
//		},
//	}
//	//获取当前玩家周边全部玩家
//	players := p.GetSurroundingPlayers()
//	//向周边的每个玩家发送MsgID:200消息，移动位置更新消息
//	for _, player := range players {
//		player.SendMsg(200, msg)
//	}
//}
//TODO
// 自行实现跨越格子的AOI处理
func (player *Player) UpdatePos(x, y, z, v float32) {
	// 处理跨越格子
	// 1. 获取玩家当前位置所处格子ID
	// 2. 获取玩家新位置所处格子ID
	// 3. 判断是否跨越格子 格子ID是否相同


	// 当前玩家所处格子ID
	curGridId := WorldMgrObj.AoiMgr.GetGIDByPos(player.X, player.Z)
	fmt.Println("curGridId = ", curGridId)
	// 玩家新位置所处格子ID
	newGridId := WorldMgrObj.AoiMgr.GetGIDByPos(x, z)
	fmt.Println("newGridId = ", newGridId)


	if curGridId != newGridId {
		// 说明跨越格子了，要处理新格子的视野，包括移除、新增其他玩家
		player.refreshAOI(x, y, z, v)
	} else {
		// 没有跨越格子则无需刷新视野
		// 更新玩家位置信息
		player.X = x
		player.Y = y
		player.Z = z
		player.V = v


		// 组建proto数据
		msg := &pb.BroadCast{
			Pid: player.Pid,
			Tp:  4,
			Data: &pb.BroadCast_P{
				P: &pb.Position{
					X: player.X,
					Y: player.Y,
					Z: player.Z,
					V: player.V,
				},
			},
		}


		// 获取玩家周围的其他玩家
		players := player.GetSurroundingPlayers()


		// 向周围的其他玩家广播移动消息
		for _, player := range players {
			player.SendMsg(200, msg)
		}
	}


}


// 刷新AOI视野
func (player *Player) refreshAOI(x, y, z, v float32) {
	// 1. 我离开旧的九宫格其他玩家的视野
	// 2. 旧的九宫格其他玩家消失在我的视野中
	// 3. 我出现在新的九宫格中的玩家视野中
	// 4. 新的九宫格的玩家出现在我的视野中


	// 获取旧九宫格所有玩家
	oldPlayerIdList := WorldMgrObj.AoiMgr.GetPIDsByPos(player.X, player.Z)


	// 获取新九宫格所有玩家
	newPlayerIdList := WorldMgrObj.AoiMgr.GetPIDsByPos(x, z)


	// 求两个玩家列表的格子的差集
	// 将old和new转换成Map
	oldMap := make(map[int]bool)
	for _, v := range oldPlayerIdList {
		oldMap[v] = true
	}


	newMap := make(map[int]bool)
	for _, v := range newPlayerIdList {
		newMap[v] = true
	}


	// 得到old数组中不在new数组中的元素
	var oldNotInNew []int
	for _, v := range oldPlayerIdList {
		if _, ok := newMap[v]; !ok {
			oldNotInNew = append(oldNotInNew, v)
		}
	}


	// 得到new数组中不在old数组中的元素
	var newNotInOld []int
	for _, v := range newPlayerIdList {
		if _, ok := oldMap[v]; !ok {
			newNotInOld = append(newNotInOld, v)
		}
	}


	fmt.Println("will remove playerId list: ", oldNotInNew)
	fmt.Println("will add playerId list: ", newNotInOld)


	// 获取需要移除视野的玩家实例
	removePlayers := make([]*Player, 0, len(oldNotInNew))


	for _, pid := range oldNotInNew {
		removePlayers = append(removePlayers, WorldMgrObj.GetPlayerByPid(int32(pid)))
	}
	// 1. 我离开旧的九宫格其他玩家的视野(广播玩家离开)
	msg := &pb.SyncPid{
		Pid: player.Pid,
	}
	for _, player := range removePlayers {
		player.SendMsg(201, msg)
	}


	// 2. 旧的九宫格其他玩家消失在我的视野中(移除旧视野中的其他玩家)
	for _, pid := range oldNotInNew {
		msg := &pb.SyncPid{
			Pid: int32(pid),
		}
		player.SendMsg(201, msg)
	}


	// 获取需要加入视野的玩家实例
	// addPlayers := make([]*Player, 0, len(newNotInOld))


	// for _, pid := range newNotInOld {
	//  addPlayers = append(addPlayers, WorldMgrObj.GetPlayerByPid(int32(pid)))
	// }


	// 3. 我出现在新的九宫格中的玩家视野中(广播玩家加入视野)
	// 4. 新的九宫格的玩家出现在我的视野中
	// 直接更新玩家最新位置信息，然后调用同步周围接口即可
	player.X = x
	player.Y = y
	player.Z = z
	player.V = v
	player.SyncSurrounding()


}
//获得当前玩家的AOI周边玩家信息
func (p *Player) GetSurroundingPlayers() []*Player {
	//得到当前AOI区域的所有pid
	pids := WorldMgrObj.AoiMgr.GetPIDsByPos(p.X, p.Z)

	//将所有pid对应的Player放到Player切片中
	players := make([]*Player, 0, len(pids))
	for _, pid := range pids {
		players = append(players, WorldMgrObj.GetPlayerByPid(int32(pid)))
	}

	return players
}