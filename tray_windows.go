//go:build windows

package main

import (
	"runtime"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	registerClass    = user32.NewProc("RegisterClassExW")
	createWindow     = user32.NewProc("CreateWindowExW")
	defWindowProc    = user32.NewProc("DefWindowProcW")
	getMessage       = user32.NewProc("GetMessageW")
	translateMessage = user32.NewProc("TranslateMessage")
	dispatchMessage  = user32.NewProc("DispatchMessageW")
	postQuit         = user32.NewProc("PostQuitMessage")
	loadImage        = user32.NewProc("LoadImageW")
	loadIcon         = user32.NewProc("LoadIconW")
	createMenu       = user32.NewProc("CreatePopupMenu")
	appendMenu       = user32.NewProc("AppendMenuW")
	trackMenu        = user32.NewProc("TrackPopupMenu")
	destroyMenu      = user32.NewProc("DestroyMenu")
	cursorPos        = user32.NewProc("GetCursorPos")
	foreground       = user32.NewProc("SetForegroundWindow")
	postMessage      = user32.NewProc("PostMessageW")
	notifyIcon       = shell32.NewProc("Shell_NotifyIconW")
	moduleHandle     = kernel32.NewProc("GetModuleHandleW")
)

const (
	wmDestroy     = 0x0002
	wmCommand     = 0x0111
	wmLeftUp      = 0x0202
	wmRightUp     = 0x0205
	wmTrayMessage = 0x8001

	nimAdd    = 0x0000
	nimDelete = 0x0002

	nifMessage = 0x0001
	nifIcon    = 0x0002
	nifTip     = 0x0004

	openItem = 1
	quitItem = 2
)

type wndClass struct {
	size       uint32
	style      uint32
	proc       uintptr
	clsExtra   int32
	wndExtra   int32
	instance   syscall.Handle
	icon       syscall.Handle
	cursor     syscall.Handle
	background syscall.Handle
	menuName   *uint16
	className  *uint16
	iconSmall  syscall.Handle
}

// The order and the padding are Windows', not ours: one field out of place and
// Shell_NotifyIcon fails without saying why.
type notify struct {
	size      uint32
	window    syscall.Handle
	id        uint32
	flags     uint32
	callback  uint32
	icon      syscall.Handle
	tip       [128]uint16
	state     uint32
	stateMask uint32
	info      [256]uint16
	version   uint32
	infoTitle [64]uint16
	infoFlags uint32
	guid      [16]byte
	balloon   syscall.Handle
}

type message struct {
	window  syscall.Handle
	kind    uint32
	wparam  uintptr
	lparam  uintptr
	time    uint32
	x, y    int32
	private uint32
}

type point struct{ x, y int32 }

func tray(address string, open func(), quit func()) {
	runtime.LockOSThread()

	instance, _, _ := moduleHandle.Call(0)
	name := must("netwatch-tray")

	handler := syscall.NewCallback(func(window syscall.Handle, kind uint32,
		wparam, lparam uintptr) uintptr {
		switch kind {
		case wmTrayMessage:
			switch lparam {
			case wmLeftUp:
				open()
			case wmRightUp:
				choose(window)
			}

		case wmCommand:
			switch wparam & 0xffff {
			case openItem:
				open()
			case quitItem:
				postQuit.Call(0)
			}

		case wmDestroy:
			postQuit.Call(0)
		}

		answer, _, _ := defWindowProc.Call(uintptr(window), uintptr(kind), wparam, lparam)

		return answer
	})

	class := wndClass{
		proc:      handler,
		instance:  syscall.Handle(instance),
		className: name,
	}
	class.size = uint32(unsafe.Sizeof(class))

	if made, _, _ := registerClass.Call(uintptr(unsafe.Pointer(&class))); made == 0 {
		return
	}

	window, _, _ := createWindow.Call(0, uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(must("netwatch"))), 0, 0, 0, 0, 0, 0, 0, instance, 0)
	if window == 0 {
		return
	}

	seen := notify{
		window:   syscall.Handle(window),
		id:       1,
		flags:    nifMessage | nifIcon | nifTip,
		callback: wmTrayMessage,
		icon:     mark(instance),
	}
	seen.size = uint32(unsafe.Sizeof(seen))
	copy(seen.tip[:], syscall.StringToUTF16("netwatch"))

	notifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&seen)))
	defer notifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&seen)))

	var said message

	for {
		got, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&said)), 0, 0, 0)
		if int32(got) <= 0 {
			break
		}

		translateMessage.Call(uintptr(unsafe.Pointer(&said)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&said)))
	}

	quit()
}

// The icon built into the exe, and the one Windows hands out when there is
// none: a tray with a blank square in it looks broken.
func mark(instance uintptr) syscall.Handle {
	const imageIcon, loadShared, defaultSize = 1, 0x8000, 0x0040

	for _, id := range []uintptr{1, 2, 3} {
		got, _, _ := loadImage.Call(instance, id, imageIcon, 0, 0,
			loadShared|defaultSize)
		if got != 0 {
			return syscall.Handle(got)
		}
	}

	got, _, _ := loadIcon.Call(0, 32512)

	return syscall.Handle(got)
}

// A menu will not go away on its own unless the window it belongs to is in
// front, and it leaves itself on screen unless the window is poked afterwards.
func choose(window syscall.Handle) {
	menu, _, _ := createMenu.Call()
	if menu == 0 {
		return
	}
	defer destroyMenu.Call(menu)

	appendMenu.Call(menu, 0, openItem, uintptr(unsafe.Pointer(must("Open netwatch"))))
	appendMenu.Call(menu, 0, quitItem, uintptr(unsafe.Pointer(must("Quit"))))

	var where point
	cursorPos.Call(uintptr(unsafe.Pointer(&where)))

	foreground.Call(uintptr(window))
	trackMenu.Call(menu, 0x0002|0x0020, uintptr(where.x), uintptr(where.y),
		0, uintptr(window), 0)
	postMessage.Call(uintptr(window), 0, 0, 0)
}

func must(text string) *uint16 {
	made, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return nil
	}

	return made
}
