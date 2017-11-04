package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"git.raceresult.com/LocalAdapterServer/gui/lamanager"
	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilectron-bundler"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
	"github.com/tehsphinx/dbg"
)

// Vars
var (
	homepage     = "index.html"
	AppName      string
	debug        = flag.Bool("d", false, "if yes, the app is in debug mode")
	la           *lamanager.Process
	restarting   bool
	restartMutex sync.Mutex
	sending      bool
	sendingMutex sync.Mutex
	a            *astilectron.Astilectron
	window       *astilectron.Window
	logWindow    *astilectron.Window
)

// MessageHandler is a functions that handles messages
type MessageHandler func(w *astilectron.Window, m MessageIn) (payload interface{}, err error)

// MessageOut represents a message going out
type MessageOut struct {
	CallbackID *int        `json:"callbackId,omitempty"`
	Name       string      `json:"name"`
	Payload    interface{} `json:"payload"`
}

// MessageIn represents a message going in
type MessageIn struct {
	CallbackID *int            `json:"callbackId,omitempty"`
	Name       string          `json:"name"`
	Payload    json.RawMessage `json:"payload"`
}

func main() {
	flag.Parse()
	astilog.FlagInit()

	// Run bootstrap
	if err := Run(); err != nil {
		astilog.Fatal(errors.Wrap(err, "running bootstrap failed"))
	}
}

func startLA() {
	la = lamanager.RunProcess()
}

func checkLA() {
	go func() {
		lastVal := false
		for {
			time.Sleep(200 * time.Millisecond)

			if window == nil {
				continue
			}

			running := la.IsRunning()
			if lastVal != running {
				lastVal = running
				if err := window.Send(MessageOut{Name: "set.running", Payload: running}); err != nil {
					astilog.Error(errors.Wrap(err, "update running state failed"))
					return
				}
			}
		}
	}()
}

func onWait(_ *astilectron.Astilectron, w *astilectron.Window, _ *astilectron.Menu, t *astilectron.Tray, _ *astilectron.Menu) error {
	// Store global variables
	window = w

	// Add listeners on tray
	t.On(astilectron.EventNameTrayEventClicked, func(e astilectron.Event) (deleteListener bool) {
		astilog.Info("Tray has been clicked!")
		return
	})

	//startLA()
	//checkLA()
	return nil
}

// handleMessages handles messages
func handleMessages(w *astilectron.Window, m MessageIn) (payload interface{}, err error) {
	switch m.Name {
	case "la.restart":
		go restartLA()
	case "log.show":
		go showLog()
	case "log.start":
		go sendLog()
	}
	return
}

func sendLog() {
	sendingMutex.Lock()
	if sending {
		sendingMutex.Unlock()
		return
	}
	sending = true
	sendingMutex.Unlock()

	for {
		time.Sleep(1 * time.Second)

		l := la.GetLog()
		if r := logWindow.Send(MessageOut{Name: "log.all", Payload: strings.Join(l, "\n")}); r != nil {
			log.Println("could not send log:", r)
			break
		}
	}

	sendingMutex.Lock()
	sending = false
	sendingMutex.Unlock()
}

func closeLA() {
	if err := la.Stop(); err != nil {
		astilog.Error("error closing local adapter:", err)
	}
}

func restartLA() {
	restartMutex.Lock()
	if restarting {
		restartMutex.Unlock()
		return
	}
	restarting = true
	restartMutex.Unlock()

	closeLA()
	time.Sleep(2 * time.Second)
	startLA()
	time.Sleep(2 * time.Second)

	restartMutex.Lock()
	restarting = false
	restartMutex.Unlock()
}

func showLog() {
	var (
		err           error
		windowOptions = &astilectron.WindowOptions{
			BackgroundColor: astilectron.PtrStr("#333"),
			Center:          astilectron.PtrBool(true),
			Height:          astilectron.PtrInt(600),
			Width:           astilectron.PtrInt(600),
		}
	)

	// Debug
	if *debug {
		windowOptions.Width = astilectron.PtrInt(*windowOptions.Width + 700)
	}

	// Init window
	if logWindow, err = a.NewWindow(filepath.Join(a.Paths().BaseDirectory(), "resources", "app", "log.html"), windowOptions); err != nil {
		log.Println(errors.Wrap(err, "new window failed"))
	}

	// Handle messages
	logWindow.On(astilectron.EventNameWindowEventMessage, handleMessageWrapper(logWindow, handleMessages))

	// Create window
	if err = logWindow.Create(); err != nil {
		log.Println(errors.Wrap(err, "creating window failed"))
	}

	// Debug
	if *debug {
		if err = logWindow.OpenDevTools(); err != nil {
			log.Println(errors.Wrap(err, "opening dev tools failed"))
		}
	}
}

