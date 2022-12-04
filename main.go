package main

import (
	"embed"
	"html/template"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/multiplay/go-ts3"
	"github.com/sirupsen/logrus"
)

var (
	server   = os.Getenv("TS3_SERVER")
	port     = os.Getenv("TS3_PORT")
	username = os.Getenv("TS3_USERNAME")
	password = os.Getenv("TS3_PASSWORD")
	serverID = os.Getenv("TS3_SERVERID")
)

type Channel struct {
	ID                   int    `ms:"cid"`
	ParentID             int    `ms:"pid"`
	ChannelOrder         int    `ms:"channel_order"`
	ChannelName          string `ms:"channel_name"`
	TotalClients         int    `ms:"total_clients"`
	NeededSubscribePower int    `ms:"channel_needed_subscribe_power"`
}

type ClientInfo struct {
	ID         int    `ms:"clid"`
	DatabaseID int    `ms:"client_database_id"`
	Nickname   string `ms:"client_nickname"`
	Type       int    `ms:"client_type"`

	ChannelID int `ms:"cid"`

	// ClientIdleTime is the time the client has been idle.
	ClientIdleTime int `ms:"client_idle_time"`
	// Version is the client version.
	Version string `ms:"client_version"`
	// Platform is the client platform.
	Platform string `ms:"client_platform"`
	// InputMuted indicates if the client mic is muted.
	InputMuted bool `ms:"client_input_muted"`
	// OutputMuted indicates if the client speakers are muted.
	OutputMuted bool `ms:"client_output_muted"`
	// OutputOnlyMuted - ?
	OutputOnlyMuted bool `ms:"client_outputonly_muted"`
	// HasInputHardware indicates if the client has a mic.
	HasInputHardware bool `ms:"client_input_hardware"`
	// HasOutputHardware indicates if the client has a mic.
	HasOutputHardware bool `ms:"client_output_hardware"`
	// DefaultChannelName is the name of the client's default channel.
	DefaultChannelName string `ms:"client_default_channel"`
	// IsRecording indicates if the client is recording.
	IsRecording bool `ms:"client_is_recording"`
	// ChannelGroupId is the ID of the channel group the client is in.
	ChannelGroupId int `ms:"client_channel_group_id"`
	// ServerGroups is the list of server groups on the client.
	// ServerGroups []int `ms:"client_servergroups"`
	// TotalConnections is the total number of times the client has connected.
	TotalConnections int `ms:"client_totalconnections"`
	// Away is set if the client is marked as away.
	Away bool `ms:"client_away"`
	// AwayMessage is the client away message.
	AwayMessage string `ms:"client_away_message"`
	// TalkPower is the talk power of the client.
	TalkPower int `ms:"client_talk_power"`
	// TalkRequest indicates if the client has requested to talk.
	TalkRequest bool `ms:"client_talk_request"`
	// TalkRequestMessage is the message the client gave when requesting to talk.
	TalkRequestMessage string `ms:"client_talk_request_msg"`
	// Description is the description of the client.
	Description string `ms:"client_description"`
	// IsTalker indicates if the client is a talker.
	IsTalker bool `ms:"client_is_talker"`
	// MonthBytesUploaded is the number of bytes uploaded this month.
	MonthBytesUploaded int `ms:"client_month_bytes_uploaded"`
	// MonthBytesDownloaded is the number of bytes downloaded this month.
	MonthBytesDownloaded int `ms:"client_month_bytes_downloaded"`
	// TotalBytesUploaded is the number of bytes uploaded total.
	TotalBytesUploaded int `ms:"client_total_bytes_uploaded"`
	// TotalBytesDownloaded is the number of bytes downloaded total.
	TotalBytesDownloaded int `ms:"client_total_bytes_downloaded"`
	// IsPrioritySpeaker indicates if the client is a priority speaker
	IsPrioritySpeaker bool `ms:"client_is_priority_speaker"`
	// PhoneticNickname is the phonetic nickname if given.
	PhoneticNickname string `ms:"client_nickname_phonetic"`
	// NeededmsViewPower is the view power necessary to view the client.
	NeededmsViewPower int `ms:"client_needed_ms_view_power"`
	// IconId is the icon ID of the client.
	IconId int `ms:"client_icon_id"`
	// IsChannelCommander indicates if the client is a channel commander.
	IsChannelCommander bool `ms:"is_channel_commander"`
}

