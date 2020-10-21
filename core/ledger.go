package core

/* This file is part of Leprechaun.
*  @author: Michael Lormann
 */
import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"database/sql"
	// go-sqlite3 is imported for its side-effect of loading the sqlite3 driver.
	_ "github.com/mattn/go-sqlite3"
)

// Ledger object stores records of purchased assets in a sql database.
type Ledger struct {
	databasePath string
	db           *sql.DB
	isOpen       bool
}

// SQLITE operations.
var (
	sqlDatabaseName        = "Leprechaun.Ledger"
	databaseInit    string = "CREATE TABLE RECORDS (ASSET, COST, ID, PRICE, SALE_ID, SOLD, STATUS, TIMESTAMP, VOLUME, TYPE, TRIGGER_PRICE)"
	recordInsert           = "INSERT INTO RECORDS VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	idSearch        string = "SELECT * FROM RECORDS WHERE ID = ?"
	// abs(PRICE) + abs(PRICE) * `margin` adjusts the price by profit margin provided.
	// E.g. to adjust a price of 2_000_000 by a 1% margin, we have 2_000_000 + 2_000_000 * 0.01 =
	// giving an adjusted price of 2_020_000
	viableRecordSearch = "SELECT * FROM RECORDS WHERE ASSET = ? AND abs(PRICE) + abs(PRICE) * ? < ?"
	getAllRecordsOp    = "SELECT * FROM RECORDS"
	typeSearchOp       = "SELECT * FROM RECORDS WHERE ASSET = ? AND TYPE = ?"
	deleteRecordOp     = "DELETE FROM RECORDS WHERE ID = ?"
)

// Ledger returns a new ledger handle
func (bot *Bot) Ledger() (l *Ledger) {
	l = &Ledger{databasePath: config.LedgerDatabase}
	l.loadDatabase()
	return
}

// ViableRecords checks the database for any records whose prices are lower
// (beyond a certain `margin`) than the value of `price`.
func (l *Ledger) ViableRecords(asset string, price float64) (records []Record, err error) {
	// TODO:: Include margin test in viable records check
	if !l.isOpen {
		l.loadDatabase()
	}
	tx, err := l.db.Begin()
	if err != nil {
		return
	}
	stmt, err := l.db.Prepare(viableRecordSearch)
	if err != nil {
		return
	}
	defer stmt.Close()
	margin := config.ProfitMargin
	rows, err := stmt.Query(asset, margin, price)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		rec := Record{}
		err = scanRows(rows, rec)
		if err != nil {
			return
		}
		records = append(records, rec)
	}
	tx.Commit()
	return
}

func scanRows(rows *sql.Rows, rec Record) (err error) {
	err = rows.Scan(&rec.Asset, &rec.Cost, &rec.ID, &rec.Price, &rec.SaleID, &rec.Sold, &rec.Status, &rec.Timestamp, &rec.Volume, &rec.Type, &rec.TriggerPrice)
	return err
}

// GetRecordByID returns a record from the database with the `id` provided.
func (l *Ledger) GetRecordByID(id string) (rec Record, err error) {
	rec = Record{}
	if !l.isOpen {
		l.loadDatabase()
	}
	tx, err := l.db.Begin()
	if err != nil {
		return
	}
	stmt, err := l.db.Prepare(idSearch)
	if err != nil {
		return
	}
	defer stmt.Close()
	err = stmt.QueryRow(id).Scan(&rec.Asset, &rec.Cost, &rec.ID, &rec.Price, &rec.SaleID, &rec.Sold, &rec.Status, &rec.Timestamp, &rec.Volume, &rec.Type, &rec.TriggerPrice)
	if err != nil {
		return
	}
	tx.Commit()
	debugf("%#v\n", rec)

	return
}

// DeleteRecord removes the record with the provided `ID` from the ledger.
func (l *Ledger) DeleteRecord(id string) (err error) {
	if !l.isOpen {
		l.loadDatabase()
	}
	tx, err := l.db.Begin()
	if err != nil {
		return
	}
	stmt, err := l.db.Prepare(deleteRecordOp)
	defer stmt.Close()
	if err != nil {
		return
	}
	res, err := stmt.Exec(id)
	if err != nil {
		return
	}
	log.Printf("delete op: %v for record with id %s", res, id)
	tx.Commit()
	return
}

