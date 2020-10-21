package core

/* This file is part of Leprechaun.
*  @author: Michael Lormann
*  `brain.go` holds the [basic] technical analysis logic for Leprechaun.
TODO: Implement Fuzzy Logic based rules.
*/

import (
	"fmt"
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
	// SignalWait ...
	SignalWait SIGNAL = "WAIT"
	// SignalBuy ...
	SignalBuy SIGNAL = "BUY"
	// SignalSell ...
	SignalSell SIGNAL = "SELL"
	// SignalLong tells the bot to initiate a long trade
	SignalLong SIGNAL = "GO_LONG"
	// SignalShort tells the bot to short sell an asset.
	SignalShort SIGNAL = "SHORT_SELL"
)

// TradeMode specifies the manner an upward or downward price trend is interpreted by Leprechaun.
// TradeMode only deals with price trend and hence it should not be the only indicator used in price
// technical analysis.
type TradeMode uint

const (
	// Contrarian Mode assumes that a price trend in any direction will be followed
	// by a reversal in the opposite direction. For example, if the price of an asset has been
	// steadily falling for a period of time, Contrarian mode lets the bot buy the asset, with
	// the hope of selling it at a higher price when the trend reverses. Conversely, an asset
	// on an uptrend is sold to hedge against losses when the price trend reverses.
	Contrarian TradeMode = iota
	// TrendFollowing mode assumes that a price movement in any direction (up or down)
	// will tend to continue in that manner. For example, if the price of an asset, say Bitcoin,
	// has been rising steadily for the past few hours, this mode assumes that the price will
	// continue to rise even further, this means the bot buys an asset on the rise with the hope
	// of selling it at an even higher price.
	TrendFollowing
)

// $$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$

// PricePosition indicates whether the current price is above or below the moving average.
type PricePosition struct {
	above, below, stable bool
}

// Hermes is the default analysis plugin for leprechaun.
// It supports two trade modes, the contrarian and trend following modes.
// It combines these modes with the principle of mean reversion, to decide
// whether to buy an asset or not.
// Other plugins that satisfy the Analyzer interface may be used instead.
type Hermes struct {
	// todo:: Imlement fuzzy logic based rules for the emit function.

	NumPrices     int           // Number of historical price points to be analyzed
	PriceInterval time.Duration // Time interval between each price point
	prices        []float64
	movingAverage float64
	mAvgWindow    int // Moving Average window
	score         int
	pos           PricePosition
	currentPrice  float64
	tradeMode     TradeMode
}

// DefaultAnalysisPlugin ...
func DefaultAnalysisPlugin(NumPrices int, PriceInterval time.Duration, tradingMode TradeMode) Analyzer {
	return &Hermes{
		NumPrices:     NumPrices,
		PriceInterval: PriceInterval,
		tradeMode:     tradingMode,
	}
}

// PriceDimensions returns two parameters. The number of past prices to be retrieved, and
// the time interval between each price point.
func (plugin *Hermes) PriceDimensions() (int, time.Duration) {
	return plugin.NumPrices, plugin.PriceInterval
}

// Analyze examines market data and determines whether there is an uptrend of downtrend of price
func (plugin *Hermes) Analyze(prices []float64) (err error) {
	// Note: this function is a work in progress, it currently holds very simple techniques that
	// will be updated later.
	// todo:: provide option to just analyze price trend without any ema, i.e. don't take mean reversion into
	// consideration.
	plugin.prices = prices
	plugin.currentPrice = prices[0] // Most recent price is the first in the slice

	// Determine the current price position with respect to the moving average
	plugin.doPricePosition()
	// Determine the price movement
	plugin.Score()
	return nil
}