type OnlineClient struct {
	ID          int    `ms:"clid"`
	DatabaseID  int    `ms:"client_database_id"`
	Nickname    string `ms:"client_nickname"`
	Type        int    `ms:"client_type"`
	Away        bool   `ms:"client_away"`
	AwayMessage string `ms:"client_away_message"`
}

func main() {
	r := mux.NewRouter()
	updated := time.Now()

	var channels []*Channel
	var users []*ClientInfo

	t, err := template.New("users").Funcs(map[string]interface{}{"yes": func(b bool) string {
		if b {
			return "yes"
		} else {
			return "no"
		}
	}, "bytes": func(i int) string { return humanize.Bytes(uint64(i)) }}).Parse(htmlTemplate)

	if err != nil {
		panic(err)
	}

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := t.Execute(w, map[string]interface{}{
			"channels": channels,
			"users":    users,
			"updated":  updated.Format(time.RFC1123),
		})

		if err != nil {
			panic(err)
		}
	})

	go func() {
		for {
			updatedChannels, updatedUsers, err := update()

			if err != nil {
				logrus.Errorf("Unable to update, err: %s", err.Error())
				continue
			}

			channels = updatedChannels
			users = updatedUsers
			updated = time.Now()

			time.Sleep(5 * time.Second)
		}
	}()

	r.PathPrefix("/static/").Handler(http.FileServer(http.FS(content)))

	http.ListenAndServe("0.0.0.0:2208", r)
}

func update() ([]*Channel, []*ClientInfo, error) {
	client, err := ts3.NewClient(server + ":" + port)

	if err != nil {
		return nil, nil, err
	}

	cmds := &ts3.ServerMethods{Client: client}
	defer cmds.Close()

	u, _ := strconv.ParseInt(serverID, 0, 10)

	time.Sleep(1 * time.Second)

	if err := cmds.Use(int(u)); err != nil {
		return nil, nil, err
	}

	time.Sleep(1 * time.Second)

	err = cmds.Login(username, password)

	if err != nil {
		return nil, nil, err
	}

	var clientList []*OnlineClient
	if _, err := cmds.ExecCmd(ts3.NewCmd("clientlist").WithResponse(&clientList)); err != nil {
		return nil, nil, err
	}

	time.Sleep(1 * time.Second)

	if len(clientList) == 0 {
		return nil, nil, err
	}

	var channels []*Channel

	if _, err := cmds.ExecCmd(ts3.NewCmd("channellist").WithResponse(&channels)); err != nil {
		return nil, nil, err
	}

	time.Sleep(1 * time.Second)

	clientInfo := make([]*ClientInfo, 0)

	for _, c := range clientList {
		var cl *ClientInfo

		_, err := cmds.ExecCmd(ts3.NewCmd("clientinfo").WithArgs(ts3.NewArg("clid", c.ID)).WithResponse(&cl))

		if err != nil {
			return nil, nil, err
		}

		time.Sleep(1 * time.Second)

		if strings.Contains(cl.Platform, "ServerQuery") {
			continue
		}

		clientInfo = append(clientInfo, cl)
	}

	sort.Slice(clientInfo, func(i, j int) bool {
		return clientInfo[i].Nickname < clientInfo[j].Nickname
	})

	return channels, clientInfo, nil
}

//go:embed html/index.html
var htmlTemplate string

//go:embed static
var content embed.FS
