package selector

import (
	"github.com/meow-pad/chinchilla/transfer/service"
	"math/rand"
	"sort"
)

func NewWeightSelector() *WeightSelector {
	return &WeightSelector{}
}

type WeightSelector struct {
	infoArr []service.Info
	weights []int
}

func (selector *WeightSelector) Select(routerId string) (string, error) {
	instLen := len(selector.infoArr)
	if instLen <= 0 {
		return "", ErrEmptyInstances
	}
	if instLen == 1 {
		return selector.infoArr[0].InstanceId, nil
	}
	r := rand.Intn(selector.weights[instLen-1]) + 1
	i := sort.SearchInts(selector.weights, r)
	return selector.infoArr[i].InstanceId, nil
}

func (selector *WeightSelector) Update(infoArr []service.Info) {
	weights := make([]int, len(infoArr))
	sum := 0
	for i, inst := range infoArr {
		sum += int(inst.Weight)
		weights[i] = sum
	}
	copy(selector.infoArr, infoArr)
	selector.weights = weights
}
