package leprechaun

/* This file is part of Leprechaun.
*  @author: Michael Lormann
*  `brain.go` holds the [basic] technical analysis logic for Leprechaun.
TODO: Implement Fuzzy Logic version.
*/

import (
	"time"

	"github.com/VividCortex/ewma"
)

// Analyzer defines the interface for an arbitrary analysis plugin
// the `Analyze` function takes in a list of historical prices of any asset
// the `PriceInterval` and `NumPrices` fields specify the duration between each
// price point and the number of prices to retrieve, and the `Emit` function
// returns the signal based on the analysis done.
type Analyzer interface {
	// PriceDimensions returns two parameters. The number of past prices to be retrieved, and
	// the interval between each price point.
	PriceDimensions() (int, time.Duration)
	// Analyze is passed the historical price data received from the exchange and presumably
	// does the technical analysis of the price data
	Analyze(prices []float64) error
	// Emit returns the final market signal based on the analysis
	Emit() SIGNAL
}

// SIGNAL is emitted by the Emit function based on results from the technical analysis
//  that tell the bot what to do
type SIGNAL string

const (
	SignalWait SIGNAL = "WAIT"
	SignalBuy  SIGNAL = "BUY"
	SignalSell SIGNAL = "SELL"
)

// $$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$

// PricePosition indicates whether the current price is above or below the moving average.
type PricePosition struct {
	above, below, stable bool
}

type DefaultAnalysisPlugin struct {
	NumPrices     int           // Number of historical price points to be analyzed
	PriceInterval time.Duration // Time interval between each price point
	prices        []float64
	movingAverage float64
	mAvgWindow    int // Moving Average window
	score         int
	pos           PricePosition
	currentPrice  float64
}

func (plugin *DefaultAnalysisPlugin) PriceDimensions() (int, time.Duration) {
	return plugin.NumPrices, plugin.PriceInterval
}

// Analyze examines market data and determines whether there is an uptrend of downtrend of price
func (plugin *DefaultAnalysisPlugin) Analyze(prices []float64) (err error) {
	// Note: this function is a work in progress, it currently holds very simple techniques that
	// will be updated later.
	// todo:: just analyze price trend without any ema
	plugin.prices = prices
	plugin.currentPrice = prices[0] // Most recent price is the first in the slice

	// Determine the current price position with respect to the moving average
	plugin.doPricePosition()

	// Score the price
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
	plugin.score = 0
	for x := 0; x < len(prices)-1; x++ {
		if prices[x] > prices[x+1] {
			plugin.score--
		} else if prices[x] < prices[x+1] {
			plugin.score++
		}
	}
	return nil
}

// doEMA computes the exponential moving average for past prices collected from the exchange.
func (plugin *DefaultAnalysisPlugin) doEMA() {
	ema := ewma.NewMovingAverage()
	for _, price := range plugin.prices {
		ema.Add(price)
	}
	// fmt.Println("EMA: ", ema.Value())
	plugin.movingAverage = ema.Value()
}

// doPricePosition determines postion of current price relative to the moving average.
func (plugin *DefaultAnalysisPlugin) doPricePosition() {
	plugin.pos = PricePosition{}
	plugin.doEMA()
	if plugin.currentPrice < plugin.movingAverage {
		// current price is below the ema
		plugin.pos.below = true
	} else if plugin.currentPrice > plugin.movingAverage {
		// current price is above the ema
		plugin.pos.above = true
	} else {
		// price is relatively stable
		plugin.pos.stable = true
	}

}

// Emit emits a BUY, SELL or WAIT signal based on data from `analyze()`
func (plugin *DefaultAnalysisPlugin) Emit() (signal SIGNAL) {
	// TODO:: USE LUNO ORER REQUEST V2 TO SEE WHAT ORDERS ARE IN THE ORDERBOOK.
	// IF AN ORDER HAS A HIGH NUMBER OF ASSET ATTACHED TO IT AND IT IS RELATIVELY CLSE TO YOUR PROFIT MARK
	// YOU CAN ALIGN WITH IT.

	// TODO:: Use Contrarian technique from python implementation
	// Price direction and ema position combined.
	// If direction is DOWN and EMA is above: SELL
	// If direction is DOWN and EMA is below: BUY
	// If direction is UP and EMA is above: SELL
	// If direction is UP and EMA is below: BUY

	if plugin.score < 0 {
		// Price trend is downward
		if plugin.pos.above {
			signal = SignalSell
		} else if plugin.pos.below {
			signal = SignalBuy
		} else {
			signal = SignalWait
		}
	} else if plugin.score > 0 {
		// Price trend is upward
		if plugin.pos.above {
			signal = SignalSell
		} else if plugin.pos.below {
			signal = SignalBuy
		} else {
			signal = SignalWait
		}
	} else {
		// Price direction is indeterminate
		return SignalWait
	}
	return signal
}
