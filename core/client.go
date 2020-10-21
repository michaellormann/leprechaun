package core

/* This file is part of Leprechaun.
*  @author: Michael Lormann
 */

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	luno "github.com/luno/luno-go"
	luno_decimal "github.com/luno/luno-go/decimal"
)

const (
	complete = luno.OrderStateComplete
	pending  = luno.OrderStatePending
)

var (
	assetNames = map[string]string{"XBT": "Bitcoin", "XRP": "Ripple Coin",
		"BCH": "Bitcoin Cash", "ETH": "Ethereum", "LTC": "Litecoin"}
	stringToIntDict = map[rune]int64{'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6,
		'7': 7, '8': 8, '9': 9}
	ctx = context.Background()
)

// OrderType indicates whether an order is long or short
type OrderType uint

const (
	// LongOrder is an order type in which an asset is bought at a certain price so it can be sold at a higher price.
	LongOrder OrderType = iota
	// ShortOrder is an order type in which an asset is sold in order to purchased at an even lower price.
	ShortOrder
)

// Custom errors
var (
	// ErrInsufficientBalance tells the user his fiat balance is low or below specfied purchase unit.
	ErrInsufficientBalance = errors.New("your fiat balance is insufficient")
	// ErrNetworkFailed represents a generic network error.
	ErrNetworkFailed = errors.New("network error. check your connection status")
	// ErrConnectionTimeout represents a connection timeout
	ErrConnectionTimeout = errors.New("your internet connection timed out")
	// PurchaseError        = errors.New("Could not complete purchase")
	// SaleError            = errors.New("Could not complete sale")
)

// Client handles all operations for a specified currency pair.
// It extends `luno.Client`
// TODO (Michael): Save locked volume/balance to file and load the stored values on every init.
type Client struct {
	Pair string
	// The client inherits all methods of `*luno.Client`
	*luno.Client
	name          string
	accountID     string
	fiatAccountID string
	assetBalance  float64
	fiatBalance   float64
	lockedBalance float64 // Fiat balance locked to complete short orders
	lockedVolume  float64 // Volume of asset locked to complete long orders
	asset         string
	currency      string
	minOrderVol   float64 // Minimum volume that can be traded on the exchange
}

// Record holds details of an asset sale or purchase
// `Asset` is the crypto asset in question. Assets are represented by three-leter code, like {`"XBT"`, `"ETH"`, `"XRP"`}
//
// `Cost` is simply the volume of the asset purchased mulitplied by its unit price. For example, a sale of
// 0.1 ETH at a unit price of #200,000 has a cost of #20,000
//
// `ID` is a string unique to each order laced on the exchange.
//
// `Price` is the market price at which the order was placed on the exchange. Leprechaun does not currently
// support limit orders.
//
// `SaleID` is the order ID for a sell order placed on the exchange.
//
// `Sold` identifies an asset in the legder that has been sold. (Redundant?)
//
// `Status` is a string specifying the status of an order on the luno exchange.
// See the Luno API docs for more info.
//
// `Timestamp` is a client-side string representation of the time at which an order is placed. This may be
// sligthly different from the time recorded by the exchange.
//
// `Volume` is the amount of the asset to be purchased or sold.
//
// `OrderType` indicates the type of order this recor holds. Orders are executed in two parts. For long
// orders, a purchase is made before a subsequent sale at a higher price, while for a short order, a sale
// is made before a subsequent purchase at a lower price.
//
// `TriggerPrice` specifies the pre-calculated price at whochh the second part of the order is executed.
// For a long order, this is the price at which to sell the asset and is always higher than the purchase price.
// For a short order, this is the price at whoch to buy the asset and is always lower than the purchase price.
type Record struct {
	Asset        string
	Cost         float64
	ID           string
	Price        float64
	SaleID       string
	Sold         bool
	Status       string
	Timestamp    string
	Volume       float64
	Type         OrderType
	TriggerPrice float64

	// Update legder code first to reflect new struct fields.
	// PPercent  float64 // Profit Percentage
}

// For string representation of a record. verbose fields are left out
type reprRecord struct {
	Asset     string
	Cost      float64
	ID        string
	Price     float64
	Timestamp string
	Volume    float64
}

