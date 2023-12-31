package selector

import (
	"github.com/meow-pad/chinchilla/transfer/common"
	"github.com/meow-pad/persian/frame/plog"
	"github.com/meow-pad/persian/frame/plog/pfield"
)

func NewCompositeSelector(selectors ...Selector) Selector {
	return &CompositeSelector{
		selectors: selectors,
	}
}

type CompositeSelector struct {
	selectors []Selector

	infoArr []common.Info
}

func (selector *CompositeSelector) Select(routerId uint64) (string, error) {
	if len(selector.infoArr) <= 0 {
		return "", ErrEmptyInstances
	}
	for _, _selector := range selector.selectors {
		id, err := _selector.Select(routerId)
		if err != nil {
			plog.Error("select service error:", pfield.Error(err))
			// 这里的处理是先忽略当前问题，继续往下查找（按需求再修改）
			continue
		}
		if len(id) > 0 {
			return id, nil
		}
	}
	return "", nil
}

func (selector *CompositeSelector) Update(infoArr []common.Info) {
	selector.infoArr = make([]common.Info, len(infoArr))
	copy(selector.infoArr, infoArr)
	for _, _selector := range selector.selectors {
		_selector.Update(infoArr)
	}
}
