package main

import (
	"fmt"
	"runtime"

	"git.wow.st/gmp/jni"
	leprechaun "github.com/michaellormann/leprechaun/bot"

	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	ui "github.com/michaellormann/leprechaun/material"

	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"github.com/jrick/logrotate/rotator"

	"gioui.org/app"
	_ "gioui.org/app/permission/storage"
	"gioui.org/text"
	"gioui.org/widget/material"
)

// App ....
type App struct {
	name        string
	logRotators map[string]*rotator.Rotator
	logBackends map[string]*log.Logger
	subsystems  []string
	config      *leprechaun.Configuration
	// err         error
	fontFile  string
	dir       string
	cwd       string
	test      bool
	font      []text.FontFace
	theme     *material.Theme
	win       *ui.Window
	jvm       jni.JVM
	jCtx      jni.Object
	isAndroid bool
}

func newApp(test bool) *App {
	wd, _ := os.Getwd()
	isAndroid := false
	if runtime.GOOS == "android" {
		isAndroid = true
	}
	return &App{name: "Leprechaun",
		test:        test,
		logBackends: map[string]*log.Logger{},
		logRotators: map[string]*rotator.Rotator{},
		fontFile:    filepath.Join(wd, "/assets/fonts/source_sans_pro_semibold.otf"),
		config:      new(leprechaun.Configuration),
		cwd:         wd,
		subsystems: []string{
			"bot",
			"startup",
			// "ui"
		},
		isAndroid: isAndroid,
	}
}

func main() {
	myApp := newApp(true)

	d, err := app.DataDir()
	if err != nil {
		myApp.Errorln("could not find user data directory")
	}
	myApp.dir = d

	// create different log backends for the bot,
	// the startup (main.go) (and the ui)?
	for _, sys := range myApp.subsystems {
		myApp.InitLogBackends(sys)
	}
	// ensure they are closed when the app.Main function
	// returns
	defer myApp.CloseLogFiles()

	// load user settings from file
	myApp.LoadConfig()

	theme := myApp.Theme()
	myApp.win = ui.CreateWindow(theme, myApp.config)
	myApp.win.InitBackends(myApp.logBackends)

	go func() {
		if err := myApp.win.Loop(); err != nil {
			// TODO (michael): Send shutdown signal
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

// Load Fonts.
func (a *App) loadFont() (err error) {
	source, err := os.Open(a.fontFile)
	if err != nil {
		log.Printf("Failed to load font: %v", err)
		a.font = gofont.Collection()
	} else {
		stat, err := source.Stat()
		if err != nil {
			log.Println(err)
		}
		bytes := make([]byte, stat.Size())
		source.Read(bytes)
		fnt, err := opentype.Parse(bytes)
		if err != nil {
			log.Println(err)
		}
		a.font = append(a.font, text.FontFace{Font: text.Font{}, Face: fnt})
	}
	return nil
}

func (a *App) Theme() *material.Theme {
	err := a.loadFont()
	if err != nil || len(a.font) == 0 {
		a.font = gofont.Collection()
	}
	return material.NewTheme(a.font)
}

// InitLogBackends creates a unique log rotator object
// for the provided subsystem.
func (a *App) InitLogBackends(subsystem string) {
	a.config.LogDir = filepath.Join(a.dir, "Leprechaun", "logs", "bot")
	logFile := filepath.Join(a.dir, "Leprechaun", "logs", subsystem, "log.txt")
	prefix := fmt.Sprintf("Leprechaun %s - ", strings.ToTitle(subsystem))
	// initialize the log rotator
	a.InitLogRotator(logFile, subsystem, 30) // backup logs for 30 days.
	logBackend := &log.Logger{}
	logBackend.SetFlags(log.LstdFlags)
	logBackend.SetOutput(a.logRotators[subsystem])
	logBackend.SetPrefix(prefix)
	logFile, _ = filepath.Abs(logFile)
	log.Printf(`%s log file==> "%s"`, subsystem, logFile)
	a.logBackends[subsystem] = logBackend
	return
}

// CloseLogFiles ensures that all subsytem loggers are
// closed before the program exits.
func (a *App) CloseLogFiles() {
	var err error
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("internal error: %v", p)
			log.Println(err)
		}
	}()
	for _, r := range a.logRotators {
		r.Close()
	}
}

// InitLogRotator creates a log rotator for each subsystem.
func (a *App) InitLogRotator(logFile, subsytem string, maxRolls int) {
	logDir, _ := filepath.Split(logFile)
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		_ = os.MkdirAll(logDir, 0755)
	}
	r, err := rotator.New(logFile, 512, false, maxRolls)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create file rotator: %v\n", err)
		os.Exit(1)
	}
	a.logRotators[subsytem] = r
}

func (a *App) callVoidMethod(obj jni.Object, name, sig string, args ...jni.Value) error {
	if obj == 0 {
		panic("invalid object")
	}
	return jni.Do(a.jvm, func(env jni.Env) error {
		cls := jni.GetObjectClass(env, obj)
		m := jni.GetMethodID(env, cls, name, sig)
		return jni.CallVoidMethod(env, obj, m, args...)
	})
}

// LoadConfig loads saved user settings from file. Default settings
// are used if no settings are found on file.
// If the *App.test = true, my custom API creds are used.
func (a *App) LoadConfig() {
	a.config.SetAppDir(a.dir)
	if a.test {
		err := a.config.TestConfig(a.dir)
		if err != nil {
			a.Errorln("Could not load test configuration: ", err)
		}
	} else {
		err := a.config.LoadConfig(a.dir)
		if err != nil {
			// settings have not been loaded. This might be the first run.
			// use default settings instead.
			log.Println("could not load saved settings. reverting to default settings.")
			err = a.config.DefaultSettings(a.dir)
			if err != nil {
				a.Errorln("Leprechaun: could not initialize default configuration.", err)
			}
		}
	}
	a.config.ExportAPIVars(a.config.APIKeyID, a.config.APIKeySecret)

}

// Errorf writes a formatted error message to the startup log file
// and stderr.
func (a *App) Errorf(format string, args ...interface{}) {
	if startupLog, ok := a.logBackends["startup"]; ok {
		startupLog.Printf(format+"/n", args...)
	}

	log.SetOutput(os.Stderr)
	log.Printf(format+"\n", args...)
	go a.CloseLogFiles()
	time.Sleep(5 * time.Second)
	os.Exit(1)
}

// Errorln writes an error message to the startup log file
// and stderr.
func (a *App) Errorln(args ...interface{}) {
	if startupLog, ok := a.logBackends["startup"]; ok {
		startupLog.Println(args...)
	}

	log.SetOutput(os.Stderr)
	log.Println(args...)
	go a.CloseLogFiles()
	time.Sleep(5 * time.Second)
	os.Exit(1)
}
