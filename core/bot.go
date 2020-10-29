package core

/* Leprechaun is a trading bot built upon the luno API.
It uses technical analysis to monitor price trends of crypto assets and executes trades based
on signals emited by the analysis engine. The simplest and primary technique involves the direction
of price for an asset, say Bitcoin. If the price has been steadily increasing

*/

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	luno "github.com/luno/luno-go"
)

var (
	// Leprechaun ...
	Leprechaun = "Leprechaun"
)

// Supported exchanges
var (
	ExchangeLuno string = "LUNO"
	// ExchangeQuidax string = "QUIDAX"
)

var (
	initialRound      bool   = false
	loggerinitialized bool   = false
	timeFormat        string = "2006-01-02 15:04:05"
	// Logger for bot-related operations.
	Logger *log.Logger

	bot    *Bot
	config *Configuration
)

// SetConfig sets the package wide Configuration values for early access. (to be used by the UI.)
func SetConfig(cfg *Configuration) {
	config = cfg
}

// Channels
var (
	UIChans *Channels
	// logChannel exposes the bot's log channel to the debug and debugf funcs. For testing only.
	logChannel chan string

	channelsInitialized bool
	// function that checks if any cancel signal has been sent from the UI.
	cancelled func() bool
)

// Errors
var (
	botStoppedMessage = "\nLeprechaun has stopped."
	// ErrCancelled is sent by the bot goroutine when it receives a signal from
	// the cancel channel
	ErrCancelled = errors.New("the trading session has been cancelled")
	// ErrInvalidAPICredentials is returned for an invalid API key ID or key Secret
	ErrInvalidAPICredentials = errors.New("invalid API credentials")
	// ErrChannelsNotInitialized is returned when the UI does not instatiate all necessary channels
	ErrChannelsNotInitialized = errors.New("unable to start multiplexing. Initialize channels first")
	// ErrInvalidPurchaseUnit is returned when the user wants to buy lower than the minimum trading volume
	ErrInvalidPurchaseUnit = errors.New("the purchase amount you specified is lower than the minimum trading volume")
)

// SetLogger sets the logger object specific for leprechaun's bot activities.
// TODO: use the slog-backend format and move this log to the main.go file or a seperate log file.
func SetLogger(logger *log.Logger) {
	// Initialize the logger
	if loggerinitialized {
		return
	}
	Logger = logger
	loggerinitialized = true
	return
}

// InitChannels sets the channels that the bot goroutine uses to communicate with the UI.
func (bot *Bot) InitChannels(chans *Channels) {
	UIChans = chans
	bot.chans = chans
	// for debug funcs
	logChannel = bot.chans.LogChan
	cancelled = func() bool {
		select {
		case <-UIChans.CancelChan:
			// Send a signal to the UI that we have recieved its STOP signal
			UIChans.StoppedChan <- struct{}{} // Note: This must come first.

			// Stop the bot if critical operation not happening
			// Check that we are not in a ledger/purchase/sale function first
			debugf("Current trading session (%s) has been stopped by the user.", bot.exchange)
			return true
		default:
			// do nothing
			return false
		}
	}
	channelsInitialized = true
}

