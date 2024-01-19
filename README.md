# v0.1 
V0.1版本实现了⼀个基础的Server框架，服务器初始化运行，然后开启协程监听端口，循环监听，当客户端连接到达就对每一个连接建立一个新协程进行异步处理

v0.2版本对客户端连接和不同的客户端链接所处理理的不同业务再做⼀层接口封装。
    采用面向对象的方式，把每个连接也作为对象。连接也有start,stop,连接地址，连接唯一id,连接信息等属性以及路由等。

v0.3版本把客户端请求的连接信息 和 请求的数据，放在一个叫Request的请求类里，这样的好处是我们可以从Request里得到全部客户端的请求信息，也为我们之后拓展框架有一定的作用，

一旦客户端有额外的含义的数据信息，都可以放在这个Request里。可以理解为每次客户端的全部请求数据，Zinx都会把它们一起放到一个Request结构体里。
    实现路由配置功能，基于设计模式：服务器添加具体路由，与客户端建立连接后，把具体路由绑定到连接上。连接的读协程读取数据后把数据和连接保存到Request请求类里。
    然后开启异步协程，把Request作为参数传进去，再里面调用连接的路由  （策略模式： 具体使用哪个策略实现是可以在运行时动态决定的）
v0.4 加入配置文件
v0.5 消息封装  对客户端和服务端数据进行封包，解包，具体是新增message类型，包括数据的头长度，数据的id，数据的真实数据，客户端按封包形式发送给服务端，服务端建立连接从里面取出被封包的数据，然后进行解包，存入request,然后同0.3
v0.6 多路由模式 引入消息管理（map存储的键为消息的id，值为对应的路由）服务器运行时添加多个路由保存起来，然后连接到达时把路由map作为连接初始化的参数保存到每个连接，当解包客户端消息id，根据id执行不同的路由
v0.7 连接的读写分离。连接新增一个通道，并且连接运行时，同时开始读写协程。数据到达时读协程把数据通过v0.6的消息管理执行路由，路由的方法改写通过通道给写协程，写协程阻塞等待通道，有数据就写会客户端
v0.8 消息队列和多任务机制   原因：主要是连接建立后，读协程读到消息就go消息管理异步发送消息到路由，然后通过连接的通道发送到写协程。
                        解决方法：根据配置文件在服务器启动时建立多个通道切片，用来模拟任务池。读协程把数据封装成request,然后根据配置文件任务池数量把request根据request里面的连接唯一id对任务池数量取模发送到对应的切片通道
                                然后切片通道把数据再发给写协程
v0.9 连接管理  添加连接管理（添加，删除，连接最大数量限制）
v1.0 连接属性设置 添加钩子函数，服务器自定义钩子函数存入服务器，把连接与服务器进行绑定，当建立连接后或断开连接就触发钩子函数



# 1 server   服务器初始化运行，然后开启协程监听端口，循环监听，当客户端连接到达就对每一个连接建立一个新协程进行异步处理
    主要属性
        MsgHandler
            Apis  存放每个MsgId所对应的处理方法的map  继承baserouter的路由自定义对象，有Handle(),PreHandle()等方法
            WorkerPoolSize // 工作池大小（len(TaskQueue)）
            TaskQueue  // chan组成的切片消息队列，类型是IRequest
        ConnMgr(对连接的增删改查) 
        OnConnStart，OnConnStop 自定义的连接操作所触发的钩子函数
    主要方法：
        AddRouter(msgId uint32, router ziface.IRouter) 根据消息类型的不同绑定路由到MsgHandler
        Serve   // 调用start，然后阻塞
        Start   // go一个监听tcp连接的协程。for循环开始阻塞等待连接，连接到达，（判断是否超出最大连接数），初始化连接（server,conn,connID,server.MsgHandler）生成唯一连接id，然后id++（用于下一次连接），为当前连接go协程，for循环进行下一次等待连接
        Stop    // 服务器关闭，清空ConnMgr
        SetOnConnStart   // 添加钩子函数
        CallOnConnStart  //调用钩子函数
# 2 Conn
    主要属性
        TcpServer
		Conn
		ConnID
		isClosed
		MsgHandler
		ExitBuffChan: make(chan bool, 1),
		msgChan:make(chan []byte), //msgChan初始化
		msgBuffChan: make(chan []byte,utils.GlobalObject.MaxMsgChanLen),
		property make(map[string]interface{}), //对链接属性map初始化
    主要方法


    ## 注意
    mmo_game是游戏服务器项目代码，引用了zinx服务器下的代码，启动的客户端在mmo_game >  client > client.exe , client_Data(配置文件)  // 启动client.exe 输入服务器的ip和端口  W A S D 鼠标右键旋转
