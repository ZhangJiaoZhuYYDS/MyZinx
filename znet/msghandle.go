// @Author zhangjiaozhu 2024/1/17 20:33:00
package znet

import (
	"fmt"
	"log"
	"strconv"
	"zinx/utils"
	"zinx/ziface"
)

type MsgHandle struct{
	Apis map[uint32] ziface.IRouter //存放每个MsgId 所对应的处理方法的map属性
	WorkerPoolSize uint32           //业务工作Worker池的数量
	TaskQueue []chan ziface.IRequest  //Worker负责取任务的消息队列
}

func NewMsgHandle() *MsgHandle {
	return &MsgHandle {
		Apis:make(map[uint32]ziface.IRouter),
		WorkerPoolSize: utils.GlobalObject.WorkerPoolSize,
		TaskQueue: make([]chan ziface.IRequest,utils.GlobalObject.WorkerPoolSize),
	}
}
//马上以非阻塞方式处理消息
func (mh *MsgHandle) DoMsgHandler(request ziface.IRequest)	{
	handler , ok  := mh.Apis[request.GetMsgID()]
	if !ok {
		log.Println("api msgId = ", request.GetMsgID(), " is not FOUND!")
		return
	}
	//执行对应处理方法
	handler.PreHandle(request)
	handler.Handle(request)
	handler.PostHandle(request)
}
//为消息添加具体的处理逻辑
func (mh *MsgHandle) AddRouter(msgId uint32, router ziface.IRouter) {
	//1 判断当前msg绑定的API处理方法是否已经存在
	if _, ok := mh.Apis[msgId]; ok {
		panic("repeated api , msgId = " + strconv.Itoa(int(msgId)))
	}
	//2 添加msg与api的绑定关系
	mh.Apis[msgId] = router
	fmt.Println("Add api msgId = ", msgId)
}
//启动一个Worker工作流程 一个Worker的工作业务，每个worker是不会退出的(目前没有设定worker的停止工作机制)，会永久的从对应的TaskQueue中等待消息，并处理。
func (mh *MsgHandle) StartOneWorker(workerID int, taskQueue chan ziface.IRequest) {
	log.Println("工作池  ",workerID,"启动")
	//不断的等待队列中的消息
	for{
		select {
		case request:= <- taskQueue:
			//有消息则取出队列的Request，并执行绑定的业务方法
			mh.DoMsgHandler(request)
		}
	}
}
//启动worker工作池 启动Worker工作池，这里根据用户配置好的WorkerPoolSize的数量来启动，然后分别给每个Worker分配一个TaskQueue，然后用一个goroutine来承载一个Worker的工作业务。
func (mh *MsgHandle) StartWorkerPool() {
	for i := 0; i < int(mh.WorkerPoolSize);i++ {
		mh.TaskQueue[i] = make(chan ziface.IRequest,utils.GlobalObject.MaxWorkerTaskLen)
		go mh.StartOneWorker(i,mh.TaskQueue[i])
	}
}

//将消息交给TaskQueue,由worker进行处理
func (mh *MsgHandle)SendMsgToTaskQueue(request ziface.IRequest) {
	//根据ConnID来分配当前的连接应该由哪个worker负责处理
	//轮询的平均分配法则

	//得到需要处理此条连接的workerID
	workerID := request.GetConnection().GetConnID() % mh.WorkerPoolSize
	fmt.Println("Add ConnID=", request.GetConnection().GetConnID()," request msgID=", request.GetMsgID(), "to workerID=", workerID)
	//将请求消息发送给任务队列
	mh.TaskQueue[workerID] <- request
}