// Run runs the main trading loop.
func (bot *Bot) Run(settings *Configuration) error {
	// setup
	if !channelsInitialized {
		return ErrChannelsNotInitialized
	}
	debug("Leprechaun is starting...")
	config = settings
	for {
		// Attempt to connect to the API and initialized clients for each assets.
		err := bot.initBot()
		if err != nil {
			// We could not connect to the luno API.
			// Probably due to a network error.
			if config.ExitOnInitFailed || err == ErrInvalidAPICredentials {
				// if the `config.ExitOnInitFailed` flag is set to true, Leprechaun will exit with an error.
				UIChans.ErrorChan <- ErrInvalidAPICredentials
				UIChans.StoppedChan <- struct{}{}
			} else {
				// We continue to try after a short wait until we connect.
				debug("Leprechaun will try to connect to the Luno API again after some time.")
				// shortSnooze()
				if cancelled() {
					return ErrCancelled
				}
				e := snooze(5) // Wait 5 minutes
				if e != nil {
					return e
				}
			}
		} else {
			// We have at least one client initialized. We can proceed with the trading loop.
			break
		}
	}
	initialRound = true
	var roundNo int = 1
	var signal SIGNAL
	for {
		// This is the main trading loop.
		for clientNo := 0; clientNo < len(bot.clients); clientNo++ {
			if cancelled() {
				return ErrCancelled
			}
			cl := bot.clients[clientNo]
			debugf("<========[ %s | Trading Round: %d ]========>", cl.name, roundNo)

			feeInfo, err := cl.FeeInfo()
			if err != nil {
				debugf("Error! Could not retrieve fee info for %s. %v", cl.Pair, err)
				continue
			}

			if cancelled() {
				return ErrCancelled
			}

			takerFee, _ := strconv.ParseFloat(feeInfo.TakerFee, 64)
			if initialRound {
				// Luno charges a taker fee for market orders.
				// we compensate for that by buying more than
				// the specified purchase Unit.
				debug(takerFee)
				debugf("30 day trading volume: %s %s. | Luno taker fee for %s is %.2f%s",
					feeInfo.ThirtyDayVolume, cl.asset, cl.name, takerFee*100, "%")

			}
			debugf("Your account balance is %.2f %s", cl.fiatBalance, cl.currency)
			currentPrice, err := cl.CurrentPrice()
			if err != nil {
				debugf("Could not retrieve price info for %s. Reason: %s", cl.name, err)
				if len(bot.clients) == 1 {
					if cancelled() {
						return ErrCancelled
					}
					// If we are only trading a  single asset. we should wait for some time.
					e := snooze(1) // Wait for a minute
					if e != nil {
						return e
					}
				}
				continue
			}

			if config.PurchaseUnit < (cl.minOrderVol * currentPrice) {
				debugf("The purchase amount you have specified %.2f can not purchase more than the minimum volume of %s that can be traded on the exchange (i.e %f %s)",
					config.PurchaseUnit, cl.name, cl.minOrderVol, cl.asset)
				UIChans.StoppedChan <- struct{}{}
				return ErrInvalidPurchaseUnit
			}
			debugf("The current price of %s(%s) is %s %.3f\n", cl.name, cl.asset, cl.currency, currentPrice)

			config.AdjustedPurchaseUnit = float64(config.PurchaseUnit) + (takerFee * float64(config.PurchaseUnit))
			canPurchase, err := cl.CheckBalanceSufficiency()
			if err != nil {
				log.Println(err)
			}
			debug("Leprechaun is analyzing market data...")
			signal, err = bot.Emit(&cl)
			debugf("Recommended action for %s based on market analysis: %v", cl.name, signal)
			if err != nil {
				debugf("Analysis for %s incomplete. Reason: %s. Will skip.", cl.name, err.Error())
				continue
			}
			if cancelled() {
				return ErrCancelled
			}
			// if signal == SignalBuy {
			if signal == SignalLong {
				// We can buy.
				if canPurchase {
					// Try to purchase `Client.asset`
					purchaseVolume := config.AdjustedPurchaseUnit / currentPrice
					record, err := cl.GoLong(purchaseVolume)
					// Quote Purchase
					// record, err := cl.PurchaseQuote()
					// if err != nil {
					// 	fmt.Println("Quote error", err)
					// 	continue
					// }

					if err != nil {
						debugf("An error occured while trying to purchase %.2f %s >> %s  ", purchaseVolume, cl.asset, err.Error())
					} else {
						updatedRecord, err := cl.UpdateOrderDetails(record)
						if err != nil {
							// N.B: User doesn't have to see this as they don't know
							// what's happening in this section.
							// Should be removed after testing is complete.
							debugf("Update failed: %v", err)
							// revert to our calulated values
							updatedRecord = record
						}
						// Save our purchase to the ledger.
						err = bot.addRecordToLedger(updatedRecord)
						if err != nil {
							debug("Error: ", err)
							e := errors.New("could not add record with id: " + record.ID + "to the ledger")
							UIChans.ErrorChan <- e
							return ErrCancelled
						}
						// Send an alert on the purchase channel
						UIChans.PurchaseChan <- struct{}{}
					}
					if cancelled() {
						return ErrCancelled
					}
				} else {
					// We don't have purchasing power.
					if cancelled() {
						return ErrCancelled
					}
					debugf("Leprechaun will not purchase any %s assets in this trading round as your balance (%s%.2f) is insufficent. Fund your account or specify a lower purchase unit.",
						cl.name, cl.currency, cl.fiatBalance)
				}

				// } else if signal == SignalSell {
			} else if signal == SignalShort {
				// We can sell
				var vol float64
				if cl.name == RippleCoin && bot.exchange == ExchangeLuno {
					// Luno only trades single units of ripple coin i.e no fractional or decimal units
					vol = math.Floor(config.AdjustedPurchaseUnit / currentPrice)
				} else {
					vol = config.AdjustedPurchaseUnit / currentPrice
				}
				volFormatted := strconv.FormatFloat(vol, 'f', -1, 64)
				purchaseVolume, _ := strconv.ParseFloat(volFormatted, 64)
				cl.GoShort(math.Abs(purchaseVolume))
				if cancelled() {
					return ErrCancelled
				}
			} else {
				// Market is indeterminate. Wait.
				debug("The ", cl.asset, " market is indeterminate at this time. Will not buy or sell.")
			}
			// TODO: Run ConfirmOrder in a goroutine
			if cancelled() {
				return ErrCancelled
			}
			// We try to complete any viable pending transaction in every round
			err = bot.CompleteLongTrades(&cl)
			if err != nil {
				debugf("An error occured while trying to cleanup pending long trades. Reason: %v", err)
			}
			err = bot.CompleteShortTrades(&cl)
			if err != nil {
				debugf("An error occured while trying to cleanup pending short trades. Reason: %v", err)
			}

			if cancelled() {
				return ErrCancelled
			}
		}
		initialRound = false
		if cancelled() {
			return ErrCancelled
		}
		err := Snooze()
		if err != nil {
			return err
		}
		roundNo++
	}
}

