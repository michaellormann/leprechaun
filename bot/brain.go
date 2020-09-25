package leprechaun

/* This file is part of Leprechaun.
*  @author: Michael Lormann
*  `brain.go` holds the [basic] technical analysis logic for Leprechaun.
TODO: Implement Fuzzy Logic version.
*/

import (
	"github.com/VividCortex/ewma"
)

// SIGNAL are signals emited by the Emit function that tell the bot what to do
type SIGNAL string

// PricePosition indicates whether the current price is above or below the moving average.
type PricePosition struct {
	above, below, stable bool
}

var (
	// WaitSignal tells Leprechaun to wait
	WaitSignal SIGNAL = "WAIT"
	// BuySignal tells Leprechaun to wait
	BuySignal SIGNAL = "BUY"
	// SellSignal tells Leprechaun to wait
	SellSignal SIGNAL = "SELL"
)

// analyze examines market data and determines whether there is an uptrend of downtrend of price
func analyze(prices []float64) (score int64) {
	// todo:: just analyze price trend without any ema
	// It is helpful to get an odd number of prices to ensure there is
	// always a clear price trend. It is possible for half of an even nunmber
	// of prices to exhibit a pattern that is equal to the other half, thus
	// making the price pattern undecided. `11`, `7` or `5` are good choices.
	// Here we retrieve 11 previous prices, 5 minutes apart.
	// Determine the price trend.
	// If the price at any point is higher than its next price it
	// signifies a drop in price, and vice versa.
	// If the score is positive, there has been a relative uptrend in price movement
	// if the score is negative, price movement has been downward
	for x := 0; x < len(prices)-1; x++ {
		if prices[x] > prices[x+1] {
			score--
		} else if prices[x] < prices[x+1] {
			score++
		}
	}
	// Todo: Calculate exponential moving average of prices with respect to time.
	// This is to determine postion of current price relative to the moving average.
	return
}

// doEma computes the exponential moving average for past prices collected from the exchange.
func doEma(prices []float64) float64 {
	ema := ewma.NewMovingAverage()
	for _, price := range prices {
		ema.Add(price)
	}
	// fmt.Println("EMA: ", ema.Value())
	return ema.Value()
}

// Emit emits a BUY, SELL or WAIT signal based on data from `analyze()`
func (cl *Client) Emit() (signal SIGNAL, err error) {
	// TODO:: USE LUNO ORER REQUEST V2 TO SEE WHAT ORDERS ARE IN THE ORDERBOOK.
	// IF AN ORDER HAS A HIGH NUMBER OF ASSET ATTACHED TO IT AND IT IS RELATIVELY CLSE TO YOUR PROFIT MARK
	// YOU CAN ALIGN WITH IT.
	var (
		prices       []float64
		currentPrice float64
		pricesErr    error
	)
	for errCount := 0; errCount < 3; errCount++ {
		prices, pricesErr = cl.PreviousPrices(11, 25)
		currentPrice, err = cl.CurrentPrice()
		if cancelled() {
			return WaitSignal, ErrCancelled
		}
		if err == nil && pricesErr == nil {
			break
		}
	}
	if err != nil {
		debug("Could not retrieve price data from the exchange. Please check your network connection!", err.Error())
		return WaitSignal, err
	}
	// Price position.
	pos := PricePosition{}
	ema := doEma(prices)
	if currentPrice < ema {
		// current price is below the ema
		pos.below = true
	} else if currentPrice > ema {
		// current price is above the ema
		pos.above = true
	} else {
		// price is relatively stable
		pos.stable = true
	}

	// TODO:: Use Contrarian technique from python implementation
	// Price direction and ema position combined.
	// If direction is DOWN and EMA is above: SELL
	// If direction is DOWN and EMA is below: BUY
	// If direction is UP and EMA is above: SELL
	// If direction is UP and EMA is below: BUY
	// Price direction
	score := analyze(prices)
	if score < 0 {
		// Price trend is downward
		if pos.above {
			signal = SellSignal
		} else if pos.below {
			signal = BuySignal
		} else {
			signal = WaitSignal
		}
	} else if score > 0 {
		// Price trend is upward
		if pos.above {
			signal = SellSignal
		} else if pos.below {
			signal = BuySignal
		} else {
			signal = WaitSignal
		}
	} else {
		// Price direction is indeterminate
	}
	return signal, nil
}
