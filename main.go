package main

import (
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

	http.ListenAndServe("0.0.0.0:2208", r)
}

func update() ([]*Channel, []*ClientInfo, error) {
	client, err := ts3.NewClient(server + ":" + port)

	if err != nil {
		return nil, nil, err
	}

	defer client.Close()

	u, _ := strconv.ParseInt(serverID, 0, 10)

	time.Sleep(1 * time.Second)

	if err := client.Use(int(u)); err != nil {
		return nil, nil, err
	}

	time.Sleep(1 * time.Second)

	err = client.Login(username, password)

	if err != nil {
		return nil, nil, err
	}

	cmds := ts3.ServerMethods{Client: client}

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

		_, err := client.ExecCmd(ts3.NewCmd("clientinfo").WithArgs(ts3.NewArg("clid", c.ID)).WithResponse(&cl))

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

const htmlTemplate = `
<!doctype html>
<html lang="en">

<head>
	<title>Teamspeak List</title>
	<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/4.1.3/css/bootstrap.min.css" />
	<link href="data:image/x-icon;base64,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" rel="icon" type="image/x-icon">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<style type="text/css">
		div.channel {
			margin-top: 20px;
			margin-bottom: 20px;
		}

		footer {
			margin-top: 40px;
		}
	</style>
</head>

<body>
	<nav class="navbar sticky-top navbar-expand-lg navbar-dark" style="background-color: #0d619e !important">
		<div class="container">
			<a class="navbar-brand" href="/">Teamspeak List</a>
		</div>
	</nav>

	<div class="container">
		{{ $users := .users }}

		{{ range $i, $channel := .channels }}
			<div class="channel table-responsive">
				<h3><small>#{{ $channel.ID }}</small> - {{ $channel.ChannelName }}</h3>

					{{ $hasUsers := false }}

					{{ range $j, $user := $users }}
						{{ if eq $user.ChannelID $channel.ID }}
							{{ $hasUsers = true }}
						{{ end }}
					{{ end }}


					{{ if $hasUsers }}
					<table style="width: 100%;" class="table table-striped table-bordered">
						<tr>
							<th>Nickname</th>
							<th>OS</th>
							<th class="text-center">Microphone Muted?</th>
							<th class="text-center">Speakers Muted?</th>
							<th class="text-center">Bandwidth Wasted</th>
						</tr>
					{{ range $j, $user := $users }}
						{{ if eq $user.ChannelID $channel.ID }}
							<tr>
								<td>{{ $user.Nickname }}{{ with $user.PhoneticNickname }}<br><small><em>noun</em>: /{{ $user.PhoneticNickname }}/</small>{{ end }}</td>
								<td>{{ $user.Platform }}</td>
								<td class="text-center">{{ yes $user.InputMuted }}</td>
								<td class="text-center">{{ yes $user.OutputMuted }}</td>
								<td class="text-center">{{ bytes $user.TotalBytesDownloaded }} down / {{ bytes $user.TotalBytesUploaded }} up</td>
							</tr>
						{{ end }}
					{{ else }}
						<p>nobody likes this channel</p>
					{{ end }}
				{{ end }}
				</table>
			</div>
		{{ else }}
			<br><br>
			<p>eep! no data yet...</p>
		{{ end }}
	</div>

	<footer class="text-muted">
		<div class="container">
			<b>updated {{ .updated }}</b><br><br>

			<em>A slightly less useless project by seejy</em>
		</div>
	</footer>
</body>

</html>
`