// NewBot create a new trading bot object
func NewBot() *Bot {
	bot = &Bot{
		name:     Leprechaun,
		exchange: ExchangeLuno,
		// id:       rand.Intn(1000),
		analysisPlugin: DefaultAnalysisPlugin(
			11,
			time.Duration(25)*time.Minute,
			config.Trade.TradingMode),
	}
	return bot
}

// SetAnalysisPlugin specifies the analyzer object used by the bot.
// It is expose for external use.
// The plugin object must satisfy the Analyzer interface.
func (bot *Bot) SetAnalysisPlugin(plugin Analyzer) {
	bot.analysisPlugin = plugin
}

// Channels returns a struct holding all chans for communicating with the bot.
func (bot *Bot) Channels() *Channels {
	return &Channels{}
}

// initBot initializes Leprechaun.
func (bot *Bot) initBot() error {
	debug("Initializing clients...")
	Assets := config.AssetsToTrade
	if len(Assets) < 1 {
		errStr := "Error! You have not specified any assets to trade. Please do so before starting the bot."
		debug(errStr)
		return ErrCancelled
	}
	for _, asset := range Assets {
		// ch := make(chan, int)
		// ch <- int
		client, err := initClient(asset)
		if err != nil {
			// Luno API rejected API key.
			if strings.Contains(err.Error(), "ErrAPIKeyNotFound") {
				debug("The API Key you have provided is invalid! Please check it and try again.")
				return ErrInvalidAPICredentials
			}
			// Could not connect to remote host.
			if strings.Contains(err.Error(), "no such host") || strings.Contains(err.Error(), "No address associated with hostname") {
				debug("Network error! Leprechaun could not connect to the Luno API. Please check your internet connection.\n")
				return err
			}
			if strings.Contains(err.Error(), "context deadline exceeded") {
				debug("Network Error! Your connection timed out. Please check your internet connection.")
				return err
			}
			debug("Could not initialize ", assetNames[asset], " client. Reason: ", err)
			return err
		}
		bot.clients = append(bot.clients, client)
	}
	clientnames := []string{}
	for _, c := range bot.clients {
		clientnames = append(clientnames, fmt.Sprintf("%s:%s", c.name, c.accountID))
	}
	debugf("%d client(s) initialized: <%v>", len(bot.clients), clientnames)
	return nil
}

// Shutdown stops the program.
func (bot *Bot) Shutdown() {
	UIChans.StoppedChan <- struct{}{}
}

// initClient creates a new client for a specifed asset
func initClient(asset string) (client Client, err error) {
	client.name = assetNames[asset]
	if asset != "XBT" && asset != "XRP" && asset != "ETH" && asset != "LTC" {
		Logger.Panicf("Error! Could not initialize client. Invalid asset (%s) specified", asset)
	}
	if len(config.APIKeyID) == 0 || len(config.APIKeySecret) == 0 {
		return client, ErrInvalidAPICredentials
	}
	client.asset = asset
	client.currency = "NGN"
	client.Pair = client.asset + client.currency // E.g. XBTNGN
	client.Client = luno.NewClient()
	client.Client.SetAuth(config.APIKeyID, config.APIKeySecret)
	if asset == "XRP" {
		client.minOrderVol = 1
	} else {
		client.minOrderVol = 0.0005
	}
	// retrieves balances and account ids
	_, err = client.AccountID()
	if err != nil {
		return
	}
	return
}

