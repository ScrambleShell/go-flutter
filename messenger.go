package flutter

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/go-flutter-desktop/go-flutter/embedder"
	"github.com/go-flutter-desktop/go-flutter/plugin"
)

type messenger struct {
	engine *embedder.FlutterEngine

	channels     map[string]plugin.ChannelHandlerFunc
	channelsLock sync.RWMutex
}

var _ plugin.BinaryMessenger = &messenger{}

func newMessenger(engine *embedder.FlutterEngine) *messenger {
	return &messenger{
		engine:   engine,
		channels: make(map[string]plugin.ChannelHandlerFunc),
	}
}

// TODO: Does flutter even send a reply!? Or is Host -> Fluter one-way?
func (m *messenger) Send(channel string, encodedMessage []byte) (encodedReply []byte, err error) {
	msg := &embedder.PlatformMessage{
		Channel: channel,
		Message: encodedMessage,
	}
	res := m.engine.SendPlatformMessage(msg)
	if err != nil {
		if ferr, ok := err.(*plugin.FlutterError); ok {
			return nil, ferr
		}
	}
	if res != embedder.KSuccess {
		return nil, errors.New("failed to send message")
	}
	return nil, nil
}

// SetChannelHandler satisfies plugin.BinaryMessenger
func (m *messenger) SetChannelHandler(channel string, channelHandler plugin.ChannelHandlerFunc) {
	m.channelsLock.Lock()
	if channelHandler == nil {
		// TODO: this is actually never really being used.. should it?
		delete(m.channels, channel)
	} else {
		m.channels[channel] = channelHandler
	}
	m.channelsLock.Unlock()
}

func (m *messenger) handlePlatformMessage(message *embedder.PlatformMessage) {
	go func() {
		m.channelsLock.RLock()
		channelHander := m.channels[message.Channel]
		m.channelsLock.RUnlock()

		if channelHander == nil {
			// TODO: what to do on a message with unregistered channel? Who is
			// responsible? Send reply back? os.Exit?
			fmt.Println("go-flutter: no handler found for channel " + message.Channel)
			os.Exit(1)
			return
		}

		// TODO: handle concurrently and return directly?
		encodedReply, err := channelHander(message.Message)
		if err != nil {
			// TODO: should we even allow errors to be returned here?
			fmt.Println("go-flutter: handling message on channel " + message.Channel + " failed")
			os.Exit(1)
		}
		if message.ExpectsReply() {
			// TODO: ?add channels and a goroutine with locked thread so that messaging is always performed from the
			// same thread?
			res := m.engine.SendPlatformMessageResponse(message.ResponseHandle, encodedReply)
			if res != embedder.KSuccess {
				fmt.Println("go-flutter: failed sending response for message on channel " + message.Channel)
			}
		}
	}()
}