// GetRecordsByType retrieves records in the ledger by order type
func (l *Ledger) GetRecordsByType(asset string, orderType OrderType) (records []Record, err error) {
	if !l.isOpen {
		l.loadDatabase()
	}
	tx, err := l.db.Begin()
	if err != nil {
		return
	}
	stmt, err := l.db.Prepare(typeSearchOp)
	defer stmt.Close()
	if err != nil {
		return
	}
	rows, err := stmt.Query(asset, orderType)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		rec := Record{}
		err = scanRows(rows, rec)
		if err != nil {
			return
		}
		records = append(records, rec)
	}
	tx.Commit()
	return
}

// AllRecords returns all purchase records stored in the ledger.
func (l *Ledger) AllRecords() (records []Record, err error) {
	if !l.isOpen {
		l.loadDatabase()
	}
	tx, err := l.db.Begin()
	if err != nil {
		return
	}
	stmt, err := l.db.Prepare(getAllRecordsOp)
	if err != nil {
		return
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		rec := Record{}
		err = scanRows(rows, rec)
		if err != nil {
			return
		}
		records = append(records, rec)
	}
	tx.Commit()
	return
}

// AddRecord adds a `Record` to the database.
func (l *Ledger) AddRecord(rec Record) (err error) {
	if !l.isOpen {
		l.loadDatabase()
	}
	debug("New Record: ", fmt.Sprintf("%+v", rec))
	tx, err := l.db.Begin()
	if err != nil {
		return
	}
	stmt, err := tx.Prepare(recordInsert)
	if err != nil {
		return
	}
	defer stmt.Close()
	_, err = stmt.Exec(rec.Asset, rec.Cost, rec.ID, rec.Price, rec.SaleID, rec.Sold, rec.Status, rec.Timestamp, rec.Volume, rec.Type, rec.Timestamp)
	if err != nil {
		// log.Fatal(err)
		debugf("Fatal error! could not add new record with id %s to the ledger. Check the luno order book for your order's details", rec.ID)
		return err
	}
	tx.Commit()
	return
}

// Save closese the database. Must be called by any external user of the ledger.
func (l *Ledger) Save() (err error) {
	if !l.isOpen {
		l.db.Close()
	}
	l.isOpen = false
	return
}

func (l *Ledger) loadDatabase() {
	dataDir := filepath.Dir(l.databasePath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		// log.Println("Data folder already exists.")
	}
	// first check if ledger db already exists
	alreadyExists := exists(l.databasePath)

	// open the database
	db, err := sql.Open("sqlite3", l.databasePath)
	if err != nil {
		log.Fatal(err)
	}
	if !alreadyExists {
		// We are just creating a new ledger
		_, err = db.Exec(databaseInit)
		if err != nil {
			Logger.Fatal("Could not initialize ledger database", err)
		}
	}
	l.db = db
	l.isOpen = true
	return
}