// NewRecord creates a new `Record` object
func NewRecord(asset string, price float64, timestamp string,
	volume float64, id string, orderType OrderType) (rec Record) {
	rec.Asset = asset
	rec.Cost = price * volume
	rec.Price = price
	rec.ID = id
	rec.SaleID = ""
	rec.Sold = false
	rec.Status = ""
	rec.Timestamp = timestamp
	rec.Volume = volume
	rec.Type = orderType
	if rec.Type == LongOrder {
		rec.TriggerPrice = rec.Price + (rec.Price * config.ProfitMargin)
	} else if rec.Type == ShortOrder {
		rec.TriggerPrice = rec.Price - (rec.Price * config.ProfitMargin)
	}
	return
}

func (rec Record) String() string {
	s := reprRecord{Asset: rec.Asset, Cost: rec.Cost, ID: rec.ID, Price: rec.Price, Timestamp: rec.Timestamp,
		Volume: rec.Volume}
	return fmt.Sprintf("%+v", s)
}

// ProfitEntry records the profit made from the sale/(re)purchase of an asset
type ProfitEntry struct {
	Asset          string
	OrderID        string
	Timestamp      string
	PurchasePrice  float64
	PurchaseVolume float64
	PurchaseCost   float64
	SalePrice      float64
	SaleVolume     float64
	SaleCost       float64
	Profit         float64
}
type reprProfitEntry struct {
	Asset      string
	ID         string
	Timestamp  string
	SalePrice  float64
	SaleVolume float64
	SaleCost   float64
	Profit     float64
}

func (entry *ProfitEntry) String() string {
	e := reprProfitEntry{Asset: entry.Asset, ID: entry.OrderID, Timestamp: entry.Timestamp, SalePrice: entry.SalePrice,
		SaleVolume: entry.SaleVolume, SaleCost: entry.SaleCost, Profit: entry.Profit}
	return fmt.Sprintf("%+v", e)
}

// Bot is our trading bot
type Bot struct {
	name           string
	clients        []Client
	exchange       string
	sessionLength  time.Duration
	id             int
	cancel         context.CancelFunc
	chans          *Channels
	analysisPlugin Analyzer
}

// PurchaseQuote buys an asset using the qoute technique
func (cl *Client) PurchaseQuote() (rec Record, err error) {
	// TODO: Make sure quote isn't expired b4 exercising it.
	// If expired recreate it by using `continue` to go to the topof the loop
	// break out of the loop.
	fmt.Println("In Purchase Quote")
	price, err := cl.CurrentPrice()
	if err != nil {
		return
	}
	purchaseUnit := config.AdjustedPurchaseUnit - (0.0099 * config.AdjustedPurchaseUnit)
	volume := purchaseUnit / price
	trimmedVolume := strconv.FormatFloat(volume, 'f', 5, 64)
	volume, err = strconv.ParseFloat(trimmedVolume, 64)
	if err != nil {
		fmt.Println("CONV ERROR: ", err)
	}
	fmt.Println("Price:", price, "Purchase unit", purchaseUnit, "Volume", volume)
	quote := luno.CreateQuoteRequest{Type: "BUY", BaseAmount: decimal(volume), Pair: cl.Pair,
		BaseAccountId: stringToInt(cl.accountID), CounterAccountId: stringToInt(cl.fiatAccountID)}
	res, err := cl.CreateQuote(ctx, &quote)
	if err != nil {
		fmt.Println("Quote request error: ", err)
		return
	}
	fmt.Println("Quote Created:")
	fmt.Printf("QQ:: %#v\n", res)
	fmt.Println("Quote ID:", res.Id)
	expiry := &res.ExpiresAt
	fmt.Println("Quote Expiry:", expiry.String())
	baseAmount, counterAmount := &res.BaseAmount, &res.CounterAmount
	fmt.Printf("Bought %s for %s\n", baseAmount.String(), counterAmount.String())
	exerciseQuote := luno.ExerciseQuoteRequest{Id: stringToInt(res.Id)}
	exercisedQuote, err := cl.ExerciseQuote(ctx, &exerciseQuote)
	rec = NewRecord(cl.asset, exercisedQuote.CounterAmount.Float64(), time.Now().String(),
		exercisedQuote.BaseAmount.Float64(), exercisedQuote.Id, LongOrder)
	return
}

