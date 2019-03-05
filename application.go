package flutter

import (
	"unsafe"

	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/pkg/errors"

	"github.com/go-flutter-desktop/go-flutter/embedder"
)

// Run executes a flutter application with the provided options.
// given limitations this method must be called by the main function directly.
//
// Run(opt) is short for NewApplication(opt).Run()
func Run(opt ...Option) (err error) {
	return NewApplication(opt...).Run()
}

// Application provides the flutter engine in a user friendly matter.
type Application struct {
	config config
	engine *embedder.FlutterEngine
	window *glfw.Window
}

// NewApplication creates a new application with provided options.
func NewApplication(opt ...Option) *Application {
	app := &Application{
		config: defaultApplicationConfig,
	}

	// The platformPlugin and textinputPlugin are currently hardcoded as we have
	// a hard link with GLFW. The plugins must be singleton and are accessed
	// directly from the flutter package to wire up with glfw. If there's going
	// to be a renderer interface, it's init would replace this configuration.
	opt = append(opt, AddPlugin(defaultPlatformPlugin))
	opt = append(opt, AddPlugin(defaultTextinputPlugin))

	// apply all configs
	for _, o := range opt {
		o(&app.config)
	}

	// TODO: validate config (some fields are mandatory)

	return app
}

// Run starts the application and waits for it to finish.
func (a *Application) Run() error {
	err := glfw.Init()
	if err != nil {
		return errors.Wrap(err, "glfw init")
	}
	defer glfw.Terminate()

	a.window, err = glfw.CreateWindow(a.config.windowInitialDimensions.x, a.config.windowInitialDimensions.y, "Loading..", nil, nil)
	if err != nil {
		return errors.Wrap(err, "creating glfw window")
	}
	defer a.window.Destroy()

	if a.config.windowIconProvider != nil {
		images, err := a.config.windowIconProvider()
		if err != nil {
			return errors.Wrap(err, "getting images from icon provider")
		}
		a.window.SetIcon(images)
	}

	if a.config.windowInitializerDeprecated != nil {
		err = a.config.windowInitializerDeprecated(a.window)
		if err != nil {
			return errors.Wrap(err, "executing window initializer")
		}
	}

	a.engine = embedder.NewFlutterEngine()

	messenger := newMessenger(a.engine)
	for _, p := range a.config.plugins {
		err = p.InitPlugin(messenger)
		if err != nil {
			return errors.Wrap(err, "failed initializing plugin")
		}

		// Extra init call for plugins that satisfy the PluginGLFW interface.
		if glfwPlugin, ok := p.(PluginGLFW); ok {
			err = glfwPlugin.InitPluginGLFW(a.window)
			if err != nil {
				return errors.Wrap(err, "failed initializing glfw plugin")
			}
		}
	}

	// Engine arguments
	a.engine.AssetsPath = a.config.assetsPath
	a.engine.IcuDataPath = a.config.icuDataPath

	// Render callbacks
	a.engine.FMakeCurrent = func(v unsafe.Pointer) bool {
		w := glfw.GoWindow(v)
		w.MakeContextCurrent()
		return true
	}
	a.engine.FClearCurrent = func(v unsafe.Pointer) bool {
		glfw.DetachCurrentContext()
		return true
	}
	a.engine.FPresent = func(v unsafe.Pointer) bool {
		w := glfw.GoWindow(v)
		w.SwapBuffers()
		return true
	}
	a.engine.FFboCallback = func(v unsafe.Pointer) int32 {
		return 0
	}
	a.engine.FMakeResourceCurrent = func(v unsafe.Pointer) bool {
		return false
	}

	a.engine.FPlatfromMessage = messenger.handlePlatformMessage

	// func(platMessage *embedder.PlatformMessage, windowPtr unsafe.Pointer) bool {
	// 	window := glfw.GoWindow(windowPtr)

	// 	hasDispatched := false

	// 	// Dispatch the message from the Flutter Engine, to all of the PluginReceivers
	// 	// having the same embedder.PlatformMessage.Channel name
	// 	for _, receivers := range c.PlatformMessageReceivers[platMessage.Channel] {
	// 		hasDispatched = receivers(platMessage, flutterEngine, windows) || hasDispatched
	// 	}

	// 	return hasDispatched
	// }

	state.notifyState = func() {
		defaultTextinputPlugin.updateEditingState()
	}

	flutterEngineIndex := a.engine.Index()
	a.window.SetUserPointer(unsafe.Pointer(&flutterEngineIndex))
	result := a.engine.Run(a.window.GLFWWindow(), a.config.vmArguments)

	if result != embedder.KSuccess {
		a.window.Destroy()
		panic("Couldn't launch the FlutterEngine")
	}

	glfwFramebufferSizeCallback := newGLFWFramebufferSizeCallback(a.config.forcePixelRatio, getScreenCoordinatesPerInch())
	width, height := a.window.GetFramebufferSize()
	glfwFramebufferSizeCallback(a.window, width, height)
	a.window.SetFramebufferSizeCallback(glfwFramebufferSizeCallback)

	a.window.SetKeyCallback(defaultTextinputPlugin.glfwKeyCallback)
	a.window.SetCharCallback(defaultTextinputPlugin.glfwCharCallback)

	a.window.SetMouseButtonCallback(glfwMouseButtonCallback)
	defer a.engine.Shutdown()

	for !a.window.ShouldClose() {
		// glfw.WaitEvents()
		glfw.PollEvents()
		embedder.FlutterEngineFlushPendingTasksNow()
	}

	return nil
}

// TODO: app.Start(), app.Wait()?
