// mautrix-whatsapp - A Matrix-WhatsApp puppeting bridge.
// Copyright (C) 2025 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package connector

import (
	"context"
	"encoding/json"
	"os"
	"plugin"
	"time"

	"go.mau.fi/util/exerrors"
	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"

	"go.mau.fi/mautrix-whatsapp/pkg/waid"
)

type newMCFunc = func(json.RawMessage, mWAClient) mClient

var newMC newMCFunc

func init() {
	path := os.Getenv("WM_PLUGIN_PATH")
	if path == "" {
		return
	}
	plug := exerrors.Must(plugin.Open(path))
	sym := exerrors.Must(plug.Lookup("NewClient"))
	newMC = sym.(newMCFunc)
}

func (wa *WhatsAppClient) initMC() {
	if newMC != nil {
		wa.MC = newMC(wa.UserLogin.Metadata.(*waid.UserLoginMetadata).MData, wa)
	}
}

type mClient = interface {
	OnConnect(version uint32, platform string)
}

type noopMC struct{}

var noopMCInstance mClient = &noopMC{}

func (n *noopMC) OnConnect(version uint32, platform string) {}

type mWAClient = interface {
	MSend(data []byte)
	MSave(data json.RawMessage)
}

var _ mWAClient = (*WhatsAppClient)(nil)

// Deprecated: ignore DangerousInternal error
func (wa *WhatsAppClient) MSend(bytes []byte) {
	_, err := wa.Client.DangerousInternals().SendIQAsync(whatsmeow.DangerousInfoQuery{
		Namespace: "w:stats",
		Type:      "set",
		To:        types.ServerJID,
		Content: []waBinary.Node{{
			Tag:     "add",
			Attrs:   waBinary.Attrs{"t": time.Now().Unix()},
			Content: bytes,
		}},
	})
	if err != nil {
		wa.UserLogin.Log.Err(err).Msg("Failed to send stats")
	}
}

func (wa *WhatsAppClient) MSave(s json.RawMessage) {
	wa.UserLogin.Metadata.(*waid.UserLoginMetadata).MData = s
	err := wa.UserLogin.Save(context.Background())
	if err != nil {
		wa.UserLogin.Log.Err(err).Msg("Failed to save MC data")
	}
}
