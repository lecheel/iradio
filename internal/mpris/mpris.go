// Package mpris exposes iradio over the MPRIS2 D-Bus interface so desktop
// media controls can drive playback.
package mpris

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"

	"iradio/internal/music"
	"iradio/internal/stations"
)

// ActionMsg is emitted by the MPRIS layer in response to D-Bus method calls.
type ActionMsg struct {
	Action string
	URL    string
}

type root struct {
	prog *tea.Program
}

func (m *root) Raise() *dbus.Error { return nil }
func (m *root) Quit() *dbus.Error {
	if m.prog != nil {
		m.prog.Send(tea.Quit())
	}
	return nil
}

type player struct {
	prog *tea.Program
}

func (m *player) Next() *dbus.Error {
	if m.prog != nil {
		m.prog.Send(ActionMsg{Action: "next"})
	}
	return nil
}

func (m *player) Previous() *dbus.Error {
	if m.prog != nil {
		m.prog.Send(ActionMsg{Action: "previous"})
	}
	return nil
}

func (m *player) Pause() *dbus.Error {
	if m.prog != nil {
		m.prog.Send(ActionMsg{Action: "pause"})
	}
	return nil
}

func (m *player) PlayPause() *dbus.Error {
	if m.prog != nil {
		m.prog.Send(ActionMsg{Action: "playpause"})
	}
	return nil
}

func (m *player) Stop() *dbus.Error {
	if m.prog != nil {
		m.prog.Send(ActionMsg{Action: "stop"})
	}
	return nil
}

func (m *player) Play() *dbus.Error {
	if m.prog != nil {
		m.prog.Send(ActionMsg{Action: "play"})
	}
	return nil
}

func (m *player) Seek(offset int64) *dbus.Error                              { return nil }
func (m *player) SetPosition(trackId dbus.ObjectPath, pos int64) *dbus.Error { return nil }
func (m *player) OpenUri(uri string) *dbus.Error {
	if m.prog != nil {
		m.prog.Send(ActionMsg{Action: "open", URL: uri})
	}
	return nil
}

// Service wraps the D-Bus connection and properties for the MPRIS2 interface.
type Service struct {
	conn       *dbus.Conn
	properties *prop.Properties
	active     bool
}