// NewSale saves a sale's profits to record
func NewSale(asset, orderID, timestamp string, purchasePrice, purchaseVolume, salePrice, saleVolume float64) error {
	entry := ProfitEntry{Asset: asset, OrderID: orderID, Timestamp: timestamp, PurchasePrice: purchasePrice, PurchaseVolume: purchaseVolume,
		SalePrice: salePrice, SaleVolume: saleVolume}
	// Collate all sales into a single all-time record
	entry.PurchaseCost = entry.PurchasePrice * entry.PurchaseVolume
	entry.SaleCost = entry.SalePrice * entry.SaleVolume
	entry.Profit = entry.SaleCost - entry.PurchaseCost
	debugf("Profit made from sale of %f %s is %f\n", entry.SaleVolume, assetNames[entry.Asset], entry.Profit)

	if !exists(config.DataDir) {
		os.MkdirAll(config.DataDir, 0755)
	}

	stats := fmt.Sprintf("%s-stats.json", strings.ToLower(assetNames[asset]))
	stats = filepath.Join(config.DataDir, stats)
	statsFile, err := os.OpenFile(stats, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		debug("Error! Could not open stats file!")
		return err
	}
	defer statsFile.Close()

	// Read previous stats from file
	previousEntry := ProfitEntry{}
	err = json.NewDecoder(statsFile).Decode(&previousEntry)
	if err != nil && err != io.EOF {
		debug("Error! Json Decode Err", err)
		return err
	}
	err = statsFile.Truncate(0)
	if err != nil {
		debug("Error! Could not truncate stats file", err)
		return err

	}
	if _, err := statsFile.Seek(0, 0); err != nil {
		debug("Seek Error:", err)
		return err
	}
	newEntry := entry
	newEntry.Profit += previousEntry.Profit
	newEntry.PurchaseCost += previousEntry.PurchaseCost
	newEntry.PurchasePrice += previousEntry.PurchasePrice
	newEntry.PurchaseVolume += previousEntry.PurchaseVolume
	newEntry.SaleCost += previousEntry.SaleCost
	newEntry.SalePrice += previousEntry.SalePrice
	newEntry.SaleVolume += previousEntry.SaleVolume
	err = json.NewEncoder(statsFile).Encode(newEntry)
	if err != nil {
		debug("Json Encode error", err)
	}

	// Sales record section
	sales := filepath.Join(config.DataDir, "sales.json")
	salesFile, err := os.OpenFile(sales, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer salesFile.Close()
	salesRecordStack := &ProfitRecordStack{} // We only want to save `maxRecordsToSave` records at most
	// Read previous records from file
	err = json.NewDecoder(salesFile).Decode(&salesRecordStack.records)
	if err != nil && err != io.EOF {
		return err
	}
	salesFile.Truncate(0)
	_, err = salesFile.Seek(0, 0)
	// Add the latest `ProfitEntry` to the stack
	salesRecordStack.appendRecord(&entry)
	// save the sale record to file
	err = json.NewEncoder(salesFile).Encode(&salesRecordStack.records)
	if err != nil {
		debug("Json sale encode error", err)
	}
	return nil
}

// GetSales returns historical records that have been saved to file.
func GetSales() (records []*ProfitEntry, err error) {
	salesFileLoc := filepath.Join(config.DataDir, "sales.json")
	salesFile, err := os.OpenFile(salesFileLoc, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer salesFile.Close()
	err = json.NewDecoder(salesFile).Decode(&records)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return

}

// AssetStats holds all time stats for an asset
type AssetStats struct {
	Asset                 string
	AllTimePurchaseVolume string
	AllTimeSalesVolume    string
	AllTimeSalesCost      string
	AllTimePurchasesCost  string
	AllTimeProfit         string
}

// GetStats returns the statistics for a given asset
func GetStats(asset string) (string, error) {
	d := AssetStats{}

	statsFile := fmt.Sprintf("%s-stats.json", strings.ToLower(assetNames[asset]))
	statsFile = filepath.Join(config.DataDir, statsFile)
	entryFile, err := os.OpenFile(statsFile, os.O_RDONLY, 0644)
	stats := ProfitEntry{}
	err = json.NewDecoder(entryFile).Decode(&stats)
	if err != nil && err != io.EOF {
		return "", err
	}
	d.Asset = " " + assetNames[asset] + "\n"
	if asset == "XBT" {
		stats.Asset = "BTC"
	}
	d.AllTimePurchaseVolume = fmt.Sprintf(" %s %s\n", strconv.FormatFloat(stats.PurchaseVolume, 'f', 2, 64),
		stats.Asset)
	d.AllTimeSalesVolume = fmt.Sprintf(" %s %s\n", strconv.FormatFloat(stats.SaleVolume, 'f', 2, 64),
		stats.Asset)
	d.AllTimePurchasesCost = fmt.Sprintf(" %s %s\n", strconv.FormatFloat(stats.PurchaseCost, 'f', 2, 64),
		config.CurrencyName)
	d.AllTimeSalesCost = fmt.Sprintf(" %s %s\n", strconv.FormatFloat(stats.SaleCost, 'f', 2, 64),
		config.CurrencyName)
	d.AllTimeProfit = fmt.Sprintf(" %s %s\n", strconv.FormatFloat(stats.Profit, 'f', 2, 64), config.CurrencyName)
	s := fmt.Sprintf("%+v\n", d)
	s = strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(s), "}"), "{")
	return s, nil

}