func showEvent(e astilectron.Event) (deleteListener bool) {
	dbg.Green("showEvent")
	window.Show()
	return
}

func showLogEvent(e astilectron.Event) (deleteListener bool) {
	dbg.Green("showLogEvent")
	go showLog()
	return
}

func closedEvent(e astilectron.Event) (deleteListener bool) {
	dbg.Green("closedEvent")
	closeLA()
	return
}

func restartEvent(e astilectron.Event) (deleteListener bool) {
	dbg.Green("restartEvent")
	restartLA()
	return
}

func windowCloseEvent(e astilectron.Event) (deleteListener bool) {
	dbg.Green("windowCloseEvent")
	dbg.Green(e)
	return
}

func minimizeEvent(e astilectron.Event) (deleteListener bool) {
	dbg.Green("minimizeEvent")
	dbg.Green(e)
	window.Hide()
	return
}

func exitEvent(e astilectron.Event) (deleteListener bool) {
	dbg.Green("exitEvent")
	closeLA()
	a.Quit()
	return
}

func Run() (err error) {
	menu := []*astilectron.MenuItemOptions{
		{
			Label:   astilectron.PtrStr("Show"),
			OnClick: showEvent,
		},
		{
			Label:   astilectron.PtrStr("Restart"),
			OnClick: restartEvent,
		},
		{
			Label:   astilectron.PtrStr("Log"),
			OnClick: showLogEvent,
		},
		{
			Label:   astilectron.PtrStr("Exit"),
			OnClick: exitEvent,
		},
		{
			Role: astilectron.MenuItemRoleClose,
		},
	}
	var (
		options = astilectron.Options{
			AppName:            AppName,
			AppIconDarwinPath:  "resources/rr.icns",
			AppIconDefaultPath: "resources/rr.png",
		}
		menuOptions = []*astilectron.MenuItemOptions{
			{
				Label:   astilectron.PtrStr(AppName),
				SubMenu: menu,
			},
		}
		trayOptions = &astilectron.TrayOptions{
			Image:   astilectron.PtrStr("resources/tray.png"),
			Tooltip: astilectron.PtrStr("race|result Local Adapter"),
		}
		trayMenuOptions = menu
		windowOptions   = &astilectron.WindowOptions{
			BackgroundColor: astilectron.PtrStr("#333"),
			Center:          astilectron.PtrBool(true),
			Height:          astilectron.PtrInt(160),
			Width:           astilectron.PtrInt(242),
			Show:            astilectron.PtrBool(false),
			SkipTaskbar:     astilectron.PtrBool(true),
			//MinimizeOnClose: astilectron.PtrBool(true),
			MessageBoxOnClose: &astilectron.MessageBoxOptions{
				Buttons:   []string{"Yes", "No"},
				ConfirmID: astilectron.PtrInt(0),
				Message:   "Are you sure you want to quit?",
				Title:     "Confirm",
				Type:      astilectron.MessageBoxTypeQuestion,
			},
		}
	)

	// Get executable path
	var p string
	if p, err = os.Executable(); err != nil {
		err = errors.Wrap(err, "os.Executable failed")
		return
	}
	p = filepath.Dir(p)

	// Make sure option paths are absolute
	if len(options.AppIconDarwinPath) > 0 && !filepath.IsAbs(options.AppIconDarwinPath) {
		options.AppIconDarwinPath = filepath.Join(p, options.AppIconDarwinPath)
	}
	if len(options.AppIconDefaultPath) > 0 && !filepath.IsAbs(options.AppIconDefaultPath) {
		options.AppIconDefaultPath = filepath.Join(p, options.AppIconDefaultPath)
	}
	if trayOptions != nil && trayOptions.Image != nil && !filepath.IsAbs(*trayOptions.Image) {
		*trayOptions.Image = filepath.Join(p, *trayOptions.Image)
	}

	// Create astilectron
	if a, err = astilectron.New(options); err != nil {
		return errors.Wrap(err, "creating new astilectron failed")
	}
	defer a.Close()
	a.HandleSignals()

	// Add listeners on Astilectron
	a.On(astilectron.EventNameAppCrash, closedEvent)
	a.On(astilectron.EventNameAppClose, closedEvent)

	// Set provisioner
	a.SetProvisioner(astibundler.NewProvisioner(Asset))

	// Restore resources
	var rp = filepath.Join(a.Paths().BaseDirectory(), "resources")
	if _, err = os.Stat(rp); os.IsNotExist(err) {
		astilog.Debugf("Restoring resources in %s", rp)
		if err = RestoreAssets(a.Paths().BaseDirectory(), "resources"); err != nil {
			err = errors.Wrapf(err, "restoring resources in %s failed", rp)
			return
		}
	} else if err != nil {
		err = errors.Wrapf(err, "stating %s failed", rp)
		return
	} else {
		astilog.Debugf("%s already exists, skipping restoring resources...", rp)
	}

	// Start
	if err = a.Start(); err != nil {
		return errors.Wrap(err, "starting astilectron failed")
	}

	// Debug
	if *debug {
		windowOptions.Width = astilectron.PtrInt(*windowOptions.Width + 700)
	}

	// Init window
	var w *astilectron.Window
	if w, err = a.NewWindow(filepath.Join(a.Paths().BaseDirectory(), "resources", "app", homepage), windowOptions); err != nil {
		return errors.Wrap(err, "new window failed")
	}

	// Handle messages
	w.On(astilectron.EventNameWindowEventMessage, handleMessageWrapper(w, handleMessages))
	w.On(astilectron.EventNameWindowCmdClose, windowCloseEvent)
	w.On(astilectron.EventNameWindowEventClosed, windowCloseEvent)
	w.On(astilectron.EventNameAppClose, windowCloseEvent)
	w.On(astilectron.EventNameWindowCmdMinimize, minimizeEvent)
	w.On(astilectron.EventNameWindowEventMinimize, minimizeEvent)

	w.On(astilectron.EventNameAppCmdStop, func(e astilectron.Event) (deleteListener bool) {
		dbg.Green("app stopped")
		return
	})
	w.On(astilectron.EventNameWindowEventResize, func(e astilectron.Event) (deleteListener bool) {
		dbg.Green("Window resized")
		return
	})
	w.On(astilectron.EventNameWindowEventMinimize, func(e astilectron.Event) (deleteListener bool) {
		dbg.Green("Window minimized")
		return
	})
	w.On(astilectron.EventNameWindowEventMaximize, func(e astilectron.Event) (deleteListener bool) {
		dbg.Green("Window maximized")
		return
	})

	// Create window
	if err = w.Create(); err != nil {
		return errors.Wrap(err, "creating window failed")
	}

	// Debug
	if *debug {
		if err = w.OpenDevTools(); err != nil {
			return errors.Wrap(err, "opening dev tools failed")
		}
	}

	// Menu
	var m *astilectron.Menu
	if len(menuOptions) > 0 {
		// Init menu
		m = a.NewMenu(menuOptions)

		// Create menu
		if err = m.Create(); err != nil {
			return errors.Wrap(err, "creating menu failed")
		}
	}

	// Tray
	t := a.NewTray(trayOptions)
	if err = t.Create(); err != nil {
		return errors.Wrap(err, "creating tray failed")
	}

	// Tray menu
	tm := t.NewMenu(trayMenuOptions)
	if err = tm.Create(); err != nil {
		return errors.Wrap(err, "creating tray menu failed")
	}

	// On wait
	if err = onWait(a, w, m, t, tm); err != nil {
		return errors.Wrap(err, "onwait failed")
	}

	// Blocking pattern
	a.Wait()
	return
}

// handleMessageWrapper handles messages
func handleMessageWrapper(w *astilectron.Window, messageHandler MessageHandler) astilectron.Listener {
	return func(e astilectron.Event) (deleteListener bool) {
		// Unmarshal message
		var m MessageIn
		var err error
		if err = e.Message.Unmarshal(&m); err != nil {
			astilog.Error(errors.Wrapf(err, "unmarshaling message %+v failed", *e.Message))
			return
		}

		// Handle message
		var p interface{}
		if p, err = messageHandler(w, m); err != nil {
			astilog.Error(errors.Wrapf(err, "handling message %+v failed", m))
			return
		}

		// Send message
		if p != nil && m.CallbackID != nil {
			var m = MessageOut{CallbackID: m.CallbackID, Name: m.Name, Payload: p}
			if err = w.Send(m); err != nil {
				astilog.Error(errors.Wrapf(err, "sending message %+v failed", m))
				return
			}
		}
		return
	}
}
