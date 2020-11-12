package core

/* This file is part of Leprechaun.
*  @author: Michael Lormann
*  `analyzer.go` holds the [basic] technical analysis logic for Leprechaun.
 */

import (
	"strings"
	"time"
)

// Analyzer defines the interface for an arbitrary analysis pipeline.
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

// PricePosition indicates whether the current price is above or below an indicator e.g. the moving average.
type PricePosition struct {
	Above, Below, Stable bool
}

var (
	// PluginHandler is the package wide handle for the registered
	// technical analysis plugins.
	PluginHandler *AnalysisPlugins
)

// AnalysisPlugins holds registered analysis plugins. Each plugin should
// 	1) Be defined in a seperate file the `plugins` package
// 	2) Register itself with a unique string identifier in its `init()` method, by calling `AnalysisPlugins.Register`
//	3) Be well documented and expose a conscise description in its Description variable. The description
// may include links to further information and explanation about the plugin.
// e.g. The Default "hermes" analyzer exposes its description as `Hermes.Description string`
//
// The UI will expose registered to the users, along with their descriptions and the selected
// by the user will be used to emit trade signals.
type AnalysisPlugins struct {
	Default Analyzer
	plugins map[string]Analyzer
}

// Register makes an analysis plugin available for use. Each plugin must
// ensure it provides a unique name. If a plugin's name clashes with that of
// a previously registered plugin, it will not be registered. All plugins are defined
// in the plugins package. see github.com/michaellormann/leprechaun/plugins.
// If only one plugin is provided, it is used as the default plugins. If multiple plugins are provided
// but the user has not specified which plugin to use, `Hermes` is used as the default plugin.
func (Plg *AnalysisPlugins) Register(name string, plugin Analyzer) {
	name = strings.ToLower(name)
	for id := range Plg.plugins {
		if name == id {
			debugf("Error! Unable to register plugin - %s, as another plugin with the same name has already been registered.", name)
			return
		}
	}
	Plg.plugins[name] = plugin
	if len(Plg.plugins) == 1 {
		Plg.Default = Plg.plugins[DefaultAnalysisPlugin]
	}
	// Logger.Printf("%s plugin registered.", name)
}

// InitPlugins returns the plugin handler to be used to access and register
// the analysis plugins.
func InitPlugins() error {
	if PluginHandler != nil {
		return nil
	}
	PluginHandler = &AnalysisPlugins{
		Default: nil,
		plugins: map[string]Analyzer{},
	}
	return nil
}

// OHLCTrend represents the general price movement of a given OHLC unit. It may be bullish or bearish.
type OHLCTrend uint

const (
	// Bullish indicates a positive price move where the closing price is higher than the opening price
	Bullish OHLCTrend = iota
	// Bearish indicates a negative price move where the opening price is higher than the closing price
	Bearish
)

// OHLC holds the Ope-High-Low-Close data for a range of prices
type OHLC struct {
	Open   float64       // Opening Price
	High   float64       // Highest Price
	Low    float64       // Lowest Price
	Close  float64       // Closing Price
	Range  float64       // Different between Opening and Closing prices
	Period time.Duration // unit of time being represented
	Trend  OHLCTrend     // Overall Price trend
	Prices *[]float64    // A pointer to the price list
}

// doOHLC to extract OHLC info from a list of prices for a given time range
func doOHLC(prices []float64) *OHLC {
	candle := &OHLC{Prices: &prices}
	candle.Close = prices[len(prices)-1]
	candle.Open = prices[0]
	candle.High = max64(prices)
	candle.Low = min64(prices)
	candle.Range = candle.Open - candle.Close
	if candle.Range < 1 {
		// Negative price movement
		candle.Trend = Bearish
	} else {
		// Positive price movement
		candle.Trend = Bullish
	}
	// candle.Period = time.Hour
	return candle

}

func min64(a []float64) float64 {
	if len(a) == 0 {
		return 0
	}
	min := a[0]
	for _, v := range a {
		if v < min {
			min = v
		}
	}
	return min
}

func max64(a []float64) float64 {
	if len(a) == 0 {
		return 0
	}
	max := a[0]
	for _, v := range a {
		if v > max {
			max = v
		}
	}
	return max
}
