package selector

import (
	"github.com/meow-pad/chinchilla/transfer/common"
	"math/rand"
	"sort"
)

func NewWeightSelector() Selector {
	return &WeightSelector{}
}

type WeightSelector struct {
	infoArr []common.Info
	weights []int
}

func (selector *WeightSelector) Select(routerId uint64) (string, error) {
	instLen := len(selector.infoArr)
	if instLen <= 0 {
		return "", common.ErrEmptyInstances
	}
	if instLen == 1 {
		return selector.infoArr[0].ServiceId(), nil
	}
	r := rand.Intn(selector.weights[instLen-1]) + 1
	i := sort.SearchInts(selector.weights, r)
	return selector.infoArr[i].ServiceId(), nil
}

func (selector *WeightSelector) Update(infoArr []common.Info) {
	weights := make([]int, len(infoArr))
	sum := 0
	for i, inst := range infoArr {
		sum += int(inst.Weight)
		weights[i] = sum
	}
	copy(selector.infoArr, infoArr)
	selector.weights = weights
}
