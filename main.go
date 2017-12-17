package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/multiplay/go-ts3"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
)

var (
	server   = os.Getenv("TS3_SERVER")
	port     = os.Getenv("TS3_PORT")
	username = os.Getenv("TS3_USERNAME")
	password = os.Getenv("TS3_PASSWORD")
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
	ServerGroups []int `ms:"client_servergroups"`
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

func main() {
	client, err := ts3.NewClient(server + ":" + port)

	if err != nil {
		panic(err)
	}

	defer client.Close()

	if err := client.Use(1); err != nil {
		panic(err)
	}

	err = client.Login(username, password)

	if err != nil {
		panic(err)
	}

	cmds := ts3.ServerMethods{Client: client}

	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "teamspeak users online. yet another useless project by seejy\n-------------------------------------------------------------\n\n")

		clientList, err := cmds.ClientList()

		if err != nil {
			logrus.Errorf("unable to get client list, err: %s", err)
			http.Error(w, "can't get client list", http.StatusInternalServerError)
			return
		}

		if len(clientList) == 0 {
			fmt.Fprint(w, "no users online")
			return
		}

		var channels []*Channel

		if _, err := cmds.ExecCmd(ts3.NewCmd("channellist").WithResponse(&channels)); err != nil {
			logrus.Errorf("unable to get chan list, err: %s", err)
			http.Error(w, "can't get chan list", http.StatusInternalServerError)
			return
		}

		clientInfo := make([]*ClientInfo, 0)

		for _, c := range clientList {
			var cl *ClientInfo

			_, err := client.ExecCmd(ts3.NewCmd("clientinfo").WithArgs(ts3.NewArg("clid", c.ID)).WithResponse(&cl))

			if err != nil {
				logrus.Errorf("unable to get client info, err: %s", err)
				http.Error(w, "can't get client info", http.StatusInternalServerError)
				return
			}

			clientInfo = append(clientInfo, cl)
		}

		for _, ch := range channels {
			fmt.Fprintf(w, "#%d - %s\n\n", ch.ID, ch.ChannelName)

			for _, c := range clientInfo {
				if strings.Contains(c.Nickname, "serveradmin") {
					continue
				}

				if c.ChannelID == ch.ID {
					fmt.Fprintf(w, "\t%s\n\t\tmic muted:\t%t\n\t\tspeakers muted:\t%t\n\n\n", c.Nickname, c.InputMuted, c.OutputMuted)
				}
			}
		}
	})

	http.ListenAndServe("0.0.0.0:2208", r)
}