// NewPurchase adds a new (re)purchase record to file an calculates the profit made therein.
// Only `maxRecordsToSave` most recent records are saved to file. (see the `recordStack.append` function)
// func NewPurchase(purchase Record) error {
func NewPurchase(asset, orderID, timestamp string, salePrice, saleVolume, purchasePrice, purchaseVolume float64) error {
	entry := ProfitEntry{Asset: asset, OrderID: orderID, Timestamp: timestamp, PurchasePrice: purchasePrice, PurchaseVolume: purchaseVolume,
		SalePrice: salePrice, SaleVolume: saleVolume}
	// Collate all sales into a single all-time record
	entry.PurchaseCost = entry.PurchasePrice * entry.PurchaseVolume
	entry.SaleCost = entry.SalePrice * entry.SaleVolume
	entry.Profit = entry.PurchaseCost - entry.SaleCost // Note. This is the reverse of the sale profit calculation.
	debugf("Profit made from sale of %f %s is %f\n", entry.SaleVolume, assetNames[entry.Asset], entry.Profit)

	if !exists(config.DataDir) {
		os.MkdirAll(config.DataDir, 0755)
	}

	stats := fmt.Sprintf("%s-stats.json", strings.ToLower(assetNames[asset]))
	stats = filepath.Join(config.DataDir, stats)
	statsFile, err := os.OpenFile(stats, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		debug("Error! Could not open stats file!")
		return err
	}
	defer statsFile.Close()

	// Read previous stats from file
	previousEntry := ProfitEntry{}
	err = json.NewDecoder(statsFile).Decode(&previousEntry)
	if err != nil && err != io.EOF {
		debug("Error! Json Decode Err", err)
		return err
	}
	err = statsFile.Truncate(0)
	if err != nil {
		debug("Error! Could not truncate stats file", err)
		return err

	}
	if _, err := statsFile.Seek(0, 0); err != nil {
		debug("Seek Error:", err)
		return err
	}
	newEntry := entry
	newEntry.Profit += previousEntry.Profit
	newEntry.PurchaseCost += previousEntry.PurchaseCost
	newEntry.PurchasePrice += previousEntry.PurchasePrice
	newEntry.PurchaseVolume += previousEntry.PurchaseVolume
	newEntry.SaleCost += previousEntry.SaleCost
	newEntry.SalePrice += previousEntry.SalePrice
	newEntry.SaleVolume += previousEntry.SaleVolume
	err = json.NewEncoder(statsFile).Encode(newEntry)
	if err != nil {
		debug("Json Encode error", err)
	}

	// purchase record section
	purchasesFileLoc := filepath.Join(config.DataDir, "purchases.json")
	stack := &ProfitRecordStack{}
	purchasesFile, err := os.OpenFile(purchasesFileLoc, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer purchasesFile.Close()
	// First read the records from file into the stack
	err = json.NewDecoder(purchasesFile).Decode(&stack.records)
	if err != nil && err != io.EOF {
		return err
	}
	if err == io.EOF {
		// There was no previous entry in the file
		stack = &ProfitRecordStack{}
	}
	purchasesFile.Truncate(0)
	_, err = purchasesFile.Seek(0, 0)
	if err != nil {
		return err
	}
	stack.appendRecord(&entry)
	err = json.NewEncoder(purchasesFile).Encode(stack.records)
	if err != nil {
		return err
	}
	return nil

}

// GetPurchases returns a list of past asset purchases made by leprechaun.
func GetPurchases() ([]*ProfitEntry, error) {
	purchasesFileLoc := filepath.Join(config.DataDir, "purchases.json")
	stack := new(ProfitRecordStack)
	purchases := stack.records
	purchasesFile, err := os.OpenFile(purchasesFileLoc, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer purchasesFile.Close()
	// Read the records from file
	err = json.NewDecoder(purchasesFile).Decode(&purchases)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return purchases, nil

}

var (
	maxRecordsToSave int = 100
)

// RecordStack holds a FIFO stack of at most `maxRecordsToSave` `Record` elements.
type RecordStack struct {
	records []Record
}

// ProfitRecordStack holds a FIFO stack of at most `maxRecordsToSave` `ProfitEntry` elements.
type ProfitRecordStack struct {
	records []*ProfitEntry
}

// appendRecord appends a value of type T to a FIFO stack (actually a slice),
//  with a max capacity of `maxRecordsToSave`
// If the stacked is filled, it's first value is popped. Note, the slice
// shouldn't be created with make(), but initialized like so: stack := []T
func (st *RecordStack) appendRecord(rec Record) {
	lnt := len(st.records)
	if lnt >= maxRecordsToSave {
		x := lnt - maxRecordsToSave
		st.records = st.records[x+1 : lnt] // pop the first `x` elements
	}
	st.records = append(st.records, rec)
}

func (stack *ProfitRecordStack) appendRecord(entry *ProfitEntry) {
	l := len(stack.records)
	if l >= maxRecordsToSave {
		x := l - maxRecordsToSave
		stack.records = stack.records[x+1 : l] // pop the first element
	}
	stack.records = append(stack.records, entry)
}
