package metrics

type FakeMetrics struct {
}

func newFakeMetrics() *FakeMetrics {
	return &FakeMetrics{}
}

func (f FakeMetrics) ScanTxTotal(_ string, _ ...float64) {
}

func (f FakeMetrics) ScanCallbackTotal(_ string, _ ...float64) {
}

func (f FakeMetrics) ScanCallbackErrorTotal(_ string, _ ...float64) {
}
