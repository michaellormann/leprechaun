package plugins

import (
	"fmt"
	"log"
	"time"

	"github.com/VividCortex/ewma"
	core "github.com/michaellormann/leprechaun/core"
)

var (
	HRM = Hermes{NumPrices: 21, PriceInterval: 60 * time.Minute}
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
	// core.PluginHandler.Register("hermes", Hermes{NumPrices: 21, PriceInterval: 60 * time.Minute})
	core.PluginHandler.Register("hermes", HRM)
	core.PluginHandler.Default = HRM
	// TODO: Expose the price dimensions parameters to the user, so they can change it if they want to.
	log.Println("hermes plugin registered")
}

// Hermes is the default analysis plugin for leprechaun.
// It supports two trade modes, the contrarian and trend following modes.
// It combines these modes with the principle of mean reversion, and the RSI oscillator to decide
// whether to buy an asset or not.
// Other plugins that satisfy the Analyzer interface may be used instead.
type Hermes struct {
	// todo:: Imlement fuzzy logic based rules for the emit function.

	NumPrices        int           // Number of historical price points to be analyzed
	PriceInterval    time.Duration // Time interval between each price point
	prices           []float64
	movingAverage    float64
	mAvgWindow       int // Moving Average window
	score            int
	pos              core.PricePosition
	currentPrice     float64
	tradeMode        core.TradeMode
	candles          []*core.OHLC
	CandlestickChart core.CandleChart
	LineChart        core.LineChart
	predictedMove    core.ChartTrend
	options          *core.AnalysisOptions
	indicatorScores  []Score
}

const (
	// NumIndicators is the number of indicators that hermes examines and scores to arrive at
	// a final score which then influences the signal emitted. The parameters examined are:
	// 1. The pattern of the most recent candlesticks in the chart (latest 5).
	//    Hermes checks if the candlesticks form a common bullish or bearish pattern
	//    and each pattern has a corresponding score.
	// 2. The Moving Average.
	//    The current price is examined to see if it is below or above the moving average. Each position
	//    has its own score with respect to the overall chart trend. The margin of the difference between
	// 	  the current price and the moving average also influences the score. The greater the margin the
	//    higher the score.
	// 3. Oscillators (Under construction)
	NumIndicators = 3
)

// Score is a weigth for each analysis parameter.
// Each parameter has its own score
type Score float64

const (
	// ScoreOne is a Full score. indicates that the overall chart pattern is expected to continue
	ScoreOne Score = 1.0
	// ScoreHalf is half a score. whether chart pattern will be maintained can't be independently determined.
	// Is applied when there is a reveral/opposition to overall chart pattern
	ScoreHalf Score = 0.5
	// ScoreQuarter is unimplemented.
	ScoreQuarter Score = 0.25
	// Score3Quarter is unimplemented.
	Score3Quarter Score = 0.75
)
const (
	// Scores for the Indicators hermes checks
	// TODO:: add bollinger bands inicator
	bullChartMajorBullPattern         = ScoreOne
	bullChartMajorBullPatternReversal = Score3Quarter
	bullChartMajorBearPattern         = ScoreHalf
	bullChartMajorBearPatternReversal = ScoreQuarter
	bearChartMajorBearPattern         = ScoreOne
	bearChartMajorBearPatternReversal = Score3Quarter
	bearChartMajorBullPattern         = ScoreHalf
	bearChartMajorBullPatternReversal = ScoreQuarter

	bullChartMinorBullPattern         = Score3Quarter
	bullChartMinorBullPatternReversal = ScoreHalf
	bullChartMinorBearPattern         = ScoreQuarter
	bullChartMinorBearPatternReversal = ScoreQuarter
	bearChartMinorBearPattern         = Score3Quarter
	bearChartMinorBearPatternReversal = ScoreHalf
	bearChartMinorBullPattern         = ScoreQuarter
	bearChartMinorBullPatternReversal = ScoreQuarter

	bullChartAboveMA   = ScoreHalf
	bullChartBelowMA   = ScoreOne
	bearChartAboveMA   = ScoreOne
	bearChartBelowMA   = ScoreHalf
	bearChartRSITop    = ScoreOne
	bearChartRSIBottom = ScoreHalf
	bullChartRSITop    = ScoreHalf
	bullChartRSIBottom = ScoreOne
)

