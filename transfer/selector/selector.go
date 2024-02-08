package selector

import (
	"github.com/meow-pad/chinchilla/transfer/common"
)

type Selector interface {
	// Select
	//  @Description: 选择一个可用服务
	//  @param service
	//  @param routerId
	//  @return string 有数据但选不出来时，该值为空字符串
	//  @return error 如果实例数组本就为空，则返回 common.ErrEmptyInstances
	//
	Select(routerId string) (string, error)

	// Update
	//  @Description: 更新可用服务列表
	//  @param infoArr
	//
	Update(instances []common.Info)
}