// Start registers the MPRIS2 service on the session bus.
func Start(p *tea.Program) *Service {
	conn, err := dbus.SessionBus()
	if err != nil {
		return &Service{active: false}
	}

	reply, err := conn.RequestName("org.mpris.MediaPlayer2.iradio", dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		return &Service{active: false}
	}

	r := &root{prog: p}
	pl := &player{prog: p}

	_ = conn.Export(r, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2")
	_ = conn.Export(pl, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player")

	propsSpec := prop.Map{
		"org.mpris.MediaPlayer2": {
			"CanQuit":             {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanRaise":            {Value: false, Writable: false, Emit: prop.EmitTrue},
			"HasTrackList":        {Value: false, Writable: false, Emit: prop.EmitTrue},
			"Identity":            {Value: "iradio Player", Writable: false, Emit: prop.EmitTrue},
			"SupportedUriSchemes": {Value: []string{"http", "https", "file"}, Writable: false, Emit: prop.EmitTrue},
			"SupportedMimeTypes":  {Value: []string{"audio/mpeg", "audio/flac", "audio/mp4", "audio/x-m4a", "audio/ogg", "audio/wav", "application/x-mpegurl", "audio/aac"}, Writable: false, Emit: prop.EmitTrue},
		},
		"org.mpris.MediaPlayer2.Player": {
			"PlaybackStatus": {Value: "Stopped", Writable: false, Emit: prop.EmitTrue},
			"Rate":           {Value: 1.0, Writable: false, Emit: prop.EmitTrue},
			"Metadata":       {Value: map[string]dbus.Variant{}, Writable: false, Emit: prop.EmitTrue},
			"Volume":         {Value: 1.0, Writable: false, Emit: prop.EmitTrue},
			"Position":       {Value: int64(0), Writable: false, Emit: prop.EmitFalse},
			"CanControl":     {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanPlay":        {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanPause":       {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanGoNext":      {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanGoPrevious":  {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanSeek":        {Value: false, Writable: false, Emit: prop.EmitTrue},
		},
	}

	props, err := prop.Export(conn, "/org/mpris/MediaPlayer2", propsSpec)
	if err != nil {
		return &Service{active: false}
	}

	return &Service{
		conn:       conn,
		properties: props,
		active:     true,
	}
}

// Update refreshes MPRIS metadata for a live radio station.
func (m *Service) Update(status string, s *stations.Station) {
	if !m.active || m.properties == nil {
		return
	}
	m.properties.Set("org.mpris.MediaPlayer2.Player", "PlaybackStatus", dbus.MakeVariant(status))

	meta := map[string]dbus.Variant{}
	if s != nil {
		trackPath := dbus.ObjectPath(fmt.Sprintf("/org/mpris/MediaPlayer2/track/radio/%s", s.ID))
		meta["mpris:trackid"] = dbus.MakeVariant(trackPath)
		meta["xesam:title"] = dbus.MakeVariant(s.NameZh + " (" + s.NameEn + ")")
		broadcaster := "RTHK 香港電台"
		if s.Region == "TW" {
			broadcaster = s.NameZh + " (Taiwan)"
		} else if s.Region == "JP" {
			broadcaster = s.NameZh + " (Japan)"
		} else if s.Region == "SG" {
			broadcaster = s.NameZh + " (Singapore)"
		} else if s.Region == "MY" {
			broadcaster = s.NameZh + " (Malaysia)"
		}
		meta["xesam:artist"] = dbus.MakeVariant([]string{broadcaster})
		meta["xesam:album"] = dbus.MakeVariant(s.Dial)
		if s.StreamURL != "" {
			meta["xesam:url"] = dbus.MakeVariant(s.StreamURL)
		}
	} else {
		meta["mpris:trackid"] = dbus.MakeVariant(dbus.ObjectPath("/org/mpris/MediaPlayer2/TrackList/NoTrack"))
	}
	m.properties.Set("org.mpris.MediaPlayer2.Player", "Metadata", dbus.MakeVariant(meta))
	m.properties.Set("org.mpris.MediaPlayer2.Player", "Position", dbus.MakeVariant(int64(0)))
}

// UpdateMusic refreshes MPRIS metadata for a local music track.
func (m *Service) UpdateMusic(status string, t *music.MusicTrack) {
	if !m.active || m.properties == nil {
		return
	}
	m.properties.Set("org.mpris.MediaPlayer2.Player", "PlaybackStatus", dbus.MakeVariant(status))
	meta := map[string]dbus.Variant{}
	if t != nil {
		trackPath := dbus.ObjectPath(fmt.Sprintf("/org/mpris/MediaPlayer2/track/music/%d", time.Now().UnixNano()))
		meta["mpris:trackid"] = dbus.MakeVariant(trackPath)
		meta["xesam:title"] = dbus.MakeVariant(t.Title)
		meta["xesam:artist"] = dbus.MakeVariant([]string{t.Artist})
		meta["xesam:album"] = dbus.MakeVariant(t.Album)
		if t.Duration > 0 {
			meta["mpris:length"] = dbus.MakeVariant(int64(t.Duration.Microseconds()))
		}
		if t.Path != "" {
			meta["xesam:url"] = dbus.MakeVariant("file://" + t.Path)
		}
	} else {
		meta["mpris:trackid"] = dbus.MakeVariant(dbus.ObjectPath("/org/mpris/MediaPlayer2/TrackList/NoTrack"))
	}
	m.properties.Set("org.mpris.MediaPlayer2.Player", "Metadata", dbus.MakeVariant(meta))
	m.properties.Set("org.mpris.MediaPlayer2.Player", "Position", dbus.MakeVariant(int64(0)))
}

// UpdatePosition updates the current track position in microseconds.
func (m *Service) UpdatePosition(pos time.Duration) {
	if !m.active || m.properties == nil {
		return
	}
	m.properties.Set("org.mpris.MediaPlayer2.Player", "Position", dbus.MakeVariant(int64(pos.Microseconds())))
}