var (
	// TrendFollowing (see core.TrendFollowing)
	TrendFollowing = core.TrendFollowing
	// Contrarian (see core.Contrarian)
	Contrarian = core.Contrarian
	// Bearish (see core.Bearish)
	Bearish = core.Bearish
	// Bullish (see core.Bullish)
	Bullish = core.Bullish
)

// Description gives a brief summary of what the plugin does and how.
func (plugin Hermes) Description() string {
	return `"Hermes is an analysis pipeline that analyzes the prices of an asset by checking a number of indicators.
	each indicator is scored and a final score is extracted from the combination of all indicators. The final score
	the determines the signal emitted by  Hermes, i.e BUY, SELL, WAIT"`
}

// SetOptions configures the plugin with the bots specifications
func (plugin Hermes) SetOptions(opts *core.AnalysisOptions) error {
	plugin.options = opts
	totalAnalysisHours := opts.AnalysisPeriod.Hours()
	plugin.NumPrices = int(totalAnalysisHours / opts.Interval.Hours())
	plugin.tradeMode = opts.Mode
	return nil
}

// SetCurrentPrice ...
func (plugin Hermes) SetCurrentPrice(price float64) error {
	plugin.currentPrice = price
	return nil
}

// SetClosingPrices ...
func (plugin Hermes) SetClosingPrices(prices []float64) error {
	plugin.prices = prices
	plugin.LineChart = core.NewLineChart(prices)
	return nil
}

// SetOHLC ...
func (plugin Hermes) SetOHLC(candles []core.OHLC) error {
	plugin.CandlestickChart = core.NewCandleChart(candles)
	return nil
}

func (plugin Hermes) addScore(score Score) {
	plugin.indicatorScores = append(plugin.indicatorScores, score)
}

