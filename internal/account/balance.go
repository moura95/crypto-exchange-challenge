package account

type Balance struct {
	Available float64
	Locked    float64
}

func (b *Balance) Total() float64 {
	return b.Available + b.Locked
}
