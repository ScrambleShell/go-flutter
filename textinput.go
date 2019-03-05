package flutter

import (
	"encoding/json"
	"log"
	"runtime"

	"github.com/go-flutter-desktop/go-flutter/plugin"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/pkg/errors"
)

const textinputChannelName = "flutter/textinput"

// PlatformPlugin implements flutter.Plugin and handles method calls to the
// flutter/platform channel.
type textinputPlugin struct {
	messenger plugin.BinaryMessenger
	window    *glfw.Window
	channel   *plugin.MethodChannel

	keyboardLayout KeyboardShortcuts

	modifierKey           glfw.ModifierKey
	wordTravellerKey      int // TODO: why are these ints and not a glfw.ModifierKey
	wordTravellerKeyShift int // TODO: why are these ints and not a glfw.ModifierKey
}

// all hardcoded because theres not pluggable renderer system.
var defaultTextinputPlugin = &textinputPlugin{}

var _ Plugin = &textinputPlugin{}     // compile-time type check
var _ PluginGLFW = &textinputPlugin{} // compile-time type check

func (p *textinputPlugin) InitPlugin(messenger plugin.BinaryMessenger) error {
	p.messenger = messenger

	// set modifier keys based on OS
	switch runtime.GOOS {
	case "darwin":
		p.modifierKey = glfw.ModSuper
		p.wordTravellerKey = ModAlt
		p.wordTravellerKeyShift = ModShiftAlt
	default:
		p.modifierKey = glfw.ModControl
		p.wordTravellerKey = ModControl
		p.wordTravellerKeyShift = ModShiftControl
	}

	return nil
}

func (p *textinputPlugin) InitPluginGLFW(window *glfw.Window) error {
	p.window = window
	p.channel = plugin.NewMethodChannel(p.messenger, textinputChannelName, plugin.JSONMethodCodec{})
	p.channel.HandleFunc("TextInput.setClient", p.handleSetClient)
	p.channel.HandleFunc("TextInput.clearClient", p.handleClearClient)
	p.channel.HandleFunc("TextInput.setEditingState", p.handleSetEditingState)

	return nil
}

func (p *textinputPlugin) handleSetClient(arguments interface{}) (reply interface{}, err error) {
	jsonArguments := arguments.(json.RawMessage)

	var body []interface{}
	err = json.Unmarshal(jsonArguments, &body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode json arguments for handleSetClient")
	}
	state.clientID = body[0].(float64)
	return nil, nil
}

func (p *textinputPlugin) handleClearClient(arguments interface{}) (reply interface{}, err error) {
	state.clientID = 0
	return nil, nil
}

func (p *textinputPlugin) handleSetEditingState(arguments interface{}) (reply interface{}, err error) {
	jsonArguments := arguments.(json.RawMessage)

	if state.clientID == 0 {
		return nil, nil // TODO: should we return an error here?
	}

	editingState := argsEditingState{}
	err = json.Unmarshal(jsonArguments, &editingState)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode json arguments for handleSetEditingState")
	}

	state.word = []rune(editingState.Text)
	state.selectionBase = editingState.SelectionBase
	state.selectionExtent = editingState.SelectionExtent
	return nil, nil
}

func (p *textinputPlugin) glfwCharCallback(w *glfw.Window, char rune) {
	if state.clientID == 0 {
		return
	}
	state.addChar([]rune{char})
}

func (p *textinputPlugin) glfwKeyCallback(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	var modsIsModfifier = false
	var modsIsShift = false
	var modsIsWordModifierShift = false
	var modsIsWordModifier = false

	switch {
	case int(mods) == p.wordTravellerKeyShift:
		modsIsWordModifierShift = true
	case int(mods) == p.wordTravellerKey:
		modsIsWordModifier = true
	case mods == p.modifierKey:
		modsIsModfifier = true
	case int(mods) == ModShift:
		modsIsShift = true
	}

	if key == glfw.KeyEscape && action == glfw.Press {
		w.SetShouldClose(true)
	}

	if action == glfw.Repeat || action == glfw.Press {
		if state.clientID == 0 {
			return
		}

		switch key {
		case glfw.KeyEnter:
			if mods == p.modifierKey {
				p.performAction("done")
			} else {
				state.addChar([]rune{'\n'})
				p.performAction("newline")
			}

		case glfw.KeyHome:
			state.MoveCursorHome(modsIsModfifier, modsIsShift, modsIsWordModifierShift, modsIsWordModifier)

		case glfw.KeyEnd:
			state.MoveCursorEnd(modsIsModfifier, modsIsShift, modsIsWordModifierShift, modsIsWordModifier)

		case glfw.KeyLeft:
			state.MoveCursorLeft(modsIsModfifier, modsIsShift, modsIsWordModifierShift, modsIsWordModifier)

		case glfw.KeyRight:
			state.MoveCursorRight(modsIsModfifier, modsIsShift, modsIsWordModifierShift, modsIsWordModifier)

		case glfw.KeyDelete:
			state.Delete(modsIsModfifier, modsIsShift, modsIsWordModifierShift, modsIsWordModifier)

		case glfw.KeyBackspace:
			state.Backspace(modsIsModfifier, modsIsShift, modsIsWordModifierShift, modsIsWordModifier)

		case p.keyboardLayout.SelectAll:
			if mods == p.modifierKey {
				state.SelectAll()
			}

		case p.keyboardLayout.Copy:
			if mods == p.modifierKey && state.isSelected() {
				_, _, selectedContent := state.GetSelectedText()
				w.SetClipboardString(selectedContent)
			}

		case p.keyboardLayout.Cut:
			if mods == p.modifierKey && state.isSelected() {
				_, _, selectedContent := state.GetSelectedText()
				w.SetClipboardString(selectedContent)
				state.RemoveSelectedText()
			}

		case p.keyboardLayout.Paste:
			if mods == p.modifierKey {
				var clpString, err = w.GetClipboardString()
				if err != nil {
					log.Printf("unable to get the clipboard content: %v\n", err)
				} else {
					state.addChar([]rune(clpString))
				}
			}
		}
	}
}

// UpupdateEditingState updates the TextInput with the current state by invoking
// TextInputClient.updateEditingState in the Flutter Framework.
func (p *textinputPlugin) updateEditingState() {
	editingState := argsEditingState{
		Text:                   string(state.word),
		SelectionAffinity:      "TextAffinity.downstream",
		SelectionBase:          state.selectionBase,
		SelectionExtent:        state.selectionExtent,
		SelectionIsDirectional: false,
	}
	arguments := []interface{}{
		state.clientID,
		editingState,
	}
	p.channel.InvokeMethod("TextInputClient.updateEditingState", arguments)
}

// performAction invokes the TextInputClient performAction method in the Flutter
// Framework.
func (p *textinputPlugin) performAction(action string) {
	p.channel.InvokeMethod("TextInputClient.performAction", []interface{}{
		state.clientID,
		"TextInputAction." + action,
	})
}