// Analyze examines market data and determines whether there is an uptrend of downtrend of price
func (plugin Hermes) analyze() (err error) {
	// Note: this function is a work in progress, it currently holds very simple techniques that
	// will be updated later.
	// todo:: provide option to just analyze price trend without any ema, i.e. don't take mean reversion into
	// consideration.

	// Determine the current price position with respect to the moving average
	plugin.doPricePosition()
	// Determine the price movement
	plugin.LineChart.DetectTrend()

	// TODO:: Check for corellation between closing price trend an candlestick chart trend.

	chartTrend := plugin.CandlestickChart.DetectTrend(plugin.CandlestickChart.Candles) // Overall trend of the candle chart
	plugin.CandlestickChart.DetectPatterns()                                           //Note: only the most recent candles are checked for common candlestick patterns

	// Compare candle chart trend and line chart trend. If they are different something might be wrong.
	log.Printf("Line chart trend for current price data is: %v", plugin.LineChart.Trend)
	log.Printf("Candle chart trend for current price data is: %v", chartTrend)

	switch chartTrend {
	case Bullish: // Overall chart sentiment for the period in question is bullish
		log.Printf("The overall trend of the %v chart is %v", plugin.PriceInterval, core.Bullish)
		for n, detectedBullishPattern := range plugin.CandlestickChart.BullishPatterns {
			log.Printf("Detected bullish pattern[%d]: %v", n, detectedBullishPattern)
			// Trend reversal may likely if there is further bearish movement.
			switch detectedBullishPattern.Pattern {
			// Major bullish patterns
			case core.BullishEngulfingPattern, core.BullishKeyReversal, core.BullishMorningStar,
				core.BullishRisingThree, core.BullishRisingTwo, core.MorningDojiStar:
				if detectedBullishPattern.PreceedingTrend.IsBullish() {
					// bullish continuation pattern.
					plugin.addScore(bullChartMajorBullPattern)
				} else if detectedBullishPattern.PreceedingTrend.IsBearish() {
					// bullish trend preceeded by a bearish pattern.
					plugin.addScore(bullChartMajorBullPatternReversal)
				}

			case core.BullishHarami, core.BullishHaramiCross:
				if detectedBullishPattern.PreceedingTrend.IsBullish() {
					// bullish continuation pattern.
					plugin.addScore(bullChartMinorBullPattern)
				} else if detectedBullishPattern.PreceedingTrend.IsBearish() {
					// bullish trend preceeded by a bearish pattern.
					plugin.addScore(bullChartMinorBullPatternReversal)
				}
			}
		}

		for n, detectedBearishPattern := range plugin.CandlestickChart.BearishPatterns {
			// TODO:: Check if these patterns occur at the `top` or `bottom` level.
			log.Printf("Detected bearish pattern[%d]: %v", n, detectedBearishPattern)
			switch detectedBearishPattern.Pattern {
			case core.BearishEngulfingPattern, core.BearishEveningStar, core.BearishKeyReversal,
				core.EveningDojiStar, core.BearishFallingTwo, core.BearishFallingThree:
				// These patterns are highly indicative of further bearish movement.
				// [REMOVE] Trend reversal is likely imminent. esp. if these patterns occur at the top.
				if detectedBearishPattern.PreceedingTrend.IsBearish() {
					// bearish continuation pattern.
					plugin.addScore(bullChartMajorBearPattern)
				} else if detectedBearishPattern.PreceedingTrend.IsBullish() {
					// bearish pattern preceeded by a bullish trend. i.e. current trend is a reversal
					plugin.addScore(bullChartMajorBearPatternReversal)
				}
			case core.BearishHarami, core.BearishHaramiCross:
				if detectedBearishPattern.PreceedingTrend.IsBullish() {
					// TODO:: Refine this segment, possibly define new scores for the above patterns
					plugin.addScore(bullChartMinorBearPatternReversal)
				} else if detectedBearishPattern.PreceedingTrend.IsBearish() {
					plugin.addScore(bullChartMinorBearPattern)
				}
			}
		}

	case Bearish: // Overall chart sentiment for the period in question is bearish.
		log.Printf("The overall trend of the %v chart is %v", plugin.PriceInterval, core.Bearish)
		for n, pattern := range plugin.CandlestickChart.BearishPatterns {
			log.Printf("Detected bearish pattern[%d]: %v", n, pattern)
			switch pattern.Pattern {
			// TODO:: Check if these patterns occur at the `top` or `bottom` level.
			case core.BearishEngulfingPattern, core.BearishEveningStar, core.BearishKeyReversal,
				core.EveningDojiStar, core.BearishFallingTwo, core.BearishFallingThree:
				// These patterns are highly indicative of further bearish movement.
				// [REMOVE] Trend reversal is likely imminent. esp. if these patterns occur at the top.
				if pattern.PreceedingTrend.IsBullish() {
					// bearish pattern preceeded by a bullish trend. i.e. current trend is a reversal
					plugin.addScore(bearChartMajorBearPatternReversal)
				} else if pattern.PreceedingTrend.IsBearish() {
					// bearish continuation pattern.
					plugin.addScore(bearChartMajorBearPattern)
				}
				// if plugin.tradeMode == TrendFollowing {
				// 	// If bearish reversal occurs near the bottom keep going, else reverse trade direction.
				// } else if plugin.tradeMode == Contrarian {
				// // If bearish reversals occurs near top or bottom, keep going.s
				// if pattern.PreceedingTrend.IsBullish() {
				// 	// bearish pattern preceeded by a bullish trend.
				// } else if pattern.PreceedingTrend.IsBearish() {
				// 	// bearish continuation pattern.
				// }
			case core.BearishHarami, core.BearishHaramiCross:
				if pattern.PreceedingTrend.IsBullish() {
					// TODO:: Refine this segment, possibly define new scores for the above patterns
					plugin.addScore(bearChartMajorBearPatternReversal - ScoreQuarter)
				} else if pattern.PreceedingTrend.IsBearish() {
					plugin.addScore(bearChartMajorBearPattern - ScoreQuarter)
				}
			}
		}

		for n, pattern := range plugin.CandlestickChart.BullishPatterns {
			log.Printf("Detected bullish pattern[%d]: %v", n, pattern)
			// Trend reversal may likely if there is further bearish movement.
			switch pattern.Pattern {
			// Major bullish patterns
			case core.BullishEngulfingPattern, core.BullishKeyReversal, core.BullishMorningStar,
				core.BullishRisingThree, core.BullishRisingTwo, core.MorningDojiStar:
				if pattern.PreceedingTrend.IsBullish() {
					// bullish continuation pattern.
					plugin.addScore(bearChartMajorBullPattern)
				} else if pattern.PreceedingTrend.IsBearish() {
					// bullish trend preceeded by a bearish pattern.
					plugin.addScore(bearChartMajorBullPatternReversal)
				}

			case core.BullishHarami, core.BullishHaramiCross:
				if pattern.PreceedingTrend.IsBullish() {
					// bullish continuation pattern.
					plugin.addScore(bearChartMajorBearPattern - ScoreQuarter)
				} else if pattern.PreceedingTrend.IsBearish() {
					// bullish trend preceeded by a bearish pattern.
					plugin.addScore(bearChartMajorBearPattern - ScoreQuarter)
				}
			}
		}
	}

	return nil
}

