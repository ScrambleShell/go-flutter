package flutter

import (
	"fmt"
	"time"

	"github.com/go-flutter-desktop/go-flutter/embedder"
	"github.com/go-gl/glfw/v3.2/glfw"
)

// dpPerInch defines the amount of display pixels per inch as defined for Flutter.
const dpPerInch = 160.0

// GLFW callbacks to the Flutter Engine
func glfwCursorPositionCallbackAtPhase(
	window *glfw.Window, phase embedder.PointerPhase,
	x float64, y float64,
) {
	winWidth, _ := window.GetSize()
	frameBuffWidth, _ := window.GetFramebufferSize()
	contentScale := float64(frameBuffWidth / winWidth)
	event := embedder.PointerEvent{
		Phase:     phase,
		X:         x * contentScale,
		Y:         y * contentScale,
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
	}

	index := *(*int)(window.GetUserPointer())
	flutterEngine := embedder.FlutterEngineByIndex(index)

	flutterEngine.SendPointerEvent(event)
}

func glfwMouseButtonCallback(window *glfw.Window, key glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if key == glfw.MouseButton1 {
		x, y := window.GetCursorPos()

		// recalculate x and y from screen cordinates to pixels
		widthPx, _ := window.GetFramebufferSize()
		width, _ := window.GetSize()
		pixelsPerScreenCoordinate := float64(widthPx) / float64(width)
		x = x * pixelsPerScreenCoordinate
		y = y * pixelsPerScreenCoordinate

		if action == glfw.Press {
			glfwCursorPositionCallbackAtPhase(window, embedder.KDown, x, y)
			window.SetCursorPosCallback(func(window *glfw.Window, x float64, y float64) {
				x = x * pixelsPerScreenCoordinate
				y = y * pixelsPerScreenCoordinate
				glfwCursorPositionCallbackAtPhase(window, embedder.KMove, x, y)
			})
		}

		if action == glfw.Release {
			glfwCursorPositionCallbackAtPhase(window, embedder.KUp, x, y)
			window.SetCursorPosCallback(nil)
		}
	}
}

var state = textModel{}

// newGLFWFramebufferSizeCallback creates a func that is called on framebuffer
// resizes. When pixelRatio is zero, the pixelRatio communicated to the Flutter
// embedder is calculated based on physical and logical screen dimensions.
func newGLFWFramebufferSizeCallback(pixelRatio float64, monitorScreenCoordinatesPerInch float64) func(*glfw.Window, int, int) {
	return func(window *glfw.Window, widthPx int, heightPx int) {
		index := *(*int)(window.GetUserPointer())
		flutterEngine := embedder.FlutterEngineByIndex(index)

		// calculate pixelRatio when it has not been forced.
		if pixelRatio == 0 {
			width, _ := window.GetSize()
			if width == 0 {
				pixelRatio = 1.0
				goto SendWindowMetricsEvent
			}

			pixelsPerScreenCoordinate := float64(widthPx) / float64(width)
			dpi := pixelsPerScreenCoordinate * monitorScreenCoordinatesPerInch
			pixelRatio = dpi / dpPerInch

			// Limit the ratio to 1 to avoid rendering a smaller UI in standard resolution monitors.
			if pixelRatio < 1.0 {
				// TODO: wrap with a sync.Once to avoid spam?
				fmt.Println("go-flutter: calculated pixelRatio limited to a minimum of 1.0")
				pixelRatio = 1.0
			}
		}

	SendWindowMetricsEvent:
		event := embedder.WindowMetricsEvent{
			Width:      widthPx,
			Height:     heightPx,
			PixelRatio: pixelRatio,
		}
		flutterEngine.SendWindowMetricsEvent(event)
	}
}

// getScreenCoordinatesPerInch returns the number of screen coordinates per
// inch for the main monitor. If the information is unavailable it returns
// a default value that assumes that a screen coordinate is one dp.
func getScreenCoordinatesPerInch() float64 {
	// TODO: multi-monitor support (#74)
	primaryMonitor := glfw.GetPrimaryMonitor()
	if primaryMonitor == nil {
		return dpPerInch
	}
	primaryMonitorMode := primaryMonitor.GetVideoMode()
	if primaryMonitorMode == nil {
		return dpPerInch
	}
	primaryMonitorWidthMM, _ := primaryMonitor.GetPhysicalSize()
	if primaryMonitorWidthMM == 0 {
		return dpPerInch
	}
	return float64(primaryMonitorMode.Width) / (float64(primaryMonitorWidthMM) / 25.4)
}