// SellQuote sells an asset using the quote technique.
func (cl *Client) SellQuote() {
	return
}

// bid places an order to buys a specified amount of an asset on the exchange
// It executes immediately.
func (cl *Client) bid(price float64, volume float64) (orderID string, err error) {
	sleep() // Error 429 safety
	cost := price * volume
	debugf("Placing bid order for NGN %.2f worth of %s (approx. %.2f %s) on the exchange...\n", cost, cl.name, volume, cl.asset)
	//Place bid order on the exchange
	req := luno.PostMarketOrderRequest{Pair: cl.Pair, Type: luno.OrderTypeBuy,
		BaseAccountId: stringToInt(cl.accountID), CounterAccountId: stringToInt(cl.fiatAccountID),
		CounterVolume: decimal(cost)}
	res, err := cl.PostMarketOrder(ctx, &req)
	orderID = res.OrderId
	if err != nil {
		return
	}
	debugf("Bid order for %.4f %s has been placed on the exchange.\n", volume, cl.asset)
	return

}

// ask places a bid order on the excahnge to sell `volume` worth of Client.asset in exhange for fiat currency.
func (cl *Client) ask(price, volume float64) (orderID string, err error) {
	// Todo: Change return types for this function
	sleep() // Error 429 safety
	cost := price * volume
	//Place ask order on the exchange
	debugf("Placing ask order for ~NGN %.2f worth of %s on the exchange...\n", cost, cl.name)
	debugf("Current price is %4f\n", price)
	req := luno.PostMarketOrderRequest{Pair: cl.Pair, Type: luno.OrderTypeSell,
		BaseAccountId: stringToInt(cl.accountID), BaseVolume: decimal(volume),
		CounterAccountId: stringToInt(cl.fiatAccountID)}
	res, err := cl.PostMarketOrder(ctx, &req)
	if err != nil {
		debugf("(in `Client.ask`) %v", err.Error())
		return
	}
	orderID = res.OrderId
	debugf("Ask order for %.4f %s has been placed on the exchange.\n", volume, cl.asset)
	return
}

// GoLong buys an asset at a specific price with the intention that the asset will
// later be sold at a higher price to realize a profit.
func (cl *Client) GoLong(volume float64) (rec Record, err error) {
	// goLong
	price, err := cl.CurrentPrice()
	if err != nil {
		return Record{}, err
	}
	ts := time.Now().Format(timeFormat)
	// Place market bid order.
	purchaseOrderID, err := cl.bid(price, volume)
	if err != nil {
		debugf("An error occured while going long!")
		return Record{}, err
	}

	debug("Order ID:", purchaseOrderID)
	cl.lockedVolume += volume

	return NewRecord(cl.asset, price, ts, volume, purchaseOrderID, LongOrder), nil
}

// GoShort sells an asset at a certain price with the aim of repurchasing the same
// volume of asset sold at a lower price in the future to realize a profit.
// TODO XXX: Implement stoploss for short sold assets
// TODO: Make short-selling an  option
func (cl *Client) GoShort(volume float64) (rec Record, err error) {
	// goShort
	price, err := cl.CurrentPrice()
	if err != nil {
		debug("Could not retrieve price info from the exchange. (in `Client.GoShort`)")
		return Record{}, err
	}
	ts := time.Now().Format(timeFormat)
	saleOrderID, err := cl.ask(price, volume)
	if err != nil {
		debug("An error occured while executing a short order!")
		return Record{}, err
	}
	cost := price * volume
	cl.lockedBalance += cost
	debug("Order ID:", saleOrderID)

	return NewRecord(cl.asset, price, ts, volume, saleOrderID, ShortOrder), nil
}

// Returns a string representation of a Client struct
func (cl Client) String() (s string) {
	s = assetNames[cl.Pair[:3]] + "client. ID: " + string(cl.accountID)
	return
}

// ConfirmOrder checks if an order placed on the exchange has been executed
func (cl *Client) ConfirmOrder(rec *Record) {
	// Make this method a goroutine
	if rec.Sold {
		sleep() // Error 429 safety
		req := luno.GetOrderRequest{Id: rec.SaleID}
		res, err := cl.GetOrder(ctx, &req)
		if err != nil {
			debug("Error! Could not confirm order: ", rec.SaleID)
			debug("Please check your network connectivity")
			debug(err.Error())
		}
		rec.Status = string(res.State)
		// Note other details of the response object should be used to update sale history and calculate profit.
		// Should be implemented by update_ledger function.
	}
	return
}