// CompleteShortTrades completes the second part of a long trade. The purchased asset is sold
// at the recorded trigger price
func (bot *Bot) CompleteShortTrades(cl *Client) error {
	ledger := bot.Ledger()
	defer ledger.Save()

	var (
		viablePendingRecords = []Record{}
	)
	// Get pending records.
	pendingRecords, err := ledger.GetRecordsByType(cl.asset, ShortOrder)
	if cancelled() {
		return ErrCancelled
	}
	if err != nil {
		// TODO: Silently print error and return
		debug(err)
		debug("There are no short trades awaiting completion in the ledger.")
	}

	if len(pendingRecords) > 0 {
		// Get current price
		currentPrice, err := cl.CurrentPrice()
		if cancelled() {
			return ErrCancelled
		}
		if err != nil {
			debugf("Error! (In `bot.CompleteLongTrades`) Could not retrieve current price. Reason: %v", err)
			return err
		}
		for _, rec := range pendingRecords {
			// Compare current asset price with the precalculated trigger price.
			if currentPrice > rec.TriggerPrice {
				// We can't repurchase yet. The repurchase price has to be lower or equal to the trigger price.
				// The trigger price is calculated when the short-sold asset is first sold.
				// For Example, if the asset was sold for #100,000 and the profit margin is 3%, the trigger price
				// is calculated to be 100,000 - (100,000 * 0.03) i.e. #97,000. The asset should be repurchased at
				// #97,000 or lower in order to realize a profit of #3000, i.e 3% of #100,000.0
				continue
			}
			// The current price is below or equal to the trigger price.
			viablePendingRecords = append(viablePendingRecords, rec)
		}
		recLen := len(viablePendingRecords)
		if recLen > 0 {
			// If there are viable assets up for repurchase them.
			debug("Found ", recLen, "short sold records viable for repurchase in the ledger")
			if cancelled() {
				return ErrCancelled
			}
			for n, rec := range viablePendingRecords {
				debugf("Trying to repurchase %d out of %d short sold %v assets\n", n+1, recLen, rec.Asset)
				orderID, err := cl.ask(currentPrice, rec.Volume)
				if err != nil {
					debugf("Error! (In `bot.CompleteLongTrades`) There was an error while selling %f %s", rec.Volume, rec.Asset)
				} else {
					debugf("%d out of %d viable assets with ID: %s bought.\n", n+1, recLen, orderID)
					debugf("Approx. profit realized is: %f", currentPrice-rec.Price)
					cl.lockedVolume -= rec.Volume // Subtract this asset from the locked volume for this client
					// record the sale of the asset
					err = NewPurchase(cl.asset, orderID, time.Now().Format(timeFormat),
						rec.Price, rec.Volume, currentPrice, rec.Volume)
					err = ledger.DeleteRecord(rec.ID)
					if err != nil {
						debugf("ERROR! Could not delete record with ID %s from the ledger.", rec.ID)
					}

				}
			}
		}

	} else {
		debug("There are no short trades awaiting completion in the ledger.")
	}
	return nil
}

