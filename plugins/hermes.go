package plugins

import (
	"fmt"
	"log"
	"time"

	"github.com/VividCortex/ewma"
	core "github.com/michaellormann/leprechaun/core"
)

func init() {
	// This is added here, because init functions execute in no given order,
	// so in the event this function is executed before the bot's init function,
	// plugins are still initialized.
	err := core.InitPlugins()
	if err != nil {
		log.Fatal("Could not initialize plugins")
	}
	// Register the plugin
	core.PluginHandler.Register("hermes", Hermes{NumPrices: 25, PriceInterval: 60 * time.Minute})
	// TODO: Expose the price dimensions parameters to the user, so they can change it if they want to.
	log.Println("hermes plugin registered")
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
	pos           core.PricePosition
	currentPrice  float64
	tradeMode     core.TradeMode
}

var (
	// TrendFollowing (see core.TrendFollowing)
	TrendFollowing = core.TrendFollowing
	// Contrarian (see core.Contrarian)
	Contrarian = core.Contrarian
)

// DefaultAnalysisPlugin ...
func DefaultAnalysisPlugin(NumPrices int, PriceInterval time.Duration, tradingMode core.TradeMode) core.Analyzer {
	return &Hermes{
		NumPrices:     NumPrices,
		PriceInterval: PriceInterval,
		tradeMode:     tradingMode,
	}
}

// PriceDimensions returns two parameters. The number of past prices to be retrieved, and
// the time interval between each price point.
func (plugin Hermes) PriceDimensions() (int, time.Duration) {
	return plugin.NumPrices, plugin.PriceInterval
}

// Analyze examines market data and determines whether there is an uptrend of downtrend of price
func (plugin Hermes) Analyze(prices []float64) (err error) {
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
func (plugin Hermes) Score() {
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
func (plugin Hermes) doEMA() {
	ema := ewma.NewMovingAverage()
	for _, price := range plugin.prices {
		ema.Add(price)
	}
	// fmt.Println("EMA: ", ema.Value())
	plugin.movingAverage = ema.Value()
}

// doPricePosition determines postion of current price relative to the moving average.
func (plugin Hermes) doPricePosition() {
	plugin.pos = core.PricePosition{}
	plugin.doEMA()
	if plugin.currentPrice < plugin.movingAverage {
		// current price is below the ema
		plugin.pos.Below = true
	} else if plugin.currentPrice > plugin.movingAverage {
		// current price is above the ema
		plugin.pos.Above = true
	} else {
		// price is relatively stable
		plugin.pos.Stable = true
	}

}

// Emit emits a BUY, SELL or WAIT signal based on data from `analyze()`
func (plugin Hermes) Emit() (signal core.SIGNAL) {
	// TODO:: USE LUNO ORDER REQUEST V2 TO SEE WHAT ORDERS ARE IN THE ORDERBOOK.
	// IF AN ORDER HAS A HIGH NUMBER OF ASSET ATTACHED TO IT AND IT IS RELATIVELY CLOSE TO YOUR PROFIT MARK
	// YOU CAN ALIGN WITH IT.

	// Price trend is downward
	if plugin.score < 0 {
		fmt.Println("Price trend is downward!")
		// The current price of the asset is above the moving average.
		if plugin.pos.Above {
			switch plugin.tradeMode {

			// Go against the mean reversion principle
			case core.Contrarian:
				// signal = SignalBuy
				// Go long. Contrarian mode dictates we expect the price to stay above the moving average
				// In this case it is expected to rise.
				signal = core.SignalLong

			// Follow the Mean reversion principle
			case TrendFollowing:
				// signal = SignalSell
				// Short sell the asset, since according to the mean reversion principle we expect
				// the price to normalize with the moving average, in this case, the price will go down.
				signal = core.SignalShort
			}
		} else if plugin.pos.Below {
			// The current price is below the moving average

			switch plugin.tradeMode {

			// Go against the mean reversion principle
			case Contrarian:
				// signal = SignalSell
				// Expect the price to keep dropping, even though it is alreay below the moving average.
				// i.e. Sell High, Buy Low
				signal = core.SignalShort

			// Follow the Mean reversion principle
			case TrendFollowing:
				// signal = SignalBuy
				// Here the price is expected to rise back towards the moving average.
				// So a long trade is initiated. i.e. Buy Low, Sell High.
				signal = core.SignalLong
			}

		} else {
			signal = core.SignalWait
		}
	} else if plugin.score > 0 { // Price trend is upward
		fmt.Println("Price trend is upward!")
		if plugin.pos.Above { // Current price is above the moving average.
			switch plugin.tradeMode {

			// Follow the mean reversion principle
			case Contrarian:
				// signal = SignalSell
				// Go short. Contrarian mode dictates we expect the price to return downwards below the moving average
				// In this case it is expected to rise.
				signal = core.SignalShort

			case TrendFollowing:
				// signal = SignalBuy
				// We expect the price trend to continue upwards. We dont follow the mean reversion in this case.
				signal = core.SignalLong
			}
		} else if plugin.pos.Below {
			switch plugin.tradeMode {

			// Follow the mean reversion principle
			case Contrarian:
				// signal = SignalBuy
				signal = core.SignalLong

			// Go against the mean reversion principle
			case TrendFollowing:
				// signal = SignalSell
				signal = core.SignalShort
			}
		} else {
			signal = core.SignalWait
		}
	} else {
		// Price direction is indeterminate
		return core.SignalWait
	}

	return signal
}