// Score the parameters examined to get a final score.
func (plugin Hermes) Score() {

}

// doEMA computes the exponential moving average for past prices collected from the exchange.
func (plugin Hermes) doEMA() {
	ema := ewma.NewMovingAverage()
	fmt.Println("In ema")
	for _, price := range plugin.prices {
		ema.Add(price)
	}
	fmt.Println("EMA: ", ema.Value())
	plugin.movingAverage = ema.Value()
}

// doPricePosition determines postion of current price relative to the moving average.
func (plugin Hermes) doPricePosition() {
	// TODO:: The margin should be compared as a percentage of the difference between the current price and
	// the most recent price point. i.e. P(n) - P(n-1) = std_margin. margin% = margin/std_margin * 100
	plugin.pos = core.PricePosition{}
	plugin.doEMA()
	if plugin.currentPrice < plugin.movingAverage {
		// current price is below the ema
		plugin.pos.Below = true
		plugin.pos.Margin = plugin.currentPrice - plugin.movingAverage
	} else if plugin.currentPrice > plugin.movingAverage {
		// current price is above the ema
		plugin.pos.Above = true
		plugin.pos.Margin = plugin.currentPrice - plugin.movingAverage
	} else {
		// price is relatively stable
		plugin.pos.Stable = true
		plugin.pos.Margin = plugin.currentPrice - plugin.movingAverage
	}
}

// Emit emits a BUY, SELL or WAIT signal based on data from `analyze()`
func (plugin Hermes) Emit() (signal core.SIGNAL, err error) {
	// TODO:: USE LUNO ORDER REQUEST V2 TO SEE WHAT ORDERS ARE IN THE ORDERBOOK.
	// IF AN ORDER HAS A HIGH NUMBER OF ASSET ATTACHED TO IT AND IT IS RELATIVELY CLOSE TO YOUR PROFIT MARK
	// YOU CAN ALIGN WITH IT.

	// TODO:: FINAL SCORING SHOULD BE IMPLEMENTED WITH FUZZY LOGIC.

	err = plugin.analyze()
	if err != nil {
		return core.SignalWait, err
	}

	// Price trend is downward
	if plugin.LineChart.Trend.IsBearish() {
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
	} else if plugin.LineChart.Trend.IsBullish() { // Price trend is upward
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
		return core.SignalWait, nil
	}

	return signal, nil
}