func (cl *Client) retrieveBalances() (err error) {
	sleep() // Error 429 safety
	assetBalanceReq := luno.GetBalancesRequest{Assets: []string{cl.asset}}
	assetBalance, err := cl.GetBalances(ctx, &assetBalanceReq)
	if err != nil {
		return err
	}
	// debugf("%#v \n", assetBalance)
	// debugln(fiatBalance)
	if assetBalance != nil && len(assetBalance.Balance) > 0 {
		for _, astBal := range assetBalance.Balance {
			if astBal.Asset == cl.asset {
				cl.accountID = astBal.AccountId
				cl.assetBalance = astBal.Balance.Float64()
			}
			if astBal.Asset == cl.currency {
				cl.fiatAccountID = astBal.AccountId
				cl.fiatBalance = astBal.Balance.Float64()
			}
		}
	}
	err = nil
	return
}

// CheckBalanceSufficiency determines whether the client has purchasing power
func (cl *Client) CheckBalanceSufficiency() (canPurchase bool, err error) {
	// Luno charges a 1% taker fee
	purchaseUnit := config.AdjustedPurchaseUnit
	if cl.fiatBalance <= 0.0 {
		cl.retrieveBalances()
	}
	if cl.fiatBalance < purchaseUnit {
		// `AdjustedPurchaseUnit` is more than available balance (NGN)
		canPurchase = false
		err = ErrInsufficientBalance
	} else {
		canPurchase = true
	}
	return
}

// StopPendingOrder tries to remove a pending order from the order book
func (cl *Client) StopPendingOrder(orderID string) (ok bool) {
	sleep() // Error 429 safety
	req := luno.StopOrderRequest{OrderId: orderID}
	res, err := cl.StopOrder(ctx, &req)
	if err != nil {
		debug(err)
		return false
	}
	if res.Success {
		return true
	}
	return
}

// CheckOrder tries to confirm if an order is still pending or not
func (cl *Client) CheckOrder(orderID string) (orderDetails luno.GetOrderResponse, err error) {
	sleep() // Error 429 safety
	req := luno.GetOrderRequest{Id: orderID}
	res, err := cl.GetOrder(ctx, &req)
	if err != nil {
		// debug(err)
		return orderDetails, err
	}
	orderDetails = *res
	return
}

// UpdateOrderDetails updates order details
func (cl *Client) UpdateOrderDetails(rec Record) (updated Record, err error) {
	orderDetails, err := cl.CheckOrder(rec.ID)
	if err != nil {
		// return record unchanged
		return rec, err
	}
	oldRec := rec
	rec.Price = orderDetails.LimitPrice.Float64()
	rec.Cost = orderDetails.FeeCounter.Float64()
	rec.Volume = orderDetails.FeeBase.Float64()
	rec.Timestamp = orderDetails.CompletedTimestamp.String()
	fmt.Println("Record updated from: ")
	fmt.Printf("%#v\n", oldRec)
	fmt.Println("To:")
	fmt.Printf("%#v\n", rec)
	return
}

// CurrentPrice retrieves the ask price for the client's asset.
func (cl *Client) CurrentPrice() (price float64, err error) {
	sleep() // Error 429 safety
	req := luno.GetTickerRequest{Pair: cl.Pair}
	res, err := cl.GetTicker(ctx, &req)
	if err != nil {
		return
	}
	price = res.Ask.Float64()
	return
}

// AccountID returns map[string]string{"asset":asset_account_id, "fiat":fiat_accont_id}
func (cl *Client) AccountID() (ID map[string]string, err error) {
	err = cl.retrieveBalances()
	if err != nil {
		return ID, err
	}
	ID = make(map[string]string)
	ID["asset"] = cl.accountID
	ID["fiat"] = cl.fiatAccountID
	return
}

