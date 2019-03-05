package flutter

import (
	"encoding/json"
	"fmt"

	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/pkg/errors"

	"github.com/go-flutter-desktop/go-flutter/plugin"
)

// platformPlugin implements flutter.Plugin and handles method calls to the
// flutter/platform channel.
type platformPlugin struct {
	messenger plugin.BinaryMessenger
	window    *glfw.Window
	channel   *plugin.MethodChannel
}

// hardcoded because there is no swappable renderer interface.
var defaultPlatformPlugin = &platformPlugin{}

var _ Plugin = &platformPlugin{}     // compile-time type check
var _ PluginGLFW = &platformPlugin{} // compile-time type check

func (p *platformPlugin) InitPlugin(messenger plugin.BinaryMessenger) error {
	p.messenger = messenger
	return nil
}

func (p *platformPlugin) InitPluginGLFW(window *glfw.Window) (err error) {
	p.window = window
	p.channel = plugin.NewMethodChannel(p.messenger, "flutter/platform", plugin.JSONMethodCodec{})

	p.channel.HandleFunc("Clipboard.setData", p.handleClipboardSetData)
	p.channel.HandleFunc("Clipboard.getData", p.handleClipboardGetData)
	p.channel.HandleFunc("SystemChrome.setApplicationSwitcherDescription", p.handleWindowSetTitle)

	return nil
}

func (p *platformPlugin) handleClipboardSetData(arguments interface{}) (reply interface{}, err error) {
	newClipboard := struct {
		Text string `json:"text"`
	}{}
	err = json.Unmarshal(arguments.(json.RawMessage), &newClipboard)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode json arguments for handleClipboardSetData")
	}
	p.window.SetClipboardString(newClipboard.Text)
	return nil, nil
}

func (p *platformPlugin) handleClipboardGetData(arguments interface{}) (reply interface{}, err error) {
	fmt.Println("handleClipboardGetData")
	requestedMime := ""
	err = json.Unmarshal(arguments.(json.RawMessage), &requestedMime)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode json arguments for handleClipboardGetData")
	}
	if requestedMime != "text/plain" {
		return nil, errors.New("obtaining mime type " + requestedMime + " from clipboard is not yet supported in go-flutter")
	}

	// TODO: this seems to be a bit bugged? Blocks when there hasn't been a copy
	// during the application lifetime..
	clipText, err := p.window.GetClipboardString()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get string from clipboard")
	}

	reply = struct {
		Text string `json:"text"`
	}{
		Text: clipText,
	}
	return reply, nil
}

func (p *platformPlugin) handleWindowSetTitle(arguments interface{}) (reply interface{}, err error) {
	jsonArguments := arguments.(json.RawMessage)

	appSwitcherDescription := struct {
		Label        string `json:"label"`
		PrimaryColor int64  `json:"primaryColor"`
	}{}
	err = json.Unmarshal(jsonArguments, &appSwitcherDescription)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode arguments")
	}
	p.window.SetTitle(appSwitcherDescription.Label)
	return nil, nil
}