// CompleteLongTrades completes the second part of a long trade. The purchased asset is sold
// at the recorded trigger price
func (bot *Bot) CompleteLongTrades(cl *Client) error {
	ledger := bot.Ledger()
	defer ledger.Save()

	var (
		viablePendingRecords = []Record{}
	)
	// Get pending records.
	pendingRecords, err := ledger.GetRecordsByType(cl.asset, LongOrder)
	if err != nil {
		// TODO: Silently print error and return
		debug(err)
		debug("There are no long trades awaiting completion in the ledger.")
	}
	if cancelled() {
		return ErrCancelled
	}
	if len(pendingRecords) > 0 {
		// Get current price
		currentPrice, err := cl.CurrentPrice()
		if err != nil {
			debugf("Error! (In `bot.CompleteLongTrades`) Could not retrieve current price. Reason: %v", err)
			return err
		}
		if cancelled() {
			return ErrCancelled
		}
		for _, rec := range pendingRecords {
			// Compare current asset price with the precalculated trigger price.
			if currentPrice < rec.TriggerPrice {
				// We can't sell the asset yet. The sale price has to be higher or equal to the trigger price.
				// The trigger price is calculated when the short-sold asset is first sold.
				// For Example, if the asset was bougth for #100,000 and the profit margin is 3%, the trigger price
				// is calculated to be 100,000 + (100,000 * 0.03) i.e. #103,000. The asset should be sold at
				// #103,000 or lower in order to realize a profit of #3000, i.e 3% of #100,000.0
				continue
			}
			// The current price is below or equal to the trigger price.
			viablePendingRecords = append(viablePendingRecords, rec)
		}
		recLen := len(viablePendingRecords)
		if recLen > 0 {
			// If there are viable assets up for repurchase them.
			debug("Found ", recLen, "short sold records viable for repurchase in the ledger")

			for n, rec := range viablePendingRecords {
				if cancelled() {
					return ErrCancelled
				}
				debugf("Trying to repurchase %d out of %d short sold %v assets\n", n+1, recLen, rec.Asset)
				orderID, err := cl.bid(currentPrice, rec.Volume)
				if err != nil {
					debugf("Error! (In `bot.CompleteLongTrades`) There was an error while selling %f %s", rec.Volume, rec.Asset)
				} else {
					debugf("%d out of %d viable assets with ID: %s bought.\n", n+1, recLen, orderID)
					debugf("Approx. profit realized is: %f", currentPrice-rec.Price)
					// cl.lockedBalance -= rec.Price // Subtract this asset from the locked balance for this client
					// record the sale of the asset
					err = NewSale(cl.asset, orderID, time.Now().Format(timeFormat),
						rec.Price, rec.Volume, currentPrice, rec.Volume)
					err = ledger.DeleteRecord(rec.ID)
					if err != nil {
						debugf("ERROR! Could not delete record with ID %s from the ledger.", rec.ID)
					}

				}
			}
		}

	} else {
		debug("There are no long trades awaiting completion in the ledger.")
	}
	return nil
}

// // TODO:: This function should be a goroutine
// func (bot *Bot) sellViableAssets(cl *Client, price float64) {
// 	// First check if there are any viable assets in the ledger for sale.
// 	debugf(`Leprechaun is checking the ledger for viable %s records...`, cl.name)
// 	ledger := bot.Lesdger()
// 	defer ledger.Save()
// 	asset := cl.asset
// 	viableRecords, err := ledger.ViableRecords(asset, price)
// 	if err != nil {
// 		// TODO: Silently print error and return
// 		debug(err)
// 		debug("There are no records in the ledger yet.")
// 	}
// 	recLen := len(viableRecords)
// 	if recLen > 0 {
// 		// If there are viable assets up for sale, sell them.
// 		debug("Found ", recLen, "records viable for sale in the ledger")
// 		for n, rec := range viableRecords {
// 			debugf("Trying to sell %d out of %d viable %v assets\n", n+1, recLen, rec.Asset)
// 			sold := cl.Sell(&rec)
// 			if sold {
// 				debugf("%d out of %d viable assets sold.\n", n+1, recLen)
// 			}
// 		}
// 	} else {
// 		debugf(`Leprechaun will not sell any assets, as there are no viable %s records in the ledger at this time.`, cl.name)
// 	}
// 	return
// }

func (bot *Bot) addRecordToLedger(rec Record) (err error) {
	ledger := bot.Ledger()
	defer ledger.Save()
	err = ledger.AddRecord(rec)
	return
}

// Emit runs the technical analysis pipeline and returns the
// signal emited by the analysis plugin
func (bot *Bot) Emit(cl *Client) (signal SIGNAL, err error) {
	retries := 3
	var (
		prices    []float64
		pricesErr error
	)
	for errCount := 0; errCount < retries; errCount++ {
		prices, pricesErr = cl.PreviousPrices(bot.analysisPlugin.PriceDimensions())
		if cancelled() {
			return SignalWait, ErrCancelled
		}
		if len(prices) > 0 && pricesErr == nil {
			break
		}
	}
	if pricesErr != nil || len(prices) == 0 {
		debug("An error occured while retrieving price data from the exchange. Please check your network connection!", err.Error())
		return SignalWait, err
	}

	// Do analysis
	log.Println(prices)
	reducedPrices := []float64{}
	if cl.name == RippleCoin {
		// Luno does not support trading ripple coin in fractional units
		for _, price := range prices {
			reducedPrices = append(reducedPrices, math.Floor(price))
		}
		prices = reducedPrices
		log.Println(reducedPrices)
	}
	if cancelled() {
		return SignalWait, ErrCancelled
	}
	err = bot.analysisPlugin.Analyze(prices)
	if err != nil {
		debugf("Analysis incomplete, due to error: (%v)", err)
		return SignalWait, err
	}
	// Emit the signal
	return bot.analysisPlugin.Emit(), nil
}