// Score the prices to determine the price trend.
func (plugin *Hermes) Score() {
	// It is helpful to get an odd number of prices to ensure there is
	// always a clear price trend. It is possible for half of an even nunmber
	// of prices to exhibit a pattern that is equal to the other half, thus
	// making the price pattern undecided. `11`, `7` or `5` are good choices.
	// Here we retrieve 11 previous prices, 5 minutes apart.
	// If the price at any point is higher than its next price it
	// signifies a drop in price, and vice versa.
	// If the score is positive, there has been a relative uptrend in price movement
	// if the score is negative, price movement has been downward
	plugin.score = 0
	for x := 0; x < len(plugin.prices)-1; x++ {
		if plugin.prices[x] > plugin.prices[x+1] {
			plugin.score--
		} else if plugin.prices[x] < plugin.prices[x+1] {
			plugin.score++
		}
	}
}

// doEMA computes the exponential moving average for past prices collected from the exchange.
func (plugin *Hermes) doEMA() {
	ema := ewma.NewMovingAverage()
	for _, price := range plugin.prices {
		ema.Add(price)
	}
	// fmt.Println("EMA: ", ema.Value())
	plugin.movingAverage = ema.Value()
}

// doPricePosition determines postion of current price relative to the moving average.
func (plugin *Hermes) doPricePosition() {
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
func (plugin *Hermes) Emit() (signal SIGNAL) {
	// TODO:: USE LUNO ORDER REQUEST V2 TO SEE WHAT ORDERS ARE IN THE ORDERBOOK.
	// IF AN ORDER HAS A HIGH NUMBER OF ASSET ATTACHED TO IT AND IT IS RELATIVELY CLOSE TO YOUR PROFIT MARK
	// YOU CAN ALIGN WITH IT.

	// Price trend is downward
	if plugin.score < 0 {
		fmt.Println("Price trend is downward!")
		// The current price of the asset is above the moving average.
		if plugin.pos.above {
			switch plugin.tradeMode {

			// Go against the mean reversion principle
			case Contrarian:
				// signal = SignalBuy
				// Go long. Contrarian mode dictates we expect the price to stay above the moving average
				// In this case it is expected to rise.
				signal = SignalLong

			// Follow the Mean reversion principle
			case TrendFollowing:
				// signal = SignalSell
				// Short sell the asset, since according to the mean reversion principle we expect
				// the price to normalize with the moving average, in this case, the price will go down.
				signal = SignalShort
			}
		} else if plugin.pos.below {
			// The current price is below the moving average

			switch plugin.tradeMode {

			// Go against the mean reversion principle
			case Contrarian:
				// signal = SignalSell
				// Expect the price to keep dropping, even though it is alreay below the moving average.
				// i.e. Sell High, Buy Low
				signal = SignalShort

			// Follow the Mean reversion principle
			case TrendFollowing:
				// signal = SignalBuy
				// Here the price is expected to rise back towards the moving average.
				// So a long trade is initiated. i.e. Buy Low, Sell High.
				signal = SignalLong
			}

		} else {
			signal = SignalWait
		}
	} else if plugin.score > 0 { // Price trend is upward
		fmt.Println("Price trend is upward!")
		if plugin.pos.above { // Current price is above the moving average.
			switch plugin.tradeMode {

			// Follow the mean reversion principle
			case Contrarian:
				// signal = SignalSell
				// Go short. Contrarian mode dictates we expect the price to return downwards below the moving average
				// In this case it is expected to rise.
				signal = SignalShort

			case TrendFollowing:
				// signal = SignalBuy
				// We expect the price trend to continue upwards. We dont follow the mean reversion in this case.
				signal = SignalLong
			}
		} else if plugin.pos.below {
			switch plugin.tradeMode {

			// Follow the mean reversion principle
			case Contrarian:
				// signal = SignalBuy
				signal = SignalLong

			// Go against the mean reversion principle
			case TrendFollowing:
				// signal = SignalSell
				signal = SignalShort
			}
		} else {
			signal = SignalWait
		}
	} else {
		// Price direction is indeterminate
		return SignalWait
	}

	return signal
}
