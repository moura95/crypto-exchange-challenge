package orderbook

import (
	"fmt"
	"time"
)

type Match struct {
	Bid        *Order
	Ask        *Order
	Price      float64
	SizeFilled float64
	Timestamp  time.Time
}

func (m Match) String() string {
	return fmt.Sprintf("[Match: %.8f @ %.2f | Buyer:%s Seller:%s]",
		m.SizeFilled, m.Price, m.Bid.UserID, m.Ask.UserID)
}