// PreviousPrices retrieves historic price data from the exchange.
// The price is determined from executed trades `interval` minutes apart,
// parameter `num` specifies the number of prices to be retrieved.
// For example: num=10, interval=5 gets prices over the last 50 minutes.
func (cl *Client) PreviousPrices(num int, interval time.Duration) (prices []float64, err error) {

	timestamps := []luno.Time{}
	allTrades := map[luno.Time]luno.Trade{}
	// Oldest first
	for i := num; i > 0; i-- {
		// -----------Currently using short interval for testing--------------
		timestamps = append(timestamps, luno.Time(time.Now().Add(time.Duration(-i)*interval)))
		// luno.Time(time.Now().Add(-24 * time.Hour))
	}
	var lastTrade luno.Trade
	for _, timestamp := range timestamps {
		sleep2() // Error 429 safety
		req := luno.ListTradesRequest{Pair: cl.Pair, Since: timestamp}
		res, err := cl.ListTrades(ctx, &req)
		if err != nil {
			return []float64{}, ErrNetworkFailed
		}
		noTrades := len(res.Trades)
		if noTrades > 0 {
			// Drop all trades but the latest one

			// Use all trades instead
			lastTrade = res.Trades[noTrades-1]
			for _, trade := range res.Trades {
				allTrades[trade.Timestamp] = trade
			}
			// allTrades[timestamp] = lastTrade
			// allTrades[timestamp] = res.Trades[0]
		} else {
			// No trades were placed before the time period specified
			allTrades[timestamp] = lastTrade
		}
	}
	// First append the current price to the list.
	currentPrice, err := cl.CurrentPrice()
	if err != nil {
		return prices, err
	}
	prices = append(prices, currentPrice)

	for _, trade := range allTrades {
		prices = append(prices, trade.Price.Float64())
	}
	// fmt.Println("Exchange prices for", cl.name, ":", prices)
	return
}

// FeeInfo retrieves taker/maker fee information for this client
func (cl *Client) FeeInfo() (info luno.GetFeeInfoResponse, err error) {
	sleep() // Error 429 safety
	req := luno.GetFeeInfoRequest{Pair: cl.Pair}
	res, err := cl.GetFeeInfo(ctx, &req)
	if err != nil {
		return
	}
	info = *res
	return
}

// TopOrders retrieves the top ask and bid orders on the exchange
func (cl *Client) TopOrders() (orders map[string]luno.OrderBookEntry) {
	sleep() // Error 429 safety
	req := luno.GetOrderBookRequest{Pair: cl.Pair}
	orderBook, err := cl.GetOrderBook(ctx, &req)
	if err != nil {
		debug(err)
	}
	topAsk := orderBook.Asks[0]
	topBid := orderBook.Bids[0]
	orders["ask"] = topAsk
	orders["bid"] = topBid
	return
}

// PendingOrders retrieves unexecuted orders still in the order book.
func (cl *Client) PendingOrders() (pendingOrders interface{}) {
	sleep() // Error 429 safety
	accID := stringToInt(cl.fiatAccountID)
	req := luno.ListPendingTransactionsRequest{Id: accID}
	res, err := cl.ListPendingTransactions(ctx, &req)
	if err != nil {
		debug(err)
	}
	pending := res.Pending
	numPending := len(pending)
	if numPending == 0 {
		debug("There are no pending transactions associated with", cl)
		pendingOrders = []string{}
	}
	debug("There are", numPending, "transactions associated with", cl)

	pendingOrders = pending
	return
}

// sleep delays the bot between each request in order to avoid exceeding the rate limit.
func sleep() {
	time.Sleep(650 * time.Millisecond)
}

// sleep2 delays the bot for slightly longer than sleep. Sometimes sleep still triggers Error 429.
func sleep2() {
	time.Sleep(800 * time.Millisecond)
}

// stringToInt converts a string of numbers to its numerical value
// without loss of precision or conversion errors up until math.MaxInt64
func stringToInt(s string) (num int64) {
	for i, v := range s {
		n := stringToIntDict[v]
		x := len(s) - i
		c := math.Pow(1e1, float64(x-1))
		num += int64(n) * int64(c)
	}
	return
}

// Decimal converts a float64 value to a Decimal representation of scale 10
func decimal(val float64) (dec luno_decimal.Decimal) {
	dec = luno_decimal.NewFromFloat64(val, 10)
	return
}
