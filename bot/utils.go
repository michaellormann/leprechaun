package leprechaun

/* This file is part of Leprechaun.
*  @author: Michael Lormann
 */

import (
	"fmt"
	"math/rand"
	"os"
	"time"
)

// Channels for communicating with the UI.
type Channels struct {
	// Log sends messages of Leprechaun's activities from the bot to the UI.
	LogChan chan string
	// Cancel delivers a signal for Leprechaun to stop running from the user to the bot through the UI.
	CancelChan chan struct{}
	// Stopped sends a signal from the bot to the UI after it has stopped.
	// To facilitate graceful exiting.
	StoppedChan chan struct{}
	// Error is for sending bot-side errors to the UI.
	ErrorChan chan error
	// PurchaseChan channel notifies the UI that a purchase has been made so it can update its displayed records
	PurchaseChan chan struct{}
	// SaleChan channel notifies the UI that a sale has been made so it can update its displayed records.
	SaleChan chan struct{}
}

// Log sets the log channel
func (c *Channels) Log(channel chan string) {
	c.LogChan = channel
}

// Cancel sets the channel with which to stop the bot
func (c *Channels) Cancel(channel chan struct{}) {
	c.CancelChan = channel
}

// BotStopped set the cahnnel through which the bot informs the UI it has stopped.
func (c *Channels) BotStopped(channel chan struct{}) {
	c.StoppedChan = channel
}

// Error sets the channel through which the bot sends error messages to the UI.
func (c *Channels) Error(channel chan error) {
	c.ErrorChan = channel
}

// Purchase sets the channel through which the bot alerts the UI to new Purchase events.
func (c *Channels) Purchase(channel chan struct{}) {
	c.PurchaseChan = channel
}

// Sale sets the channel through which the bot alerts the UI to new Sale events.
func (c *Channels) Sale(channel chan struct{}) {
	c.SaleChan = channel
}

// TODO: After testing debug should be changed to bot.log() function
func debug(v ...interface{}) {
	// write to stdout
	// log.Println(v...)

	// Send log message to UI over channel
	if config.Verbose {
		time := time.Now().Format("15:04:05")
		logChannel <- time + " " + fmt.Sprint(v...)
	}

	// write to the log file
	Logger.Print(v...)
}

func debugf(format string, v ...interface{}) {
	// write to stdout
	// log.Printf(format, v...)

	// Send log message to UI over channel
	if config.Verbose {
		time := time.Now().Format("15:04:05")
		logChannel <- time + " " + fmt.Sprintf(format, v...)
	}

	// write to log file
	Logger.Printf(format, v...)
}

// Snooze pauses Leprechaun's main loop for some time between each trading round
func Snooze() error {
	var minutes int32
	if config.RandomSnooze {
		snoozeIntervals := config.SnoozeTimes
		rand.Seed(time.Now().Unix())
		rand.Shuffle(len(snoozeIntervals), func(i int, j int) {
			snoozeIntervals[i], snoozeIntervals[j] = snoozeIntervals[j], snoozeIntervals[i]
		})
		minutes = snoozeIntervals[rand.Intn(len(snoozeIntervals))]
	} else {
		minutes = config.SnoozePeriod
	}
	err := snooze(minutes)
	if err != nil {
		// debugf("error: %s occured while snoozing", err)
		return err
	}
	return nil
}

func snooze(mins int32) error {
	minutes := time.Duration(mins)
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	snoozeEnd := time.NewTimer(minutes * time.Minute)
	defer snoozeEnd.Stop()
	// debugf("Leprechaun is snoozing for %d minutes\n", minutes)
	debugf("Snoozing...")
	for {
		select {
		case <-tick.C:
			// check if user has stopped the bot every 5 seconds
			if cancelled() {
				return ErrCancelled
			}
		case <-snoozeEnd.C:
			// Snooze period has elapsed.
			return nil
		default:
			// do nothing
			// time.Sleep(2)
		}
	}
}

func shorterSnooze() {
	minutes := 1
	debugf("Leprechaun is snoozing for %d minute\n", minutes)
	time.Sleep(time.Duration(minutes) * time.Minute)
}

// exists returns true if `path` exists, otherwise false.
func exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true

}
