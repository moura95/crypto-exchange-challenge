package engine

const (
	PriceTick  = 0.01
	AmountTick = 0.00000001
)

type Pair struct {
	Base  string // BTC
	Quote string // BRL
}

func (p Pair) String() string {
	return p.Base + "/" + p.Quote
}

func (p Pair) IsValid() bool {
	return p.Base != "" && p.Quote != "" && p.Quote == "BRL"
}
