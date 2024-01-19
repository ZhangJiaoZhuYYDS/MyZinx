// @Author zhangjiaozhu 2024/1/18 22:49:00
package api

import (
	"zinx/ziface"
	"zinx/znet"
)

type Test struct {
	znet.BaseRouter
}
func (*Test) Handle(request ziface.IRequest)  {

}
